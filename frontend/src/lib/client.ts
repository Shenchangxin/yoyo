import { asArray, asBool, bool, boolOr, errMessage, num, pick, str } from "./normalize";
import { getLocale } from "./i18n";
import type { AppConfig, Approval, Attachment, AuthMode, ContextUsage, FileHit, Health, Hunk, RunStatus, SessionTrace, SkillInfo, SpillBlob, Thread, ThreadChannel, TraceArtifact, TraceEvent, TraceStats } from "./protocol";

export class ApiError extends Error {
  status: number;
  body: any;
  constructor(status: number, body: any, message: string) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

// Vite must see this glob so production builds embed the generated Service.
// `@vite-ignore` of a runtime path 404s in the installer (no /bindings/*.js on the asset server),
// then every call falls through to `/api/*` which the desktop webview does not serve.
const boundService = import.meta.glob("../../bindings/**/internal/desktop/service.ts");

function unwrapService(mod: any): any | null {
  const svc = mod?.Service ?? mod?.default ?? mod;
  if (svc && typeof (svc.ListSessions || svc.Health || svc.listSessions) === "function") return svc;
  return null;
}

function nativeDesktop(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as any;
  return !!(w.__YOYO_DESKTOP__ || w._wails?.environment?.OS);
}

function namedDesktopService(): any {
  const name = (method: string) => `github.com/Shenchangxin/yoyo/internal/desktop.Service.${method}`;
  return new Proxy(
    {},
    {
      get(_target, prop) {
        if (typeof prop !== "string" || prop === "then") return undefined;
        return (...args: any[]) =>
          import("@wailsio/runtime").then((mod: any) => {
            const byName = mod.Call?.ByName;
            if (typeof byName !== "function") throw new Error("Wails Call.ByName missing");
            return byName(name(prop), ...args);
          });
      },
    },
  );
}

async function wailsService(): Promise<any | null> {
  if (import.meta.env.VITE_E2E) return null;
  const loaders = Object.values(boundService);
  if (loaders.length) {
    try {
      const svc = unwrapService(await loaders[0]!());
      if (svc) return svc;
    } catch {
      /* generated module missing at runtime */
    }
  }
  if (nativeDesktop()) return namedDesktopService();
  return null;
}

function svcMethod(s: any, ...names: string[]): ((...args: any[]) => any) | null {
  if (!s) return null;
  for (const n of names) {
    if (typeof s[n] === "function") return s[n].bind(s);
  }
  return null;
}

async function http<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      headers: { "Content-Type": "application/json" },
      ...init,
      signal: init?.signal ?? AbortSignal.timeout(8000),
    });
  } catch (e) {
    throw new Error(`${path}: ${errMessage(e)}`);
  }
  const text = await res.text();
  let body: any = text;
  try {
    body = text ? JSON.parse(text) : {};
  } catch {
    /* keep text */
  }
  if (!res.ok) {
    const raw = typeof body === "object" && body?.error ? body.error : text || res.statusText;
    throw new ApiError(res.status, body, `${path}: ${raw}`);
  }
  return body as T;
}

export function parseAuthMode(v: any): AuthMode {
  const s = str(v).toLowerCase().replace(/[-\s]/g, "_");
  if (s === "ask" || s === "ask_every_time" || s === "strict") return "ask";
  if (s === "auto_edit" || s === "autoedit") return "auto_edit";
  if (s === "full" || s === "full_access") return "full";
  if (!s) return "default";
  return "default";
}

export function threadOf(v: any): Thread {
  const workspace = str(pick(v, "origin_workspace", "OriginWorkspace", "workspace", "Workspace"));
  const worktree = str(pick(v, "worktree", "Worktree"));
  const isolate = bool(pick(v, "isolate", "Isolate"));
  const toolRoot = isolate && worktree ? worktree : str(pick(v, "workspace", "Workspace"), workspace);
  return {
    id: str(pick(v, "id", "ID")),
    title: str(pick(v, "title", "Title"), "thread"),
    workspace,
    toolRoot,
    worktree,
    originWorkspace: workspace,
    isolate,
    harness: str(pick(v, "harness", "Harness")),
    createdAt: str(pick(v, "created_at", "CreatedAt")),
    archived: bool(pick(v, "archived", "Archived")),
    pinned: bool(pick(v, "pinned", "Pinned")),
    model: str(pick(v, "model", "Model")),
    authMode: parseAuthMode(pick(v, "auth_mode", "AuthMode", "authMode")),
    pinnedSkills: asArray(pick(v, "pinned_skills", "PinnedSkills")).map(String).filter(Boolean),
    loadedSkills: asArray(pick(v, "loaded_skills", "LoadedSkills")).map(String).filter(Boolean),
    channel: str(pick(v, "channel", "Channel")) === "video" ? "video" : "agent",
    interrupted: bool(pick(v, "interrupted", "Interrupted")),
    queued: num(pick(v, "queued", "Queued")),
  };
}

export function healthOf(v: any): Health {
  const usage = pick(v, "usage") || {};
  return {
    ok: bool(pick(v, "ok")),
    harness: str(pick(v, "harness")),
    model: str(pick(v, "model")),
    version: str(pick(v, "version"), "0.1.0"),
    isolated: bool(pick(v, "isolated")),
    isolationKind: str(pick(v, "isolation_kind", "isolationKind") || pick(pick(pick(v, "isolation"), "os") || {}, "kind", "Kind")),
    budgetUsd: num(pick(v, "budget_usd", "budgetUsd")),
    usageUsd: num(pick(usage, "usd")),
    workspaceReady: bool(pick(v, "workspace_ready", "workspaceReady")),
    videoWorkspace: str(pick(v, "video_workspace", "VideoWorkspace", "videoWorkspace")),
    videoWorkspaceReady: bool(pick(v, "video_workspace_ready", "videoWorkspaceReady")),
  };
}

export function configOf(v: any): AppConfig {
  return {
    provider: str(pick(v, "provider"), "openai"),
    model: str(pick(v, "model")),
    baseUrl: str(pick(v, "base_url", "BaseURL", "baseUrl")),
    workspace: str(pick(v, "workspace", "Workspace")),
    videoWorkspace: str(pick(v, "video_workspace", "VideoWorkspace", "videoWorkspace")),
    autoAllow: bool(pick(v, "auto_allow", "AutoAllow", "autoAllow")),
    maxBudgetUsd: num(pick(v, "max_budget_usd", "MaxBudgetUSD", "maxBudgetUsd")),
    usdPerMtok: num(pick(v, "usd_per_mtok", "USDPerMTok", "usdPerMtok")),
    models: asArray(pick(v, "models", "Models")).map(String).filter(Boolean),
    closeToTray: bool(pick(v, "close_to_tray", "CloseToTray", "closeToTray")),
    companionEnabled: boolOr(pick(v, "companion_enabled", "CompanionEnabled", "companionEnabled"), true),
    updateUrl: str(pick(v, "update_url", "UpdateURL", "updateUrl")),
    keymap: (pick(v, "keymap", "Keymap") || {}) as Record<string, string>,
    locale: str(pick(v, "locale", "Locale")),
    alwaysOnTop: bool(pick(v, "always_on_top", "AlwaysOnTop", "alwaysOnTop")),
    startAtLogin: bool(pick(v, "start_at_login", "StartAtLogin", "startAtLogin")),
    notificationsEnabled: boolOr(pick(v, "notifications_enabled", "NotificationsEnabled", "notificationsEnabled"), true),
    notifyWhenUnfocusedOnly: bool(pick(v, "notify_when_unfocused_only", "NotifyWhenUnfocusedOnly", "notifyWhenUnfocusedOnly")),
    uiScale: num(pick(v, "ui_scale", "UIScale", "uiScale"), 1) || 1,
    updateChannel: str(pick(v, "update_channel", "UpdateChannel", "updateChannel"), "nightly"),
    theme: str(pick(v, "theme", "Theme"), "system"),
    paletteDark: str(pick(v, "palette_dark", "PaletteDark", "paletteDark"), "ink"),
    paletteLight: str(pick(v, "palette_light", "PaletteLight", "paletteLight"), "neutral"),
    gateMode: str(pick(v, "gate_mode", "GateMode", "gateMode"), "manual"),
    crashResume: boolOr(pick(v, "crash_resume", "CrashResume", "crashResume"), true),
    searchUrl: str(pick(v, "search_url", "SearchURL", "searchUrl")),
    searchKey: str(pick(v, "search_key", "SearchKey", "searchKey")),
  };
}

