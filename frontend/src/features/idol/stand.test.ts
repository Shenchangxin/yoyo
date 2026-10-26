/**
 * 拥有：通话情绪、点头和宠物表情编号的断言。
 * 不拥有：Spine 片段和画布。不改服务端。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { nodFor, petFace, standMood } from "./stand.ts";

describe("call stand", () => {
  /** 断言七种情绪保留原样，无法识别的收到平静。 */
  it("keeps the seven emotions and falls back to calm", () => {
    assert.equal(standMood("HAPPY"), "happy");
    assert.equal(standMood("surprised"), "surprised");
    assert.equal(standMood("nope"), "calm");
  });

  /** 断言高兴点头、生气摇头、思考不动，并且说话、思考、听和拨号选到对应宠物表情。 */
  it("nods with the mood and picks a pet face for the phase", () => {
    assert.equal(nodFor("happy"), "yes");
    assert.equal(nodFor("angry"), "no");
    assert.equal(nodFor("thinking"), null);
    assert.equal(petFace("speaking", "happy"), "46");
    assert.equal(petFace("speaking", "calm"), "39");
    assert.equal(petFace("thinking", "happy"), "30");
    assert.equal(petFace("listening", "sad"), "12");
    assert.equal(petFace("dialing", "calm"), "35");
    assert.equal(petFace("off", "happy"), "02");
  });
});
