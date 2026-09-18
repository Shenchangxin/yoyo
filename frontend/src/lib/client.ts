import { asArray, asBool, bool, boolOr, errMessage, num, pick, str } from "./normalize";
import { getLocale } from "./i18n";
import type { AppConfig, Approval, Attachment, ContextUsage, FileHit, Health, Hunk, SessionTrace, SkillInfo, SpillBlob, Thread, TraceArtifact, TraceEvent, TraceStats } from "./protocol";

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

export function threadOf(v: any): Thread {
  return {
    id: str(pick(v, "id", "ID")),
    title: str(pick(v, "title", "Title"), "thread"),
    workspace: str(pick(v, "workspace", "Workspace")),
    harness: str(pick(v, "harness", "Harness")),
    createdAt: str(pick(v, "created_at", "CreatedAt")),
    archived: bool(pick(v, "archived", "Archived")),
    pinned: bool(pick(v, "pinned", "Pinned")),
    model: str(pick(v, "model", "Model")),
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
    budgetUsd: num(pick(v, "budget_usd", "budgetUsd")),
    usageUsd: num(pick(usage, "usd")),
    workspaceReady: bool(pick(v, "workspace_ready", "workspaceReady")),
  };
}

export function configOf(v: any): AppConfig {
  return {
    provider: str(pick(v, "provider"), "openai"),
    model: str(pick(v, "model")),
    baseUrl: str(pick(v, "base_url", "BaseURL", "baseUrl")),
    workspace: str(pick(v, "workspace", "Workspace")),
    autoAllow: bool(pick(v, "auto_allow", "AutoAllow", "autoAllow")),
    maxBudgetUsd: num(pick(v, "max_budget_usd", "MaxBudgetUSD", "maxBudgetUsd")),
    usdPerMtok: num(pick(v, "usd_per_mtok", "USDPerMTok", "usdPerMtok")),
    models: asArray(pick(v, "models", "Models")).map(String).filter(Boolean),
    closeToTray: bool(pick(v, "close_to_tray", "CloseToTray", "closeToTray")),
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
    auto_allow: cfg.autoAllow,
    max_budget_usd: cfg.maxBudgetUsd,
    usd_per_mtok: cfg.usdPerMtok,
    models: cfg.models,
    close_to_tray: cfg.closeToTray,
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

export async function setAPIKey(value: string): Promise<void> {
  const s = await wailsService();
  if (s?.SetAPIKey) return s.SetAPIKey(value);
  await http("/api/key", { method: "POST", body: JSON.stringify({ value }) });
}

export async function createSession(workspace: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.CreateSession
    ? await s.CreateSession(workspace)
    : await http("/api/sessions", { method: "POST", body: JSON.stringify({ workspace }) });
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

export async function queueList(sessionID: string): Promise<{ text?: string }[]> {
  const s = await wailsService();
  if (s?.QueueList) return asArray(await s.QueueList(sessionID));
  if (s) return [];
  return asArray(await http(`/api/sessions/${sessionID}/queue`));
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

export async function evolve(k = 3, opts?: { rounds?: number; sealed?: boolean; promote?: boolean }): Promise<any> {
  const body = { k, rounds: opts?.rounds ?? 0, sealed: !!opts?.sealed, promote: !!opts?.promote };
  const s = await wailsService();
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

export async function forkSession(id: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ForkSession
    ? await s.ForkSession(id)
    : await http(`/api/sessions/${id}/fork`, { method: "POST" });
  return threadOf(raw);
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

export async function compactSession(id: string): Promise<string> {
  const s = await wailsService();
  if (s?.CompactSession) return String(await s.CompactSession(id) || "");
  const raw = await http<any>(`/api/sessions/${id}/compact`, { method: "POST" });
  return str(pick(raw, "note"));
}

export async function steer(id: string, text: string): Promise<void> {
  const s = await wailsService();
  if (s?.Steer) return s.Steer(id, text);
  await http(`/api/sessions/${id}/steer`, { method: "POST", body: JSON.stringify({ text }) });
}

export async function searchFiles(workspace: string, query: string): Promise<FileHit[]> {
  const s = await wailsService();
  const raw = s?.SearchFiles
    ? await s.SearchFiles(workspace, query)
    : await http(`/api/fs/search?workspace=${encodeURIComponent(workspace || "")}&q=${encodeURIComponent(query)}`);
  return asArray(raw).map((v) => ({ path: str(pick(v, "path", "Path")), kind: str(pick(v, "kind", "Kind"), "file") })).filter((h) => h.path);
}

export async function listSkills(): Promise<SkillInfo[]> {
  const s = await wailsService();
  const raw = s?.ListSkills ? await s.ListSkills() : await http("/api/skills");
  return asArray(raw).map((v) => ({ name: str(pick(v, "name", "Name")), description: str(pick(v, "description", "Description")) })).filter((x) => x.name);
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
