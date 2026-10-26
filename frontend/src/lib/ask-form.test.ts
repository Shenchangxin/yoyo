/**
 * 拥有：提问表单的建议项、空闲答复，以及只显示本会话那一张卡。
 * 不拥有：卡片的渲染。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { askCanIdle, chosenAnswer, idleAnswer, parseAskCommand, pendingAsk } from "./ask-form.ts";
import type { Approval } from "./protocol.ts";

const formJSON = JSON.stringify({
  title: "方向",
  questions: [
    {
      id: "direction",
      prompt: "按这个方向改吗？",
      recommended: "yes",
      options: [
        { id: "yes", label: "按这个方向" },
        { id: "no", label: "先别改操作" },
      ],
    },
    {
      id: "look",
      prompt: "画面？",
      options: [
        { id: "ink", label: "暗黑水墨" },
        { id: "bright", label: "更亮" },
      ],
    },
  ],
});

describe("ask form", () => {
  /** 一份载荷里的每道题都在。没写建议时用第一项。空闲答复带上建议和空闲说明。 */
  it("pages every question in one payload and defaults the suggestion", () => {
    const form = parseAskCommand(formJSON);
    assert.equal(form.questions.length, 2);
    assert.equal(form.questions[1].recommended, "ink");
    assert.equal(askCanIdle(form), true);
    assert.match(idleAnswer(form), /direction: yes — 按这个方向/);
    assert.match(idleAnswer(form), /look: ink — 暗黑水墨/);
    assert.match(idleAnswer(form), /note: operator idle; used recommended/);
  });

  /** 纯文本问题不能空闲。操作者后来的选择写进答复，并注明已作答。 */
  it("keeps an operator's later choice and does not idle a free-text question", () => {
    const form = parseAskCommand("登录完成了吗？");
    assert.equal(askCanIdle(form), false);
    const answered = chosenAnswer(parseAskCommand(formJSON), { direction: "no", look: "bright" }, {});
    assert.match(answered, /direction: no — 先别改操作/);
    assert.match(answered, /note: operator answered/);
  });

  /** 只取出本会话的 ask_user，shell 和其他会话不算。 */
  it("shows one card for this session only", () => {
    const mine: Approval = { id: "a", action: "ask_user", command: formJSON, path: "", level: "", sessionId: "s1" };
    const other: Approval = { id: "b", action: "ask_user", command: formJSON, path: "", level: "", sessionId: "s2" };
    const shell: Approval = { id: "c", action: "shell", command: "rm", path: "", level: "", sessionId: "s1" };
    assert.equal(pendingAsk([shell, other, mine], "s1")?.id, "a");
  });
});
