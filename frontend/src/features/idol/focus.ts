/**
 * 拥有「切到某位偶像」的请求：陪伴窗点了她先开口的那句时，桌面服务发出 yoyo:idol-focus，偶像台要选中她。
 * 偶像台可能还没挂上，所以请求先记在这里，挂上时取走；已经挂着就用窗口事件通知它。不改服务端。
 */

let pending = "";

/** requestIdolFocus 记下要选中的偶像，并通知已经挂着的偶像台。 */
export function requestIdolFocus(id: string) {
  const next = String(id || "").trim();
  if (!next) return;
  pending = next;
  if (typeof window !== "undefined") window.dispatchEvent(new CustomEvent("yoyo-idol-focus", { detail: next }));
}

/** takeIdolFocus 取走还没处理的请求，没有时返回空串。 */
export function takeIdolFocus(): string {
  const id = pending;
  pending = "";
  return id;
}

/** 应用启动时就听桌面服务的 yoyo:idol-focus，切页之前到的请求也不会丢。端到端测试和浏览器里没有桌面桥，不听。 */
function install() {
  if (typeof window === "undefined" || (import.meta as any)?.env?.VITE_E2E) return;
  if (!(window as any)._wails?.environment?.OS) return;
  void import("@wailsio/runtime").then((mod: any) => {
    mod?.Events?.On?.("yoyo:idol-focus", (raw: any) => {
      const src = Array.isArray(raw?.data) ? raw.data[0] : raw?.data ?? raw;
      requestIdolFocus(typeof src === "string" ? src : String(src ?? ""));
    });
  }).catch(() => {});
}

install();
