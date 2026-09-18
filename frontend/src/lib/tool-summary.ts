import type { Item } from "./protocol";

export function parseMaybeJSON(raw: unknown): Record<string, unknown> {
  if (raw && typeof raw === "object" && !Array.isArray(raw)) return raw as Record<string, unknown>;
  if (typeof raw !== "string") return {};
  const t = raw.trim();
  if (!t) return {};
  try {
    const o = JSON.parse(t) as unknown;
    if (o && typeof o === "object" && !Array.isArray(o)) return o as Record<string, unknown>;
  } catch {
    /* not JSON */
  }
  return {};
}

export function toolName(item: Item): string {
  return String(item.name || item.payload?.name || "tool");
}

export function toolArgs(item: Item): Record<string, unknown> {
  return parseMaybeJSON(item.payload?.arguments ?? item.text);
}

export function toolResultBody(item: Item): string {
  const c = item.payload?.content ?? item.text;
  if (typeof c === "string") return c;
  if (c == null) return "";
  try {
    return JSON.stringify(c, null, 2);
  } catch {
    return String(c);
  }
}

export function toolDetail(item: Item): string {
  const args = item.type === "tool_call"
    ? toolArgs(item)
    : { ...toolArgs(item), ...parseMaybeJSON(item.payload) };
  const paths = args.paths;
  const firstPath = Array.isArray(paths) ? String(paths[0] || "") : "";
  const candidates = [
    args.path,
    args.file,
    args.cmd,
    args.command,
    args.query,
    args.url,
    args.app,
    args.title,
    firstPath,
    args.old_str,
  ];
  for (const c of candidates) {
    const s = String(c || "").trim();
    if (s) return s;
  }
  if (item.type === "tool_result") {
    const line = toolResultBody(item).split(/\r?\n/).find((l) => l.trim());
    if (line) return line.trim();
  }
  return "";
}

export function isArtifactTool(name: string): boolean {
  return name === "apply_patch" || name === "cite_sources" || name.startsWith("office_");
}

export function formatElapsed(ms?: number): string {
  if (!ms || ms <= 0) return "";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const s = ms / 1000;
  return `${s.toFixed(s < 10 ? 1 : 0)}s`;
}

export function toolElapsedMs(item: Item): number {
  return Number(item.payload?.elapsed_ms || 0) || 0;
}

export function patchFileCount(body: string): number {
  const m = body.match(/\*\*\* (?:Add|Update|Delete) File:/g);
  return m?.length || 0;
}

export function formatToolBody(item: Item): string {
  if (item.type === "tool_call") {
    const raw = item.payload?.arguments ?? item.text;
    if (typeof raw === "string") {
      const obj = parseMaybeJSON(raw);
      if (Object.keys(obj).length) {
        try {
          return JSON.stringify(obj, null, 2);
        } catch {
          return raw;
        }
      }
      return raw;
    }
    try {
      return JSON.stringify(raw ?? {}, null, 2);
    } catch {
      return String(raw || "");
    }
  }
  return toolResultBody(item);
}
