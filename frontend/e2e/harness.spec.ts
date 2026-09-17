import { expect, test } from "@playwright/test";
import { mockApi } from "./mock";

test("harness is a single rail entry with an overview", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    harness: { active: "aaa1111", refs: { active: "aaa1111", staging: "bbb2222" } },
  });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Harbor", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Evolve", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByTestId("harness-workspace")).toBeVisible();
  await expect(page.getByTestId("harness-overview")).toBeVisible();
  await expect(page.getByText("Staging differs from active")).toBeVisible();
  await expect(page.getByRole("tab", { name: "Overview" })).toHaveAttribute("aria-selected", "true");
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

test("prove tab has one primary eval button", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: /^Harness/ }).click();
  await page.getByRole("tab", { name: "Prove" }).click();
  await expect(page.getByRole("button", { name: "Run suite" })).toBeVisible();
  await expect(page.getByRole("button", { name: "More evals" })).toBeVisible();
  await expect(page.getByRole("button", { name: "TB subset" })).toHaveCount(0);
});

test("settings hides the thread rail", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await expect(page.getByRole("button", { name: "New chat" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Back" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Shortcuts" })).toBeVisible();
});
