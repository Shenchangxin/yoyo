/**
 * 本文件拥有画布浮层的宿主根，以及请 Yoyo 切视频栏的事件。
 * 不拥有菜单内容。它只改页面上的根节点和本地跳转，不改服务端。
 */
import { useCanvasThemeStore } from "@yingce/stores/canvas/use-canvas-theme-store";

const POPUP_ROOT_ID = "yingce-popup-root";

/** 明暗对应宿主根上的类名。dark 为真时返回 yoyo-host-dark，否则返回 yoyo-host-light。 */
function hostThemeClass(dark: boolean) {
  return dark ? "yoyo-host-dark" : "yoyo-host-light";
}

/** 画布主题明确时用它，否则看传入值，再否则看文档是否带 dark。explicit 是布尔值时直接决定明暗。 */
function popupThemeDark(explicit?: boolean) {
  const { active, theme } = useCanvasThemeStore.getState();
  if (active && (theme === "light" || theme === "dark")) return theme === "dark";
  if (typeof explicit === "boolean") return explicit;
  return typeof document !== "undefined" && document.documentElement.classList.contains("dark");
}

/** syncYoyoPopupRoot 把传送出去的浮层放在带令牌的宿主根上，不直接挂到没有令牌的 document.body。dark 交给 popupThemeDark 决定宿主根的明暗类和 colorScheme。 */
export function syncYoyoPopupRoot(dark?: boolean) {
  if (typeof document === "undefined") return null;
  let root = document.getElementById(POPUP_ROOT_ID);
  if (!root) {
    root = document.createElement("div");
    root.id = POPUP_ROOT_ID;
    document.body.appendChild(root);
  }
  const { active, theme } = useCanvasThemeStore.getState();
  const resolvedDark = popupThemeDark(dark);
  root.className = `yingce-island yingce-popup-root app-user-overlays app-product-overlays ${hostThemeClass(resolvedDark)}`;
  root.style.colorScheme = resolvedDark ? "dark" : "light";
  if (active && (theme === "light" || theme === "dark")) root.dataset.canvasColor = theme;
  else delete root.dataset.canvasColor;
  return root;
}

/** yoyoCanvasPopupContainer 返回这块带令牌的根；没有文档时退回传入节点。node 只在没有文档时退回。 */
export function yoyoCanvasPopupContainer(node?: HTMLElement): HTMLElement {
  if (typeof document === "undefined") return node || (undefined as unknown as HTMLElement);
  return syncYoyoPopupRoot() || document.body;
}

/** canvasOverlayHost 让菜单和检查器离开画布视口，同时保住画布配色。不在宿主里就用 document.body。 */
export function canvasOverlayHost(): HTMLElement {
  if (typeof document === "undefined") return undefined as unknown as HTMLElement;
  if (isYoyoCanvasHost()) return syncYoyoPopupRoot() || document.body;
  return document.body;
}

/** isYoyoCanvasHost 看窗口上有没有画布宿主标记。 */
export function isYoyoCanvasHost(): boolean {
  return typeof window !== "undefined" && Boolean((window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__);
}

/** requestHostNavigation 在托管岛里请 Yoyo 切视频栏，而不是只改内部记忆路由。to 放进 yoyo:host-navigate 事件的 detail。 */
export function requestHostNavigation(to: string): boolean {
  if (!isYoyoCanvasHost() || typeof window === "undefined") return false;
  window.dispatchEvent(new CustomEvent("yoyo:host-navigate", { detail: { to } }));
  return true;
}

/** goWorkspacePath 宿主接住了就返回，否则交给备用跳转。to 先交给 requestHostNavigation。fallback 在宿主没接住时调用，并带上 to。 */
export function goWorkspacePath(to: string, fallback?: (to: string) => void) {
  if (requestHostNavigation(to)) return;
  fallback?.(to);
}
