/**
 * 拥有：Buddy 栏目的端到端用例：分组和底部坞、安装进度、一键配置不回显密钥、登录、空状态推荐、装到系统的确认、自定义 agent，
 * 以及第一次检测期间先出骨架、界面照常能用。
 * 不拥有：Buddy 的后端。假后端在 buddy-mock.ts。每条 test 的标题就是它断言什么。
 */

import { expect, test, type Page } from "@playwright/test";
import { YOYO_BUDDY, buddyFixture } from "./buddy-mock";
import { mockApi } from "./mock";

const LOGIN = { methods: [{ id: "claude-login", name: "Log in with Claude", type: "agent" }] };
const CLAUDE = buddyFixture("claude", { name: "Claude Code", vendor: "Anthropic", order: 1, auth: LOGIN });
const CODEX = buddyFixture("codex", {
  name: "Codex",
  vendor: "OpenAI",
  order: 2,
  install: { kind: "system", path: "C:/Users/me/AppData/Roaming/npm/codex-acp.cmd" },
  version: "0.4.1",
  state: "online",
  checkedAt: "2026-11-09T08:00:00Z",
});
const GEMINI = buddyFixture("gemini", {
  name: "Gemini CLI",
  vendor: "Google",
  order: 3,
  install: { kind: "managed", source: "npm", path: "C:/Users/me/AppData/Local/yoyo/agents/gemini" },
  version: "0.9.0",
  state: "needs_config",
  config: { secrets: [{ name: "GEMINI_API_KEY", label: "Gemini API key", hasKey: false }] },
});
const QWEN = buddyFixture("qwen", { name: "Qwen Code", vendor: "Alibaba", order: 4, featured: false });

/** dock 是侧栏底部坞里的 Buddy 图标。有要处理的项时名字后面带数字说明。 */
function dock(page: Page) {
  return page.locator(".rail-dock").getByRole("button", { name: /^Buddy\b/ });
}

/** openBuddy 从底部坞打开 Buddy，等总览出来。 */
async function openBuddy(page: Page) {
  await page.goto("/");
  await dock(page).click();
  await expect(page.getByTestId("buddy-overview")).toBeVisible();
}

/** rpc 等下一个发往 /api/rpc、方法名是 method 的请求，返回它的参数。 */
async function rpc(page: Page, method: string, act: () => Promise<void>) {
  const sent = page.waitForRequest((r) => r.url().endsWith("/api/rpc") && r.postDataJSON()?.method === method);
  await act();
  return (await sent).postDataJSON().params;
}

test("buddy groups agents and the dock has eight icons", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CLAUDE, CODEX, GEMINI, QWEN] });
  await page.goto("/");
  await expect(page.locator(".rail-dock .glass-icon")).toHaveCount(8);
  await openBuddy(page);
  const overview = page.getByTestId("buddy-overview");
  const online = overview.getByRole("region", { name: "Online" });
  await expect(online).toContainText("yoyo");
  await expect(online).toContainText("Codex");
  await expect(overview.getByRole("region", { name: "Needs attention" })).toContainText("Gemini CLI");
  await expect(overview.getByRole("region", { name: "Available" })).toContainText("Claude Code");
  await expect(overview.getByText("Qwen Code")).toHaveCount(0);
  await overview.getByRole("button", { name: "See 1 more" }).click();
  await expect(overview.getByRole("region", { name: "More agents" })).toContainText("Qwen Code");
  await expect(page.getByTestId("buddy-rail")).toContainText("Codex");
  await expect(page.locator(".rail-dock").getByRole("button", { name: "Buddy · 1 need attention" })).toBeVisible();
  await dock(page).click();
  await expect(page.getByTestId("buddy-workspace")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column")).toBeVisible();
});

test("installing an agent shows each phase, then asks to sign in", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX, CLAUDE] });
  await openBuddy(page);
  const params = await rpc(page, "buddy.install", () => page.getByTestId("buddy-row-claude").getByRole("button", { name: "Install", exact: true }).click());
  expect(params).toEqual({ id: "claude", mode: "managed" });
  const status = page.getByTestId("buddy-status-claude");
  await expect(status).toContainText("Preparing Node runtime");
  await expect(status).toContainText("Installing dependencies");
  await expect(status).toContainText("Needs sign-in", { timeout: 10_000 });
  await expect(page.getByText("Claude Code is installed. Sign in to finish.")).toBeVisible();
  await expect(page.getByTestId("buddy-overview").getByRole("region", { name: "Needs attention" })).toContainText("Claude Code");
});

