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

test("slash plan toggles the plan chip", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  const box = page.getByRole("textbox", { name: "Message" });
  await expect(box).toBeEnabled({ timeout: 15_000 });
  await box.fill("/plan");
  await box.press("Enter");
  await expect(page.getByRole("button", { name: "Plan", exact: true })).toHaveAttribute("aria-pressed", "true");
});

test("control shortcuts section is reachable", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Control", exact: true }).click();
  await page.getByRole("button", { name: "Shortcuts" }).click();
  await expect(page.getByText("palette", { exact: true })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "palette" })).toHaveValue("Mod+K");
});

test("thread context menu can delete a chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: /Demo thread/ }).click({ button: "right" });
  await page.getByRole("button", { name: "Delete", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(page.getByRole("button", { name: /Demo thread/ })).toHaveCount(0);
});

test("control mcp lists running servers", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    plugins: { mcp: [{ name: "demo-mcp", command: "npx" }] },
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Control", exact: true }).click();
  await page.getByRole("button", { name: "MCP", exact: true }).click();
  await expect(page.getByText("demo-mcp")).toBeVisible();
  await expect(page.getByRole("button", { name: "Stop" })).toBeVisible();
});
