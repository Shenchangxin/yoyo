import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { gateIndex, projectShots, shotReady, STUDIO_GATES } from "./studio-model.ts";

describe("studio gates", () => {
  it("orders setup through delivery", () => {
    assert.deepEqual(STUDIO_GATES, ["setup", "script", "audio", "visuals", "compose", "delivery"]);
    assert.equal(gateIndex("setup"), 0);
    assert.equal(gateIndex("delivery"), 5);
    assert.equal(gateIndex("unknown"), 1);
  });

  it("reads shots from the project list first", () => {
    assert.equal(shotReady({ image_hash: "a" }), true);
    assert.equal(shotReady({ caption: "x" }), false);
    const shots = projectShots({
      id: "1",
      title: "t",
      shots: [{ id: "s1", image_hash: "h" }],
      storyboard: { shots: [{ id: "old" }] },
    });
    assert.equal(shots[0]?.id, "s1");
  });
});
