/** 拥有陪伴通话事件解析的断言。不拥有实现。不改服务端。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { companionCallOn } from "./companion-call.ts";

describe("companion call event", () => {
  /** 从 Wails 信封里读出通话开着。 */
  it("reads the Wails envelope instead of treating the event object as off", () => {
    assert.equal(companionCallOn({ name: "yoyo:call", data: true }), true);
    assert.equal(companionCallOn({ name: "yoyo:call", data: false }), false);
    assert.equal(companionCallOn({ name: "yoyo:call", data: [true] }), true);
    assert.equal(companionCallOn("true"), true);
    assert.equal(companionCallOn(true), true);
    assert.equal(companionCallOn({ on: true }), true);
  });
});
