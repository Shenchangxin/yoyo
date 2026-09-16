import type { Page } from "@playwright/test";

function sid(s: any) {
  return String(s.id || s.ID);
}

export async function mockApi(
  page: Page,
  workspace = "",
  extra?: { sessions?: any[]; plugins?: any; events?: any[]; running?: boolean; config?: Record<string, any>; approvals?: any[]; context?: any },
) {
  let sessions = [...(extra?.sessions || [])];
  let cfg: any = {
    provider: "openai",
    model: "gpt-4.1",
    base_url: "https://api.openai.com/v1",
    workspace,
    auto_allow: false,
    max_budget_usd: 0,
    usd_per_mtok: 0,
    models: [],
    ...(extra?.config || {}),
  };
  if (extra?.config?.workspace === undefined) cfg.workspace = workspace;
  await page.route("**/api/**", async (route) => {
    const method = route.request().method();
    const path = new URL(route.request().url()).pathname;
    const body = () => {
      try {
        return route.request().postDataJSON() || {};
      } catch {
        return {};
      }
    };

    if (path.endsWith("/api/health")) {
      return route.fulfill({ json: { ok: true, harness: "deadbeef", model: cfg.model, version: "0.1.0", isolated: false, workspace_ready: !!workspace } });
    }
    if (path.endsWith("/api/config") && method === "GET") {
      return route.fulfill({ json: cfg });
    }
    if (path.endsWith("/api/config") && method === "POST") {
      cfg = { ...cfg, ...body() };
      return route.fulfill({ json: cfg });
    }
    if (path.endsWith("/api/sessions") && method === "POST") {
      const t = { id: `s${sessions.length + 1}`, title: "New chat", workspace };
      sessions = [t, ...sessions];
      return route.fulfill({ json: t });
    }
    if (path.endsWith("/api/sessions")) {
      return route.fulfill({ json: sessions });
    }
    const sessOp = path.match(/\/api\/sessions\/([^/]+)(?:\/([^/]+))?$/);
    if (sessOp) {
      const id = decodeURIComponent(sessOp[1]);
      const op = sessOp[2] || "";
      if (op === "trajectory") {
        return route.fulfill({ json: extra?.events || [] });
      }
      if (op === "running") {
        return route.fulfill({ json: { running: !!extra?.running } });
      }
      if (op === "retry" && method === "POST") {
        return route.fulfill({ json: { ok: true } });
      }
      if (op === "messages" && method === "POST") {
        return route.fulfill({ json: { ok: true, async: true } });
      }
      if (op === "events") {
        return route.fulfill({ status: 200, body: "", contentType: "text/event-stream" });
      }
      if (op === "queue") {
        return route.fulfill({ json: [] });
      }
      if (op === "context") {
        return route.fulfill({
          json: extra?.context ?? { tokens: 0, budget: 0, window: 0, prefix_tokens: 0, dynamic_tokens: 0, schema_tokens: 0 },
        });
      }
      const idx = sessions.findIndex((s) => sid(s) === id);
      if (method === "DELETE" && !op) {
        sessions = sessions.filter((s) => sid(s) !== id);
        return route.fulfill({ json: { ok: true } });
      }
      if (method === "POST" && op === "pin") {
        if (idx >= 0) sessions[idx] = { ...sessions[idx], pinned: !!body().pinned };
        return route.fulfill({ json: sessions[idx] || { id, pinned: !!body().pinned } });
      }
      if (method === "POST" && op === "archive") {
        if (idx >= 0) sessions[idx] = { ...sessions[idx], archived: !!body().archived };
        return route.fulfill({ json: sessions[idx] || { id, archived: !!body().archived } });
      }
      if (method === "POST" && op === "title") {
        if (idx >= 0) sessions[idx] = { ...sessions[idx], title: String(body().title || sessions[idx].title) };
        return route.fulfill({ json: { ok: true } });
      }
      if (method === "POST" && op === "fork") {
        const src = idx >= 0 ? sessions[idx] : { id, title: "thread", workspace };
        const t = { ...src, id: `fork-${id}`, title: `fork of ${src.title || id}` };
        sessions = [t, ...sessions];
        return route.fulfill({ json: t });
      }
    }
    if (path.endsWith("/api/plugins")) {
      return route.fulfill({ json: extra?.plugins ?? { fibers: [], mcp: [] } });
    }
    if (path.endsWith("/api/running")) {
      const ids = extra?.running && sessions[0] ? [sid(sessions[0])] : [];
      return route.fulfill({ json: { ids } });
    }
    if (path.endsWith("/api/harness") || path.endsWith("/api/playbook") || path.endsWith("/api/archive")) {
      return route.fulfill({ json: {} });
    }
    if (path.includes("/api/skills") || path.includes("/api/fs/search") || path.includes("/api/logs") || path.includes("/api/doctor") || path.includes("/api/about") || path.includes("/api/key")) {
      return route.fulfill({ json: [] });
    }
    if (path.includes("/api/approvals")) {
      return route.fulfill({ json: extra?.approvals ?? [] });
    }
    if (path.includes("/api/context")) {
      return route.fulfill({ json: extra?.context ?? { tokens: 0, budget: 0 } });
    }
    return route.fulfill({ status: 200, json: {} });
  });
}
