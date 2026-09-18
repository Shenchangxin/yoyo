import type { Item } from "./protocol";
import { looksLikeHTML, looksLikePDF } from "./html-preview";

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

export function isRichResult(name: string, body: string, path = ""): boolean {
  if (looksLikeHTML(body) || looksLikePDF(path)) return true;
  const n = name.toLowerCase();
  return n.includes("mcp") && looksLikeHTML(body);
}

export function isPlanTool(name: string): boolean {
  return name === "update_plan";
}

export function isOutcomeToolName(name: string, body = "", path = ""): boolean {
  return isPlanTool(name) || isArtifactTool(name) || isRichResult(name, body, path);
}

/** Match the Go emit: failures land in content as `ERROR:…`, not always `ok:false`. */
export function isToolFailed(item?: Item | null): boolean {
  if (!item) return false;
  if (item.payload?.ok === false) return true;
  if (item.payload?.error) return true;
  const body = toolResultBody(item);
  return body.startsWith("ERROR:");
}

export function formatElapsed(ms?: number): string {
  if (!ms || ms <= 0) return "";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const s = ms / 1000;
  return `${s.toFixed(s < 10 ? 1 : 0)}s`;
}

/** Wall-clock label for a whole work span ("Worked for 1m 12s"). */
export function formatSpan(ms?: number): string {
  if (!ms || ms <= 0) return "";
  if (ms < 1000) return formatElapsed(ms);
  const total = Math.round(ms / 1000);
  if (total < 60) return `${total}s`;
  const m = Math.floor(total / 60);
  const s = total % 60;
  if (m < 60) return s ? `${m}m ${s}s` : `${m}m`;
  const h = Math.floor(m / 60);
  const rm = m % 60;
  return rm ? `${h}h ${rm}m` : `${h}h`;
}

export function toolElapsedMs(item: Item): number {
  return Number(item.payload?.elapsed_ms || 0) || 0;
}

export function itemTimeMs(item?: Item | null): number {
  if (!item?.ts) return 0;
  const t = Date.parse(item.ts);
  return Number.isFinite(t) ? t : 0;
}

/**
 * Coarse verb family for a tool. Drives the natural-language summary
 * ("Read 3 files · Ran 2 commands") and the live shimmer line ("Reading …").
 * Raw names stay visible in the expanded audit rows; this is narration only.
 */
export type ToolKind = "read" | "edit" | "search" | "run" | "web" | "list" | "plan" | "skill" | "think" | "other";

const KIND_EXACT: Record<string, ToolKind> = {
  read_file: "read",
  view_image: "read",
  office_query: "read",
  recall_context: "read",
  connector_read: "read",
  clipboard_read: "read",
  write_file: "edit",
  edit_file: "edit",
  apply_patch: "edit",
  create_file: "edit",
  delete_file: "edit",
  office_create: "edit",
  office_render: "edit",
  grep: "search",
  glob: "search",
  tool_search: "search",
  memory_search: "search",
  list_skills: "search",
  search_files: "search",
  list_dir: "list",
  project_list: "list",
  schedule_list: "list",
  bash: "run",
  shell: "run",
  exec: "run",
  run_command: "run",
  execute_command: "run",
  git_status: "run",
  git_diff: "run",
  web_search: "web",
  web_fetch: "web",
  fetch_url: "web",
  browser_snapshot: "web",
  update_plan: "plan",
  load_skill: "skill",
};

export function toolKind(name: string): ToolKind {
  const n = name.toLowerCase();
  if (KIND_EXACT[n]) return KIND_EXACT[n];
  if (/(^|_)(read|view|cat|open)(_|$)/.test(n)) return "read";
  if (/(write|edit|patch|create|delete|remove|rename|mkdir)/.test(n)) return "edit";
  if (/(search|grep|glob|find|query|lookup)/.test(n)) return "search";
  if (/(exec|bash|shell|cmd|command|terminal|run|git_)/.test(n)) return "run";
  if (/(web|http|fetch|browser|url|crawl)/.test(n)) return "web";
  if (/(^|_)(list|ls|dir|tree)(_|$)/.test(n)) return "list";
  return "other";
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
