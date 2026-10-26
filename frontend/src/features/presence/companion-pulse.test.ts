/** 拥有陪伴脉搏解析和气泡文案的断言。不拥有窗口。不改服务端。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  bubbleLine,
  bubbleParts,
  itemEndsWork,
  itemStartsWork,
  mergePulse,
  pulseFrom,
  showLoading,
  type CompanionPulse,
} from "./companion-pulse.ts";

const now = 1_000_000;

/** 造一条加载脉搏，再盖上要测的字段。 */
function pulse(over: Partial<CompanionPulse> = {}): CompanionPulse {
  return { title: "web_search", body: "Working", session: "s1", until: now + 8_000, kind: "loading", ...over };
}

describe("companion pulse channel", () => {
  /** 能解析脉搏，加载停留得更久。 */
  it("parses yoyo:pulse and uses a longer ttl for loading", () => {
    const p = pulseFrom({ kind: "loading", title: "bash", body: "Working", session: "abc" }, now);
    assert.equal(p?.kind, "loading");
    assert.equal(p?.until, now + 16_000);
    assert.equal(pulseFrom({ title: "" }, now), null);
  });

  /** 用户和工具算开始干活，steer 不算。 */
  it("starts work on user/tool, not on steer", () => {
    assert.equal(itemStartsWork({ type: "user", source: "user" }), true);
    assert.equal(itemStartsWork({ type: "user", source: "steer" }), false);
    assert.equal(itemStartsWork({ type: "tool_call", source: "runtime" }), true);
    assert.equal(itemEndsWork({ type: "turn_end" }), true);
  });

  /** 完成会换掉加载，加载盖不住待批准。 */
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

  /** 等批准时不显示加载。 */
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

  /** 脉搏拆成眉题和正文。 */
  it("splits a pulse into a kicker and wrapping copy", () => {
    const parts = bubbleParts({
      approvals: 0,
      pulse: pulse({ title: "web_search", body: "Looking up the long source list now" }),
      approvalLine: "Needs your OK",
      emotionLine: "Thinking",
      now,
    });
    assert.equal(parts.kicker, "web_search");
    assert.equal(parts.copy, "Looking up the long source list now");
    assert.equal(
      bubbleLine({
        approvals: 0,
        pulse: pulse({ title: "web_search", body: "Looking up the long source list now" }),
        approvalLine: "Needs your OK",
        emotionLine: "Thinking",
        now,
      }),
      "web_search · Looking up the long source list now",
    );
  });
});
