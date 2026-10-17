import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { laneAcceptsSession, sessionLane } from "./video-lanes.ts";

const projects = [
  { kind: "game" as const, sessionId: "s-game" },
  { kind: "drama" as const, sessionId: "s-drama" },
  { kind: "studio" as const, sessionId: "s-studio" },
];

describe("video lane isolation", () => {
  it("maps a bound session to its workshop", () => {
    assert.equal(sessionLane("s-game", projects), "game");
    assert.equal(sessionLane("s-drama", projects), "drama");
    assert.equal(sessionLane("s-studio", projects), "studio");
    assert.equal(sessionLane("s-free", projects), "");
  });

  it("keeps create off workshop threads", () => {
    assert.equal(laneAcceptsSession("create", "s-free", projects), true);
    assert.equal(laneAcceptsSession("create", "s-game", projects), false);
    assert.equal(laneAcceptsSession("create", "s-drama", projects), false);
  });

  it("keeps game off drama, studio, and unbound create threads", () => {
    assert.equal(laneAcceptsSession("game", "s-game", projects), true);
    assert.equal(laneAcceptsSession("game", "s-free", projects), false);
    assert.equal(laneAcceptsSession("game", "s-drama", projects), false);
    assert.equal(laneAcceptsSession("drama", "s-game", projects), false);
    assert.equal(laneAcceptsSession("drama", "s-drama", projects), true);
  });
});
