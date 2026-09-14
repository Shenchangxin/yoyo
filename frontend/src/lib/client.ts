export type Health = { ok: boolean; harness?: string; model?: string };

async function wailsService(): Promise<any | null> {
  try {
    const mod = await import("../../bindings/github.com/Shenchangxin/yoyo/internal/desktop/service.js");
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
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.json();
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

export async function send(sessionID: string, text: string): Promise<string> {
  const s = await wailsService();
  if (s?.Send) return s.Send(sessionID, text);
  const r = await http<{ text: string }>(`/api/sessions/${sessionID}/messages`, {
    method: "POST",
    body: JSON.stringify({ text }),
  });
  return r.text;
}

export async function harness(): Promise<any> {
  const s = await wailsService();
  if (s?.Harness) return s.Harness();
  return http("/api/harness");
}

export async function checkout(hash: string): Promise<void> {
  const s = await wailsService();
  if (s?.Checkout) return s.Checkout(hash);
  await http("/api/harness/checkout", { method: "POST", body: JSON.stringify({ hash }) });
}

export async function rollback(): Promise<void> {
  const s = await wailsService();
  if (s?.Rollback) return s.Rollback();
  await http("/api/harness/rollback", { method: "POST" });
}

export async function runEval(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEval) return s.RunEval();
  return http("/api/eval", { method: "POST" });
}

export async function evolve(): Promise<any> {
  const s = await wailsService();
  if (s?.Evolve) return s.Evolve();
  return http("/api/evolve", { method: "POST" });
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
