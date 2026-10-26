/**
 * 拥有：把上下文用量展开成分片，以及从追踪里取出模型调用、缓存命中和回合耗时。
 * 不拥有：用量的采集，以及用量条的渲染。
 */

import type { ContextUsage, SessionTrace, TraceEvent } from "./protocol";

/** CTX_SLICE_IDS 分片顺序：系统、规则、目录、技能、工具、记忆、今日、笔记、对话、空闲。 */
export const CTX_SLICE_IDS = ["system", "rules", "catalog", "skills", "tools", "memory", "today", "notes", "chat", "free"] as const;
/** CtxSliceId CTX_SLICE_IDS 里的一个 id。 */
export type CtxSliceId = (typeof CTX_SLICE_IDS)[number];

/** CtxSlice 一片的 id 和 token 数。 */
export type CtxSlice = {
  id: CtxSliceId;
  tokens: number;
};

/** CtxBreakdown 展开后的用量。pct 是已用占容量的百分比。返回里的 prefix 是系统片，dynamic 是笔记片。slices 按 CTX_SLICE_IDS 排列。 */
export type CtxBreakdown = {
  used: number;
  capacity: number;
  pct: number;
  prefix: number;
  schema: number;
  dynamic: number;
  chat: number;
  free: number;
  rules: number;
  catalog: number;
  skills: number;
  memory: number;
  today: number;
  notes: number;
  providerPrompt: number;
  cachedTokens: number;
  cacheReported: boolean;
  cacheStable: boolean;
  trigger: string;
  hydrated: number;
  elided: number;
  layers: string[];
  note: string;
  slices: CtxSlice[];
};

/** estimateTokens 与 runtime.estimateTokens 同一启发：空串为 0，否则为 UTF-8 字节长度加 3 后整除 4。 */
export function estimateTokens(s: string): number {
  if (!s) return 0;
  return Math.floor((new TextEncoder().encode(s).length + 3) / 4);
}

/** contextBreakdown 已用等于 max(0, tokens) 加 max(0, draftTokens)。容量为大于 0 的 window，否则大于 0 的 budget，否则非 0 的 fallbackWindow，否则已用大于 0 时等于已用，否则为 0。slicesKnown 为假时，规则、目录、技能、记忆、今日为 0，笔记取 max(0, dynamicTokens)，系统取 max(0, prefixTokens)，对话取 max(0, tokens - prefixTokens - schemaTokens - dynamicTokens) 再加草稿。slicesKnown 为真时，系统取 identityTokens，笔记取 notesTokens，对话取 chatTokens 再加草稿，这些都先与 0 取大。空闲为 max(0, 容量 - 已用)。容量大于 0 时百分比为 round(已用 / 容量 * 100) 且不超过 100，否则为 0。 */
export function contextBreakdown(ctx: ContextUsage | undefined, fallbackWindow = 0, draftTokens = 0): CtxBreakdown {
  const prefix = Math.max(0, ctx?.prefixTokens || 0);
  const schema = Math.max(0, ctx?.schemaTokens || 0);
  const dynamic = Math.max(0, ctx?.dynamicTokens || 0);
  const baseUsed = Math.max(0, ctx?.tokens || 0);
  const draft = Math.max(0, draftTokens);
  const used = baseUsed + draft;
  const window = ctx?.window && ctx.window > 0 ? ctx.window : 0;
  const budget = ctx?.budget && ctx.budget > 0 ? ctx.budget : 0;
  const capacity = window || budget || fallbackWindow || (used > 0 ? used : 0);
  const detailed = !!ctx?.slicesKnown;
  const rules = detailed ? Math.max(0, ctx?.rulesTokens || 0) : 0;
  const catalog = detailed ? Math.max(0, ctx?.catalogTokens || 0) : 0;
  const skills = detailed ? Math.max(0, ctx?.skillBodyTokens || 0) : 0;
  const memory = detailed ? Math.max(0, ctx?.memoryTokens || 0) : 0;
  const today = detailed ? Math.max(0, ctx?.todayTokens || 0) : 0;
  const notes = detailed ? Math.max(0, ctx?.notesTokens || 0) : dynamic;
  const system = detailed ? Math.max(0, ctx?.identityTokens || 0) : prefix;
  const chat = (detailed ? Math.max(0, ctx?.chatTokens || 0) : Math.max(0, baseUsed - prefix - schema - dynamic)) + draft;
  const free = Math.max(0, capacity - used);
  const pct = capacity > 0 ? Math.min(100, Math.round((used / capacity) * 100)) : 0;
  return {
    used,
    capacity,
    pct,
    prefix: system,
    schema,
    dynamic: notes,
    chat,
    free,
    rules,
    catalog,
    skills,
    memory,
    today,
    notes,
    providerPrompt: Math.max(0, ctx?.providerPrompt || 0),
    cachedTokens: Math.max(0, ctx?.cachedTokens || 0),
    cacheReported: !!ctx?.cacheReported,
    cacheStable: !!ctx?.cacheStable,
    trigger: ctx?.trigger || "",
    hydrated: Math.max(0, ctx?.hydrated || 0),
    elided: Math.max(0, ctx?.elided || 0),
    layers: ctx?.layers || [],
    note: ctx?.note || "",
    slices: [
      { id: "system", tokens: system },
      { id: "rules", tokens: rules },
      { id: "catalog", tokens: catalog },
      { id: "skills", tokens: skills },
      { id: "tools", tokens: schema },
      { id: "memory", tokens: memory },
      { id: "today", tokens: today },
      { id: "notes", tokens: notes },
      { id: "chat", tokens: chat },
      { id: "free", tokens: free },
    ],
  };
}

