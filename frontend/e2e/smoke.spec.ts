import { expect, test } from "@playwright/test";
import { mockApi } from "./mock";

test("send is disabled without a workspace", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: /Set up this workstation/i })).toBeVisible();
  await page.getByRole("button", { name: "Skip for now" }).click();
  await expect(page.getByRole("textbox", { name: "Message" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Send" })).toBeDisabled();
});

test("new chat is available after setup skip", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await page.getByRole("button", { name: "Skip for now" }).click();
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
});

test("command palette opens", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
  await page.keyboard.press("Control+k");
  await expect(page.getByPlaceholder("Jump to a thread or action…")).toBeVisible();
});

test("review sheet appears on a narrow viewport", async ({ page }) => {
  await page.setViewportSize({ width: 900, height: 800 });
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: /Set up this workstation/i })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
  await expect(page.getByRole("tablist", { name: "Review" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Toggle review" })).toHaveAttribute("aria-pressed", "true");
});
