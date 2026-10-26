/**
 * 拥有：判断工作区路径是否可用，以及拼接会话的文件系统根。
 * 不拥有：目录选择器，以及工作区的创建。
 */

/** pathReady 操作者选了真实目录时为真。空串、当前目录和单独的点不算。p 为假时用空字符串，去掉首尾空白后，与 .、./、.\\ 比较是否相等，看开头，为假时走另一支，传给 test。 */
export function pathReady(p: string | undefined | null): boolean {
  const s = (p || "").trim();
  if (!s || s === "." || s === "./" || s === ".\\") return false;
  if (s.startsWith("/") || s.startsWith("\\\\")) return true;
  return /^[A-Za-z]:[\\/]/.test(s);
}

/** workspaceReady 健康检查已就绪，或路径本身可用，即为真。healthReady 为真时返回真。workspace 传给 pathReady。 */
export function workspaceReady(healthReady: boolean | undefined, workspace: string): boolean {
  if (healthReady) return true;
  return pathReady(workspace);
}

/** sessionFsRoot 有隔离工作树时用 toolRoot，否则用会话工作区。t 读取 toolRoot、workspace，为空时不读。fallback 作为后备值。 */
export function sessionFsRoot(
  t?: { toolRoot?: string; workspace?: string } | null,
  fallback = "",
): string {
  return (t?.toolRoot || t?.workspace || fallback || "").trim();
}

/** joinWorkspace 拼接相对路径。Windows UNC 和 POSIX 绝对路径原样保留。root 为假时用空字符串，做替换。rel 做替换，看开头，为假时走另一支，作为分支结果，传给 test。 */
export function joinWorkspace(root: string, rel: string): string {
  if (!rel) return root || "";
  const n = rel.replace(/\\/g, "/");
  if (/^[a-zA-Z]:[\\/]/.test(rel) || rel.startsWith("/") || rel.startsWith("\\\\") || n.startsWith("//")) {
    return rel;
  }
  const base = (root || "").replace(/[\\/]+$/, "");
  return base ? `${base}/${rel.replace(/^[\\/]+/, "")}` : rel;
}

/** recentWorkspaces 按出现顺序去重，丢掉空白。paths 用来遍历。 */
export function recentWorkspaces(paths: Array<string | undefined | null>): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of paths) {
    const p = (raw || "").trim();
    if (!p || seen.has(p)) continue;
    seen.add(p);
    out.push(p);
  }
  return out;
}