export function approvalOf(v: any): Approval {
  const req = pick(v, "request", "Request") || v;
  return {
    id: str(pick(v, "id", "ID")),
    action: str(pick(req, "action", "Action")),
    command: str(pick(req, "command", "Command")),
    path: str(pick(req, "path", "Path")),
    level: str(pick(req, "level", "Level")),
    sessionId: str(pick(req, "session_id", "SessionID", "sessionId") ?? pick(v, "session_id", "SessionID", "sessionId")),
  };
}

export function contextOf(v: any): ContextUsage {
  return {
    tokens: num(pick(v, "tokens", "Tokens")),
    budget: num(pick(v, "budget", "Budget")),
    window: num(pick(v, "window", "Window")),
    prefixTokens: num(pick(v, "prefix_tokens", "PrefixTokens")),
    dynamicTokens: num(pick(v, "dynamic_tokens", "DynamicTokens")),
    schemaTokens: num(pick(v, "schema_tokens", "SchemaTokens")),
    providerPrompt: num(pick(v, "provider_prompt", "ProviderPrompt")),
    cachedTokens: num(pick(v, "cached_tokens", "CachedTokens")),
    cacheReported: !!pick(v, "cache_reported", "CacheReported"),
    cacheStable: !!pick(v, "cache_stable", "CacheStable"),
    prefixHash: str(pick(v, "prefix_hash", "PrefixHash")),
    dynamicAt: str(pick(v, "dynamic_at", "DynamicAt")),
    trigger: str(pick(v, "trigger", "Trigger")),
    hydrated: num(pick(v, "hydrated", "Hydrated")),
    note: str(pick(v, "note", "Note")),
    layers: asArray(pick(v, "layers", "Layers")).map(String),
    elided: num(pick(v, "elided", "Elided")),
  };
}

export function hunkOf(v: any): Hunk {
  return {
    id: str(pick(v, "id", "ID")),
    file: str(pick(v, "file", "File")),
    header: str(pick(v, "header", "Header")),
    body: str(pick(v, "body", "Body")),
  };
}

export async function health(): Promise<Health> {
  const s = await wailsService();
  return healthOf(s?.Health ? await s.Health() : await http("/api/health"));
}

export async function getConfig(): Promise<AppConfig> {
  const s = await wailsService();
  return configOf(s?.GetConfig ? await s.GetConfig() : await http("/api/config"));
}

export async function setConfig(cfg: AppConfig): Promise<void> {
  const body = {
    provider: cfg.provider,
    model: cfg.model,
    base_url: cfg.baseUrl,
    workspace: cfg.workspace,
    video_workspace: cfg.videoWorkspace || "",
    auto_allow: cfg.autoAllow,
    max_budget_usd: cfg.maxBudgetUsd,
    usd_per_mtok: cfg.usdPerMtok,
    models: cfg.models,
    close_to_tray: cfg.closeToTray,
    companion_enabled: cfg.companionEnabled !== false,
    CompanionEnabled: cfg.companionEnabled !== false,
    update_url: cfg.updateUrl,
    locale: cfg.locale || "",
    Locale: cfg.locale || "",
    keymap: cfg.keymap || {},
    always_on_top: !!cfg.alwaysOnTop,
    start_at_login: !!cfg.startAtLogin,
    notifications_enabled: cfg.notificationsEnabled !== false,
    notify_when_unfocused_only: !!cfg.notifyWhenUnfocusedOnly,
    ui_scale: cfg.uiScale || 1,
    update_channel: cfg.updateChannel || "nightly",
    theme: cfg.theme || "system",
    palette_dark: cfg.paletteDark || "ink",
    palette_light: cfg.paletteLight || "neutral",
  };
  const s = await wailsService();
  if (s?.SetConfig) {
    await s.SetConfig(body);
    const loc = svcMethod(s, "SetLocale", "setLocale");
    if (loc && cfg.locale) await loc(cfg.locale);
    return;
  }
  await http("/api/config", { method: "POST", body: JSON.stringify(body) });
}

export async function raiseSession(id: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "RaiseSession", "raiseSession");
  if (fn) {
    await fn(id);
    return;
  }
  await raiseWindow();
}

export async function raiseWindow(): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "RaiseWindow", "raiseWindow");
  if (fn) await fn();
}

export async function showCompanion(): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "ShowCompanion", "showCompanion");
  if (fn) await fn();
}

export async function dismissCompanion(): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "HideCompanion", "hideCompanion") || svcMethod(s, "DismissCompanion", "dismissCompanion");
  if (fn) await fn();
}

export async function setCompanionCaption(on: boolean): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "SetCompanionCaption", "setCompanionCaption");
  if (fn) await fn(on);
}

export async function placeCompanion(x: number, y: number): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "PlaceCompanion", "placeCompanion");
  if (fn) await fn(Math.round(x), Math.round(y));
}

export async function setAPIKey(value: string): Promise<void> {
  const s = await wailsService();
  if (s?.SetAPIKey) return s.SetAPIKey(value);
  await http("/api/key", { method: "POST", body: JSON.stringify({ value }) });
}

export async function createSession(workspace: string, channel: ThreadChannel = "agent"): Promise<Thread> {
  const s = await wailsService();
  if (s?.CreateSessionOn) {
    return threadOf(await s.CreateSessionOn(workspace, channel));
  }
  if (s?.CreateSession) {
    const t = threadOf(await s.CreateSession(workspace));
    if (channel === "video" && s.SetSessionChannel) {
      return threadOf(await s.SetSessionChannel(t.id, "video"));
    }
    if (channel === "agent") return t;
  }
  const raw = await http("/api/sessions", { method: "POST", body: JSON.stringify({ workspace, channel }) });
  return threadOf(raw);
}

export async function listSessions(): Promise<Thread[]> {
  const s = await wailsService();
  const raw = s?.ListSessions ? await s.ListSessions() : await http("/api/sessions");
  return asArray(raw).map(threadOf).filter((t) => t.id);
}

export async function trajectory(id: string): Promise<any[]> {
  const s = await wailsService();
  if (s?.Trajectory) return asArray(await s.Trajectory(id));
  return asArray(await http(`/api/sessions/${id}/trajectory`));
}

