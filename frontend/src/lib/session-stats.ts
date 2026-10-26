/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 */

/**
 * 拥有：把会话追踪、大纲和转录折成统计，以及 token、时长和时钟的紧凑格式。
 * 不拥有：Cordis 投影，以及统计药丸的渲染。格式来自 deepseek-harness 的 StatsPills 与 token-format，再折进 Yoyo 的 SessionTrace、大纲和条目。
 */
import { modelCallsOf } from "./context-usage";
import type { Item, SessionTrace } from "./protocol";
import type { OutlineTurn } from "./turn-outline";

/** SessionStats 一段会话的回合数、步数、模型与工具耗时，以及输入输出 token。ttft 是首 token 延迟，decode 是其后的生成阶段。 */
export type SessionStats = {
  turns: number;
  steps: number;
  llmMs: number;
  toolMs: number;
  ttftMs: number;
  ttftSteps: number;
  decodeMs: number;
  decodeTokens: number;
  uncachedInputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  outputTokens: number;
};

/** 全零的统计，供累加前使用。 */
const emptyStats = (): SessionStats => ({
  turns: 0,
  steps: 0,
  llmMs: 0,
  toolMs: 0,
  ttftMs: 0,
  ttftSteps: 0,
  decodeMs: 0,
  decodeTokens: 0,
  uncachedInputTokens: 0,
  cacheReadTokens: 0,
  cacheWriteTokens: 0,
  outputTokens: 0,
});

/** billedInputTokens 未缓存输入、缓存读和缓存写之和。 */
export function billedInputTokens(stats: Pick<SessionStats, "uncachedInputTokens" | "cacheReadTokens" | "cacheWriteTokens">): number {
  return stats.uncachedInputTokens + stats.cacheReadTokens + stats.cacheWriteTokens;
}

/** formatCompactTokens 紧凑 token。不足一千取整，不足一百万用 K，再大用 M。一百以上的缩写取整，否则保留一位小数。 */
export function formatCompactTokens(value: number): string {
  const n = Math.max(0, value);
  const scaled = (candidate: number): string =>
    candidate >= 100 ? String(Math.round(candidate)) : String(Math.round(candidate * 10) / 10);
  if (n < 1_000) return String(Math.round(n));
  if (n < 1_000_000) return `${scaled(n / 1_000)}K`;
  return `${scaled(n / 1_000_000)}M`;
}

/** formatExactTokens 带千分位的精确 token。 */
export function formatExactTokens(value: number): string {
  const digits = String(Math.max(0, Math.round(value)));
  const groups: string[] = [];
  for (let end = digits.length; end > 0; end -= 3) {
    groups.unshift(digits.slice(Math.max(0, end - 3), end));
  }
  return groups.join(",");
}

/** formatCompactDuration 不足一分钟用秒并保留一位小数，否则用分和两位秒。 */
export function formatCompactDuration(ms: number): string {
  const s = ms / 1_000;
  if (s < 60) return `${Math.round(s * 10) / 10}s`;
  const whole = Math.round(s);
  return `${Math.floor(whole / 60)}m${String(whole % 60).padStart(2, "0")}s`;
}

/** formatTokensPerSecond 十以上取整，十以下保留一位小数。 */
export function formatTokensPerSecond(tps: number): string {
  const clamped = Math.max(0, tps);
  return clamped >= 10 ? String(Math.round(clamped)) : String(Math.round(clamped * 10) / 10);
}

/** 在给定小数位下，把命中数折成百分比的整数单位。 */
function roundedPercentUnits(cacheReadTokens: number, denominator: number, decimalPlaces: 0 | 1): number {
  const unitsPerPercent = decimalPlaces === 0 ? 1 : 10;
  const scale = unitsPerPercent * 100;
  const doubledScale = scale * 2;
  const denominatorQuotient = Math.floor(denominator / doubledScale);
  const denominatorRemainder = denominator % doubledScale;
  let lower = 0;
  let upper = scale;
  while (lower < upper) {
    const candidate = Math.floor((lower + upper + 1) / 2);
    const factor = candidate * 2 - 1;
    const threshold = factor * denominatorQuotient
      + Math.ceil(factor * denominatorRemainder / doubledScale);
    if (cacheReadTokens >= threshold) lower = candidate;
    else upper = candidate - 1;
  }
  return lower;
}

/** 把百分比单位格式化成整数，或去掉末尾零的一位小数。 */
function displayPercentUnits(units: number, decimalPlaces: 0 | 1): string {
  if (decimalPlaces === 0) return String(units);
  const whole = Math.floor(units / 10);
  const tenths = units % 10;
  return tenths === 0 ? String(whole) : `${whole}.${tenths}`;
}

/**
 * formatCacheHitPercent 给出可展示的缓存命中占比。
 * 提示词为零时返回空。全部命中为 100。差一点满命中时保留 99 开头，避免把部分命中四舍五入成 100。
 */
