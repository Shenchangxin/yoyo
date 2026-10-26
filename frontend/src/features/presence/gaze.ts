/** 拥有指针相对宠物的视线，以及闲置脸上的切换。不拥有情绪球本体，也不另开模型循环。本地计算，不改服务端。 */
import { useLayoutEffect, useRef } from "react";
import type { MoodMate } from "./types";

const AMBIENT = new Set([
  "00", "01", "02", "04", "06", "08", "09",
  "10", "11", "12", "14", "15", "18", "19", "20",
  "22", "23", "24", "25", "29", "48",
]);
/** 指针快到这个程度时，算一次猛地看过来。 */
const DART_SPEED = 1.85;
const GAZE_EPS = 0.004;
const FAR_GAZE = 0.28;
const SHY_MS = 1800;

/** gazeGain 把远处一瞥收成看向宠物，穿过身体时不突然跳变。dist 减去 outer。size 乘以 0.2，乘以 1.05。 */
export function gazeGain(dist: number, size: number): number {
  const inner = Math.max(size * 0.2, 16);
  const outer = Math.max(size * 1.05, 72);
  const span = Math.max(outer - inner, 1);
  const u = Math.min(1, Math.max(0, (outer - dist) / span));
  const s = u * u * (3 - 2 * u);
  return FAR_GAZE + (1 - FAR_GAZE) * s;
}

/** gazeFromPoint 把指针相对盒子的位置收成视线。rect 读取 left、width、top、height。x 减去 cx。y 减去 cy。 */
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
    ny: (dy / dist) * mag * 0.62,
  };
}

/**
 * gazeFromScreen 用窗口在屏幕上的原点，再走同一套视线。
 * rect 读取 left、top、width、height。
 * screenX 传给 gazeFromPoint。
 * screenY 传给 gazeFromPoint。
 * originX 加上 rect.left。
 * originY 加上 rect.top。
 */
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

/** isIdleEmotion 判断这个表情是不是闲置组。id 传给 AMBIENT.has。 */
export function isIdleEmotion(id: string): boolean {
  return AMBIENT.has(id);
}

/** curiousId 把闲置脸换成好奇，干活的脸不动。id 作为分支结果，传给 isIdleEmotion。 */
export function curiousId(id: string): string {
  return isIdleEmotion(id) ? "03" : id;
}

/** dartId 只在闲置脸碰上很快的指针时换成受惊。id 与 00、06 比较是否相等，传给 isIdleEmotion。speed 小于 DART_SPEED。 */
export function dartId(id: string, speed: number): string {
  if (!isIdleEmotion(id) || speed < DART_SPEED) return id;
  if (id === "00" || id === "06") return "07";
  return "13";
}

/** lingerId 在闲置脸上停留久了就换成害羞。id 传给 isIdleEmotion，传给 curiousId。lingering 为假时走另一支。heldMs 大于等于 4600，大于等于 2800。 */
export function lingerId(id: string, lingering: boolean, heldMs = SHY_MS): string {
  if (!lingering || !isIdleEmotion(id)) return curiousId(id);
  if (heldMs >= 4600) return "22";
  if (heldMs >= 2800) return "09";
  return "14";
}

