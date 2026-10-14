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

test("cacheHitPct is cached / prompt, null when unknown", () => {
  assert.equal(cacheHitPct(80, 100, true), 80);
  assert.equal(cacheHitPct(1, 3, true), 33.3);
  assert.equal(cacheHitPct(0, 100, true), 0);
  assert.equal(cacheHitPct(12, 0, true), 0);
  assert.equal(cacheHitPct(12, 0, false), null);
  assert.equal(cacheHitPct(12, 100, false), null);
  assert.equal(cacheHitPct(12, 100), 12);
});

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

test("contextBreakdown still reports cache fields", () => {
  const br = contextBreakdown({
    tokens: 100, budget: 200, window: 400, note: "", layers: [], elided: 0,
    cachedTokens: 50, cacheReported: true, providerPrompt: 90,
  });
  assert.equal(br.cachedTokens, 50);
  assert.equal(br.providerPrompt, 90);
  assert.equal(cacheHitPct(br.cachedTokens, br.providerPrompt, br.cacheReported), 55.6);
});

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
