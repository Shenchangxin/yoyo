/**
 * 拥有：从一段话里认出本地文件路径，以及预览时该把路径接到哪个目录。
 * 不拥有：预览和打开文件夹。那是会话页的事。
 */

const FILE_EXT = "html?|xhtml|csv|tsv|pdf|png|jpe?g|gif|webp|svg|md|markdown|txt|json|css|jsx?|tsx?|go|py|docx|xlsx|pptx|mp4|webm|mp3|wav";
const WIN_PATH = new RegExp(
  String.raw`[A-Za-z]:[\\/](?:[^\\/:*?"<>|\r\n]+[\\/])*[^\\/:*?"<>|\r\n]+?\.(?:${FILE_EXT})(?![A-Za-z0-9])`,
  "gi",
);
const REL_PATH = new RegExp(
  "`([^`\\n]+?\\.(?:" + FILE_EXT + "))`|(?:^|[\\s（(「『])((?:[^\\s\\\\/:*?\"<>|`]+[\\\\/])+[^\\s\\\\/:*?\"<>|`]+?\\.(?:" + FILE_EXT + "))(?![A-Za-z0-9])",
  "gi",
);
const BARE_PATH = new RegExp(
  `(?:^|[\\s\`（(「『])([^\\s\\\\/:*?"<>|\`，。、；：！？)）」』]{1,80}\\.(?:${FILE_EXT}))(?![A-Za-z0-9])`,
  "gi",
);

/** fileBase 只留最后一段文件名。 */
export function fileBase(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] || path;
}

/** fileKind 用扩展名给气泡一行短标签。 */
export function fileKind(path: string): string {
  const ext = ((path.match(/\.([A-Za-z0-9]+)$/) || [])[1] || "").toLowerCase();
  const named: Record<string, string> = {
    html: "HTML",
    htm: "HTML",
    xhtml: "HTML",
    csv: "CSV",
    tsv: "TSV",
    md: "Markdown",
    markdown: "Markdown",
    pdf: "PDF",
    png: "PNG",
    jpg: "JPEG",
    jpeg: "JPEG",
    gif: "GIF",
    webp: "WebP",
    svg: "SVG",
    json: "JSON",
    txt: "TXT",
    docx: "Word",
    xlsx: "Excel",
    pptx: "PowerPoint",
  };
  return named[ext] || (ext ? ext.toUpperCase() : "File");
}

const DELIVERABLE = /\.(html?|xhtml|csv|tsv|pdf|png|jpe?g|gif|webp|svg|md|markdown|txt|json|docx|xlsx|pptx)$/i;

/** deliverablePaths 只留适合做成文件气泡的文档和图片。扫描清单里的视频、音频不铺成一排气泡。 */
export function deliverablePaths(text: string): string[] {
  return filePathsIn(text).filter((path) => DELIVERABLE.test(path));
}

/** filePathsIn 收下绝对路径、带斜杠的相对路径和单独的文件名。更长的路径已经包含的文件名不再单独列出。 */
export function filePathsIn(text: string): string[] {
  WIN_PATH.lastIndex = 0;
  REL_PATH.lastIndex = 0;
  BARE_PATH.lastIndex = 0;
  const seen = new Set<string>();
  const out: string[] = [];
  const push = (raw: string) => {
    const path = raw.replace(/[.,;，。、)）\]]+$/, "").trim();
    if (!path || /^https?:/i.test(path) || seen.has(path)) return;
    seen.add(path);
    out.push(path);
  };
  for (const hit of text.match(WIN_PATH) || []) push(hit);
  for (const match of text.matchAll(REL_PATH)) push(match[1] || match[2] || "");
  for (const match of text.matchAll(BARE_PATH)) push(match[1] || "");
  const rooted = out.filter((path) => isAbsPath(path) || /[\\/]/.test(path));
  const bases = new Set(rooted.map((path) => fileBase(path).toLowerCase()));
  return out.filter((path) => rooted.includes(path) || !bases.has(fileBase(path).toLowerCase()));
}

/** hidePaths 把正文里的目录换成文件名。气泡上不再重复整段路径。 */
export function hidePaths(text: string): string {
  const paths = deliverablePaths(text).slice().sort((a, b) => b.length - a.length);
  let out = text;
  for (const path of paths) {
    if (path === fileBase(path)) continue;
    out = out.split(path).join(fileBase(path));
  }
  return out;
}

/** isBrowserPreviewPath 在扩展名是 html、htm 或 xhtml 时为真。 */
export function isBrowserPreviewPath(path: string): boolean {
  return /\.(html?|xhtml)$/i.test(path || "");
}

/** isAbsPath 认 Windows 盘符、以 / 开头和 UNC。 */
export function isAbsPath(path: string): boolean {
  return /^[A-Za-z]:[\\/]/.test(path) || path.startsWith("/") || path.startsWith("\\\\");
}

/** joinedPath 绝对路径原样返回。相对路径接到 workspace。 */
export function joinedPath(workspace: string | undefined, path: string): string {
  if (!path) return "";
  if (isAbsPath(path) || !workspace) return path;
  const sep = workspace.includes("\\") ? "\\" : "/";
  return `${workspace.replace(/[\\/]+$/, "")}${sep}${path.replace(/^[\\/]+/, "")}`;
}

/** previewTarget 绝对路径用父目录加文件名，这样预览不会跳出那个目录。相对路径仍用 workspace。 */
export function previewTarget(workspace: string | undefined, path: string): { workspace?: string; path: string } {
  if (!isAbsPath(path)) return { workspace, path };
  const parts = path.split(/[\\/]/);
  const base = parts.pop() || path;
  return { workspace: parts.join(path.includes("\\") ? "\\" : "/"), path: base };
}
