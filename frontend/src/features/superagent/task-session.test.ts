import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Ticket } from "./desk-model.ts";
import { taskSessionItems, type TaskLine } from "./task-session.ts";

function ticket(inputs: { id: string; text: string }[]): Ticket {
  return {
    id: "tkt-4",
    stage: "working",
    action: "",
    person: false,
    origin: "",
    title: "先定方向",
    inputs: inputs.map((row) => ({ ...row, human: true, delivered: true })),
  };
}

describe("task session order", () => {
  /** 短句不能借旧句子里的同样几个字，插到更早的回复前面。 */
  it("keeps a shorter later instruction after the turn that came before it", () => {
    const rows = ticket([
      { id: "in-7", text: "先定方向" },
      { id: "in-12", text: "找不到的话就新建吧" },
      { id: "in-18", text: "新建吧" },
      { id: "in-19", text: "咕咕便利店改编：新建 gugu-mart 并写初始文档" },
    ]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "r1", role: "assistant", text: "目录是空的" },
      { id: "a2", role: "user", source: "assignment", text: "先定方向\n找不到的话就新建吧" },
      { id: "r2", role: "assistant", text: "先核对现状" },
      { id: "c2", role: "tool_call", name: "shell", text: "{}" },
      { id: "a3", role: "user", source: "assignment", text: "先定方向\n找不到的话就新建吧\n新建吧\n咕咕便利店改编：新建 gugu-mart 并写初始文档" },
      { id: "r3", role: "assistant", text: "按你的指示新建" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先定方向",
      "目录是空的",
      "找不到的话就新建吧",
      "先核对现状",
      "{}",
      "新建吧",
      "咕咕便利店改编：新建 gugu-mart 并写初始文档",
      "按你的指示新建",
    ]);
  });

  /** 还没写进日志的新指令留在末尾，不插进已经发生的回复中间。 */
  it("appends an instruction the worker has not read yet", () => {
    const rows = ticket([
      { id: "in-7", text: "先定方向" },
      { id: "in-20", text: "再补一张图" },
    ]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "r1", role: "assistant", text: "看过了" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先定方向",
      "看过了",
      "再补一张图",
    ]);
  });
});
