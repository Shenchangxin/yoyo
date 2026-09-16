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