export function sessionTraceOf(v: any): SessionTrace {
  if (v && typeof v === "object" && v.result && typeof v.result === "object" && !Array.isArray(v.result)) {
    v = v.result;
  }
  const statsRaw = pick(v, "stats", "Stats") || {};
  const stats: TraceStats = {
    events: num(pick(statsRaw, "events", "Events")),
    users: num(pick(statsRaw, "users", "Users")),
    assistants: num(pick(statsRaw, "assistants", "Assistants")),
    toolCalls: num(pick(statsRaw, "tool_calls", "ToolCalls")),
    toolResults: num(pick(statsRaw, "tool_results", "ToolResults")),
    errors: num(pick(statsRaw, "errors", "Errors")),
    compactions: num(pick(statsRaw, "compactions", "Compactions")),
    deltas: num(pick(statsRaw, "deltas", "Deltas")),
    tokens: num(pick(statsRaw, "tokens", "Tokens")),
    durationMs: num(pick(statsRaw, "duration_ms", "DurationMs")),
    spillBytes: num(pick(statsRaw, "spill_bytes", "SpillBytes")),
  };
  const events: TraceEvent[] = asArray(pick(v, "events", "Events")).map((ev: any, i: number) => ({
    index: num(pick(ev, "index", "Index"), i),
    ts: str(pick(ev, "ts", "TS")),
    type: str(pick(ev, "type", "Type")),
    source: str(pick(ev, "source", "Source")),
    lane: str(pick(ev, "lane", "Lane")) || "dialog",
    round: str(pick(ev, "round", "Round")),
    name: str(pick(ev, "name", "Name")),
    id: str(pick(ev, "id", "ID")),
    summary: str(pick(ev, "summary", "Summary")),
    detail: str(pick(ev, "detail", "Detail")),
    bytes: num(pick(ev, "bytes", "Bytes")),
    elapsedMs: num(pick(ev, "elapsed_ms", "ElapsedMs")),
    spillId: str(pick(ev, "spill_id", "SpillID", "spillId")),
    tokens: num(pick(ev, "tokens", "Tokens")),
    error: bool(pick(ev, "error", "Error")),
  }));
  const artifacts: TraceArtifact[] = asArray(pick(v, "artifacts", "Artifacts")).map((art: any) => ({
    kind: str(pick(art, "kind", "Kind")),
    id: str(pick(art, "id", "ID")),
    label: str(pick(art, "label", "Label")) || str(pick(art, "id", "ID")),
    bytes: num(pick(art, "bytes", "Bytes")),
    preview: str(pick(art, "preview", "Preview")),
  }));
  return {
    sessionId: str(pick(v, "session_id", "SessionID", "sessionId")),
    title: str(pick(v, "title", "Title")),
    workspace: str(pick(v, "workspace", "Workspace")),
    harness: str(pick(v, "harness", "Harness")),
    model: str(pick(v, "model", "Model")),
    createdAt: str(pick(v, "created_at", "CreatedAt")),
    startedAt: str(pick(v, "started_at", "StartedAt")),
    endedAt: str(pick(v, "ended_at", "EndedAt")),
    stats,
    events,
    artifacts,
  };
}

export function spillBlobOf(v: any): SpillBlob {
  if (v && typeof v === "object" && v.result && typeof v.result === "object" && !Array.isArray(v.result)) {
    v = v.result;
  }
  return {
    id: str(pick(v, "id", "ID")),
    bytes: num(pick(v, "bytes", "Bytes")),
    text: str(pick(v, "text", "Text")),
    truncated: bool(pick(v, "truncated", "Truncated")),
  };
}

export async function sessionTrace(id: string): Promise<SessionTrace> {
  const s = await wailsService();
  const raw = s?.SessionTrace ? await s.SessionTrace(id) : await http(`/api/sessions/${id}/trace`);
  return sessionTraceOf(raw);
}

export async function spillBlob(sessionID: string, blobID: string): Promise<SpillBlob> {
  const s = await wailsService();
  const raw = s?.SpillBlob
    ? await s.SpillBlob(sessionID, blobID)
    : await http(`/api/sessions/${sessionID}/spill/${encodeURIComponent(blobID)}`);
  return spillBlobOf(raw);
}

export async function retry(sessionID: string): Promise<void> {
  const s = await wailsService();
  const bound = svcMethod(s, "RetrySession", "retrySession");
  if (bound) {
    await bound(sessionID);
    return;
  }
  if (s?.StartSendOpts) {
    await s.StartSendOpts(sessionID, "", false, [{ name: "__resume__", path: "__resume__" }]);
    return;
  }
  try {
    await http(`/api/sessions/${sessionID}/retry`, { method: "POST" });
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      await send(sessionID, "", { attachments: [{ name: "__resume__", path: "__resume__" }] });
      return;
    }
    throw e;
  }
}

export async function send(sessionID: string, text: string, opts?: { plan?: boolean; attachments?: Attachment[] }): Promise<{ queued?: boolean }> {
  const s = await wailsService();
  try {
    if (s?.StartSendOpts) {
      const raw = await s.StartSendOpts(sessionID, text, !!opts?.plan, opts?.attachments || []);
      return { queued: isQueuedResult(raw) };
    }
    if (s?.StartSend) {
      const raw = await s.StartSend(sessionID, text, !!opts?.plan);
      return { queued: isQueuedResult(raw) };
    }
    const raw = await http<any>(`/api/sessions/${sessionID}/messages`, {
      method: "POST",
      body: JSON.stringify({ text, async: true, plan: !!opts?.plan, attachments: opts?.attachments || [] }),
    });
    return { queued: !!raw?.queued || isQueuedResult(raw) };
  } catch (e) {
    const m = errMessage(e);
    if (/queued/i.test(m)) return { queued: true };
    throw e;
  }
}

function isQueuedResult(raw: any): boolean {
  if (raw == null) return false;
  if (typeof raw === "string") return /queued/i.test(raw);
  if (typeof raw === "object") return !!(raw.queued || raw.Queued);
  return false;
}

export async function queueList(sessionID: string): Promise<{ id?: string; text?: string; plan?: boolean }[]> {
  const s = await wailsService();
  if (s?.QueueList) return asArray(await s.QueueList(sessionID));
  if (s) return [];
  return asArray(await http(`/api/sessions/${sessionID}/queue`));
}

export async function queueCancel(sessionID: string, itemID: string): Promise<boolean> {
  const s = await wailsService();
  if (s?.QueueCancel) return asBool(await s.QueueCancel(sessionID, itemID));
  if (s) return false;
  const r = await http<any>(`/api/sessions/${sessionID}/queue?id=${encodeURIComponent(itemID)}`, { method: "DELETE" });
  return asBool(pick(r, "ok", "Ok") ?? r);
}

export async function queueReorder(sessionID: string, itemID: string, delta: number): Promise<boolean> {
  const s = await wailsService();
  if (s?.QueueReorder) return asBool(await s.QueueReorder(sessionID, itemID, delta));
  if (s) return false;
  const r = await http<any>(`/api/sessions/${sessionID}/queue`, {
    method: "POST",
    body: JSON.stringify({ id: itemID, delta }),
  });
  return asBool(pick(r, "ok", "Ok") ?? r);
}

export async function applySessionWorktree(sessionID: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ApplySessionWorktree
    ? await s.ApplySessionWorktree(sessionID)
    : await http(`/api/sessions/${sessionID}/worktree`, { method: "POST" });
  return threadOf(raw);
}

export async function running(sessionID: string): Promise<boolean> {
  const s = await wailsService();
  if (s?.Running) {
    const v = await s.Running(sessionID);
    return asBool(v);
  }
  const r = await http<any>(`/api/sessions/${sessionID}/running`);
  return asBool(r);
}

