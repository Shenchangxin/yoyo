import type { ContextUsage, SessionTrace, TraceEvent } from "./protocol";

export const CTX_SLICE_IDS = ["system", "tools", "dynamic", "chat", "free"] as const;
export type CtxSliceId = (typeof CTX_SLICE_IDS)[number];

export type CtxSlice = {
  id: CtxSliceId;
  tokens: number;
};

export type CtxBreakdown = {
  used: number;
  capacity: number;
  pct: number;
  prefix: number;
  schema: number;
  dynamic: number;
  chat: number;
  free: number;
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

/** Same heuristic as runtime.estimateTokens: ~4 bytes/token. */
export function estimateTokens(s: string): number {
  if (!s) return 0;
  return Math.floor((new TextEncoder().encode(s).length + 3) / 4);
}

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
  const chat = Math.max(0, baseUsed - prefix - schema - dynamic) + draft;
  const free = Math.max(0, capacity - used);
  const pct = capacity > 0 ? Math.min(100, Math.round((used / capacity) * 100)) : 0;
  return {
    used,
    capacity,
    pct,
    prefix,
    schema,
    dynamic,
    chat,
    free,
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
      { id: "system", tokens: prefix },
      { id: "tools", tokens: schema },
      { id: "dynamic", tokens: dynamic },
      { id: "chat", tokens: chat },
      { id: "free", tokens: free },
    ],
  };
}

/** Cached share of the provider prompt. Null when the provider did not report. */
export function cacheHitPct(cached: number, prompt: number, reported?: boolean): number | null {
  if (reported === false) return null;
  if (prompt <= 0) return reported ? 0 : null;
  return Math.min(100, Math.round((Math.max(0, cached) / prompt) * 1000) / 10);
}

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

export function modelCallsOf(trace: SessionTrace | null | undefined): ModelCall[] {
  const out: ModelCall[] = [];
  for (const ev of trace?.events || []) {
    if (ev.type !== "model_usage") continue;
    out.push(modelCallOf(ev));
  }
  return out;
}

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

/** Wall time of the latest user → turn_end pair. */
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

export function formatDurationMs(ms: number): string {
  if (!ms || ms <= 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(ms < 10_000 ? 1 : 0)}s`;
  const m = Math.floor(ms / 60_000);
  const s = Math.round((ms % 60_000) / 1000);
  return `${m}m ${s}s`;
}

export function formatHitPct(n: number | null): string {
  if (n == null || Number.isNaN(n)) return "—";
  if (Number.isInteger(n)) return `${n}%`;
  return `${n.toFixed(1)}%`;
}

function deltaMs(from: string, to: string): number {
  if (!from || !to) return 0;
  const a = Date.parse(from);
  const b = Date.parse(to);
  if (!Number.isFinite(a) || !Number.isFinite(b) || b < a) return 0;
  return b - a;
}
