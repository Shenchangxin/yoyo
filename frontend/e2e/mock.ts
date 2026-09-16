import type { Page } from "@playwright/test";

export async function mockApi(page: Page, workspace = "") {
  await page.route("**/api/**", async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    if (url.includes("/api/health")) {
      return route.fulfill({ json: { ok: true, harness: "deadbeef", model: "gpt-4.1", version: "0.1.0", isolated: false } });
    }
    if (url.includes("/api/config") && method === "GET") {
      return route.fulfill({
        json: {
          provider: "openai",
          model: "gpt-4.1",
          workspace,
          auto_allow: false,
          max_budget_usd: 0,
          usd_per_mtok: 0,
          models: [],
        },
      });
    }
    if (url.includes("/api/sessions")) {
      return route.fulfill({ json: [] });
    }
    if (url.includes("/api/running")) {
      return route.fulfill({ json: { ids: [] } });
    }
    if (url.includes("/api/harness") || url.includes("/api/plugins") || url.includes("/api/playbook") || url.includes("/api/archive")) {
      return route.fulfill({ json: {} });
    }
    if (url.includes("/api/approvals") || url.includes("/api/context")) {
      return route.fulfill({ json: [] });
    }
    return route.fulfill({ status: 200, json: {} });
  });
}
