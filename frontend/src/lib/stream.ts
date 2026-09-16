import * as api from "./client";
import { pick, str } from "./normalize";
import type { Item, ItemType } from "./protocol";

function payload(raw: any): Record<string, any> {
  const p = pick(raw, "payload", "Payload");
  return p && typeof p === "object" ? p : {};
}

export function itemFromEvent(raw: any, idx = 0): Item {
  const src = unwrapEvent(raw);
  const p = payload(src);
  const type = str(pick(src, "type", "Type"), "system") as ItemType;
  const sessionId = str(pick(src, "session_id", "SessionID", "sessionId"));
  const ts = str(pick(src, "ts", "TS"));
  const text = str(pick(p, "text", "content", "error"));
  return {
    key: `${sessionId}:${ts}:${type}:${idx}:${str(pick(p, "name", "id"))}`,
    type,
    sessionId,
    source: str(pick(src, "source", "Source")),
    ts,
    text,
    name: str(pick(p, "name", "action")),
    delta: !!pick(p, "delta"),
    payload: p,
  };
}

export function mergeItem(list: Item[], ev: Item): Item[] {
  if (ev.type === "turn_end" || ev.type === "system") return list;
  if (ev.type === "user") {
    const next = list.filter((x) => !(x.source === "ui" && x.type === "user" && x.text === ev.text));
    if (ev.key && next.some((x) => x.key === ev.key)) return next;
    return [...next, ev];
  }
  if (!(ev.type === "assistant" && ev.delta)) {
    if (ev.key && list.some((x) => x.key === ev.key)) return list;
  }
  if (ev.type === "assistant" && ev.text) {
    const next = list.slice();
    for (let i = next.length - 1; i >= 0; i--) {
      if (next[i].type === "assistant") {
        next[i] = {
          ...next[i],
          text: ev.delta ? next[i].text + ev.text : ev.text,
          payload: { ...next[i].payload, ...ev.payload },
        };
        return next;
      }
    }
    return [...next, ev];
  }
  return [...list, ev];
}

function unwrapEvent(e: any): any {
  if (e == null) return e;
  if (e.data !== undefined && e.data !== null && typeof e.data === "object") return unwrapEvent(e.data);
  if (Array.isArray(e) && e.length === 1) return unwrapEvent(e[0]);
  return e;
}

export function subscribeSession(
  sessionId: string,
  onItem: (item: Item) => void,
): () => void {
  let alive = true;
  let stopLive = () => {};

  (async () => {
    const Events = await wailsEvents();
    if (!alive) return;
    if (Events?.On) {
      const seed = await api.trajectory(sessionId).catch(() => []);
      if (!alive) return;
      seed.forEach((raw, i) => onItem(itemFromEvent(raw, i)));
      const off = Events.On("yoyo:item", (e: any) => {
        const item = itemFromEvent(e);
        if (item.sessionId && item.sessionId !== sessionId) return;
        onItem(item);
      });
      stopLive = typeof off === "function" ? off : () => {};
      return;
    }

    if (typeof EventSource !== "undefined") {
      const es = new EventSource(`/api/sessions/${sessionId}/events`);
      es.onmessage = (msg) => {
        try {
          onItem(itemFromEvent(JSON.parse(msg.data)));
        } catch {
          /* ignore malformed */
        }
      };
      stopLive = () => es.close();
      return;
    }

    const tick = async () => {
      const seed = await api.trajectory(sessionId).catch(() => []);
      if (!alive) return;
      seed.forEach((raw, i) => onItem(itemFromEvent(raw, i)));
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
  if (import.meta.env.VITE_E2E) return null;
  if (typeof window !== "undefined" && !(window as any)._wails?.environment?.OS) return null;
  try {
    const mod: any = await import("@wailsio/runtime");
    return mod.Events || mod.events || null;
  } catch {
    return null;
  }
}
