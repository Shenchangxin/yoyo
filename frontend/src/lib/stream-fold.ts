import { asBool, num, pick, str } from "./normalize";
import type { Item, ItemType } from "./protocol";
import { sameUserTurnText } from "./mentions";
import { ConversationFold } from "./conversation-fold";

/**
 * 拥有：把原始事件、实时通知和增量折成转录条目，并限制界面上保留的热窗口。
 * 不拥有：会话订阅和序号游标，那在 stream；块组装，那在 stream-chunk。
 */

/** 取出事件载荷。没有对象载荷时为空对象。raw 传给 pick。 */
function payload(raw: any): Record<string, any> {
  const p = pick(raw, "payload", "Payload");
  return p && typeof p === "object" ? p : {};
}

/**
 * coerceTS 把各种时间收成 ISO 字符串。零值和 Go 的零时间变成空串。
 * v 判断是否为空，用 instanceof 看是不是 Date，读取 Time、seconds、Seconds、sec、nanos、Nanos，看开头，调用 getTime，调用 toISOString，用 typeof 看类型，传给 Number.isFinite，传给 Date。
 */
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

/** itemKey 一条转录的稳定键。助手和推理按回合并到同一条，增量用序号区分。opts 读取 sessionId、type、id、round、ts、name 等。 */
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
  if (type === "assistant_stream") {
    const rid = id || round;
    return `${sessionId}:${type}:${rid}:${idx}:${opts.delta ? "d" : "f"}`;
  }
  if (id && round && id !== round) return `${sessionId}:${type}:${round}:${id}`;
  if (id) return `${sessionId}:${type}:${id}`;
  if (round) return `${sessionId}:${type}:${round}:${name}:${idx}`;
  return `${sessionId}:${ts}:${type}:${name}:${opts.delta ? "d" : "f"}:${idx}:${(opts.text || "").slice(0, 80)}`;
}

/** 会带大段正文的工具调用。界面上把超长参数收成摘要，避免整段进热窗口。 */
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

/** 超长的写文件类参数只留路径和开头，并记下原始字节数。type 与 tool_call 比较是否不等。name 传给 HEAVY_CALLS.has。p 读取 arguments、path，展开传递。 */
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

/** HOT_TRANSCRIPT_TURNS 渲染器热窗口。种子、发送和实时合并都停在这个用户回合数以内。 */
export const HOT_TRANSCRIPT_TURNS = 8;
/** BROWSE_TRANSCRIPT_TURNS 向上滚动最多展开到的用户回合数。发送时收回热窗口。 */
export const BROWSE_TRANSCRIPT_TURNS = 16;
/** UI_TEXT_CAP 界面上一条非增量正文的长度上限。 */
export const UI_TEXT_CAP = 24_000;
/** UI_RESULT_CAP 界面上一条工具结果的长度上限。 */
export const UI_RESULT_CAP = 1_200;

/** 超过上限的正文截断并加省略号。s 读取 length，截取，为假时走另一支，有的分支原样返回。n 小于等于 s.length，传给 slice。 */
function capText(s: string, n = UI_TEXT_CAP): string {
  if (!s || s.length <= n) return s;
  return `${s.slice(0, n)}\n…`;
}

/** itemFromEvent 把一条原始事件收成条目。收束后的流磁带丢掉。非增量正文按种类截断。raw 传给 unwrapEvent。idx 写入字段 idx。 */
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
  if (slim.stream != null) {
    slim = { ...slim };
    delete slim.stream;
  }
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

/** eventFingerprint 去重指纹。增量带上序号或文本尾部，收束条目带上 id 和键。raw 传给 unwrapEvent，传给 itemFromEvent。 */
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

/** closesAssistant 这些种类之后开始新的模型回合，助手正文不能跨过去合并。type 是 user、tool_call、tool_result、error、turn_end 或 compaction 时返回真。 */
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

