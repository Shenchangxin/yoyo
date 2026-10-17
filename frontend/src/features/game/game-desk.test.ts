import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { askAgentText, deskPage, pageComplete } from "./game-desk.ts";
import type { GameRow } from "./game-model.ts";

const g = {
  stepIntent: "Goal",
  stepBuild: "Make",
  stepPlay: "Play",
  stepCheck: "Review",
  stepIntentLead: "goal lead",
  stepBuildLead: "make lead",
  stepPlayLead: "play lead",
  stepCheckLead: "review lead",
  askIntent: "Ask to make this",
  askBuild: "Ask to make it",
  askPlay: "Say what to change",
  askCheck: "Ask to check it",
  askRepair: "Ask to fix it",
  askWithGoal: "Make the current world from this play goal (do not create a new world): {goal}",
  copilotIntent: "i",
  copilotBuild: "b",
  copilotPlay: "p",
  copilotCheck: "c",
  statusDraft: "draft",
  statusScene: "scene",
  statusRules: "rules",
  statusPlaced: "placed",
  statusAssets: "look",
  statusCompiled: "playable",
};

const compiled: GameRow = {
  id: "w",
  intent: "Collect three relics and reach the shrine.",
  stage: "compiled",
  playable: "game-worlds/x/playable/index.html",
};

describe("game desk pages", () => {
  it("keeps a chosen page even when the world is already playable", () => {
    assert.equal(deskPage(compiled, "intent"), "intent");
    assert.equal(deskPage(compiled, ""), "play");
  });

  it("marks playable worlds complete through play, not check", () => {
    assert.equal(pageComplete("intent", compiled), true);
    assert.equal(pageComplete("build", compiled), true);
    assert.equal(pageComplete("play", compiled), true);
    assert.equal(pageComplete("check", compiled), false);
  });

  it("asks the copilot with the play goal, not a tool name", () => {
    const text = askAgentText("build", compiled, g);
    assert.match(text, /Collect three relics/);
    assert.equal(text.includes("game_"), false);
    assert.match(text, /current world|do not create/i);
  });
});
