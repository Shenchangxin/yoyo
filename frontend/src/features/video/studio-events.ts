/**
 * 本文件拥有视频工作台对后台事件的订阅。配音、合成、生图、一键成片有进展时服务端推 studio_state，成片台做画面时推 film_state，界面据此重读。
 * 不拥有任何工程状态，只把编号和载荷交给回调。不在桌面壳里时什么也不做，调用方靠兜底轮询。
 */
import { videoItemOf } from "./video-item";

/** 尝试拿到 Wails 事件。不在桌面壳里时返回空。 */
async function wailsEvents(): Promise<{ On?: (name: string, cb: (e: unknown) => void) => () => void } | null> {
  try {
    const mod: any = await import("@wailsio/runtime");
    return mod.Events || null;
  } catch {
    return null;
  }
}

/** subscribeVideoEvents 听 yoyo:item 里种类在 kinds 里、载荷带编号的事件，把种类、编号和载荷交给 onEvent。返回退订函数。 */
export function subscribeVideoEvents(kinds: string[], onEvent: (kind: string, id: string, payload: Record<string, unknown>) => void): () => void {
  let off: (() => void) | undefined;
  let alive = true;
  void wailsEvents().then(/** Events 没有 On 就不订。 */ (Events) => {
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:item", /** ev 用 videoItemOf 拆开，只认 kinds 里的种类。 */ (ev) => {
      const { kind, payload } = videoItemOf(ev);
      if (!kinds.includes(kind)) return;
      const id = String(payload.id || "");
      if (id) onEvent(kind, id, payload);
    });
  });
  return () => {
    alive = false;
    off?.();
  };
}

/** subscribeStudioEvents 听 studio_state，把短片编号和载荷交给 onStudio。返回退订函数。 */
export function subscribeStudioEvents(onStudio: (id: string, payload: Record<string, unknown>) => void): () => void {
  return subscribeVideoEvents(["studio_state"], /** id 和 payload 原样交给 onStudio。 */ (_kind, id, payload) => onStudio(id, payload));
}