/** usePresencePointer 跟着指针改视线和脸。opts 读取 emotionId、onActivity、mateRef、reduced、glyph、follow 等。 */
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
  const dartFace = useRef("13");
  const nearSince = useRef(0);
  const lastFace = useRef("");
  const actAt = useRef(0);
  const onAct = useRef(opts.onActivity);
  onAct.current = opts.onActivity;

  /** 按悬停、速度和停留换表情。hovering 为真时才继续。speed 传给 dartId。lingering 传给 lingerId。heldMs 传给 lingerId。 */
  const face = (hovering: boolean, speed = 0, lingering = false, heldMs = 0) => {
    const mate = opts.mateRef.current;
    if (!mate || opts.reduced || opts.glyph) return;
    const now = Date.now();
    let next = emotionRef.current;
    if (now < dartUntil.current) next = dartFace.current;
    else {
      const darted = dartId(next, speed);
      if (darted === "13" || darted === "07") {
        dartUntil.current = now + 900;
        dartFace.current = darted;
        next = darted;
      } else if (hovering) next = lingerId(next, lingering, heldMs);
    }
    if (next === lastFace.current) return;
    lastFace.current = next;
    mate.setEmotion(next);
    if ((next === "13" || next === "07") && now - bounceAt.current > 900) {
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
    /** 取当前宿主元素。 */
    const host = () => opts.hostRef.current;
    let raf = 0;
    let pending: { x: number; y: number; screen: boolean; ox?: number; oy?: number } | null = null;
    let lastNear = false;

    /**
     * 按指针位置更新视线和脸。
     * x 减去 cx，减去 last.x，传给 gazeFromScreen，传给 gazeFromPoint，写入字段 x。
     * y 减去 cy，减去 last.y，传给 gazeFromScreen，传给 gazeFromPoint，写入字段 y。
     * screen 为真时才继续。
     * ox 为空时改用后备值，作为分支结果，传给 gazeFromScreen。
     * oy 为空时改用后备值，作为分支结果，传给 gazeFromScreen。
     */
    const point = (x: number, y: number, screen: boolean, ox?: number, oy?: number) => {
      const mate = opts.mateRef.current;
      const el = host();
      if (!el || !mate) return;
      const r = el.getBoundingClientRect();
      if (r.width < 8) return;
      const originX = screen ? (ox ?? window.screenX) : 0;
      const originY = screen ? (oy ?? window.screenY) : 0;
      const cx = originX + r.left + r.width / 2;
      const cy = originY + r.top + r.height / 2;
      const dist = Math.hypot(x - cx, y - cy);
      const enter = Math.max(r.width * 0.72, 64);
      const leave = enter + 28;
      const near = lastNear ? dist < leave : dist < enter;
      const g = screen
        ? gazeFromScreen(r, x, y, ox ?? window.screenX, oy ?? window.screenY)
        : gazeFromPoint(r, x, y);
      const gain = gazeGain(dist, r.width);
      const follow = { nx: g.nx * gain, ny: g.ny * gain };
      const prev = lastGaze.current;
      if (Math.hypot(follow.nx - prev.nx, follow.ny - prev.ny) >= GAZE_EPS || near !== lastNear) {
        lastGaze.current = follow;
        mate.setGaze(follow.nx, follow.ny, near && !opts.screenFollow);
      }
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
      const heldMs = near ? now - nearSince.current : 0;
      const lingering = near && heldMs >= SHY_MS;
      if (near !== lastNear) {
        lastNear = near;
        hoverRef.current = near;
        if (near) nearSince.current = now;
        if (!opts.screenFollow) {
          face(near, speed, false, 0);
          if (near && now - bounceAt.current > 1600 && now >= dartUntil.current) {
            bounceAt.current = now;
            if (typeof mate.nudge === "function") mate.nudge();
            else mate.bounce();
          }
        }
      } else if (speed >= DART_SPEED && now >= dartUntil.current) {
        face(near, speed, lingering, heldMs);
      } else if (lingering && !opts.screenFollow) {
        face(near, speed, true, heldMs);
      }
    };

    /** 把挂起的指针点消化掉。 */
    const flush = () => {
      raf = 0;
      const p = pending;
      pending = null;
      if (p) point(p.x, p.y, p.screen, p.ox, p.oy);
    };
    /** 把指针点留到下一帧。x 写入字段 x。y 写入字段 y。screen 写入字段 screen。ox 写入字段 ox。oy 写入字段 oy。 */
    const schedule = (x: number, y: number, screen: boolean, ox?: number, oy?: number) => {
      pending = { x, y, screen, ox, oy };
      if (!raf) raf = requestAnimationFrame(flush);
    };

    /** 记下窗口内的指针。e 读取 clientX、clientY。 */
    const onMove = (e: PointerEvent) => schedule(e.clientX, e.clientY, false);
    if (!opts.screenFollow) window.addEventListener("pointermove", onMove, { passive: true });

    /** 指针离开就放下视线。 */
    const onLeave = () => {
      opts.mateRef.current?.clearGaze?.();
      lastGaze.current = { nx: 99, ny: 99 };
      if (hoverRef.current || Date.now() < dartUntil.current) {
        hoverRef.current = false;
        lastNear = false;
        dartUntil.current = 0;
        nearSince.current = 0;
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
      /** __yoyoCursor 调用 schedule。x 传给 schedule。y 传给 schedule。wx 传给 schedule。wy 传给 schedule。 */
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
