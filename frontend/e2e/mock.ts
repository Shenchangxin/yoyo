import type { Page } from "@playwright/test";

function sid(s: any) {
  return String(s.id || s.ID);
}

export async function mockApi(
  page: Page,
  workspace = "",
  extra?: { sessions?: any[]; plugins?: any; events?: any[]; running?: boolean; config?: Record<string, any>; approvals?: any[]; context?: any; artifacts?: any[]; spill?: Record<string, any>; trace?: any; harness?: any; files?: any[]; skills?: any[]; browser?: any; inbox?: any[]; queue?: any },
) {
  let sessions = [...(extra?.sessions || [])];
  let canvases: { id: string; title: string; session_id?: string }[] = [];
  let cfg: any = {
    provider: "openai",
    model: "gpt-4.1",
    base_url: "https://api.openai.com/v1",
    workspace,
    video_workspace: workspace ? `${String(workspace).replace(/\\/g, "/")}/video` : "C:/tmp/video",
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
    // Playwright's ** /api/ ** glob also matches Vite modules such as
    // /src/vendor/yingce-canvas/services/api/request.ts. Only stub the backend.
    if (!path.startsWith("/api/") && path !== "/api") {
      return route.continue();
    }
    const body = () => {
      try {
        return route.request().postDataJSON() || {};
      } catch {
        return {};
      }
    };

    if (path.endsWith("/api/rpc")) {
      const b = body();
      const rpcMethod = String(b.method || "");
      const params = b.params || {};
      const ok = (result: any) => route.fulfill({ json: { jsonrpc: "2.0", id: b.id ?? 1, result } });
      if (rpcMethod === "canvas.http") {
        const p = String(params.path || "").replace(/^\/api\/?/, "").replace(/^\//, "");
        const m = String(params.method || "GET").toUpperCase();
        if (p === "auth/session") {
          return ok({
            code: 0,
            data: {
              user: { id: "local", username: "yoyo", displayName: "Yoyo", role: "admin", status: "active" },
              features: { shortDramaEnabled: true, taskCenterEnabled: true, pluginCenterEnabled: true, creditsEnabled: false, customChannelsEnabled: true, frontendModelsEnabled: true },
              runtimeLimits: { activeTaskLimit: 8 },
            },
          });
        }
        if (p === "public/appearance") {
          return ok({
            code: 0,
            data: {
              appearance: {
                brandName: "Yoyo",
                brandSlug: "yoyo",
                seoTitle: "Yoyo",
                canvas: { agentName: "Yoyo" },
              },
            },
          });
        }
        if (p === "features") {
          return ok({ code: 0, data: { features: { shortDramaEnabled: true, pluginCenterEnabled: true, taskCenterEnabled: true } } });
        }
        if (p === "canvas-projects" && m === "GET") {
          return ok({ code: 0, data: { projects: canvases, total: canvases.length, page: 1, pageSize: 40, hasMore: false } });
        }
        if (p === "canvas-projects" && m === "POST") {
          const project = { id: `c${canvases.length + 1}`, title: params.body?.title || "Board", nodes: [], connections: [], revision: 1 };
          canvases = [...canvases, project];
          return ok({ code: 0, data: { project } });
        }
        if (p.startsWith("canvas-projects/") && (m === "GET" || m === "PUT")) {
          const id = p.split("/")[1] || "c1";
          const hit = canvases.find((c) => c.id === id);
          return ok({ code: 0, data: { project: hit || { id, title: "Board", nodes: [], connections: [], revision: 1 } } });
        }
        if (p === "model-catalog" || p === "channels/system") {
          return ok({ code: 0, data: { source: "system", channels: [], models: [] } });
        }
        if (p === "assets" || p.startsWith("assets/")) {
          return ok({ code: 0, data: { assets: [], kindCounts: {}, categoryCounts: {}, folderCounts: {}, page: 1, pageSize: 40, total: 0, hasMore: false } });
        }
        if (p === "asset-folders" || p.startsWith("asset-folders")) {
          return ok({ code: 0, data: { folders: [] } });
        }
        if (p === "skills" || p.startsWith("skills/")) {
          if (p === "skills/presets") return ok({ code: 0, data: { presets: [] } });
          if (p === "skills/added") return ok({ code: 0, data: { skills: [] } });
          return ok({ code: 0, data: { skills: [], totalCount: 0, total: 0, hasMore: false, categories: [] } });
        }
        if (p === "plugins" || p.startsWith("plugins")) {
          return ok({ code: 0, data: { plugins: [], states: {}, statuses: {} } });
        }
        if (p === "projects" || p.startsWith("projects/")) {
          if (p === "projects") {
            return ok({ code: 0, data: { projects: [], page: 1, pageSize: 50, total: 0, hasMore: false } });
          }
          return ok({ code: 0, data: { project: { id: "p1", name: "Board", status: "active" }, units: [], metrics: { unitCount: 0 }, canvasCounts: {} } });
        }
        if (p === "user-data/snapshot") {
          return ok({ code: 0, data: { assets: [], projects: [] } });
        }
        return ok({ code: 0, data: {} });
      }
      if (rpcMethod === "video.modes") {
        return ok([
          { id: "drama", title: "Short drama", ready: true },
          { id: "canvas", title: "Infinite canvas", ready: true },
          { id: "creative", title: "Creative", ready: false },
        ]);
      }
      if (rpcMethod === "video.status") {
        return ok({ ffmpeg: true, media_base: "" });
      }
      if (rpcMethod === "canvas.bind") {
        const cid = String(params.project_id || params.canvas_id || "");
        const sid = String(params.session_id || "");
        canvases = canvases.map((c) => (c.id === cid ? { ...c, session_id: sid } : c));
        return ok({ ok: true });
      }
      if (rpcMethod === "video.history") {
        return ok(canvases.map((c) => ({ kind: "canvas", id: c.id, title: c.title, session_id: c.session_id || "" })));
      }
      if (rpcMethod === "video.session.project") {
        const sid = String(params.session_id || "");
        const hit = canvases.find((c) => c.session_id === sid);
        return ok(hit ? { session_id: sid, kind: "canvas", id: hit.id, canvas_id: hit.id, title: hit.title } : { session_id: sid });
      }
      if (rpcMethod === "drama.list") {
        return ok([]);
      }
      if (rpcMethod === "video.styles") {
        return ok([]);
      }
      if (rpcMethod === "video.providers.list") {
        return ok([]);
      }
      return ok({});
    }

    if (path.endsWith("/api/health")) {
      return route.fulfill({ json: { ok: true, harness: "deadbeef", model: cfg.model, version: "0.1.0", isolated: false, workspace_ready: !!workspace, video_workspace: cfg.video_workspace, video_workspace_ready: true } });
    }
    if (path.endsWith("/api/config") && method === "GET") {
      return route.fulfill({ json: cfg });
    }
    if (path.endsWith("/api/config") && method === "POST") {
      cfg = { ...cfg, ...body() };
      return route.fulfill({ json: cfg });
    }
    if (path.endsWith("/api/sessions") && method === "POST") {
      const ch = String(body().channel || "agent") === "video" ? "video" : "agent";
      const t = { id: `s${sessions.length + 1}`, title: "New chat", workspace, channel: ch };
      sessions = [t, ...sessions];
      return route.fulfill({ json: t });
    }
    if (path.endsWith("/api/sessions")) {
      return route.fulfill({ json: sessions });
    }
    const spillOp = path.match(/\/api\/sessions\/([^/]+)\/spill\/([^/]+)$/);
    if (spillOp) {
      const blob = extra?.spill?.[spillOp[2]] ?? { id: spillOp[2], bytes: 11, text: "hello world", truncated: false };
      return route.fulfill({ json: blob });
    }
    const sessOp = path.match(/\/api\/sessions\/([^/]+)(?:\/([^/]+))?$/);
    if (sessOp) {
      const id = decodeURIComponent(sessOp[1]);
      const op = sessOp[2] || "";
      if (op === "trajectory") {
        return route.fulfill({ json: extra?.events || [] });
      }
      if (op === "trace") {
        return route.fulfill({ json: extra?.trace ?? mockTrace(id, extra) });
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
      if (method === "POST" && op === "model") {
        if (idx >= 0) sessions[idx] = { ...sessions[idx], model: String(body().model || "") };
        return route.fulfill({ json: sessions[idx] || { id, model: String(body().model || "") } });
      }
      if (method === "POST" && op === "auth") {
        const mode = String(body().mode || body().auth_mode || "default");
        if (idx >= 0) sessions[idx] = { ...sessions[idx], auth_mode: mode };
        return route.fulfill({ json: sessions[idx] || { id, auth_mode: mode } });
      }
      if (method === "POST" && op === "workspace") {
        const nextWs = String(body().workspace || workspace);
        if (idx >= 0) sessions[idx] = { ...sessions[idx], workspace: nextWs };
        return route.fulfill({ json: sessions[idx] || { id, workspace: nextWs } });
      }
    }
    if (path.endsWith("/api/plugins")) {
      return route.fulfill({ json: extra?.plugins ?? { fibers: [], mcp: [] } });
    }
    if (path.endsWith("/api/running")) {
      const ids = extra?.running && sessions[0] ? [sid(sessions[0])] : [];
      return route.fulfill({ json: { ids } });
    }
    if (path.endsWith("/api/harness")) {
      return route.fulfill({ json: extra?.harness ?? { active: "deadbeef", refs: { active: "deadbeef", staging: "" } } });
    }
    if (path.endsWith("/api/harness/reveal") || path.endsWith("/api/harness/lineage")) {
      return route.fulfill({ json: { ok: true, nodes: extra?.harness?.lineage || [] } });
    }
    if (path.endsWith("/api/playbook") || path.endsWith("/api/archive")) {
      return route.fulfill({ json: {} });
    }
    if (path.endsWith("/api/eval") || path.includes("/api/eval/")) {
      return route.fulfill({ json: { metrics: { held_in_pass: 1, held_in_total: 1, held_out_pass: 1, held_out_total: 1, safety_fail: 0 }, results: [] } });
    }
    if (path.includes("/api/fs/search") || path.includes("/api/workspace/tree")) {
      return route.fulfill({
        json: extra?.files ?? [
          { path: "src/main.go", kind: "file" },
          { path: "README.md", kind: "file" },
          { path: "src", kind: "dir" },
        ],
      });
    }
    if (path.endsWith("/api/logs") && method === "POST") {
      return route.fulfill({ json: { ok: true } });
    }
    if (path.endsWith("/api/logs")) {
      return route.fulfill({
        json: {
          dir: "C:/tmp/yoyo/logs",
          path: "C:/tmp/yoyo/logs/yoyo.log",
          process: "gui",
          text: '{"ts":"2026-09-21T00:00:00Z","level":"info","msg":"boot","component":"boot"}',
          lines: [{ ts: "2026-09-21T00:00:00Z", level: "info", msg: "boot", component: "boot", raw: '{"msg":"boot","component":"boot"}' }],
        },
      });
    }
    if (path.endsWith("/api/journal")) {
      return route.fulfill({ json: { journal: [{ Time: "2026-09-21T00:00:00Z", Type: "checkout", Payload: "hash" }], path: "C:/tmp/yoyo/journal/chain.jsonl" } });
    }
    if (path.endsWith("/api/logs/export") || path.endsWith("/api/logs/diagnose")) {
      return route.fulfill({ json: { path: "C:/tmp/support.zip", text: "redacted tail" } });
    }
    if (path.endsWith("/api/skills")) {
      return route.fulfill({ json: extra?.skills ?? [] });
    }
    if (path.includes("/api/skills") || path.includes("/api/doctor") || path.includes("/api/about") || path.includes("/api/key")) {
      return route.fulfill({ json: [] });
    }
    if (path.includes("/api/workspace/file")) {
      const u = new URL(route.request().url());
      const p = u.searchParams.get("path") || "";
      const html = /\.html?$/i.test(p);
      const office = /\.(docx|xlsx|pptx|pdf)$/i.test(p);
      const go = /\.go$/i.test(p);
      return route.fulfill({
        json: {
          path: p,
          text: office ? "" : html ? "<!doctype html><html><body><h1>preview</h1></body></html>" : go ? "package main\n\nfunc Hello() string {\n\treturn \"yoyo\"\n}\n" : "package main\n",
          lang: html ? "html" : go ? "go" : "text",
          html,
          binary: office,
          truncated: false,
          bytes: 32,
        },
      });
    }
    if (path.includes("/api/workspace/hunks")) {
      return route.fulfill({ json: { diff: "", hunks: [] } });
    }
    if (path.includes("/api/approvals")) {
      return route.fulfill({ json: extra?.approvals ?? [] });
    }
    if (path.includes("/api/context")) {
      return route.fulfill({ json: extra?.context ?? { tokens: 0, budget: 0 } });
    }
    if (path.endsWith("/api/browser/view")) {
      return route.fulfill({ json: extra?.browser ?? { live: false, lane: "isolated", log: [] } });
    }
    if (path.endsWith("/api/browser/takeover")) {
      return route.fulfill({ json: { ok: true } });
    }
    if (path.endsWith("/api/inbox")) {
      return route.fulfill({ json: extra?.inbox ?? [] });
    }
    if (path.endsWith("/api/review/queue")) {
      return route.fulfill({ json: extra?.queue ?? { offers: [], inbox: [], drafts: [], browser: [] } });
    }
    return route.fulfill({ status: 200, json: {} });
  });
}

function mockTrace(id: string, extra?: { sessions?: any[]; events?: any[]; artifacts?: any[]; context?: any }) {
  const events = extra?.events || [];
  const projected: any[] = [];
  let tools = 0;
  let users = 0;
  let errors = 0;
  let deltas = 0;
  for (let i = 0; i < events.length; i++) {
    const ev = events[i] || {};
    const p = ev.payload || {};
    if (p.delta) {
      deltas++;
      continue;
    }
    if (ev.type === "user") users++;
    if (ev.type === "tool_call") tools++;
    if (ev.type === "error") errors++;
    const text = String(p.text || p.content || p.arguments || p.note || "");
    projected.push({
      index: i,
      ts: ev.ts || "",
      type: ev.type || "",
      source: ev.source || "",
      lane: ev.type === "tool_call" || ev.type === "tool_result" || ev.type === "approval" ? "exec" : ev.type === "error" ? "error" : ev.type === "compaction" ? "memory" : "dialog",
      round: p.round || "",
      name: p.name || "",
      id: p.id || "",
      summary: text.slice(0, 120) || ev.type,
      detail: text.slice(0, 4000),
      bytes: p.bytes || 0,
      elapsed_ms: p.elapsed_ms || 0,
      spill_id: p.spill_id || "",
      tokens: p.tokens || 0,
      error: ev.type === "error",
    });
  }
  return {
    session_id: id,
    title: extra?.sessions?.[0]?.title || id,
    stats: {
      events: events.length,
      users,
      tool_calls: tools,
      errors,
      deltas,
      tokens: extra?.context?.tokens || 0,
      duration_ms: 0,
      spill_bytes: extra?.artifacts?.reduce((n: number, a: any) => n + (a.bytes || 0), 0) || 0,
    },
    events: projected,
    artifacts: extra?.artifacts || [],
  };
}
