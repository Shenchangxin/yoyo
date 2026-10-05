/** True when the operator picked a real directory, not cwd/".". */
export function pathReady(p: string | undefined | null): boolean {
  const s = (p || "").trim();
  if (!s || s === "." || s === "./" || s === ".\\") return false;
  if (s.startsWith("/") || s.startsWith("\\\\")) return true;
  return /^[A-Za-z]:[\\/]/.test(s);
}

export function workspaceReady(healthReady: boolean | undefined, workspace: string): boolean {
  if (healthReady) return true;
  return pathReady(workspace);
}

/** Per-chat cwd: isolate worktree when present, else the session workspace. */
export function sessionFsRoot(
  t?: { toolRoot?: string; workspace?: string } | null,
  fallback = "",
): string {
  return (t?.toolRoot || t?.workspace || fallback || "").trim();
}

/** Join a workspace-relative path. Keeps Windows UNC and POSIX abs intact. */
export function joinWorkspace(root: string, rel: string): string {
  if (!rel) return root || "";
  const n = rel.replace(/\\/g, "/");
  if (/^[a-zA-Z]:[\\/]/.test(rel) || rel.startsWith("/") || rel.startsWith("\\\\") || n.startsWith("//")) {
    return rel;
  }
  const base = (root || "").replace(/[\\/]+$/, "");
  return base ? `${base}/${rel.replace(/^[\\/]+/, "")}` : rel;
}

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