export async function runningIDs(): Promise<string[]> {
  const s = await wailsService();
  if (s?.RunningIDs) {
    const v = await s.RunningIDs();
    return asArray(v).map(String).filter(Boolean);
  }
  if (s) return [];
  const r = await http<any>("/api/running");
  return asArray(pick(r, "ids", "IDs") ?? r).map(String).filter(Boolean);
}

export async function runningStatus(): Promise<RunStatus[]> {
  const s = await wailsService();
  const raw = s?.RunningStatus ? await s.RunningStatus() : s ? [] : pick(await http<any>("/api/running"), "runs", "Runs") ?? [];
  return asArray(raw).map((v) => ({
    id: str(pick(v, "id", "ID")),
    startedAt: str(pick(v, "started_at", "StartedAt")),
    lastTool: str(pick(v, "last_tool", "LastTool")),
  })).filter((x) => x.id);
}

export async function contextUsage(sessionID: string): Promise<ContextUsage> {
  const s = await wailsService();
  const raw = s?.ContextUsage ? await s.ContextUsage(sessionID) : await http(`/api/sessions/${sessionID}/context`);
  return contextOf(raw);
}

export async function interrupt(sessionID: string): Promise<void> {
  const s = await wailsService();
  if (s?.Interrupt) return s.Interrupt(sessionID);
  await http(`/api/sessions/${sessionID}/interrupt`, { method: "POST" });
}

export async function approvals(): Promise<Approval[]> {
  const s = await wailsService();
  const raw = s?.PendingApprovals ? await s.PendingApprovals() : await http("/api/approvals");
  return asArray(raw).map(approvalOf).filter((a) => a.id);
}

export async function resolveApproval(id: string, decision: string): Promise<void> {
  const s = await wailsService();
  if (s?.ResolveApproval) return s.ResolveApproval(id, decision);
  await http("/api/approvals", { method: "POST", body: JSON.stringify({ id, decision }) });
}

export async function harness(): Promise<any> {
  const s = await wailsService();
  if (s?.Harness) return s.Harness();
  return http("/api/harness");
}

export async function checkout(hash: string, l3 = false): Promise<void> {
  const s = await wailsService();
  if (l3 && s?.CheckoutL3) return s.CheckoutL3(hash);
  if (!l3 && s?.Checkout) return s.Checkout(hash);
  await http("/api/harness/checkout", { method: "POST", body: JSON.stringify({ hash, l3 }) });
}

export async function rollback(): Promise<void> {
  const s = await wailsService();
  if (s?.Rollback) return s.Rollback();
  await http("/api/harness/rollback", { method: "POST" });
}

export async function revealHarness(hash = ""): Promise<void> {
  const s = await wailsService();
  if (s?.RevealHarness) return s.RevealHarness(hash);
  await http("/api/harness/reveal", { method: "POST", body: JSON.stringify({ hash }) });
}

export async function diff(a: string, b: string): Promise<any> {
  const s = await wailsService();
  if (s?.DiffDetail) return s.DiffDetail(a, b);
  return http(`/api/harness/diff?a=${encodeURIComponent(a)}&b=${encodeURIComponent(b)}`);
}

export async function runEval(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEval) return s.RunEval();
  return http("/api/eval", { method: "POST" });
}

export async function bestOfN(n: number): Promise<any> {
  const s = await wailsService();
  if (s?.BestOfN) return s.BestOfN(n);
  return http("/api/eval/best", { method: "POST", body: JSON.stringify({ n }) });
}

export async function evolve(k = 3, opts?: {
  rounds?: number;
  sealed?: boolean;
  promote?: boolean;
  behavior?: boolean;
  index?: boolean;
  baselines?: number;
  maxUsd?: number;
}): Promise<any> {
  const body = {
    k,
    rounds: opts?.rounds ?? 0,
    sealed: !!opts?.sealed,
    promote: !!opts?.promote,
    behavior: !!opts?.behavior,
    index: !!opts?.index,
    baselines: opts?.baselines ?? 0,
    max_usd: opts?.maxUsd ?? 0,
  };
  const s = await wailsService();
  if (s?.EvolveLab) {
    return s.EvolveLab(body.k, body.rounds, body.sealed, body.promote, body.behavior, body.index, body.baselines, body.max_usd);
  }
  if (s?.EvolveRun) return s.EvolveRun(body.k, body.rounds, body.sealed, body.promote);
  if (s?.EvolveK && !body.rounds && !body.sealed && !body.promote) return s.EvolveK(k);
  if (s?.Evolve && k === 3 && !body.rounds && !body.sealed && !body.promote) return s.Evolve();
  return http("/api/evolve", { method: "POST", body: JSON.stringify(body) });
}

export async function archive(): Promise<any[]> {
  const s = await wailsService();
  if (s?.Archive) return asArray(await s.Archive());
  return asArray(await http("/api/archive"));
}

export async function plugins(): Promise<any> {
  const s = await wailsService();
  if (s?.Plugins) return s.Plugins();
  return http("/api/plugins");
}

export async function unloadFiber(name: string): Promise<void> {
  const s = await wailsService();
  if (s?.UnloadFiber) return s.UnloadFiber(name);
}

export async function forkSession(id: string, from = ""): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ForkSessionFrom && from
    ? await s.ForkSessionFrom(id, from)
    : s?.ForkSession && !from
      ? await s.ForkSession(id)
      : await http(`/api/sessions/${id}/fork`, { method: "POST", body: from ? JSON.stringify({ from }) : undefined });
  return threadOf(raw);
}

export async function dumpSession(id: string): Promise<{ id: string; path: string }> {
  const s = await wailsService();
  const raw = s?.DumpSession
    ? await s.DumpSession(id)
    : await http(`/api/sessions/${id}/dump`);
  const paths = pick(raw, "paths", "Paths") || {};
  return {
    id: str(pick(raw, "id", "ID"), id),
    path: str(pick(paths, "dump", "Dump"), str(pick(raw, "path", "Path"))),
  };
}

export async function renameSession(id: string, title: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "RenameSession", "renameSession");
  if (fn) {
    await fn(id, title);
    return;
  }
  await http(`/api/sessions/${id}/title`, { method: "POST", body: JSON.stringify({ title }) });
}

export async function playbook(): Promise<any> {
  const s = await wailsService();
  if (s?.Playbook) return s.Playbook();
  return http("/api/playbook");
}

export async function ratePlaybook(id: string, helpful: boolean): Promise<any> {
  const s = await wailsService();
  if (s?.RatePlaybook) return s.RatePlaybook(id, helpful);
  return http("/api/playbook", { method: "POST", body: JSON.stringify({ id, helpful }) });
}

export async function workspaceHunks(workspace: string): Promise<{ diff: string; hunks: Hunk[] }> {
  const s = await wailsService();
  const raw = s?.WorkspaceHunks
    ? await s.WorkspaceHunks(workspace)
    : await http(`/api/workspace/hunks?workspace=${encodeURIComponent(workspace || "")}`);
  return {
    diff: str(pick(raw, "diff", "Diff")),
    hunks: asArray(pick(raw, "hunks", "Hunks")).map(hunkOf).filter((h) => h.id),
  };
}

export type FilePreview = {
  path: string;
  text: string;
  lang: string;
  html: boolean;
  binary: boolean;
  truncated: boolean;
  bytes: number;
};

