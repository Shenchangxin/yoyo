const POPUP_ROOT_ID = "yingce-popup-root";

function hostThemeClass(dark: boolean) {
  return dark ? "yoyo-host-dark" : "yoyo-host-light";
}

/** Keep portaled Ant/yc overlays on a token-bearing host root, not raw document.body. */
export function syncYoyoPopupRoot(dark = typeof document !== "undefined" && document.documentElement.classList.contains("dark")) {
  if (typeof document === "undefined") return null;
  let root = document.getElementById(POPUP_ROOT_ID);
  if (!root) {
    root = document.createElement("div");
    root.id = POPUP_ROOT_ID;
    document.body.appendChild(root);
  }
  root.className = `yingce-island yingce-popup-root app-user-overlays app-product-overlays ${hostThemeClass(dark)}`;
  root.style.colorScheme = dark ? "dark" : "light";
  return root;
}

export function yoyoCanvasPopupContainer(node?: HTMLElement): HTMLElement {
  if (typeof document === "undefined") return node || (undefined as unknown as HTMLElement);
  const dark = document.documentElement.classList.contains("dark");
  return syncYoyoPopupRoot(dark) || document.body;
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
