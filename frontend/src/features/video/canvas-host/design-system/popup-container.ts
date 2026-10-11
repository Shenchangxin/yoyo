import { useCanvasThemeStore } from "@yingce/stores/canvas/use-canvas-theme-store";

const POPUP_ROOT_ID = "yingce-popup-root";

function hostThemeClass(dark: boolean) {
  return dark ? "yoyo-host-dark" : "yoyo-host-light";
}

function popupThemeDark(explicit?: boolean) {
  const { active, theme } = useCanvasThemeStore.getState();
  if (active && (theme === "light" || theme === "dark")) return theme === "dark";
  if (typeof explicit === "boolean") return explicit;
  return typeof document !== "undefined" && document.documentElement.classList.contains("dark");
}

/** Keep portaled Ant/yc overlays on a token-bearing host root, not raw document.body. */
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

export function yoyoCanvasPopupContainer(node?: HTMLElement): HTMLElement {
  if (typeof document === "undefined") return node || (undefined as unknown as HTMLElement);
  return syncYoyoPopupRoot() || document.body;
}

/** Menus and inspectors that must escape the canvas viewport, while keeping the canvas color theme. */
export function canvasOverlayHost(): HTMLElement {
  if (typeof document === "undefined") return undefined as unknown as HTMLElement;
  if (isYoyoCanvasHost()) return syncYoyoPopupRoot() || document.body;
  return document.body;
}

export function isYoyoCanvasHost(): boolean {
  return typeof window !== "undefined" && Boolean((window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__);
}

/** Hosted island: ask Yoyo to switch the Video rail instead of only changing the inner memory router. */
export function requestHostNavigation(to: string): boolean {
  if (!isYoyoCanvasHost() || typeof window === "undefined") return false;
  window.dispatchEvent(new CustomEvent("yoyo:host-navigate", { detail: { to } }));
  return true;
}

export function goWorkspacePath(to: string, fallback?: (to: string) => void) {
  if (requestHostNavigation(to)) return;
  fallback?.(to);
}
