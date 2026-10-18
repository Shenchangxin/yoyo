import type { Item } from "./protocol";
import { ClientAssistantStream, frameFromItem } from "./assistant-stream";
import { sameUserTurnText } from "./mentions";
import {
  BlockAssembler,
  parseContentBlocks,
  textFromBlocks,
  type ContentBlock,
} from "./stream-chunk";

type AttemptState = {
  attemptId: string;
  round: string;
  turn: number;
  step: number;
  assembler: BlockAssembler;
  streaming: boolean;
  interrupted: boolean;
  key: string;
  nextIndex: number;
};

function cloneItem(it: Item, patch: Partial<Item> & { payload?: Record<string, any> }): Item {
  const payload = { ...it.payload, ...(patch.payload || {}) };
  delete payload.stream;
  return {
    ...it,
    ...patch,
    payload,
  };
}

function assistantKey(sessionId: string, round: string): string {
  return `${sessionId}:assistant:${round}`;
}

function isLedger(type: string): boolean {
  return type === "turn_end" || type === "turn_start" || type === "step_start" || type === "step_end"
    || type === "inbox" || type === "system" || type === "eval" || type === "evolve" || type === "file_change" || type === "ask_user";
}

function blocksOf(it: Item): ContentBlock[] | undefined {
  return parseContentBlocks(it.payload?.blocks);
}

function applyBlocks(it: Item, blocks: ContentBlock[], extra?: Record<string, any>): Item {
  const text = textFromBlocks(blocks, "text");
  const reasoning = textFromBlocks(blocks, "reasoning");
  const payload: Record<string, any> = {
    ...it.payload,
    text,
    blocks,
    ...(reasoning ? { reasoning } : {}),
    ...extra,
  };
  delete payload.stream;
  return cloneItem(it, { text, payload });
}

function dropStreamTape(payload: Record<string, any> | undefined): Record<string, any> {
  if (!payload || payload.stream == null) return payload || {};
  const next = { ...payload };
  delete next.stream;
  return next;
}

function userRequestId(it: Item): string {
  return String(it.payload?.request_id || it.payload?.requestId || "");
}

/**
 * Seat fold: live StreamChunk frames update one assistant attempt; durable
 * settlement replaces it; transients stay until step_end.
 */
export class ConversationFold {
  private list: Item[] = [];
  private readonly stream = new ClientAssistantStream();
  private attempts = new Map<string, AttemptState>();

  static from(items: Item[]): ConversationFold {
    const f = new ConversationFold();
    f.list = items.slice();
    f.restoreLiveAttempts();
    return f;
  }

  items(): Item[] {
    return this.list;
  }

  replace(items: Item[]): Item[] {
    this.stream.reset();
    this.attempts.clear();
    this.list = items.slice();
    this.restoreLiveAttempts();
    return this.list;
  }

  trim(items: Item[]): void {
    this.list = items;
    const keys = new Set(items.map((x) => x.key));
    for (const [id, st] of this.attempts) {
      if (keys.has(st.key)) continue;
      this.attempts.delete(id);
      this.stream.retire(id);
    }
  }

  accept(ev: Item): Item[] {
    this.acceptCore(ev);
    return this.list;
  }

  private acceptCore(ev: Item): void {
    if (!ev) return;
    if (ev.type === "assistant_stream") {
      this.acceptStream(ev);
      return;
    }
    if (ev.type === "step_end" || ev.type === "turn_end") {
      this.stream.retire();
      this.settleStreaming();
      return;
    }
    if (isLedger(ev.type)) return;
    if (ev.type === "compaction") {
      const kind = String(ev.payload?.kind || "");
      if (kind !== "checkpoint") return;
    }

    if (ev.type === "user") {
      this.acceptUser(ev);
      return;
    }
    if (ev.type === "assistant") {
      this.acceptSettlement(ev);
      return;
    }
    if (ev.type === "reasoning") {
      this.acceptReasoning(ev);
      return;
    }
    if (ev.type === "tool_call" || ev.type === "tool_result") {
      this.upsertByKey(ev);
      return;
    }
    if (ev.key && this.list.some((x) => x.key === ev.key)) return;
    this.list = [...this.list, ev];
  }

  private acceptUser(ev: Item): void {
    const steer = ev.source === "steer" || ev.name === "steer";
    const requestId = userRequestId(ev);
    const incoming = ev.text;
    if (ev.source === "ui") {
      if (this.list.some((x) => x.type === "user" && x.source !== "steer" && sameUserTurnText(x.text, incoming))) return;
      this.list = [...this.list, ev];
      return;
    }
    const next = this.list.slice();
    if (steer) {
      while (next.length) {
        const last = next[next.length - 1];
        if (last.type !== "user") break;
        if (last.source === "steer" && last.text === incoming) {
          next.pop();
          continue;
        }
        break;
      }
      this.list = [...next, { ...ev, source: "steer" }];
      return;
    }
    if (ev.key && next.some((x) => x.key === ev.key)) {
      this.list = next;
      return;
    }
    if (requestId && next.some((x) => x.type === "user" && x.source !== "ui" && x.source !== "steer" && userRequestId(x) === requestId)) {
      this.list = next;
      return;
    }
    // Replace the composer echo in place even if a live assistant already
    // landed after it. Appending the durable user would split the turn in two.
    for (let i = next.length - 1; i >= 0; i--) {
      const it = next[i];
      if (it.type !== "user" || it.source === "steer") continue;
      const rid = userRequestId(it);
      const echo = it.source === "ui" && ((requestId && rid === requestId) || sameUserTurnText(it.text, incoming));
      if (echo) {
        next[i] = ev;
        this.list = next;
        return;
      }
      break;
    }
    this.list = [...next, ev];
  }

