import { useEffect, useLayoutEffect, useMemo, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { Phone } from "lucide-react";
import { useBusyMs } from "./busy";
import { presenceLine, presenceOf } from "./director";
import { PresenceSprite } from "./PresenceRuntime";
import {
  bubbleLine,
  bubbleParts,
  itemEndsWork,
  itemStartsWork,
  mergePulse,
  pulseFrom,
  showLoading,
  type CompanionPulse,
} from "./companion-pulse";
import { useMotionReduced } from "../../lib/motion";
import { subscribeItems, subscribeSessions } from "../../lib/stream";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { IdolCall } from "../idol/IdolCall";
import { warmStand } from "../idol/stand/StandView";
import { primeCallAudio } from "../idol/useIdolCall";
import { companionCallOn } from "./companion-call";
import { imageURL } from "../idol/model";
import { skillCallLabel } from "../../lib/tool-summary";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";

const KEEP = 80;
const AWAKE_MS = 12_000;

export function CompanionApp() {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const voicePhase = useUI((s) => s.voicePhase);
  const [booted, setBooted] = useState(false);
  const [running, setRunning] = useState(false);
  const [items, setItems] = useState<Item[]>([]);
  const [approvals, setApprovals] = useState(0);
  const [pulse, setPulse] = useState<CompanionPulse | null>(null);
  const [spark, setSpark] = useState(0);
  const [inCall, setInCall] = useState(false);
  const [reach, setReach] = useState("");
  const [face, setFace] = useState("");
  const [pinnedID, setPinnedID] = useState("");
  const [idolKind, setIdolKind] = useState("");
  const [idolColor, setIdolColor] = useState("#c47b8a");
  const [idolName, setIdolName] = useState("");
  const [idolStand, setIdolStand] = useState("");
  const [paused, setPaused] = useState(false);
  const idleAt = useRef(Date.now());
  const [now, setNow] = useState(() => Date.now());
  const sessionRef = useRef("");
  const dragged = useRef(false);
  const pulseChannel = useRef(false);

  const bump = () => { idleAt.current = Date.now(); };

  const applyPulse = (next: CompanionPulse) => {
    bump();
    if (next.session) sessionRef.current = next.session;
    setPulse((cur) => mergePulse(cur, next));
    setSpark((n) => n + 1);
  };

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
      off = Events.On("yoyo:call", (raw: any) => {
        setInCall(companionCallOn(raw));
      });
    });
    return () => {
      alive = false;
      off?.();
    };
  }, []);

  useEffect(() => {
    const t = window.setInterval(() => {
      void api.health().then((h) => {
        setPaused(!!h.paused);
        if (h.paused) setInCall(false);
      }).catch(() => {});
    }, 2000);
    return () => window.clearInterval(t);
  }, []);

  useEffect(() => {
    if (inCall) return;
    const tick = () => {
      void api.idolCall("reach", {}).then((r) => {
        const line = String(r?.caption || r?.speak || "").trim();
        if (line) setReach(line);
      }).catch(() => {});
    };
    const id = window.setInterval(tick, 45000);
    return () => window.clearInterval(id);
  }, [inCall]);

  useEffect(() => {
    warmStand(idolStand);
  }, [idolStand]);

  useEffect(() => {
    const load = () => {
      void api.idolCall("snapshot").then((s) => {
        const pinned = s?.pinned;
        setPinnedID(pinned?.id ? String(pinned.id) : "");
        setIdolKind(pinned?.kind ? String(pinned.kind) : "");
        setIdolColor(pinned?.color ? String(pinned.color) : "#c47b8a");
        setIdolName(pinned?.name ? String(pinned.name) : "");
        setIdolStand(pinned?.outfit_id || pinned?.OutfitID ? String(pinned.outfit_id || pinned.OutfitID) : "");
        setFace(pinned && s.portrait ? String(s.portrait) : "");
      }).catch(() => {});
    };
    load();
    if (import.meta.env.VITE_E2E) return;
    let off: (() => void) | undefined;
    let alive = true;
    void import("@wailsio/runtime").then((mod: any) => {
      const Events = mod.Events;
      if (!alive || !Events?.On) return;
      off = Events.On("yoyo:idol", () => load());
    });
    return () => {
      alive = false;
      off?.();
    };
  }, [inCall]);

  useEffect(() => {
    if (import.meta.env.VITE_E2E) return;
    let off: (() => void) | undefined;
    let alive = true;
    void import("@wailsio/runtime").then((mod: any) => {
      const Events = mod.Events;
      if (!alive || !Events?.On) return;
      pulseChannel.current = true;
      off = Events.On("yoyo:pulse", (raw: any) => {
        const next = pulseFrom(raw);
        if (!next) return;
        applyPulse(next);
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
      if (itemStartsWork(it)) setRunning(true);
      if (it.type === "turn_end") setRunning(false);
      if (pulseChannel.current) {
        if (itemEndsWork(it)) {
          setPulse((cur) => (cur?.kind === "loading" ? { ...cur, until: Date.now() } : cur));
        }
        return;
      }
      if (itemStartsWork(it) && it.type !== "assistant" && it.type !== "reasoning") {
        applyPulse({
          title: String(skillCallLabel(it) || it.name || it.text || copy.presence.loading).trim().slice(0, 96) || copy.presence.loading,
          body: copy.presence.loading,
          session: it.sessionId || sessionRef.current,
          kind: "loading",
          until: Date.now() + 16_000,
        });
      }
      if (itemEndsWork(it)) {
        setPulse((cur) => (cur?.kind === "loading" ? { ...cur, until: Date.now() } : cur));
      }
    });
    const offSess = subscribeSessions(() => { void pull(); });
    return () => {
      alive = false;
      window.clearInterval(t);
      offItems();
      offSess();
    };
  }, [copy.presence.loading]);

  const idleMs = running ? 0 : now - idleAt.current;
  const activePulse = pulse && now < pulse.until ? pulse : null;
  const reachPulse = !activePulse && reach ? { title: copy.idol.reach, body: reach, session: "", kind: "reach", until: now + 20000 } : null;
  const shownPulse = activePulse || reachPulse;
  const loading = showLoading(running, pulse, approvals, now);
  const busyMs = useBusyMs(running || loading, `${running}:${loading}`);
  const interrupted = !running && items.some((it) => it.type === "error" && /interrupt|cancel|stopped/i.test(`${it.text || ""} ${it.name || ""}`));
  const emotion = useMemo(() => presenceOf({
    booted,
    running,
    items,
    approvals,
    windowFocused: running || idleMs < AWAKE_MS,
    idleMs,
    voicePhase,
    now,
    moduleLoading: loading && !running,
    interrupted,
    busyMs,
  }), [booted, running, items, approvals, idleMs, now, voicePhase, loading, interrupted, busyMs]);

  const emotionLine = presenceLine(emotion, copy.presence);
  const bubble = bubbleParts({
    approvals,
    pulse: shownPulse,
    approvalLine: copy.presence.approval,
    emotionLine,
    now,
  });
  const line = bubbleLine({
    approvals,
    pulse: shownPulse,
    approvalLine: copy.presence.approval,
    emotionLine,
    now,
  });
  const showLine = !!shownPulse || emotion.live || loading || emotion.tips === "approval" || emotion.tips === "ask" || approvals > 0;
  const bubbleRef = useRef<HTMLButtonElement>(null);
  const scrollRef = useRef<HTMLSpanElement>(null);

  useLayoutEffect(() => {
    const el = bubbleRef.current;
    if (!showLine || !el) {
      void api.setCompanionCaption(false);
      return;
    }
    const publish = () => {
      const box = scrollRef.current;
      if (box) box.scrollTop = box.scrollHeight;
      void api.setCompanionCaptionSize(el.offsetWidth, el.offsetHeight);
      void api.setCompanionCaption(true);
    };
    publish();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(publish);
    ro.observe(el);
    return () => ro.disconnect();
  }, [showLine, line, loading]);
  useEffect(() => () => { void api.setCompanionCaption(false); }, []);

  const grab = useRef<{ dx: number; dy: number; sx: number; sy: number } | null>(null);
  const dragStart = (e: ReactPointerEvent<HTMLElement>) => {
    if (e.button !== 0) return;
    if ((e.target as HTMLElement).closest?.("button, input, textarea, a, [data-companion-call]")) return;
    dragged.current = false;
    grab.current = { dx: e.screenX - window.screenX, dy: e.screenY - window.screenY, sx: e.screenX, sy: e.screenY };
    e.currentTarget.setPointerCapture(e.pointerId);
  };
  const dragMove = (e: ReactPointerEvent<HTMLElement>) => {
    const g = grab.current;
    if (!e.buttons || !g) return;
    if (Math.abs(e.screenX - g.sx) + Math.abs(e.screenY - g.sy) > 3) dragged.current = true;
    if (!/Windows/i.test(navigator.userAgent)) {
      void api.placeCompanion(e.screenX - g.dx, e.screenY - g.dy);
    }
  };
  const dragEnd = () => { grab.current = null; };

  const open = () => {
    const sid = activePulse?.session || sessionRef.current;
    if (sid) void api.raiseSession(sid);
    else void api.raiseWindow();
  };

  if (inCall) {
    return (
      <div
        className="companion-stage companion-stage-call"
        data-testid="companion-stage"
        data-context-menu="yoyo-companion"
        onPointerDown={dragStart}
        onPointerMove={dragMove}
        onPointerUp={dragEnd}
        onPointerCancel={dragEnd}
      >
        <IdolCall
          paused={paused}
          approvals={approvals}
          name={idolName}
          color={idolColor}
          portrait={face}
          kind={idolKind}
          stand={idolStand}
          onHangup={() => setInCall(false)}
        />
      </div>
    );
  }

  return (
    <div
      className="companion-stage"
      data-testid="companion-stage"
      data-context-menu="yoyo-companion"
    >
      <div
        className="companion-pet-wrap"
        data-testid="companion-pet-wrap"
        data-context-menu="yoyo-companion"
        data-emotion={emotion.emotionId}
        data-live={emotion.live ? "true" : undefined}
        data-loading={loading ? "true" : undefined}
        onPointerEnter={() => bump()}
        onPointerDown={dragStart}
        onPointerMove={dragMove}
        onPointerUp={dragEnd}
        onPointerCancel={dragEnd}
        onClick={() => { if (!dragged.current) open(); }}
      >
        {face ? (
          <img alt="" className="companion-portrait" src={imageURL(face)} />
        ) : pinnedID && idolKind !== "pet" ? (
          <span className="companion-letter" style={{ background: idolColor }}>{Array.from(idolName || "偶")[0]}</span>
        ) : (
        <PresenceSprite
          emotionId={emotion.emotionId}
          size={168}
          reduced={reduced}
          followPointer
          screenFollow
          spark={spark}
          onActivity={bump}
        />
        )}
        {pinnedID && !paused ? (
          <button
            type="button"
            className="companion-call"
            data-testid="companion-call"
            data-companion-call=""
            aria-label={copy.idol.call}
            title={copy.idol.call}
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              primeCallAudio();
              setInCall(true);
            }}
          >
            <Phone className="size-4" strokeWidth={1.75} />
          </button>
        ) : null}
      </div>
      <button
        ref={bubbleRef}
        type="button"
        className={cn("companion-bubble", showLine ? "is-on" : "is-off", loading && "is-loading")}
        data-testid="companion-bubble"
        data-loading={loading ? "true" : undefined}
        data-kind={approvals > 0 ? "approval" : (activePulse?.kind || emotion.tips || undefined)}
        aria-busy={loading}
        aria-live="polite"
        onWheel={(e) => e.stopPropagation()}
        onClick={(e) => {
          e.stopPropagation();
          open();
        }}
      >
        <span
          className="companion-bubble-text"
          key={`${bubble.kicker}|${activePulse?.kind || emotion.tips || ""}|${loading ? "load" : "idle"}|${approvals}`}
        >
          {bubble.kicker ? <span className="companion-bubble-kicker">{bubble.kicker}</span> : null}
          <span ref={scrollRef} className="companion-bubble-scroll">
            <span className="companion-bubble-copy">{bubble.copy}</span>
          </span>
        </span>
        {loading ? <span className="companion-dots" aria-hidden><i /><i /><i /></span> : null}
      </button>
    </div>
  );
}
