export function pick(v: any, ...keys: string[]): any {
  if (v == null || typeof v !== "object") return undefined;
  for (const k of keys) {
    if (v[k] !== undefined && v[k] !== null) return v[k];
  }
  return undefined;
}

export function asArray(value: any): any[] {
  if (Array.isArray(value)) return value;
  if (value == null) return [];
  if (typeof value !== "object") return [];
  if (Array.isArray(value.result)) return value.result;
  if (Array.isArray(value.sessions)) return value.sessions;
  if (Array.isArray(value.events)) return value.events;
  if (Array.isArray(value.hunks)) return value.hunks;
  return [];
}

export function str(v: any, fallback = ""): string {
  if (v == null) return fallback;
  return String(v);
}

export function num(v: any, fallback = 0): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

/** Coerce Wails/JSON truthy wrappers. Never treat a Promise or bare object as true. */
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

export function bool(v: any): boolean {
  return asBool(v);
}

export function boolOr(v: any, fallback: boolean): boolean {
  if (v === undefined || v === null) return fallback;
  return asBool(v);
}

export function errMessage(e: any): string {
  if (!e) return "unknown error";
  if (typeof e === "string") return e;
  return String(e.message || e.error || e);
}
