// 锁住设置导航：按「你的助手」开头的分组顺序；开发者一组默认不进侧栏；旧链接和搬了家的区块落到现行页；开发者区块仍能搜到。
import { test } from "node:test";
import assert from "node:assert/strict";
import { isDeveloperTab, navTabs, resolveSettingsTarget, SETTINGS_GROUPS, SETTINGS_SECTIONS, SETTINGS_TABS } from "./registry.ts";

test("groups start with the assistant and end with developer", () => {
  assert.deepEqual([...SETTINGS_GROUPS], ["agent", "link", "model", "app", "system", "developer"]);
  assert.equal(navTabs()[0].key, "proactive");
});

test("developer tabs stay out of the sidebar until shown", () => {
  const hidden = navTabs(false).map((t) => t.key);
  assert.ok(!hidden.includes("extensions") && !hidden.includes("advanced"));
  const shown = navTabs(true).map((t) => t.key);
  assert.ok(shown.includes("extensions") && shown.includes("advanced"));
  assert.ok(isDeveloperTab("advanced") && !isDeveloperTab("policy"));
  assert.ok(!shown.includes("idol"), "idol opens from its own module");
});

test("every tab key is unique and every section belongs to a tab", () => {
  const keys = SETTINGS_TABS.map((t) => t.key);
  assert.equal(new Set(keys).size, keys.length);
  for (const s of SETTINGS_SECTIONS) assert.ok(keys.includes(s.tab), s.id);
});

test("old links and moved sections land on the current page", () => {
  assert.deepEqual(resolveSettingsTarget("policy", "policy-isolation"), { tab: "advanced", section: "policy-isolation" });
  assert.deepEqual(resolveSettingsTarget("isolation"), { tab: "advanced", section: "policy-isolation" });
  assert.deepEqual(resolveSettingsTarget("advanced", "advanced-doctor"), { tab: "advanced", section: "advanced-logs" });
  assert.deepEqual(resolveSettingsTarget("memory"), { tab: "policy", section: "policy-data" });
  assert.deepEqual(resolveSettingsTarget("generation", "generation-image"), { tab: "generation", section: "generation-image" });
  assert.deepEqual(resolveSettingsTarget("nonsense"), { tab: "proactive", section: "" });
});

test("developer sections can still be searched", () => {
  const dev = SETTINGS_SECTIONS.filter((s) => isDeveloperTab(s.tab));
  assert.ok(dev.length >= 4);
  for (const s of dev) assert.match(s.keywords || "", /developer/);
});
