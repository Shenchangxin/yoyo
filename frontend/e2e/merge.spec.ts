import { expect, test } from "@playwright/test";
import { foldLiveIntoSeed, itemFromEvent, mergeItem, mergePendingUsers, replayEvents, unwrapEvent } from "../src/lib/stream-fold";
import { layoutRows } from "../src/lib/transcript-layout";
import { latestTaskPlan, parsePlanText } from "../src/lib/plan";
import { toolDetail, toolName, isArtifactTool, isToolFailed } from "../src/lib/tool-summary";
import { artifactPreviewOpen, artifactShouldShow, artifactView } from "../src/lib/artifact-preview";
import type { Item } from "../src/lib/protocol";

function liveUser(text: string, extra?: Record<string, any>): Item {
  return itemFromEvent({
    type: "user",
    session_id: "s",
    source: "user",
    payload: { text, ...extra },
  });
}

function uiUser(text: string): Item {
  return {
    key: `ui:s:${text}:1`,
    type: "user",
    sessionId: "s",
    source: "ui",
    ts: new Date().toISOString(),
    text,
    name: "",
    delta: false,
    payload: { text },
  };
}

test("live second user is kept even with blank timestamps and no payload id", () => {
  let list: Item[] = [];
  list = mergeItem(list, liveUser("first question"));
  list = mergeItem(list, itemFromEvent({
    type: "assistant",
    session_id: "s",
    payload: { text: "first answer", id: "s:r1" },
  }));
  list = mergeItem(list, uiUser("second question"));
  list = mergeItem(list, liveUser("second question"));
  const users = list.filter((x) => x.type === "user");
  const assistants = list.filter((x) => x.type === "assistant");
  expect(users.map((x) => x.text)).toEqual(["first question", "second question"]);
  expect(assistants.map((x) => x.text)).toEqual(["first answer"]);
});

test("same prompt on turn two still adds a second user bubble", () => {
  let list: Item[] = [];
  list = mergeItem(list, liveUser("ok", { id: "s:user:1" }));
  list = mergeItem(list, itemFromEvent({
    type: "assistant",
    session_id: "s",
    payload: { text: "one", id: "s:r1" },
  }));
  list = mergeItem(list, uiUser("ok"));
  list = mergeItem(list, liveUser("ok", { id: "s:user:2" }));
  expect(list.filter((x) => x.type === "user")).toHaveLength(2);
  expect(list.filter((x) => x.source === "ui")).toHaveLength(0);
});

test("reused tool call ids stay distinct across rounds", () => {
  const a = itemFromEvent({
    type: "tool_call",
    session_id: "s",
    payload: { id: "w1", name: "read_file", round: "s:r1" },
  });
  const b = itemFromEvent({
    type: "tool_call",
    session_id: "s",
    payload: { id: "w1", name: "read_file", round: "s:r3" },
  });
  expect(a.key).not.toEqual(b.key);
  const list = mergeItem(mergeItem([], a), b);
  expect(list).toHaveLength(2);
});

test("pending ui user survives seed of the previous turn", () => {
  const seed = replayEvents([
    { type: "user", session_id: "s", ts: "2026-01-01T00:00:00Z", payload: { text: "ok", id: "s:user:1" } },
    { type: "assistant", session_id: "s", ts: "2026-01-01T00:00:01Z", payload: { text: "one", id: "s:r1" } },
  ]);
  const live = [uiUser("ok")];
  const out = mergePendingUsers(seed, live);
  expect(out.filter((x) => x.type === "user")).toHaveLength(2);
});

test("WailsEvent envelope unwraps to the trace payload", () => {
  const inner = { type: "assistant", session_id: "s", payload: { text: "round two", id: "s:r2" } };
  const it = itemFromEvent({ name: "yoyo:item", data: inner });
  expect(it.type).toBe("assistant");
  expect(it.text).toBe("round two");
  expect(it.payload.id).toBe("s:r2");
  expect(unwrapEvent({ name: "yoyo:item", data: [inner] }).type).toBe("assistant");
});

