import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { classifyLive } from "./stream-cursor.ts";

/**
 * 拥有：实时游标对过期、连续、跨会话、跳号和乱序通知的约定。
 * 不拥有：回拉和事件应用的实现。
 */

describe("classifyLive", () => {
  /** 游标已经覆盖的通知丢掉。 */
  it("drops notices the cursor already covers", () => {
    assert.equal(classifyLive(10, 10, true, true), "stale");
    assert.equal(classifyLive(10, 4, false, false), "stale");
  });

  /** 下一号、属于本会话且事件内联时应用。 */
  it("applies the next own inline notice", () => {
    assert.equal(classifyLive(10, 11, true, true), "apply");
  });

  /** 别的会话的连续通知只推动游标。 */
  it("moves past another session's contiguous notice without applying it", () => {
    assert.equal(classifyLive(10, 11, false, true), "advance");
  });

  /** 跳号就回拉。跳号的通知即使来自别的会话，也不能证明中间的序号已经到了。 */
  it("pulls on a jump, even when the jumping notice is foreign", () => {
    // 序号 13 的外来通知证明不了本会话的 11 和 12。
    assert.equal(classifyLive(10, 13, false, true), "pull");
    assert.equal(classifyLive(10, 13, true, true), "pull");
  });

  /** 本会话的通知如果没带上事件正文，回拉补齐。 */
  it("pulls an own notice whose event body was not inlined", () => {
    assert.equal(classifyLive(10, 11, true, false), "pull");
  });

  /** 没有序号的通知按普通事件处理：本会话应用，别的会话只前进。 */
  it("treats a notice without a seq as a plain event", () => {
    assert.equal(classifyLive(10, 0, true, true), "apply");
    assert.equal(classifyLive(10, 0, false, true), "advance");
  });

  /** 会话交错时，连续性相对全局游标判断。 */
  it("interleaved sessions are contiguous against the global cursor", () => {
    let cursor = 10;
    const steps = [
      [11, true],
      [12, false],
      [13, true],
    ].map(([seq, own]) => {
      const step = classifyLive(cursor, seq as number, own as boolean, true);
      if (step === "apply" || step === "advance") cursor = seq as number;
      return step;
    });
    assert.deepEqual(steps, ["apply", "advance", "apply"]);
  });

  /** 乱序通知先回拉；回拉把游标推到头部之后，迟到的较小序号变为过期。 */
  it("an out-of-order notice pulls, then is stale once the pull covered it", () => {
    // 12 先于 11 到达渲染器。
    assert.equal(classifyLive(10, 12, true, true), "pull");
    // 回拉把游标推到头部 12，迟到的 11 变为过期。
    assert.equal(classifyLive(12, 11, true, true), "stale");
  });
});
