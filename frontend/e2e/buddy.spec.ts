/**
 * 拥有：Buddy 的端到端用例。Buddy 是会话今天栏里的一页，和今天、任务、记忆并排：分组和页签上的数字、安装进度、
 * 就地配置不回显密钥、登录、空状态推荐、装到系统前先看完整命令、自定义 agent、在这台电脑上发现的应用，
 * 以及第一次检测期间先出占位、对话照常能用。
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

/** tab 是今天栏里的 Buddy 页签。有要处理的项时后面带数字。 */
function tab(page: Page) {
  return page.getByTestId("work-tab-buddy");
}

/** openBuddy 翻到今天栏里的 Buddy 页，等名单出来。 */
async function openBuddy(page: Page) {
  await page.goto("/");
  await tab(page).click();
  await expect(page.getByTestId("buddy-overview")).toBeVisible();
}

/** rpc 等下一个发往 /api/rpc、方法名是 method 的请求，返回它的参数。 */
async function rpc(page: Page, method: string, act: () => Promise<void>) {
  const sent = page.waitForRequest((r) => r.url().endsWith("/api/rpc") && r.postDataJSON()?.method === method);
  await act();
  return (await sent).postDataJSON().params;
}

test("buddy is a tab beside today, tasks and memory, and the dock is back to seven icons", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CLAUDE, CODEX, GEMINI, QWEN] });
  await page.goto("/");
  await expect(page.locator(".rail-dock .glass-icon")).toHaveCount(7);
  await expect(page.locator(".rail-dock").getByRole("button", { name: /^Buddy\b/ })).toHaveCount(0);
  const tabs = page.getByTestId("work-pane").getByRole("tablist").getByRole("tab");
  await expect(tabs).toHaveCount(4);
  await expect(tab(page)).toHaveText(/Buddy\s*1/);
  await openBuddy(page);
  await expect(tab(page)).toHaveAttribute("aria-selected", "true");
  const overview = page.getByTestId("buddy-overview");
  const online = overview.getByRole("region", { name: "Online" });
  await expect(online).toContainText("yoyo");
  await expect(online).toContainText("Codex");
  await expect(overview.getByRole("region", { name: "Needs attention" })).toContainText("Gemini CLI");
  await expect(overview.getByRole("region", { name: "Available" })).toContainText("Claude Code");
  await expect(overview.getByText("Qwen Code")).toHaveCount(0);
  await overview.getByRole("button", { name: "See 1 more" }).click();
  await expect(overview.getByRole("region", { name: "More agents" })).toContainText("Qwen Code");
  // 对话不换页：Buddy 开着的时候中间还是主会话。
  await expect(page.getByTestId("conversation-column")).toBeVisible();
  await expect(page.getByTestId("composer-input")).toBeVisible();
  await page.getByTestId("work-tab-now").click();
  await expect(page.getByTestId("buddy-pane")).toHaveCount(0);
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

test("setup happens in place, sends the key once and never shows it again", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX, GEMINI] });
  await openBuddy(page);
  await page.getByTestId("buddy-row-gemini").getByRole("button", { name: "Set up", exact: true }).click();
  // 不弹对话框：翻到这一项的详情，要填的东西在最上面。
  await expect(page.getByRole("dialog")).toHaveCount(0);
  const detail = page.getByTestId("buddy-detail");
  const setup = detail.getByTestId("buddy-setup");
  await expect(setup.getByRole("heading", { name: "Set up Gemini CLI" })).toBeVisible();
  await setup.getByTestId("buddy-key-GEMINI_API_KEY").fill("sk-e2e-first-123");
  const first = await rpc(page, "buddy.configure", () => setup.getByRole("button", { name: "Save and check" }).click());
  expect(first.secrets).toEqual({ GEMINI_API_KEY: "sk-e2e-first-123" });
  await expect(page.getByTestId("buddy-detail-status")).toContainText("Online", { timeout: 10_000 });
  await expect(page.getByText("Gemini CLI is ready")).toBeVisible();
  await expect(setup).toHaveCount(0);

  const key = detail.getByTestId("buddy-key-GEMINI_API_KEY");
  await expect(key).toHaveValue("");
  await expect(detail.getByTestId("buddy-config").getByText("Saved", { exact: true })).toBeVisible();
  await key.fill("sk-e2e-second-456");
  const second = await rpc(page, "buddy.configure", () => detail.getByTestId("buddy-config").getByRole("button", { name: "Save", exact: true }).click());
  expect(second.secrets).toEqual({ GEMINI_API_KEY: "sk-e2e-second-456" });
  await expect(key).toHaveValue("");
  await expect(detail.getByRole("status").filter({ hasText: "Saved" })).toBeVisible();
  expect(await page.content()).not.toContain("sk-e2e-");
  await page.getByTestId("buddy-back").click();
  await expect(page.getByTestId("buddy-status-gemini")).toContainText("Online");
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