export async function previewWorkspaceFile(workspace: string, path: string): Promise<FilePreview> {
  const s = await wailsService();
  const raw = s?.PreviewWorkspaceFile
    ? await s.PreviewWorkspaceFile(workspace, path)
    : await http(`/api/workspace/file?workspace=${encodeURIComponent(workspace || "")}&path=${encodeURIComponent(path || "")}`);
  return {
    path: str(pick(raw, "path", "Path")),
    text: str(pick(raw, "text", "Text")),
    lang: str(pick(raw, "lang", "Lang")),
    html: bool(pick(raw, "html", "HTML")),
    binary: bool(pick(raw, "binary", "Binary")),
    truncated: bool(pick(raw, "truncated", "Truncated")),
    bytes: num(pick(raw, "bytes", "Bytes")),
  };
}

export async function applyHunks(workspace: string, ids: string[]): Promise<void> {
  const s = await wailsService();
  if (s?.ApplyHunks) return s.ApplyHunks(workspace, ids);
  await http("/api/workspace/hunks", { method: "POST", body: JSON.stringify({ workspace, ids }) });
}

export async function reverseHunks(workspace: string, ids: string[], snapshot: string): Promise<void> {
  const s = await wailsService();
  if (s?.ReverseHunks) return s.ReverseHunks(workspace, ids, snapshot);
  await http("/api/workspace/hunks", { method: "POST", body: JSON.stringify({ workspace, ids, snapshot, reverse: true }) });
}

export async function runEvalSafety(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalSafety) return s.RunEvalSafety();
  return http("/api/eval/safety", { method: "POST" });
}

export async function runEvalTB(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalTB) return s.RunEvalTB();
  return http("/api/eval/tb", { method: "POST" });
}

export async function runEvalSealed(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalSealed) return s.RunEvalSealed();
  return http("/api/eval/sealed", { method: "POST" });
}

export async function runEvalTransfer(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalTransfer) return s.RunEvalTransfer();
  return http("/api/eval/transfer", { method: "POST" });
}

export async function runEvalIndex(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalIndex) return s.RunEvalIndex();
  return http("/api/eval/index", { method: "POST" });
}

export async function runEvalBehavior(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalBehavior) return s.RunEvalBehavior();
  return http("/api/eval/behavior", { method: "POST" });
}

export async function lastEval(): Promise<any> {
  const s = await wailsService();
  if (s?.LastEval) return s.LastEval();
  return http("/api/eval/last");
}

export async function lastEvolve(): Promise<any> {
  const s = await wailsService();
  if (s?.LastEvolve) return s.LastEvolve();
  return http("/api/evolve/last");
}

export async function bestOfModels(models: string[]): Promise<any> {
  const s = await wailsService();
  if (s?.BestOfModels) return s.BestOfModels(models);
  return http("/api/eval/best", { method: "POST", body: JSON.stringify({ models }) });
}

export async function applyUpdate(): Promise<void> {
  const s = await wailsService();
  if (s?.ApplyUpdate) return s.ApplyUpdate();
  await http("/api/update", { method: "POST", body: JSON.stringify({}) });
}

export async function checkoutSafe(hash: string): Promise<void> {
  try {
    await checkout(hash, false);
  } catch (e: any) {
    const l3 = e?.body?.l3 || String(e.message || e).includes("L3 confirmation");
    if (!l3) throw e;
    const surfaces = (e?.body?.surfaces || ["loop_preset/policy_pack"]).join(", ");
    const zh = getLocale() === "zh-CN";
    const ok = window.confirm(
      zh ? `L3 门禁：将改动 ${surfaces}。确认 checkout？` : `L3 gate: changing ${surfaces}. Confirm checkout?`,
    );
    if (!ok) throw e;
    await checkout(hash, true);
  }
}

export async function deleteSession(id: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "DeleteSession", "deleteSession");
  if (fn) {
    await fn(id);
    return;
  }
  await http(`/api/sessions/${id}`, { method: "DELETE" });
}

export async function archiveSession(id: string, archived: boolean): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ArchiveSession
    ? await s.ArchiveSession(id, archived)
    : await http(`/api/sessions/${id}/archive`, { method: "POST", body: JSON.stringify({ archived }) });
  return threadOf(raw);
}

export async function pinSession(id: string, pinned: boolean): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.PinSession
    ? await s.PinSession(id, pinned)
    : await http(`/api/sessions/${id}/pin`, { method: "POST", body: JSON.stringify({ pinned }) });
  return threadOf(raw);
}

export async function exportSession(id: string): Promise<string> {
  const s = await wailsService();
  if (s?.ExportSession) return String(await s.ExportSession(id) || "");
  const raw = await http<any>(`/api/sessions/${id}/export`);
  return str(pick(raw, "markdown"));
}

export async function setSessionModel(id: string, model: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionModel
    ? await s.SetSessionModel(id, model)
    : await http(`/api/sessions/${id}/model`, { method: "POST", body: JSON.stringify({ model }) });
  return threadOf(raw);
}

export async function setSessionAuthMode(id: string, mode: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionAuthMode
    ? await s.SetSessionAuthMode(id, mode)
    : await http(`/api/sessions/${id}/auth`, { method: "POST", body: JSON.stringify({ mode }) });
  return threadOf(raw);
}

export async function setSessionWorkspace(id: string, workspace: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionWorkspace
    ? await s.SetSessionWorkspace(id, workspace)
    : await http(`/api/sessions/${id}/workspace`, { method: "POST", body: JSON.stringify({ workspace }) });
  return threadOf(raw);
}

export async function setSessionIsolate(id: string, isolate: boolean): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionIsolate
    ? await s.SetSessionIsolate(id, isolate)
    : await http(`/api/sessions/${id}/isolate`, { method: "POST", body: JSON.stringify({ isolate }) });
  return threadOf(raw);
}

export async function setSessionPinnedSkills(id: string, names: string[]): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionPinnedSkills
    ? await s.SetSessionPinnedSkills(id, names)
    : await http(`/api/sessions/${id}/skills`, { method: "POST", body: JSON.stringify({ names }) });
  return threadOf(raw);
}

export async function compactSession(id: string, focus?: string): Promise<string> {
  const s = await wailsService();
  if (s?.CompactSessionFocus) return String(await s.CompactSessionFocus(id, focus || "") || "");
  if (s?.CompactSession && !focus) return String(await s.CompactSession(id) || "");
  const raw = await http<any>(`/api/sessions/${id}/compact`, { method: "POST", body: JSON.stringify({ focus: focus || "" }) });
  return str(pick(raw, "note"));
}

export async function steer(id: string, text: string): Promise<void> {
  const s = await wailsService();
  if (s?.Steer) return s.Steer(id, text);
  await http(`/api/sessions/${id}/steer`, { method: "POST", body: JSON.stringify({ text }) });
}

export async function workspaceTree(workspace: string): Promise<FileHit[]> {
  const s = await wailsService();
  const raw = s?.WorkspaceTree
    ? await s.WorkspaceTree(workspace)
    : await http(`/api/workspace/tree?workspace=${encodeURIComponent(workspace || "")}`);
  return asArray(raw).map((v) => ({ path: str(pick(v, "path", "Path")), kind: str(pick(v, "kind", "Kind"), "file") })).filter((h) => h.path);
}

