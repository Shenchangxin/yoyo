import type { CanvasFocus, VideoPane } from "../../../lib/protocol";
import { useUI } from "../../../lib/store";

function normalizePath(to: string): string {
  const raw = String(to || "").trim();
  if (!raw) return "/";
  const path = raw.split("?")[0].split("#")[0];
  if (path.length > 1 && path.endsWith("/")) return path.slice(0, -1);
  return path || "/";
}

export function pathToHostPane(to: string): { pane: VideoPane; canvasFocus?: CanvasFocus } | null {
  const path = normalizePath(to);
  if (path === "/" || path === "/create") return { pane: "create" };
  if (path === "/canvas") return { pane: "canvas", canvasFocus: "library" };
  if (path === "/drama") return { pane: "drama" };
  if (path === "/assets") return { pane: "assets" };
  if (path === "/skills") return { pane: "skills" };
  if (path === "/plugins" || path === "/plugins/eagle") return { pane: "plugins" };
  if (path === "/tasks") return { pane: "tasks" };
  return null;
}

export function openHostPaneForPath(to: string): boolean {
  const mapped = pathToHostPane(to);
  if (!mapped) return false;
  useUI.getState().openVideoPane(mapped.pane, mapped.canvasFocus ? { canvasFocus: mapped.canvasFocus } : undefined);
  return true;
}

export function installHostNavigation() {
  const onWorkspaceNavigate = (raw: Event) => {
    const to = (raw as CustomEvent<{ to?: string }>).detail?.to;
    if (!to || !openHostPaneForPath(to)) return;
    raw.preventDefault();
    raw.stopImmediatePropagation();
  };
  const onHostNavigate = (raw: Event) => {
    const to = (raw as CustomEvent<{ to?: string }>).detail?.to;
    if (to) openHostPaneForPath(to);
  };
  window.addEventListener("workspace:navigate", onWorkspaceNavigate, true);
  window.addEventListener("yoyo:host-navigate", onHostNavigate);
  return () => {
    window.removeEventListener("workspace:navigate", onWorkspaceNavigate, true);
    window.removeEventListener("yoyo:host-navigate", onHostNavigate);
  };
}
