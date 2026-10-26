/**
 * 拥有：会话正文宽度的拖拽和本机记忆。
 * 不拥有：消息排版。只写本地偏好，不调用 api。
 */
import { useCallback, useEffect, useState } from "react";

const KEY = "yoyo.conversation.contentWidth";
const MIN = 640;

/** 读出本机记下的正文宽度。 */
function readPref(): number | null {
  const raw = localStorage.getItem(KEY);
  if (raw == null) return null;
  const n = Number(raw);
  return Number.isFinite(n) && n > 0 ? n : null;
}

/** useConversationWidth 拖左右边改宽度，松手后记到本机。 */
export function useConversationWidth(enabled: boolean): { width: number | null; bind: (side: "left" | "right") => object } {
  const [width, setWidth] = useState<number | null>(() => (enabled ? readPref() : null));

  useEffect(() => {
    if (!enabled) return;
    setWidth(readPref());
  }, [enabled]);

  /** 从某一侧按下，跟着指针改宽度。 */
  const drag = useCallback((side: "left" | "right") => (e: React.PointerEvent<HTMLDivElement>) => {
    if (!enabled) return;
    e.preventDefault();
    const startX = e.clientX;
    const start = width || Math.min(920, Math.max(MIN, window.innerWidth * 0.5));
    /** 按拖动方向加减宽度。 */
    const move = (ev: PointerEvent) => {
      const dx = side === "right" ? ev.clientX - startX : startX - ev.clientX;
      const next = Math.max(MIN, Math.round(start + dx * 2));
      setWidth(next);
    };
    /** 松开后把宽度记到本机。 */
    const up = () => {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", up);
      setWidth((w) => {
        if (w) localStorage.setItem(KEY, String(w));
        return w;
      });
    };
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", up);
  }, [enabled, width]);

  return {
    width: enabled ? width : null,
    /** bind 交回某一侧拖拽条的类名和按下处理。 */
    bind: (side) => ({
      className: "sc-width",
      "data-side": side,
      onPointerDown: drag(side),
    }),
  };
}