export async function searchFiles(workspace: string, query: string): Promise<FileHit[]> {
  const s = await wailsService();
  const raw = s?.SearchFiles
    ? await s.SearchFiles(workspace, query)
    : await http(`/api/fs/search?workspace=${encodeURIComponent(workspace || "")}&q=${encodeURIComponent(query)}`);
  return asArray(raw).map((v) => ({ path: str(pick(v, "path", "Path")), kind: str(pick(v, "kind", "Kind"), "file") })).filter((h) => h.path);
}

export async function listSkills(workspace?: string): Promise<SkillInfo[]> {
  const s = await wailsService();
  const raw = s?.ListSkillsFor
    ? await s.ListSkillsFor(workspace || "")
    : s?.ListSkills
      ? await s.ListSkills()
      : await http(`/api/skills${workspace ? `?workspace=${encodeURIComponent(workspace)}` : ""}`);
  return asArray(raw).map((v) => ({
    name: str(pick(v, "name", "Name")),
    description: str(pick(v, "description", "Description")),
    body: str(pick(v, "body", "Body")),
    dir: str(pick(v, "dir", "Dir")),
    source: str(pick(v, "source", "Source")),
    icon: str(pick(v, "icon", "Icon")),
    displayName: str(pick(v, "display_name", "DisplayName", "displayName")),
    files: str(pick(v, "files", "Files")),
    slug: str(pick(v, "slug", "Slug")),
    incomplete: str(pick(v, "incomplete", "Incomplete")) === "1" || bool(pick(v, "incomplete", "Incomplete")),
  })).filter((x) => x.name);
}

export async function getSkill(workspace: string, name: string): Promise<SkillInfo | null> {
  const s = await wailsService();
  const raw = s?.GetSkill
    ? await s.GetSkill(workspace || "", name)
    : await http(`/api/skills?workspace=${encodeURIComponent(workspace || "")}&name=${encodeURIComponent(name)}`);
  const info = {
    name: str(pick(raw, "name", "Name")),
    description: str(pick(raw, "description", "Description")),
    body: str(pick(raw, "body", "Body")),
    dir: str(pick(raw, "dir", "Dir")),
    source: str(pick(raw, "source", "Source")),
    icon: str(pick(raw, "icon", "Icon")),
    displayName: str(pick(raw, "display_name", "DisplayName", "displayName")),
    files: str(pick(raw, "files", "Files")),
    slug: str(pick(raw, "slug", "Slug")),
    incomplete: str(pick(raw, "incomplete", "Incomplete")) === "1" || bool(pick(raw, "incomplete", "Incomplete")),
  };
  return info.name ? info : null;
}

export type SkillMarketItem = {
  slug: string;
  name: string;
  purpose: string;
  prerequisites: string;
  category: string;
  featured: boolean;
  icon?: string;
  displayName?: string;
  version?: string;
  hasScripts?: boolean;
};

export type SkillMarketCatalog = {
  source: string;
  fetchedAt: string;
  items: SkillMarketItem[];
};

export async function skillMarket(refresh = false): Promise<SkillMarketCatalog> {
  const s = await wailsService();
  const raw = s?.SkillMarket
    ? await s.SkillMarket(refresh)
    : await http(`/api/skills/market${refresh ? "?refresh=1" : ""}`);
  return {
    source: str(pick(raw, "source", "Source"), "infometa/workbuddyskills"),
    fetchedAt: str(pick(raw, "fetched_at", "FetchedAt")),
    items: asArray(pick(raw, "items", "Items")).map((v) => ({
      slug: str(pick(v, "slug", "Slug")),
      name: str(pick(v, "name", "Name")),
      purpose: str(pick(v, "purpose", "Purpose")),
      prerequisites: str(pick(v, "prerequisites", "Prerequisites")),
      category: str(pick(v, "category", "Category")),
      featured: bool(pick(v, "featured", "Featured")),
      icon: str(pick(v, "icon", "Icon")),
      displayName: str(pick(v, "display_name", "DisplayName", "displayName")),
      version: str(pick(v, "version", "Version")),
      hasScripts: bool(pick(v, "has_scripts", "HasScripts", "hasScripts")),
    })).filter((x) => x.slug),
  };
}

export type SkillInstallResult = {
  ok: boolean;
  warning?: string;
  files?: string[];
};

export async function installMarketSkill(slug: string): Promise<SkillInstallResult> {
  const s = await wailsService();
  const raw = s?.InstallMarketSkill
    ? await s.InstallMarketSkill(slug)
    : await http("/api/skills/market", { method: "POST", body: JSON.stringify({ slug }) });
  return {
    ok: bool(pick(raw, "ok", "OK")) || raw == null,
    warning: str(pick(raw, "warning", "Warning")),
    files: asArray(pick(raw, "files", "Files")).map((v) => str(v)).filter(Boolean),
  };
}

export async function uninstallMarketSkill(slug: string): Promise<void> {
  const s = await wailsService();
  if (s?.UninstallMarketSkill) {
    await s.UninstallMarketSkill(slug);
    return;
  }
  await http("/api/skills/market", { method: "POST", body: JSON.stringify({ slug, op: "uninstall" }) });
}

export async function openPath(path: string): Promise<void> {
  const s = await wailsService();
  if (s?.OpenPath) return s.OpenPath(path);
  await http("/api/open", { method: "POST", body: JSON.stringify({ path, kind: "path" }) });
}

export async function openInEditor(path: string): Promise<void> {
  const s = await wailsService();
  if (s?.OpenInEditor) return s.OpenInEditor(path);
  await http("/api/open", { method: "POST", body: JSON.stringify({ path, kind: "editor" }) });
}

export async function openWorkspaceTerminal(dir: string): Promise<void> {
  const s = await wailsService();
  if (s?.OpenWorkspaceTerminal) return s.OpenWorkspaceTerminal(dir);
  await http("/api/open", { method: "POST", body: JSON.stringify({ path: dir, kind: "terminal" }) });
}

export async function popOutThread(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.PopOutThread) return s.PopOutThread(id);
}

export async function pickFolder(): Promise<string> {
  const s = await wailsService();
  if (s?.PickFolder) return str(await s.PickFolder());
  return "";
}

export async function pickFiles(): Promise<string[]> {
  const s = await wailsService();
  if (s?.PickFiles) return asArray(await s.PickFiles()).map(String).filter(Boolean);
  return [];
}

export async function startMCP(name: string, command: string, args: string[]): Promise<void> {
  const s = await wailsService();
  if (s?.StartMCP) return s.StartMCP(name, command, args);
  await http("/api/mcp/start", { method: "POST", body: JSON.stringify({ name, command, args }) });
}

export async function startMCPHTTP(name: string, endpoint: string): Promise<void> {
  const s = await wailsService();
  if (s?.StartMCPHTTP) return s.StartMCPHTTP(name, endpoint);
  await http("/api/mcp/http", { method: "POST", body: JSON.stringify({ name, endpoint }) });
}

export async function inboxList(): Promise<any[]> {
  const s = await wailsService();
  if (s?.InboxList) return asArray(await s.InboxList());
  return asArray(await http("/api/inbox"));
}

export async function inboxRead(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.InboxMarkRead) return s.InboxMarkRead(id);
  await http("/api/inbox", { method: "POST", body: JSON.stringify({ id }) });
}

export async function inboxDismiss(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.InboxDismiss) { await s.InboxDismiss(id); return; }
  await http("/api/inbox", { method: "POST", body: JSON.stringify({ id, op: "dismiss" }) });
}

export async function isolationReport(): Promise<any> {
  const s = await wailsService();
  if (s?.IsolationReport) return s.IsolationReport();
  return http("/api/isolation");
}

