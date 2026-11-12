/**
 * 拥有：输入框 @agent 的端到端用例：Agent 分区的顺序和别名搜索、芯片和去重、派不了的行打开 Buddy、发出去的话带着记号、多 agent 提示，以及用户气泡里的 agent 名字。
 * 不拥有：路由怎么读这些记号（Go 的 route_agents_test.go）。假后端在 mock.ts 和 buddy-mock.ts。每条 test 的标题就是它断言什么。
 */

import { expect, test, type Page } from "@playwright/test";
import { YOYO_BUDDY, buddyFixture } from "./buddy-mock";
import { mockApi } from "./mock";

const CODEX = buddyFixture("codex", {
  name: "Codex",
  vendor: "OpenAI",
  order: 2,
  install: { kind: "system", path: "C:/Users/me/AppData/Roaming/npm/codex-acp.cmd" },
  state: "online",
});
const CLAUDE = buddyFixture("claude", {
  name: "Claude Code",
  vendor: "Anthropic",
  order: 1,
  aliases: ["克劳德"],
  install: { kind: "managed", source: "npm", path: "C:/Users/me/AppData/Local/yoyo/agents/claude" },
  state: "needs_auth",
  auth: { methods: [{ id: "claude-login", name: "Log in with Claude", type: "agent" }] },
});
const GEMINI = buddyFixture("gemini", { name: "Gemini CLI", vendor: "Google", order: 3 });
const BUDDIES = [YOYO_BUDDY, CLAUDE, CODEX, GEMINI];
const SESSIONS = [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }];

/** composer 打开首页，等主输入框可编辑，点进去。编辑区是 contenteditable，没有 textbox 角色，所以按测试标记找。 */
async function composer(page: Page) {
  await page.goto("/");
  const box = page.getByTestId("composer-input");
  await expect(box).toHaveAttribute("contenteditable", "true", { timeout: 15_000 });
  await box.click();
  return box;
}

/** mentions 是 @ 浮层。Buddy 列表在启动后才取，所以先等 Agent 分区出来。 */
async function mentions(page: Page) {
  const list = page.getByRole("listbox", { name: "Mention suggestions" });
  await expect(list.getByText("Agents", { exact: true })).toBeVisible({ timeout: 10_000 });
  return list;
}

test("@ puts installed agents first, routable ones on top, and matches aliases", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  const box = await composer(page);
  await box.pressSequentially("@");
  const list = await mentions(page);
  const rows = list.getByRole("option");
  await expect(rows.nth(0)).toHaveAccessibleName(/^yoyo/);
  await expect(rows.nth(1)).toHaveAccessibleName(/^Codex Online/);
  await expect(rows.nth(2)).toHaveAccessibleName(/^Claude Code/);
  await expect(rows.nth(2)).toHaveAttribute("aria-disabled", "true");
  await expect(rows.nth(1)).not.toHaveAttribute("aria-disabled", "true");
  await expect(list.getByRole("option", { name: /Gemini/ })).toHaveCount(0);
  await expect(list.getByText("Files & folders")).toBeVisible();
  await box.pressSequentially("克劳");
  await expect(list.getByRole("option", { name: /^Claude Code/ })).toBeVisible();
  await expect(list.getByRole("option", { name: /^(yoyo|Codex)\b/ })).toHaveCount(0);
});

test("picking an agent pins one chip with its name, and picking it again adds nothing", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  const box = await composer(page);
  await box.pressSequentially("@cod");
  const list = await mentions(page);
  await list.getByRole("option", { name: /^Codex/ }).click();
  const chip = page.getByTestId("mention-chip");
  await expect(chip).toHaveAttribute("aria-label", "@agent:codex");
  await expect(chip).toHaveAttribute("data-kind", "agent");
  await expect(chip).toHaveText("Codex");
  await box.pressSequentially("@agent:cod");
  await expect(list.getByRole("option", { name: /^Codex/ })).toBeVisible();
  await page.keyboard.press("Enter");
  await expect(page.getByTestId("mention-chip")).toHaveCount(1);
  await expect(box).not.toContainText("@agent:cod");
});

test("an agent that can't take tasks opens Buddy with what to do first instead of pinning", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  const box = await composer(page);
  await box.pressSequentially("@clau");
  const list = await mentions(page);
  await expect(list.getByRole("option", { name: /^Claude Code Needs sign-in/ })).toHaveAttribute("data-active", "true");
  await page.keyboard.press("Enter");
  await expect(page.getByText("Sign in to Claude Code first")).toBeVisible();
  const detail = page.getByTestId("buddy-detail");
  await expect(detail).toBeVisible();
  await expect(detail.getByRole("heading", { name: "Claude Code" })).toBeVisible();
  await expect(page.getByTestId("buddy-login")).toBeVisible();
  // Buddy 在今天栏里翻开，对话和正在写的那句话都还在。
  await expect(page.getByTestId("work-tab-buddy")).toHaveAttribute("aria-selected", "true");
  await expect(page.getByTestId("conversation-column")).toBeVisible();
  await expect(box).toBeVisible();
});

test("two agents show the split hint once and the message carries both tokens", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  const box = await composer(page);
  await box.pressSequentially("@cod");
  await mentions(page);
  await page.keyboard.press("Enter");
  await expect(page.getByTestId("agents-hint")).toHaveCount(0);
  await box.pressSequentially("@yoy");
  await mentions(page);
  await page.keyboard.press("Enter");
  const hint = page.getByTestId("agents-hint");
  await expect(hint).toContainText("Codex, yoyo will split the work");
  await hint.getByRole("button", { name: "Don't show this again" }).click();
  await expect(hint).toHaveCount(0);
  await box.pressSequentially("fix the login page");
  const sent = page.waitForRequest((r) => r.method() === "POST" && /\/api\/sessions\/[^/]+\/messages$/.test(new URL(r.url()).pathname));
  await page.keyboard.press("Enter");
  const text = String((await sent).postDataJSON()?.text || "");
  expect(text).toContain("@agent:codex");
  expect(text).toContain("@agent:yoyo");
  expect(text).toContain("fix the login page");
});

test("the user bubble names the agents it was sent to", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: SESSIONS,
    buddies: BUDDIES,
    events: [{ type: "user", session_id: "s1", ts: "2026-11-09T08:00:00Z", payload: { text: "@agent:codex @agent:claude 修登录页" } }],
  });
  await page.goto("/");
  const chips = page.locator(".user-bubble").getByTestId("user-mention-chip");
  await expect(chips).toHaveCount(2, { timeout: 15_000 });
  await expect(chips.nth(0)).toContainText("Codex", { timeout: 10_000 });
  await expect(chips.nth(1)).toContainText("Claude Code");
  await expect(chips.nth(0)).toHaveAttribute("title", "codex");
  await expect(page.locator(".user-bubble")).toContainText("修登录页");
});
