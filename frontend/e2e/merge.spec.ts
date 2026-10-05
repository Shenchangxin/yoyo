import { expect, test } from "@playwright/test";
import { dropTurnErrors, eventFingerprint, foldLiveIntoSeed, foldTurnErrors, HOT_TRANSCRIPT_TURNS, itemFromEvent, lastUserTurns, mergeItem, mergePendingUsers, parseLiveNotice, replayEvents, UI_TEXT_CAP, unwrapEvent, userTurnCount, withLiveTail } from "../src/lib/stream-fold";
import { layoutRows, processGroupLive, tailProcessPairs } from "../src/lib/transcript-layout";
import { latestTaskPlan, parsePlanText } from "../src/lib/plan";
import { toolDetail, toolName, isArtifactTool, isToolFailed } from "../src/lib/tool-summary";
import { artifactPreviewOpen, artifactShouldShow, artifactView } from "../src/lib/artifact-preview";
import { structureSig } from "../src/lib/stream-live";
import type { Item } from "../src/lib/protocol";
import { itemMatchesTurn, isOutlineActive, itemTurnKey, neighborTurn, outlineTitle, outlineTurnKey, pendingOutline, pickActiveTurnKey, rowMatchesJump, turnJumpAliases } from "../src/lib/turn-outline";

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

