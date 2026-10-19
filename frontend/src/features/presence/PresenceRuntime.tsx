import { createContext, useCallback, useContext, useId, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { cn } from "../../lib/utils";
import { createMate } from "./engine";
import { entryMotion, playMotion } from "./flourish";
import { usePresencePointer } from "./gaze";
import type { MoodMate } from "./types";
import type { PresenceSlotId } from "./types";

const PRIORITY: Record<PresenceSlotId, number> = {
  boot: 100,
  module: 80,
  home: 20,
  composer: 15,
};

type Slot = { id: PresenceSlotId; el: HTMLElement; size: number };

type PresenceSlots = {
  claim: (id: PresenceSlotId, el: HTMLElement, size: number) => () => void;
  activeId: PresenceSlotId | null;
  celebrate: () => void;
};

const SlotsCtx = createContext<PresenceSlots | null>(null);

export function PresenceRuntime(props: {
  emotionId: string;
  reduced: boolean;
  children?: ReactNode;
}) {
  const [slots, setSlots] = useState<Slot[]>([]);
  const hostRef = useRef<HTMLDivElement | null>(null);
  if (!hostRef.current && typeof document !== "undefined") {
    const el = document.createElement("div");
    el.className = "presence-engine";
    el.setAttribute("aria-hidden", "true");
    hostRef.current = el;
  }
  const mateRef = useRef<MoodMate | null>(null);
  const kindRef = useRef<string | null>(null);
  const faceRef = useRef("");

  const active = useMemo(() => {
    let best: Slot | null = null;
    for (const s of slots) {
      if (!best || PRIORITY[s.id] > PRIORITY[best.id]) best = s;
    }
    return best;
  }, [slots]);
  const activeId = active?.id ?? null;

  const claim = useCallback((id: PresenceSlotId, el: HTMLElement, size: number) => {
    setSlots((prev) => {
      const i = prev.findIndex((s) => s.id === id);
      if (i >= 0 && prev[i]!.el === el && prev[i]!.size === size) return prev;
      const next = prev.filter((s) => s.id !== id);
      next.push({ id, el, size });
      return next;
    });
    return () => {
      setSlots((prev) => {
        const next = prev.filter((s) => s.el !== el);
        return next.length === prev.length ? prev : next;
      });
    };
  }, []);

  const glyph = !!active && active.size < 64;
  const idle = active?.id === "home";
  const kind = `${glyph ? "glyph" : "stage"}:${idle ? "idle" : "work"}`;

  useLayoutEffect(() => {
    const host = hostRef.current;
    if (!host || !active) {
      mateRef.current?.setActive(false);
      return;
    }
    host.style.width = `${active.size}px`;
    host.style.height = `${active.size}px`;
    if (host.parentElement !== active.el) active.el.appendChild(host);
    const visible = typeof document === "undefined" || document.visibilityState === "visible";
    host.dataset.emotion = props.emotionId;
    if (!mateRef.current || kindRef.current !== kind) {
      mateRef.current?.destroy();
      host.replaceChildren();
      mateRef.current = createMate(host, {
        emotion: props.emotionId,
        idle,
        lite: glyph || props.reduced,
        autostart: !props.reduced && visible,
      });
      kindRef.current = kind;
      faceRef.current = props.emotionId;
      if (props.reduced) mateRef.current.renderStatic();
    } else if (faceRef.current !== props.emotionId) {
      mateRef.current.setEmotion(props.emotionId);
      mateRef.current.setActive(!props.reduced && visible);
      if (props.reduced) mateRef.current.renderStatic();
      else playMotion(mateRef.current, entryMotion(props.emotionId));
      faceRef.current = props.emotionId;
    } else {
      mateRef.current.setActive(!props.reduced && visible);
      if (props.reduced) mateRef.current.renderStatic();
    }
  }, [active, glyph, idle, kind, props.emotionId, props.reduced]);

  useLayoutEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const apply = () => {
      const on = document.visibilityState === "visible" && !!active && !props.reduced;
      mateRef.current?.setActive(on);
    };
    const io = new IntersectionObserver(() => apply());
    io.observe(host);
    document.addEventListener("visibilitychange", apply);
    return () => {
      io.disconnect();
      document.removeEventListener("visibilitychange", apply);
    };
  }, [active, props.reduced]);

  usePresencePointer({
    mateRef,
    hostRef,
    emotionId: props.emotionId,
    reduced: props.reduced,
    glyph,
    follow: true,
  });

  useLayoutEffect(() => {
    if (import.meta.env.VITE_E2E) return;
    let off: (() => void) | undefined;
    let alive = true;
    void import("@wailsio/runtime").then((mod: any) => {
      const Events = mod.Events;
      if (!alive || !Events?.On) return;
      off = Events.On("yoyo:companion", (raw: any) => {
        const action = typeof raw === "string" ? raw : String(raw?.data ?? raw ?? "");
        if (action.includes("nudge")) mateRef.current?.celebrate(1);
      });
    });
    return () => {
      alive = false;
      off?.();
    };
  }, []);

  useLayoutEffect(() => () => {
    mateRef.current?.destroy();
    mateRef.current = null;
    hostRef.current?.remove();
  }, []);

  const celebrate = useCallback(() => {
    mateRef.current?.celebrate(1);
  }, []);
  const ctx = useMemo<PresenceSlots>(
    () => ({ claim, activeId, celebrate }),
    [claim, activeId, celebrate],
  );

  return (
    <SlotsCtx.Provider value={ctx}>
      {props.children}
    </SlotsCtx.Provider>
  );
}

