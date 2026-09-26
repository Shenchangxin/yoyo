import { useEffect, useMemo, useRef, useState } from "react";
import { presenceLine, presenceOf } from "./director";
import { PresenceSprite } from "./PresenceRuntime";
import { useMotionReduced } from "../../lib/motion";
import { subscribeItems, subscribeSessions } from "../../lib/stream";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";

const KEEP = 80;
const AWAKE_MS = 12_000;
const PULSE_MS = 7000;

type Pulse = { title: string; body: string; session: string; until: number; kind: string };

function pulseFrom(raw: any): Pulse | null {
  const p = raw && typeof raw === "object" && "title" in raw ? raw : raw?.data || raw;
  if (!p || typeof p !== "object") return null;
  const title = String(p.title || "").trim();
  if (!title) return null;
  return {
    title,
    body: String(p.body || "").trim(),
    session: String(p.session || "").trim(),
    kind: String(p.kind || "").trim(),
    until: Date.now() + PULSE_MS,
  };
}

export function CompanionApp() {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const voicePhase = useUI((s) => s.voicePhase);
  const [booted, setBooted] = useState(false);
  const [running, setRunning] = useState(false);
  const [items, setItems] = useState<Item[]>([]);
  const [approvals, setApprovals] = useState(0);
  const [hover, setHover] = useState(false);
  const [pulse, setPulse] = useState<Pulse | null>(null);
  const [spark, setSpark] = useState(0);
  const idleAt = useRef(Date.now());
  const [now, setNow] = useState(() => Date.now());
  const sessionRef = useRef("");
  const dragged = useRef(false);

  const bump = () => { idleAt.current = Date.now(); };

  useEffect(() => {
    document.documentElement.classList.add("companion");
    document.documentElement.style.colorScheme = "normal";
    document.body.classList.remove("bg-background");
    void import("@wailsio/runtime").then((mod: any) => {
      mod.Window?.SetBackgroundColour?.(0, 0, 0, 0);
    });
    return () => document.documentElement.classList.remove("companion");
  }, []);

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => {
    if (import.meta.env.VITE_E2E) return;
    let off: (() => void) | undefined;
    let alive = true;
    void import("@wailsio/runtime").then((mod: any) => {
      const Events = mod.Events;
      if (!alive || !Events?.On) return;
      off = Events.On("yoyo:pulse", (raw: any) => {
        const next = pulseFrom(raw);
        if (!next) return;
        bump();
        if (next.session) sessionRef.current = next.session;
        setPulse(next);
        setSpark((n) => n + 1);
      });
    });
    return () => {
      alive = false;
      off?.();
    };
  }, []);

  useEffect(() => {
    let alive = true;
    const pull = async () => {
      try {
        const [ids, pending] = await Promise.all([
          api.runningIDs().catch(() => [] as string[]),
          api.approvals().catch(() => [] as { id: string }[]),
        ]);
        if (!alive) return;
        const busy = ids.length > 0;
        setRunning(busy);
        if (busy) bump();
        if (ids[0]) sessionRef.current = ids[0];
        setApprovals(pending.length);
        setBooted(true);
      } catch {
        if (alive) setBooted(true);
      }
    };
    void pull();
    const t = window.setInterval(pull, 2000);
    const offItems = subscribeItems((it) => {
      if (it.sessionId) sessionRef.current = it.sessionId;
      bump();
      setItems((prev) => {
        const next = [...prev, it];
        return next.length > KEEP ? next.slice(-KEEP) : next;
      });
    });
    const offSess = subscribeSessions(() => { void pull(); });
    return () => {
      alive = false;
      window.clearInterval(t);
      offItems();
      offSess();
    };
  }, []);

  const idleMs = running ? 0 : now - idleAt.current;
  const emotion = useMemo(() => presenceOf({
    booted,
    running,
    items,
    approvals,
    windowFocused: running || idleMs < AWAKE_MS,
    idleMs,
    voicePhase,
    now,
  }), [booted, running, items, approvals, idleMs, now, voicePhase]);

  const activePulse = pulse && now < pulse.until ? pulse : null;
  const line = activePulse
    ? (activePulse.body ? `${activePulse.title} · ${activePulse.body}` : activePulse.title)
    : presenceLine(emotion, copy.presence);
  const showLine = !!activePulse || emotion.live || hover || emotion.tips === "approval" || emotion.tips === "ask";

  useEffect(() => {
    void api.setCompanionCaption(showLine);
  }, [showLine]);
  useEffect(() => () => { void api.setCompanionCaption(false); }, []);

  const grab = useRef<{ dx: number; dy: number; sx: number; sy: number } | null>(null);

  const open = () => {
    const sid = activePulse?.session || sessionRef.current;
    if (sid) void api.raiseSession(sid);
    else void api.raiseWindow();
  };

  return (
    <div
      className="companion-stage"
      data-testid="companion-stage"
    >
      <div
        className="companion-pet-wrap"
        onPointerEnter={() => setHover(true)}
        onPointerLeave={() => setHover(false)}
        onPointerDown={(e) => {
          dragged.current = false;
          grab.current = { dx: e.screenX - window.screenX, dy: e.screenY - window.screenY, sx: e.screenX, sy: e.screenY };
          e.currentTarget.setPointerCapture(e.pointerId);
        }}
        onPointerMove={(e) => {
          const g = grab.current;
          if (!e.buttons || !g) return;
          if (Math.abs(e.screenX - g.sx) + Math.abs(e.screenY - g.sy) > 3) dragged.current = true;
          if (!/Windows/i.test(navigator.userAgent)) {
            void api.placeCompanion(e.screenX - g.dx, e.screenY - g.dy);
          }
        }}
        onPointerUp={() => { grab.current = null; }}
        onPointerCancel={() => { grab.current = null; }}
        onClick={() => { if (!dragged.current) open(); }}
      >
        <PresenceSprite
          emotionId={emotion.emotionId}
          size={148}
          reduced={reduced}
          followPointer
          screenFollow
          spark={spark}
          onActivity={bump}
        />
      </div>
      <button
        type="button"
        className={cn("companion-bubble", showLine ? "is-on" : "is-off")}
        data-testid="companion-bubble"
        onPointerEnter={() => setHover(true)}
        onPointerLeave={() => setHover(false)}
        onClick={(e) => {
          e.stopPropagation();
          open();
        }}
      >
        {line}
      </button>
    </div>
  );
}
