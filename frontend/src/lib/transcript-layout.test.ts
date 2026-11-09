import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Item, ItemType } from "./protocol.ts";
import {
  clipPreview,
  foldGroupStatus,
  hasConversation,
  hydrateLayout,
  isGroupCardRow,
  isMentionInject,
  isWakeReceipt,
  layoutRows,
  type AgentPart,
  type LayoutRow,
} from "./transcript-layout.ts";
import { presentationPolicyFor } from "./presentation-policy.ts";

/**
 * 拥有：一个操作者回合如何收成一行、过程轨、产出和折叠的约定，以及主会话里任务组卡片、成员状态和唤醒回执排在哪里。
 * 不拥有：排版函数的实现，以及这些行的渲染。
 */

/** 补齐转录条目的缺省字段。 */
function item(p: Partial<Item> & { key: string; type: ItemType }): Item {
  return {
    sessionId: "s",
    source: "",
    ts: "2026-10-13T00:00:00.000Z",
    text: "",
    name: "",
    delta: false,
    payload: {},
    ...p,
  } as Item;
}

const user = item({ key: "u1", type: "user", text: "build it", source: "user", payload: { id: "t1" } });
/** 一次读文件的调用和结果。 */
const read = (id: string) => [
  item({ key: `${id}:call`, type: "tool_call", name: "read_file", payload: { id, name: "read_file", arguments: '{"path":"a.ts"}' } }),
  item({ key: `${id}:result`, type: "tool_result", name: "read_file", text: "contents", payload: { id, name: "read_file" } }),
];
/** 一次写文件的调用和结果。 */
const write = (id: string) => [
  item({
    key: `${id}:call`, type: "tool_call", name: "write_file",
    payload: { id, name: "write_file", arguments: '{"path":"out.ts","content":"x"}' },
  }),
  item({ key: `${id}:result`, type: "tool_result", name: "write_file", text: "ok", payload: { id, name: "write_file" } }),
];
/** 一条助手正文。 */
const assistant = (key: string, text: string) => item({ key, type: "assistant", text });

/** 取出第一行智能体里的部件。 */
function agentParts(rows: LayoutRow[]): AgentPart[] {
  const agent = rows.find((r) => r.kind === "agent");
  assert.ok(agent && agent.kind === "agent", "expected an agent row");
  return agent.parts;
}

describe("layoutRows: one turn stays one agent row", () => {
  /** 回合中的检查点和批准不把这一轮拆成两行。 */
  it("a mid-turn checkpoint and an approval do not split the turn", () => {
    const rows = layoutRows([
      user,
      assistant("a1", "reading first"),
      ...read("c1"),
      item({ key: "ap1", type: "approval", payload: { id: "ap1" } }),
      item({ key: "cp1", type: "compaction", payload: { kind: "checkpoint" } }),
      ...read("c2"),
      assistant("a2", "done"),
    ]);
    assert.deepEqual(rows.map((r) => r.kind), ["user", "agent"]);
  });

  /** 后台命令完成不另开一条用户消息，也不把这一轮拆开。 */
  it("a finished background job does not open a user row", () => {
    const rows = layoutRows([
      user,
      ...read("c1"),
      item({
        key: "job1",
        type: "user",
        source: "job",
        name: "job",
        text: "job call_1 finished\n快照已保存\n",
      }),
      assistant("a2", "done"),
    ]);
    assert.deepEqual(rows.map((r) => r.kind), ["user", "agent"]);
    assert.equal(rows.filter((r) => r.kind === "user").length, 1);
  });

  /** 没有打开的回合时，检查点单独成一条分隔。 */
  it("a checkpoint with no open turn stays a solo divider", () => {
    const rows = layoutRows([
      user,
      item({ key: "cp0", type: "compaction", payload: { kind: "checkpoint" } }),
      assistant("a1", "ok"),
    ]);
    assert.deepEqual(rows.map((r) => r.kind), ["user", "solo", "agent"]);
  });
});

