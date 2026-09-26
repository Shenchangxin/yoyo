import { useLayoutEffect, useRef } from "react";
import type { MoodMate } from "./types";

const AMBIENT = new Set(["00", "01", "02", "04", "06", "10", "11", "12", "14", "15", "18", "19", "20"]);
/** Pointer speed in CSS pixels per millisecond that reads as a dart. */
const DART_SPEED = 1.85;
const GAZE_EPS = 0.004;

export function gazeFromPoint(rect: { left: number; top: number; width: number; height: number }, x: number, y: number): { nx: number; ny: number } {
  const cx = rect.left + rect.width / 2;
  const cy = rect.top + rect.height / 2;
  const dx = x - cx;
  const dy = y - cy;
  const dist = Math.hypot(dx, dy);
  if (dist < 0.5) return { nx: 0, ny: 0 };
  const reach = Math.max(rect.width * 0.38, 48);
  const mag = Math.min(1, dist / reach);
  return {
    nx: (dx / dist) * mag,
    ny: (dy / dist) * mag * 0.82,
  };
}

export function gazeFromScreen(
  rect: { left: number; top: number; width: number; height: number },
  screenX: number,
  screenY: number,
  originX = typeof window === "undefined" ? 0 : window.screenX,
  originY = typeof window === "undefined" ? 0 : window.screenY,
): { nx: number; ny: number } {
  return gazeFromPoint(
    {
      left: originX + rect.left,
      top: originY + rect.top,
      width: rect.width,
      height: rect.height,
    },
    screenX,
    screenY,
  );
}

export function isIdleEmotion(id: string): boolean {
  return AMBIENT.has(id);
}

export function curiousId(id: string): string {
  return isIdleEmotion(id) ? "03" : id;
}

export function dartId(id: string, speed: number): string {
  if (!isIdleEmotion(id) || speed < DART_SPEED) return id;
  return "13";
}