/** isTranscriptNoise 不进转录正文的事件。检查点压缩留下，开始标记和没有溢出层的压缩视为噪声。ev 读取 type、payload。 */
export function isTranscriptNoise(ev: Item): boolean {
  if (
    ev.type === "system" ||
    ev.type === "turn_end" ||
    ev.type === "inbox" ||
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

/** 两条用户正文是否同一回合。比较前先按提及规则归一。a 传给 sameUserTurnText。b 传给 sameUserTurnText。 */
function sameUserText(a: string, b: string): boolean {
  return sameUserTurnText(a, b);
}

/** turnHasUserText 当前打开的回合（最后一条助手之前）是否已经有这段用户正文。list 读取 length，按下标读取。text 传给 sameUserText。 */
export function turnHasUserText(list: Item[], text: string): boolean {
  for (let i = list.length - 1; i >= 0; i--) {
    const it = list[i];
    if (it.type === "assistant" || it.type === "turn_end" || it.type === "error") return false;
    if (it.type === "user" && it.source !== "steer" && sameUserText(it.text, text)) return true;
  }
  return false;
}

/**
 * mergeItem 把一条实时或重放事件折进转录。
 * 助手增量并入当前回合里进行中的尝试。
 * list 传给 ConversationFold.from。
 * ev 传给 accept。
 */
export function mergeItem(list: Item[], ev: Item): Item[] {
  return foldTurnErrors(ConversationFold.from(list).accept(ev));
}

/** 用户消息、助手、推理和工具是回合有进展。引导备注不算。it 的 type 为 user 时返回真。 */
function isTurnProgress(it: Item): boolean {
  if (it.type === "user" && it.source !== "steer") return true;
  return it.type === "assistant" || it.type === "reasoning" || it.type === "tool_call" || it.type === "tool_result";
}

/**
 * foldTurnErrors 错误卡片是回合末尾的芯片，不是一封信。
 * 最近一次进展之后只留最后一张。重试或新的用户消息会盖过停止、达到回合上限和供应商失败。
 * list 读取 length，做过滤，按下标读取，作为分支结果，有的分支原样返回。
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

/** dropTurnErrors 操作者恢复回合时，下一 token 到来之前先去掉错误芯片。list 读取 length，做过滤，作为分支结果。 */
export function dropTurnErrors(list: Item[]): Item[] {
  const next = list.filter((it) => it.type !== "error");
  return next.length === list.length ? list : next;
}

/** dropTrailingErrors 去掉末尾错误芯片，恢复时仍是同一回合。items 传给 dropTurnErrors。 */
export function dropTrailingErrors(items: Item[]): Item[] {
  return dropTurnErrors(items);
}

/** mergePendingUsers 保留 JSONL 种子还没追上的乐观作曲气泡。正文已经在种子里的不再追加。seed 原样返回。live 做过滤。 */
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

/** foldLiveIntoSeed 把实时条目重放到轨迹种子上，迟到的种子不会丢掉后面的回合。seed 传给 ConversationFold.from。live 读取 length，用来遍历。 */
export function foldLiveIntoSeed(seed: Item[], live: Item[]): Item[] {
  if (!live.length) return seed;
  const fold = ConversationFold.from(seed);
  for (const it of live) fold.accept(it);
  return foldTurnErrors(fold.items());
}

/**
 * withLiveTail 操作者在看更早的窗口时，仍把进行中的热包留在末尾。
 * 身份用载荷 id、序号或键。整份 JSONL 不倒进来。
 * browse 调用 some，做映射，作为分支结果，传给 foldLiveIntoSeed，展开传递。
 * live 读取 length，截取，按下标读取，传给 foldLiveIntoSeed。
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

/** 用户回合的身份：载荷 id，否则序号，否则键。it 读取 payload、seq、key。 */
function liveTurnId(it: Item): string {
  const id = String(it.payload?.id || "").trim();
  if (id) return id;
  if (typeof it.seq === "number" && it.seq > 0) return `seq:${it.seq}`;
  return it.key || "";
}

/** lastUserTurns 只留末尾若干个非引导用户回合及其后的条目。items 读取 length，截取，按下标读取，作为分支结果，有的分支原样返回。users 小于等于 0，大于等于 n。 */
export function lastUserTurns(items: Item[], users = 3): Item[] {
  if (users <= 0 || items.length === 0) return items;
  let n = 0;
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "user" && items[i].source !== "steer") {
      n++;
      if (n >= users) {
        return i === 0 ? items : items.slice(i);
      }
    }
  }
  return items;
}

/** userTurnCount 非引导的用户回合数。items 用来遍历。 */
export function userTurnCount(items: Item[]): number {
  let n = 0;
  for (const it of items) {
    if (it.type === "user" && it.source !== "steer") n++;
  }
  return n;
}

