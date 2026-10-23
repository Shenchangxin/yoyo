import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { cycleScore, setScore, topicCanFilm, topicIsScheduled, topicMatchesQuery, topicPrimaryAction, topicScoreAvg, trendMatchesQuery } from "./studio-topic.ts";

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

  it("lets a failed film be made again", () => {
    assert.equal(topicCanFilm({}), true);
    assert.equal(topicCanFilm({ studio_project_id: "abc" }), false);
    assert.equal(topicCanFilm({ studio_project_id: "abc" }, { auto_phase: "script" }), false);
    assert.equal(topicCanFilm({ studio_project_id: "abc" }, { auto_phase: "done", result_hash: "v" }), false);
    assert.equal(topicCanFilm({ studio_project_id: "abc" }, { auto_phase: "failed" }), true);
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

  it("sets a score and clears the same tick", () => {
    assert.deepEqual(setScore({ heat: 5 }, "heat", 3), { heat: 3 });
    assert.deepEqual(setScore({ heat: 3 }, "heat", 3), {});
    assert.deepEqual(setScore({ heat: 5, feasibility: 4 }, "timing", 2), { heat: 5, feasibility: 4, timing: 2 });
  });

  it("filters topics and hot rows by query", () => {
    assert.equal(topicMatchesQuery({ title: "春日穿搭", angle: "通勤" }, "穿搭"), true);
    assert.equal(topicMatchesQuery({ title: "春日穿搭" }, "夜路"), false);
    assert.equal(trendMatchesQuery({ title: "热搜A", hot: "120w" }, "热搜"), true);
    assert.equal(trendMatchesQuery({ title: "热搜A" }, ""), true);
  });
});
