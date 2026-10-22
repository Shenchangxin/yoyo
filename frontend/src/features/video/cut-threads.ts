const KEY = "yoyo-cut-threads";

function read(): Record<string, string> {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) || "{}");
    if (!raw || typeof raw !== "object") return {};
    const out: Record<string, string> = {};
    for (const [id, sessionId] of Object.entries(raw as Record<string, unknown>)) {
      const sid = String(sessionId || "").trim();
      if (id.trim() && sid) out[id.trim()] = sid;
    }
    return out;
  } catch {
    return {};
  }
}

export function cutThreadId(projectId: string): string {
  return read()[(projectId || "").trim()] || "";
}

export function isCutThread(sessionId: string): boolean {
  const id = (sessionId || "").trim();
  if (!id) return false;
  return Object.values(read()).includes(id);
}

/** One chat per cut. A session that moves to a new cut leaves the previous one. */
export function rememberCutThread(projectId: string, sessionId: string) {
  const id = (projectId || "").trim();
  const sid = (sessionId || "").trim();
  if (!id || !sid) return;
  const next: Record<string, string> = {};
  for (const [k, v] of Object.entries(read())) {
    if (k === id || v === sid) continue;
    next[k] = v;
  }
  next[id] = sid;
  try {
    localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    /* ignore */
  }
}
