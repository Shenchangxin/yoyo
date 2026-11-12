import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Item, ItemType } from "./protocol.ts";
import { learnChangeOf } from "./learn-events.ts";
import { dayDistance, fill, percent, whenLabel } from "./learn-format.ts";
import { isLearnQuizItem, parseQuizPosted, quizPostedOf } from "./learn-quiz.ts";
import { classifyToolPairs, pairTools } from "./transcript-layout.ts";

/**
 * 拥有：出题工具的结果能被读成题卡、在对话里算产出，学习变化事件的两种信封都认得，以及到期日的说法。
 * 不拥有：题卡的渲染和判分。
 */

/** 补齐转录条目的缺省字段。 */
function item(p: Partial<Item> & { key: string; type: ItemType }): Item {
  return { sessionId: "s", source: "", ts: "2026-11-12T00:00:00.000Z", text: "", name: "", delta: false, payload: {}, ...p } as Item;
}

const posted = JSON.stringify({
  status: "posed",
  path_id: "path-1",
  question: { id: "q-1", point_id: "kp-1", question_type: "choice", prompt: "天空的颜色", options: [{ label: "A", body: "红" }, { label: "B", body: "蓝" }] },
  notice: "The question is on the learner's answer card and this turn ends here.",
});

describe("learn quiz card source", () => {
  /** 结果里有路径和题目 id 才读得出题卡；题面里没有标准答案。 */
  it("reads the path and the public question from a learn_quiz result", () => {
    const got = parseQuizPosted(posted);
    assert.ok(got);
    assert.equal(got.pathId, "path-1");
    assert.equal(got.question.id, "q-1");
    assert.deepEqual(got.question.options?.map((o) => o.label), ["A", "B"]);
    assert.equal(JSON.stringify(got).includes("expected"), false);
  });

  /** 不是 JSON、缺路径或缺题目 id 的正文不画卡片。简答题没有选项时补成空数组。 */
  it("returns null for bodies it cannot use and fills missing options", () => {
    assert.equal(parseQuizPosted("ERROR: path busy"), null);
    assert.equal(parseQuizPosted(JSON.stringify({ question: { id: "q" } })), null);
    assert.equal(parseQuizPosted(JSON.stringify({ path_id: "p", question: { prompt: "x" } })), null);
    const short = parseQuizPosted(JSON.stringify({ path_id: "p", question: { id: "q", question_type: "short", prompt: "x" } }));
    assert.deepEqual(short?.question.options, []);
    assert.equal(quizPostedOf(null), null);
  });

  /** 出题的结果到了之后这一对工具算产出，题卡画在对话里；结果没到或调用失败时仍在过程轨。 */
  it("classifies a finished learn_quiz pair as an artifact", () => {
    const call = item({ key: "c1:call", type: "tool_call", name: "learn_quiz", payload: { id: "c1", name: "learn_quiz", arguments: "{}" } });
    const result = item({ key: "c1:result", type: "tool_result", name: "learn_quiz", payload: { id: "c1", name: "learn_quiz", content: posted } });
    assert.equal(isLearnQuizItem(call), true);
    assert.equal(isLearnQuizItem(item({ key: "x", type: "tool_call", name: "read_file", payload: { name: "read_file" } })), false);
    const done = pairTools([call, result]);
    assert.equal(classifyToolPairs(done).get(done[0].key), "artifact");
    assert.equal(quizPostedOf(done[0].result)?.question.id, "q-1");
    const open = pairTools([call]);
    assert.equal(classifyToolPairs(open).get(open[0].key), "process");
    const failed = pairTools([call, item({ key: "c1:result", type: "tool_result", name: "learn_quiz", payload: { id: "c1", name: "learn_quiz", ok: false, content: "ERROR: switch mode" } })]);
    assert.equal(classifyToolPairs(failed).get(failed[0].key), "process");
  });
});

describe("learn change events", () => {
  /** 同进程桌面的事件在 event 里，隔离进程的在顶层，两种都读得出范围、对象和种类。 */
  it("reads both envelopes and ignores other kinds", () => {
    const nested = learnChangeOf({ name: "yoyo:item", data: { seq: 3, type: "system", event: { type: "system", source: "learn", item_kind: "learn", payload: { scope: "path", id: "p1", kind: "graded" } } } });
    assert.deepEqual(nested, { scope: "path", id: "p1", kind: "graded" });
    const flat = learnChangeOf({ name: "yoyo:item", data: { seq: 4, type: "system", item_kind: "learn", payload: { scope: "practice", id: "", kind: "reviewed" } } });
    assert.deepEqual(flat, { scope: "practice", id: "", kind: "reviewed" });
    assert.equal(learnChangeOf({ name: "yoyo:item", data: { type: "system", item_kind: "video_route", payload: { id: "x" } } }), null);
    assert.equal(learnChangeOf({ name: "yoyo:item", data: { seq: 5, type: "assistant" } }), null);
    assert.equal(learnChangeOf(null), null);
  });
});

describe("learn formatting", () => {
  /** 模板只换给了值的名字；百分比夹在 0 到 100。 */
  it("fills templates and clamps percentages", () => {
    assert.equal(fill("{n} 项到期，{m} 留着", { n: 3 }), "3 项到期，{m} 留着");
    assert.equal(percent(0.826), 83);
    assert.equal(percent(1.4), 100);
    assert.equal(percent(undefined), 0);
  });

  /** 到期日按日历日算：当天和已过期都说今天，之后用相对的说法，太远写日期。 */
  it("labels due dates by calendar day", () => {
    const now = new Date(2026, 10, 12, 22, 0, 0);
    assert.equal(dayDistance(new Date(2026, 10, 12, 23, 30).toISOString(), now), 0);
    assert.equal(dayDistance(new Date(2026, 10, 13, 0, 5).toISOString(), now), 1);
    assert.equal(dayDistance("not a date", now), null);
    assert.equal(whenLabel(new Date(2026, 10, 10).toISOString(), "zh-CN", "今天", now), "今天");
    assert.equal(whenLabel(new Date(2026, 10, 13, 9).toISOString(), "zh-CN", "今天", now), "明天");
    assert.equal(whenLabel(new Date(2026, 10, 15, 9).toISOString(), "zh-CN", "今天", now), "3天后");
    assert.equal(whenLabel(new Date(2026, 10, 15, 9).toISOString(), "en", "today", now), "in 3 days");
    assert.match(whenLabel(new Date(2027, 2, 1).toISOString(), "en", "today", now), /Mar/);
    assert.equal(whenLabel(undefined, "en", "today", now), "");
  });
});
