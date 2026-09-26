import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  bubbleLine,
  itemEndsWork,
  itemStartsWork,
  mergePulse,
  pulseFrom,
  showLoading,
  type CompanionPulse,
} from "./companion-pulse.ts";

const now = 1_000_000;

function pulse(over: Partial<CompanionPulse> = {}): CompanionPulse {
  return { title: "web_search", body: "Working", session: "s1", until: now + 8_000, kind: "loading", ...over };
}

describe("companion pulse channel", () => {
  it("parses yoyo:pulse and uses a longer ttl for loading", () => {
    const p = pulseFrom({ kind: "loading", title: "bash", body: "Working", session: "abc" }, now);
    assert.equal(p?.kind, "loading");
    assert.equal(p?.until, now + 16_000);
    assert.equal(pulseFrom({ title: "" }, now), null);
  });

  it("starts work on user/tool, not on steer", () => {
    assert.equal(itemStartsWork({ type: "user", source: "user" }), true);
    assert.equal(itemStartsWork({ type: "user", source: "steer" }), false);
    assert.equal(itemStartsWork({ type: "tool_call", source: "runtime" }), true);
    assert.equal(itemEndsWork({ type: "turn_end" }), true);
  });

  it("clears loading on done and never lets loading cover an approval", () => {
    const done = mergePulse(pulse(), pulse({ kind: "done", title: "Yoyo", body: "Turn finished" }), now);
    assert.equal(done.kind, "done");
    const kept = mergePulse(
      pulse({ kind: "approval", title: "Needs OK", until: now + 5_000 }),
      pulse({ kind: "loading", title: "bash" }),
      now,
    );
    assert.equal(kept.kind, "approval");
  });

  it("hides loading dots while an approval is waiting", () => {
    assert.equal(showLoading(true, pulse(), 1, now), false);
    assert.equal(showLoading(true, pulse(), 0, now), true);
    assert.equal(showLoading(false, pulse({ kind: "done" }), 0, now), false);
    assert.equal(
      bubbleLine({
        approvals: 1,
        pulse: pulse(),
        approvalLine: "Needs your OK",
        emotionLine: "Thinking",
        now,
      }),
      "Needs your OK",
    );
  });
});
