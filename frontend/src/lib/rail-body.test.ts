import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { railBody } from "./rail-body.ts";

/**
 * 拥有：各大模块的侧栏不能回落到会话列表。
 * 不拥有：侧栏组件怎么画。
 */

describe("railBody", () => {
  /** 会话表面把列表收进历史。视频和设置沿用各自的导航。 */
  it("keeps history collapsed on the agent surface", () => {
    assert.equal(railBody("agent"), "home");
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

  /**
   * Buddy 不再是一个表面，也没有自己的侧栏：它在会话的今天栏里，和今天、任务、记忆并排。
   * 所以七个表面各有各的侧栏，里面没有 buddy 这一种。
   */
  it("has no sidebar of its own for buddy", () => {
    const bodies = (["agent", "video", "settings", "pages", "skills", "harness", "idol"] as const).map((surface) => railBody(surface));
    assert.deepEqual(bodies, ["home", "video", "settings", "learn", "skills", "harness", "idol"]);
    assert.equal(bodies.includes("buddy" as never), false);
  });
});
