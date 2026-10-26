/**
 * 本文件拥有把画布内部地址翻译成 Yoyo 的设置栏或视频栏。
 * 不拥有那些页面。它只改本地界面，不改服务端，也不是另一条模型循环。
 */
import { videoRouteFromLocation, type SettingsTab } from "../../../lib/protocol";
import { useUI } from "../../../lib/store";
import { useCanvasHost } from "./session";

/** 去掉查询和哈希，并去掉结尾的斜杠。to 转成字符串去掉空白，空的返回 /；否则去掉查询和哈希，再去掉结尾斜杠。 */
function normalizePath(to: string): string {
  const raw = String(to || "").trim();
  if (!raw) return "/";
  const path = raw.split("?")[0].split("#")[0];
  if (path.length > 1 && path.endsWith("/")) return path.slice(0, -1);
  return path || "/";
}

/** 把地址拆成路径和查询。解析失败就按问号切开。to 转成字符串去掉空白后解析，失败就按问号切开。 */
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

/** settingsHostTarget 把设置地址收成 Yoyo 的设置栏。岛内那几项返回空，留给画布自己。to 先拆成路径和查询，再按路径和 section 收成设置栏。 */
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

/** openHostPaneForPath 打开对应的设置或视频栏。认不出的地址返回 false。to 先交给 settingsHostTarget，否则交给 videoRouteFromLocation。 */
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

/** openHostedVideoHref 只在画布宿主标记打开时，才按地址切栏。to 在宿主标记打开时交给 openHostPaneForPath。 */
export function openHostedVideoHref(to: string) {
  if (typeof window === "undefined" || !(window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__) return false;
  return openHostPaneForPath(to);
}

/** installHostNavigation 接住工作区和宿主的跳转，能翻译的就拦住内部路由。 */
export function installHostNavigation() {
  /** 工作区跳转能翻译时，阻止它继续走内部路由。raw 从 detail.to 取出地址，翻译成功就阻止默认并停止传播。 */
  const onWorkspaceNavigate = (raw: Event) => {
    const to = (raw as CustomEvent<{ to?: string }>).detail?.to;
    if (!to || !openHostPaneForPath(to)) return;
    raw.preventDefault();
    raw.stopImmediatePropagation();
  };
  /** 宿主跳转直接按地址切栏。raw 从 detail.to 取出地址，有地址就切栏。 */
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