export function usePresencePointer(opts: {
  mateRef: { current: MoodMate | null };
  hostRef: { current: HTMLElement | null };
  emotionId: string;
  reduced?: boolean;
  glyph?: boolean;
  follow?: boolean;
  screenFollow?: boolean;
  onActivity?: () => void;
}) {
  const emotionRef = useRef(opts.emotionId);
  emotionRef.current = opts.emotionId;
  const hoverRef = useRef(false);
  const dartUntil = useRef(0);
  const bounceAt = useRef(0);
  const lastPt = useRef({ x: 0, y: 0, t: 0 });
  const lastGaze = useRef({ nx: 99, ny: 99 });
  const lastFace = useRef("");
  const actAt = useRef(0);
  const onAct = useRef(opts.onActivity);
  onAct.current = opts.onActivity;

  const face = (hovering: boolean, speed = 0) => {
    const mate = opts.mateRef.current;
    if (!mate || opts.reduced || opts.glyph) return;
    const now = Date.now();
    let next = emotionRef.current;
    if (now < dartUntil.current) next = "13";
    else {
      const darted = dartId(next, speed);
      if (darted === "13") {
        dartUntil.current = now + 900;
        next = "13";
      } else if (hovering) next = curiousId(next);
    }
    if (next === lastFace.current) return;
    lastFace.current = next;
    mate.setEmotion(next);
    if (next === "13" && now - bounceAt.current > 900) {
      bounceAt.current = now;
      mate.bounce();
    }
  };

  useLayoutEffect(() => {
    lastFace.current = "";
    face(hoverRef.current);
  }, [opts.emotionId, opts.glyph, opts.reduced]);

  useLayoutEffect(() => {
    if (opts.follow === false || opts.reduced) return;
    const host = () => opts.hostRef.current;
    let raf = 0;
    let pending: { x: number; y: number; screen: boolean; ox?: number; oy?: number } | null = null;
    let lastNear = false;

    const point = (x: number, y: number, screen: boolean, ox?: number, oy?: number) => {
      const mate = opts.mateRef.current;
      const el = host();
      if (!el || !mate) return;
      const r = el.getBoundingClientRect();
      if (r.width < 8) return;
      const g = screen
        ? gazeFromScreen(r, x, y, ox ?? window.screenX, oy ?? window.screenY)
        : gazeFromPoint(r, x, y);
      const { nx, ny } = g;
      const prev = lastGaze.current;
      if (Math.hypot(nx - prev.nx, ny - prev.ny) >= GAZE_EPS) {
        lastGaze.current = { nx, ny };
        mate.setGaze(nx, ny);
      }
      const originX = screen ? (ox ?? window.screenX) : 0;
      const originY = screen ? (oy ?? window.screenY) : 0;
      const cx = originX + r.left + r.width / 2;
      const cy = originY + r.top + r.height / 2;
      const dist = Math.hypot(x - cx, y - cy);
      const enter = Math.max(r.width * 0.72, 64);
      const leave = enter + 28;
      const near = lastNear ? dist < leave : dist < enter;
      const t = performance.now();
      const last = lastPt.current;
      const dt = last.t ? t - last.t : 0;
      const speed = dt > 6 ? Math.hypot(x - last.x, y - last.y) / dt : 0;
      lastPt.current = { x, y, t };
      const now = Date.now();
      if (onAct.current && now - actAt.current > 800) {
        actAt.current = now;
        onAct.current();
      }
      if (near !== lastNear) {
        lastNear = near;
        hoverRef.current = near;
        face(near, speed);
        if (near && now - bounceAt.current > 1600 && now >= dartUntil.current) {
          bounceAt.current = now;
          mate.bounce();
        }
      } else if (speed >= DART_SPEED && now >= dartUntil.current) {
        face(near, speed);
      }
    };

    const flush = () => {
      raf = 0;
      const p = pending;
      pending = null;
      if (p) point(p.x, p.y, p.screen, p.ox, p.oy);
    };
    const schedule = (x: number, y: number, screen: boolean, ox?: number, oy?: number) => {
      pending = { x, y, screen, ox, oy };
      if (!raf) raf = requestAnimationFrame(flush);
    };

    const onMove = (e: PointerEvent) => schedule(e.clientX, e.clientY, false);
    if (!opts.screenFollow) window.addEventListener("pointermove", onMove, { passive: true });

    const onLeave = () => {
      opts.mateRef.current?.clearGaze?.();
      lastGaze.current = { nx: 99, ny: 99 };
      if (hoverRef.current || Date.now() < dartUntil.current) {
        hoverRef.current = false;
        lastNear = false;
        dartUntil.current = 0;
        lastFace.current = "";
        face(false);
      }
    };
    if (!opts.screenFollow) {
      window.addEventListener("pointerleave", onLeave);
      document.addEventListener("mouseleave", onLeave);
    }

    const prevCursor = (window as unknown as { __yoyoCursor?: (x: number, y: number, wx?: number, wy?: number) => void }).__yoyoCursor;
    if (opts.screenFollow) {
      (window as unknown as { __yoyoCursor?: (x: number, y: number, wx?: number, wy?: number) => void }).__yoyoCursor = (x, y, wx, wy) => {
        schedule(x, y, true, wx, wy);
      };
    }

    return () => {
      if (raf) cancelAnimationFrame(raf);
      if (!opts.screenFollow) window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerleave", onLeave);
      document.removeEventListener("mouseleave", onLeave);
      const w = window as unknown as { __yoyoCursor?: (x: number, y: number, wx?: number, wy?: number) => void };
      if (opts.screenFollow && w.__yoyoCursor && w.__yoyoCursor !== prevCursor) {
        if (prevCursor) w.__yoyoCursor = prevCursor;
        else delete w.__yoyoCursor;
      }
    };
  }, [opts.follow, opts.reduced, opts.glyph, opts.screenFollow]);
}