test("unwrap does not walk into a typed event's nested data field", () => {
  const it = itemFromEvent({
    type: "assistant",
    session_id: "s",
    payload: { text: "keep me", data: { text: "eaten" }, id: "s:r2" },
  });
  expect(it.text).toBe("keep me");
});

test("PascalCase Wails bindings still fold as assistant text", () => {
  const it = itemFromEvent({
    Type: "assistant",
    SessionID: "s",
    Payload: { Text: "hello", Id: "s:r1", Delta: true },
  });
  expect(it.type).toBe("assistant");
  expect(it.text).toBe("hello");
  expect(it.delta).toBe(true);
  expect(it.payload.id).toBe("s:r1");
});

test("late seed cannot drop a live second assistant round", () => {
  const seed = replayEvents([
    { type: "user", session_id: "s", payload: { text: "first question" } },
    { type: "assistant", session_id: "s", payload: { text: "first answer", id: "s:r1" } },
  ]);
  const live = replayEvents([
    { type: "user", session_id: "s", payload: { text: "first question" } },
    { type: "assistant", session_id: "s", payload: { text: "first answer", id: "s:r1" } },
    { type: "user", session_id: "s", payload: { text: "second question" } },
    { type: "assistant", session_id: "s", payload: { text: "second answer", id: "s:r2" } },
  ]);
  const out = foldLiveIntoSeed(seed, live);
  expect(out.filter((x) => x.type === "assistant").map((x) => x.text)).toEqual(["first answer", "second answer"]);
});

test("two-turn layout keeps separate agent rows", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "first question" } },
    { type: "assistant", session_id: "s", payload: { text: "first answer", id: "s:r1" } },
    { type: "user", session_id: "s", payload: { text: "second question" } },
    { type: "assistant", session_id: "s", payload: { text: "## Report\n\n- item one\n- item two", id: "s:r2" } },
  ]);
  const rows = layoutRows(items);
  expect(rows.map((r) => r.kind)).toEqual(["user", "agent", "user", "agent"]);
  const second = rows[3];
  expect(second.kind).toBe("agent");
  if (second.kind === "agent") {
    const text = second.parts.find((p) => p.kind === "item" && p.item.type === "assistant");
    expect(text?.kind === "item" ? text.item.text : "").toContain("item two");
  }
});

test("tool detail prefers path over raw json", () => {
  const item = itemFromEvent({
    type: "tool_call",
    session_id: "s",
    payload: { id: "c1", name: "read_file", arguments: "{\"path\":\"src/main.go\"}" },
  });
  expect(toolName(item)).toBe("read_file");
  expect(toolDetail(item)).toBe("src/main.go");
  expect(isArtifactTool("office_create")).toBe(true);
  expect(isArtifactTool("read_file")).toBe(false);
  expect(isArtifactTool("write_file")).toBe(true);
  expect(isArtifactTool("str_replace")).toBe(true);
});

test("ERROR: content marks a tool result as failed", () => {
  const ok = itemFromEvent({
    type: "tool_result",
    session_id: "s",
    payload: { id: "c1", name: "read_file", content: "package main" },
  });
  const bad = itemFromEvent({
    type: "tool_result",
    session_id: "s",
    payload: { id: "c2", name: "shell", content: "ERROR: exit 1" },
  });
  expect(isToolFailed(ok)).toBe(false);
  expect(isToolFailed(bad)).toBe(true);
});

