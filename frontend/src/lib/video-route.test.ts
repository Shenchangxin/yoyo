import assert from "node:assert/strict";
import test from "node:test";
import { isYingcePane, videoRouteFromLocation, yingceHref } from "./protocol.ts";

/**
 * 拥有：视频地址在画布交接、库、工作室、创作台和插件之间如何保留查询与入口。
 * 不拥有：岛内路由的实现。
 */

/** 画布项目交接把查询留在编辑器路由上，href 原样沿用该入口。 */
test("canvas project handoff keeps the query on the editor route", () => {
  const route = videoRouteFromLocation("/canvas/board%201?conversation=creation%3A1&mode=handoff&asset=a&asset=b");
  assert.equal(route?.pane, "canvas");
  assert.equal(route?.canvasFocus, "editor");
  assert.equal(route?.projectId, "board 1");
  assert.equal(route?.entry, "/canvas/board%201?conversation=creation%3A1&mode=handoff&asset=a&asset=b");
  assert.equal(yingceHref("canvas", "board 1", "editor", route?.entry), route?.entry);
});

/** 已有画布的智能体链接留在库上，并保留 agent=1。 */
test("an existing-canvas agent link stays on the library and keeps agent=1", () => {
  const route = videoRouteFromLocation("/canvas?agent=1");
  assert.equal(route?.pane, "canvas");
  assert.equal(route?.canvasFocus, "library");
  assert.equal(route?.entry, "/canvas?agent=1");
  assert.equal(yingceHref("canvas", "old-board", "library", route?.entry), "/canvas?agent=1");
});

/** 工作室是视频表面自己的面板。它的岛内地址是 /create。 */
test("studio is a first-class video pane, not a yingce route", () => {
  const route = videoRouteFromLocation("/studio");
  assert.equal(route?.pane, "studio");
  assert.equal(yingceHref("studio", "", "library", ""), "/create");
});

/** 创作台自己占一个面板。带去画布的入口不会把它收成画布项目。 */
test("create is the conductor, not a yingce workbench", () => {
  assert.equal(isYingcePane("create"), false);
  assert.equal(yingceHref("create", "board", "editor", "/canvas/board?agent=1"), "/canvas");
});

/** 游戏是视频表面自己的面板。 */
test("game is a first-class video pane, not a yingce route", () => {
  assert.equal(isYingcePane("game"), false);
});

/** Eagle 库留在插件面板上。换到资产面板时入口不再沿用插件路径。 */
test("eagle library stays on the plugins pane across a remount", () => {
  const route = videoRouteFromLocation("/plugins/eagle");
  assert.equal(route?.pane, "plugins");
  assert.equal(route?.entry, "/plugins/eagle");
  assert.equal(yingceHref("plugins", "", "library", route?.entry), "/plugins/eagle");
  assert.equal(yingceHref("plugins", "", "library", ""), "/plugins");
  assert.equal(yingceHref("assets", "", "library", "/plugins/eagle"), "/assets");
});
