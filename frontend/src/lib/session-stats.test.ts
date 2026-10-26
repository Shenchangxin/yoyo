/**
 * 拥有：缓存命中展示、紧凑格式，以及统计折叠对用量、回合和工具耗时的约定。
 * 不拥有：追踪的采集和药丸的渲染。
 */

import assert from "node:assert/strict";
import { test } from "node:test";
import {
  billedInputTokens,
  deriveSessionStats,
  formatCacheHitPercent,
  formatCompactDuration,
  formatCompactTokens,
  formatMessageClock,
  formatTokensPerSecond,
} from "./session-stats.ts";
import type { SessionTrace, TraceEvent } from "./protocol.ts";

/** 补齐追踪事件缺省字段，便于只写与本例有关的项。 */
function ev(partial: Partial<TraceEvent> & { type: string }): TraceEvent {
  return {
    index: 0,
    ts: "",
    source: "",
    lane: "",
    round: "",
    name: "",
    id: "",
    summary: "",
    detail: "",
    bytes: 0,
    elapsedMs: 0,
    spillId: "",
    tokens: 0,
    error: false,
    ...partial,
  };
}

/** 提示词为零返回空，满命中为 100，差一点满命中时停在 99 开头。 */
test("formatCacheHitPercent stays honest on a near miss", () => {
  assert.equal(formatCacheHitPercent(0, 0), null);
  assert.equal(formatCacheHitPercent(80, 100), "80");
  assert.equal(formatCacheHitPercent(100, 100), "100");
  const near = formatCacheHitPercent(999, 1000);
  assert.ok(near && near.startsWith("99"));
  assert.notEqual(near, "100");
});

/** 紧凑 token、时长和每秒 token 按档位取整。 */
test("formatCompactTokens and duration and tps", () => {
  assert.equal(formatCompactTokens(517), "517");
  assert.equal(formatCompactTokens(12_200), "12.2K");
  assert.equal(formatCompactTokens(517_000), "517K");
  assert.equal(formatCompactDuration(45_200), "45.2s");
  assert.equal(formatCompactDuration(162_000), "2m42s");
  assert.equal(formatTokensPerSecond(4.21), "4.2");
  assert.equal(formatTokensPerSecond(40.6), "41");
});

/** 大纲回合、模型用量和成对工具时间折进同一份统计。 */
test("deriveSessionStats folds usage, turns, and tool walls", () => {
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
      events: 0, users: 0, assistants: 0, toolCalls: 0, toolResults: 0,
      errors: 0, compactions: 0, deltas: 0, tokens: 0, durationMs: 0, spillBytes: 0,
    },
    events: [
      ev({ type: "user", ts: "2026-01-01T00:00:00.000Z" }),
      ev({ type: "model_usage", round: "r1", promptTokens: 100, completionTokens: 20, cachedTokens: 40, elapsedMs: 1000, ttftMs: 200 }),
      ev({ type: "tool_call", id: "t1", ts: "2026-01-01T00:00:01.000Z" }),
      ev({ type: "tool_result", id: "t1", ts: "2026-01-01T00:00:03.000Z" }),
      ev({ type: "assistant" }),
    ],
    artifacts: [],
  };
  const stats = deriveSessionStats({
    outlineTurns: [{ seq: 1, title: "hi", id: "u1", key: "u1", kind: "user" }],
    trace,
  });
  assert.equal(stats.turns, 1);
  assert.equal(stats.steps, 1);
  assert.equal(stats.llmMs, 1000);
  assert.equal(stats.toolMs, 2000);
  assert.equal(stats.ttftSteps, 1);
  assert.equal(stats.ttftMs, 200);
  assert.equal(stats.decodeMs, 800);
  assert.equal(stats.decodeTokens, 20);
  assert.equal(stats.cacheReadTokens, 40);
  assert.equal(stats.uncachedInputTokens, 60);
  assert.equal(stats.outputTokens, 20);
  assert.equal(billedInputTokens(stats), 100);
});

/** 同一天只剩时分。 */
test("formatMessageClock same day is HH:mm", () => {
  const now = Date.now();
  const clock = formatMessageClock(now - 60_000, { md: "{m}/{d}", ymd: "{y}/{m}/{d}" }, now);
  assert.match(clock, /^\d{2}:\d{2}$/);
});
