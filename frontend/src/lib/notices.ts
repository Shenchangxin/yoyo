/**
 * 拥有：通知的一行摘要、一天后自行过期，以及从转录条目决定铃铛要不要留一条。
 * 不拥有：通知列表的状态。那是 store 的事。
 */

import { classifyItem, shortError, type ErrorCopy } from "./error";
import type { Item, Notice } from "./protocol";

/** noticeSummary 收成一行。连续空白折成一个空格，超过 max 则截断并加省略号。长回复不折成一堵文字。 */
export function noticeSummary(text: string, max = 140): string {
  const s = (text || "").replace(/\s+/g, " ").trim();
  return s.length > max ? `${s.slice(0, max - 1)}…` : s;
}

/** NOTICE_TTL_MS 铃铛通知自行离开前的停留时间。关掉或打开会更早移除。 */
export const NOTICE_TTL_MS = 24 * 60 * 60 * 1000;

/** noticeExpired 从事件发生时起算，不是从它到达屏幕时起算，因此重放的旧事件到达时已经过期。读不出时间的通知一直留到被关掉。 */
export function noticeExpired(n: Pick<Notice, "ts">, now = Date.now(), ttl = NOTICE_TTL_MS): boolean {
  const at = Date.parse(n.ts);
  return Number.isFinite(at) && now - at >= ttl;
}

/** noticeFromItem 铃铛只留该回来看的已完成回合和错误。批准不是通知，批准区会带着操作显示它们，这里再留一份会多出一行，并且比决定活得更久。主动停止会落成 canceled 卡片，回合停在计划步骤未完成时有自己的计划卡片，上面再加一行已完成会互相矛盾。正在聚焦窗口里看的会话已经显示了结果。 */
export function noticeFromItem(
  item: Item,
  copy: { app: { turnFinished: string; errorNotice: string }; transcript: ErrorCopy },
  viewing = false,
): Notice | null {
  if (item.type !== "turn_end" && item.type !== "error") return null;
  if (viewing) return null;
  const id = item.key || `${item.type}:${item.sessionId}:${item.ts}`;
  if (item.type === "error") {
    if (classifyItem(item).kind === "canceled") return null;
    return { id, title: copy.app.errorNotice, body: shortError(item, copy.transcript), ts: item.ts, sessionId: item.sessionId };
  }
  if (item.payload?.stop === "plan_open") return null;
  return { id, title: copy.app.turnFinished, body: noticeSummary(item.text), ts: item.ts, sessionId: item.sessionId };
}
