import assert from "node:assert/strict";
import test from "node:test";
import { videoRouteFromLocation, yingceHref } from "./protocol.ts";

test("canvas project handoff keeps the query on the editor route", () => {
  const route = videoRouteFromLocation("/canvas/board%201?conversation=creation%3A1&mode=handoff&asset=a&asset=b");
  assert.equal(route?.pane, "canvas");
  assert.equal(route?.canvasFocus, "editor");
  assert.equal(route?.projectId, "board 1");
  assert.equal(route?.entry, "/canvas/board%201?conversation=creation%3A1&mode=handoff&asset=a&asset=b");
  assert.equal(yingceHref("canvas", "board 1", "editor", route?.entry), route?.entry);
});

test("an existing-canvas agent link stays on the library and keeps agent=1", () => {
  const route = videoRouteFromLocation("/canvas?agent=1");
  assert.equal(route?.pane, "canvas");
  assert.equal(route?.canvasFocus, "library");
  assert.equal(route?.entry, "/canvas?agent=1");
  assert.equal(yingceHref("canvas", "old-board", "library", route?.entry), "/canvas?agent=1");
});

test("studio is a first-class video pane, not a yingce route", () => {
  const route = videoRouteFromLocation("/studio");
  assert.equal(route?.pane, "studio");
  assert.equal(yingceHref("studio", "", "library", ""), "/create");
});

test("switching back to create drops a leftover canvas entry", () => {
  assert.equal(yingceHref("create", "board", "editor", "/canvas/board?agent=1"), "/create");
});

test("eagle library stays on the plugins pane across a remount", () => {
  const route = videoRouteFromLocation("/plugins/eagle");
  assert.equal(route?.pane, "plugins");
  assert.equal(route?.entry, "/plugins/eagle");
  assert.equal(yingceHref("plugins", "", "library", route?.entry), "/plugins/eagle");
  assert.equal(yingceHref("plugins", "", "library", ""), "/plugins");
  assert.equal(yingceHref("assets", "", "library", "/plugins/eagle"), "/assets");
});
