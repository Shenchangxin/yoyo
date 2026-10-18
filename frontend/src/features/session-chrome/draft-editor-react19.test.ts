import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "node:test";

const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "DraftEditor.tsx"), "utf8");

test("DraftEditor defers TipTap first paint so Yingce remounts cannot read a null view", () => {
  assert.match(src, /immediatelyRender:\s*false/);
  assert.match(src, /function liveEditor/);
  assert.match(src, /!editor\.isDestroyed/);
  assert.match(src, /sc-placeholder/);
});
