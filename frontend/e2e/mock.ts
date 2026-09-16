import type { Page } from "@playwright/test";

export async function mockApi(
  page: Page,
  workspace = "",
  extra?: { sessions?: any[]; plugins?: any },
) {
  let sessions = [...(extra?.sessions || [])];
  await page.route("**/api/**", async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    const path = new URL(url).pathname;

    if (path.endsWith("/api/health")) {
      return route.fulfill({ json: { ok: true, harness: "deadbeef", model: "gpt-4.1", version: "0.1.0", isolated: false, workspace_ready: !!workspace } });
    }
    if (path.endsWith("/api/config") && method === "GET") {
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
    if (path.endsWith("/api/sessions") && method === "POST") {
      const t = { id: `s${sessions.length + 1}`, title: "New chat", workspace };
      sessions = [t, ...sessions];
      return route.fulfill({ json: t });
    }
    if (path.endsWith("/api/sessions")) {
      return route.fulfill({ json: sessions });
    }
    const del = path.match(/\/api\/sessions\/([^/]+)$/);
    if (del && method === "DELETE") {
      const id = decodeURIComponent(del[1]);
      sessions = sessions.filter((s) => String(s.id || s.ID) !== id);
      return route.fulfill({ json: { ok: true } });
    }
    if (path.endsWith("/api/plugins")) {
      return route.fulfill({ json: extra?.plugins ?? { fibers: [], mcp: [] } });
    }
    if (path.endsWith("/api/running")) {
      return route.fulfill({ json: { ids: [] } });
    }
    if (path.includes("/api/harness") || path.includes("/api/playbook") || path.includes("/api/archive")) {
      return route.fulfill({ json: {} });
    }
    if (path.includes("/api/skills") || path.includes("/api/fs/search") || path.includes("/api/logs") || path.includes("/api/doctor") || path.includes("/api/about") || path.includes("/api/key")) {
      return route.fulfill({ json: [] });
    }
    if (path.includes("/api/approvals") || path.includes("/api/context")) {
      return route.fulfill({ json: [] });
    }
    return route.fulfill({ status: 200, json: {} });
  });
}
