import { asBool, pick, str } from "./normalize";
import type { Item, ItemType } from "./protocol";

function payload(raw: any): Record<string, any> {
  const p = pick(raw, "payload", "Payload");
  return p && typeof p === "object" ? p : {};
}

export function coerceTS(v: any): string {
  if (v == null || v === "") return "";
  if (typeof v === "string") {
    if (v.startsWith("0001-01-01")) return "";
    return v;
  }
  if (typeof v === "number" && Number.isFinite(v)) return new Date(v).toISOString();
  if (v instanceof Date && !Number.isNaN(v.getTime())) return v.toISOString();
  if (typeof v === "object") {
    if (typeof v.Time === "string") return coerceTS(v.Time);
    const sec = v.seconds ?? v.Seconds ?? v.sec;
    const nsec = v.nanos ?? v.Nanos ?? 0;
    if (typeof sec === "number") return new Date(sec * 1000 + nsec / 1e6).toISOString();
  }
  return "";
}

export function itemKey(opts: {
  sessionId: string;
  type: string;
  id?: string;
  round?: string;
  ts?: string;
  name?: string;
  text?: string;
  delta?: boolean;
  idx?: number;
}): string {
  const sessionId = opts.sessionId || "";
  const type = opts.type || "system";
  const id = (opts.id || "").trim();
  const round = (opts.round || "").trim();
  const ts = opts.ts || "";
  const name = opts.name || "";
  const idx = opts.idx || 0;
  if (type === "assistant") {
    const rid = id || round;
    if (rid) return `${sessionId}:assistant:${rid}`;
  }
  if (id && round && id !== round) return `${sessionId}:${type}:${round}:${id}`;
  if (id) return `${sessionId}:${type}:${id}`;
  if (round) return `${sessionId}:${type}:${round}:${name}:${idx}`;
  return `${sessionId}:${ts}:${type}:${name}:${opts.delta ? "d" : "f"}:${idx}:${(opts.text || "").slice(0, 80)}`;
}

export function itemFromEvent(raw: any, idx = 0): Item {
  const src = unwrapEvent(raw);
  const p = payload(src);
  const type = str(pick(src, "type", "Type"), "system") as ItemType;
  const sessionId = str(pick(src, "session_id", "SessionID", "sessionId"));
  const ts = coerceTS(pick(src, "ts", "TS", "Ts"));
  const text = str(pick(p, "text", "Text", "content", "Content", "error", "Error"));
  const name = str(pick(p, "name", "Name", "action", "Action"));
  const delta = asBool(pick(p, "delta", "Delta"));
  const id = str(pick(p, "id", "Id", "ID"));
  const round = str(pick(p, "round", "Round", "round_id", "roundId", "RoundID"));
  return {
    key: itemKey({ sessionId, type, id, round, ts, name, text, delta, idx }),
    type,
    sessionId,
    source: str(pick(src, "source", "Source")),
    ts,
    text,
    name,
    delta,
    payload: { ...p, ...(id ? { id } : {}), ...(round ? { round } : {}), delta },
  };
}

export function eventFingerprint(raw: any): string {
  const it = itemFromEvent(raw);
  const id = str(it.payload?.id) || str(it.payload?.round);
  if (it.delta) return `${it.sessionId}|${it.ts}|${it.type}|d|${it.text}|${id}`;
  if (id) return `${it.sessionId}|${it.type}|${id}|${it.text}|${it.name}`;
  return `${it.sessionId}|${it.ts}|${it.type}|${it.text}|${it.name}|${it.key}`;
}

/** A new model round starts after these; do not merge assistant text across them. */
export function closesAssistant(type: ItemType): boolean {
  return (
    type === "user" ||
    type === "tool_call" ||
    type === "tool_result" ||
    type === "error" ||
    type === "turn_end" ||
    type === "compaction"
  );
}

export function isTranscriptNoise(ev: Item): boolean {
  if (ev.type === "system" || ev.type === "turn_end" || ev.type === "eval" || ev.type === "evolve") {
    return true;
  }
  if (ev.type !== "compaction") return false;
  const kind = str(ev.payload?.kind);
  if (kind === "checkpoint") return false;
  if (kind === "checkpoint_start") return true;
  const layers = ev.payload?.layers;
  if (Array.isArray(layers) && layers.some((x) => String(x) === "overflow")) return false;
  return true;
}

function assistantRound(ev: Item): string {
  return str(ev.payload?.id) || str(ev.payload?.round);
}

function openAssistantIndex(list: Item[]): number {
  for (let i = list.length - 1; i >= 0; i--) {
    const t = list[i].type;
    if (t === "assistant") return i;
    if (closesAssistant(t)) return -1;
  }
  return -1;
}

function assistantIndexFor(list: Item[], ev: Item): number {
  const rid = assistantRound(ev);
  if (rid) {
    for (let i = list.length - 1; i >= 0; i--) {
      if (list[i].type === "user") return -1;
      if (list[i].type === "assistant" && assistantRound(list[i]) === rid) return i;
    }
    return -1;
  }
  return openAssistantIndex(list);
}

function sameUserText(a: string, b: string): boolean {
  return a.trim() === b.trim();
}

function steerText(text: string): string {
  return text.replace(/^User steering \(apply now\):\s*/i, "").trim();
}