test("plan and file_change stay inside the agent turn", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "plan it" } },
    { type: "assistant", session_id: "s", payload: { text: "drafting", id: "s:r1" } },
    { type: "tool_call", session_id: "s", payload: { id: "p1", name: "update_plan", arguments: "{}" } },
    { type: "tool_result", session_id: "s", payload: { id: "p1", name: "update_plan", content: "1. look\n2. patch" } },
    { type: "plan", session_id: "s", payload: { name: "update_plan", text: "1. look\n2. patch", id: "p1" } },
    { type: "tool_call", session_id: "s", payload: { id: "a1", name: "apply_patch", arguments: "{}" } },
    { type: "tool_result", session_id: "s", payload: { id: "a1", name: "apply_patch", content: "*** Add File: README.md" } },
    { type: "file_change", session_id: "s", payload: { id: "a1", name: "apply_patch", paths: ["README.md"] } },
    { type: "assistant", session_id: "s", payload: { text: "done", id: "s:r2" } },
  ]);
  const rows = layoutRows(items);
  expect(rows.map((r) => r.kind)).toEqual(["user", "agent"]);
  const agent = rows[1];
  expect(agent.kind).toBe("agent");
  if (agent.kind !== "agent") return;
  expect(agent.parts.map((p) => p.kind)).toEqual(["item", "process", "artifact", "item"]);
  const process = agent.parts[1];
  expect(process.kind).toBe("process");
  if (process.kind !== "process") return;
  expect(process.items.filter((it) => it.type === "tool_call").map((it) => it.name)).toEqual(["update_plan"]);
  const artifacts = agent.parts[2];
  expect(artifacts.kind).toBe("artifact");
  if (artifacts.kind !== "artifact") return;
  expect(artifacts.items.filter((it) => it.type === "tool_call").map((it) => it.name)).toEqual(["apply_patch"]);
  const plan = latestTaskPlan(items);
  expect(plan?.steps.map((s) => s.step)).toEqual(["look", "patch"]);
});

test("plan text parses statuses and explanation", () => {
  const plan = parsePlanText("Ship the chip\n\n1. [complete] Read the shell\n2. [in_progress] Move the checklist\n3. [pending] Cover with a test\n");
  expect(plan?.explanation).toBe("Ship the chip");
  expect(plan?.steps).toEqual([
    { step: "Read the shell", status: "complete" },
    { step: "Move the checklist", status: "in_progress" },
    { step: "Cover with a test", status: "pending" },
  ]);
});

test("composer chip drops the previous turn's plan", () => {
  const prior = [
    { type: "user", session_id: "s", payload: { text: "first" } },
    { type: "tool_call", session_id: "s", payload: { id: "p1", name: "update_plan", arguments: "{\"plan\":[{\"step\":\"old\",\"status\":\"in_progress\"}]}" } },
    { type: "tool_result", session_id: "s", payload: { id: "p1", name: "update_plan", content: "1. [in_progress] old" } },
    { type: "plan", session_id: "s", payload: { name: "update_plan", text: "1. [in_progress] old", id: "p1" } },
  ];
  expect(latestTaskPlan(replayEvents(prior))?.steps).toEqual([{ step: "old", status: "in_progress" }]);
  expect(latestTaskPlan(replayEvents([...prior, { type: "user", session_id: "s", payload: { text: "second" } }]))).toBeNull();
  const next = replayEvents([
    ...prior,
    { type: "user", session_id: "s", payload: { text: "second" } },
    { type: "tool_call", session_id: "s", payload: { id: "p2", name: "update_plan", arguments: "{\"plan\":[{\"step\":\"new\",\"status\":\"pending\"}]}" } },
    { type: "plan", session_id: "s", payload: { name: "update_plan", text: "1. [pending] new", id: "p2" } },
  ]);
  expect(latestTaskPlan(next)?.steps).toEqual([{ step: "new", status: "pending" }]);
});

test("steer does not clear the current turn plan", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "do it" } },
    { type: "tool_call", session_id: "s", payload: { id: "p1", name: "update_plan", arguments: "{\"plan\":[{\"step\":\"keep\",\"status\":\"in_progress\"}]}" } },
    { type: "plan", session_id: "s", payload: { name: "update_plan", text: "1. [in_progress] keep", id: "p1" } },
    { type: "user", session_id: "s", source: "steer", payload: { text: "User steering (apply now): hurry" } },
  ]);
  expect(latestTaskPlan(items)?.steps).toEqual([{ step: "keep", status: "in_progress" }]);
});

