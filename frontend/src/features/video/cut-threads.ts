/**
 * 本文件拥有剪辑工程和聊天会话在 localStorage 里的对照。不拥有聊天内容。
 * 它只改本机存储，不改服务端。
 */
const KEY = "yoyo-cut-threads";

/** 读出工程 id 到会话 id 的对照，坏数据当成空。 */
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

/** cutThreadId 返回这个剪辑工程记下的会话，没有就是空字符串。projectId 去掉空白后当作对照表的键。 */
export function cutThreadId(projectId: string): string {
  return read()[(projectId || "").trim()] || "";
}

/** isCutThread 判断这个会话是不是某条剪辑记下的。sessionId 去掉空白后看是否出现在对照表的值里，空的直接返回假。 */
export function isCutThread(sessionId: string): boolean {
  const id = (sessionId || "").trim();
  if (!id) return false;
  return Object.values(read()).includes(id);
}

/** rememberCutThread 一条剪辑记一条聊天。会话挪到新剪辑时离开上一条。projectId 去掉空白后当作新键，并清掉同键旧项。sessionId 去掉空白后写到这个键上，并清掉别的键上相同的会话。 */
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
    /* 忽略 */
  }
}
