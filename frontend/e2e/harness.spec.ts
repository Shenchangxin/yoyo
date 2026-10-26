/**
 * 拥有：材料轨的端到端用例，包括总览、提议和证明。
 * 不拥有：晋升公式。每条 test 的标题就是这则用例在断言什么。
 */

import { expect, test } from "@playwright/test";
import { mockApi } from "./mock";

test("harness is a single rail entry with an overview", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    harness: { active: "aaa1111", refs: { active: "aaa1111", staging: "bbb2222" } },
  });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Harbor", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Evolve", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByTestId("harness-workspace")).toBeVisible();
  await expect(page.getByTestId("harness-overview")).toBeVisible();
  await expect(page.getByText("Checks passed. You can start using this new version.")).toBeVisible();
  await expect(page.getByRole("tab", { name: "Status" })).toHaveAttribute("aria-selected", "true");
});

test("harness overview shows evolution path and artifact actions", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    harness: {
      active: "aaa1111",
      refs: { active: "aaa1111", staging: "bbb2222" },
      lineage: [
        {
          hash: "bbb2222",
          parent: "aaa1111",
          note: "ace stage",
          refs: ["staging"],
          changes: [{ surface: "playbook", op: "added", detail: "Write the output file before exploring." }],
        },
        { hash: "aaa1111", note: "seeded default harness", refs: ["active"], seed: true },
      ],
    },
  });
  await page.goto("/");
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByTestId("harness-lineage")).toBeVisible();
  await expect(page.getByText("Write the output file before exploring.")).toBeVisible();
  await expect(page.getByRole("heading", { name: "What changed" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Open files" }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Open saved copies" })).toBeVisible();
});

test("harness chat dock is off until toggled", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByRole("button", { name: "Toggle chat" })).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByRole("button", { name: "Expand" })).toHaveCount(0);
  await page.getByRole("button", { name: "Toggle chat" }).click();
  await expect(page.getByRole("button", { name: "Expand" })).toBeVisible();
});

test("propose tab can evolve this thread without moving active", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: /Demo thread/ }).click();
  await page.getByRole("button", { name: /^Harness/ }).click();
  await page.getByRole("tab", { name: "Improve" }).click();
  await expect(page.getByRole("button", { name: "Improve the agent" })).toBeVisible();
  const overlay = page.getByTestId("overlay-thread");
  await expect(overlay).toBeVisible();
  await expect(overlay).toContainText("Only this chat");
  await expect(page.getByTestId("overlay-library")).toContainText("No reusable lesson");
  await overlay.getByRole("button", { name: "Improve this chat" }).click();
  await expect(page.getByTestId("overlay-next")).toContainText("No safer change stood out. This chat stays as it is.");
  await expect(page.getByTestId("overlay-next")).toContainText("4 tries");
  await expect(page.getByTestId("overlay-next")).toContainText("No reusable lesson saved.");
  const spend = page.getByTestId("lab-spend");
  await expect(spend).toBeVisible();
  await expect(spend).toContainText("This chat");
  await expect(spend).toContainText("4 tries");
});

test("propose tab can prove a pinned overlay without checkout copy", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    overlayAdopt: true,
  });
  await page.goto("/");
  await page.getByRole("button", { name: /Demo thread/ }).click();
  await page.getByRole("button", { name: /^Harness/ }).click();
  await page.getByRole("tab", { name: "Improve" }).click();
  const overlay = page.getByTestId("overlay-thread");
  await overlay.getByRole("button", { name: "Improve this chat" }).click();
  await expect(page.getByTestId("overlay-next")).toContainText("This chat now follows an updated version");
  await overlay.getByRole("button", { name: "Check this chat" }).click();
  await expect(page.getByRole("tab", { name: "Check" })).toHaveAttribute("aria-selected", "true");
});

test("prove tab has one primary eval button", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: /^Harness/ }).click();
  await page.getByRole("tab", { name: "Check" }).click();
  await expect(page.getByRole("button", { name: "Check", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "More checks" })).toBeVisible();
  await expect(page.getByRole("button", { name: "TB subset" })).toHaveCount(0);
});

