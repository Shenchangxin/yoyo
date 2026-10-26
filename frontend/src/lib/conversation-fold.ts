/**
 * 拥有：把助手流帧折进同一条尝试。持久收束替换它，瞬时状态留到 step_end。
 * 不拥有：帧的序号规则。那是 ClientAssistantStream 的事。
 */

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

/** 复制条目并盖上补丁。补丁里的 payload 与原载荷合并，并丢掉 stream。it 读取 payload，展开传递。patch 读取 payload，展开传递。 */
function cloneItem(it: Item, patch: Partial<Item> & { payload?: Record<string, any> }): Item {
  const payload = { ...it.payload, ...(patch.payload || {}) };
  delete payload.stream;
  return {
    ...it,
    ...patch,
    payload,
  };
}

/** 助手条目的键，按会话和回合。sessionId 拼进字符串。round 拼进字符串。 */
function assistantKey(sessionId: string, round: string): string {
  return `${sessionId}:assistant:${round}`;
}

/** 账本类：回合与步骤的起止、收件箱、系统、评测、演化、文件变更、ask_user。type 是 turn_end、turn_start、step_start、step_end、inbox、system、eval、evolve、file_change 或 ask_user 时返回真。 */
function isLedger(type: string): boolean {
  return type === "turn_end" || type === "turn_start" || type === "step_start" || type === "step_end"
    || type === "inbox" || type === "system" || type === "eval" || type === "evolve" || type === "file_change" || type === "ask_user";
}

/** 从载荷解析内容块。解析不了则没有。it 读取 payload。 */
function blocksOf(it: Item): ContentBlock[] | undefined {
  return parseContentBlocks(it.payload?.blocks);
}

/** 用内容块重写正文和推理，并丢掉 stream。有推理块才写入 reasoning。it 读取 payload，传给 cloneItem。blocks 传给 textFromBlocks，写入字段 blocks。extra 展开传递。 */
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

/** 载荷里没有 stream 时原样返回。有则去掉 stream。payload 为假时改用后备值，读取 stream，展开传递。 */
function dropStreamTape(payload: Record<string, any> | undefined): Record<string, any> {
  if (!payload || payload.stream == null) return payload || {};
  const next = { ...payload };
  delete next.stream;
  return next;
}

/** 取出 request_id，没有则取 requestId。it 读取 payload。 */
function userRequestId(it: Item): string {
  return String(it.payload?.request_id || it.payload?.requestId || "");
}

/** ConversationFold 座位折叠。实时块更新同一次尝试，持久收束替换它，瞬时状态留到 step_end 或 turn_end。 */
export class ConversationFold {
  private list: Item[] = [];
  private readonly stream = new ClientAssistantStream();
  private attempts = new Map<string, AttemptState>();

  /** from 用现有条目建一份折叠，并恢复仍在流式的尝试。items 截取。 */
  static from(items: Item[]): ConversationFold {
    const f = new ConversationFold();
    f.list = items.slice();
    f.restoreLiveAttempts();
    return f;
  }

  /** items 当前条目列表。 */
  items(): Item[] {
    return this.list;
  }

  /** replace 换成另一份列表。先清掉流状态，再恢复仍在流式的尝试。items 截取。 */
  replace(items: Item[]): Item[] {
    this.stream.reset();
    this.attempts.clear();
    this.list = items.slice();
    this.restoreLiveAttempts();
    return this.list;
  }

  /** trim 换成裁过的列表。键已经不在列表里的尝试退休。items 做映射，记到 this.list。 */
  trim(items: Item[]): void {
    this.list = items;
    const keys = new Set(items.map((x) => x.key));
    for (const [id, st] of this.attempts) {
      if (keys.has(st.key)) continue;
      this.attempts.delete(id);
      this.stream.retire(id);
    }
  }

  /** accept 折进一条并返回列表。ev 传给 acceptCore。 */
  accept(ev: Item): Item[] {
    this.acceptCore(ev);
    return this.list;
  }

  /**
   * 流帧走 acceptStream。step_end 与 turn_end 退休流并收束。账本类忽略。compaction 只留 kind 为 checkpoint 的。用户、助手、推理、工具分别处理。键已存在则忽略，否则追加。
   * ev 读取 type、payload、key，为假时走另一支，传给 acceptStream，传给 acceptUser，传给 acceptSettlement，传给 acceptReasoning，传给 upsertByKey。
   */
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

  /**
   * 界面来源且已有相同正文的非引导用户消息则忽略，否则追加。引导先去掉末尾重复的同文引导再追加。键或非界面来源的同一 requestId 已存在则忽略。作曲回声即使后面已经有实时助手，也在原位替换，避免正式用户消息把回合拆成两段。
   * ev 读取 source、name、text、key，赋给 next[i]，传给 userRequestId，展开传递。
   */
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
    // 作曲回声即使后面已经有实时助手，也在原位替换。把正式用户消息追加在后面会把回合拆成两段。
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

  /**
   * 帧无法解析则返回。重新对齐时先重置；开始帧再收一次，其他帧丢掉。开始走 beginAttempt。块推进组装器，stream_index 为 nextIndex 减一。结束时 abandoned 去掉该键上的空正文并删除尝试。其他结束只把 streaming 标上。
   * ev 传给 frameFromItem，传给 beginAttempt。
   */
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

  /** 同一 attemptId 已有则返回。没有对应助手条目时追加一条空的流式助手。ev 读取 payload、sessionId、source、ts、seq。turn 写入字段 turn。step 写入字段 step。 */
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

  /** 仍在流式且已有流式条目时，序号不高于已有 stream_index 则忽略，否则把正文接上并丢掉 stream。否则用收束条目替换或追加。不再流式或已中断则删除尝试并退休。ev 读取 payload、key、sessionId、delta、text，传给 blocksOf，传给 cloneItem。 */
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

  /** 增量按回合把推理正文接上。非增量则写进该回合助手的内容块，换掉原有推理块。没有助手条目且给了这条推理时用它创建。ev 读取 payload、delta、sessionId、key、text，传给 patchAssistant，展开传递。 */
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

  /** 仍标着 delta 或 streaming 的助手收成非流式，并丢掉 stream。清空尝试。 */
  private settleStreaming(): void {
    this.list = this.list.map((it) => {
      if (it.type !== "assistant") return it;
      if (!it.delta && !it.payload?.streaming) return it;
      return cloneItem(it, { delta: false, payload: dropStreamTape({ ...it.payload, streaming: false, delta: false }) });
    });
    this.attempts.clear();
  }

  /**
   * 没有键则追加。工具调用的进度事件不覆盖还没有进度的调用。工具结果的增量在已有非增量正文时忽略，否则把 content 接上。非增量的工具结果清掉 delta，content 空则留原来的。工具调用不带进度时删掉载荷里的 progress。
   * ev 读取 key、type、payload、delta、text，展开传递。
   */
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

  /** 重置后，为仍是 delta 或 streaming 的助手重建组装器和流状态。nextIndex 为 stream_index 加一，没有序号则为 0。 */
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

  /** 按键改一条助手。没有这条且给了 create 时，用函数处理 create 后追加。key 与 x.key 比较是否相等。fn 被调用，并传入 create。 */
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

/** mergeSeats 从列表建一份折叠并收下这一条。list 传给 ConversationFold.from。ev 传给 accept。 */
export function mergeSeats(list: Item[], ev: Item): Item[] {
  return ConversationFold.from(list).accept(ev);
}

