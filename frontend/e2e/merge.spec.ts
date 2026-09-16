import { expect, test } from "@playwright/test";
import { itemFromEvent, mergeItem, mergePendingUsers, replayEvents } from "../src/lib/stream-fold";
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