test("settings replaces the thread list and returns via sessions", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "hello from demo" } },
    ],
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await expect(page.getByRole("navigation", { name: "Settings" })).toBeVisible();
  await expect(page.getByRole("button", { name: "General", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Demo thread/ })).toHaveCount(0);
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await expect(page.getByRole("button", { name: /Demo thread/ })).toBeVisible();
  await expect(page.getByTestId("conversation-column").getByText("hello from demo")).toBeVisible();
});

test("zh-CN primary harness verbs have no leftover paper words", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    harness: { active: "aaa1111", refs: { active: "aaa1111", staging: "bbb2222" } },
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Appearance", exact: true }).click();
  await page.getByRole("radio", { name: "中文" }).click();
  await page.getByRole("button", { name: "会话", exact: true }).click();
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByRole("tab", { name: "状态" })).toBeVisible();
  await expect(page.getByText("检查已通过。可以开始使用这个新版本。")).toBeVisible();
  await expect(page.getByText("held-in")).toHaveCount(0);
  await expect(page.getByText("canary")).toHaveCount(0);
  await page.getByRole("tab", { name: "改进" }).click();
  await expect(page.getByRole("button", { name: "改进这个对话" })).toBeVisible();
  await expect(page.getByRole("button", { name: "改进助手" })).toBeVisible();
  await page.getByRole("tab", { name: "检查" }).click();
  await expect(page.getByRole("button", { name: "检查", exact: true })).toBeVisible();
  await page.getByRole("tab", { name: "使用" }).click();
  await expect(page.getByRole("button", { name: "使用这个版本" })).toBeVisible();
});

test("first visit shows three steps when there is no version yet", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { harness: { refs: {} } });
  await page.goto("/");
  await page.getByRole("button", { name: /^Harness/ }).click();
  const first = page.getByTestId("harness-first-run");
  await expect(first).toBeVisible();
  await expect(first).toContainText("Three steps");
  await expect(first).toContainText("Improve this chat, or the agent for every chat.");
  await first.getByRole("button", { name: "Improve" }).click();
  await expect(page.getByRole("tab", { name: "Improve" })).toHaveAttribute("aria-selected", "true");
});

test("chat can jump to improve this chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "hello from demo" } },
    ],
  });
  await page.goto("/");
  await page.getByRole("button", { name: /Demo thread/ }).click();
  await expect(page.getByTestId("chat-not-working")).toBeVisible();
  await page.getByTestId("chat-not-working").click();
  await expect(page.getByTestId("harness-workspace")).toBeVisible();
  await expect(page.getByRole("tab", { name: "Improve" })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("button", { name: "Improve this chat" })).toBeVisible();
});

test("laboratory switch reveals owner tables", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Appearance", exact: true }).click();
  await page.getByRole("switch", { name: "Show laboratory" }).click();
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await page.getByRole("button", { name: /^Harness/ }).click();
  await page.getByRole("tab", { name: "Improve" }).click();
  await expect(page.getByRole("heading", { name: "Same-budget comparison" })).toBeVisible();
  await expect(page.getByRole("columnheader", { name: "Try" })).toBeVisible();
  const advanced = page.getByText("Advanced", { exact: true });
  await advanced.nth(0).click();
  await expect(page.getByText("Try in parallel")).toBeVisible();
  await advanced.nth(1).click();
  await expect(page.getByText("Full check set")).toBeVisible();
  await page.getByRole("tab", { name: "Check" }).click();
  await page.getByRole("button", { name: "More checks" }).click();
  await expect(page.getByRole("menuitem", { name: "Full check set" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menuitem", { name: "Full check set" })).toHaveCount(0);
  await expect(page.getByTestId("harness-workspace")).toBeVisible();
  await page.getByRole("tab", { name: "Use" }).click();
  await expect(page.getByText("deadbeef")).toBeVisible();
  await expect(page.getByRole("button", { name: "Compare any two versions" })).toBeVisible();
});