export async function connectors(): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorsList) {
    return { accounts: asArray(await s.ConnectorsList()), catalog: asArray(s.ConnectorCatalog ? await s.ConnectorCatalog() : []) };
  }
  return http("/api/connectors");
}

export async function connectorConnect(row: any): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorConnect) return s.ConnectorConnect(row);
  return http("/api/connectors", { method: "POST", body: JSON.stringify(row) });
}

export async function connectorAuthURL(provider: string, clientId: string, redirect: string): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorAuthURL) return s.ConnectorAuthURL(provider, clientId, redirect);
  return http("/api/connectors/oauth", { method: "POST", body: JSON.stringify({ provider, client_id: clientId, redirect }) });
}

export async function connectorComplete(provider: string, code: string, clientId = "", secret = "", redirect = "http://127.0.0.1:3080/oauth"): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorComplete) return s.ConnectorComplete(provider, code, clientId, secret, redirect);
  return http("/api/connectors/oauth", {
    method: "POST",
    body: JSON.stringify({ provider, code, client_id: clientId, secret, redirect }),
  });
}

export async function connectorDisconnect(id: string): Promise<boolean> {
  const s = await wailsService();
  if (s?.ConnectorDisconnect) return asBool(await s.ConnectorDisconnect(id));
  if (s) return false;
  const r = await http<any>(`/api/connectors?id=${encodeURIComponent(id)}`, { method: "DELETE" });
  return asBool(pick(r, "ok", "Ok") ?? true);
}

export async function triggerWebhook(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.TriggerWebhook) { await s.TriggerWebhook(id); return; }
  await http(`/api/hooks/${encodeURIComponent(id)}`, { method: "POST" });
}

export async function memoryList(q = "", kind = ""): Promise<any[]> {
  const s = await wailsService();
  if (s?.MemoryList) return asArray(await s.MemoryList(q, kind));
  const qs = new URLSearchParams({ q, kind });
  return asArray(await http(`/api/memory?${qs}`));
}

export async function memoryPromote(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.MemoryPromote) { await s.MemoryPromote(id); return; }
  await http("/api/memory", { method: "POST", body: JSON.stringify({ op: "promote", id }) });
}

export async function memoryForget(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.MemoryForget) { await s.MemoryForget(id); return; }
  await http("/api/memory", { method: "POST", body: JSON.stringify({ op: "forget", id }) });
}

export async function memoryWrite(kind: string, text: string, project = ""): Promise<any> {
  const s = await wailsService();
  if (s?.MemoryWrite) return s.MemoryWrite(kind, text, project);
  return http("/api/memory", { method: "POST", body: JSON.stringify({ kind, text, project }) });
}

export async function scheduleList(): Promise<any[]> {
  const s = await wailsService();
  if (s?.ScheduleList) return asArray(await s.ScheduleList());
  return asArray(await http("/api/schedule"));
}

export async function scheduleCreate(job: any): Promise<any> {
  const s = await wailsService();
  if (s?.ScheduleCreate) return s.ScheduleCreate(job);
  return http("/api/schedule", { method: "POST", body: JSON.stringify(job) });
}

