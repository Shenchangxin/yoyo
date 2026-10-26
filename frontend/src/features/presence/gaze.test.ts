/** 拥有视线和闲置脸切换的断言。不拥有情绪球。不改服务端。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { curiousId, dartId, gazeFromPoint, gazeFromScreen, gazeGain, isIdleEmotion, lingerId } from "./gaze.ts";

describe("gazeFromPoint", () => {
  const box = { left: 0, top: 0, width: 140, height: 140 };

  /** 指针离开一小段后，视线就完全偏向那边。 */
  it("looks fully aside once the pointer is a short reach away", () => {
    const g = gazeFromPoint(box, 200, 70);
    assert.ok(g.nx > 0.9);
    assert.ok(Math.abs(g.ny) < 0.15);
  });

  /** 竖直视线比水平视线更浅。 */
  it("keeps vertical gaze shallower than horizontal", () => {
    const side = gazeFromPoint(box, 200, 70);
    const down = gazeFromPoint(box, 70, 200);
    assert.ok(Math.abs(down.ny) < Math.abs(side.nx));
    assert.ok(Math.abs(down.ny) < 0.7);
  });

  /** 指针在中心时视线是零。 */
  it("stays near zero at the origin", () => {
    const g = gazeFromPoint(box, 70, 70);
    assert.equal(g.nx, 0);
    assert.equal(g.ny, 0);
  });

  /** 屏幕跟随用传入的窗口原点。 */
  it("uses an explicit window origin for screen follow", () => {
    const g = gazeFromScreen(box, 400, 70, 200, 0);
    assert.ok(g.nx > 0.9);
  });
});

describe("gazeGain", () => {
  /** 从远处一瞥到看全，中间是连续的。 */
  it("eases from a far glance to a full look without a step", () => {
    const far = gazeGain(400, 168);
    const mid = gazeGain(120, 168);
    const near = gazeGain(8, 168);
    assert.ok(far < mid && mid < near);
    assert.ok(far <= 0.3);
    assert.ok(near > 0.95);
  });
});

describe("curiousId", () => {
  /** 闲置脸换成好奇，干活的脸不动。 */
  it("promotes idle to curious and leaves work alone", () => {
    assert.equal(isIdleEmotion("02"), true);
    assert.equal(isIdleEmotion("10"), true);
    assert.equal(curiousId("02"), "03");
    assert.equal(curiousId("30"), "30");
  });
});

describe("dartId", () => {
  /** 只有闲置脸在指针很快时才受惊。 */
  it("startles only ambient faces on a fast pointer", () => {
    assert.equal(dartId("02", 2.4), "13");
    assert.equal(dartId("02", 0.2), "02");
    assert.equal(dartId("30", 4), "30");
    assert.equal(dartId("00", 2.4), "07");
  });
});

describe("lingerId", () => {
  /** 在闲置脸上停留久了就害羞。 */
  it("goes shy after lingering on an idle face", () => {
    assert.equal(lingerId("02", true), "14");
    assert.equal(lingerId("02", false), "03");
    assert.equal(lingerId("30", true), "30");
    assert.equal(lingerId("02", true, 3000), "09");
    assert.equal(lingerId("02", true, 5000), "22");
  });
});
