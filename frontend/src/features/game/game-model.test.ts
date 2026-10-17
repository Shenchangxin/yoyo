import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { gameNext, gameStepOf, hasPlayable, qaStatus, stepDone, type GameRow } from "./game-model.ts";

const compiled: GameRow = {
  id: "99774c3880224c7697b9340f55c605cd",
  title: "Relic run",
  stage: "compiled",
  playable: "game-worlds/3d-99774c38/playable/index.html",
};

describe("game desk next action", () => {
  it("plays a compiled world instead of repairing it", () => {
    assert.equal(hasPlayable(compiled), true);
    assert.equal(qaStatus(compiled), "pending");
    assert.equal(gameNext(compiled), "play");
    assert.equal(gameStepOf(compiled.stage, compiled), "play");
  });

  it("treats host QA PASS as the check step", () => {
    const row = { ...compiled, stage: "compiled", verification: { status: "PASS" } };
    assert.equal(qaStatus(row), "pass");
    assert.equal(gameNext(row), "play");
    assert.equal(gameStepOf(row.stage, row), "check");
    assert.equal(stepDone(row.stage, "play", row), true);
  });

  it("repairs only after QA FAIL", () => {
    const row = { ...compiled, verification: { status: "FAIL" } };
    assert.equal(qaStatus(row), "fail");
    assert.equal(gameNext(row), "repair");
    assert.equal(gameStepOf(row.stage, row), "play");
  });
});