describe("layoutRows: one rail and one artifact list per turn", () => {
  /** 写文件前后的步骤收进工作块，最后的回答单独成信。 */
  it("steps around a file write fold into work, then the letter", () => {
    const rows = layoutRows([
      user,
      assistant("a1", "reading"),
      ...read("r1"),
      ...write("w1"),
      ...read("r2"),
      assistant("a2", "done"),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["work", "item"]);
    assert.ok(parts[0].kind === "work");
    assert.deepEqual(parts[0].parts.map((p) => p.kind), ["process", "artifact", "process"]);
    const process = parts[0].parts[0];
    assert.ok(process.kind === "process" && process.items.some((it) => it.key === "a1"));
    const artifact = parts[0].parts[1];
    assert.ok(artifact.kind === "artifact" && artifact.items[0].payload.id === "w1");
    assert.ok(parts[1].kind === "item" && parts[1].item.key === "a2");
  });

  /** 回合中的检查点留在它发生的位置，在信之前的工作块里。 */
  it("a mid-turn checkpoint sits where it happened, inside work before the letter", () => {
    const rows = layoutRows([
      user,
      ...read("r1"),
      item({ key: "cp1", type: "compaction", payload: { kind: "checkpoint" } }),
      ...read("r2"),
      assistant("a2", "done"),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["work", "item"]);
    assert.ok(parts[0].kind === "work");
    assert.deepEqual(parts[0].parts.map((p) => p.kind), ["process", "item", "process"]);
    const divider = parts[0].parts[1];
    assert.ok(divider.kind === "item" && divider.item.type === "compaction");
  });
});

describe("layoutAgentParts: process groups split on an assistant reply", () => {
  /** 步骤里的回答折进过程轨，只留最后一条紧凑的信。 */
  it("a step reply folds into the process rail, leaving one compact letter", () => {
    const rows = layoutRows([
      user,
      assistant("a0", "I'll start by loading the pipeline."),
      ...read("r1"),
      assistant("a1", "done"),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["work", "item"]);
    assert.ok(parts[0].kind === "work");
    assert.deepEqual(parts[0].parts.map((p) => p.kind), ["process"]);
    assert.ok(parts[0].parts[0].kind === "process" && parts[0].parts[0].items.some((it) => it.key === "a0"));
    assert.ok(parts[1].kind === "item" && parts[1].item.key === "a1");
  });

  /** 后面没有紧凑回答时，工具之前已经收束的回复留在过程轨里。 */
  it("a settled reply before tools stays inside the process rail when no compact answer follows", () => {
    const rows = layoutRows([
      user,
      assistant("a0", "reading the source first"),
      ...read("r1"),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["work"]);
    assert.ok(parts[0].kind === "work");
    assert.deepEqual(parts[0].parts.map((p) => p.kind), ["process"]);
    assert.ok(parts[0].parts[0].kind === "process" && parts[0].parts[0].items.some((it) => it.key === "a0"));
  });

  /** 回答之后再关计划步骤，回答仍留在过程轨或信上，不会消失。 */
  it("closing a plan step after the answer keeps the answer in the rail, not lost", () => {
    const plan = item({
      key: "p1:call", type: "tool_call", name: "update_plan",
      payload: { id: "p1", name: "update_plan", arguments: '{"plan":[]}' },
    });
    const rows = layoutRows([
      user,
      ...read("r1"),
      assistant("a1", "done"),
      plan,
      item({ key: "p1:result", type: "tool_result", name: "update_plan", text: "ok", payload: { id: "p1", name: "update_plan" } }),
    ]);
    const parts = agentParts(rows);
    const nested = parts.flatMap((p) => p.kind === "work" ? p.parts : [p]);
    const inRail = nested.some((p) => p.kind === "process" && p.items.some((it) => it.key === "a1"));
    const asLetter = nested.some((p) => (p.kind === "item" || p.kind === "stream") && p.item.key === "a1");
    assert.ok(inRail || asLetter, "answer must stay visible");
  });

  /** 末尾空的助手消息不会盖住上一条真正的回答。 */
  it("an empty trailing assistant does not hide the last real answer", () => {
    const rows = layoutRows([
      user,
      assistant("a1", "done"),
      assistant("a2", ""),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["item"]);
    assert.ok(parts[0].kind === "item" && parts[0].item.key === "a1");
  });

  /** 进行中的回合在工具到达前，不把叙述提升成信。 */
  it("a running turn never promotes narration to a letter before tools arrive", () => {
    const rows = layoutRows([
      user,
      assistant("a0", "I'll start by loading the pipeline."),
    ], { running: true });
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["stream"]);
    assert.ok(parts[0].kind === "stream" && parts[0].item.key === "a0");
  });

  /** 工具回合进行中时，先前的回复留着，打开的助手按流渲染。 */
  it("a running tool round keeps the prior reply and streams the open assistant", () => {
    const rows = layoutRows([
      user,
      assistant("a0", "reading first"),
      ...read("r1"),
      item({ key: "a1", type: "assistant", text: "still working", delta: true }),
    ], { running: true });
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["work", "stream"]);
    assert.ok(parts[1].kind === "stream" && parts[1].item.key === "a1");
  });

  /** 同一回合收束后，流的键仍是那封信的键。 */
  it("settling the same turn keeps the stream key as the letter", () => {
    const items = [
      user,
      assistant("a0", "reading first"),
      ...read("r1"),
      assistant("a1", "done"),
    ];
    const live = agentParts(layoutRows(items, { running: true }));
    assert.deepEqual(live.map((p) => p.kind), ["work", "item"]);
    const settled = agentParts(layoutRows(items));
    assert.deepEqual(settled.map((p) => p.kind), ["work", "item"]);
    assert.ok(settled[1].kind === "item" && settled[1].item.key === "a1");
    assert.equal(live[1].key, settled[1].key);
  });
});

describe("work details folding", () => {
  /** 标准策略在紧凑回答之前折叠已完成的工作。 */
  it("standard policy folds completed work before the compact answer", () => {
    const parts = agentParts(layoutRows([
      user,
      assistant("a0", "reading first"),
      ...read("r1"),
      assistant("a1", "done"),
    ], { policy: presentationPolicyFor("standard") }));
    assert.ok(parts[0].kind === "work" && parts[0].folded);
    assert.ok(parts[1].kind === "item" && parts[1].item.key === "a1");
  });

  /** 啰嗦档不折叠工作块。 */
  it("verbose never folds", () => {
    const parts = agentParts(layoutRows([
      user,
      assistant("a0", "reading first"),
      ...read("r1"),
      assistant("a1", "done"),
    ], { policy: presentationPolicyFor("verbose") }));
    assert.ok(parts[0].kind === "work" && !parts[0].folded);
  });

  /** 下一条输入的时机让最近一个已完成回合保持展开。 */
  it("next-input leaves the latest completed turn open", () => {
    const parts = agentParts(layoutRows([
      user,
      assistant("a0", "reading first"),
      ...read("r1"),
      assistant("a1", "done"),
    ], { policy: presentationPolicyFor("standard", "next-input") }));
    assert.ok(parts[0].kind === "work" && !parts[0].folded);
  });
});

describe("layoutRows: desk notices stay their own row", () => {
  /** 简报跟在已有回复后面时，不把上一回合收进过程轨。 */
  it("keeps a pushed brief beside the previous answer", () => {
    const rows = layoutRows([
      user,
      assistant("a1", "done"),
      item({
        key: "brief",
        type: "assistant",
        source: "desk",
        text: "早上这一份在这里。",
        payload: { kind: "brief", chip: "晨间简报 · 11月2日", id: "brief:morning:2026-11-02" },
      }),
    ]);
    assert.equal(rows.length, 3);
    assert.equal(rows[2].kind, "agent");
    if (rows[2].kind === "agent") {
      assert.equal(rows[2].parts.length, 1);
      const part = rows[2].parts[0];
      assert.equal(part.kind, "item");
      if (part.kind === "item") assert.equal(part.item.payload?.chip, "晨间简报 · 11月2日");
    }
    if (rows[1].kind === "agent") {
      const answer = rows[1].parts.find((part) => part.kind === "item" || part.kind === "stream");
      assert.ok(answer && (answer.kind === "item" || answer.kind === "stream"));
      if (answer && (answer.kind === "item" || answer.kind === "stream")) assert.equal(answer.item.text, "done");
    }
  });

  /** 工人等人接手的报告跟在回复后面，自己成一行。 */
  it("keeps a worker handoff beside the previous answer", () => {
    const rows = layoutRows([
      user,
      assistant("a1", "已经开工。"),
      item({
        key: "ask",
        type: "assistant",
        source: "worker",
        text: "找不到 gugu-mart。请确认路径。",
        payload: { kind: "handoff", ticket: "tkt-4" },
      }),
    ]);
    assert.equal(rows.length, 3);
    assert.equal(rows[2].kind, "agent");
    if (rows[2].kind === "agent") {
      assert.equal(rows[2].parts.length, 1);
      const part = rows[2].parts[0];
      assert.equal(part.kind, "item");
      if (part.kind === "item") assert.equal(part.item.text, "找不到 gugu-mart。请确认路径。");
    }
  });

  /** 同一张工单后一次提问留下「需要你」，前一次改成普通状态。 */
  it("folds an older handoff once a newer question arrives", () => {
    const rows = layoutRows([
      item({
        key: "old",
        type: "assistant",
        source: "worker",
        text: "找不到 gugu-mart。",
        payload: { kind: "handoff", ticket: "tkt-4" },
      }),
      item({
        key: "new",
        type: "assistant",
        source: "worker",
        text: "是自立题材，还是既有原作？",
        payload: { kind: "handoff", ticket: "tkt-4" },
      }),
    ]);
    const kinds = rows.map((row) => {
      if (row.kind !== "agent") return "";
      const part = row.parts[0];
      return part.kind === "item" ? String(part.item.payload?.kind || "") : "";
    });
    assert.deepEqual(kinds, ["status", "handoff"]);
  });

  /** 用户能读到的简报算消息记录。看不见的系统事件不算。 */
  it("treats a visible desk note as a conversation and ignores a system event", () => {
    const brief = item({
      key: "brief",
      type: "assistant",
      source: "desk",
      text: "早上这一份在这里。",
      payload: { kind: "brief" },
    });
    const system = item({ key: "sys", type: "system", text: "prompt", payload: { kind: "prompt" } });
    assert.equal(hasConversation([system]), false);
    assert.equal(hasConversation([brief, system]), true);
    assert.equal(hasConversation([user, brief]), true);
  });
});

/** 一行里所有条目的 key，过程轨和工作块里的也算。 */
function rowKeys(row: LayoutRow): string[] {
  if (row.kind !== "agent") return [row.item.key];
  const out: string[] = [];
  const walk = (parts: AgentPart[]) => {
    for (const part of parts) {
      if (part.kind === "item" || part.kind === "stream") out.push(part.item.key);
      else if (part.kind === "work") walk(part.parts);
      else out.push(...part.items.map((it) => it.key));
    }
  };
  walk(row.parts);
  return out;
}

describe("layoutRows: task groups in the main chat", () => {
  /** task_split 时写进主会话的任务组卡片。 */
  const card = (group: string) => item({
    key: `card:${group}`,
    type: "assistant",
    source: "worker",
    text: "任务组「todo」· 分工 2 份",
    payload: { id: `group:${group}`, kind: "group", group, title: "todo", mode: "split", members: [] },
  });
  /** 成员写回的进行中或做完。 */
  const status = (key: string, group: string) => item({
    key, type: "assistant", source: "worker", text: "接口写好了。", payload: { kind: "status", ticket: "tkt-1", state: "ready", group },
  });
  const split = [
    item({ key: "split:call", type: "tool_call", name: "task_split", payload: { id: "split", name: "task_split", arguments: "{}" } }),
    item({ key: "split:result", type: "tool_result", name: "task_split", text: "开了任务组 grp-1", payload: { id: "split", name: "task_split" } }),
  ];
  const wake = item({
    key: "wake",
    type: "context_injection",
    source: "wake",
    name: "wake",
    text: "这是任务组 grp-1「todo」的回报，不是人说的话。",
    payload: { name: "wake", group: "grp-1", state: "done", title: "todo" },
  });

  /** 卡片在 task_split 这次调用里写下，排在路由这一轮的回复后面；这一轮还在跑时也一样，路由那一行仍是最后一轮的内容。 */
  it("puts the card after the router's reply", () => {
    for (const running of [false, true]) {
      const rows = layoutRows([user, split[0], card("grp-1"), split[1], assistant("a1", "拆成了两份。")], { running });
      assert.deepEqual(rows.map((row) => row.kind), ["user", "agent", "agent"], `running ${running}`);
      assert.deepEqual(rowKeys(rows[1]), ["split:call", "split:result", "a1"], `running ${running}`);
      assert.equal(isGroupCardRow(rows[1]), false);
      assert.equal(isGroupCardRow(rows[2]), true);
      assert.ok(rows[2].kind === "agent" && rows[2].copyText === "", "a card row has nothing to copy");
    }
  });

  /** 前面没有路由的条目时卡片直接成行；后面来了别的单独行时，卡片先落在路由那一行后面。 */
  it("places the card before a later notice", () => {
    assert.deepEqual(layoutRows([user, card("grp-1")]).map(isGroupCardRow), [false, true]);
    const ask = item({ key: "ask", type: "assistant", source: "worker", text: "用哪个数据库？", payload: { kind: "handoff", ticket: "tkt-2", group: "grp-1" } });
    const rows = layoutRows([user, split[0], card("grp-1"), split[1], ask]);
    assert.deepEqual(rows.map(rowKeys), [["u1"], ["split:call", "split:result"], ["card:grp-1"], ["ask"]]);
  });

  /** 有卡片的任务组，成员的进行中和做完不再单独成行；没有卡片的任务组和单张工单照旧。 */
  it("folds member status into the card", () => {
    const items = [user, card("grp-1"), status("s1", "grp-1"), status("s2", "grp-9"), item({ ...status("s3", ""), payload: { kind: "status", ticket: "tkt-7", state: "ready" } })];
    assert.deepEqual(foldGroupStatus(items).map((it) => it.key), ["u1", "card:grp-1", "s2", "s3"]);
    assert.deepEqual(layoutRows(items).map(rowKeys), [["u1"], ["card:grp-1"], ["s2"], ["s3"]]);
    const plain = [user, status("s2", "grp-9")];
    assert.equal(foldGroupStatus(plain), plain);
  });

  /** 同一成员前一次提问被新的提问盖过后变成状态，也一起折进卡片；最新的提问照常成行。 */
  it("folds an older member question once a newer one arrives", () => {
    const ask = (key: string) => item({ key, type: "assistant", source: "worker", text: key, payload: { kind: "handoff", ticket: "tkt-2", group: "grp-1" } });
    assert.deepEqual(layoutRows([card("grp-1"), ask("q1"), ask("q2")]).map(rowKeys), [["card:grp-1"], ["q2"]]);
  });

  /** 唤醒回执单独成行，不折进用户气泡，也不进路由那一轮；路由的回复跟在后面。 */
  it("gives the wake receipt its own row before the router's reply", () => {
    assert.equal(isWakeReceipt(wake), true);
    assert.equal(isMentionInject(wake), false);
    const mention = item({ key: "m", type: "context_injection", source: "mention", text: "@a.ts" });
    assert.equal(isMentionInject(mention), true);
    const rows = layoutRows([user, assistant("a1", "拆成了两份。"), wake, assistant("a2", "两份都做完了。")]);
    assert.deepEqual(rows.map((row) => row.kind), ["user", "agent", "solo", "agent"]);
    assert.deepEqual(rows.map(rowKeys), [["u1"], ["a1"], ["wake"], ["a2"]]);
  });
});

describe("hydrateLayout", () => {
  /** 流式正文换新时只补文本，座位不重建。 */
  it("patches streaming text without rebuilding seats", () => {
    const rows = layoutRows([
      user,
      item({ key: "a0", type: "assistant", text: "he", delta: true, payload: { streaming: true } }),
    ], { running: true });
    const next = [
      user,
      item({ key: "a0", type: "assistant", text: "hello", delta: true, payload: { streaming: true } }),
    ];
    const hydrated = hydrateLayout(rows, next);
    assert.equal(hydrated[1].kind, "agent");
    if (hydrated[1].kind === "agent") {
      const part = hydrated[1].parts[0];
      assert.ok(part.kind === "stream" || part.kind === "item");
      if (part.kind === "stream" || part.kind === "item") assert.equal(part.item.text, "hello");
    }
  });
});

describe("clipPreview", () => {
  /** 短正文原样保留。 */
  it("keeps short bodies intact", () => {
    assert.deepEqual(clipPreview("a\nb\nc"), { text: "a\nb\nc", hidden: 0 });
  });

  /** 长正文保留头尾，中间记为隐藏行。 */
  it("keeps a head and a tail when the body is long", () => {
    const text = ["1", "2", "3", "4", "5", "6", "7", "8"].join("\n");
    const got = clipPreview(text, 5);
    assert.equal(got.hidden, 3);
    assert.equal(got.text, "1\n2\n3\n…\n7\n8");
  });
});
