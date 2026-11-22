import assert from "node:assert/strict";
import test from "node:test";
import { buddyScrollKey, readPaneRoute, readPaneTop, resetPanePlace, writePaneRoute, writePaneTop } from "./pane-place.ts";

test("pane route stays in memory until reset", () => {
  resetPanePlace();
  assert.equal(readPaneRoute().tab, "now");
  writePaneRoute("memory", "group-1");
  assert.deepEqual(readPaneRoute(), { tab: "memory", groupId: "group-1" });
  writePaneRoute("nope", "x");
  assert.equal(readPaneRoute().tab, "memory");
  resetPanePlace();
  assert.equal(readPaneRoute().tab, "now");
  assert.equal(readPaneRoute().groupId, "");
});

test("pane tops default to the top and zero is forgotten", () => {
  resetPanePlace();
  assert.equal(readPaneTop("tasks"), 0);
  writePaneTop("tasks", 480);
  writePaneTop("memory", 120);
  assert.equal(readPaneTop("tasks"), 480);
  assert.equal(readPaneTop("memory"), 120);
  writePaneTop("tasks", 0);
  assert.equal(readPaneTop("tasks"), 0);
  assert.equal(readPaneTop("memory"), 120);
  resetPanePlace();
  assert.equal(readPaneTop("memory"), 0);
});

test("buddy screens keep separate keys", () => {
  assert.equal(buddyScrollKey(null, ""), "buddy");
  assert.equal(buddyScrollKey(null, "claude"), "buddy:agent:claude");
  assert.equal(buddyScrollKey({ kind: "prefs" }, "claude"), "buddy:prefs");
  assert.equal(buddyScrollKey({ kind: "log", id: "codex" }, ""), "buddy:log:codex");
});
