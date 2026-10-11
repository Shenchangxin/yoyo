import { videoRouteFromLocation, type SettingsTab } from "../../../lib/protocol";
import { useUI } from "../../../lib/store";
import { useCanvasHost } from "./session";

function normalizePath(to: string): string {
  const raw = String(to || "").trim();
  if (!raw) return "/";
  const path = raw.split("?")[0].split("#")[0];
  if (path.length > 1 && path.endsWith("/")) return path.slice(0, -1);
  return path || "/";
}

function parseTo(to: string): { path: string; search: URLSearchParams } {
  const raw = String(to || "").trim();
  try {
    const u = new URL(raw, "https://yoyo.local");
    return { path: normalizePath(u.pathname), search: u.searchParams };
  } catch {
    const [p, q] = raw.split("?");
    return { path: normalizePath(p), search: new URLSearchParams(q || "") };
  }
}

const ISLAND_SETTINGS = new Set(["preferences", "prompts", "agent-memory", "diagnostics"]);

export function settingsHostTarget(to: string): { tab: SettingsTab; section?: string } | null {
  const { path, search } = parseTo(to);
  if (path === "/settings" || path.startsWith("/settings/")) {
    const section = search.get("section") || "";
    if (ISLAND_SETTINGS.has(section)) return null;
    if (section === "storage") return { tab: "generation", section: "generation-storage" };
    if (section === "runninghub") return { tab: "generation", section: "generation-workflow" };
    if (section === "channels" || section === "models") return { tab: "generation", section: "generation-image" };
    if (section === "diagnostics") return { tab: "advanced", section: "advanced-logs" };
    if (section === "agent-memory") return { tab: "personal", section: "personal-memory" };
    return { tab: "generation" };
  }
  if (path.startsWith("/admin/settings/storage") || path === "/admin/settings/oss") {
    return { tab: "generation", section: "generation-storage" };
  }
  return null;
}

export function openHostPaneForPath(to: string): boolean {
  const settings = settingsHostTarget(to);
  if (settings) {
    useUI.getState().openSettings(settings.tab, settings.section);
    return true;
  }
  const mapped = videoRouteFromLocation(to);
  if (!mapped) return false;
  if (mapped.projectId) useCanvasHost.setState({ projectId: mapped.projectId });
  useUI.getState().openVideoPane(mapped.pane, { canvasFocus: mapped.canvasFocus, entry: mapped.entry });
  return true;
}

export function openHostedVideoHref(to: string) {
  if (typeof window === "undefined" || !(window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__) return false;
  return openHostPaneForPath(to);
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
