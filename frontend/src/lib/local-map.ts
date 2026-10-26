/**
 * 拥有：本地存储里的布尔映射，用来记住展开和折叠。
 * 不拥有：哪一块界面在用这份映射。
 */

/** readLocalMap 读出布尔映射。键为空、缺失、损坏，或值不是布尔时，跳过或返回空对象。 */
export function readLocalMap(key: string): Record<string, boolean> {
  if (!key) return {};
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return {};
    const o = JSON.parse(raw) as unknown;
    if (!o || typeof o !== "object" || Array.isArray(o)) return {};
    const out: Record<string, boolean> = {};
    for (const [k, v] of Object.entries(o as Record<string, unknown>)) {
      if (typeof v === "boolean") out[k] = v;
    }
    return out;
  } catch {
    return {};
  }
}

/** writeLocalMap 写回映射。键为空或配额不足时忽略。 */
export function writeLocalMap(key: string, value: Record<string, boolean>) {
  if (!key) return;
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    /* 配额不足时忽略 */
  }
}
