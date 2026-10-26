/**
 * 拥有：判断预览种类，以及把 HTML 包成检查器 iframe 能显示的文档。
 * 不拥有：iframe 的挂载，以及文件字节的读取。
 */

/** PreviewKind 文本、HTML、Markdown、图片、PDF、音频、视频、Office 或二进制。 */
export type PreviewKind = "text" | "html" | "markdown" | "image" | "pdf" | "audio" | "video" | "office" | "binary";

/** looksLikeNumberedDump read_file 的倾倒形如行号加竖线。那是源码，不是文档。只看去掉开头空白后的前 120 字。raw 为假时用空字符串，调用 trimStart。 */
export function looksLikeNumberedDump(raw: string): boolean {
  const t = (raw || "").trimStart().slice(0, 120);
  return /^\s*\d+\|/.test(t);
}

/** looksLikeHTML 行号倾倒排除。其余看前 400 字：以 doctype html 或 html 开头，或含 mcp-ui。raw 去掉首尾空白，截取，传给 looksLikeNumberedDump。 */
export function looksLikeHTML(raw: string): boolean {
  if (looksLikeNumberedDump(raw)) return false;
  const t = raw.trim().slice(0, 400);
  if (!t) return false;
  const lower = t.toLowerCase();
  return lower.startsWith("<!doctype html") || lower.startsWith("<html") || lower.includes("mcp-ui");
}

/** looksLikeMCPUI 行号倾倒排除。正文含 mcp-ui，或工具名含 mcp 且正文像 HTML。raw 为假时用空字符串，转成小写，传给 looksLikeNumberedDump，传给 looksLikeHTML。toolName 转成小写，看是否包含。 */
export function looksLikeMCPUI(raw: string, toolName = ""): boolean {
  if (looksLikeNumberedDump(raw)) return false;
  const t = (raw || "").toLowerCase();
  if (t.includes("mcp-ui")) return true;
  return toolName.toLowerCase().includes("mcp") && looksLikeHTML(raw);
}

/**
 * workspaceHTMLPathFromOpen 检查器该打开的工作区 HTML。http、https 和 data 一律返回空。去掉 lane=，剥掉 workspace://。file 地址取出路径，Windows 盘符前的斜杠去掉。解析失败返回空。
 * raw 为假时用空字符串，去掉首尾空白，做替换。
 */
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

/**
 * workspacePreviewPath 路径像 HTML 文件则用它。browser_open 和 browser_takeover 再从 url 或路径里收工作区路径。其余返回空。
 * name 与 browser_open、browser_takeover 比较是否相等。
 * args 读取 path、file、url。
 */
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

/** srcdoc iframe 是独立源。localStorage 会抛错并弄停画布游戏，会话 df1ede293929bc0f 即如此。垫片在读取失败时改用内存。 */
const PREVIEW_STORAGE_GUARD = `<script data-yoyo-preview-guard>
(function(){
  function mem(){
    var s = {};
    return {
      getItem: function(k){ return Object.prototype.hasOwnProperty.call(s, k) ? s[k] : null; },
      setItem: function(k,v){ s[String(k)] = String(v); },
      removeItem: function(k){ delete s[k]; },
      clear: function(){ s = {}; },
      key: function(i){ return Object.keys(s)[i] || null; },
      get length(){ return Object.keys(s).length; }
    };
  }
  function shim(name){
    try { void window[name].getItem("__yoyo"); }
    catch (e) {
      try { Object.defineProperty(window, name, { value: mem(), configurable: true }); } catch (err) {}
    }
  }
  shim("localStorage");
  shim("sessionStorage");
})();
</script>`;

