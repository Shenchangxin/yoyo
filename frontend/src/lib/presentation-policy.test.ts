import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  coerceCollapseTiming,
  coerceTranscriptView,
  presentationPolicyFor,
  processGrouped,
  turnWorkFolded,
} from "./presentation-policy.ts";

/**
 * 拥有：详略策略在折叠与分组上的约定。
 * 不拥有：策略的实现，以及转录行的真实排版。
 */

describe("presentationPolicyFor", () => {
  /** 四种档位映射到稳定的折叠、分组和过程细节开关。 */
  it("maps compact/standard/detailed/verbose to stable capabilities", () => {
    assert.equal(presentationPolicyFor("compact").foldCompletedTurns, true);
    assert.equal(presentationPolicyFor("compact").stepGrouping, "collapsed");
    assert.equal(presentationPolicyFor("compact").liveProcessDetail, false);
    assert.equal(presentationPolicyFor("standard").liveProcessDetail, true);
    assert.equal(presentationPolicyFor("detailed").stepGrouping, "history");
    assert.equal(presentationPolicyFor("verbose").foldCompletedTurns, false);
    assert.equal(presentationPolicyFor("verbose").stepGrouping, "none");
  });

  /** expanded 收成详细，无法识别的档位收成标准；空的折叠时机收成回合结束。 */
  it("coerces aliases", () => {
    assert.equal(coerceTranscriptView("expanded"), "detailed");
    assert.equal(coerceTranscriptView("nope"), "standard");
    assert.equal(coerceCollapseTiming("next-input"), "next-input");
    assert.equal(coerceCollapseTiming(""), "completion");
  });

  /** 标准档在进行中不折叠，回合结束后折叠。下一条输入时机要等下一条用户消息。啰嗦档不折叠。 */
  it("folds completed work on completion, not while live", () => {
    const standard = presentationPolicyFor("standard");
    assert.equal(turnWorkFolded(standard, true, false), false);
    assert.equal(turnWorkFolded(standard, false, false), true);
    assert.equal(turnWorkFolded(presentationPolicyFor("standard", "next-input"), false, false), false);
    assert.equal(turnWorkFolded(presentationPolicyFor("standard", "next-input"), false, true), true);
    assert.equal(turnWorkFolded(presentationPolicyFor("verbose"), false, true), false);
  });

  /** 紧凑档即使进行中也分组。详细档进行中不分组。啰嗦档不分组。 */
  it("groups process members except verbose and an open detailed turn", () => {
    assert.equal(processGrouped(presentationPolicyFor("compact"), true, false), true);
    assert.equal(processGrouped(presentationPolicyFor("detailed"), true, false), false);
    assert.equal(processGrouped(presentationPolicyFor("verbose"), false, true), false);
  });
});
