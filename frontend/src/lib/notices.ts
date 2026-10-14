import { classifyItem, shortError, type ErrorCopy } from "./error";
import type { Item, Notice } from "./protocol";

/** One line for a notice row. Long replies are clipped, never wrapped into a wall of text. */
export function noticeSummary(text: string, max = 140): string {
  const s = (text || "").replace(/\s+/g, " ").trim();
  return s.length > max ? `${s.slice(0, max - 1)}…` : s;
}

/** How long a bell notice stays before it leaves on its own. Dismissing or opening it removes it sooner. */
export const NOTICE_TTL_MS = 24 * 60 * 60 * 1000;

/**
 * A notice ages from when its event happened, not when it reached the screen, so a replayed
 * old event arrives already expired. A notice with no readable time is kept until dismissed.
 */
export function noticeExpired(n: Pick<Notice, "ts">, now = Date.now(), ttl = NOTICE_TTL_MS): boolean {
  const at = Date.parse(n.ts);
  return Number.isFinite(at) && now - at >= ttl;
}

/**
 * What the bell keeps: finished turns and errors the user should come back to.
 * - Approvals are not notices. The approvals section shows them with their actions,
 *   and keeping a copy here made a second row that outlived the decision.
 * - A deliberate Stop lands as a "canceled" card, and a turn that ended with plan
 *   steps open has its own plan card. A "finished" row on top would contradict them.
 * - A session the user is watching in a focused window already shows the result.
 */
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
