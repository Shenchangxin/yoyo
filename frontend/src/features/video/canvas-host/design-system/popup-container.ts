export function yoyoCanvasPopupContainer(node?: HTMLElement): HTMLElement {
  if (typeof document === "undefined") return node || (undefined as unknown as HTMLElement);
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
