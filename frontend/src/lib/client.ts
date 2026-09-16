import { asArray, asBool, bool, errMessage, num, pick, str } from "./normalize";
import type { AppConfig, Approval, Attachment, ContextUsage, FileHit, Health, Hunk, SkillInfo, Thread } from "./protocol";

export class ApiError extends Error {
  status: number;
  body: any;
  constructor(status: number, body: any, message: string) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

async function wailsService(): Promise<any | null> {
  if (import.meta.env.VITE_E2E) return null;
  if (typeof window !== "undefined" && !(window as any)._wails?.environment?.OS) return null;
  try {
    const spec = "../../bindings/github.com/Shenchangxin/yoyo/internal/desktop/service.js";
    const mod = await import(/* @vite-ignore */ spec);
    const svc = (mod as any).Service ?? (mod as any).default ?? mod;
    if (svc && typeof (svc.ListSessions || svc.Health) === "function") return svc;
    return null;
  } catch {
    return null;
  }
}

async function http<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...init,
    signal: init?.signal ?? AbortSignal.timeout(8000),
  });
  const text = await res.text();
  let body: any = text;
  try {
    body = text ? JSON.parse(text) : {};
  } catch {
    /* keep text */
  }
  if (!res.ok) {
    const msg = typeof body === "object" && body?.error ? body.error : text || res.statusText;
    throw new ApiError(res.status, body, msg);
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
  };
}

export function approvalOf(v: any): Approval {
  const req = pick(v, "request", "Request") || {};
  return {
    id: str(pick(v, "id", "ID")),
    action: str(pick(req, "action", "Action")),
    command: str(pick(req, "command", "Command")),
    path: str(pick(req, "path", "Path")),
    level: str(pick(req, "level", "Level")),
    sessionId: str(pick(req, "session_id", "SessionID", "sessionId")),
  };
}

export function contextOf(v: any): ContextUsage {
  return {
    tokens: num(pick(v, "tokens", "Tokens")),
    budget: num(pick(v, "budget", "Budget")),
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
    keymap: cfg.keymap || {},
  };
  const s = await wailsService();
  if (s?.SetConfig) return s.SetConfig(body);
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

export async function send(sessionID: string, text: string, opts?: { plan?: boolean; attachments?: Attachment[] }): Promise<{ queued?: boolean }> {
  const s = await wailsService();
  if (s?.StartSendOpts) {
    await s.StartSendOpts(sessionID, text, !!opts?.plan, opts?.attachments || []);
    return {};
  }
  if (s?.StartSend) {
    await s.StartSend(sessionID, text, !!opts?.plan);
    return {};
  }
  const raw = await http<any>(`/api/sessions/${sessionID}/messages`, {
    method: "POST",
    body: JSON.stringify({ text, async: true, plan: !!opts?.plan, attachments: opts?.attachments || [] }),
  });
  return { queued: !!raw?.queued };
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

export async function evolve(k = 3): Promise<any> {
  const s = await wailsService();
  if (s?.EvolveK) return s.EvolveK(k);
  if (s?.Evolve && k === 3) return s.Evolve();
  return http("/api/evolve", { method: "POST", body: JSON.stringify({ k }) });
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
  if (s?.RenameSession) return s.RenameSession(id, title);
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
    if (!window.confirm(`L3 gate: changing ${surfaces}. Confirm checkout?`)) throw e;
    await checkout(hash, true);
  }
}

export async function deleteSession(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.DeleteSession) return s.DeleteSession(id);
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
