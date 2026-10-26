/**
 * 本文件拥有视频通道隔离的断言。不改服务端。
 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { laneAcceptsSession, sessionLane } from "./video-lanes.ts";

const projects = [
  { kind: "canvas" as const, sessionId: "s-canvas" },
  { kind: "drama" as const, sessionId: "s-drama" },
  { kind: "studio" as const, sessionId: "s-studio" },
];

describe("video lane isolation", () => {
  /** 已绑定的会话回到自己的工种。 */
  it("maps a bound session to its workshop", () => {
    assert.equal(sessionLane("s-canvas", projects), "canvas");
    assert.equal(sessionLane("s-drama", projects), "drama");
    assert.equal(sessionLane("s-studio", projects), "studio");
    assert.equal(sessionLane("s-free", projects), "");
  });

  /** 创作通道不收画布和短剧的会话。 */
  it("keeps create off workshop threads", () => {
    assert.equal(laneAcceptsSession("create", "s-free", projects), true);
    assert.equal(laneAcceptsSession("create", "s-canvas", projects), false);
    assert.equal(laneAcceptsSession("create", "s-drama", projects), false);
  });

  /** 画布和短剧互不接收对方的会话。 */
  it("keeps canvas off drama, studio, and unbound create threads", () => {
    assert.equal(laneAcceptsSession("canvas", "s-canvas", projects), true);
    assert.equal(laneAcceptsSession("canvas", "s-free", projects), false);
    assert.equal(laneAcceptsSession("canvas", "s-drama", projects), false);
    assert.equal(laneAcceptsSession("drama", "s-canvas", projects), false);
    assert.equal(laneAcceptsSession("drama", "s-drama", projects), true);
  });
});
