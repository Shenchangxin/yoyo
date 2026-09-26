import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { curiousId, dartId, gazeFromPoint, gazeFromScreen, isIdleEmotion } from "./gaze.ts";

describe("gazeFromPoint", () => {
  const box = { left: 0, top: 0, width: 140, height: 140 };

  it("looks fully aside once the pointer is a short reach away", () => {
    const g = gazeFromPoint(box, 200, 70);
    assert.ok(g.nx > 0.9);
    assert.ok(Math.abs(g.ny) < 0.15);
  });

  it("stays near zero at the origin", () => {
    const g = gazeFromPoint(box, 70, 70);
    assert.equal(g.nx, 0);
    assert.equal(g.ny, 0);
  });

  it("uses an explicit window origin for screen follow", () => {
    const g = gazeFromScreen(box, 400, 70, 200, 0);
    assert.ok(g.nx > 0.9);
  });
});

describe("curiousId", () => {
  it("promotes idle to curious and leaves work alone", () => {
    assert.equal(isIdleEmotion("02"), true);
    assert.equal(isIdleEmotion("10"), true);
    assert.equal(curiousId("02"), "03");
    assert.equal(curiousId("30"), "30");
  });
});

describe("dartId", () => {
  it("startles only ambient faces on a fast pointer", () => {
    assert.equal(dartId("02", 2.4), "13");
    assert.equal(dartId("02", 0.2), "02");
    assert.equal(dartId("30", 4), "30");
  });
});
