import { looksLikeHTML, looksLikeHTMLFile } from "./html-preview";
import { toolArgs, toolName } from "./tool-summary";
import type { Item } from "./protocol";

export type ArtifactKind = "html" | "diff" | "code" | "text";

export type ArtifactView = {
  kind: ArtifactKind;
  path: string;
  title: string;
  lang: string;
  html: string;
  diff: string;
  code: string;
  added?: number;
  removed?: number;
  created?: boolean;
};

const LANG: Record<string, string> = {
  go: "go", ts: "ts", tsx: "tsx", js: "js", jsx: "jsx", mjs: "js", cjs: "js",
  json: "json", css: "css", html: "html", htm: "html", md: "md", py: "python",
  rs: "rust", yml: "yaml", yaml: "yaml", toml: "toml", sh: "bash", vue: "vue",
};

export function langFromPath(path: string): string {
  const ext = (path.split(".").pop() || "").toLowerCase();
  return LANG[ext] || ext;
}

export function pathFromPatch(patch: string): string {
  const m = (patch || "").match(/\*\*\*\s+(?:Add|Update) File:\s*(.+)/);
  return m ? m[1].trim() : "";
}

export function prefixLines(text: string, mark: string): string {
  if (!text) return "";
  return text.split("\n").map((ln) => mark + ln).join("\n");
}

export function strReplaceToDiff(path: string, oldStr: string, newStr: string): string {
  const file = path || "file";
  return [`--- a/${file}`, `+++ b/${file}`, "@@", prefixLines(oldStr, "-"), prefixLines(newStr, "+")].join("\n");
}

export function applyPatchToDiff(patch: string): string {
  const lines = (patch || "").replace(/\r\n/g, "\n").split("\n");
  const out: string[] = [];
  for (const ln of lines) {
    if (ln.startsWith("*** Begin Patch") || ln.startsWith("*** End Patch")) continue;
    if (ln.startsWith("*** Add File:")) {
      out.push("--- /dev/null");
      out.push(`+++ b/${ln.slice("*** Add File:".length).trim()}`);
      continue;
    }
    if (ln.startsWith("*** Update File:")) {
      const p = ln.slice("*** Update File:".length).trim();
      out.push(`--- a/${p}`);
      out.push(`+++ b/${p}`);
      continue;
    }
    out.push(ln);
  }
  return out.join("\n").trim();
}

export function fenced(lang: string, text: string, cap = 12000): string {
  const body = (text || "").slice(0, cap).replace(/```/g, "``\u200b`");
  return "```" + (lang || "text") + "\n" + body + "\n```";
}

export function isElidedArg(raw: string): boolean {
  return /^\s*\[elided\s/i.test(raw || "");
}

export function artifactPreviewOpen(view: ArtifactView): boolean {
  return artifactShouldShow(view);
}

/** A preview card is only for a changed fragment (or an HTML document). Full-file code dumps are not a change. */
export function diffHasEdits(diff: string): boolean {
  const plus: string[] = [];
  const minus: string[] = [];
  for (const raw of (diff || "").split("\n")) {
    if (raw.startsWith("+++") || raw.startsWith("---") || raw.startsWith("@@")) continue;
    if (raw.startsWith("+")) plus.push(raw.slice(1));
    else if (raw.startsWith("-")) minus.push(raw.slice(1));
  }
  if (!plus.length && !minus.length) return false;
  return plus.join("\n") !== minus.join("\n");
}

export function artifactShouldShow(view: ArtifactView): boolean {
  if (view.kind === "html" && (looksLikeHTML(view.html) || looksLikeHTMLFile(view.path))) return true;
  if (view.kind === "diff" && diffHasEdits(view.diff)) return true;
  return false;
}

function payloadNum(item: Item | undefined, key: string): number {
  const n = Number(item?.payload?.[key] || 0);
  return Number.isFinite(n) ? n : 0;
}

export function artifactView(call?: Item, result?: Item): ArtifactView {
  const item = call || result;
  const name = item ? toolName(item) : "";
  const args = {
    ...(result ? toolArgs(result) : {}),
    ...(call ? toolArgs(call) : {}),
  };
  const resultPatch = String(result?.payload?.patch || "");
  const patch = String(args.patch || resultPatch || "");
  const path = String(args.path || args.file || pathFromPatch(patch) || result?.payload?.paths?.[0] || "");
  const rawContent = String(args.content || "");
  const content = isElidedArg(rawContent) ? "" : rawContent;
  const oldStr = isElidedArg(String(args.old_str || args.oldStr || "")) ? "" : String(args.old_str || args.oldStr || "");
  const newStr = isElidedArg(String(args.new_str || args.newStr || "")) ? "" : String(args.new_str || args.newStr || "");
  const htmlBody = looksLikeHTML(content) ? content : "";
  const unified = patch ? (patch.includes("*** ") ? applyPatchToDiff(patch) : patch) : "";
  const stats = {
    added: payloadNum(result, "added") || payloadNum(call, "added"),
    removed: payloadNum(result, "removed") || payloadNum(call, "removed"),
    created: !!(result?.payload?.created || call?.payload?.created),
  };
  const extra = { added: stats.added || undefined, removed: stats.removed || undefined, created: stats.created || undefined };
  if ((name === "write_file" || name === "create_file") && (htmlBody || looksLikeHTMLFile(path))) {
    return { kind: "html", path, title: path || "HTML", lang: "html", html: htmlBody, diff: unified, code: htmlBody, ...extra };
  }
  if (name === "apply_patch" && patch) {
    return { kind: "diff", path, title: path || "Patch", lang: langFromPath(path), html: "", diff: applyPatchToDiff(patch), code: "", ...extra };
  }
  if (unified && diffHasEdits(unified)) {
    return { kind: "diff", path, title: path || "Edit", lang: langFromPath(path), html: "", diff: unified, code: "", ...extra };
  }
  if ((name === "str_replace" || name === "edit_file") && (oldStr || newStr)) {
    return {
      kind: "diff",
      path,
      title: path || "Edit",
      lang: langFromPath(path),
      html: "",
      diff: strReplaceToDiff(path, oldStr, newStr),
      code: "",
      ...extra,
    };
  }
  if ((name === "str_replace" || name === "edit_file") && looksLikeHTMLFile(path)) {
    return { kind: "html", path, title: path || "HTML", lang: "html", html: "", diff: unified, code: "", ...extra };
  }
  if ((name === "write_file" || name === "create_file") && content) {
    return { kind: "code", path, title: path || "File", lang: langFromPath(path), html: "", diff: unified, code: content, ...extra };
  }
  if (unified) {
    return { kind: "diff", path, title: path || name || "File", lang: langFromPath(path), html: "", diff: unified, code: "", ...extra };
  }
  return { kind: "text", path, title: path || name || "Artifact", lang: langFromPath(path), html: "", diff: "", code: "", ...extra };
}