export async function scheduleCancel(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.ScheduleCancel) { await s.ScheduleCancel(id); return; }
  await http(`/api/schedule?id=${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function projectsList(): Promise<any[]> {
  const s = await wailsService();
  if (s?.ProjectsList) return asArray(await s.ProjectsList());
  return asArray(await http("/api/projects"));
}

export async function projectCreate(p: any): Promise<any> {
  const s = await wailsService();
  if (s?.ProjectCreate) return s.ProjectCreate(p);
  return http("/api/projects", { method: "POST", body: JSON.stringify(p) });
}

export async function reviewQueue(): Promise<any> {
  const s = await wailsService();
  if (s?.ReviewQueue) return s.ReviewQueue();
  return http("/api/review/queue");
}

export type BrowserView = {
  url: string;
  title: string;
  lane: string;
  profile: string;
  headed: boolean;
  live: boolean;
  text: string;
  screenshot: string;
  log: { op?: string; detail?: string; url?: string; ts?: string }[];
};

export async function browserView(): Promise<BrowserView> {
  const s = await wailsService();
  const raw = s?.BrowserView ? await s.BrowserView() : await http("/api/browser/view");
  return {
    url: str(pick(raw, "url", "URL")),
    title: str(pick(raw, "title", "Title")),
    lane: str(pick(raw, "lane", "Lane"), "isolated"),
    profile: str(pick(raw, "profile", "Profile")),
    headed: bool(pick(raw, "headed", "Headed")),
    live: bool(pick(raw, "live", "Live")),
    text: str(pick(raw, "text", "Text")),
    screenshot: str(pick(raw, "screenshot", "Screenshot")),
    log: asArray(pick(raw, "log", "Log")).map((row: any) => ({
      op: str(pick(row, "op")),
      detail: str(pick(row, "detail")),
      url: str(pick(row, "url")),
      ts: str(pick(row, "ts", "TS")),
    })),
  };
}

export async function browserTakeover(): Promise<void> {
  const s = await wailsService();
  if (s?.BrowserTakeover) {
    await s.BrowserTakeover();
    return;
  }
  await http("/api/browser/takeover", { method: "POST", body: "{}" });
}

export async function clipboardRead(): Promise<string> {
  const s = await wailsService();
  if (s?.ClipboardRead) return String(await s.ClipboardRead() || "");
  return "";
}

export async function captureScreenshot(): Promise<string> {
  const s = await wailsService();
  if (s?.Screenshot) {
    await s.Screenshot("");
    return ".yoyo/captures/shot.png";
  }
  return "";
}

export async function admitExpert(dir: string): Promise<any> {
  const s = await wailsService();
  if (s?.AdmitExpert) return s.AdmitExpert(dir);
  return http("/api/expert/admit", { method: "POST", body: JSON.stringify({ dir }) });
}

export async function distillExpert(name: string, description: string, body: string): Promise<any> {
  const s = await wailsService();
  if (s?.DistillExpert) return s.DistillExpert(name, description, body);
  return http("/api/expert/distill", { method: "POST", body: JSON.stringify({ name, description, body }) });
}

export async function stopMCP(name: string): Promise<void> {
  const s = await wailsService();
  if (s?.StopMCP) return s.StopMCP(name);
  await http("/api/mcp/stop", { method: "POST", body: JSON.stringify({ name }) });
}

export async function about(): Promise<any> {
  const s = await wailsService();
  if (s?.About) return s.About();
  return http("/api/about");
}

export async function doctor(): Promise<any> {
  const s = await wailsService();
  if (s?.Doctor) return s.Doctor();
  return http("/api/doctor");
}

export async function logs(limit = 80): Promise<any> {
  const s = await wailsService();
  if (s?.Logs) return s.Logs(limit);
  return http("/api/logs");
}

export async function journal(limit = 80): Promise<any> {
  const s = await wailsService();
  if (s?.Journal) return s.Journal(limit);
  return http("/api/journal");
}

export async function exportDiagnostics(): Promise<string> {
  const s = await wailsService();
  if (s?.ExportDiagnostics) {
    const path = await s.ExportDiagnostics();
    return String(path || "");
  }
  const r = await http<{ path?: string }>("/api/logs/export", { method: "POST", body: "{}" });
  return String(r?.path || "");
}

export async function diagnoseSession(session: string): Promise<string> {
  const s = await wailsService();
  if (s?.DiagnoseSession) return String((await s.DiagnoseSession(session)) || "");
  const r = await http<{ text?: string }>("/api/logs/diagnose", { method: "POST", body: JSON.stringify({ session }) });
  return String(r?.text || "");
}

export async function reportFrontend(events: { ts?: string; level: string; msg: string; stack?: string; source?: string; session_id?: string }[]): Promise<void> {
  if (!events.length) return;
  const s = await wailsService();
  if (s?.FrontendLogs) {
    await s.FrontendLogs(events);
    return;
  }
  await http("/api/logs", { method: "POST", body: JSON.stringify({ events }) });
}

export async function checkUpdate(): Promise<any> {
  const s = await wailsService();
  if (s?.CheckUpdate) return s.CheckUpdate();
  return http("/api/update");
}

export async function testProvider(): Promise<any> {
  const s = await wailsService();
  if (s?.TestProvider) return s.TestProvider();
  return http("/api/provider/test", { method: "POST", body: "{}" });
}

export async function replaceMCP(servers: { name: string; command: string; args: string[]; endpoint?: string }[]): Promise<void> {
  const s = await wailsService();
  if (s?.ReplaceMCP) return s.ReplaceMCP(servers);
  await http("/api/mcp/replace", { method: "POST", body: JSON.stringify({ servers }) });
}

export async function revealLogs(): Promise<void> {
  const s = await wailsService();
  if (s?.RevealLogs) return s.RevealLogs();
}

export async function revealJournal(): Promise<void> {
  const s = await wailsService();
  if (s?.RevealJournal) return s.RevealJournal();
}

export async function keyStatus(): Promise<any> {
  const s = await wailsService();
  if (s?.KeyStatus) return s.KeyStatus();
  return http("/api/key/status");
}

export async function quit(): Promise<void> {
  if (import.meta.env.VITE_E2E) return;
  try {
    const mod: any = await import("@wailsio/runtime");
    if (mod.Events?.Emit) {
      mod.Events.Emit("yoyo:do-quit");
      return;
    }
  } catch {
    /* browser */
  }
  window.close();
}

export { errMessage };

export async function videoCall(method: string, params: Record<string, any> = {}): Promise<any> {
  const s = await wailsService();
  const fn = svcMethod(s, "VideoCall", "videoCall");
  if (fn) return fn(method, params);
  let res: Response;
  try {
    res = await fetch("/api/rpc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
      signal: AbortSignal.timeout(120000),
    });
  } catch (e) {
    throw new Error(`${method}: ${errMessage(e)}`);
  }
  const text = await res.text();
  let body: any = text;
  try {
    body = text ? JSON.parse(text) : {};
  } catch {
    /* keep */
  }
  if (!res.ok) throw new ApiError(res.status, body, `${method}: ${typeof body === "object" && body?.error ? body.error : text || res.statusText}`);
  if (body?.error) throw new ApiError(500, body, String(body.error.message || body.error));
  return body?.result ?? body;
}

export const video = {
  status: () => videoCall("video.status"),
  modes: () => videoCall("video.modes"),
  templates: () => videoCall("video.templates"),
  providers: (serviceType = "") => videoCall("video.providers.list", { service_type: serviceType }),
  upsertProvider: (provider: any, apiKey = "") => videoCall("video.providers.upsert", { provider, api_key: apiKey }),
  deleteProvider: (id: string) => videoCall("video.providers.delete", { id }),
  testProvider: (id: string) => videoCall("video.providers.test", { id }),
  styles: () => videoCall("video.styles"),
  allStyles: () => videoCall("video.styles.all"),
  upsertStyle: (s: Record<string, any>) => videoCall("video.styles.upsert", s),
  deleteStyle: (id: string) => videoCall("video.styles.delete", { id }),
  settings: () => videoCall("video.settings.get"),
  setSetting: (key: string, value: string) => videoCall("video.settings.set", { key, value }),
  jobs: (episodeId = "") => videoCall("video.jobs.list", { episode_id: episodeId }),
  cancelJob: (id: string) => videoCall("video.jobs.cancel", { id }),
  retryJob: (id: string) => videoCall("video.jobs.retry", { id }),
  applyJob: (id: string) => videoCall("video.jobs.apply", { id }),
  listDramas: () => videoCall("drama.list"),
  createDrama: (d: Record<string, any>) => videoCall("drama.create", d),
  getDrama: (id: string) => videoCall("drama.get", { id }),
  updateDrama: (d: Record<string, any>) => videoCall("drama.update", d),
  deleteDrama: (id: string) => videoCall("drama.delete", { id }),
  episodes: (dramaId: string) => videoCall("drama.episodes", { drama_id: dramaId }),
  createEpisode: (dramaId: string, title: string, content: string) => videoCall("drama.episode.create", { drama_id: dramaId, title, content }),
  updateEpisode: (ep: Record<string, any>) => videoCall("drama.episode.update", ep),
  deleteEpisode: (id: string) => videoCall("drama.episode.delete", { id }),
  bundle: (episodeId: string) => videoCall("drama.bundle", { episode_id: episodeId }),
  bind: (sessionId: string, episodeId: string) => videoCall("drama.bind", { session_id: sessionId, episode_id: episodeId }),
  stage: (sessionId: string, episodeId: string, stage: string) => videoCall("drama.stage.run", { session_id: sessionId, episode_id: episodeId, stage }),
  saveAsset: (kind: string, id: string, fields: Record<string, any>) => videoCall("drama.assets.save", { kind, id, ...fields }),
  createAsset: (kind: string, episodeId: string, fields: Record<string, any>) => videoCall("drama.assets.create", { kind, episode_id: episodeId, ...fields }),
  deleteAsset: (kind: string, id: string) => videoCall("drama.assets.delete", { kind, id }),
  generateAsset: (kind: string, id: string, episodeId = "") => videoCall("drama.assets.generate", { kind, id, episode_id: episodeId }),
  generateMissingAssets: (episodeId: string) => videoCall("drama.assets.generate_missing", { episode_id: episodeId }),
  uploadAsset: (kind: string, id: string, dataB64: string) => videoCall("drama.assets.upload", { kind, id, data_b64: dataB64 }),
  saveShots: (episodeId: string, shots: any[], replace = false) => videoCall("drama.shots.save", { episode_id: episodeId, shots, replace }),
  updateShot: (s: Record<string, any>) => videoCall("drama.shots.update", s),
  deleteShot: (id: string) => videoCall("drama.shots.delete", { id }),
  generateShot: (id: string) => videoCall("drama.shots.generate", { id }),
  generateMissingShots: (episodeId: string) => videoCall("drama.shots.generate_missing", { episode_id: episodeId }),
  merge: (episodeId: string, shotIds: string[]) => videoCall("drama.merge", { episode_id: episodeId, shot_ids: shotIds }),
  importHuobao: (dbPath: string, staticDir = "") => videoCall("drama.import", { db_path: dbPath, static_dir: staticDir }),
  skipRewrite: (episodeId: string) => videoCall("drama.skip_rewrite", { episode_id: episodeId }),
  canvasHttp: (method: string, path: string, extra: Record<string, any> = {}) => videoCall("canvas.http", { method, path, ...extra }),
  canvasBind: (sessionId: string, projectId: string) => videoCall("canvas.bind", { session_id: sessionId, project_id: projectId }),
  history: () => videoCall("video.history"),
  sessionProject: (sessionId: string) => videoCall("video.session.project", { session_id: sessionId }),
};
