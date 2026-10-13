import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { classifyLive } from "./stream-cursor.ts";

describe("classifyLive", () => {
  it("drops notices the cursor already covers", () => {
    assert.equal(classifyLive(10, 10, true, true), "stale");
    assert.equal(classifyLive(10, 4, false, false), "stale");
  });

  it("applies the next own inline notice", () => {
    assert.equal(classifyLive(10, 11, true, true), "apply");
  });

  it("moves past another session's contiguous notice without applying it", () => {
    assert.equal(classifyLive(10, 11, false, true), "advance");
  });

  it("pulls on a jump, even when the jumping notice is foreign", () => {
    // A foreign notice at 13 proves nothing about our 11 and 12.
    assert.equal(classifyLive(10, 13, false, true), "pull");
    assert.equal(classifyLive(10, 13, true, true), "pull");
  });

  it("pulls an own notice whose event body was not inlined", () => {
    assert.equal(classifyLive(10, 11, true, false), "pull");
  });

  it("treats a notice without a seq as a plain event", () => {
    assert.equal(classifyLive(10, 0, true, true), "apply");
    assert.equal(classifyLive(10, 0, false, true), "advance");
  });

  it("interleaved sessions are contiguous against the global cursor", () => {
    let cursor = 10;
    const steps = [
      [11, true],
      [12, false],
      [13, true],
    ].map(([seq, own]) => {
      const step = classifyLive(cursor, seq as number, own as boolean, true);
      if (step === "apply" || step === "advance") cursor = seq as number;
      return step;
    });
    assert.deepEqual(steps, ["apply", "advance", "apply"]);
  });

  it("an out-of-order notice pulls, then is stale once the pull covered it", () => {
    // 12 overtakes 11 on the way to the renderer.
    assert.equal(classifyLive(10, 12, true, true), "pull");
    // The pull moved the cursor to the head (12); the late 11 is now stale.
    assert.equal(classifyLive(12, 11, true, true), "stale");
  });
});
