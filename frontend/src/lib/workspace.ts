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
