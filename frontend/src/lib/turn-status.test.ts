import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Item, ItemType } from "./protocol.ts";
import { deriveTurnStatus, statusBarVisible } from "./turn-status.ts";

/**
 * 拥有：作曲栏状态在空闲、思考、工具和正文流出时的约定。
 * 不拥有：状态栏的渲染。
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

const user = item({ key: "u1", type: "user", text: "go", source: "user" });

describe("deriveTurnStatus", () => {
  /** 线程空闲时不给出状态。 */
  it("is silent when the thread is idle", () => {
    assert.equal(deriveTurnStatus([user, item({ key: "a1", type: "assistant", text: "done" })], false), null);
  });

  /** 第一个 token 出现前是思考，栏要显示。 */
  it("thinks before the first token", () => {
    const st = deriveTurnStatus([user], true);
    assert.ok(st);
    assert.equal(st.phase, "thinking");
    assert.equal(st.verb, "think");
    assert.equal(statusBarVisible(st), true);
  });

  /** 进行中的工具报出动词和参数里的路径。 */
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

  /** 回答光标还在动时，作曲栏保持为流出。 */
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

  /** 正文看起来已经停住、线程仍在跑时，栏继续显示，因为后面还可能长出工具。 */
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
