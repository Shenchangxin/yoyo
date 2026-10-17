import * as api from "./client";
import type { Item } from "./protocol";
import {
  eventFingerprint,
  itemFromEvent,
  parseLiveNotice,
  replayEvents,
} from "./stream-fold";
import { classifyLive } from "./stream-cursor";

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
  withLiveTail,
  parseLiveNotice,
  replayEvents,
  unwrapEvent,
  eventText,
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

/** `resync`: the window on screen is live; merge into it instead of replacing it. */
export type SeedOptions = { resync: boolean };

// Open session streams by id. While a turn is known to be running the UI asks
// for a catch-up, so a dropped tail event is repaired without waiting for the
// next notice to expose the gap.
const catchUps = new Map<string, Set<() => void>>();

export function catchUpSession(sessionId: string): void {
  const fns = catchUps.get(sessionId);
  if (fns) for (const fn of fns) fn();
}

export function subscribeSession(
  sessionId: string,
  onItem: (item: Item) => void,
  onSeed?: (items: Item[], page: TrajectoryPageView | undefined, opts: SeedOptions) => void,
): () => void {
  let alive = true;
  let stopLive = () => {};
  // Global hub seq applied so far. Advanced only by in-order notices, pulls and
  // seeds; the rules live in stream-cursor.ts.
  let cursor = 0;
  let seeded = false;
  let pulling = false;
  let pullAgain = false;
  let resyncing = false;
  const seen = new Set<string>();
  const lastIndex = new Map<string, number>();

  const noteFrame = (item: Item, frame?: { index?: number; index_from?: number }) => {
    const from = frame?.index_from || 0;
    if (!from || !item.delta) return false;
    const rid = String(item.payload?.id || item.payload?.round || item.key);
    const have = lastIndex.get(rid) || 0;
    const gapped = have > 0 && from > have + 1;
    lastIndex.set(rid, Math.max(have, frame?.index || from));
    return gapped;
  };

  const applyItem = (raw: any, item: Item) => {
    if (seen.size > 4000) seen.clear();
    const fp = eventFingerprint(raw);
    if (seen.has(fp)) return;
    seen.add(fp);
    onItem(item);
  };

  const applySeed = (raws: any[], page: TrajectoryPageView | undefined, resync: boolean) => {
    (raws || []).forEach((raw) => seen.add(eventFingerprint(raw)));
    const items = replayEvents(raws || []);
    lastIndex.clear();
    if (onSeed) onSeed(items, page, { resync });
    else items.forEach(onItem);
    seeded = true;
  };

  // One event from a pull, in seq order. Pulled events are whole trace events,
  // so their seq travels on the event itself.
  const applyPulled = (raw: any) => {
    const notice = parseLiveNotice(raw);
    const seq = notice.seq;
    if (seq > 0 && seq <= cursor) return; // already applied via a live notice
    if (seq > cursor) cursor = seq;
    if (!notice.event) return;
    const item = itemFromEvent(notice.event);
    if (item.sessionId && item.sessionId !== sessionId) return;
    const event = notice.event.seq || notice.event.Seq ? notice.event : { ...notice.event, seq };
    applyItem(event, item);
  };

  // Apply the ring tail after the cursor. A batch that is not complete cannot
  // be trusted, so it is repaired by a resync. The follow-up pull of a resync
  // passes mayResync=false: it is best effort, which means a ring that keeps
  // overflowing cannot make the two loop against each other.
  const pullSince = async (mayResync = true): Promise<void> => {
    if (!alive) return;
    if (pulling) {
      pullAgain = true;
      return;
    }
    pulling = true;
    let gapped = false;
    try {
      do {
        pullAgain = false;
        const batch = await api.liveSync(sessionId, cursor).catch(() => null);
        if (!alive) return;
        if (!batch) break; // transport error: keep the cursor, a later notice retries
        if (batch.gap && mayResync) {
          gapped = true;
          break;
        }
        for (const raw of batch.events) applyPulled(raw);
        cursor = Math.max(cursor, batch.head);
      } while (pullAgain && alive);
    } finally {
      pulling = false;
    }
    if (gapped && alive) void resync();
  };

  // The trajectory is the durable record. When the live tail has a hole, reload
  // it and merge it with what is on screen, so streamed text in flight survives.
  const resync = async (): Promise<void> => {
    if (resyncing || !alive) return;
    resyncing = true;
    try {
      const page: TrajectoryPageView | null = await api.trajectoryPage(sessionId).catch(() => null);
      if (!alive || !page) return;
      // Take the page's cursor even if it is lower: a restarted hub numbers from
      // zero again, and keeping a higher cursor would freeze the stream.
      cursor = page.hubSeq || 0;
      applySeed(page.events, page, true);
    } finally {
      resyncing = false;
    }
    if (alive) void pullSince(false);
  };

  const onLive = (raw: any) => {
    // Nothing is applied before the seed. The seed's pull replays every event
    // after the page's hubSeq from the ring, including the ones that arrive now.
    if (!seeded) return;
    const notice = parseLiveNotice(raw);
    const own = !notice.sessionId || notice.sessionId === sessionId;
    switch (classifyLive(cursor, notice.seq, own, !!notice.event)) {
      case "stale":
        return;
      case "pull":
        void pullSince();
        return;
      case "advance":
        cursor = Math.max(cursor, notice.seq);
        return;
      case "apply": {
        const event = notice.event.seq || notice.event.Seq ? notice.event : { ...notice.event, seq: notice.seq };
        const item = itemFromEvent(event);
        cursor = Math.max(cursor, notice.seq);
        if (item.sessionId && item.sessionId !== sessionId) return;
        if (noteFrame(item, notice.frame)) {
          void pullSince();
        }
        applyItem(event, item);
        return;
      }
    }
  };

  const open = async (): Promise<void> => {
    const page = await loadSeedPage(sessionId);
    if (!alive) return;
    cursor = page.hubSeq || 0;
    applySeed(page.events, page, false);
    void pullSince();
  };

  (async () => {
    const Events = await wailsEvents();
    if (!alive) return;
    if (Events?.On) {
      const off = Events.On("yoyo:item", (e: any) => onLive(e));
      stopLive = typeof off === "function" ? off : () => {};
      await open();
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
      await open();
      if (!alive) es.close();
      return;
    }

    const tick = async () => {
      const page = await loadSeedPage(sessionId);
      if (!alive) return;
      applySeed(page.events, page, false);
    };
    await tick();
    const t = window.setInterval(tick, 2500);
    stopLive = () => window.clearInterval(t);
  })();

  const catchUp = () => {
    if (alive && seeded) void pullSince();
  };
  const registry = catchUps.get(sessionId) ?? new Set<() => void>();
  catchUps.set(sessionId, registry);
  registry.add(catchUp);

  return () => {
    alive = false;
    stopLive();
    registry.delete(catchUp);
    if (registry.size === 0 && catchUps.get(sessionId) === registry) catchUps.delete(sessionId);
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