test("no external agent shows the welcome card with three picks", async ({ page }) => {
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
  await expect(tab(page)).toHaveText("Buddy");
  const params = await rpc(page, "buddy.install", () => overview.getByRole("button", { name: "Install Codex" }).click());
  expect(params.id).toBe("codex");
  await expect(overview.getByTestId("buddy-row-codex")).toBeVisible();
});

test("installing to the system asks first and shows the full command", async ({ page }) => {
  const command = "npm install -g @agentclientprotocol/claude-agent-acp";
  const claude = { ...CLAUDE, canManage: false, systemInstall: { available: true, command } };
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX, claude] });
  let installs = 0;
  page.on("request", (r) => {
    if (r.url().endsWith("/api/rpc") && r.postDataJSON()?.method === "buddy.install") installs++;
  });
  await openBuddy(page);
  await page.getByTestId("buddy-row-claude").getByRole("button", { name: "Install to system", exact: true }).click();
  const confirm = page.getByRole("alertdialog", { name: "Install Claude Code to the system?" });
  await expect(confirm.getByTestId("buddy-command")).toHaveText(command);
  // 人还没点运行，什么都没装。取消之后也一样。
  expect(installs).toBe(0);
  await confirm.getByRole("button", { name: "Cancel" }).click();
  await expect(confirm).toHaveCount(0);
  expect(installs).toBe(0);
  await page.getByTestId("buddy-row-claude").getByRole("button", { name: "Install to system", exact: true }).click();
  const params = await rpc(page, "buddy.install", () => confirm.getByRole("button", { name: "Run it" }).click());
  expect(params).toEqual({ id: "claude", mode: "system" });
  await expect(confirm).toHaveCount(0);
});

test("a custom agent is added on its own page and opens its detail", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX] });
  await openBuddy(page);
  await page.getByTestId("buddy-overview").getByRole("button", { name: "Add a custom agent" }).click();
  const form = page.getByTestId("buddy-custom");
  await expect(form.getByRole("heading", { name: "Add a custom agent" })).toBeVisible();
  await form.getByLabel("Name", { exact: true }).fill("My Agent");
  await form.getByLabel("Command", { exact: true }).fill("my-agent");
  await form.getByLabel("Arguments", { exact: true }).fill("--acp");
  const params = await rpc(page, "buddy.add", () => form.getByRole("button", { name: "Add", exact: true }).click());
  expect(params).toEqual({ name: "My Agent", command: "my-agent", args: ["--acp"], env: {} });
  await expect(page.getByTestId("buddy-detail").getByRole("heading", { name: "My Agent" })).toBeVisible();
  await expect(page.getByTestId("buddy-detail-status")).toContainText("Online", { timeout: 10_000 });
});

