import { asBool, num, pick, str } from "./normalize";
import type { Item, ItemType } from "./protocol";
import { sameUserTurnText } from "./mentions";
import { pruneHotTurns } from "./transcript-layout";

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
  if (type === "assistant" || type === "reasoning") {
    const rid = id || round;
    if (rid) return `${sessionId}:${type}:${rid}`;
  }
  if (id && round && id !== round) return `${sessionId}:${type}:${round}:${id}`;
  if (id) return `${sessionId}:${type}:${id}`;
  if (round) return `${sessionId}:${type}:${round}:${name}:${idx}`;
  return `${sessionId}:${ts}:${type}:${name}:${opts.delta ? "d" : "f"}:${idx}:${(opts.text || "").slice(0, 80)}`;
}

const HEAVY_CALLS = new Set([
  "write_file",
  "create_file",
  "apply_patch",
  "str_replace",
  "edit_file",
  "office_create",
  "office_edit",
  "cite_sources",
]);

function slimToolCallPayload(type: string, name: string, p: Record<string, any>): Record<string, any> {
  if (type !== "tool_call" || !HEAVY_CALLS.has(name)) return p;
  const args = p.arguments;
  if (typeof args !== "string" || args.length < 1600) return p;
  try {
    const m = JSON.parse(args) as Record<string, unknown>;
    let changed = false;
    for (const key of ["content", "patch", "body", "old_str", "new_str"]) {
      const v = m[key];
      if (typeof v !== "string" || v.length <= 800) continue;
      m[key] = `[elided ${key} ${v.length} chars]\n${v.slice(0, 120)}`;
      changed = true;
    }
    if (!changed) return p;
    const path = String(m.path || p.path || "");
    return { ...p, arguments: JSON.stringify(m), bytes: args.length, ...(path ? { path } : {}) };
  } catch {
    const m = args.match(/"path"\s*:\s*"((?:\\.|[^"\\])*)"/);
    const path = m ? m[1].replace(/\\"/g, '"') : "";
    return { ...p, arguments: JSON.stringify(path ? { path } : {}), bytes: args.length, ...(path ? { path } : {}) };
  }
}

/** Renderer hot window. Seed, send, and live merge stay inside this. */
export const HOT_TRANSCRIPT_TURNS = 8;
/** Scroll-up may grow the window this far; send snaps back to HOT. */
export const BROWSE_TRANSCRIPT_TURNS = 16;
export const UI_TEXT_CAP = 24_000;
export const UI_RESULT_CAP = 1_200;

function capText(s: string, n = UI_TEXT_CAP): string {
  if (!s || s.length <= n) return s;
  return `${s.slice(0, n)}\n…`;
}

export function itemFromEvent(raw: any, idx = 0): Item {
  const src = unwrapEvent(raw);
  const p = payload(src);
  const type = str(pick(src, "type", "Type"), "system") as ItemType;
  const sessionId = str(pick(src, "session_id", "SessionID", "sessionId"));
  const ts = coerceTS(pick(src, "ts", "TS", "Ts"));
  const rawText = str(pick(p, "text", "Text", "content", "Content", "error", "Error"));
  const name = str(pick(p, "name", "Name", "action", "Action"));
  const delta = asBool(pick(p, "delta", "Delta"));
  const id = str(pick(p, "id", "Id", "ID"));
  const round = str(pick(p, "round", "Round", "round_id", "roundId", "RoundID"));
  let slim = slimToolCallPayload(type, name, p);
  const text = delta ? rawText : capText(rawText, type === "tool_result" ? UI_RESULT_CAP : UI_TEXT_CAP);
  if (!delta && type === "tool_result" && typeof slim.content === "string" && slim.content.length > UI_RESULT_CAP) {
    slim = { ...slim, content: capText(slim.content, UI_RESULT_CAP), bytes: slim.content.length };
  }
  return {
    key: itemKey({ sessionId, type, id, round, ts, name, text, delta, idx }),
    type,
    sessionId,
    source: str(pick(src, "source", "Source")),
    ts,
    text,
    name,
    delta,
    payload: { ...slim, ...(id ? { id } : {}), ...(round ? { round } : {}), delta },
    seq: num(pick(src, "seq", "Seq")) || undefined,
  };
}