/** True if the open turn (after the last assistant) already has this user text. */
export function turnHasUserText(list: Item[], text: string): boolean {
  for (let i = list.length - 1; i >= 0; i--) {
    const it = list[i];
    if (it.type === "assistant" || it.type === "turn_end" || it.type === "error") return false;
    if (it.type === "user" && it.source !== "steer" && sameUserText(it.text, text)) return true;
  }
  return false;
}

function dropTrailingOptimistic(list: Item[], text: string, steer: boolean): Item[] {
  const out = list.slice();
  while (out.length) {
    const last = out[out.length - 1];
    if (last.type !== "user") break;
    if (steer) {
      if (last.source === "steer" && sameUserText(steerText(last.text), text)) {
        out.pop();
        continue;
      }
      break;
    }
    if (last.source === "ui" && sameUserText(last.text, text)) {
      out.pop();
      continue;
    }
    break;
  }
  return out;
}

/**
 * Fold a live or replayed event into the transcript.
 * Assistant deltas only join the in-progress bubble of the current round —
 * never the previous user turn (Codex/Claude API-round grouping).
 */
export function mergeItem(list: Item[], ev: Item): Item[] {
  if (!ev) return list;
  if (ev.type === "turn_end" || ev.type === "system" || ev.type === "eval" || ev.type === "evolve") {
    return list;
  }
  if (ev.type === "compaction" && isTranscriptNoise(ev)) return list;

  if (ev.type === "user") {
    const steer = ev.source === "steer" || ev.name === "steer";
    const incoming = steer ? steerText(ev.text) : ev.text;
    const row = steer ? { ...ev, source: "steer", text: incoming } : ev;
    if (row.source === "ui") {
      if (turnHasUserText(list, incoming)) return list;
      return [...list, row];
    }
    const next = dropTrailingOptimistic(list, incoming, steer);
    if (row.key && next.some((x) => x.key === row.key)) return next;
    return [...next, row];
  }

  if (ev.type === "assistant") {
    const idx = assistantIndexFor(list, ev);
    if (idx >= 0) {
      const cur = list[idx];
      if (ev.delta && cur.text && !cur.delta) return list;
      const text = ev.delta ? cur.text + (ev.text || "") : (ev.text || cur.text);
      const next = list.slice();
      next[idx] = {
        ...cur,
        text,
        delta: ev.delta,
        payload: { ...cur.payload, ...ev.payload, delta: ev.delta },
      };
      return next;
    }
    if (!ev.text && !ev.delta) return list;
    return [...list, { ...ev, delta: false }];
  }

  if (ev.key && list.some((x) => x.key === ev.key)) return list;
  return [...list, ev];
}

/** Keep optimistic composer bubbles that the JSONL seed has not caught up to. */
export function mergePendingUsers(seed: Item[], live: Item[]): Item[] {
  const pending = live.filter((x) => x.source === "ui" && x.type === "user");
  if (!pending.length) return seed;
  let out = seed;
  for (const p of pending) {
    if (turnHasUserText(out, p.text)) continue;
    out = [...out, p];
  }
  return out;
}

/** Replay live items onto a trajectory seed so a late seed cannot drop round 2. */
export function foldLiveIntoSeed(seed: Item[], live: Item[]): Item[] {
  if (!live.length) return seed;
  let out = seed;
  for (const it of live) out = mergeItem(out, it);
  return out;
}

export function lastUserTurns(items: Item[], users = 3): Item[] {
  if (users <= 0 || items.length === 0) return items;
  let n = 0;
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "user" && items[i].source !== "steer") {
      n++;
      if (n >= users) return items.slice(i);
    }
  }
  return items;
}

/** Drop the error chips at the tail so a resume can keep the same turn. */
export function dropTrailingErrors(items: Item[]): Item[] {
  let i = items.length;
  while (i > 0 && items[i - 1].type === "error") i--;
  return i === items.length ? items : items.slice(0, i);
}

export function replayEvents(raws: any[]): Item[] {
  let acc: Item[] = [];
  raws.forEach((raw, i) => {
    acc = mergeItem(acc, itemFromEvent(raw, i));
  });
  return acc;
}

function isTraceEnvelopeType(t: unknown): boolean {
  return typeof t === "string" && t.length > 0 && !t.startsWith("yoyo:");
}

/**
 * Peel only the Wails v3 `{ name, data }` envelope (or a 1-length args array).
 * Never walk an arbitrary `.data` / `.detail` once a trace event is in hand —
 * that used to swallow the second-round payload.
 */
export function unwrapEvent(e: any, depth = 0): any {
  if (e == null || depth > 4) return e;
  if (typeof e === "string") {
    const t = e.trim();
    if ((t.startsWith("{") && t.endsWith("}")) || (t.startsWith("[") && t.endsWith("]"))) {
      try {
        return unwrapEvent(JSON.parse(t), depth + 1);
      } catch {
        return e;
      }
    }
    return e;
  }
  if (Array.isArray(e)) {
    if (e.length === 1) return unwrapEvent(e[0], depth + 1);
    const hit = e.find((x) => x && typeof x === "object" && isTraceEnvelopeType(x.type ?? x.Type));
    return hit ? unwrapEvent(hit, depth + 1) : e;
  }
  if (typeof e !== "object") return e;
  if (isTraceEnvelopeType(e.type ?? e.Type)) return e;
  const named = typeof e.name === "string";
  if (named && e.data != null) return unwrapEvent(e.data, depth + 1);
  return e;
}