export function formatCacheHitPercent(
  cacheReadTokens: number,
  promptTokens: number,
  decimalPlaces: 0 | 1 = 0,
): string | null {
  if (promptTokens === 0) return null;
  const missedInputTokens = promptTokens - cacheReadTokens;
  if (missedInputTokens === 0) return "100";

  const roundedUnits = roundedPercentUnits(cacheReadTokens, promptTokens, decimalPlaces);
  const fullHitUnits = decimalPlaces === 0 ? 100 : 1_000;
  if (roundedUnits < fullHitUnits) return displayPercentUnits(roundedUnits, decimalPlaces);

  let distinguishingPlaces = 1;
  let scaledDoubleGap = missedInputTokens * 200;
  const denominatorTens = Math.floor(promptTokens / 10);
  while (scaledDoubleGap <= denominatorTens) {
    scaledDoubleGap *= 10;
    distinguishingPlaces += 1;
  }
  const denominatorOnes = promptTokens % 10;
  let roundedLoss = 5;
  for (let loss = 1; loss < 5; loss += 1) {
    const factor = loss * 2 + 1;
    const threshold = factor * denominatorTens + Math.floor(factor * denominatorOnes / 10);
    if (scaledDoubleGap <= threshold) {
      roundedLoss = loss;
      break;
    }
  }
  return `99.${"9".repeat(distinguishingPlaces - 1)}${10 - roundedLoss}`;
}

/** 两个时间戳的毫秒差。缺一、无法解析或结束早于开始时为零。 */
function deltaMs(from: string, to: string): number {
  if (!from || !to) return 0;
  const a = Date.parse(from);
  const b = Date.parse(to);
  if (!Number.isFinite(a) || !Number.isFinite(b) || b < a) return 0;
  return b - a;
}

/** 按追踪里成对的工具调用和结果累加工具耗时。 */
function toolMsOfTrace(trace: SessionTrace | null | undefined): number {
  const open = new Map<string, string>();
  let ms = 0;
  for (const ev of trace?.events || []) {
    const key = ev.id || ev.round || ev.name;
    if (ev.type === "tool_call") {
      if (key) open.set(key, ev.ts);
      continue;
    }
    if (ev.type !== "tool_result") continue;
    const started = (key && open.get(key)) || "";
    if (key) open.delete(key);
    ms += deltaMs(started, ev.ts);
  }
  return ms;
}

/** 按转录条目里成对的工具调用和结果累加工具耗时。 */
function toolMsOfItems(items: Item[] | undefined): number {
  const open = new Map<string, string>();
  let ms = 0;
  for (const it of items || []) {
    const key = String(it.payload?.id || it.name || it.key || "");
    if (it.type === "tool_call") {
      if (key) open.set(key, it.ts);
      continue;
    }
    if (it.type !== "tool_result") continue;
    const started = (key && open.get(key)) || "";
    if (key) open.delete(key);
    ms += deltaMs(started, it.ts);
  }
  return ms;
}

/** deriveSessionStats 从大纲、条目和追踪折出统计。引导备注不算回合。模型调用优先，没有则用助手条目当步数。工具耗时优先用追踪。 */
export function deriveSessionStats(input: {
  outlineTurns?: OutlineTurn[];
  items?: Item[];
  trace?: SessionTrace | null;
}): SessionStats {
  const stats = emptyStats();
  const calls = modelCallsOf(input.trace);
  const outlineTurns = (input.outlineTurns || []).filter((t) => t.kind !== "steer");
  const userTurns = (input.items || []).filter((it) => it.type === "user" && it.source !== "steer");
  stats.turns = outlineTurns.length || userTurns.length;
  stats.steps = calls.length || (input.items || []).filter((it) => it.type === "assistant").length;
  stats.toolMs = toolMsOfTrace(input.trace) || toolMsOfItems(input.items);

  for (const call of calls) {
    stats.llmMs += Math.max(0, call.elapsedMs);
    stats.outputTokens += Math.max(0, call.completionTokens);
    stats.cacheReadTokens += Math.max(0, call.cachedTokens);
    const prompt = Math.max(0, call.promptTokens);
    stats.uncachedInputTokens += Math.max(0, prompt - Math.max(0, call.cachedTokens));
    if (call.ttftMs > 0) {
      stats.ttftMs += call.ttftMs;
      stats.ttftSteps += 1;
    }
    const decode = call.elapsedMs > 0
      ? Math.max(0, call.elapsedMs - Math.max(0, call.ttftMs))
      : 0;
    if (decode > 0 && call.completionTokens > 0) {
      stats.decodeMs += decode;
      stats.decodeTokens += call.completionTokens;
    }
  }
  return stats;
}

/** formatMessageClock 同一天只显示时分，同一年显示月日和时分，更早则带上年份。月日模板由调用方提供。无效时间返回空串。 */
export function formatMessageClock(time: number, templates: { md: string; ymd: string }, now = Date.now()): string {
  if (!Number.isFinite(time) || time <= 0) return "";
  const d = new Date(time);
  const n = new Date(now);
  const pad = (v: number) => String(v).padStart(2, "0");
  const clock = `${pad(d.getHours())}:${pad(d.getMinutes())}`;
  if (d.getFullYear() === n.getFullYear() && d.getMonth() === n.getMonth() && d.getDate() === n.getDate()) {
    return clock;
  }
  const month = String(d.getMonth() + 1);
  const day = String(d.getDate());
  if (d.getFullYear() === n.getFullYear()) {
    return `${templates.md.replace("{m}", month).replace("{d}", day)} ${clock}`;
  }
  return `${templates.ymd.replace("{y}", String(d.getFullYear())).replace("{m}", month).replace("{d}", day)} ${clock}`;
}