export function eventFingerprint(raw: any): string {
  const src = unwrapEvent(raw);
  const seq = num(pick(src, "seq", "Seq"));
  const it = itemFromEvent(raw);
  const id = str(it.payload?.id) || str(it.payload?.round);
  if (it.delta) {
    if (seq) return `${it.sessionId}|${it.type}|d|${id}|${seq}`;
    return `${it.sessionId}|${it.type}|d|${id}|${it.text.length}|${it.text.slice(-16)}`;
  }
  if (id) return `${it.sessionId}|${it.type}|${id}|${it.name}|${it.key}`;
  return `${it.sessionId}|${it.ts}|${it.type}|${it.name}|${it.key}`;
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
  if (
    ev.type === "system" ||
    ev.type === "turn_end" ||
    ev.type === "eval" ||
    ev.type === "evolve" ||
    ev.type === "file_change" ||
    ev.type === "ask_user"
  ) {
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

function sameUserText(a: string, b: string): boolean {
  return sameUserTurnText(a, b);
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

function textDeltaIndex(list: Item[], ev: Item): number {
  const rid = assistantRound(ev);
  if (rid) {
    for (let i = list.length - 1; i >= 0; i--) {
      if (list[i].type === "user") return -1;
      if (list[i].type === ev.type && assistantRound(list[i]) === rid) return i;
    }
    return -1;
  }
  if (ev.type === "assistant") return openAssistantIndex(list);
  for (let i = list.length - 1; i >= 0; i--) {
    if (list[i].type === ev.type) return i;
    if (closesAssistant(list[i].type)) return -1;
  }
  return -1;
}

function roundNum(id: string): number {
  const m = /:r(\d+)$/.exec(id);
  return m ? Number(m[1]) : 0;
}

function lastOpenAssistant(list: Item[]): Item | null {
  for (let i = list.length - 1; i >= 0; i--) {
    if (list[i].type === "user" && list[i].source !== "steer") return null;
    if (list[i].type === "assistant") return list[i];
  }
  return null;
}

function foldTextDelta(list: Item[], ev: Item): Item[] {
  const idx = textDeltaIndex(list, ev);
  if (idx >= 0) {
    const cur = list[idx];
    if (ev.delta && cur.text && !cur.delta) return list;
    const text = ev.delta ? cur.text + (ev.text || "") : (ev.text || cur.text);
    const next = list.slice();
    next[idx] = {
      ...cur,
      text,
      delta: ev.delta,
      payload: { ...cur.payload, ...ev.payload, text, delta: ev.delta },
    };
    return next;
  }
  if (!ev.text && !ev.delta) return list;
  if (ev.delta) {
    const last = lastOpenAssistant(list);
    if (last) {
      const incoming = roundNum(assistantRound(ev));
      const have = roundNum(assistantRound(last));
      if (incoming > 0 && have > 0 && incoming < have) return list;
    }
  }
  return [...list, ev];
}

function foldToolResult(list: Item[], ev: Item): Item[] {
  const idx = ev.key ? list.findIndex((x) => x.key === ev.key) : -1;
  if (idx >= 0) {
    const cur = list[idx];
    const curBody = String(cur.payload?.content ?? cur.text ?? "");
    if (ev.delta && curBody && !cur.delta) return list;
    const add = String(ev.payload?.content ?? ev.text ?? "");
    const body = ev.delta ? curBody + add : (add || curBody);
    const next = list.slice();
    next[idx] = {
      ...cur,
      ...ev,
      text: body,
      delta: ev.delta,
      payload: { ...cur.payload, ...ev.payload, content: body, delta: ev.delta },
    };
    return next;
  }
  return [...list, ev];
}

/**
 * Fold a live or replayed event into the transcript.
 * Assistant deltas only join the in-progress bubble of the current round —
 * never the previous user turn (Codex/Claude API-round grouping).
 */
export function mergeItem(list: Item[], ev: Item): Item[] {
  return foldTurnErrors(mergeItemCore(list, ev));
}

function mergeItemCore(list: Item[], ev: Item): Item[] {
  if (!ev) return list;
  if (ev.type === "turn_end" || ev.type === "system" || ev.type === "eval" || ev.type === "evolve" || ev.type === "file_change" || ev.type === "ask_user") {
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

  if (ev.type === "assistant" || ev.type === "reasoning") {
    return foldTextDelta(list, ev);
  }

  if (ev.type === "tool_result") {
    return foldToolResult(list, ev);
  }

  if (ev.type === "tool_call") {
    return foldToolCall(list, ev);
  }

  if (ev.key && list.some((x) => x.key === ev.key)) return list;
  return [...list, ev];
}

function foldToolCall(list: Item[], ev: Item): Item[] {
  const idx = ev.key ? list.findIndex((x) => x.key === ev.key) : -1;
  if (idx < 0) return [...list, ev];
  const cur = list[idx];
  const curProg = !!cur.payload?.progress;
  const nextProg = !!ev.payload?.progress;
  if (nextProg && !curProg) return list;
  const payload = { ...cur.payload, ...ev.payload };
  if (!nextProg) delete payload.progress;
  const next = list.slice();
  next[idx] = { ...cur, ...ev, payload };
  return next;
}

function isTurnProgress(it: Item): boolean {
  if (it.type === "user" && it.source !== "steer") return true;
  return it.type === "assistant" || it.type === "reasoning" || it.type === "tool_call" || it.type === "tool_result";
}

/**
 * Error cards are the terminal chip of a turn, not a letter. Keep at most
 * the last one after the latest progress; a retry or a new user message
 * supersedes Stopped / max-turns / provider failures.
 */
export function foldTurnErrors(list: Item[]): Item[] {
  if (!list.length) return list;
  let lastProgress = -1;
  let hasError = false;
  for (let i = 0; i < list.length; i++) {
    if (list[i].type === "error") hasError = true;
    else if (isTurnProgress(list[i])) lastProgress = i;
  }
  if (!hasError) return list;
  let keep = -1;
  for (let i = lastProgress + 1; i < list.length; i++) {
    if (list[i].type === "error") keep = i;
  }
  const next = list.filter((it, i) => it.type !== "error" || i === keep);
  return next.length === list.length ? list : next;
}

/** Operator resumed the turn: the chip is gone before the next token. */
export function dropTurnErrors(list: Item[]): Item[] {
  const next = list.filter((it) => it.type !== "error");
  return next.length === list.length ? list : next;
}

/** Drop the error chips at the tail so a resume can keep the same turn. */
export function dropTrailingErrors(items: Item[]): Item[] {
  return dropTurnErrors(items);
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

/**
 * Keep the live hot pack visible while the operator browses an older window.
 * Identity is payload.id / seq / key — never dump the full JSONL.
 */
export function withLiveTail(browse: Item[], live: Item[]): Item[] {
  if (!live.length) return browse;
  let lastLiveUser = -1;
  for (let i = live.length - 1; i >= 0; i--) {
    if (live[i].type === "user" && live[i].source !== "steer") {
      lastLiveUser = i;
      break;
    }
  }
  if (lastLiveUser < 0) return foldLiveIntoSeed(browse, live);
  const liveKey = liveTurnId(live[lastLiveUser]);
  const inBrowse = browse.some((it) => it.type === "user" && it.source !== "steer" && liveTurnId(it) === liveKey);
  if (inBrowse) return foldLiveIntoSeed(browse, live);
  const have = new Set(browse.map((x) => x.key));
  const tail = live.slice(lastLiveUser).filter((x) => !have.has(x.key));
  return tail.length ? [...browse, ...tail] : browse;
}

function liveTurnId(it: Item): string {
  const id = String(it.payload?.id || "").trim();
  if (id) return id;
  if (typeof it.seq === "number" && it.seq > 0) return `seq:${it.seq}`;
  return it.key || "";
}

export function lastUserTurns(items: Item[], users = 3): Item[] {
  if (users <= 0 || items.length === 0) return pruneHotTurns(items);
  let n = 0;
  let sliced = items;
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "user" && items[i].source !== "steer") {
      n++;
      if (n >= users) {
        sliced = i === 0 ? items : items.slice(i);
        break;
      }
    }
  }
  return pruneHotTurns(sliced);
}

export function userTurnCount(items: Item[]): number {
  let n = 0;
  for (const it of items) {
    if (it.type === "user" && it.source !== "steer") n++;
  }
  return n;
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

export type LiveNotice = {
  seq: number;
  sessionId: string;
  event?: any;
};

/** Peel a notify/pull envelope without treating `type` as the event itself. */
export function parseLiveNotice(raw: any): LiveNotice {
  const e = unwrapEvent(raw);
  if (e == null || typeof e !== "object") return { seq: 0, sessionId: "", event: e };
  const seq = num(e.seq ?? e.Seq);
  const nested = e.event ?? e.Event;
  const sessionId = str(pick(e, "session_id", "SessionID", "sessionId") || pick(nested, "session_id", "SessionID", "sessionId"));
  if (nested && typeof nested === "object" && (nested.type || nested.Type || nested.payload || nested.Payload)) {
    return { seq, sessionId, event: nested };
  }
  if (e.payload != null || e.Payload != null) {
    return { seq, sessionId, event: e };
  }
  return { seq, sessionId };
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
