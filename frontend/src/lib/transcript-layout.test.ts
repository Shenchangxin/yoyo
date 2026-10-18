import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Item, ItemType } from "./protocol.ts";
import { clipPreview, hydrateLayout, layoutRows, type AgentPart, type LayoutRow } from "./transcript-layout.ts";
import { presentationPolicyFor } from "./presentation-policy.ts";

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

  it("a running turn never promotes narration to a letter before tools arrive", () => {
    const rows = layoutRows([
      user,
      assistant("a0", "I'll start by loading the pipeline."),
    ], { running: true });
    const parts = agentParts(rows);
    assert.deepEqual(parts.map((p) => p.kind), ["stream"]);
    assert.ok(parts[0].kind === "stream" && parts[0].item.key === "a0");
  });

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

  it("verbose never folds", () => {
    const parts = agentParts(layoutRows([
      user,
      assistant("a0", "reading first"),
      ...read("r1"),
      assistant("a1", "done"),
    ], { policy: presentationPolicyFor("verbose") }));
    assert.ok(parts[0].kind === "work" && !parts[0].folded);
  });

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
  it("keeps short bodies intact", () => {
    assert.deepEqual(clipPreview("a\nb\nc"), { text: "a\nb\nc", hidden: 0 });
  });

  it("keeps a head and a tail when the body is long", () => {
    const text = ["1", "2", "3", "4", "5", "6", "7", "8"].join("\n");
    const got = clipPreview(text, 5);
    assert.equal(got.hidden, 3);
    assert.equal(got.text, "1\n2\n3\n…\n7\n8");
  });
});
