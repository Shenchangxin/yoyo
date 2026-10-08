export type PreviewKind = "text" | "html" | "markdown" | "image" | "pdf" | "audio" | "video" | "office" | "binary";

/** `read_file` dumps look like `     1|<!DOCTYPE html>`. Those are source, not documents. */
export function looksLikeNumberedDump(raw: string): boolean {
  const t = (raw || "").trimStart().slice(0, 120);
  return /^\s*\d+\|/.test(t);
}

export function looksLikeHTML(raw: string): boolean {
  if (looksLikeNumberedDump(raw)) return false;
  const t = raw.trim().slice(0, 400);
  if (!t) return false;
  const lower = t.toLowerCase();
  return lower.startsWith("<!doctype html") || lower.startsWith("<html") || lower.includes("mcp-ui");
}

export function looksLikeMCPUI(raw: string, toolName = ""): boolean {
  if (looksLikeNumberedDump(raw)) return false;
  const t = (raw || "").toLowerCase();
  if (t.includes("mcp-ui")) return true;
  return toolName.toLowerCase().includes("mcp") && looksLikeHTML(raw);
}

/** Workspace HTML the right inspector should open — never remote http(s) or data:. */
export function workspaceHTMLPathFromOpen(raw: string): string {
  let s = (raw || "").trim().replace(/\s+lane=\S+/i, "").trim();
  if (!s || /^(https?:|data:)/i.test(s)) return "";
  if (/^workspace:\/\//i.test(s)) s = s.replace(/^workspace:\/\//i, "");
  if (/^file:/i.test(s)) {
    try {
      const u = new URL(s);
      let p = decodeURIComponent(u.pathname || "");
      if (/^\/[a-zA-Z]:/.test(p)) p = p.slice(1);
      s = p;
    } catch {
      return "";
    }
  }
  s = s.replace(/\\/g, "/").replace(/^\/+/, "");
  return s;
}

export function workspacePreviewPath(name: string, args: Record<string, unknown>): string {
  const path = String(args.path || args.file || "").trim();
  if (looksLikeHTMLFile(path)) return path;
  if (name === "browser_open" || name === "browser_takeover") {
    const fromURL = workspaceHTMLPathFromOpen(String(args.url || path || ""));
    return fromURL;
  }
  return "";
}

const NO_SCRIPT_CSP = `<meta http-equiv="Content-Security-Policy" content="script-src 'none'; object-src 'none'">`;

export function disablePreviewScripts(html: string): string {
  const t = html || "";
  if (/http-equiv\s*=\s*["']Content-Security-Policy["']/i.test(t)) return t;
  if (/<\/head>/i.test(t)) return t.replace(/<\/head>/i, `${NO_SCRIPT_CSP}</head>`);
  if (/<head[^>]*>/i.test(t)) return t.replace(/<head[^>]*>/i, (m) => m + NO_SCRIPT_CSP);
  if (/<html[^>]*>/i.test(t)) return t.replace(/<html[^>]*>/i, (m) => `${m}<head>${NO_SCRIPT_CSP}</head>`);
  return t;
}

export function looksLikeHTMLFile(path: string): boolean {
  return /\.(html?|xhtml)$/i.test(path || "");
}

const PREVIEW_SCROLL_MARK = "data-yoyo-preview-scroll";
const PREVIEW_SCROLL_STYLE = `<style ${PREVIEW_SCROLL_MARK}>html{color-scheme:light;scrollbar-width:thin;scrollbar-color:rgba(24,24,24,.28) transparent}html::-webkit-scrollbar,body::-webkit-scrollbar,*::-webkit-scrollbar{width:7px;height:7px}html::-webkit-scrollbar-track,body::-webkit-scrollbar-track,*::-webkit-scrollbar-track,html::-webkit-scrollbar-corner,body::-webkit-scrollbar-corner,*::-webkit-scrollbar-corner{background:transparent}html::-webkit-scrollbar-thumb,body::-webkit-scrollbar-thumb,*::-webkit-scrollbar-thumb{background-color:rgba(24,24,24,.22);border:2px solid transparent;border-radius:999px;background-clip:padding-box}html::-webkit-scrollbar-thumb:hover,body::-webkit-scrollbar-thumb:hover,*::-webkit-scrollbar-thumb:hover{background-color:rgba(24,24,24,.4)}</style>`;

function withPreviewScrollbars(html: string): string {
  if (!html || html.includes(PREVIEW_SCROLL_MARK)) return html;
  if (/<\/head>/i.test(html)) return html.replace(/<\/head>/i, `${PREVIEW_SCROLL_STYLE}</head>`);
  if (/<head[^>]*>/i.test(html)) return html.replace(/<head[^>]*>/i, (m) => m + PREVIEW_SCROLL_STYLE);
  if (/<html[^>]*>/i.test(html)) return html.replace(/<html[^>]*>/i, (m) => `${m}<head>${PREVIEW_SCROLL_STYLE}</head>`);
  return html;
}

/** Wrap a fragment so the inspector iframe can render it. */
export function asPreviewDocument(html: string, opts?: { scripts?: boolean }): string {
  const t = (html || "").trim();
  if (!t) return "";
  let doc = looksLikeHTML(t)
    ? withPreviewScrollbars(t)
    : withPreviewScrollbars(
      `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><style>body{margin:1.25rem;font:13px/1.55 system-ui,sans-serif;color:#1c1c1a;background:#fff}img{max-width:100%}table{border-collapse:collapse;width:100%}td,th{border:1px solid #ddd;padding:6px 8px;text-align:left}</style></head><body>${t}</body></html>`,
    );
  if (!opts?.scripts) doc = disablePreviewScripts(doc);
  return doc;
}

export function looksLikePDF(path: string, mime = ""): boolean {
  return /\.pdf$/i.test(path) || mime.toLowerCase() === "application/pdf";
}

export function looksLikeOffice(path: string): boolean {
  return /\.(docx|xlsx|pptx|pdf)$/i.test(path || "");
}

export function looksLikeImage(path: string, mime = ""): boolean {
  return /\.(png|jpe?g|gif|webp|svg|bmp|ico|avif)$/i.test(path || "") || mime.toLowerCase().startsWith("image/");
}

export function looksLikeMarkdown(path: string): boolean {
  return /\.(md|markdown)$/i.test(path || "");
}

export function looksLikeAudio(path: string, mime = ""): boolean {
  return /\.(mp3|wav|ogg|m4a|flac|aac)$/i.test(path || "") || mime.toLowerCase().startsWith("audio/");
}

export function looksLikeVideo(path: string, mime = ""): boolean {
  return /\.(mp4|webm|mov|m4v)$/i.test(path || "") || mime.toLowerCase().startsWith("video/");
}

export function previewKindFromPath(path: string, mime = ""): PreviewKind {
  if (looksLikePDF(path, mime)) return "pdf";
  if (looksLikeImage(path, mime)) return "image";
  if (looksLikeAudio(path, mime)) return "audio";
  if (looksLikeVideo(path, mime)) return "video";
  if (looksLikeHTMLFile(path)) return "html";
  if (looksLikeMarkdown(path)) return "markdown";
  if (/\.(docx|xlsx|pptx)$/i.test(path || "")) return "office";
  return "text";
}

export function needsBlobPreview(path: string, kind = "", mime = ""): boolean {
  const k = kind || previewKindFromPath(path, mime);
  return k === "pdf" || k === "image" || k === "audio" || k === "video" || /\.docx$/i.test(path || "");
}

export function extractHTML(raw: string): string {
  const t = raw.trim();
  if (looksLikeHTML(t)) return t;
  return "";
}

export async function blobFromBase64(b64: string, mime: string): Promise<Blob> {
  await new Promise<void>((resolve) => {
    if (typeof requestAnimationFrame === "function") requestAnimationFrame(() => resolve());
    else resolve();
  });
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  const step = 1 << 18;
  for (let i = 0; i < bin.length; i += step) {
    const end = Math.min(i + step, bin.length);
    for (let j = i; j < end; j++) bytes[j] = bin.charCodeAt(j);
    if (end < bin.length) await new Promise((r) => setTimeout(r, 0));
  }
  return new Blob([bytes], { type: mime || "application/octet-stream" });
}