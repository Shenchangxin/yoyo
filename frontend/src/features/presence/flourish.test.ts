/** 拥有进场动作选择的断言。不拥有情绪球。不改服务端。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { entryMotion } from "./flourish.ts";

describe("entryMotion", () => {
  /** 做完一轮会庆祝，干活的脸保持安静。 */
  it("celebrates a finished turn and stays quiet on work faces", () => {
    assert.equal(entryMotion("33"), "celebrate");
    assert.equal(entryMotion("47"), "flourish");
    assert.equal(entryMotion("31"), "bounce");
    assert.equal(entryMotion("48"), "nudge");
    assert.equal(entryMotion("30"), null);
    assert.equal(entryMotion("36"), null);
    assert.equal(entryMotion("02"), null);
  });
});