test("failed update_plan is skipped for the composer chip", () => {
  const items = replayEvents([
    { type: "tool_call", session_id: "s", payload: { id: "p1", name: "update_plan", arguments: "{\"plan\":[{\"step\":\"old\",\"status\":\"pending\"}]}" } },
    { type: "tool_result", session_id: "s", payload: { id: "p1", name: "update_plan", content: "1. [pending] old" } },
    { type: "plan", session_id: "s", payload: { name: "update_plan", text: "1. [pending] old", id: "p1" } },
    { type: "tool_call", session_id: "s", payload: { id: "p2", name: "update_plan", arguments: "{\"plan\":[{\"step\":\"new\",\"status\":\"in_progress\"}]}" } },
    { type: "tool_result", session_id: "s", payload: { id: "p2", name: "update_plan", content: "ERROR: empty plan" } },
  ]);
  const plan = latestTaskPlan(items);
  expect(plan?.steps).toEqual([{ step: "old", status: "pending" }]);
});

test("ask_user does not split the agent turn", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "ask me" } },
    { type: "assistant", session_id: "s", payload: { text: "one question", id: "s:r1" } },
    { type: "ask_user", session_id: "s", payload: { question: "which file?" } },
    { type: "assistant", session_id: "s", payload: { text: "thanks", id: "s:r2" } },
  ]);
  const rows = layoutRows(items);
  expect(rows.map((r) => r.kind)).toEqual(["user", "agent"]);
  const agent = rows[1];
  expect(agent.kind).toBe("agent");
  if (agent.kind !== "agent") return;
  expect(agent.parts.filter((p) => p.kind === "item")).toHaveLength(2);
});

test("process tools batch separately from artifacts", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "look around" } },
    { type: "tool_call", session_id: "s", payload: { id: "c1", name: "read_file", arguments: "{\"path\":\"src/main.go\"}" } },
    { type: "tool_result", session_id: "s", payload: { id: "c1", name: "read_file", content: "ok", elapsed_ms: 12 } },
    { type: "tool_call", session_id: "s", payload: { id: "c2", name: "grep", arguments: "{\"query\":\"main\"}" } },
    { type: "tool_result", session_id: "s", payload: { id: "c2", name: "grep", content: "src/main.go" } },
    { type: "tool_call", session_id: "s", payload: { id: "c3", name: "apply_patch", arguments: "{}" } },
    { type: "tool_result", session_id: "s", payload: { id: "c3", name: "apply_patch", content: "*** Update File: src/main.go" } },
    { type: "assistant", session_id: "s", payload: { text: "patched", id: "s:r1" } },
  ]);
  const rows = layoutRows(items);
  expect(rows.map((r) => r.kind)).toEqual(["user", "agent"]);
  const agent = rows[1];
  expect(agent.kind).toBe("agent");
  if (agent.kind !== "agent") return;
  expect(agent.parts.map((p) => p.kind)).toEqual(["process", "artifact", "item"]);
  const process = agent.parts[0];
  expect(process.kind).toBe("process");
  if (process.kind !== "process") return;
  expect(process.live).toBe(false);
  expect(process.items.filter((it) => it.type === "tool_call").map((it) => it.name)).toEqual(["read_file", "grep"]);
});

test("unmatched tool call marks the process group live", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "read it" } },
    { type: "tool_call", session_id: "s", payload: { id: "c1", name: "read_file", arguments: "{}" } },
  ]);
  const rows = layoutRows(items);
  const agent = rows[1];
  expect(agent.kind).toBe("agent");
  if (agent.kind !== "agent") return;
  expect(agent.parts).toHaveLength(1);
  expect(agent.parts[0].kind).toBe("process");
  if (agent.parts[0].kind !== "process") return;
  expect(agent.parts[0].live).toBe(true);
});

test("steer stays inside the running agent turn", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "go" } },
    { type: "assistant", session_id: "s", payload: { text: "working", id: "s:r1" } },
    { type: "user", session_id: "s", source: "steer", payload: { text: "User steering (apply now): stop the shell" } },
    { type: "tool_call", session_id: "s", payload: { id: "c1", name: "read_file", arguments: "{}" } },
    { type: "tool_result", session_id: "s", payload: { id: "c1", name: "read_file", content: "ok" } },
    { type: "assistant", session_id: "s", payload: { text: "stopped", id: "s:r2" } },
  ]);
  const rows = layoutRows(items);
  expect(rows.map((r) => r.kind)).toEqual(["user", "agent"]);
});