  private acceptStream(ev: Item): void {
    const frame = frameFromItem(ev);
    if (!frame) return;
    let result = this.stream.acceptFrame(frame);
    if (result?.type === "rebaseline") {
      this.stream.reset();
      if (frame.kind === "start") {
        result = this.stream.acceptFrame(frame);
      } else {
        return;
      }
    }
    if (!result) return;
    if (result.type === "start") {
      this.beginAttempt(ev, result.attemptId, result.turn, result.step);
      return;
    }
    if (result.type === "chunk") {
      const st = this.attempts.get(result.attemptId);
      if (!st) return;
      st.assembler.push(result.chunk);
      st.nextIndex = this.stream.activeAttempt?.nextIndex ?? st.nextIndex;
      const lastIndex = Math.max(0, st.nextIndex - 1);
      const blocks = st.assembler.blocks();
      this.patchAssistant(st.key, (it) =>
        applyBlocks(it, blocks, {
          streaming: true,
          delta: true,
          attempt_id: st.attemptId,
          stream_index: lastIndex,
        }),
      );
      return;
    }
    if (result.type === "end") {
      const st = this.attempts.get(result.attemptId);
      if (!st) return;
      if (result.outcome === "abandoned") {
        this.list = this.list.filter((x) => x.key !== st.key || x.text.trim() !== "");
        this.attempts.delete(result.attemptId);
        return;
      }
      st.streaming = true;
      this.patchAssistant(st.key, (it) => cloneItem(it, { payload: { ...it.payload, streaming: true } }));
    }
  }

  private beginAttempt(ev: Item, attemptId: string, turn: number, step: number): void {
    const existing = this.attempts.get(attemptId);
    if (existing) return;
    const round = String(ev.payload?.round || ev.payload?.id || attemptId);
    const key = assistantKey(ev.sessionId, round);
    this.attempts.set(attemptId, {
      attemptId,
      round,
      turn,
      step,
      assembler: new BlockAssembler(),
      streaming: true,
      interrupted: false,
      key,
      nextIndex: 0,
    });
    if (!this.list.some((x) => x.key === key)) {
      this.list = [
        ...this.list,
        {
          key,
          type: "assistant",
          sessionId: ev.sessionId,
          source: ev.source || "model",
          ts: ev.ts,
          text: "",
          name: "",
          delta: true,
          payload: {
            id: round,
            round,
            attempt_id: attemptId,
            turn,
            step,
            streaming: true,
            delta: true,
          },
          seq: ev.seq,
        },
      ];
    }
  }

  private acceptSettlement(ev: Item): void {
    const round = String(ev.payload?.id || ev.payload?.round || "");
    const attemptId = String(ev.payload?.attempt_id || ev.payload?.attemptId || round);
    const blocks = blocksOf(ev);
    const key = ev.key || assistantKey(ev.sessionId, round);
    const idx = this.list.findIndex((x) => x.key === key || (x.type === "assistant" && String(x.payload?.id || x.payload?.round) === round));
    const streaming = !ev.payload?.interrupted && (!!ev.delta || !!ev.payload?.streaming);
    if (idx >= 0 && streaming && (this.list[idx].delta || this.list[idx].payload?.streaming || this.list[idx].payload?.delta)) {
      const cur = this.list[idx];
      const have = Number(cur.payload?.stream_index || 0);
      const incomingIndex = Number(ev.payload?.index || 0);
      if (incomingIndex && incomingIndex <= have) return;
      const text = `${cur.text || ""}${ev.text || ""}`;
      const payload = dropStreamTape({
        ...cur.payload,
        ...ev.payload,
        id: round,
        round,
        text,
        streaming: true,
        delta: true,
        stream_index: incomingIndex || have,
      });
      const copy = this.list.slice();
      copy[idx] = cloneItem(cur, { text, delta: true, payload });
      this.list = copy;
      return;
    }
    const payload = dropStreamTape({
      ...ev.payload,
      id: round,
      round,
      blocks: blocks || ev.payload?.blocks,
      streaming,
      interrupted: !!ev.payload?.interrupted,
      delta: streaming,
      ...(streaming && ev.payload?.index != null ? { stream_index: ev.payload.index } : {}),
    });
    const next = cloneItem(ev, {
      key,
      delta: streaming,
      text: ev.text || (blocks ? textFromBlocks(blocks, "text") : ev.text),
      payload,
    });
    if (idx >= 0) {
      const copy = this.list.slice();
      copy[idx] = { ...copy[idx], ...next, key: copy[idx].key, payload: dropStreamTape({ ...copy[idx].payload, ...next.payload }) };
      this.list = copy;
    } else {
      this.list = [...this.list, next];
    }
    if (!streaming || ev.payload?.interrupted) {
      this.attempts.delete(attemptId);
      this.stream.retire(attemptId);
    }
  }

