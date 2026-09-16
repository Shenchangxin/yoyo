import type { ContextUsage } from "./protocol";

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
  elided: number;
  layers: string[];
  note: string;
  slices: CtxSlice[];
};

export function contextBreakdown(ctx: ContextUsage | undefined, fallbackWindow = 0): CtxBreakdown {
  const prefix = Math.max(0, ctx?.prefixTokens || 0);
  const schema = Math.max(0, ctx?.schemaTokens || 0);
  const dynamic = Math.max(0, ctx?.dynamicTokens || 0);
  const used = Math.max(0, ctx?.tokens || 0);
  const window = ctx?.window && ctx.window > 0 ? ctx.window : 0;
  const budget = ctx?.budget && ctx.budget > 0 ? ctx.budget : 0;
  const capacity = window || budget || fallbackWindow || (used > 0 ? used : 0);
  const chat = Math.max(0, used - prefix - schema - dynamic);
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
