import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { laneAcceptsSession, sessionLane } from "./video-lanes.ts";

const projects = [
  { kind: "canvas" as const, sessionId: "s-canvas" },
  { kind: "drama" as const, sessionId: "s-drama" },
  { kind: "studio" as const, sessionId: "s-studio" },
];

describe("video lane isolation", () => {
  it("maps a bound session to its workshop", () => {
    assert.equal(sessionLane("s-canvas", projects), "canvas");
    assert.equal(sessionLane("s-drama", projects), "drama");
    assert.equal(sessionLane("s-studio", projects), "studio");
    assert.equal(sessionLane("s-free", projects), "");
  });

  it("keeps create off workshop threads", () => {
    assert.equal(laneAcceptsSession("create", "s-free", projects), true);
    assert.equal(laneAcceptsSession("create", "s-canvas", projects), false);
    assert.equal(laneAcceptsSession("create", "s-drama", projects), false);
  });

  it("keeps canvas off drama, studio, and unbound create threads", () => {
    assert.equal(laneAcceptsSession("canvas", "s-canvas", projects), true);
    assert.equal(laneAcceptsSession("canvas", "s-free", projects), false);
    assert.equal(laneAcceptsSession("canvas", "s-drama", projects), false);
    assert.equal(laneAcceptsSession("drama", "s-canvas", projects), false);
    assert.equal(laneAcceptsSession("drama", "s-drama", projects), true);
  });
});
