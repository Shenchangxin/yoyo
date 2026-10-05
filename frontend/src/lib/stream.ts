import * as api from "./client";
import type { Item } from "./protocol";
import {
  eventFingerprint,
  itemFromEvent,
  parseLiveNotice,
  replayEvents,
} from "./stream-fold";

export {
  BROWSE_TRANSCRIPT_TURNS,
  HOT_TRANSCRIPT_TURNS,
  UI_TEXT_CAP,
  closesAssistant,
  dropTrailingErrors,
  dropTurnErrors,
  foldTurnErrors,
  eventFingerprint,
  foldLiveIntoSeed,
  isTranscriptNoise,
  itemFromEvent,
  lastUserTurns,
  mergeItem,
  mergePendingUsers,
  parseLiveNotice,
  replayEvents,
  unwrapEvent,
  userTurnCount,
} from "./stream-fold";

export function emptyTrajectoryPage(): TrajectoryPageView {
  return { events: [], headSeq: 0, tailSeq: 0, older: false, hubSeq: 0 };
}

async function loadSeedPage(sessionId: string): Promise<TrajectoryPageView> {
  try {
    return await api.trajectoryPage(sessionId);
  } catch {
    // Never dump Trajectory() into the desktop webview. An empty window is
    // recoverable via live notices; a 10MB JSONL is not.
    return emptyTrajectoryPage();
  }
}

export type TrajectoryPageView = {
  events: any[];
  headSeq: number;
  tailSeq: number;
  older: boolean;
  hubSeq: number;
};

export function subscribeSession(
  sessionId: string,
  onItem: (item: Item) => void,
  onSeed?: (items: Item[], page?: TrajectoryPageView) => void,
): () => void {
  let alive = true;
  let stopLive = () => {};
  let seeded = false;
  let hubSeq = 0;
  let pulling = false;
  let pullAgain = false;
  const buffered: { raw: any; item: Item; seq: number }[] = [];
  const seen = new Set<string>();

  const applyItem = (raw: any, item: Item) => {
    if (seen.size > 4000) seen.clear();
    const fp = eventFingerprint(raw);
    if (seen.has(fp)) return;
    seen.add(fp);
    onItem(item);
  };

  const applySeed = (raws: any[], page?: TrajectoryPageView) => {
    (raws || []).forEach((raw) => seen.add(eventFingerprint(raw)));
    const items = replayEvents(raws || []);
    if (onSeed) onSeed(items, page);
    else items.forEach(onItem);
    seeded = true;
    for (const row of buffered) {
      if (row.seq > 0 && row.seq <= hubSeq) continue;
      const fp = eventFingerprint(row.raw);
      if (seen.has(fp)) continue;
      seen.add(fp);
      onItem(row.item);
      if (row.seq > hubSeq) hubSeq = row.seq;
    }
    buffered.length = 0;
  };

  const pullSince = async (after: number) => {
    if (!alive || after < 0) return;
    if (pulling) {
      pullAgain = true;
      return;
    }
    pulling = true;
    try {
      let from = after;
      do {
        pullAgain = false;
        const batch = await api.liveSince(sessionId, from).catch(() => []);
        if (!alive) return;
        for (const raw of batch) {
          const notice = parseLiveNotice(raw);
          const event = notice.event;
          if (notice.seq > hubSeq) hubSeq = notice.seq;
          if (!event) continue;
          const item = itemFromEvent(event);
          if (item.sessionId && item.sessionId !== sessionId) continue;
          applyItem(event, item);
        }
        from = hubSeq;
      } while (pullAgain && alive);
    } finally {
      pulling = false;
    }
  };

  const onLive = (raw: any) => {
    const notice = parseLiveNotice(raw);
    if (notice.sessionId && notice.sessionId !== sessionId) return;
    if (!seeded) {
      if (!notice.event) return;
      const item = itemFromEvent(notice.event);
      if (item.sessionId && item.sessionId !== sessionId) return;
      buffered.push({ raw: notice.event, item, seq: notice.seq });
      return;
    }
    if (notice.seq > 0 && notice.seq <= hubSeq) return;
    if (!notice.event) {
      void pullSince(hubSeq);
      return;
    }
    const item = itemFromEvent(notice.event);
    if (item.sessionId && item.sessionId !== sessionId) return;
    const gap = notice.seq > 0 && notice.seq > hubSeq + 1;
    if (gap) void pullSince(hubSeq);
    if (notice.seq > hubSeq) hubSeq = notice.seq;
    const event = notice.event.seq || notice.event.Seq ? notice.event : { ...notice.event, seq: notice.seq };
    applyItem(event, item);
  };

  (async () => {
    const Events = await wailsEvents();
    if (!alive) return;
    if (Events?.On) {
      const off = Events.On("yoyo:item", (e: any) => onLive(e));
      stopLive = typeof off === "function" ? off : () => {};
      const page = await loadSeedPage(sessionId);
      if (!alive) return;
      hubSeq = page.hubSeq || 0;
      applySeed(page.events, page);
      if (hubSeq > 0) void pullSince(hubSeq);
      return;
    }

    if (typeof EventSource !== "undefined") {
      const es = new EventSource(`/api/sessions/${sessionId}/events`);
      es.onmessage = (msg) => {
        try {
          onLive(JSON.parse(msg.data));
        } catch {
          /* ignore malformed */
        }
      };
      stopLive = () => es.close();
      const page = await loadSeedPage(sessionId);
      if (!alive) {
        es.close();
        return;
      }
      hubSeq = page.hubSeq || 0;
      applySeed(page.events, page);
      return;
    }

    const tick = async () => {
      const page = await loadSeedPage(sessionId);
      if (!alive) return;
      hubSeq = page.hubSeq || hubSeq;
      applySeed(page.events, page);
    };
    await tick();
    const t = window.setInterval(tick, 2500);
    stopLive = () => window.clearInterval(t);
  })();

  return () => {
    alive = false;
    stopLive();
  };
}

export function subscribeStopped(onStop: (sessionId: string) => void): () => void {
  let off: (() => void) | undefined;
  let alive = true;
  (async () => {
    const Events = await wailsEvents();
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:stopped", (e: any) => {
      const src = Array.isArray(e) ? e[0] : e?.data ?? e;
      const id = String(src?.session_id || src?.SessionID || src || "");
      if (id) onStop(id);
    });
  })();
  return () => {
    alive = false;
    off?.();
  };
}

export function subscribeItems(onItem: (item: Item) => void): () => void {
  let off: (() => void) | undefined;
  let alive = true;
  (async () => {
    const Events = await wailsEvents();
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:item", (e: any) => {
      const notice = parseLiveNotice(e);
      if (!notice.event) return;
      onItem(itemFromEvent(notice.event));
    });
  })();
  return () => {
    alive = false;
    off?.();
  };
}

export function subscribeSessions(onChange: () => void): () => void {
  let off: any;
  let alive = true;
  (async () => {
    const Events = await wailsEvents();
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:sessions", () => onChange());
  })();
  return () => {
    alive = false;
    if (typeof off === "function") off();
  };
}

async function wailsEvents(): Promise<any | null> {
  const viteE2E = (import.meta as any)?.env?.VITE_E2E;
  if (viteE2E) return null;
  if (typeof window !== "undefined" && !(window as any)._wails?.environment?.OS) return null;
  try {
    const mod: any = await import("@wailsio/runtime");
    return mod.Events || mod.events || null;
  } catch {
    return null;
  }
}
