import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  coerceCollapseTiming,
  coerceTranscriptView,
  presentationPolicyFor,
  processGrouped,
  turnWorkFolded,
} from "./presentation-policy.ts";

describe("presentationPolicyFor", () => {
  it("maps compact/standard/detailed/verbose to stable capabilities", () => {
    assert.equal(presentationPolicyFor("compact").foldCompletedTurns, true);
    assert.equal(presentationPolicyFor("compact").stepGrouping, "collapsed");
    assert.equal(presentationPolicyFor("compact").liveProcessDetail, false);
    assert.equal(presentationPolicyFor("standard").liveProcessDetail, true);
    assert.equal(presentationPolicyFor("detailed").stepGrouping, "history");
    assert.equal(presentationPolicyFor("verbose").foldCompletedTurns, false);
    assert.equal(presentationPolicyFor("verbose").stepGrouping, "none");
  });

  it("coerces aliases", () => {
    assert.equal(coerceTranscriptView("expanded"), "detailed");
    assert.equal(coerceTranscriptView("nope"), "standard");
    assert.equal(coerceCollapseTiming("next-input"), "next-input");
    assert.equal(coerceCollapseTiming(""), "completion");
  });

  it("folds completed work on completion, not while live", () => {
    const standard = presentationPolicyFor("standard");
    assert.equal(turnWorkFolded(standard, true, false), false);
    assert.equal(turnWorkFolded(standard, false, false), true);
    assert.equal(turnWorkFolded(presentationPolicyFor("standard", "next-input"), false, false), false);
    assert.equal(turnWorkFolded(presentationPolicyFor("standard", "next-input"), false, true), true);
    assert.equal(turnWorkFolded(presentationPolicyFor("verbose"), false, true), false);
  });

  it("groups process members except verbose and an open detailed turn", () => {
    assert.equal(processGrouped(presentationPolicyFor("compact"), true, false), true);
    assert.equal(processGrouped(presentationPolicyFor("detailed"), true, false), false);
    assert.equal(processGrouped(presentationPolicyFor("verbose"), false, true), false);
  });
});
