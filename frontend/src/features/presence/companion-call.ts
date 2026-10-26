/**
 * 拥有：判断 yoyo:call 载荷是否表示陪伴通话开着。Wails 送来的是带 name 和 data 的信封。
 * 不拥有：通话循环。只读事件，不改服务端。
 */

/** companionCallOn 剥开信封后判断陪伴通话是不是开着。真值、字符串 true 或 1，以及对象上的 on 为真时算开着。raw 传给 peel。 */
export function companionCallOn(raw: unknown): boolean {
  const data = peel(raw);
  if (data === true || data === "true" || data === 1 || data === "1") return true;
  if (data && typeof data === "object" && !Array.isArray(data)) {
    const on = (data as { on?: unknown }).on;
    return on === true || on === "true";
  }
  return false;
}

/** 剥掉字符串、单元素数组，以及带 name 和 data 的信封。raw 去掉首尾空白，取第一段，判断是否为空，与 null 比较是否不等，读取 length、data，用 typeof 看类型，作为分支结果，传给 peel。depth 大于 4，加上 1。 */
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
