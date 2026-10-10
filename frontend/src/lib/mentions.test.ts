import assert from "node:assert/strict";
import test from "node:test";
import {
  mergeMentionPins,
  pinsFromAttachments,
  sameUserTurnText,
} from "./mentions.ts";

test("pinsFromAttachments skips images and uses the basename", () => {
  const pins = pinsFromAttachments([
    { name: "cover.png", path: ".yoyo/uploads/cover.png", mime: "image/png" },
    { name: "我的小说.md", path: ".yoyo/uploads/我的小说.md", mime: "text/markdown" },
  ]);
  assert.equal(pins.length, 1);
  assert.equal(pins[0].kind, "file");
  assert.equal(pins[0].label, "我的小说.md");
  assert.equal(pins[0].token, "@file:.yoyo/uploads/我的小说.md");
});

test("mergeMentionPins dedupes stamped @file tokens", () => {
  const fromText = [{ token: "@file:.yoyo/uploads/a.md", kind: "file" as const, label: "a.md", detail: ".yoyo/uploads/a.md" }];
  const fromAtt = pinsFromAttachments([{ name: "a.md", path: ".yoyo/uploads/a.md" }]);
  assert.equal(mergeMentionPins(fromText, fromAtt).length, 1);
});

test("sameUserTurnText treats stamped @file as the same optimistic turn", () => {
  assert.equal(sameUserTurnText("@skill:novel-to-game", "@skill:novel-to-game\n@file:.yoyo/uploads/novel.md"), true);
  assert.equal(sameUserTurnText("@skill:novel-to-game\n@file:.yoyo/uploads/novel.md", "@skill:novel-to-game"), true);
  assert.equal(sameUserTurnText("", "@file:.yoyo/uploads/novel.md"), true);
  assert.equal(sameUserTurnText("@skill:novel-to-game", "@skill:other-skill"), false);
  assert.equal(sameUserTurnText("adapt this", "adapt this\n@file:.yoyo/uploads/novel.md"), true);
});
