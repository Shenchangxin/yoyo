/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized from ConversationWidthControls.
 */
import { useCallback, useEffect, useState } from "react";

const KEY = "yoyo.conversation.contentWidth";
const MIN = 640;

function readPref(): number | null {
  const raw = localStorage.getItem(KEY);
  if (raw == null) return null;
  const n = Number(raw);
  return Number.isFinite(n) && n > 0 ? n : null;
}

export function useConversationWidth(enabled: boolean): { width: number | null; bind: (side: "left" | "right") => object } {
  const [width, setWidth] = useState<number | null>(() => (enabled ? readPref() : null));

  useEffect(() => {
    if (!enabled) return;
    setWidth(readPref());
  }, [enabled]);

  const drag = useCallback((side: "left" | "right") => (e: React.PointerEvent<HTMLDivElement>) => {
    if (!enabled) return;
    e.preventDefault();
    const startX = e.clientX;
    const start = width || Math.min(920, Math.max(MIN, window.innerWidth * 0.5));
    const move = (ev: PointerEvent) => {
      const dx = side === "right" ? ev.clientX - startX : startX - ev.clientX;
      const next = Math.max(MIN, Math.round(start + dx * 2));
      setWidth(next);
    };
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
    bind: (side) => ({
      className: "sc-width",
      "data-side": side,
      onPointerDown: drag(side),
    }),
  };
}
