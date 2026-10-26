/**
 * 拥有：草稿编辑器在 React 19 下延迟第一次绘制的断言。
 * 不拥有：编辑器。不调用 api。
 */
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "node:test";

const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "DraftEditor.tsx"), "utf8");

/** 草稿编辑器推迟第一次绘制，重新挂载时不会去读空视图。 */
test("DraftEditor defers TipTap first paint so Yingce remounts cannot read a null view", () => {
  assert.match(src, /immediatelyRender:\s*false/);
  assert.match(src, /function liveEditor/);
  assert.match(src, /!editor\.isDestroyed/);
  assert.match(src, /sc-placeholder/);
});
