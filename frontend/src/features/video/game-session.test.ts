import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { asGameProject, exclusiveGameProjects, gameSessionId, sessionBoundToOtherGame } from "./game-session.ts";

describe("game session isolation", () => {
  it("returns the bound copilot for that world only", () => {
    const projects = [
      { id: "a", kind: "game" as const, title: "A", sessionId: "s-a" },
      { id: "b", kind: "game" as const, title: "B", sessionId: "s-b" },
      { id: "c", kind: "studio" as const, title: "Short", sessionId: "s-c" },
    ];
    assert.equal(gameSessionId("a", projects), "s-a");
    assert.equal(gameSessionId("b", projects), "s-b");
    assert.equal(gameSessionId("missing", projects), "");
    assert.equal(gameSessionId("", projects), "");
  });

  it("does not reuse another world's chat id", () => {
    const projects = [
      { id: "a", kind: "game" as const, title: "A", sessionId: "s-a" },
    ];
    assert.notEqual(gameSessionId("b", projects), "s-a");
    assert.equal(sessionBoundToOtherGame("s-a", "b", projects), true);
    assert.equal(sessionBoundToOtherGame("s-a", "a", projects), false);
  });

  it("clears a stolen session from the previous world", () => {
    const projects = [
      { id: "a", kind: "game" as const, title: "A", sessionId: "s-shared" },
      { id: "b", kind: "game" as const, title: "B", sessionId: "" },
    ];
    const next = exclusiveGameProjects(projects, { id: "b", kind: "game", title: "B" }, "s-shared");
    assert.equal(gameSessionId("b", next), "s-shared");
    assert.equal(gameSessionId("a", next), "");
    assert.equal(sessionBoundToOtherGame("s-shared", "b", next), false);
  });

  it("marks a workspace project as a world", () => {
    const p = asGameProject({ id: "world-1", title: "神庙", sessionId: "s2" });
    assert.equal(p.kind, "game");
    assert.equal(p.sessionId, "s2");
  });
});
