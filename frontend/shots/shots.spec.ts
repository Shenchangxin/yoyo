/**
 * 拥有：把会话表面的几屏渲染出来存成图：今天、任务、记忆、Buddy，以及宽窗口下的主会话。
 * 不拥有：断言。图给人看，改界面前后各出一套对照。数据来自 fixture.ts。
 *
 * SHOTS_TAG 是文件名前缀（before、after……），SHOTS_OUT 是输出目录，默认 shots/out（不进版本库）。
 */

import fs from "node:fs";
import path from "node:path";
import { test, type Page } from "@playwright/test";
import { mockDay } from "./fixture";

// 默认存到 shots/out。不放在 test-results 里：Playwright 每次开跑都会清空那个目录，上一套图就没了。
const out = path.resolve(process.env.SHOTS_OUT || "shots/out");
const tag = process.env.SHOTS_TAG || "shot";

async function settle(page: Page) {
  await page.waitForTimeout(1200);
}

async function shoot(page: Page, name: string) {
  fs.mkdirSync(out, { recursive: true });
  await page.screenshot({ path: path.join(out, `${tag}-${name}.png`) });
}

async function open(page: Page, theme: "dark" | "light" = "dark") {
  await page.addInitScript((mode) => {
    try {
      localStorage.setItem("yoyo-theme", mode);
      localStorage.setItem("yoyo-locale", "zh-CN");
    } catch {
      /* 存不进去时用默认主题 */
    }
  }, theme);
  await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
  await page.goto("/");
  await page.getByTestId("work-pane").waitFor({ timeout: 30_000 });
  await settle(page);
}

test("today", async ({ page }) => {
  await mockDay(page);
  await open(page);
  await shoot(page, "today-dark");
});

test("tasks", async ({ page }) => {
  await mockDay(page);
  await open(page);
  await page.getByTestId("work-tab-tasks").click();
  await settle(page);
  await shoot(page, "tasks-dark");
});

test("memory", async ({ page }) => {
  await mockDay(page);
  await open(page);
  await page.getByTestId("work-tab-memory").click();
  await settle(page);
  await shoot(page, "memory-dark");
});

test("light", async ({ page }) => {
  await mockDay(page);
  await open(page, "light");
  await shoot(page, "today-light");
});

test("wide", async ({ page }) => {
  await page.setViewportSize({ width: 1920, height: 1080 });
  await mockDay(page);
  await open(page);
  await shoot(page, "today-wide-dark");
});

test("composer", async ({ page }) => {
  await mockDay(page);
  await open(page);
  const column = page.getByTestId("composer-column");
  fs.mkdirSync(out, { recursive: true });
  await column.screenshot({ path: path.join(out, `${tag}-composer-dark.png`) });
});

/** openBuddy 翻到 Buddy：现在是今天栏里的页签；改造前是底部坞上的一个图标。 */
async function openBuddy(page: Page) {
  const tab = page.getByTestId("work-tab-buddy");
  if (await tab.count()) {
    await tab.click();
  } else {
    await page.locator(".rail-dock").getByRole("button", { name: /Buddy/ }).first().click();
  }
  await settle(page);
}

test("buddy", async ({ page }) => {
  await mockDay(page);
  await open(page);
  await openBuddy(page);
  await shoot(page, "buddy-dark");
});

test("buddy pages", async ({ page }) => {
  await mockDay(page);
  await open(page);
  await openBuddy(page);
  if (!(await page.getByTestId("buddy-pane").count())) return;
  const pane = page.getByTestId("work-pane");
  // 在这台电脑上发现的：桌面应用里带着命令行。
  await page.getByTestId("buddy-row-claude").getByRole("button").first().click();
  await settle(page);
  await shoot(page, "buddy-found-dark");
  await page.getByTestId("buddy-back").click();
  // 只装了桌面应用：装到系统前先把完整命令给人看。
  await page.getByTestId("buddy-row-cursor").getByRole("button", { name: "装到系统", exact: true }).click();
  await settle(page);
  await shoot(page, "buddy-confirm-dark");
  await page.getByTestId("buddy-confirm").getByRole("button", { name: "取消" }).click();
  // 需要配置：要填的东西在详情最上面。
  await page.getByTestId("buddy-row-qwen").getByRole("button", { name: "配置", exact: true }).click();
  await settle(page);
  await shoot(page, "buddy-setup-dark");
  await page.getByTestId("buddy-back").click();
  // 在线的：派任务的设置。
  await page.getByTestId("buddy-row-opencode").getByRole("button").first().click();
  await settle(page);
  await pane.getByTestId("buddy-config").scrollIntoViewIfNeeded();
  await shoot(page, "buddy-online-dark");
  await page.getByTestId("buddy-back").click();
  await page.getByRole("button", { name: "偏好" }).click();
  await settle(page);
  await shoot(page, "buddy-prefs-dark");
});

test("buddy light", async ({ page }) => {
  await mockDay(page);
  await open(page, "light");
  await openBuddy(page);
  await shoot(page, "buddy-light");
});
