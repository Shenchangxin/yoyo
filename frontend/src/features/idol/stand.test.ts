import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { nodFor, petFace, standMood } from "./stand.ts";

describe("call stand", () => {
  it("keeps the seven emotions and falls back to calm", () => {
    assert.equal(standMood("HAPPY"), "happy");
    assert.equal(standMood("surprised"), "surprised");
    assert.equal(standMood("nope"), "calm");
  });

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
