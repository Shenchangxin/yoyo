import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Item, ItemType } from "./protocol.ts";
import { layoutRows, type AgentPart, type LayoutRow } from "./transcript-layout.ts";

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
const read = (id: string) => [
  item({ key: `${id}:call`, type: "tool_call", name: "read_file", payload: { id, name: "read_file", arguments: '{"path":"a.ts"}' } }),
  item({ key: `${id}:result`, type: "tool_result", name: "read_file", text: "contents", payload: { id, name: "read_file" } }),
];
const write = (id: string) => [
  item({
    key: `${id}:call`, type: "tool_call", name: "write_file",
    payload: { id, name: "write_file", arguments: '{"path":"out.ts","content":"x"}' },
  }),
  item({ key: `${id}:result`, type: "tool_result", name: "write_file", text: "ok", payload: { id, name: "write_file" } }),
];
const assistant = (key: string, text: string) => item({ key, type: "assistant", text });

function agentParts(rows: LayoutRow[]): AgentPart[] {
  const agent = rows.find((r) => r.kind === "agent");
  assert.ok(agent && agent.kind === "agent", "expected an agent row");
  return agent.parts;
}

describe("layoutRows: one turn stays one agent row", () => {
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
  it("steps around a file write fold into one rail, then the files, then the letter", () => {
    const rows = layoutRows([
      user,
      assistant("a1", "reading"),
      ...read("r1"),
      ...write("w1"),
      ...read("r2"),
      assistant("a2", "done"),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["process", "artifact", "item"]);
    const artifact = parts[1];
    assert.ok(artifact.kind === "artifact" && artifact.items[0].payload.id === "w1");
  });

  it("a mid-turn checkpoint sits where it happened, between rail and letter", () => {
    const rows = layoutRows([
      user,
      ...read("r1"),
      item({ key: "cp1", type: "compaction", payload: { kind: "checkpoint" } }),
      ...read("r2"),
      assistant("a2", "done"),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["process", "item", "process", "item"]);
    const divider = parts[1];
    assert.ok(divider.kind === "item" && divider.item.type === "compaction");
  });
});

describe("layoutAgentParts: only the final text is a letter", () => {
  it("narration before a tool call stays in the rail, not a second answer", () => {
    const rows = layoutRows([
      user,
      assistant("a0", "I'll start by loading the pipeline."),
      ...read("r1"),
      assistant("a1", "done"),
    ]);
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["process", "item"]);
    const letter = parts[1];
    assert.ok(letter.kind === "item" && letter.item.key === "a1");
  });

  it("a turn still running on a tool call shows no letter for its last narration", () => {
    const rows = layoutRows([
      user,
      assistant("a0", "reading the source first"),
      ...read("r1"),
    ]);
    assert.deepEqual(agentParts(rows).map((p) => p.kind), ["process"]);
  });

  it("closing a plan step after the answer keeps the answer a letter", () => {
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
    assert.ok(parts.some((p) => p.kind === "item" && p.item.key === "a1"), "answer must stay visible");
  });

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
});
