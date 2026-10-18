import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Item, ItemType } from "./protocol.ts";
import { deriveTurnStatus, statusBarVisible } from "./turn-status.ts";

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

const user = item({ key: "u1", type: "user", text: "go", source: "user" });

describe("deriveTurnStatus", () => {
  it("is silent when the thread is idle", () => {
    assert.equal(deriveTurnStatus([user, item({ key: "a1", type: "assistant", text: "done" })], false), null);
  });

  it("thinks before the first token", () => {
    const st = deriveTurnStatus([user], true);
    assert.ok(st);
    assert.equal(st.phase, "thinking");
    assert.equal(st.verb, "think");
    assert.equal(statusBarVisible(st), true);
  });

  it("names the in-flight tool", () => {
    const st = deriveTurnStatus([
      user,
      item({ key: "a0", type: "assistant", text: "reading first" }),
      item({
        key: "c1:call", type: "tool_call", name: "read_file",
        payload: { id: "c1", name: "read_file", arguments: '{"path":"a.ts"}' },
      }),
    ], true);
    assert.ok(st);
    assert.equal(st.phase, "tool");
    assert.equal(st.verb, "read");
    assert.equal(st.detail, "a.ts");
    assert.equal(statusBarVisible(st), true);
  });

  it("keeps the composer bar while the answer caret is moving", () => {
    const st = deriveTurnStatus([
      user,
      item({ key: "a1", type: "assistant", text: "hello", delta: true }),
    ], true);
    assert.ok(st);
    assert.equal(st.phase, "streaming");
    assert.equal(st.live, true);
    assert.equal(statusBarVisible(st), true);
  });

  it("keeps the bar when a settled-looking stream might still grow tools", () => {
    const st = deriveTurnStatus([
      user,
      item({ key: "a1", type: "assistant", text: "looking around" }),
    ], true);
    assert.ok(st);
    assert.equal(st.phase, "streaming");
    assert.equal(st.live, false);
    assert.equal(statusBarVisible(st), true);
  });
});
