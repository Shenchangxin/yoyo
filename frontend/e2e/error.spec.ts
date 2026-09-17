import { expect, test } from "@playwright/test";
import { bannerError } from "../src/lib/error";
import { errMessage } from "../src/lib/normalize";
import { en } from "../src/lib/copy";
import { mockApi } from "./mock";

test("banner keeps the backend reason next to the generic title", () => {
  const t = en.transcript;
  expect(bannerError("Failed to fetch", t)).toContain("Failed to fetch");
  expect(bannerError("/api/health: vault: missing key \"default\"", t)).toContain("vault: missing key");
  expect(bannerError("Something went wrong", t)).toBe("Something went wrong");
});

test("errMessage unwraps nested causes", () => {
  const inner = new Error("vault: missing key");
  expect(errMessage(new Error("health", { cause: inner }))).toContain("vault: missing key");
});

test("boot failure shows the backend reason in the top banner", async ({ page }) => {
  await page.route("**/api/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path.endsWith("/api/health")) {
      return route.fulfill({ status: 500, json: { error: "vault: missing key \"default\"" } });
    }
    return route.fulfill({ json: {} });
  });
  await page.goto("/");
  await expect(page.getByTestId("app-error")).toContainText("vault: missing key");
  await expect(page.getByTestId("app-error").getByRole("button", { name: "Retry" })).toBeVisible();
});

test("successful boot does not show the unknown-error banner", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
  await expect(page.getByText("Something went wrong")).toHaveCount(0);
});
