/**
 * 拥有：画布空间索引的两条纯函数用例。
 * 不拥有：索引的实现。每条 test 的标题就是这则用例在断言什么。
 */

import { expect, test } from "@playwright/test";
import { buildCanvasSpatialIndex, canvasNodeBounds } from "../src/vendor/yingce-canvas/lib/canvas/canvas-spatial-index";

test("spatial index queries intersecting entries in source order", () => {
  const index = buildCanvasSpatialIndex([
    { id: "far", bounds: { left: 2048, top: 0, right: 2200, bottom: 120 }, value: "far" },
    { id: "near", bounds: { left: 900, top: 40, right: 1100, bottom: 160 }, value: "near" },
    { id: "cross", bounds: { left: 1000, top: -40, right: 2100, bottom: 40 }, value: "cross" },
  ], 1024);
  expect(index.query({ left: 950, top: 0, right: 1150, bottom: 100 })).toEqual(["near", "cross"]);
  expect(index.query({ left: 2100, top: 0, right: 2200, bottom: 100 })).toEqual(["far"]);
});

test("spatial index builds bounds from node geometry", () => {
  expect(canvasNodeBounds({ position: { x: -20, y: 30 }, width: 120, height: 80 })).toEqual({
    left: -20, top: 30, right: 100, bottom: 110,
  });
});
