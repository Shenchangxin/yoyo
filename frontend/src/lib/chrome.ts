import { Window } from "@wailsio/runtime";

function os(): string {
  return (window as any)._wails?.environment?.OS || "";
}

export function isMac(): boolean {
  return os() === "darwin" || /Mac/i.test(navigator.platform);
}

export function applyOsClass() {
  const n = os();
  const key = n === "darwin" || isMac() ? "mac" : n === "windows" || /Win/i.test(navigator.userAgent) ? "windows" : "linux";
  document.documentElement.dataset.os = key;
}

/** The phone page is this same UI, served by the browser door. */
export function browserSurface(): boolean {
  if (typeof document === "undefined") return false;
  if (document.documentElement.dataset.browser === "1") return true;
  try {
    return !!sessionStorage.getItem("yoyo_browser");
  } catch {
    return false;
  }
}

/** A real desktop window. The phone has no native dialogs or window chrome. */
export function desktopWindow(): boolean {
  if (browserSurface()) return false;
  const n = os();
  if (n === "windows" || n === "linux" || n === "darwin") return true;
  return typeof window !== "undefined" && !!(window as any).__YOYO_DESKTOP__;
}

export function showCaptionButtons(): boolean {
  if (browserSurface() || isMac()) return false;
  const n = os();
  if (n === "windows" || n === "linux") return true;
  if (typeof window === "undefined") return false;
  if (window.matchMedia("(pointer: coarse), (max-width: 799px)").matches) return false;
  return true;
}

export const chrome = {
  minimise() {
    Window.Minimise().catch(() => {});
  },
  toggleMaximise() {
    Window.ToggleMaximise().catch(() => {});
  },
  close() {
    Window.Close().catch(() => {});
  },
  setAlwaysOnTop(v: boolean) {
    Window.SetAlwaysOnTop(v).catch(() => {});
  },
  setZoom(v: number) {
    Window.SetZoom(v).catch(() => {});
  },
};