test("mention inject does not render as a prior user turn", () => {
  const items = replayEvents([
    { type: "context_injection", source: "mention", session_id: "s", payload: { text: "## @skill:arxiv-watcher\nUse scripts" } },
    { type: "user", session_id: "s", payload: { text: "@skill:arxiv-watcher 帮我查询一下关于agent自进化的论文" } },
    { type: "assistant", session_id: "s", payload: { text: "好", id: "s:r1" } },
  ]);
  const rows = layoutRows(items);
  expect(rows.map((r) => r.kind)).toEqual(["user", "agent"]);
  expect(rows[0].kind === "user" ? rows[0].item.text : "").toContain("帮我查询");
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
  expect(artifactPreviewOpen(view)).toBe(false);
  expect(artifactPreviewOpen(diff)).toBe(false);
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

test("assistant token deltas concatenate and stay live until the snapshot", () => {
  let list: Item[] = [];
  list = mergeItem(list, itemFromEvent({
    type: "assistant", session_id: "s", payload: { text: "he", id: "s:r1", delta: true },
  }));
  list = mergeItem(list, itemFromEvent({
    type: "assistant", session_id: "s", payload: { text: "llo", id: "s:r1", delta: true },
  }));
  expect(list).toHaveLength(1);
  expect(list[0].text).toBe("hello");
  expect(list[0].delta).toBe(true);
  list = mergeItem(list, itemFromEvent({
    type: "assistant", session_id: "s", payload: { text: "hello", id: "s:r1" },
  }));
  expect(list[0].text).toBe("hello");
  expect(list[0].delta).toBe(false);
});

test("reasoning deltas share a round key", () => {
  let list: Item[] = [];
  list = mergeItem(list, itemFromEvent({
    type: "reasoning", session_id: "s", payload: { text: "why ", id: "s:r1", round: "s:r1", delta: true },
  }));
  list = mergeItem(list, itemFromEvent({
    type: "reasoning", session_id: "s", payload: { text: "not", id: "s:r1", round: "s:r1", delta: true },
  }));
  expect(list).toHaveLength(1);
  expect(list[0].text).toBe("why not");
  expect(list[0].delta).toBe(true);
});

test("shell stdout deltas concatenate without closing the process group", () => {
  let list: Item[] = [];
  list = mergeItem(list, itemFromEvent({
    type: "tool_call", session_id: "s", payload: { id: "c1", name: "shell", round: "s:r1" },
  }));
  list = mergeItem(list, itemFromEvent({
    type: "tool_result", session_id: "s", payload: { id: "c1", name: "shell", content: "hel", round: "s:r1", delta: true },
  }));
  list = mergeItem(list, itemFromEvent({
    type: "tool_result", session_id: "s", payload: { id: "c1", name: "shell", content: "lo", round: "s:r1", delta: true },
  }));
  expect(list.filter((x) => x.type === "tool_result")).toHaveLength(1);
  expect(String(list.find((x) => x.type === "tool_result")?.payload?.content)).toBe("hello");
  expect(processGroupLive(list)).toBe(true);
  list = mergeItem(list, itemFromEvent({
    type: "tool_result", session_id: "s", payload: { id: "c1", name: "shell", content: "hello", round: "s:r1" },
  }));
  expect(processGroupLive(list)).toBe(false);
});

test("write_file live items do not keep the full file body", () => {
  const body = "x".repeat(4000);
  const args = JSON.stringify({ path: "a.ts", content: body });
  const it = itemFromEvent({
    type: "tool_call",
    session_id: "s",
    payload: { id: "w1", name: "write_file", arguments: args, round: "s:r1" },
  });
  expect(String(it.payload?.arguments)).not.toContain(body);
  expect(String(it.payload?.arguments)).toContain("a.ts");
  expect(it.payload?.bytes).toBe(args.length);
});

test("transcript keeps prior turns and all slim tool pairs", () => {
  const raw: any[] = [
    { type: "user", session_id: "s", payload: { text: "old" } },
    { type: "assistant", session_id: "s", payload: { text: "ok", id: "s:r0" } },
  ];
  for (let i = 0; i < 40; i++) {
    raw.push({ type: "tool_call", session_id: "s", payload: { id: `c${i}`, name: "read_file", arguments: "{}" } });
    raw.push({ type: "file_change", session_id: "s", payload: { id: `c${i}`, path: "x.ts" } });
    raw.push({ type: "tool_result", session_id: "s", payload: { id: `c${i}`, name: "read_file", content: "ok" } });
  }
  raw.push({ type: "user", session_id: "s", payload: { text: "new" } });
  const items = replayEvents(raw);
  expect(items.filter((x) => x.type === "file_change")).toHaveLength(0);
  expect(items.filter((x) => x.type === "user").map((x) => x.text)).toEqual(["old", "new"]);
  expect(items.filter((x) => x.type === "tool_call")).toHaveLength(40);
});

test("process rail expands from the tail", () => {
  const pairs = Array.from({ length: 80 }, (_, i) => ({ key: `c${i}`, extra: [] as Item[] }));
  const page = tailProcessPairs(pairs, 60);
  expect(page.hidden).toBe(20);
  expect(page.visible.map((p) => p.key)).toEqual(pairs.slice(20).map((p) => p.key));
});

test("live notices peel nested events and leave pull-only seqs", () => {
  const inline = parseLiveNotice({ seq: 3, type: "user", session_id: "s", payload: { text: "hi" } });
  expect(inline.seq).toBe(3);
  expect(inline.event.type).toBe("user");
  const nested = parseLiveNotice({
    seq: 4,
    session_id: "s",
    type: "assistant",
    event: { type: "assistant", session_id: "s", payload: { text: "ok" } },
  });
  expect(nested.seq).toBe(4);
  expect(nested.event.type).toBe("assistant");
  const pull = parseLiveNotice({ seq: 8, session_id: "s", type: "assistant" });
  expect(pull.seq).toBe(8);
  expect(pull.event).toBeUndefined();
});

test("structureSig ignores token text", () => {
  const a: Item[] = [{ key: "1", type: "assistant", sessionId: "s", source: "", ts: "", text: "he", name: "", delta: true, payload: {} }];
  const b: Item[] = [{ key: "1", type: "assistant", sessionId: "s", source: "", ts: "", text: "hello", name: "", delta: true, payload: {} }];
  expect(structureSig(a)).toBe(structureSig(b));
});

test("hot window keeps only the last N user turns", () => {
  const raw: any[] = [];
  for (let i = 0; i < 20; i++) {
    raw.push({ type: "user", session_id: "s", payload: { text: `u${i}`, id: `u${i}` } });
    raw.push({ type: "assistant", session_id: "s", payload: { text: `a${i}`, id: `r${i}` } });
  }
  const items = lastUserTurns(replayEvents(raw), HOT_TRANSCRIPT_TURNS);
  expect(userTurnCount(items)).toBe(HOT_TRANSCRIPT_TURNS);
  expect(items.find((x) => x.text === "u0")).toBeUndefined();
  expect(items.filter((x) => x.type === "user").map((x) => x.text).pop()).toBe("u19");
});

test("settled assistant text is capped for the renderer", () => {
  const body = "x".repeat(UI_TEXT_CAP + 4000);
  const it = itemFromEvent({
    type: "assistant",
    session_id: "s",
    payload: { text: body, id: "r1" },
  });
  expect(it.text.length).toBeLessThan(body.length);
  expect(it.text.endsWith("…")).toBe(true);
  const delta = itemFromEvent({
    type: "assistant",
    session_id: "s",
    payload: { text: body, id: "r1", delta: true },
  });
  expect(delta.text).toBe(body);
});

test("delta fingerprints stay unique across seqs without storing the slice", () => {
  const a = eventFingerprint({
    seq: 11,
    type: "assistant",
    session_id: "s",
    payload: { text: "hello world this is a slice", id: "r1", delta: true },
  });
  const b = eventFingerprint({
    seq: 12,
    type: "assistant",
    session_id: "s",
    payload: { text: " and more tokens here", id: "r1", delta: true },
  });
  expect(a).not.toEqual(b);
  expect(a.includes("hello world")).toBe(false);
  expect(b.includes("and more tokens")).toBe(false);
});

test("interrupted card stays only while it is the turn's terminal chip", () => {
  const stopped = itemFromEvent({
    type: "error",
    session_id: "s",
    payload: { kind: "canceled", title: "Stopped", hint: "This turn was interrupted.", id: "e1" },
  });
  let list = replayEvents([
    { type: "user", session_id: "s", payload: { text: "go", id: "u1" } },
    { type: "assistant", session_id: "s", payload: { text: "working", id: "r1" } },
  ]);
  list = mergeItem(list, stopped);
  expect(list.filter((x) => x.type === "error")).toHaveLength(1);
  list = mergeItem(list, itemFromEvent({
    type: "tool_call",
    session_id: "s",
    payload: { id: "c2", name: "read_file", round: "s:r1" },
  }));
  expect(list.filter((x) => x.type === "error")).toHaveLength(0);
  expect(list.some((x) => x.type === "tool_call")).toBe(true);
});

test("continue-this-turn drops the interrupted card before new tokens", () => {
  const list = [
    itemFromEvent({ type: "user", session_id: "s", payload: { text: "go", id: "u1" } }),
    itemFromEvent({ type: "assistant", session_id: "s", payload: { text: "working", id: "r1" } }),
    itemFromEvent({ type: "error", session_id: "s", payload: { kind: "canceled", title: "Stopped", id: "e1" } }),
  ];
  expect(dropTurnErrors(list).filter((x) => x.type === "error")).toHaveLength(0);
  const folded = foldTurnErrors(list);
  expect(folded.filter((x) => x.type === "error")).toHaveLength(1);
});

test("outline title peels mentions and caps runes", () => {
  expect(outlineTitle("@file:src/loop.go  Fix the parser\nnow")).toBe("Fix the parser now");
  expect([...outlineTitle("字".repeat(50))].length).toBe(43);
});

test("itemFromEvent keeps store seq for outline jump", () => {
  const it = itemFromEvent({
    type: "user",
    session_id: "s",
    seq: 12,
    payload: { text: "hello", id: "s:user:1" },
  });
  expect(it.seq).toBe(12);
  expect(itemMatchesTurn(it, { seq: 12, title: "hello", id: "s:user:1", key: "s:user:1" })).toBeTruthy();
});

test("pending outline appends optimistic ui turns", () => {
  const store = [{ seq: 1, title: "first", id: "u1", key: "u1" }];
  const extra = pendingOutline(store, [uiUser("second question")]);
  expect(extra).toHaveLength(2);
  expect(extra[1].title).toBe("second question");
  expect(pendingOutline(store, [uiUser("first")])).toEqual(store);
});

test("neighbor turn walks the directory", () => {
  const turns = [
    { seq: 1, title: "a", id: "a", key: "a" },
    { seq: 3, title: "b", id: "b", key: "b" },
    { seq: 5, title: "c", id: "c", key: "c" },
  ];
  const mid = outlineTurnKey(turns[1]);
  expect(outlineTurnKey(neighborTurn(turns, mid, -1)!)).toBe(outlineTurnKey(turns[0]));
  expect(outlineTurnKey(neighborTurn(turns, mid, 1)!)).toBe(outlineTurnKey(turns[2]));
  expect(neighborTurn(turns, outlineTurnKey(turns[0]), -1)).toBeNull();
});

test("outline active is a single non-empty key", () => {
  const a = { seq: 1, title: "a", id: "", key: "" };
  const b = { seq: 2, title: "b", id: "", key: "" };
  const c = { seq: 0, title: "c", id: "", key: "" };
  expect(isOutlineActive(a, "")).toBeFalsy();
  expect(isOutlineActive(c, "")).toBeFalsy();
  expect(isOutlineActive(a, outlineTurnKey(a))).toBeTruthy();
  expect(isOutlineActive(b, outlineTurnKey(a))).toBeFalsy();
  expect(pickActiveTurnKey([
    { key: "seq:1", top: 10 },
    { key: "seq:2", top: 40 },
    { key: "seq:3", top: 90 },
    { key: "", top: 41 },
  ], 50)).toBe("seq:2");
});

test("itemTurnKey prefers durable id over hub seq", () => {
  const it = itemFromEvent({
    type: "user",
    session_id: "s",
    seq: 99,
    payload: { text: "hello", id: "s:user:1" },
  });
  expect(itemTurnKey(it)).toBe("s:user:1");
  expect(itemMatchesTurn(it, { seq: 12, title: "hello", id: "s:user:1", key: "s:user:1" })).toBeTruthy();
  expect(itemMatchesTurn(it, { seq: 99, title: "other", id: "s:user:2", key: "s:user:2" })).toBeFalsy();
  expect(rowMatchesJump(itemTurnKey(it), turnJumpAliases({ seq: 12, id: "s:user:1", key: "s:user:1" }))).toBeTruthy();
});

test("neighbor turn miss does not wrap to the other end", () => {
  const turns = [
    { seq: 1, title: "a", id: "a", key: "a" },
    { seq: 3, title: "b", id: "b", key: "b" },
  ];
  expect(neighborTurn(turns, "missing", 1)).toBeNull();
  expect(neighborTurn(turns, "missing", -1)).toBeNull();
});

test("agent layout rows share the operator turnKey", () => {
  const items = replayEvents([
    { type: "user", session_id: "s", seq: 4, payload: { text: "go", id: "s:user:1" } },
    { type: "assistant", session_id: "s", payload: { text: "ok", id: "s:r1" } },
  ]);
  const rows = layoutRows(items);
  expect(rows).toHaveLength(2);
  expect(rows[0].kind).toBe("user");
  expect(rows[1].kind).toBe("agent");
  if (rows[0].kind === "user" && rows[1].kind === "agent") {
    expect(rows[0].turnKey).toBe("s:user:1");
    expect(rows[1].turnKey).toBe("s:user:1");
  }
});

test("withLiveTail pins the live pack onto a browse window", () => {
  const browse = replayEvents([
    { type: "user", session_id: "s", payload: { text: "old", id: "u0" } },
    { type: "assistant", session_id: "s", payload: { text: "a0", id: "r0" } },
  ]);
  const live = replayEvents([
    { type: "user", session_id: "s", payload: { text: "new", id: "u9" } },
    { type: "assistant", session_id: "s", payload: { text: "a9", id: "r9" } },
  ]);
  const out = withLiveTail(browse, live);
  expect(out.filter((x) => x.type === "user").map((x) => x.text)).toEqual(["old", "new"]);
});
