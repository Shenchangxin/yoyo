import assert from "node:assert/strict";
import test from "node:test";
import { overlayActiveIndex } from "./mentions.ts";
import { filterSlash, slashCatalog, slashQuery } from "./slash.ts";
import { en } from "./copy.ts";

test("slashQuery matches / at the caret after start or whitespace", () => {
  assert.equal(slashQuery("/"), "/");
  assert.equal(slashQuery("/plan"), "/plan");
  assert.equal(slashQuery("/PLAN"), "/plan");
  assert.equal(slashQuery(" /"), "/");
  assert.equal(slashQuery("note /arxiv"), "/arxiv");
  assert.equal(slashQuery("a@b.test /new"), "/new");
  assert.equal(slashQuery("hello\n/"), "/");
});

test("slashQuery does not treat glued paths or email as a slash trigger", () => {
  assert.equal(slashQuery(""), null);
  assert.equal(slashQuery("hello"), null);
  assert.equal(slashQuery("src/main"), null);
  assert.equal(slashQuery("see src/"), null);
  assert.equal(slashQuery("/plan more"), null);
});

test("filterSlash keeps the command catalog when the prefix is /", () => {
  const items = slashCatalog(en);
  assert.ok(filterSlash("/", items).length >= 8);
  assert.ok(filterSlash("/pl", items).some((s) => s.cmd === "/plan"));
});

test("overlayActiveIndex keeps the highlighted token across list refresh", () => {
  const first = [{ token: "@file:a" }, { token: "@session:s2" }, { token: "@skill:x" }];
  assert.equal(overlayActiveIndex(first, "@session:s2"), 1);
  const refreshed = [{ token: "@file:b" }, { token: "@session:s2" }];
  assert.equal(overlayActiveIndex(refreshed, "@session:s2"), 1);
  assert.equal(overlayActiveIndex(refreshed, "@skill:x"), 0);
  assert.equal(overlayActiveIndex([], "@session:s2"), 0);
});
