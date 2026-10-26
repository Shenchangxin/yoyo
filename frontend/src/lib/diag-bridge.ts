/**
 * 拥有：把渲染进程的错误攒一批，再交给前端上报。
 * 不拥有：上报请求本身。那是 client 的事。
 */

import { reportFrontend } from "./client";

type FrontEv = { level: string; msg: string; stack?: string; source?: string };

const buf: FrontEv[] = [];
let timer: number | undefined;
let last = 0;
const MIN_MS = 400;

/** 把缓冲区里的事件一次发出，并清空缓冲。 */
function flush() {
  timer = undefined;
  if (!buf.length) return;
  const events = buf.splice(0, buf.length);
  void reportFrontend(events).catch(() => {});
}

/** ResizeObserver 循环告警不上报。msg 传给 test。 */
function ignoreFront(msg: string) {
  return /ResizeObserver loop/i.test(msg);
}

/** reportRenderer 收入一条前端事件。距上一条不足 40 毫秒且缓冲已超过 24 条时丢掉。满 8 条立即发送，否则最多等 400 毫秒。正文截到 2000，堆栈截到 4000。ev 读取 msg、level、stack、source。 */
export function reportRenderer(ev: FrontEv) {
  if (ignoreFront(ev.msg || "")) return;
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

/** installFrontendLogBridge 监听 window 的 error 和未处理的 rejection，并送去上报。 */
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
