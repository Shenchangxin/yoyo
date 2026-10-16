import type { Page } from "@playwright/test";

/** Minimal valid PDF so Chromium's viewer can load in the inspector iframe. */
const MINI_PDF_B64 = "JVBERi0xLjEKMSAwIG9iago8PC9UeXBlL0NhdGFsb2cvUGFnZXMgMiAwIFI+PmVuZG9iagoyIDAgb2JqCjw8L1R5cGUvUGFnZXMvS2lkc1szIDAgUl0vQ291bnQgMT4+ZW5kb2JqCjMgMCBvYmoKPDwvVHlwZS9QYWdlL01lZGlhQm94WzAgMCAzIDNdPj5lbmRvYmoKeHJlZgowIDQKMDAwMDAwMDAwMCA2NTUzNSBmIAowMDAwMDAwMDA5IDAwMDAwIG4gCjAwMDAwMDAwNTIgMDAwMDAgbiAKMDAwMDAwMDEwMSAwMDAwMCBuIAp0cmFpbGVyCjw8L1NpemUgNC9Sb290IDEgMCBSPj4Kc3RhcnR4cmVmCjE0NwolJUVPRg==";

function sid(s: any) {
  return String(s.id || s.ID);
}

export async function mockApi(
  page: Page,
  workspace = "",
  extra?: { sessions?: any[]; studios?: any[]; plugins?: any; events?: any[]; trajectories?: Record<string, any[]>; running?: boolean; config?: Record<string, any>; approvals?: any[]; context?: any; artifacts?: any[]; spill?: Record<string, any>; trace?: any; harness?: any; files?: any[]; skills?: any[]; packs?: any[]; browser?: any; inbox?: any[]; queue?: any; update?: any },
) {
  let sessions = [...(extra?.sessions || [])];
  let canvases: { id: string; title: string; session_id?: string }[] = [];
  let studios: any[] = [...(extra?.studios || [])];
  let topics: any[] = [
    { id: "topic-1", title: "春日穿搭", status: "ready", source_platform: "weibo", angle: "通勤怎么穿", scores: { heat: 5, feasibility: 4 } },
    { id: "topic-2", title: "还没想好", status: "inbox" },
  ];
  let pages: any[] = [];
  let paused = false;
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
          { id: "creative", title: "Shorts", ready: true },
        ]);
      }
      if (rpcMethod === "runtime.pwsh.status" || rpcMethod === "runtime.pwsh.install") {
        return ok({
          ok: true,
          needed: false,
          ready: true,
          phase: "ready",
          percent: 100,
          version: "7.6.7",
          shell: "posix",
        });
      }
      if (rpcMethod === "video.status") {
        return ok({
          ffmpeg: true,
          media_base: "",
          ffmpeg_install: { ready: true, phase: "ready", percent: 100 },
          remotion: { ready: false, bundled: false, node: "/usr/bin/node", install: { phase: "idle", percent: 0 } },
          whisper: false,
          whisper_bin: "",
        });
      }
      if (rpcMethod === "video.templates") {
        return ok([
          { service_type: "image", provider: "aliyun", name: "Qwen Image", base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1", model: "qwen-image-plus", models: "[]" },
          { service_type: "image", provider: "jimeng", name: "Jimeng Image", base_url: "https://visual.volcengineapi.com", model: "jimeng_t2i_v40", models: "[]" },
          { service_type: "video", provider: "vidu", name: "Vidu", base_url: "https://api.vidu.cn", model: "viduq2", models: "[]" },
          { service_type: "video", provider: "jimeng", name: "Jimeng Video", base_url: "https://visual.volcengineapi.com", model: "jimeng_vgfm_t2v_l20", models: "[]" },
          { service_type: "music", provider: "minimax", name: "MiniMax Music", base_url: "https://api.minimax.io", model: "music-2.0", models: '["music-2.0","music-1.5"]' },
          { service_type: "music", provider: "mureka", name: "Mureka", base_url: "https://api.mureka.ai", model: "mureka-6", models: "[]" },
          { service_type: "music", provider: "elevenlabs", name: "ElevenLabs Sound", base_url: "https://api.elevenlabs.io/v1", model: "eleven_text_to_sound_v2", models: "[]" },
          { service_type: "music", provider: "stability", name: "Stability Audio", base_url: "https://api.stability.ai", model: "stable-audio-2.5", models: "[]" },
          { service_type: "stock", provider: "unsplash", name: "Unsplash", base_url: "https://api.unsplash.com", model: "", models: "[]" },
          { service_type: "stock", provider: "pexels", name: "Pexels Photos", base_url: "https://api.pexels.com/v1", model: "", models: "[]" },
          { service_type: "stock", provider: "pexels", name: "Pexels Videos", base_url: "https://api.pexels.com/videos", model: "", models: "[]" },
          { service_type: "asr", provider: "groq", name: "Groq Whisper", base_url: "https://api.groq.com/openai/v1", model: "whisper-large-v3-turbo", models: "[]" },
          { service_type: "asr", provider: "deepgram", name: "Deepgram", base_url: "https://api.deepgram.com", model: "nova-3", models: "[]" },
        ]);
      }
      if (rpcMethod === "video.ffmpeg.install") {
        return ok({ ok: true, ffmpeg: true, ffmpeg_install: { ready: true, phase: "ready", percent: 100 } });
      }
      if (rpcMethod === "canvas.bind") {
        const cid = String(params.project_id || params.canvas_id || "");
        const sid = String(params.session_id || "");
        canvases = canvases.map((c) => (c.id === cid ? { ...c, session_id: sid } : c));
        return ok({ ok: true });
      }
      if (rpcMethod === "video.history") {
        return ok([
          ...studios.map((s) => ({
            kind: "studio",
            id: s.id,
            title: s.title,
            session_id: s.session_id || "",
            status: s.status || "draft",
            gate: s.gate || "script",
          })),
          ...canvases.map((c) => ({ kind: "canvas", id: c.id, title: c.title, session_id: c.session_id || "" })),
        ]);
      }
      if (rpcMethod === "video.settings.get") {
        return ok({ content_language: "zh", studio_aspect: "9:16", studio_publish_platform: "douyin" });
      }
      if (rpcMethod === "video.settings.set") {
        return ok({ ok: true });
      }
      if (rpcMethod === "video.trends.fetch") {
        return ok({ trends: [], updated: 0 });
      }
      if (rpcMethod === "video.topic.list") {
        const status = String(params.status || "");
        return ok(status ? topics.filter((t) => t.status === status) : topics);
      }
      if (rpcMethod === "video.topic.save") {
        const id = String(params.id || `topic-${topics.length + 1}`);
        const row = { ...(topics.find((t) => t.id === id) || {}), ...params, id };
        topics = [row, ...topics.filter((t) => t.id !== id)];
        return ok(row);
      }
      if (rpcMethod === "video.topic.schedule") {
        const id = String(params.id || "");
        const hit = topics.find((t) => t.id === id);
        if (!hit) return ok({ ok: false });
        if (hit.status === "scheduled") {
          return ok({ ok: true, duplicate: true, job: { id: "job-sched" }, topic: hit });
        }
        hit.status = "scheduled";
        return ok({ ok: true, job: { id: "job-sched", kind: "once" }, topic: hit });
      }
      if (rpcMethod === "video.topic.unschedule") {
        const id = String(params.id || "");
        const hit = topics.find((t) => t.id === id);
        if (hit && hit.status === "scheduled") hit.status = hit.angle ? "ready" : "inbox";
        return ok({ ok: true, cancelled: 1, topic: hit });
      }
      if (rpcMethod === "video.cut.open" || rpcMethod === "video.cut.create") {
        return ok({
          id: "cut-1",
          title: String(params.title || "Cut"),
          source_hash: String(params.hash || "cas-assembled"),
          session_id: String(params.session_id || ""),
        });
      }
      if (rpcMethod === "video.cut.bind") {
        return ok({ ok: true });
      }
      if (rpcMethod === "video.studio.list") {
        return ok(studios);
      }
      if (rpcMethod === "video.studio.delete") {
        const id = String(params.id || "");
        studios = studios.filter((s) => s.id !== id);
        return ok({ ok: true });
      }
      if (rpcMethod === "video.studio.create") {
        const sid = String(params.session_id || "");
        const topicId = String(params.topic_id || "");
        const fromTopic = topics.find((t) => t.id === topicId);
        const row = {
          ok: true,
          id: `studio-${studios.length + 1}`,
          title: String(params.title || fromTopic?.title || "Short"),
          status: "draft",
          gate: "script",
          furthest_gate: "script",
          script_status: "needs_review",
          storyboard: { shots: [] },
          delivery: { status: "draft" },
          session_id: sid,
          topic_id: String(params.topic_id || ""),
        };
        if (sid) {
          studios = studios.map((s) => (s.session_id === sid ? { ...s, session_id: "" } : s));
        }
        studios = [row, ...studios];
        if (row.topic_id) {
          topics = topics.map((t) => (t.id === row.topic_id ? { ...t, status: "in_production", studio_project_id: row.id } : t));
        }
        return ok(row);
      }
      if (rpcMethod === "video.studio.bind") {
        const id = String(params.project_id || params.id || studios[0]?.id || "studio-1");
        const sid = String(params.session_id || "");
        studios = studios.map((s) => {
          if (s.id === id) return { ...s, session_id: sid };
          if (sid && s.session_id === sid) return { ...s, session_id: "" };
          return s;
        });
        const hit = studios.find((s) => s.id === id) || studios[0] || { id, title: "Short", status: "draft", gate: "script", furthest_gate: "script", script_status: "needs_review", storyboard: { shots: [] }, delivery: { status: "draft" }, session_id: sid };
        return ok(hit);
      }
      if (rpcMethod === "video.studio.get" || rpcMethod === "video.studio.update") {
        const id = String(params.id || studios[0]?.id || "studio-1");
        const hit = studios.find((s) => s.id === id);
        return ok(hit || { id, title: "Short", status: "draft", gate: "script", furthest_gate: "script", script_status: "needs_review", storyboard: { shots: [] }, delivery: { status: "draft" } });
      }
      if (rpcMethod === "video.studio.catalog") {
        return ok({ characters: [], styles: [], presets: [], templates: [], voices: [], platforms: ["douyin", "xhs", "bilibili"], tones: ["natural"] });
      }
      if (rpcMethod === "video.studio.preflight") {
        return ok({ gate: params.gate || "script", ok: true, furthest: "script", spends: [] });
      }
      if (rpcMethod === "video.topic.expand") {
        return ok({ id: params.id, title: "Topic", options: [{ id: "opt-1", title: "Angle A", angle: "clear facts", summary: "no fabricated experience" }] });
      }
      if (rpcMethod === "video.topic.choose") {
        return ok({ id: params.id, title: "Topic", angle: "clear facts" });
      }
      if (rpcMethod === "video.studio.publish.platforms") {
        return ok([{ id: "douyin", label: "抖音", creator_url: "https://example.com", analytics_url: "https://example.com" }]);
      }
      if (rpcMethod === "video.studio.publish.list") {
        return ok([]);
      }
      if (rpcMethod.startsWith("video.studio.publish.")) {
        return ok({ ok: true, id: "pub-mock", status: "draft", title: "Short" });
      }
      if (rpcMethod === "video.studio.remotion.status") {
        return ok({ ready: false, bundled: false, hint: "optional", node: "/usr/bin/node", install: { phase: "idle", percent: 0 } });
      }
      if (rpcMethod === "video.studio.remotion.install") {
        return ok({ ready: true, bundled: false, hint: "ready", node: "/usr/bin/node", source: "managed", install: { ready: true, phase: "ready", percent: 100 } });
      }
      if (rpcMethod === "video.studio.persona.template") {
        return ok({ markdown: "# persona" });
      }
      if (rpcMethod === "video.studio.persona.create") {
        return ok({ id: "page-mock", title: "persona", persona_page_id: "page-mock" });
      }
      if (rpcMethod === "video.studio.ops.wrap") {
        return ok({ cas_hash: "wrap", media_url: "" });
      }
      if (rpcMethod === "video.studio.media.put") {
        return ok({ cas_hash: "upload-mock", media_url: "", kind: "image" });
      }
      if (rpcMethod === "video.session.project") {
        const sid = String(params.session_id || "");
        const studio = studios.find((s) => s.session_id === sid);
        if (studio) {
          return ok({
            session_id: sid,
            kind: "studio",
            id: studio.id,
            studio_id: studio.id,
            title: studio.title,
            gate: studio.gate,
            status: studio.status,
          });
        }
        const hit = canvases.find((c) => c.session_id === sid);
        return ok(hit ? { session_id: sid, kind: "canvas", id: hit.id, canvas_id: hit.id, title: hit.title } : { session_id: sid });
      }
      if (rpcMethod === "drama.list") {
        return ok([]);
      }
      if (rpcMethod === "drama.episodes") {
        return ok([]);
      }
      if (rpcMethod === "drama.bundle") {
        return ok(null);
      }
      if (rpcMethod === "drama.episode.get") {
        return ok(null);
      }
      if (rpcMethod === "video.styles") {
        return ok([]);
      }
      if (rpcMethod === "video.providers.list") {
        return ok([]);
      }
      if (rpcMethod === "skins.list") {
        return ok([
          { id: "builtin.ink", name: "Ink", builtin: true, preserveEvidence: true },
          { id: "builtin.neutral", name: "Neutral", builtin: true, preserveEvidence: true },
        ]);
      }
      if (rpcMethod === "skins.resolve") {
        return ok({
          id: params.id || "builtin.ink",
          name: "Ink",
          builtin: true,
          mode: params.mode || "dark",
          preserveEvidence: true,
          tokens: {},
          windowRgb: [28, 28, 26],
          materials: { grain: 0.04, wallpaperFit: "cover", wallpaperDim: 0.45, radiusScale: 1 },
          contrast: { foreground: 12, muted: 5, pass: true },
        });
      }
      if (rpcMethod.startsWith("skins.")) {
        return ok({ ok: true });
      }
      return ok({});
    }

    if (path.includes("/api/models/catalog")) {
      return route.fulfill({ json: { version: 3, fetchedAt: "", source: "mock", providers: {} } });
    }

    if (path.endsWith("/api/health")) {
      return route.fulfill({ json: { ok: true, harness: "deadbeef", model: cfg.model, version: "0.1.0", isolated: false, workspace_ready: !!workspace, video_workspace: cfg.video_workspace, video_workspace_ready: true, paused } });
    }
    if (path.endsWith("/api/pause")) {
      if (method === "POST") paused = !!body().paused;
      return route.fulfill({ json: { paused } });
    }
    if (path.endsWith("/api/pages") || path.includes("/api/pages?")) {
      if (method === "POST") {
        const b = body();
        const preview = String(b.content || "").replace(/\s+/g, " ").trim().slice(0, 160);
        const p = {
          id: b.id || `p${pages.length + 1}`,
          title: b.title || "Untitled",
          content: b.content || "",
          revision: 1,
          space_id: b.space_id || "home",
          parent_id: b.parent_id || "",
          preview,
          updated_at: new Date().toISOString(),
        };
        pages = [p, ...pages.filter((x) => x.id !== p.id)];
        return route.fulfill({ json: p });
      }
      return route.fulfill({ json: pages });
    }
    if (path.includes("/api/pages/")) {
      const id = decodeURIComponent(path.split("/api/pages/")[1] || "").split("?")[0];
      if (id === "reviews" || id === "propose") {
        return route.fulfill({ json: id === "reviews" ? [] : { id: "r1", hash: "h1", title: "Note" } });
      }
      if (method === "DELETE") {
        pages = pages.filter((p) => p.id !== id);
        return route.fulfill({ json: { ok: true } });
      }
      const pg = pages.find((p) => p.id === id) || { id, title: "Untitled", content: "", revision: 1 };
      return route.fulfill({ json: pg });
    }
    if (path.endsWith("/api/profiles")) {
      if (method === "POST") return route.fulfill({ json: body() });
      return route.fulfill({ json: [{ id: "assistant", name: "Assistant", research: true, memory: true }] });
    }
    if (path.endsWith("/api/schedule") || path.endsWith("/api/schedule/retry")) {
      return route.fulfill({ json: method === "GET" ? [] : { id: "job-1", kind: "follow" } });
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
      const t = { id: `s${sessions.length + 1}`, title: "New chat", workspace, channel: ch, auth_mode: "full" };
      sessions = [t, ...sessions];
      return route.fulfill({ json: t });
    }
    if (path.endsWith("/api/sessions")) {
      return route.fulfill({ json: sessions });
    }
    const trajPage = path.match(/\/api\/sessions\/([^/]+)\/trajectory\/(page|around)$/);
    if (trajPage) {
      const id = decodeURIComponent(trajPage[1]);
      const events = extra?.trajectories?.[id] ?? extra?.events ?? [];
      return route.fulfill({
        json: { events, head_seq: 0, tail_seq: events.length, older: false, hub_seq: events.length },
      });
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
      if (method === "POST" && op === "restore") {
        return route.fulfill({ json: { note: "restored 1 files" } });
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
      if (method === "POST" && op === "profile") {
        if (idx >= 0) sessions[idx] = { ...sessions[idx], profile_id: String(body().profile_id || "") };
        return route.fulfill({ json: sessions[idx] || { id, profile_id: String(body().profile_id || "") } });
      }
      if (method === "POST" && op === "page") {
        if (idx >= 0) sessions[idx] = { ...sessions[idx], page_id: String(body().page_id || "") };
        return route.fulfill({ json: sessions[idx] || { id, page_id: String(body().page_id || "") } });
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
    if (path.includes("/api/packs")) {
      return route.fulfill({
        json: extra?.packs ?? [
          {
            id: "superpowers",
            name: "Superpowers",
            description: "Methodology pack",
            methodology: true,
            installed: false,
            enabled: false,
            enable_global: false,
            skill_count: 0,
            known: true,
            origin: { kind: "github", repo: "obra/superpowers" },
          },
          {
            id: "novel-to-game",
            name: "NovelToGame",
            description: "Domain pack: turn a novel into a playable game",
            methodology: false,
            installed: false,
            enabled: false,
            enable_global: false,
            skill_count: 7,
            known: true,
            origin: { kind: "github", repo: "zenstory-ai/novel-to-game" },
          },
        ],
      });
    }
    if (path.endsWith("/api/skills")) {
      return route.fulfill({ json: extra?.skills ?? [] });
    }
    if (path.includes("/api/skills") || path.includes("/api/doctor") || path.includes("/api/about") || path.includes("/api/key")) {
      return route.fulfill({ json: [] });
    }
    if (path.includes("/api/workspace/blob")) {
      const u = new URL(route.request().url());
      const p = u.searchParams.get("path") || "";
      const pdf = /\.pdf$/i.test(p);
      if (pdf) await new Promise((r) => setTimeout(r, 450));
      return route.fulfill({
        json: {
          path: p,
          kind: pdf ? "pdf" : /\.(png|jpe?g|gif|webp)$/i.test(p) ? "image" : /\.docx$/i.test(p) ? "office" : "binary",
          mime: pdf ? "application/pdf" : /\.docx$/i.test(p) ? "application/vnd.openxmlformats-officedocument.wordprocessingml.document" : "application/octet-stream",
          bytes: 32,
          truncated: false,
          base64: pdf ? MINI_PDF_B64 : "",
        },
      });
    }
    if (path.includes("/api/workspace/file")) {
      const u = new URL(route.request().url());
      const p = u.searchParams.get("path") || "";
      const html = /\.html?$/i.test(p);
      const pdf = /\.pdf$/i.test(p);
      const office = /\.(docx|xlsx|pptx)$/i.test(p);
      const go = /\.go$/i.test(p);
      return route.fulfill({
        json: {
          path: p,
          text: pdf || office ? "" : html ? "<!doctype html><html><body><h1>preview</h1></body></html>" : go ? "package main\n\nfunc Hello() string {\n\treturn \"yoyo\"\n}\n" : "package main\n",
          lang: html ? "html" : go ? "go" : "text",
          html,
          kind: pdf ? "pdf" : office ? "office" : html ? "html" : go ? "text" : "text",
          mime: pdf ? "application/pdf" : html ? "text/html" : go ? "text/plain" : "text/plain",
          binary: office || pdf,
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
    if (path.endsWith("/api/open")) {
      return route.fulfill({ json: { ok: true } });
    }
    if (path.includes("/api/update")) {
      return route.fulfill({
        json: extra?.update ?? {
          phase: "idle",
          staged: false,
          auto_check: true,
          feed: "https://www.modelscope.ai/datasets/yoyo-agent/yoyo/resolve/master/channels/latest.json",
          current: { version: "0.4.0" },
        },
      });
    }
    if (path.endsWith("/api/inbox")) {
      return route.fulfill({ json: extra?.inbox ?? [] });
    }
    if (path.endsWith("/api/review/queue")) {
      return route.fulfill({ json: extra?.queue ?? { offers: [], inbox: [], drafts: [], browser: [] } });
    }
    if (path.endsWith("/api/skins")) {
      if (method === "GET") {
        const u = new URL(route.request().url());
        const id = u.searchParams.get("id") || "";
        if (id && u.searchParams.get("resolve")) {
          return route.fulfill({
            json: {
              id: id || "builtin.ink",
              name: "Ink",
              builtin: !id || id.startsWith("builtin."),
              mode: u.searchParams.get("mode") || "dark",
              preserveEvidence: true,
              tokens: {},
              windowRgb: [28, 28, 26],
              materials: { grain: 0.04, wallpaperFit: "cover", wallpaperDim: 0.45, radiusScale: 1 },
              contrast: { foreground: 12, muted: 5, pass: true },
            },
          });
        }
        if (id) {
          return route.fulfill({ json: { info: { id, name: id, builtin: true, preserveEvidence: true }, dark: {}, light: {} } });
        }
        return route.fulfill({
          json: [
            { id: "builtin.ink", name: "Ink", builtin: true, preserveEvidence: true },
            { id: "builtin.neutral", name: "Neutral", builtin: true, preserveEvidence: true },
          ],
        });
      }
      return route.fulfill({ json: { ok: true } });
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
      tokens: p.tokens || p.prompt_tokens || 0,
      prompt_tokens: p.prompt_tokens || 0,
      completion_tokens: p.completion_tokens || 0,
      cached_tokens: p.cached_tokens || 0,
      ttft_ms: p.ttft_ms || 0,
      cache_reported: !!p.cache_reported || (p.cached_tokens || 0) > 0,
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
