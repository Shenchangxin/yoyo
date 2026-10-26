/**
 * 拥有：判断当前是 Mac、浏览器门还是桌面窗口，以及最小化、最大化、关闭、置顶和缩放。
 * 不拥有：标题栏按钮的绘制。
 */

import { Window } from "@wailsio/runtime";

/** 读 Wails 报告的操作系统名。没有时为空串。 */
function os(): string {
  return (window as any)._wails?.environment?.OS || "";
}

/** isMac 系统名为 darwin，或平台字符串像 Mac。 */
export function isMac(): boolean {
  return os() === "darwin" || /Mac/i.test(navigator.platform);
}

/** applyOsClass 把 mac、windows 或 linux 写到文档根的 data-os。 */
export function applyOsClass() {
  const n = os();
  const key = n === "darwin" || isMac() ? "mac" : n === "windows" || /Win/i.test(navigator.userAgent) ? "windows" : "linux";
  document.documentElement.dataset.os = key;
}

/** browserSurface 手机页是同一套界面，由浏览器门提供。文档标了 browser，或会话存储里有 yoyo_browser。 */
export function browserSurface(): boolean {
  if (typeof document === "undefined") return false;
  if (document.documentElement.dataset.browser === "1") return true;
  try {
    return !!sessionStorage.getItem("yoyo_browser");
  } catch {
    return false;
  }
}

/** desktopWindow 真正的桌面窗口。手机页没有原生对话框和窗口饰条。浏览器门直接为假。 */
export function desktopWindow(): boolean {
  if (browserSurface()) return false;
  const n = os();
  if (n === "windows" || n === "linux" || n === "darwin") return true;
  return typeof window !== "undefined" && !!(window as any).__YOYO_DESKTOP__;
}

/** showCaptionButtons Mac 和浏览器门不显示。Windows 和 Linux 显示。其余环境在粗指针或宽度不超过 799 时隐藏。 */
export function showCaptionButtons(): boolean {
  if (browserSurface() || isMac()) return false;
  const n = os();
  if (n === "windows" || n === "linux") return true;
  if (typeof window === "undefined") return false;
  if (window.matchMedia("(pointer: coarse), (max-width: 799px)").matches) return false;
  return true;
}

/** chrome 调原生窗口。调用失败时忽略。 */
export const chrome = {
  /** minimise 最小化窗口。 */
  minimise() {
    Window.Minimise().catch(() => {});
  },
  /** toggleMaximise 在最大化与还原之间切换。 */
  toggleMaximise() {
    Window.ToggleMaximise().catch(() => {});
  },
  /** close 关闭窗口。 */
  close() {
    Window.Close().catch(() => {});
  },
  /** setAlwaysOnTop 设置窗口是否置顶。 */
  setAlwaysOnTop(v: boolean) {
    Window.SetAlwaysOnTop(v).catch(() => {});
  },
  /** setZoom 设置窗口缩放。 */
  setZoom(v: number) {
    Window.SetZoom(v).catch(() => {});
  },
};
