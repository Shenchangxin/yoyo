/**
 * 拥有：把 Harness 指针、表面和检查结论收成文案表里的句子。
 * 不拥有：文案表本身，以及评测数字怎么算出来。
 */

import type { Copy } from "./copy";

/** pointerLabel 把 active、staging、canary、head 收成对应文案，其余原样返回。比较时不区分大小写。copy 读取 rsi。id 转成小写，有的分支原样返回。 */
export function pointerLabel(copy: Copy, id: string): string {
  const k = id.toLowerCase();
  if (k === "active") return copy.rsi.active;
  if (k === "staging") return copy.rsi.staging;
  if (k === "canary") return copy.rsi.canary;
  if (k === "head") return copy.rsi.head;
  return id;
}

/** surfaceLabel 表面名里含 playbook、instruction、middleware、skill、loop、policy 时取对应文案，对不上则原样返回。copy 读取 rsi。surface 转成小写，有的分支原样返回。 */
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

/** checkHeadline 没有指标时未知。安全失败则阻断。留入集有题且全过，并且留出集为空或全过，才算就绪。copy 读取 rsi。metrics 读取 safetyFail、heldInTotal、heldInPass、heldOutTotal、heldOutPass，为假时走另一支。 */
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
