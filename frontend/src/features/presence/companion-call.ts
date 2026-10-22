/** Whether a `yoyo:call` payload means the companion call is open. Wails delivers `{ name, data }`. */
export function companionCallOn(raw: unknown): boolean {
  const data = peel(raw);
  if (data === true || data === "true" || data === 1 || data === "1") return true;
  if (data && typeof data === "object" && !Array.isArray(data)) {
    const on = (data as { on?: unknown }).on;
    return on === true || on === "true";
  }
  return false;
}

function peel(raw: unknown, depth = 0): unknown {
  if (raw == null || depth > 4) return raw;
  if (typeof raw === "string") {
    const t = raw.trim();
    if ((t.startsWith("{") && t.endsWith("}")) || (t.startsWith("[") && t.endsWith("]"))) {
      try { return peel(JSON.parse(t), depth + 1); } catch { return raw; }
    }
    return raw;
  }
  if (Array.isArray(raw)) return raw.length === 1 ? peel(raw[0], depth + 1) : raw;
  if (typeof raw === "object" && raw !== null && "data" in raw && "name" in raw) {
    return peel((raw as { data: unknown }).data, depth + 1);
  }
  return raw;
}
