import { createContext, useCallback, useContext, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { cn } from "../../lib/utils";
import { createMate } from "./engine";
import { usePresencePointer } from "./gaze";
import type { MoodMate } from "./types";
import type { PresenceSlotId } from "./types";

const PRIORITY: Record<PresenceSlotId, number> = {
  boot: 100,
  module: 80,
  process: 60,
  avatar: 40,
  home: 20,
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
    if (!mateRef.current || kindRef.current !== kind) {
      mateRef.current?.destroy();
      host.replaceChildren();
      mateRef.current = createMate(host, {
        emotion: props.emotionId,
        idle,
        lite: glyph || props.reduced,
        autostart: !props.reduced && visible,
        eyeScale: glyph ? 1.6 : 1,
      });
      kindRef.current = kind;
      if (props.reduced) mateRef.current.renderStatic();
    } else {
      mateRef.current.setEmotion(props.emotionId);
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
      className={cn("presence-anchor relative shrink-0", props.className)}
      style={{ width: props.size, height: props.size }}
      data-presence-slot={props.id}
      data-presence-active={active ? "true" : "false"}
      data-testid={`presence-${props.id}`}
      onClick={() => {
        slots?.celebrate();
        props.onClick?.();
      }}
    />
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
      eyeScale: glyph ? 1.6 : 1,
    });
    mate.current = inst;
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
    mate.current?.setEmotion(props.emotionId);
  }, [props.emotionId]);

  useLayoutEffect(() => {
    if (!props.spark) return;
    mate.current?.celebrate(1);
  }, [props.spark]);

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
      onClick={() => {
        mate.current?.celebrate(1);
        props.onClick?.();
      }}
    />
  );
}

export function PresenceStamp(props: { className?: string; size?: number }) {
  const n = props.size ?? 28;
  return (
    <span
      className={cn("presence-stamp relative inline-block shrink-0", props.className)}
      style={{ width: n, height: n }}
      aria-hidden
    />
  );
}

export const MODULE_PRESENCE_SIZE = 228;

export function PresenceModuleLoading(props: { label?: string; className?: string }) {
  return (
    <div
      className={cn(
        "flex min-h-[min(28rem,calc(100vh-12rem))] w-full flex-1 flex-col items-center justify-center gap-4",
        props.className,
      )}
      data-testid="presence-module-loading"
      aria-busy="true"
    >
      <PresenceAnchor id="module" size={MODULE_PRESENCE_SIZE} />
      {props.label ? <p className="text-[13px] text-muted">{props.label}</p> : null}
    </div>
  );
}