/** disablePreviewScripts 已经有内容安全策略则不动。否则写入禁止 script-src 和 object-src 的策略。html 为假时用空字符串后，做替换，传给 test。 */
export function disablePreviewScripts(html: string): string {
  const t = html || "";
  if (/http-equiv\s*=\s*["']Content-Security-Policy["']/i.test(t)) return t;
  if (/<\/head>/i.test(t)) return t.replace(/<\/head>/i, `${NO_SCRIPT_CSP}</head>`);
  if (/<head[^>]*>/i.test(t)) return t.replace(/<head[^>]*>/i, (m) => m + NO_SCRIPT_CSP);
  if (/<html[^>]*>/i.test(t)) return t.replace(/<html[^>]*>/i, (m) => `${m}<head>${NO_SCRIPT_CSP}</head>`);
  return t;
}

/** looksLikeHTMLFile 扩展名是 html、htm 或 xhtml。path 为假时用空字符串，传给 test。 */
export function looksLikeHTMLFile(path: string): boolean {
  return /\.(html?|xhtml)$/i.test(path || "");
}

const PREVIEW_SCROLL_MARK = "data-yoyo-preview-scroll";
const PREVIEW_SCROLL_STYLE = `<style ${PREVIEW_SCROLL_MARK}>html{color-scheme:light;scrollbar-width:thin;scrollbar-color:rgba(24,24,24,.28) transparent}html::-webkit-scrollbar,body::-webkit-scrollbar,*::-webkit-scrollbar{width:7px;height:7px}html::-webkit-scrollbar-track,body::-webkit-scrollbar-track,*::-webkit-scrollbar-track,html::-webkit-scrollbar-corner,body::-webkit-scrollbar-corner,*::-webkit-scrollbar-corner{background:transparent}html::-webkit-scrollbar-thumb,body::-webkit-scrollbar-thumb,*::-webkit-scrollbar-thumb{background-color:rgba(24,24,24,.22);border:2px solid transparent;border-radius:999px;background-clip:padding-box}html::-webkit-scrollbar-thumb:hover,body::-webkit-scrollbar-thumb:hover,*::-webkit-scrollbar-thumb:hover{background-color:rgba(24,24,24,.4)}</style>`;

/** 已有滚动条标记则不动。有 head 就插进去，只有 html 时补一个 head。都没有则原样返回。 */
function withPreviewScrollbars(html: string): string {
  if (!html || html.includes(PREVIEW_SCROLL_MARK)) return html;
  if (/<\/head>/i.test(html)) return html.replace(/<\/head>/i, `${PREVIEW_SCROLL_STYLE}</head>`);
  if (/<head[^>]*>/i.test(html)) return html.replace(/<head[^>]*>/i, (m) => m + PREVIEW_SCROLL_STYLE);
  if (/<html[^>]*>/i.test(html)) return html.replace(/<html[^>]*>/i, (m) => `${m}<head>${PREVIEW_SCROLL_STYLE}</head>`);
  return html;
}

/** 正文为空或已有垫片则不动。有 head 就插进去，只有 html 时补一个 head。都没有则放在前面。 */
function withPreviewStorageGuard(html: string): string {
  if (!html || html.includes("data-yoyo-preview-guard")) return html;
  if (/<head[^>]*>/i.test(html)) return html.replace(/<head[^>]*>/i, (m) => m + PREVIEW_STORAGE_GUARD);
  if (/<html[^>]*>/i.test(html)) return html.replace(/<html[^>]*>/i, (m) => `${m}<head>${PREVIEW_STORAGE_GUARD}</head>`);
  return PREVIEW_STORAGE_GUARD + html;
}

/**
 * asPreviewDocument 包成检查器 iframe 能渲染的文档。空文本返回空。已像 HTML 则补滚动条，否则套上固定的文档壳。未要求脚本时禁用脚本，要求脚本时加上存储垫片。
 * html 为假时用空字符串，去掉首尾空白后，为假时走另一支，拼进字符串，传给 looksLikeHTML，传给 withPreviewScrollbars。
 * opts 读取 scripts，为空时不读。
 */
export function asPreviewDocument(html: string, opts?: { scripts?: boolean }): string {
  const t = (html || "").trim();
  if (!t) return "";
  let doc = looksLikeHTML(t)
    ? withPreviewScrollbars(t)
    : withPreviewScrollbars(
      `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><style>body{margin:1.25rem;font:13px/1.55 system-ui,sans-serif;color:#1c1c1a;background:#fff}img{max-width:100%}table{border-collapse:collapse;width:100%}td,th{border:1px solid #ddd;padding:6px 8px;text-align:left}</style></head><body>${t}</body></html>`,
    );
  if (!opts?.scripts) doc = disablePreviewScripts(doc);
  else doc = withPreviewStorageGuard(doc);
  return doc;
}

/** looksLikePDF 扩展名是 pdf，或 mime 为 application/pdf。path 传给 test。 */
export function looksLikePDF(path: string, mime = ""): boolean {
  return /\.pdf$/i.test(path) || mime.toLowerCase() === "application/pdf";
}

/** looksLikeOffice 扩展名是 docx、xlsx、pptx 或 pdf。path 为假时用空字符串，传给 test。 */
export function looksLikeOffice(path: string): boolean {
  return /\.(docx|xlsx|pptx|pdf)$/i.test(path || "");
}

/** looksLikeImage 扩展名是 png、jpg、jpeg、gif、webp、svg、bmp、ico、avif，或 mime 以 image/ 开头。path 为假时用空字符串，传给 test。 */
export function looksLikeImage(path: string, mime = ""): boolean {
  return /\.(png|jpe?g|gif|webp|svg|bmp|ico|avif)$/i.test(path || "") || mime.toLowerCase().startsWith("image/");
}

/** looksLikeMarkdown 扩展名是 md 或 markdown。path 为假时用空字符串，传给 test。 */
export function looksLikeMarkdown(path: string): boolean {
  return /\.(md|markdown)$/i.test(path || "");
}

/** looksLikeAudio 扩展名是 mp3、wav、ogg、m4a、flac、aac，或 mime 以 audio/ 开头。path 为假时用空字符串，传给 test。 */
export function looksLikeAudio(path: string, mime = ""): boolean {
  return /\.(mp3|wav|ogg|m4a|flac|aac)$/i.test(path || "") || mime.toLowerCase().startsWith("audio/");
}

/** looksLikeVideo 扩展名是 mp4、webm、mov、m4v，或 mime 以 video/ 开头。path 为假时用空字符串，传给 test。 */
export function looksLikeVideo(path: string, mime = ""): boolean {
  return /\.(mp4|webm|mov|m4v)$/i.test(path || "") || mime.toLowerCase().startsWith("video/");
}

/**
 * previewKindFromPath 按 pdf、图片、音频、视频、HTML、Markdown、docx、xlsx、pptx 的顺序判断，其余为文本。
 * path 为假时用空字符串，传给 looksLikePDF，传给 looksLikeImage，传给 looksLikeAudio，传给 looksLikeVideo，传给 looksLikeHTMLFile，传给 looksLikeMarkdown，传给 test。
 * mime 传给 looksLikePDF，传给 looksLikeImage，传给 looksLikeAudio，传给 looksLikeVideo。
 */
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

/**
 * needsBlobPreview pdf、图片、音频、视频，或 docx，需要二进制预览。xlsx 和 pptx 不在此列。
 * path 为假时用空字符串，传给 previewKindFromPath，传给 test。
 * kind 为假时改用后备值后，与 pdf、image、audio、video 比较是否相等。
 * mime 传给 previewKindFromPath。
 */
export function needsBlobPreview(path: string, kind = "", mime = ""): boolean {
  const k = kind || previewKindFromPath(path, mime);
  return k === "pdf" || k === "image" || k === "audio" || k === "video" || /\.docx$/i.test(path || "");
}

/** extractHTML 像 HTML 则返回去掉两端空白的原文，否则空串。raw 去掉首尾空白后，传给 looksLikeHTML。 */
export function extractHTML(raw: string): string {
  const t = raw.trim();
  if (looksLikeHTML(t)) return t;
  return "";
}

/** blobFromBase64 先让出一帧，再用 atob 解码。按 2 的 18 次方为一步拷贝，步与步之间让出事件循环。mime 空则用 application/octet-stream。b64 传给 atob。 */
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