  private acceptReasoning(ev: Item): void {
    const round = String(ev.payload?.id || ev.payload?.round || "");
    if (ev.delta) {
      const rkey = `${ev.sessionId}:reasoning:${round}`;
      const idx = this.list.findIndex((x) =>
        x.key === rkey
        || x.key === ev.key
        || (x.type === "reasoning" && String(x.payload?.id || x.payload?.round) === round),
      );
      if (idx >= 0) {
        const cur = this.list[idx];
        const copy = this.list.slice();
        copy[idx] = {
          ...cur,
          ...ev,
          key: cur.key,
          text: `${cur.text || ""}${ev.text || ""}`,
          delta: true,
          payload: { ...cur.payload, ...ev.payload, delta: true },
        };
        this.list = copy;
        return;
      }
      this.list = [...this.list, { ...ev, key: ev.key || rkey, delta: true }];
      return;
    }
    const key = assistantKey(ev.sessionId, round);
    this.patchAssistant(key, (it) => {
      const blocks = blocksOf(it) || [];
      const without = blocks.filter((b) => b.type !== "reasoning");
      return applyBlocks(it, [...without, { type: "reasoning", text: ev.text }]);
    }, ev);
  }

  private settleStreaming(): void {
    this.list = this.list.map((it) => {
      if (it.type !== "assistant") return it;
      if (!it.delta && !it.payload?.streaming) return it;
      return cloneItem(it, { delta: false, payload: dropStreamTape({ ...it.payload, streaming: false, delta: false }) });
    });
    this.attempts.clear();
  }

  private upsertByKey(ev: Item): void {
    const idx = ev.key ? this.list.findIndex((x) => x.key === ev.key) : -1;
    if (idx < 0) {
      this.list = [...this.list, ev];
      return;
    }
    const cur = this.list[idx];
    if (ev.type === "tool_call" && ev.payload?.progress && !cur.payload?.progress) return;
    if (ev.type === "tool_result" && ev.delta) {
      const curBody = String(cur.payload?.content ?? cur.text ?? "");
      if (curBody && !cur.delta) return;
      const add = String(ev.payload?.content ?? ev.text ?? "");
      const body = curBody + add;
      const next = this.list.slice();
      next[idx] = {
        ...cur,
        ...ev,
        text: body,
        delta: true,
        payload: { ...cur.payload, ...ev.payload, content: body, delta: true },
      };
      this.list = next;
      return;
    }
    const payload = { ...cur.payload, ...ev.payload };
    if (ev.type === "tool_call" && !ev.payload?.progress) delete payload.progress;
    let delta = ev.delta;
    if (ev.type === "tool_result" && !ev.delta) {
      delta = false;
      payload.delta = false;
      if (payload.content == null && cur.payload?.content != null) payload.content = cur.payload.content;
    }
    const next = this.list.slice();
    next[idx] = {
      ...cur,
      ...ev,
      delta,
      text: ev.type === "tool_result" && !ev.delta
        ? String(payload.content ?? ev.text ?? cur.text ?? "")
        : ev.text,
      payload,
    };
    this.list = next;
  }

  private restoreLiveAttempts(): void {
    this.stream.reset();
    this.attempts.clear();
    for (const it of this.list) {
      if (it.type !== "assistant") continue;
      if (!it.delta && !it.payload?.streaming) continue;
      const round = String(it.payload?.id || it.payload?.round || "");
      const attemptId = String(it.payload?.attempt_id || it.payload?.attemptId || round);
      if (!attemptId) continue;
      const assembler = new BlockAssembler();
      assembler.hydrateFromBlocks(blocksOf(it) || []);
      const streamIndex = it.payload?.stream_index;
      const nextIndex = typeof streamIndex === "number" ? streamIndex + 1 : 0;
      this.attempts.set(attemptId, {
        attemptId,
        round,
        turn: Number(it.payload?.turn || 0),
        step: Number(it.payload?.step || 0),
        assembler,
        streaming: true,
        interrupted: !!it.payload?.interrupted,
        key: it.key || assistantKey(it.sessionId, round),
        nextIndex,
      });
      this.stream.hydrate({
        attemptId,
        turn: Number(it.payload?.turn || 0),
        step: Number(it.payload?.step || 0),
        nextIndex,
      });
    }
  }

  private patchAssistant(key: string, fn: (it: Item) => Item, create?: Item): void {
    const idx = this.list.findIndex((x) => x.key === key);
    if (idx < 0) {
      if (create) this.list = [...this.list, fn(create)];
      return;
    }
    const next = this.list.slice();
    next[idx] = fn(this.list[idx]);
    this.list = next;
  }
}

export function mergeSeats(list: Item[], ev: Item): Item[] {
  return ConversationFold.from(list).accept(ev);
}