/** replayEvents 按顺序把原始事件折成转录，并收拢错误芯片。raws 为假时改用后备值，逐项处理。 */
export function replayEvents(raws: any[]): Item[] {
  const fold = new ConversationFold();
  (raws || []).forEach((raw, i) => {
    fold.accept(itemFromEvent(raw, i));
  });
  return foldTurnErrors(fold.items());
}

/**
 * attachStreamFrame 把流帧上的种类、序号和块写进条目载荷。助手流或系统条目会改成 assistant_stream。
 * item 读取 type、payload，为假时走另一支，展开传递。
 * frame 读取 kind、attempt_id、revision、index、index_from、chunk 等，为假时走另一支。
 */
export function attachStreamFrame(item: Item, frame?: StreamFrame): Item {
  if (!item || !frame) return item;
  if (item.type !== "assistant_stream" && frame.kind !== "start" && frame.kind !== "chunk" && frame.kind !== "end") {
    return item;
  }
  const p = { ...item.payload };
  if (frame.kind) p.kind = frame.kind;
  if (frame.attempt_id) p.attempt_id = frame.attempt_id;
  if (frame.revision) p.revision = frame.revision;
  if (frame.index != null) p.index = frame.index;
  if (frame.index_from != null) p.index_from = frame.index_from;
  if (frame.chunk) p.chunk = frame.chunk;
  if (frame.turn) p.turn = frame.turn;
  if (frame.step) p.step = frame.step;
  if (frame.outcome) p.outcome = frame.outcome;
  const type = item.type === "assistant_stream" || item.type === "system" ? "assistant_stream" as const : item.type;
  return { ...item, type, payload: p };
}

/** 类型字符串非空且不是 yoyo: 前缀时，视为追踪信封。t 读取 length，看开头，用 typeof 看类型。 */
function isTraceEnvelopeType(t: unknown): boolean {
  return typeof t === "string" && t.length > 0 && !t.startsWith("yoyo:");
}

/** StreamFrame 一条助手流帧。index_from 到 index 是这段合并增量覆盖的序号。正文在 chunk 里。 */
export type StreamFrame = {
  kind?: string;
  attempt_id?: string;
  id?: string;
  round?: string;
  revision?: number;
  index?: number;
  index_from?: number;
  block?: string;
  delta?: string;
  chunk?: Record<string, any>;
  turn?: number;
  step?: number;
  outcome?: string;
  committed_seq?: number;
};

/** LiveNotice 一条实时通知：全局序号、会话、类型、内联事件和可选流帧。 */
export type LiveNotice = {
  seq: number;
  sessionId: string;
  type?: string;
  event?: any;
  frame?: StreamFrame;
};

/** noticeCarriesInline 这条通知自己就能按序应用。内联事件，或带开始、块、结束、增量帧，都算。 */
export function noticeCarriesInline(notice: LiveNotice): boolean {
  if (notice.event) return true;
  const kind = notice.frame?.kind;
  return kind === "start" || kind === "chunk" || kind === "end" || kind === "delta";
}

/** 从帧的 chunk 取出这段正文。工具参数增量在 argumentsDelta。 */
function frameChunkText(chunk: Record<string, any> | undefined): string {
  if (!chunk) return "";
  if (typeof chunk.text === "string" && chunk.text) return chunk.text;
  if (typeof chunk.argumentsDelta === "string") return chunk.argumentsDelta;
  if (typeof chunk.arguments_delta === "string") return chunk.arguments_delta;
  return "";
}

/**
 * eventFromNotice 还原出可折进转录的事件。已有内联事件就用它。
 * 只有帧时，用帧上的一块正文拼出最小事件，避免再回拉整段环。
 * notice 读取 event、frame、type、seq、sessionId。
 */
