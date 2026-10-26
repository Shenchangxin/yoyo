/**
 * 拥有：从 Wails 和 JSON 的松散形状里取出字段、数组、字符串、数字、布尔和错误正文。
 * 不拥有：具体接口的类型。
 */

/** pick 按顺序返回对象上第一个既不是 undefined 也不是 null 的字段。不是对象时返回 undefined。 */
export function pick(v: any, ...keys: string[]): any {
  if (v == null || typeof v !== "object") return undefined;
  for (const k of keys) {
    if (v[k] !== undefined && v[k] !== null) return v[k];
  }
  return undefined;
}

/** asArray 本身是数组则用它。否则依次取 result、sessions、events、Events、hunks。都没有则空数组。 */
export function asArray(value: any): any[] {
  if (Array.isArray(value)) return value;
  if (value == null) return [];
  if (typeof value !== "object") return [];
  if (Array.isArray(value.result)) return value.result;
  if (Array.isArray(value.sessions)) return value.sessions;
  if (Array.isArray(value.events)) return value.events;
  if (Array.isArray(value.Events)) return value.Events;
  if (Array.isArray(value.hunks)) return value.hunks;
  return [];
}

/** str 转成字符串。null 或 undefined 用回退。 */
export function str(v: any, fallback = ""): string {
  if (v == null) return fallback;
  return String(v);
}

/** num 转成有限数字。否则用回退。 */
export function num(v: any, fallback = 0): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

/** asBool 收紧 Wails 和 JSON 的真值包装。Promise 或光秃对象不当成真。数字非 0 为真。字符串认 true、1、yes。对象只认 running、result、value；仅有 ok 且没有 running 时为假。 */
export function asBool(v: any): boolean {
  if (typeof v === "boolean") return v;
  if (typeof v === "number") return v !== 0;
  if (typeof v === "string") {
    const s = v.trim().toLowerCase();
    return s === "true" || s === "1" || s === "yes";
  }
  if (v && typeof v === "object") {
    if (typeof v.running === "boolean") return v.running;
    if (typeof v.result === "boolean") return v.result;
    if (typeof v.value === "boolean") return v.value;
    if (typeof v.ok === "boolean" && v.running === undefined) return false;
  }
  return false;
}

/** bool 与 asBool 相同。 */
export function bool(v: any): boolean {
  return asBool(v);
}

/** boolOr null 或 undefined 用回退，其余走 asBool。 */
export function boolOr(v: any, fallback: boolean): boolean {
  if (v === undefined || v === null) return fallback;
  return asBool(v);
}

/** errMessage 取出错误正文，原因还没包含在正文里时接上。没有错误时为 unknown error。 */
export function errMessage(e: any): string {
  if (!e) return "unknown error";
  if (typeof e === "string") return e;
  const msg = e.message || e.error || e.Message;
  const cause = e.cause != null ? errMessage(e.cause) : "";
  const primary = msg != null && msg !== e ? String(msg) : String(e);
  if (cause && cause !== primary && !primary.includes(cause)) return `${primary}: ${cause}`;
  return primary;
}