test("write_file and str_replace land as artifacts with a rendered diff", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "edit it" } },
    { type: "tool_call", session_id: "s", payload: { id: "w1", name: "write_file", arguments: "{\"path\":\"web/index.html\",\"content\":\"<!doctype html><html><body>hi</body></html>\"}" } },
    { type: "tool_result", session_id: "s", payload: { id: "w1", name: "write_file", content: "wrote web/index.html" } },
    { type: "tool_call", session_id: "s", payload: { id: "e1", name: "str_replace", arguments: "{\"path\":\"src/main.go\",\"old_str\":\"old\",\"new_str\":\"new\"}" } },
    { type: "tool_result", session_id: "s", payload: { id: "e1", name: "str_replace", content: "replaced 1 occurrence(s) in src/main.go" } },
  ]);
  const rows = layoutRows(items);
  const agent = rows[1];
  expect(agent.kind).toBe("agent");
  if (agent.kind !== "agent") return;
  expect(agent.parts.map((p) => p.kind)).toEqual(["artifact"]);
  const view = artifactView(items.find((it) => it.type === "tool_call" && it.name === "write_file"), items.find((it) => it.type === "tool_result" && it.name === "write_file"));
  expect(view.kind).toBe("html");
  expect(view.path).toBe("web/index.html");
  const diff = artifactView(items.find((it) => it.type === "tool_call" && it.name === "str_replace"));
  expect(diff.kind).toBe("diff");
  expect(diff.diff).toContain("-old");
  expect(diff.diff).toContain("+new");
  expect(artifactPreviewOpen(view)).toBe(true);
  expect(artifactPreviewOpen(diff)).toBe(true);
  const code = artifactView({
    type: "tool_call",
    name: "write_file",
    payload: { name: "write_file", arguments: JSON.stringify({ path: "webui/src/pages/Users.tsx", content: "export function Users() { return null }" }) },
  } as Item);
  expect(code.kind).toBe("code");
  expect(artifactPreviewOpen(code)).toBe(false);
  expect(artifactShouldShow(code)).toBe(false);
});

test("unchanged source dumps stay in the process rail, not preview cards", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", payload: { text: "dump the page" } },
    { type: "tool_call", session_id: "s", payload: { id: "w1", name: "write_file", arguments: JSON.stringify({ path: "webui/src/pages/Users.tsx", content: "export function Users() { return null }" }) } },
    { type: "tool_result", session_id: "s", payload: { id: "w1", name: "write_file", content: "wrote webui/src/pages/Users.tsx" } },
    { type: "tool_call", session_id: "s", payload: { id: "e1", name: "str_replace", arguments: JSON.stringify({ path: "src/main.go", old_str: "old", new_str: "old" }) } },
    { type: "tool_result", session_id: "s", payload: { id: "e1", name: "str_replace", content: "unchanged src/main.go" } },
    { type: "tool_call", session_id: "s", payload: { id: "e2", name: "str_replace", arguments: JSON.stringify({ path: "src/main.go", old_str: "old", new_str: "new" }) } },
    { type: "tool_result", session_id: "s", payload: { id: "e2", name: "str_replace", content: "replaced 1 occurrence(s) in src/main.go" } },
  ]);
  const rows = layoutRows(items);
  const agent = rows[1];
  expect(agent.kind).toBe("agent");
  if (agent.kind !== "agent") return;
  expect(agent.parts.map((p) => p.kind)).toEqual(["process", "artifact"]);
  const dump = artifactView(items.find((it) => it.type === "tool_call" && it.payload?.id === "w1"));
  const noop = artifactView(items.find((it) => it.type === "tool_call" && it.payload?.id === "e1"));
  const edit = artifactView(items.find((it) => it.type === "tool_call" && it.payload?.id === "e2"));
  expect(artifactShouldShow(dump)).toBe(false);
  expect(artifactShouldShow(noop)).toBe(false);
  expect(artifactShouldShow(edit)).toBe(true);
});