/** cacheHitPct reported 为假时返回空。prompt 不大于 0 时，reported 为真则返回 0，否则返回空。其余为 min(100, round(max(0, cached) / prompt * 1000) / 10)。 */
export function cacheHitPct(cached: number, prompt: number, reported?: boolean): number | null {
  if (reported === false) return null;
  if (prompt <= 0) return reported ? 0 : null;
  return Math.min(100, Math.round((Math.max(0, cached) / prompt) * 1000) / 10);
}

/** ModelCall 一次模型调用。promptTokens 没有时用事件上的 tokens。cachedTokens 大于 0 也视为报告过缓存。 */
export type ModelCall = {
  round: string;
  ts: string;
  promptTokens: number;
  completionTokens: number;
  cachedTokens: number;
  elapsedMs: number;
  ttftMs: number;
  cacheReported: boolean;
};

/** modelCallsOf 按事件顺序取出 type 为 model_usage 的调用。 */
export function modelCallsOf(trace: SessionTrace | null | undefined): ModelCall[] {
  const out: ModelCall[] = [];
  for (const ev of trace?.events || []) {
    if (ev.type !== "model_usage") continue;
    out.push(modelCallOf(ev));
  }
  return out;
}

/** 提示 token 取 promptTokens，没有则取 tokens。cacheReported 为真，或 cachedTokens 大于 0。 */
function modelCallOf(ev: TraceEvent): ModelCall {
  const prompt = ev.promptTokens || ev.tokens || 0;
  return {
    round: ev.round || ev.id || "",
    ts: ev.ts || "",
    promptTokens: prompt,
    completionTokens: ev.completionTokens || 0,
    cachedTokens: ev.cachedTokens || 0,
    elapsedMs: ev.elapsedMs || 0,
    ttftMs: ev.ttftMs || 0,
    cacheReported: !!ev.cacheReported || (ev.cachedTokens || 0) > 0,
  };
}

/** callStats 最后一次调用，以及提示、补全、缓存 token 的合计。平均耗时和平均首 token 只平均大于 0 的调用，并用 round。没有这样的调用时平均为 0。 */
export function callStats(calls: ModelCall[]): {
  last: ModelCall | null;
  avgElapsed: number;
  avgTtft: number;
  totalPrompt: number;
  totalCompletion: number;
  totalCached: number;
} {
  const last = calls.length ? calls[calls.length - 1] : null;
  let elapsedN = 0;
  let elapsedSum = 0;
  let ttftN = 0;
  let ttftSum = 0;
  let totalPrompt = 0;
  let totalCompletion = 0;
  let totalCached = 0;
  for (const c of calls) {
    totalPrompt += c.promptTokens;
    totalCompletion += c.completionTokens;
    totalCached += c.cachedTokens;
    if (c.elapsedMs > 0) {
      elapsedN++;
      elapsedSum += c.elapsedMs;
    }
    if (c.ttftMs > 0) {
      ttftN++;
      ttftSum += c.ttftMs;
    }
  }
  return {
    last,
    avgElapsed: elapsedN ? Math.round(elapsedSum / elapsedN) : 0,
    avgTtft: ttftN ? Math.round(ttftSum / ttftN) : 0,
    totalPrompt,
    totalCompletion,
    totalCached,
  };
}

/** lastTurnMs 最后一条 user 与最后一条 turn_end 的时间差。没有事件、解析失败，或结束早于开始时为 0。 */
export function lastTurnMs(events: TraceEvent[] | undefined): number {
  if (!events?.length) return 0;
  let userTs = "";
  let endTs = "";
  for (const ev of events) {
    if (ev.type === "user") userTs = ev.ts;
    if (ev.type === "turn_end") endTs = ev.ts;
  }
  return deltaMs(userTs, endTs);
}

/** formatDurationMs 非正数为破折号。不足 1000 用毫秒。不足 60000 用秒，不足 10000 保留一位小数。否则为分钟加剩余秒。 */
export function formatDurationMs(ms: number): string {
  if (!ms || ms <= 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(ms < 10_000 ? 1 : 0)}s`;
  const m = Math.floor(ms / 60_000);
  const s = Math.round((ms % 60_000) / 1000);
  return `${m}m ${s}s`;
}

/** formatHitPct null 或 NaN 为破折号。整数不带小数，其余保留一位。 */
export function formatHitPct(n: number | null): string {
  if (n == null || Number.isNaN(n)) return "—";
  if (Number.isInteger(n)) return `${n}%`;
  return `${n.toFixed(1)}%`;
}

/** 两个时间戳的毫秒差。任一为空、解析失败，或结束早于开始时为 0。 */
function deltaMs(from: string, to: string): number {
  if (!from || !to) return 0;
  const a = Date.parse(from);
  const b = Date.parse(to);
  if (!Number.isFinite(a) || !Number.isFinite(b) || b < a) return 0;
  return b - a;
}
