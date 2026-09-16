import { Window } from "@wailsio/runtime";

function os(): string {
  return (window as any)._wails?.environment?.OS || "";
}

export function isMac(): boolean {
  return os() === "darwin" || /Mac/i.test(navigator.platform);
}

export function showCaptionButtons(): boolean {
  if (isMac()) return false;
  const n = os();
  return n === "windows" || n === "linux" || n === "";
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
