import * as api from "./client";
import type { Item } from "./protocol";
import {
  eventFingerprint,
  itemFromEvent,
  replayEvents,
} from "./stream-fold";

export {
  closesAssistant,
  dropTrailingErrors,
  eventFingerprint,
  isTranscriptNoise,
  itemFromEvent,
  lastUserTurns,
  mergeItem,
  mergePendingUsers,
  replayEvents,
  unwrapEvent,
} from "./stream-fold";

export function subscribeSession(
  sessionId: string,
  onItem: (item: Item) => void,
  onSeed?: (items: Item[]) => void,
): () => void {
  let alive = true;
  let stopLive = () => {};
  let seeded = false;
  const buffered: { raw: any; item: Item }[] = [];
  const seen = new Set<string>();

  const applySeed = (raws: any[]) => {
    (raws || []).forEach((raw) => seen.add(eventFingerprint(raw)));
    const items = replayEvents(raws || []);
    if (onSeed) onSeed(items);
    else items.forEach(onItem);
    seeded = true;
    for (const row of buffered) {
      const fp = eventFingerprint(row.raw);
      if (seen.has(fp)) continue;
      seen.add(fp);
      onItem(row.item);
    }
    buffered.length = 0;
  };

  const onLive = (raw: any) => {
    const item = itemFromEvent(raw);
    if (item.sessionId && item.sessionId !== sessionId) return;
    if (!seeded) {
      buffered.push({ raw, item });
      return;
    }
    const fp = eventFingerprint(raw);
    if (seen.has(fp)) return;
    seen.add(fp);
    onItem(item);
  };

  (async () => {
    const Events = await wailsEvents();
    if (!alive) return;
    if (Events?.On) {
      const off = Events.On("yoyo:item", (e: any) => onLive(e));
      stopLive = typeof off === "function" ? off : () => {};
      const seed = await api.trajectory(sessionId).catch(() => []);
      if (!alive) return;
      applySeed(seed);
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
      const seed = await api.trajectory(sessionId).catch(() => []);
      if (!alive) {
        es.close();
        return;
      }
      applySeed(seed);
      return;
    }

    const tick = async () => {
      const seed = await api.trajectory(sessionId).catch(() => []);
      if (!alive) return;
      applySeed(seed);
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

export function subscribeItems(onItem: (item: Item) => void): () => void {
  let off: (() => void) | undefined;
  let alive = true;
  (async () => {
    const Events = await wailsEvents();
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:item", (e: any) => onItem(itemFromEvent(e)));
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
