/**
 * 拥有：对学习数据变化的订阅。服务端每提交一次写入就推一条 learn 事件，界面据此重读那一块。
 * 不拥有：任何学习状态。不在桌面壳里时什么也不做，调用方靠进入页面时的读取兜底。
 */

import { unwrapEvent } from "./stream-fold";

/** LearnChange 是一次变化：哪一块、哪个对象、变化的种类。 */
export type LearnChange = { scope: string; id: string; kind: string };

/** learnChangeOf 从 yoyo:item 的事件里取出学习变化。不是学习事件时返回 null。同进程桌面的事件在 event 里，隔离进程的在顶层。 */
export function learnChangeOf(raw: unknown): LearnChange | null {
  const e: any = unwrapEvent(raw);
  if (!e || typeof e !== "object") return null;
  const nested = e.event && typeof e.event === "object" ? e.event : null;
  const kind = String(e.item_kind ?? e.ItemKind ?? nested?.item_kind ?? nested?.ItemKind ?? "");
  if (kind !== "learn") return null;
  const payload = e.payload ?? e.Payload ?? nested?.payload ?? nested?.Payload;
  if (!payload || typeof payload !== "object") return null;
  return { scope: String(payload.scope || ""), id: String(payload.id || ""), kind: String(payload.kind || "") };
}

/** 尝试拿到 Wails 事件。不在桌面壳里时返回空。 */
async function wailsEvents(): Promise<{ On?: (name: string, cb: (e: unknown) => void) => () => void } | null> {
  try {
    const mod: any = await import("@wailsio/runtime");
    return mod.Events || null;
  } catch {
    return null;
  }
}

const SAY_EVENT = "yoyo:learn-say";

/** LearnSay 是学习者在题卡上说的一句话，要交给那段导师对话。 */
export type LearnSay = { sessionId: string; text: string };

/** emitLearnSay 把题卡上的作答交给导师对话。对话里的卡片够不着发送，所以经窗口事件转一手。 */
export function emitLearnSay(say: LearnSay): void {
  if (typeof window === "undefined" || !say.text.trim()) return;
  window.dispatchEvent(new CustomEvent<LearnSay>(SAY_EVENT, { detail: say }));
}

/** onLearnSay 听题卡上的作答，交给 handler。返回退订函数。 */
export function onLearnSay(handler: (say: LearnSay) => void): () => void {
  if (typeof window === "undefined") return () => {};
  const listener = (e: Event) => {
    const detail = (e as CustomEvent<LearnSay>).detail;
    if (detail && typeof detail.text === "string") handler(detail);
  };
  window.addEventListener(SAY_EVENT, listener);
  return () => window.removeEventListener(SAY_EVENT, listener);
}

/** subscribeLearn 听学习变化，把每一条交给 onChange。返回退订函数。 */
export function subscribeLearn(onChange: (change: LearnChange) => void): () => void {
  let off: (() => void) | undefined;
  let alive = true;
  void wailsEvents().then(/** Events 没有 On 就不订。 */ (Events) => {
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:item", /** ev 不是学习事件时跳过。 */ (ev) => {
      const change = learnChangeOf(ev);
      if (change) onChange(change);
    });
  });
  return () => {
    alive = false;
    off?.();
  };
}
