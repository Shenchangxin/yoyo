import { expect, test, type Page } from "@playwright/test";
import { mockApi } from "./mock";

function island(page: Page) {
  return page.getByTestId("canvas-studio-shell");
}

async function openVideo(page: Page) {
  await page.getByRole("button", { name: "Video" }).click();
}

async function openVideoChat(page: Page) {
  await openVideo(page);
  await expect(island(page)).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(island(page)).toHaveCount(0);
}

test("infinite canvas starts as a conversation like short drama", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Video" })).toBeVisible();
  await openVideoChat(page);
  await expect(page.getByTestId("video-mode-switch")).toBeVisible();
  await page.getByTestId("video-mode-switch").click();
  await page.getByTestId("video-mode-canvas").click();
  await expect(page.getByTestId("canvas-project-chip")).toBeVisible();
  await expect(page.getByTestId("empty-turn")).toBeVisible();
  await expect(page.getByTestId("canvas-studio")).toHaveCount(0);
});

test("opening the canvas board mounts the canvas studio", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await openVideoChat(page);
  await page.getByTestId("video-mode-switch").click();
  await page.getByTestId("video-mode-canvas").click();
  await page.getByTestId("canvas-project-chip").click();
  await page.getByTestId("canvas-open-board").click();
  await expect(island(page)).toBeVisible();
  await expect(page.getByText("创作工作台")).toHaveCount(0);
});

test("drama workshop still opens from the same switch", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-drama").click();
  await expect(island(page)).toBeVisible();
  await expect(page.getByTestId("drama-agent-page")).toBeVisible({ timeout: 30_000 });
  await page.getByTestId("drama-open-board").first().click();
  await expect(page.getByTestId("drama-studio")).toBeVisible();
});

test("video rail hosts the canvas workspace shell instead of the inner create workbench", async ({ page }) => {
  test.setTimeout(60_000);
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await expect(page.getByTestId("video-shell-nav")).toBeVisible();
  await expect(page.getByTestId("video-shell-create")).toBeVisible();
  await expect(island(page)).toBeVisible({ timeout: 15_000 });
  await expect(island(page)).toHaveAttribute("data-pane", "create");
  await expect(page.getByText("创作工作台")).toHaveCount(0);
  await expect(page.getByText(/聊聊创作想法/)).toBeVisible({ timeout: 30_000 });
  const featuredCover = page.locator(".creation-featured-media img").first();
  await expect(featuredCover).toHaveAttribute("src", /\/short-drama-styles\//);
  await expect.poll(async () => featuredCover.evaluate((el) => (el as HTMLImageElement).naturalWidth)).toBeGreaterThan(40);
  await expect(page.getByRole("button", { name: "New chat" })).toHaveCount(0);

  await page.getByTestId("video-shell-drama").click();
  await expect(island(page)).toBeVisible();
  await expect(island(page)).toHaveAttribute("data-pane", "drama");
  await expect(page.getByTestId("drama-agent-page")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText(/聊聊这部短剧/)).toBeVisible();
  await expect(page.getByText("先贴一章原文")).toBeVisible();
  await expect(page.getByTestId("drama-agent-input")).toBeVisible();
  await expect(page.getByTestId("video-mode-switch")).toHaveCount(0);
  await expect(page.locator(".creation-mode-tabs")).toHaveCount(0);
  await expect(page.getByTestId("drama-studio")).toHaveCount(0);
  await expect(page.getByText("开始一部新短剧")).toHaveCount(0);
  await expect(page.getByText("创作工作台")).toHaveCount(0);

  await page.getByTestId("video-shell-canvas").click();
  await expect(island(page)).toBeVisible();
  await expect(page.getByText("创作工作台")).toHaveCount(0);
});

test("assets skills plugins and history open from the video rail", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await expect(page.getByTestId("video-shell-nav")).toBeVisible();
  await expect(page.getByTestId("video-shell-create")).toBeVisible();
  await expect(page.getByTestId("video-shell-assets")).toBeVisible();
  await expect(page.getByTestId("video-shell-skills")).toBeVisible();
  await expect(page.getByTestId("video-shell-plugins")).toBeVisible();
  await expect(page.getByTestId("video-shell-tasks")).toBeVisible();
  await page.getByTestId("video-shell-assets").click();
  const assetsCover = page.locator(".assets-empty-banner-frame img").first();
  await expect(assetsCover).toHaveAttribute("src", /\/short-drama-styles\//);
  await expect.poll(async () => assetsCover.evaluate((el) => (el as HTMLImageElement).naturalWidth)).toBeGreaterThan(40);

  for (const [testid, pane] of [
    ["video-shell-assets", "assets"],
    ["video-shell-skills", "skills"],
    ["video-shell-plugins", "plugins"],
    ["video-shell-tasks", "tasks"],
  ] as const) {
    await page.getByTestId(testid).click();
    await expect(island(page)).toBeVisible();
    await expect(island(page)).toHaveAttribute("data-pane", pane);
    await expect(page.getByText("创作工作台")).toHaveCount(0);
  }

  await page.keyboard.press("ControlOrMeta+k");
  await page.getByTestId("palette-video-assets").click();
  await expect(island(page)).toHaveAttribute("data-pane", "assets");

  await page.keyboard.press("ControlOrMeta+k");
  await page.getByRole("option", { name: "Open Skills" }).click();
  await expect(page.getByTestId("video-shell-nav")).toHaveCount(0);
});

test("new canvas appears in creation history and can be opened", async ({ page }) => {
  test.setTimeout(60_000);
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-tasks").click();
  await expect(page.getByTestId("video-history")).toBeVisible();
  await page.getByTestId("video-history-new-canvas").click();
  await expect(page.getByTestId("video-project-canvas-c1")).toBeVisible();
  await page.getByTestId("video-project-canvas-c1").click();
  await expect(island(page)).toHaveAttribute("data-pane", "canvas");
});
