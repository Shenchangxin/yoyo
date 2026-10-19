import type { Copy } from "./copy";

export function pointerLabel(copy: Copy, id: string): string {
  const k = id.toLowerCase();
  if (k === "active") return copy.rsi.active;
  if (k === "staging") return copy.rsi.staging;
  if (k === "canary") return copy.rsi.canary;
  if (k === "head") return copy.rsi.head;
  return id;
}

export function surfaceLabel(copy: Copy, surface: string): string {
  const k = surface.toLowerCase();
  if (k.includes("playbook")) return copy.rsi.surfacePlaybook;
  if (k.includes("instruction")) return copy.rsi.surfaceInstruction;
  if (k.includes("middleware")) return copy.rsi.surfaceMiddleware;
  if (k.includes("skill")) return copy.rsi.surfaceSkill;
  if (k.includes("loop")) return copy.rsi.surfaceLoop;
  if (k.includes("policy")) return copy.rsi.surfacePolicy;
  return surface;
}

export function checkHeadline(
  copy: Copy,
  metrics: { heldInPass: number; heldInTotal: number; heldOutPass: number; heldOutTotal: number; safetyFail: number } | null,
): string {
  if (!metrics) return copy.rsi.checkUnknown;
  if (metrics.safetyFail > 0) return copy.rsi.checkBlocked;
  const inOk = metrics.heldInTotal > 0 && metrics.heldInPass === metrics.heldInTotal;
  const outOk = metrics.heldOutTotal === 0 || metrics.heldOutPass === metrics.heldOutTotal;
  return inOk && outOk ? copy.rsi.checkReady : copy.rsi.checkNotReady;
}
