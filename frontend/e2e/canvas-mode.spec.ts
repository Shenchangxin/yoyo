import { expect, test } from "@playwright/test";
import { mockApi } from "./mock";

test("infinite canvas occupies the video stage", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Video" })).toBeVisible();
  await page.getByRole("button", { name: "Video" }).click();
  await expect(page.getByTestId("video-mode-switch")).toBeVisible();
  await expect(page.getByTestId("video-mode-switch")).toContainText(/Short drama|drama/i);
  await page.getByTestId("video-mode-switch").click();
  await page.getByTestId("video-mode-canvas").click();
  await expect(page.getByTestId("canvas-studio").or(page.getByTestId("canvas-studio-loading"))).toBeVisible({ timeout: 30_000 });
});

test("drama workshop still opens from the same switch", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await expect(page.getByTestId("video-mode-switch")).toBeVisible();
  await expect(page.getByTestId("video-mode-switch")).toContainText(/Short drama|drama/i);
  await expect(page.getByTestId("drama-project-chip")).toBeVisible();
  await page.getByTestId("drama-project-chip").click();
  await expect(page.getByRole("menuitem").filter({ hasText: /New series/i })).toBeVisible();
});
