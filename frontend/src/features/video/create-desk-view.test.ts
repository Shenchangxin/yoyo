/** 锁住创作桌视图的优先级：不在创作桌是 off，剪辑先于成片先于翻拍，都没有是起步。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { createDeskView } from "./create-desk-view.ts";

const none = { createDesk: true, cutBound: false, filmOpen: false, remakeOpen: false };

describe("create desk view", () => {
  /** 每种组合各一条。 */
  it("keeps cut over film over remake", () => {
    assert.equal(createDeskView({ ...none, createDesk: false, cutBound: true }), "off");
    assert.equal(createDeskView(none), "launch");
    assert.equal(createDeskView({ ...none, cutBound: true, filmOpen: true, remakeOpen: true }), "cut");
    assert.equal(createDeskView({ ...none, filmOpen: true, remakeOpen: true }), "film");
    assert.equal(createDeskView({ ...none, remakeOpen: true }), "remake");
  });
});