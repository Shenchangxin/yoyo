/**
 * 拥有：通话情绪、点头和宠物表情编号的断言。
 * 不拥有：Spine 片段和画布。不改服务端。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { cueMood, nodFor, petFace, standMood } from "./stand.ts";

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

  /** 断言语气提示归到立绘情绪，和服务端同一张表；认不出时为空，立绘保持原来的心情。 */
  it("reads her delivery cue as a mood", () => {
    assert.equal(cueMood("shyly"), "shy");
    assert.equal(cueMood("laughing nervously"), "happy");
    assert.equal(cueMood("trying to hide her disappointment"), "sad");
    assert.equal(cueMood("surprised"), "surprised");
    assert.equal(cueMood(""), "");
    assert.equal(cueMood("robotic"), "");
  });
});