export function PresenceAnchor(props: {
  id: PresenceSlotId;
  size: number;
  className?: string;
  onClick?: () => void;
  fallback?: boolean;
}) {
  const slots = useContext(SlotsCtx);
  const ref = useRef<HTMLDivElement>(null);
  const claim = slots?.claim;
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el || !claim) return;
    return claim(props.id, el, props.size);
  }, [claim, props.id, props.size]);
  const active = slots?.activeId === props.id;
  return (
    <div
      ref={ref}
      className={cn("presence-anchor relative shrink-0", props.size >= 64 && "presence-float", props.className)}
      style={{ width: props.size, height: props.size }}
      data-presence-slot={props.id}
      data-presence-active={active ? "true" : "false"}
      data-testid={`presence-${props.id}`}
      onClick={() => {
        slots?.celebrate();
        props.onClick?.();
      }}
    >
      {props.fallback && !active ? <PresenceStamp size={props.size} live className="presence-anchor-fallback" /> : null}
    </div>
  );
}

/** Standalone instance for the companion window or exclusive module loaders. */
export function PresenceSprite(props: {
  emotionId: string;
  size?: number;
  idle?: boolean;
  lite?: boolean;
  reduced?: boolean;
  className?: string;
  onClick?: () => void;
  followPointer?: boolean;
  screenFollow?: boolean;
  spark?: number;
  onActivity?: () => void;
}) {
  const box = useRef<HTMLDivElement>(null);
  const mate = useRef<MoodMate | null>(null);
  const faceRef = useRef("");
  const size = props.size ?? 140;
  const glyph = size < 64;

  useLayoutEffect(() => {
    const el = box.current;
    if (!el) return;
    const inst = createMate(el, {
      emotion: props.emotionId,
      idle: false,
      lite: props.lite ?? (glyph || !!props.reduced),
      autostart: !props.reduced,
      shape: "blob",
    });
    mate.current = inst;
    faceRef.current = props.emotionId;
    if (props.reduced) inst.renderStatic();
    const vis = () => {
      inst.setActive(document.visibilityState === "visible" && !props.reduced);
    };
    document.addEventListener("visibilitychange", vis);
    return () => {
      document.removeEventListener("visibilitychange", vis);
      inst.destroy();
      mate.current = null;
    };
  }, [glyph, props.lite, props.reduced, size]);

  useLayoutEffect(() => {
    if (!mate.current) return;
    if (faceRef.current === props.emotionId) return;
    mate.current.setEmotion(props.emotionId);
    if (!props.reduced) playMotion(mate.current, entryMotion(props.emotionId));
    faceRef.current = props.emotionId;
  }, [props.emotionId, props.reduced]);

  useLayoutEffect(() => {
    if (!props.spark) return;
    const motion = entryMotion(props.emotionId);
    if (motion === "celebrate" || motion === "flourish") return;
    if (typeof mate.current?.nudge === "function") mate.current.nudge();
    else mate.current?.bounce();
  }, [props.spark, props.emotionId]);

  const instRef = mate;
  usePresencePointer({
    mateRef: instRef,
    hostRef: box,
    emotionId: props.emotionId,
    reduced: props.reduced,
    glyph,
    follow: props.followPointer !== false,
    screenFollow: !!props.screenFollow,
    onActivity: props.onActivity,
  });

  return (
    <div
      ref={box}
      className={cn("presence-engine", props.className)}
      style={{ width: size, height: size }}
      data-testid="presence-sprite"
      data-emotion={props.emotionId}
      onClick={() => {
        mate.current?.celebrate(1);
        props.onClick?.();
      }}
    />
  );
}

