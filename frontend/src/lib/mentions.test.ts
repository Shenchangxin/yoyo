import assert from "node:assert/strict";
import test from "node:test";
import {
  activeAtTail,
  childPaths,
  composeDraft,
  formatTriggerPath,
  mentionCrumbs,
  mentionQueryOf,
  mentionTokenAt,
  mergeMentionPins,
  pinsFromAttachments,
  sameUserTurnText,
  settleBareAtPaths,
  suggestSkills,
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

test("suggestSkills matches Chinese novel-to-game intent", () => {
  const hits = suggestSkills(
    [
      { name: "plan-first", description: "Keep a visible task plan." },
      { name: "novel-to-game", description: "Turn a novel into a playable game.", slug: "novel-to-game" },
    ],
    "把这本小说做成游戏",
  );
  assert.equal(hits[0]?.name, "novel-to-game");
});

test("mentionTokenAt treats @path like harness, not a kind picker", () => {
  assert.equal(mentionTokenAt("@src/tu", 7)?.token, "@src/tu");
  assert.equal(mentionQueryOf("@src/tu").query, "src/tu");
  assert.equal(mentionQueryOf("@file:src/main.go").kind, "file");
  assert.equal(activeAtTail("read @src/"), "@src/");
  assert.equal(activeAtTail("mail a@b.test"), null);
  assert.deepEqual(mentionQueryOf('@"docs/design n'), { kind: "any", query: "docs/design n", quoted: true });
});

test("mentionQueryOf does not treat @harness.md as the harness pin", () => {
  assert.equal(mentionQueryOf("@harness").kind, "harness");
  assert.equal(mentionQueryOf("@harness.md").kind, "any");
  assert.equal(mentionQueryOf("@harness.md").query, "harness.md");
});

test("settleBareAtPaths rewrites leftover @path to wire tokens", () => {
  const files = [
    { path: "src/main.go", kind: "file" },
    { path: "src", kind: "dir" },
  ];
  assert.equal(settleBareAtPaths("fix @src/main.go please", files), "fix @file:src/main.go please");
  assert.equal(settleBareAtPaths("list @src/", files), "list @folder:src");
  assert.equal(settleBareAtPaths("keep @file:src/main.go", files), "keep @file:src/main.go");
  assert.equal(settleBareAtPaths("mail a@b.test @missing", files), "mail a@b.test @missing");
  assert.equal(settleBareAtPaths("fix @loop.go", [{ path: "src/loop.go", kind: "file" }]), "fix @loop.go");
});

test("composeDraft keeps session titles readable for the model", () => {
  const body = composeDraft(
    [{ token: "@session:s2", kind: "session", label: "Earlier chat", detail: "s2" }],
    "continue that",
  );
  assert.equal(body.includes("@session:s2"), true);
  assert.equal(body.includes("Earlier chat"), true);
  assert.equal(body.includes("continue that"), true);
});

test("folder drill keeps @path/ in the draft and builds crumbs", () => {
  assert.equal(formatTriggerPath("src", false, true), "@src/");
  assert.equal(formatTriggerPath("my dir", true, true), '@"my dir/');
  const crumbs = mentionCrumbs("src/module1/", false, true, "Workspace");
  assert.equal(crumbs?.[0].label, "Workspace");
  assert.equal(crumbs?.[0].query, "@");
  assert.equal(crumbs?.at(-1)?.label, "module1");
  assert.equal(crumbs?.at(-1)?.current, true);
  assert.equal(mentionCrumbs("src/", false, false, "Workspace"), undefined);
  assert.deepEqual(
    childPaths([{ path: "src/main.go", kind: "file" }, { path: "src/lib/x.ts", kind: "file" }, { path: "src/lib", kind: "dir" }], "src"),
    [{ path: "src/main.go", kind: "file" }, { path: "src/lib", kind: "dir" }],
  );
});

test("suggestSkills does not invent a removed game-world skill", () => {
  const hits = suggestSkills(
    [
      { name: "novel-to-game", description: "Turn a novel into a playable game.", slug: "novel-to-game" },
    ],
    "生成一个第三人称收集游戏世界",
  );
  assert.ok(!hits.some((h) => h.name === "yoyo-game"));
});
