import { reportFrontend } from "./client";

type FrontEv = { level: string; msg: string; stack?: string; source?: string };

const buf: FrontEv[] = [];
let timer: number | undefined;
let last = 0;
const MIN_MS = 400;

function flush() {
  timer = undefined;
  if (!buf.length) return;
  const events = buf.splice(0, buf.length);
  void reportFrontend(events).catch(() => {});
}

export function reportRenderer(ev: FrontEv) {
  const now = Date.now();
  if (now - last < 40 && buf.length > 24) return;
  last = now;
  buf.push({
    level: ev.level || "error",
    msg: String(ev.msg || "").slice(0, 2000),
    stack: ev.stack ? String(ev.stack).slice(0, 4000) : undefined,
    source: ev.source,
  });
  if (buf.length >= 8) {
    flush();
    return;
  }
  if (timer == null) timer = window.setTimeout(flush, MIN_MS);
}

export function installFrontendLogBridge() {
  window.addEventListener("error", (e) => {
    reportRenderer({
      level: "error",
      msg: e.message || "window.onerror",
      stack: e.error?.stack,
      source: e.filename,
    });
  });
  window.addEventListener("unhandledrejection", (e) => {
    const reason = e.reason;
    reportRenderer({
      level: "error",
      msg: reason instanceof Error ? reason.message : String(reason),
      stack: reason instanceof Error ? reason.stack : undefined,
      source: "unhandledrejection",
    });
  });
}
