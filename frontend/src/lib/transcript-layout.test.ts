import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Item, ItemType } from "./protocol.ts";
import { clipPreview, hydrateLayout, layoutRows, type AgentPart, type LayoutRow } from "./transcript-layout.ts";
import { presentationPolicyFor } from "./presentation-policy.ts";

/**
 * 拥有：一个操作者回合如何收成一行、过程轨、产出和折叠的约定。
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
