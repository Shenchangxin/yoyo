import { expect, test } from "@playwright/test";
import { foldLiveIntoSeed, itemFromEvent, mergeItem, mergePendingUsers, replayEvents, unwrapEvent } from "../src/lib/stream-fold";
import { layoutRows } from "../src/lib/transcript-layout";
import { toolDetail, toolName, isArtifactTool } from "../src/lib/tool-summary";
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
});
