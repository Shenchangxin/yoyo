import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { railBody } from "./rail-body.ts";

/**
 * 拥有：各大模块的侧栏不能回落到会话列表。
 * 不拥有：侧栏组件怎么画。
 */

describe("railBody", () => {
  /** 会话表面保留会话列表。编排页也留着这份列表。视频和设置沿用各自的导航。 */
  it("keeps the session list on the agent surface only", () => {
    assert.equal(railBody("agent"), "sessions");
    assert.equal(railBody("orchestrator"), "sessions");
    assert.equal(railBody("video"), "video");
    assert.equal(railBody("settings"), "settings");
  });

  /** 学习、技能、Harness 和偶像各用自己的侧栏。 */
  it("gives every other module its own sidebar", () => {
    assert.equal(railBody("pages"), "learn");
    assert.equal(railBody("skills"), "skills");
    assert.equal(railBody("harness"), "harness");
    assert.equal(railBody("idol"), "idol");
  });
});