test("one-click setup sends the key once and never shows it again", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX, GEMINI] });
  await openBuddy(page);
  await page.getByTestId("buddy-row-gemini").getByRole("button", { name: "Set up", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Set up Gemini CLI" });
  await dialog.getByTestId("buddy-key-GEMINI_API_KEY").fill("sk-e2e-first-123");
  const first = await rpc(page, "buddy.configure", () => dialog.getByRole("button", { name: "Save and check" }).click());
  expect(first.secrets).toEqual({ GEMINI_API_KEY: "sk-e2e-first-123" });
  await expect(dialog).toHaveCount(0);
  await expect(page.getByTestId("buddy-status-gemini")).toContainText("Online", { timeout: 10_000 });
  await expect(page.getByText("Gemini CLI is ready")).toBeVisible();

  await page.getByTestId("buddy-row-gemini").getByRole("button", { name: /^Gemini CLI/ }).click();
  const detail = page.getByTestId("buddy-detail");
  const key = detail.getByTestId("buddy-key-GEMINI_API_KEY");
  await expect(key).toHaveValue("");
  await expect(detail.getByTestId("buddy-config").getByText("Saved", { exact: true })).toBeVisible();
  await key.fill("sk-e2e-second-456");
  const second = await rpc(page, "buddy.configure", () => detail.getByRole("button", { name: "Save", exact: true }).click());
  expect(second.secrets).toEqual({ GEMINI_API_KEY: "sk-e2e-second-456" });
  await expect(key).toHaveValue("");
  await expect(detail.getByRole("status").filter({ hasText: "Saved" })).toBeVisible();
  expect(await page.content()).not.toContain("sk-e2e-");
});

test("sign in sends buddy.login and the agent comes online", async ({ page }) => {
  const claude = { ...CLAUDE, install: { kind: "managed", source: "npm", path: "C:/agents/claude" }, state: "needs_auth" };
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX, claude] });
  await openBuddy(page);
  const params = await rpc(page, "buddy.login", () => page.getByTestId("buddy-row-claude").getByRole("button", { name: "Sign in", exact: true }).click());
  expect(params.id).toBe("claude");
  await expect(page.getByTestId("buddy-status-claude")).toContainText("Online", { timeout: 10_000 });
  await expect(page.getByText("Claude Code is ready")).toBeVisible();
});

test("no external agent shows the empty state with three picks", async ({ page }) => {
  const codex = buddyFixture("codex", { name: "Codex", order: 2 });
  const gemini = buddyFixture("gemini", { name: "Gemini CLI", order: 3 });
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CLAUDE, codex, gemini, QWEN] });
  await openBuddy(page);
  const overview = page.getByTestId("buddy-overview");
  await expect(overview.getByText("No agent apps yet")).toBeVisible();
  for (const name of ["Claude Code", "Codex", "Gemini CLI"]) {
    await expect(overview.getByRole("button", { name: `Install ${name}` })).toBeVisible();
  }
  await expect(overview.getByTestId("buddy-row-claude")).toHaveCount(0);
  const params = await rpc(page, "buddy.install", () => overview.getByRole("button", { name: "Install Codex" }).click());
  expect(params.id).toBe("codex");
  await expect(overview.getByTestId("buddy-row-codex")).toBeVisible();
});

test("installing to the system asks first and shows the full command", async ({ page }) => {
  const command = "npm install -g @agentclientprotocol/claude-agent-acp";
  const claude = { ...CLAUDE, canManage: false, systemInstall: { available: true, command } };
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX, claude] });
  await openBuddy(page);
  await page.getByTestId("buddy-row-claude").getByRole("button", { name: "Install to system", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Install Claude Code to the system?" });
  await expect(dialog.getByTestId("buddy-command")).toHaveText(command);
  const params = await rpc(page, "buddy.install", () => dialog.getByRole("button", { name: "Run it" }).click());
  expect(params).toEqual({ id: "claude", mode: "system" });
  await expect(dialog).toHaveCount(0);
});

test("a custom agent can be added and opens its page", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX] });
  await openBuddy(page);
  await page.getByTestId("buddy-overview").getByRole("button", { name: "Add a custom agent" }).click();
  const dialog = page.getByRole("dialog", { name: "Add a custom agent" });
  await dialog.getByLabel("Name", { exact: true }).fill("My Agent");
  await dialog.getByLabel("Command", { exact: true }).fill("my-agent");
  await dialog.getByLabel("Arguments", { exact: true }).fill("--acp");
  const params = await rpc(page, "buddy.add", () => dialog.getByRole("button", { name: "Add", exact: true }).click());
  expect(params).toEqual({ name: "My Agent", command: "my-agent", args: ["--acp"], env: {} });
  await expect(page.getByTestId("buddy-detail").getByRole("heading", { name: "My Agent" })).toBeVisible();
  await expect(page.getByTestId("buddy-detail-status")).toContainText("Online", { timeout: 10_000 });
});

test("opening Buddy during the first scan shows a skeleton and the app stays usable", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX] });
  let scanning = true;
  await page.route("**/api/rpc", async (route) => {
    const body = route.request().postDataJSON();
    if (body?.method !== "buddy.list" || !scanning) return route.fallback();
    await new Promise((done) => setTimeout(done, 800));
    return route.fulfill({ json: { jsonrpc: "2.0", id: body.id ?? 1, result: { buddies: [YOYO_BUDDY], detecting: true } } });
  });
  await page.goto("/");
  await dock(page).click();
  await expect(page.getByTestId("buddy-skeleton")).toBeVisible();
  await dock(page).click();
  await expect(page.getByTestId("conversation-column")).toBeVisible();
  await dock(page).click();
  await expect(page.getByTestId("buddy-skeleton")).toBeVisible();
  scanning = false;
  await expect(page.getByTestId("buddy-overview").getByRole("region", { name: "Online" })).toContainText("Codex", { timeout: 10_000 });
  await expect(page.getByTestId("buddy-skeleton")).toHaveCount(0);
});