test("apps already on this computer get their own group: connect a CLI, or see what a desktop app still needs", async ({ page }) => {
  const command = "curl https://cursor.com/install -fsS | bash";
  const claude = buddyFixture("claude", {
    name: "Claude Code", order: 1, cliOnly: true,
    host: { kind: "cli", path: "C:/Users/me/AppData/Roaming/Claude/claude-code/2.1.10/claude.exe", app: "Claude" },
  });
  const cursor = buddyFixture("cursor", {
    name: "Cursor", order: 11, canManage: false, systemInstall: { available: true, command },
    host: { kind: "app", path: "C:/Users/me/AppData/Local/Programs/cursor/Cursor.exe", app: "Cursor" },
  });
  const opencode = buddyFixture("opencode", {
    name: "OpenCode", order: 12, install: { kind: "system", source: "path", path: "C:/Users/me/.opencode/bin/opencode.exe" }, state: "online",
    host: { kind: "acp", path: "C:/Users/me/.opencode/bin/opencode.exe" },
  });
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, claude, cursor, opencode] });
  await openBuddy(page);
  const overview = page.getByTestId("buddy-overview");
  const found = overview.getByRole("region", { name: "Found on this computer" });
  await expect(found.getByTestId("buddy-status-claude")).toHaveText("The Claude desktop app ships its command line. Only the adapter yoyo talks to is missing.");
  await expect(found.getByTestId("buddy-status-cursor")).toContainText("The Cursor desktop app is installed. It does not take tasks itself");
  // 厂商安装器放在 PATH 之外、能直接连的命令已经算装好了，在「在线」里。
  await expect(overview.getByRole("region", { name: "Online" })).toContainText("OpenCode");
  await expect(found).not.toContainText("OpenCode");

  // 只装了桌面应用：要装的是它的命令行，装之前照样先看完整命令。
  await found.getByTestId("buddy-row-cursor").getByRole("button", { name: "Install to system", exact: true }).click();
  const confirm = page.getByRole("alertdialog", { name: "Install Cursor to the system?" });
  await expect(confirm.getByTestId("buddy-command")).toHaveText(command);
  await confirm.getByRole("button", { name: "Cancel" }).click();

  // 详情里写明是在哪找到的。
  await found.getByTestId("buddy-row-claude").getByRole("button").first().click();
  const host = page.getByTestId("buddy-host");
  await expect(host).toContainText("Found inside the Claude desktop app");
  await expect(host).toContainText("claude-code/2.1.10/claude.exe");
  // 已经有命令行的，这一步叫「连接」：装的是 yoyo 托管的适配器。
  const params = await rpc(page, "buddy.install", () => page.getByTestId("buddy-detail").getByRole("button", { name: "Connect", exact: true }).click());
  expect(params).toEqual({ id: "claude", mode: "managed" });
});

test("an adapter can be told not to use the CLI already installed here", async ({ page }) => {
  const claude = buddyFixture("claude", {
    name: "Claude Code", order: 1, install: { kind: "managed", source: "npm", path: "C:/agents/claude" }, state: "online",
    config: { hostCli: { path: "C:/Users/me/.local/bin/claude.exe", on: true } },
  });
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, claude] });
  await openBuddy(page);
  await page.getByTestId("buddy-row-claude").getByRole("button").first().click();
  const config = page.getByTestId("buddy-config");
  const toggle = config.getByTestId("buddy-host-cli");
  await expect(toggle).toHaveAttribute("aria-checked", "true");
  await expect(config).toContainText("C:/Users/me/.local/bin/claude.exe");
  await toggle.click();
  const params = await rpc(page, "buddy.configure", () => config.getByRole("button", { name: "Save", exact: true }).click());
  expect(params).toEqual({ id: "claude", hostCli: false });
});

test("opening Buddy during the first scan shows placeholders and the conversation stays usable", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { buddies: [YOYO_BUDDY, CODEX] });
  let scanning = true;
  await page.route("**/api/rpc", async (route) => {
    const body = route.request().postDataJSON();
    if (body?.method !== "buddy.list" || !scanning) return route.fallback();
    await new Promise((done) => setTimeout(done, 800));
    return route.fulfill({ json: { jsonrpc: "2.0", id: body.id ?? 1, result: { buddies: [YOYO_BUDDY], detecting: true } } });
  });
  await page.goto("/");
  await tab(page).click();
  await expect(page.getByTestId("buddy-skeleton")).toBeVisible();
  await expect(page.getByTestId("conversation-column")).toBeVisible();
  await page.getByTestId("work-tab-now").click();
  await expect(page.getByTestId("buddy-skeleton")).toHaveCount(0);
  await tab(page).click();
  await expect(page.getByTestId("buddy-skeleton")).toBeVisible();
  scanning = false;
  await expect(page.getByTestId("buddy-overview").getByRole("region", { name: "Online" })).toContainText("Codex", { timeout: 10_000 });
  await expect(page.getByTestId("buddy-skeleton")).toHaveCount(0);
});
