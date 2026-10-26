import type { Item } from "./protocol";
import { looksLikeHTML, looksLikeHTMLFile, looksLikePDF } from "./html-preview";

/**
 * 拥有：从工具条目里取出名称、参数、正文、成败和给操作者看的一句摘要。
 * 不拥有：工具的执行，以及转录里工具行如何成组。
 */

/** parseMaybeJSON 对象原样返回。字符串能解析成对象时返回该对象，否则返回空对象。 */
export function parseMaybeJSON(raw: unknown): Record<string, unknown> {
  if (raw && typeof raw === "object" && !Array.isArray(raw)) return raw as Record<string, unknown>;
  if (typeof raw !== "string") return {};
  const t = raw.trim();
  if (!t) return {};
  try {
    const o = JSON.parse(t) as unknown;
    if (o && typeof o === "object" && !Array.isArray(o)) return o as Record<string, unknown>;
  } catch {
    /* 不是 JSON */
  }
  return {};
}

/** toolName 工具名。条目和载荷都没有时用 tool。 */
export function toolName(item: Item): string {
  return String(item.name || item.payload?.name || "tool");
}

/** skillCallLabel 给操作者看的技能 id。不是技能运行时调用时返回空串。 */
export function skillCallLabel(item: Item): string {
  const tagged = String(item.payload?.skill || "").trim();
  if (tagged) return tagged;
  const name = toolName(item);
  const args = toolArgs(item);
  switch (name) {
    case "load_skill": {
      const skill = String(args.name || "").trim();
      return skill || name;
    }
    case "read_skill_file": {
      const skill = String(args.skill || "").trim();
      const path = String(args.path || "").trim();
      if (skill && path) return `${skill} · ${path}`;
      return skill || name;
    }
    case "run_skill_script": {
      const skill = String(args.skill || "").trim();
      const script = String(args.script || "").trim();
      if (skill && script) return `${skill} · ${script}`;
      return skill || script || name;
    }
    case "list_skills":
      return name;
    default:
      return "";
  }
}

/** toolArgs 解析参数，并用载荷上的 path、paths 补上缺失的路径。 */
export function toolArgs(item: Item): Record<string, unknown> {
  if (!item) return {};
  const parsed = parseMaybeJSON(item.payload?.arguments ?? item.text);
  const out: Record<string, unknown> = { ...parsed };
  const payloadPath = String(item.payload?.path || "").trim();
  if (payloadPath && !String(out.path || "").trim()) out.path = payloadPath;
  const paths = item.payload?.paths;
  if (Array.isArray(paths) && paths.length) {
    if (!out.paths) out.paths = paths;
    if (!String(out.path || "").trim()) out.path = String(paths[0] || "");
  }
  return out;
}

/** toolResultBody 工具结果正文。对象则格式化成 JSON。 */
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

/** toolDetail 一行摘要：技能标签优先，否则取路径、命令、查询等第一个非空参数。结果没有参数时用正文第一行。 */
export function toolDetail(item: Item): string {
  const skill = skillCallLabel(item);
  if (skill) return skill;
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

/** isArtifactTool 会产出操作者可打开的文件的工具。office_ 前缀也算。 */
export function isArtifactTool(name: string): boolean {
  switch (name) {
    case "apply_patch":
    case "cite_sources":
    case "write_file":
    case "str_replace":
    case "edit_file":
    case "create_file":
      return true;
    default:
      return name.startsWith("office_");
  }
}

/** isRichResult 结果值得按富预览展示：PDF、HTML 文件，或带 HTML 正文的产出工具与 MCP。 */
export function isRichResult(name: string, body: string, path = ""): boolean {
  if (looksLikePDF(path) && (isArtifactTool(name) || name.startsWith("office_"))) return true;
  if (looksLikeHTMLFile(path) && isArtifactTool(name)) return true;
  if (looksLikeHTML(body)) {
    const n = name.toLowerCase();
    return isArtifactTool(name) || n.includes("mcp");
  }
  return name.toLowerCase().includes("mcp") && looksLikeHTML(body);
}

/** toolResultUnchanged str_replace 或 write_file 没有改动字节。正文以 unchanged 开头。 */
export function toolResultUnchanged(item?: Item | null): boolean {
  if (!item) return false;
  const body = String(item.payload?.content || item.text || "").trim();
  return /^unchanged\b/i.test(body);
}

/** isPlanTool 是不是更新计划的工具。 */
export function isPlanTool(name: string): boolean {
  return name === "update_plan";
}

/** isOutcomeToolName 产出文件，或结果适合富预览。 */
export function isOutcomeToolName(name: string, body = "", path = ""): boolean {
  return isArtifactTool(name) || isRichResult(name, body, path);
}

/** isToolFailed 与 Go 侧发出的失败一致：正文以 ERROR: 开头也算失败，不只看 ok 为假。 */
export function isToolFailed(item?: Item | null): boolean {
  if (!item) return false;
  if (item.payload?.ok === false) return true;
  if (item.payload?.error) return true;
  const body = toolResultBody(item);
  return body.startsWith("ERROR:");
}

/** formatElapsed 单次耗时。不足一秒用毫秒，十秒内保留一位小数。 */
export function formatElapsed(ms?: number): string {
  if (!ms || ms <= 0) return "";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const s = ms / 1000;
  return `${s.toFixed(s < 10 ? 1 : 0)}s`;
}

/** formatSpan 一整段工作的墙钟，例如「工作了 1 分 12 秒」所用的紧凑写法。不足一秒沿用 formatElapsed。 */
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

/** toolElapsedMs 条目载荷里的耗时毫秒。没有则为零。 */
export function toolElapsedMs(item: Item): number {
  return Number(item.payload?.elapsed_ms || 0) || 0;
}

/** itemTimeMs 条目时间戳的毫秒。无法解析则为零。 */
export function itemTimeMs(item?: Item | null): number {
  if (!item?.ts) return 0;
  const t = Date.parse(item.ts);
  return Number.isFinite(t) ? t : 0;
}

/**
 * ToolKind 工具的粗动词族，只用于叙述。
 * 自然语言摘要（读了几个文件、跑了几条命令）和进行中的闪烁行用它。展开的审计行仍显示原始工具名。
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
  list_skills: "skill",
  load_skill: "skill",
  read_skill_file: "skill",
  run_skill_script: "skill",
};

/** toolKind 先查精确表，再按名字里的动词归族。对不上则为 other。 */
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

/** patchFileCount 补丁正文里 Add、Update、Delete 文件头的个数。 */
export function patchFileCount(body: string): number {
  const m = body.match(/\*\*\* (?:Add|Update|Delete) File:/g);
  return m?.length || 0;
}

/** formatToolBody 调用尽量把参数格式化成 JSON。结果用工具结果正文。 */
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
