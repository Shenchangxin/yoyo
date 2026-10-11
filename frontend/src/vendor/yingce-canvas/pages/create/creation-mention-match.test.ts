import assert from "node:assert/strict";
import test from "node:test";

// @ts-expect-error -- Node 原生 TypeScript 测试运行器需要保留扩展名。
import { promptContainsResolvedReference, removeResolvedReferenceTokens } from "./creation-mention-match.ts";

const picture1 = { label: "图片1", token: "@[attachment:pic-1]" };
const picture10 = { label: "图片10", token: "@[attachment:pic-10]" };
const storyboard = { label: "分镜", token: "@[skill:storyboard]" };
const storyboardPlus = { label: "分镜增强", token: "@[skill:storyboard-plus]" };

test("删除较短的引用名不会伤到更长的引用", () => {
  const prompt = "@图片1 放在左边，@图片10 放在右边，@分镜 之后接 @分镜增强";
  const next = removeResolvedReferenceTokens(prompt, [picture1, picture10, storyboard, storyboardPlus]);
  assert.equal(next.includes("@图片10"), false);
  assert.equal(next.includes("@分镜增强"), false);
  assert.equal(next.includes("@图片1"), false);

  const onlyShort = removeResolvedReferenceTokens(prompt, [picture1, storyboard]);
  assert.match(onlyShort, /@图片10/);
  assert.match(onlyShort, /@分镜增强/);
  assert.doesNotMatch(onlyShort, /@图片1(?!0)/);
  assert.doesNotMatch(onlyShort, /@分镜(?!增强)/);
});

test("选中引用时按名字边界匹配", () => {
  const prompt = "使用 @图片10 和 @分镜增强";
  assert.equal(promptContainsResolvedReference(prompt, picture1.label, picture1.token), false);
  assert.equal(promptContainsResolvedReference(prompt, picture10.label, picture10.token), true);
  assert.equal(promptContainsResolvedReference(prompt, storyboard.label, storyboard.token), false);
  assert.equal(promptContainsResolvedReference(prompt, storyboardPlus.label, storyboardPlus.token), true);
  assert.equal(promptContainsResolvedReference("看 @[attachment:pic-1]", picture1.label, picture1.token), true);
});