/** Rest-face letterhead: cream blob + conjugated eyes. Not a live engine. */
export function PresenceStamp(props: { className?: string; size?: number; live?: boolean }) {
  const n = props.size ?? 36;
  const raw = useId();
  const gid = `psg-${raw.replace(/[^a-zA-Z0-9_-]/g, "")}`;
  return (
    <span
      className={cn("presence-stamp relative inline-block shrink-0", props.live && "is-live", props.className)}
      style={{ width: n, height: n }}
      aria-hidden
      data-testid="presence-stamp"
      data-live={props.live ? "true" : undefined}
    >
      <svg
        viewBox="-15 -15 259 259"
        width={n}
        height={n}
        focusable="false"
        className="block h-full w-full overflow-visible"
        shapeRendering="geometricPrecision"
      >
        <defs>
          <radialGradient id={gid} cx="38%" cy="32%" r="75%">
            <stop offset="0%" stopColor="#fffaf3" />
            <stop offset="62%" stopColor="#F3F0EA" />
            <stop offset="100%" stopColor="#e4ddd2" />
          </radialGradient>
        </defs>
        <circle className="presence-stamp-body" cx="114.27" cy="114.27" r="114.2" fill={`url(#${gid})`} />
        <g className="presence-stamp-face">
          <ellipse className="presence-stamp-eye" cx="86.4" cy="108.6" rx="10.8" ry="22.4" fill="#1A1A1A" />
          <ellipse className="presence-stamp-eye" cx="142.1" cy="108.6" rx="10.8" ry="22.4" fill="#1A1A1A" />
          <path
            className="presence-stamp-mouth"
            d="M78 148C100 174 128.5 174 150.5 148C128 166 100.5 166 78 148Z"
            fill="#1A1A1A"
          />
        </g>
      </svg>
    </span>
  );
}

export const COMPOSER_PRESENCE_SIZE = 64;

export function PresencePerch(props: { className?: string; size?: number }) {
  const size = props.size ?? COMPOSER_PRESENCE_SIZE;
  return (
    <div className={cn("presence-perch", props.className)} data-testid="presence-perch">
      <PresenceAnchor id="composer" size={size} fallback />
    </div>
  );
}

export const MODULE_PRESENCE_SIZE = 228;

export function PresenceModuleLoading(props: { label?: string; className?: string; size?: number }) {
  return (
    <div
      className={cn(
        "presence-module-loading flex min-h-[min(28rem,calc(100vh-12rem))] w-full flex-1 flex-col items-center justify-center gap-4",
        props.className,
      )}
      data-testid="presence-module-loading"
      aria-busy="true"
    >
      <div className="presence-module-shell">
        <PresenceAnchor id="module" size={props.size ?? MODULE_PRESENCE_SIZE} />
      </div>
      {props.label ? <p className="text-[13px] text-muted">{props.label}</p> : null}
    </div>
  );
}
