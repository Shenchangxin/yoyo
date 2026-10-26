import assert from "node:assert/strict";
import test from "node:test";
import { overlayActiveIndex } from "./mentions.ts";
import { filterSlash, slashCatalog, slashQuery } from "./slash.ts";
import { en } from "./copy.ts";

/**
 * 拥有：斜杠触发位置、粘连路径不算命令，以及高亮项在列表刷新后保持。
 * 不拥有：命令目录的文案，以及提及菜单的实现。
 */

/** 行首或空白后的斜杠算命令，大小写收成小写。 */
test("slashQuery matches / at the caret after start or whitespace", () => {
  assert.equal(slashQuery("/"), "/");
  assert.equal(slashQuery("/plan"), "/plan");
  assert.equal(slashQuery("/PLAN"), "/plan");
  assert.equal(slashQuery(" /"), "/");
  assert.equal(slashQuery("note /arxiv"), "/arxiv");
  assert.equal(slashQuery("a@b.test /new"), "/new");
  assert.equal(slashQuery("hello\n/"), "/");
});

/** 空串、普通文字、粘在词中间的路径，以及斜杠后面还有内容时，都不触发。 */
test("slashQuery does not treat glued paths or email as a slash trigger", () => {
  assert.equal(slashQuery(""), null);
  assert.equal(slashQuery("hello"), null);
  assert.equal(slashQuery("src/main"), null);
  assert.equal(slashQuery("see src/"), null);
  assert.equal(slashQuery("/plan more"), null);
});

/** 前缀只有斜杠时保留目录，更长前缀只留匹配的命令。 */
test("filterSlash keeps the command catalog when the prefix is /", () => {
  const items = slashCatalog(en);
  assert.ok(filterSlash("/", items).length >= 8);
  assert.ok(filterSlash("/pl", items).some((s) => s.cmd === "/plan"));
});

/** 列表刷新后按 token 保持高亮。找不到或列表为空时回到第一项。 */
test("overlayActiveIndex keeps the highlighted token across list refresh", () => {
  const first = [{ token: "@file:a" }, { token: "@session:s2" }, { token: "@skill:x" }];
  assert.equal(overlayActiveIndex(first, "@session:s2"), 1);
  const refreshed = [{ token: "@file:b" }, { token: "@session:s2" }];
  assert.equal(overlayActiveIndex(refreshed, "@session:s2"), 1);
  assert.equal(overlayActiveIndex(refreshed, "@skill:x"), 0);
  assert.equal(overlayActiveIndex([], "@session:s2"), 0);
});
