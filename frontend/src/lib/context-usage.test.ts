/**
 * 拥有：缓存命中比例、时长格式、调用平均，以及分片已知时对话可以为 0。
 * 不拥有：用量的采集。
 */

import assert from "node:assert/strict";
import { test } from "node:test";
import {
  cacheHitPct,
  callStats,
  contextBreakdown,
  formatDurationMs,
  formatHitPct,
  lastTurnMs,
  modelCallsOf,
} from "./context-usage.ts";
import type { SessionTrace, TraceEvent } from "./protocol.ts";

/** 已报告时为缓存除以提示词，一位小数。未报告时为空。提示词为 0 且已报告则为 0。 */
test("cacheHitPct is cached / prompt, null when unknown", () => {
  assert.equal(cacheHitPct(80, 100, true), 80);
  assert.equal(cacheHitPct(1, 3, true), 33.3);
  assert.equal(cacheHitPct(0, 100, true), 0);
  assert.equal(cacheHitPct(12, 0, true), 0);
  assert.equal(cacheHitPct(12, 0, false), null);
  assert.equal(cacheHitPct(12, 100, false), null);
  assert.equal(cacheHitPct(12, 100), 12);
});

/** 空命中为破折号，整数不带小数。时长按毫秒、一位小数秒、整数秒和分秒。 */
test("formatHitPct and formatDurationMs", () => {
  assert.equal(formatHitPct(null), "—");
  assert.equal(formatHitPct(80), "80%");
  assert.equal(formatHitPct(33.3), "33.3%");
  assert.equal(formatDurationMs(0), "—");
  assert.equal(formatDurationMs(240), "240ms");
  assert.equal(formatDurationMs(1240), "1.2s");
  assert.equal(formatDurationMs(12_000), "12s");
  assert.equal(formatDurationMs(65_000), "1m 5s");
});

/** 只取出 model_usage。平均耗时和平均首 token 为两次的整数平均。最后一次的命中按缓存除以提示词。 */
test("modelCallsOf reads usage rows and callStats averages timed calls", () => {
  const trace: SessionTrace = {
    sessionId: "s1",
    title: "",
    workspace: "",
    harness: "",
    model: "",
    createdAt: "",
    startedAt: "",
    endedAt: "",
    stats: {
      events: 2, users: 0, assistants: 0, toolCalls: 0, toolResults: 0,
      errors: 0, compactions: 0, deltas: 0, tokens: 0, durationMs: 4000, spillBytes: 0,
    },
    events: [
      ev({ type: "model_usage", round: "s1:r1", promptTokens: 100, completionTokens: 20, cachedTokens: 40, elapsedMs: 1000, ttftMs: 200, cacheReported: true }),
      ev({ type: "assistant", round: "s1:r1" }),
      ev({ type: "model_usage", round: "s1:r2", promptTokens: 200, completionTokens: 10, cachedTokens: 160, elapsedMs: 500, ttftMs: 100, cacheReported: true }),
    ],
    artifacts: [],
  };
  const calls = modelCallsOf(trace);
  assert.equal(calls.length, 2);
  assert.equal(calls[1].round, "s1:r2");
  const stats = callStats(calls);
  assert.equal(stats.last?.round, "s1:r2");
  assert.equal(stats.avgElapsed, 750);
  assert.equal(stats.avgTtft, 150);
  assert.equal(stats.totalPrompt, 300);
  assert.equal(stats.totalCached, 200);
  assert.equal(cacheHitPct(stats.last!.cachedTokens, stats.last!.promptTokens, true), 80);
});

/** 最后一条 user 到最后一条 turn_end 为 3500 毫秒。空列表为 0。 */
test("lastTurnMs is the latest user to turn_end wall", () => {
  const events: TraceEvent[] = [
    ev({ type: "user", ts: "2026-01-01T00:00:00.000Z" }),
    ev({ type: "turn_end", ts: "2026-01-01T00:00:02.000Z" }),
    ev({ type: "user", ts: "2026-01-01T00:00:10.000Z" }),
    ev({ type: "turn_end", ts: "2026-01-01T00:00:13.500Z" }),
  ];
  assert.equal(lastTurnMs(events), 3500);
  assert.equal(lastTurnMs([]), 0);
});

/** slicesKnown 且 chatTokens 为 0 时，对话片为 0，笔记片为 notesTokens。 */
test("a new session reports chat as zero when the ledger says so", () => {
  const br = contextBreakdown({
    tokens: 6000, budget: 100000, window: 128000, note: "", layers: [], elided: 0,
    prefixTokens: 4000, schemaTokens: 1800, dynamicTokens: 200,
    identityTokens: 700, rulesTokens: 900, catalogTokens: 400, skillBodyTokens: 0,
    memoryTokens: 80, todayTokens: 0, notesTokens: 200, chatTokens: 0, slicesKnown: true,
  }, 0, 0);
  assert.equal(br.chat, 0);
  assert.equal(br.slices.find((s) => s.id === "chat")?.tokens, 0);
  assert.equal(br.slices.find((s) => s.id === "notes")?.tokens, 200);
  assert.equal(br.slices.find((s) => s.id === "skills")?.tokens, 0);
});

/** 展开结果仍带上 cachedTokens 和 providerPrompt。50 除以 90 再按公式取一位是 55.6。 */
test("contextBreakdown still reports cache fields", () => {
  const br = contextBreakdown({
    tokens: 100, budget: 200, window: 400, note: "", layers: [], elided: 0,
    cachedTokens: 50, cacheReported: true, providerPrompt: 90,
  });
  assert.equal(br.cachedTokens, 50);
  assert.equal(br.providerPrompt, 90);
  assert.equal(cacheHitPct(br.cachedTokens, br.providerPrompt, br.cacheReported), 55.6);
});

/** 补齐追踪事件缺省字段，便于只写与本例有关的项。 */
function ev(partial: Partial<TraceEvent>): TraceEvent {
  return {
    index: 0,
    ts: "",
    type: "",
    source: "",
    lane: "dialog",
    round: "",
    name: "",
    id: "",
    summary: "",
    detail: "",
    bytes: 0,
    elapsedMs: 0,
    spillId: "",
    tokens: 0,
    promptTokens: 0,
    completionTokens: 0,
    cachedTokens: 0,
    ttftMs: 0,
    cacheReported: false,
    error: false,
    ...partial,
  };
}
