/**
 * 本文件拥有从 yoyo:item 里取出视频路由和剪辑状态。不拥有转录条目。
 * 它只读事件，不改服务端。
 */
import { unwrapEvent } from "../../lib/stream-fold";

/** VideoItem 是一条视频系统事件的种类和载荷。 */
export type VideoItem = { kind: string; payload: Record<string, unknown> };

/** videoItemOf 取出事件种类和载荷。同进程桌面送的是 NoticeOf，种类和载荷在 event 里；隔离进程把 item_kind 和 payload 放在顶层。raw 先剥掉 Wails 的 { name, data } 信封。 */
export function videoItemOf(raw: unknown): VideoItem {
  const e = unwrapEvent(raw);
  if (!e || typeof e !== "object") return { kind: "", payload: {} };
  const nested = (e.event ?? e.Event) as Record<string, any> | undefined;
  const kind = String(e.item_kind ?? e.ItemKind ?? nested?.item_kind ?? nested?.ItemKind ?? "");
  const body = e.payload ?? e.Payload ?? nested?.payload ?? nested?.Payload;
  return { kind, payload: body && typeof body === "object" ? body as Record<string, unknown> : {} };
}