export function eventFromNotice(notice: LiveNotice): any | undefined {
  if (notice.event) return notice.event;
  const f = notice.frame;
  const type = notice.type;
  if (!f || !type || !noticeCarriesInline(notice)) return undefined;
  const payload: Record<string, any> = {};
  if (f.kind && f.kind !== "delta") payload.kind = f.kind;
  if (f.attempt_id) payload.attempt_id = f.attempt_id;
  if (f.id) payload.id = f.id;
  if (f.round) payload.round = f.round;
  if (!payload.id && f.round) payload.id = f.round;
  if (!payload.id && type !== "assistant_stream" && f.attempt_id) payload.id = f.attempt_id;
  if (!payload.round && payload.id && type !== "assistant_stream") payload.round = payload.id;
  if (f.revision) payload.revision = f.revision;
  if (f.index != null) payload.index = f.index;
  if (f.index_from != null) payload.index_from = f.index_from;
  if (f.turn) payload.turn = f.turn;
  if (f.step) payload.step = f.step;
  if (f.outcome) payload.outcome = f.outcome;
  if (f.block) payload.block = f.block;
  if (f.chunk) payload.chunk = f.chunk;
  const text = frameChunkText(f.chunk) || f.delta || "";
  if (type === "tool_result") {
    payload.delta = true;
    payload.content = text;
  } else if (type === "assistant" || type === "reasoning") {
    payload.delta = true;
    payload.text = text;
  } else if (type === "assistant_stream") {
    payload.delta = true;
  }
  return { seq: notice.seq, session_id: notice.sessionId, type, payload };
}

/** 从通知或嵌套事件上读出流帧。没有帧对象时返回空。raw 读取 frame、Frame，为空时不读。 */
function readStreamFrame(raw: any): StreamFrame | undefined {
  const src = raw?.frame ?? raw?.Frame;
  if (!src || typeof src !== "object") return undefined;
  const hasIndex = src.index != null || src.Index != null;
  const hasFrom = src.index_from != null || src.IndexFrom != null || src.indexFrom != null;
  const index = num(src.index ?? src.Index);
  const from = num(src.index_from ?? src.IndexFrom ?? src.indexFrom);
  const committed = num(src.committed_seq ?? src.CommittedSeq ?? src.committedSeq);
  return {
    kind: str(src.kind ?? src.Kind) || undefined,
    attempt_id: str(src.attempt_id ?? src.AttemptID ?? src.attemptId) || undefined,
    id: str(src.id ?? src.ID) || undefined,
    round: str(src.round ?? src.Round) || undefined,
    revision: num(src.revision ?? src.Revision) || undefined,
    index: hasIndex ? index : undefined,
    index_from: hasFrom ? from : hasIndex ? index : undefined,
    block: str(src.block ?? src.Block) || undefined,
    delta: str(src.delta ?? src.Delta) || undefined,
    chunk: src.chunk ?? src.Chunk,
    turn: num(src.turn ?? src.Turn) || undefined,
    step: num(src.step ?? src.Step) || undefined,
    outcome: str(src.outcome ?? src.Outcome) || undefined,
    committed_seq: committed || undefined,
  };
}

/** parseLiveNotice 剥开通知或回拉信封。type 本身不当成事件。嵌套事件优先。raw 传给 unwrapEvent。 */
export function parseLiveNotice(raw: any): LiveNotice {
  const e = unwrapEvent(raw);
  if (e == null || typeof e !== "object") return { seq: 0, sessionId: "", event: e };
  const seq = num(e.seq ?? e.Seq);
  const nested = e.event ?? e.Event;
  const sessionId = str(pick(e, "session_id", "SessionID", "sessionId") || pick(nested, "session_id", "SessionID", "sessionId"));
  const type = str(pick(nested, "type", "Type") || pick(e, "type", "Type")) || undefined;
  const frame = readStreamFrame(e) || readStreamFrame(nested);
  if (nested && typeof nested === "object" && (nested.type || nested.Type || nested.payload || nested.Payload)) {
    return { seq, sessionId, type, event: nested, frame };
  }
  if (e.payload != null || e.Payload != null) {
    return { seq, sessionId, type, event: e, frame };
  }
  return { seq, sessionId, type, frame };
}

/**
 * unwrapEvent 只剥 Wails v3 的 `{ name, data }` 信封，或长度为 1 的参数数组。
 * 已经拿到追踪事件后，不再顺着任意的 data 或 detail 往下走，那会吞掉下一回合的载荷。
 * e 去掉首尾空白，取第一段，判断是否为空，读取 length、type、Type、name、data，调用 find，用 typeof 看类型，作为分支结果，传给 unwrapEvent。
 * depth 大于 4，加上 1。
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

/** eventText 从剥开后的事件取出纯文本。对象没有单独的文本字段时返回空串。raw 传给 unwrapEvent。 */
export function eventText(raw: any): string {
  const d = unwrapEvent(raw);
  if (typeof d === "string") return d.trim();
  if (typeof d === "number" || typeof d === "boolean") return String(d);
  return "";
}
