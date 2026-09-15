export type Health = {
  ok: boolean;
  harness?: string;
  model?: string;
  version?: string;
  usage?: { usd?: number; input_tokens?: number; output_tokens?: number };
  budget_usd?: number;
};

class ApiError extends Error {
  status: number;
  body: any;
  constructor(status: number, body: any, message: string) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

async function wailsService(): Promise<any | null> {
  try {
    const spec = "../../bindings/github.com/Shenchangxin/yoyo/internal/desktop/service.js";
    const mod = await import(/* @vite-ignore */ spec);
    return (mod as any).Service ?? (mod as any).default ?? null;
  } catch {
    return null;
  }
}

async function http<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...init,
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

export async function health(): Promise<Health> {
  const s = await wailsService();
  if (s?.Health) return s.Health();
  return http("/api/health");
}

export async function getConfig(): Promise<any> {
  const s = await wailsService();
  if (s?.GetConfig) return s.GetConfig();
  return http("/api/config");
}

export async function setConfig(cfg: any): Promise<void> {
  const s = await wailsService();
  if (s?.SetConfig) return s.SetConfig(cfg);
  await http("/api/config", { method: "POST", body: JSON.stringify(cfg) });
}

export async function setAPIKey(value: string): Promise<void> {
  const s = await wailsService();
  if (s?.SetAPIKey) return s.SetAPIKey(value);
  await http("/api/key", { method: "POST", body: JSON.stringify({ value }) });
}

export async function createSession(workspace: string): Promise<any> {
  const s = await wailsService();
  if (s?.CreateSession) return s.CreateSession(workspace);
  return http("/api/sessions", { method: "POST", body: JSON.stringify({ workspace }) });
}

export async function listSessions(): Promise<any[]> {
  const s = await wailsService();
  if (s?.ListSessions) return s.ListSessions();
  return http("/api/sessions");
}

export async function trajectory(id: string): Promise<any[]> {
  const s = await wailsService();
  if (s?.Trajectory) return s.Trajectory(id);
  return http(`/api/sessions/${id}/trajectory`);
}

export async function send(sessionID: string, text: string, opts?: { async?: boolean; plan?: boolean }): Promise<string> {
  const s = await wailsService();
  if (opts?.async) {
    if (s?.StartSend) {
      await s.StartSend(sessionID, text, !!opts.plan);
      return "";
    }
    await http(`/api/sessions/${sessionID}/messages`, {
      method: "POST",
      body: JSON.stringify({ text, async: true, plan: !!opts.plan }),
    });
    return "";
  }
  if (s?.Send) return s.Send(sessionID, text);
  const r = await http<{ text: string }>(`/api/sessions/${sessionID}/messages`, {
    method: "POST",
    body: JSON.stringify({ text, plan: !!opts?.plan }),
  });
  return r.text;
}

export async function running(sessionID: string): Promise<boolean> {
  const s = await wailsService();
  if (s?.Running) return !!s.Running(sessionID);
  const r = await http<{ running?: boolean }>(`/api/sessions/${sessionID}/running`);
  return !!r.running;
}

export async function contextUsage(sessionID: string): Promise<any> {
  const s = await wailsService();
  if (s?.ContextUsage) return s.ContextUsage(sessionID);
  return http(`/api/sessions/${sessionID}/context`);
}

export async function interrupt(sessionID: string): Promise<void> {
  const s = await wailsService();
  if (s?.Interrupt) return s.Interrupt(sessionID);
  await http(`/api/sessions/${sessionID}/interrupt`, { method: "POST" });
}

export async function approvals(): Promise<any[]> {
  const s = await wailsService();
  if (s?.PendingApprovals) return s.PendingApprovals();
  return http("/api/approvals");
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
  if (s?.Archive) return s.Archive();
  return http("/api/archive");
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

export async function forkSession(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.ForkSession) return s.ForkSession(id);
  return http(`/api/sessions/${id}/fork`, { method: "POST" });
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

export async function workspaceDiff(workspace: string): Promise<string> {
  const s = await wailsService();
  if (s?.WorkspaceDiff) return s.WorkspaceDiff(workspace);
  const r = await http<{ diff?: string }>(`/api/workspace/diff?workspace=${encodeURIComponent(workspace || "")}`);
  return r.diff || "";
}

export async function runEvalSafety(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalSafety) return s.RunEvalSafety();
  return http("/api/eval/safety", { method: "POST" });
}

export async function workspaceHunks(workspace: string): Promise<{ diff?: string; hunks?: any[] }> {
  const s = await wailsService();
  if (s?.WorkspaceHunks) return s.WorkspaceHunks(workspace);
  return http(`/api/workspace/hunks?workspace=${encodeURIComponent(workspace || "")}`);
}

export async function applyHunks(workspace: string, ids: string[]): Promise<void> {
  const s = await wailsService();
  if (s?.ApplyHunks) return s.ApplyHunks(workspace, ids);
  await http("/api/workspace/hunks", { method: "POST", body: JSON.stringify({ workspace, ids }) });
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
