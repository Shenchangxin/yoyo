import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { asStudioProject, studioSessionId } from "./studio-session.ts";

describe("studio session isolation", () => {
  it("returns the bound copilot for that short only", () => {
    const projects = [
      { id: "a", kind: "studio" as const, title: "A", sessionId: "s-a" },
      { id: "b", kind: "studio" as const, title: "B", sessionId: "s-b" },
      { id: "c", kind: "canvas" as const, title: "Board", sessionId: "s-c" },
    ];
    assert.equal(studioSessionId("a", projects), "s-a");
    assert.equal(studioSessionId("b", projects), "s-b");
    assert.equal(studioSessionId("missing", projects), "");
    assert.equal(studioSessionId("", projects), "");
  });

  it("does not reuse another short's chat id", () => {
    const projects = [
      { id: "a", kind: "studio" as const, title: "A", sessionId: "s-a" },
    ];
    assert.notEqual(studioSessionId("b", projects), "s-a");
  });

  it("marks a workspace project as a short", () => {
    const p = asStudioProject({ id: "studio-1", title: "春日通勤", sessionId: "s2" });
    assert.equal(p.kind, "studio");
    assert.equal(p.sessionId, "s2");
  });
});
