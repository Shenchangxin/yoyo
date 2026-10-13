import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { cycleScore, topicIsScheduled, topicPrimaryAction, topicScoreAvg } from "./studio-topic.ts";

describe("topic card helpers", () => {
  it("averages filled scores only", () => {
    assert.equal(topicScoreAvg(undefined), 0);
    assert.equal(topicScoreAvg({ heat: 5, feasibility: 4 }), 4.5);
    assert.equal(topicScoreAvg({ heat: 0, risk: 3 }), 3);
  });

  it("opens an existing short instead of producing a clone", () => {
    assert.equal(topicPrimaryAction({}), "produce");
    assert.equal(topicPrimaryAction({ studio_project_id: "abc" }), "open");
  });

  it("treats scheduled as pending only before a short exists", () => {
    assert.equal(topicIsScheduled({ status: "scheduled" }), true);
    assert.equal(topicIsScheduled({ status: "scheduled", studio_project_id: "abc" }), false);
    assert.equal(topicIsScheduled({ status: "ready" }), false);
  });

  it("cycles 1–5 then clears", () => {
    assert.equal(cycleScore(0), 1);
    assert.equal(cycleScore(5), 0);
    assert.equal(cycleScore(3), 4);
  });
});
