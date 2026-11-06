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

  /** 工人已经说过话时，完成报告仍留在会话末尾。 */
  it("appends the closing report after the worker has spoken", () => {
    const rows = ticket([{ id: "in-7", text: "先定方向" }]);
    rows.assignments = [{ id: "asg-1", state: "ended", lines: [{ state: "ready", summary: "可以试玩了" }] }];
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "r1", role: "assistant", text: "先写页面" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先定方向",
      "先写页面",
      "可以试玩了",
    ]);
  });

  /** 派工正文和输入只差换行时，说明留在工人开口之前，不补到记录末尾。 */
  it("keeps a brief in front of the work when only whitespace differs", () => {
    const rows = ticket([{ id: "in-1", text: "背景\n\n目标产物：一份清单" }]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "背景\n目标产物：一份清单" },
      { id: "r1", role: "assistant", text: "开始查" },
      { id: "c1", role: "tool_call", name: "web_search", text: "{}" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "背景\n\n目标产物：一份清单",
      "开始查",
      "{}",
    ]);
  });

  /** 对不上的新说明出现在它发生的位置，后面的回复不再跑到它前面。 */
  it("shows an unmatched assignment before the reply that followed it", () => {
    const rows = ticket([{ id: "in-1", text: "另一件还没送进日志的事" }]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先去看公开的招聘页" },
      { id: "r1", role: "assistant", text: "正在打开" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先去看公开的招聘页",
      "正在打开",
      "另一件还没送进日志的事",
    ]);
  });

  /** 后来的派工快照又带上已经说过的那句时，不再插进后面的回复中间。 */
  it("does not repeat a line the worker was already told", () => {
    const rows = ticket([
      { id: "in-7", text: "先定方向" },
      { id: "in-23", text: "直接向下进行" },
    ]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "h1", role: "user", source: "human", text: "直接向下进行" },
      { id: "r1", role: "assistant", text: "继续做" },
      { id: "a2", role: "user", source: "assignment", text: "先定方向\n直接向下进行" },
      { id: "r2", role: "assistant", text: "核对现状" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先定方向",
      "直接向下进行",
      "继续做",
      "核对现状",
    ]);
  });

  /** 夹在两次真实说明中间、工人日志里没有的系统备注，不补到最新一句后面。 */
  it("leaves an unmatched note where it was skipped instead of after the latest line", () => {
    const rows = ticket([
      { id: "in-7", text: "先定方向" },
      { id: "in-36", text: "policy off -> observe" },
      { id: "in-42", text: "把天数延长" },
    ]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "r1", role: "assistant", text: "看过了" },
      { id: "h1", role: "user", source: "human", text: "把天数延长" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先定方向",
      "看过了",
      "把天数延长",
    ]);
  });

  /** 更早的完成报告留在它发生的位置，不再整段贴到最新一句后面。最近一次收束还在，并且日志里没有这句话时，仍留在末尾。 */
  it("keeps only the latest closing report after the log", () => {
    const rows = ticket([{ id: "in-7", text: "先定方向" }]);
    rows.assignments = [
      { id: "asg-1", state: "ended", lines: [{ state: "blocked", summary: "先确认方向" }] },
      { id: "asg-2", state: "ended", lines: [{ state: "ready", summary: "可以试玩了" }] },
      { id: "asg-3", state: "running", lines: [{ state: "failed", summary: "连接中断" }] },
    ];
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "r1", role: "assistant", text: "先写页面" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先定方向",
      "先写页面",
      "连接中断",
    ]);
  });

  /** 重启说明夹在两次说明中间时，后面的新句子仍然对在它发生的位置。 */
  it("peels the next instruction across a host restart note", () => {
    const rows = ticket([
      { id: "in-7", text: "先定方向" },
      { id: "in-15", text: "宿主重启了。先看工作目录、git 和 GitHub，不要重复已经做过的外部动作。" },
      { id: "in-18", text: "新建吧" },
    ]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "r1", role: "assistant", text: "看过了" },
      { id: "a2", role: "user", source: "assignment", text: "先定方向\n新建吧" },
      { id: "r2", role: "assistant", text: "按你的指示新建" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), [
      "先定方向",
      "看过了",
      "新建吧",
      "按你的指示新建",
    ]);
  });

  /** 重启说明不再出现在任务会话里。 */
  it("drops the host restart note", () => {
    const rows = ticket([
      { id: "in-7", text: "先定方向" },
      { id: "in-8", text: "宿主重启了。先看工作目录、git 和 GitHub，不要重复已经做过的外部动作。" },
    ]);
    const lines: TaskLine[] = [
      { id: "a1", role: "user", source: "assignment", text: "先定方向" },
      { id: "n1", role: "user", text: "宿主重启了。先看工作目录、git 和 GitHub，不要重复已经做过的外部动作。" },
    ];
    assert.deepEqual(taskSessionItems(rows, lines).map((item) => item.text), ["先定方向"]);
  });
});
