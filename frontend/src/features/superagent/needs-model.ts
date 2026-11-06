/**
 * 拥有：从编排快照和待审队列里算出「需要你」的条目。纯函数，不拉数据。
 * 不拥有：怎么处理一条，也不拥有快照放在哪。那在 needs.ts。
 */

import type { DeskView, Ticket, Trigger } from "./desk-model.ts";
import { dueDelta, dueItems, ticketLine, visibleIncoming } from "./work-model.ts";

/** NeedKind 是一条要人决定的事从哪来。proposal 是等你点头才发出去的提案，personal 是在等你回答的个人事务。 */
export type NeedKind = "approval" | "question" | "review" | "failed" | "incoming" | "due" | "draft" | "inbox" | "proposal" | "personal";

export type NeedItem = {
  id: string;
  kind: NeedKind;
  title: string;
  detail: string;
  /** source 是来源的短名：github、feishu、calendar、desk、mail、personal 或 session。 */
  source: string;
  at: string;
  ticketId?: string;
  triggerId?: string;
  offerId?: string;
  inboxId?: string;
  sessionId?: string;
  due?: string;
  proposalId?: string;
  hash?: string;
  draftId?: string;
  personalId?: string;
  /** status 是个人事务在等什么：waiting_input 等回答，waiting_approval 等批准。 */
  status?: string;
};

/** ReviewQueue 是待审队列里和人有关的几块：工具审批、收件、待发送的草稿、待发送的提案、在等你的个人事务。 */
export type ReviewQueue = { offers: unknown[]; inbox: unknown[]; drafts: unknown[]; proposals: unknown[]; waiting: unknown[] };

export const EMPTY_QUEUE: ReviewQueue = { offers: [], inbox: [], drafts: [], proposals: [], waiting: [] };

/** normalizeQueue 只留下这几块数组。 */
export function normalizeQueue(raw: unknown): ReviewQueue {
  const q = (raw ?? {}) as Record<string, unknown>;
  const list = (value: unknown) => (Array.isArray(value) ? value : []);
  return {
    offers: list(q.offers ?? q.Offers),
    inbox: list(q.inbox ?? q.Inbox),
    drafts: list(q.drafts ?? q.Drafts),
    proposals: list(q.proposals ?? q.Proposals),
    waiting: list(q.waiting ?? q.Waiting),
  };
}

function text(value: unknown): string {
  return typeof value === "string" ? value : value == null ? "" : String(value);
}

function field(row: unknown, ...names: string[]): unknown {
  if (!row || typeof row !== "object") return undefined;
  const rec = row as Record<string, unknown>;
  for (const name of names) {
    if (rec[name] !== undefined && rec[name] !== null) return rec[name];
  }
  return undefined;
}

/** ask_user 的问题在输入框上方那张卡里回答，不算工具审批。 */
function approvalOffer(offer: unknown): boolean {
  const req = field(offer, "request", "Request");
  return text(field(req, "action", "Action")) !== "ask_user";
}

function ticketKind(ticket: Ticket): NeedKind {
  if (ticket.action === "review") return "review";
  if (ticket.action === "blocked" || ticket.action === "failed" || ticket.action === "delivery") return "failed";
  return "question";
}

/** 没有时间的排在最后。 */
function waited(at: string): number {
  const t = Date.parse(at);
  return Number.isFinite(t) ? t : Number.MAX_SAFE_INTEGER;
}

/** todayDue 是过期和今天到期的日程。以后几天的归到「接下来」。 */
export function todayDue(rows: Trigger[], now = new Date()): Trigger[] {
  return dueItems(rows, now).filter((row) => (dueDelta(row.due || "", now) ?? 1) <= 0);
}

/** laterDue 是以后两周内到期的日程，近的在前。 */
export function laterDue(rows: Trigger[], now = new Date()): Trigger[] {
  return dueItems(rows, now)
    .filter((row) => (dueDelta(row.due || "", now) ?? 0) > 0)
    .sort((a, b) => (dueDelta(a.due || "", now) ?? 0) - (dueDelta(b.due || "", now) ?? 0));
}

/**
 * needItems 把要人决定的事排成一列。过期和今天到期的日程在最前，其余按等了多久排，等得久的在前。
 * 只看得出名字、看不出要做什么的条目不进来。
 */
export function needItems(view: DeskView | null, queue: ReviewQueue, now = new Date()): NeedItem[] {
  const due: NeedItem[] = [];
  const rest: NeedItem[] = [];
  for (const offer of queue.offers) {
    if (!approvalOffer(offer)) continue;
    const id = text(field(offer, "id", "ID"));
    if (!id) continue;
    const req = field(offer, "request", "Request");
    rest.push({
      id: `offer:${id}`,
      kind: "approval",
      title: text(field(req, "action", "Action")) || "approval",
      detail: text(field(req, "command", "Command", "path", "Path")),
      source: "session",
      at: text(field(offer, "at", "At", "created_at", "CreatedAt")),
      offerId: id,
      sessionId: text(field(req, "session_id", "SessionID")),
    });
  }
  for (const ticket of view?.tickets ?? []) {
    if (ticket.stage === "ended" || !ticket.person) continue;
    rest.push({
      id: `ticket:${ticket.id}`,
      kind: ticketKind(ticket),
      title: ticket.title || ticket.id,
      detail: ticketLine(ticket),
      source: ticket.origin || "task",
      at: ticket.updatedAt || "",
      ticketId: ticket.id,
    });
  }
  for (const row of visibleIncoming(view?.queued ?? [])) {
    rest.push({
      id: `trigger:${row.id}`,
      kind: "incoming",
      title: row.text || row.id,
      detail: row.sender || "",
      source: row.source,
      at: row.at || "",
      triggerId: row.id,
    });
  }
  for (const row of todayDue(view?.queued ?? [], now)) {
    due.push({
      id: `due:${row.id}`,
      kind: "due",
      title: row.text || row.id,
      detail: "",
      source: "calendar",
      at: row.at || "",
      triggerId: row.id,
      due: row.due,
    });
  }
  for (const draft of queue.drafts) {
    if (field(draft, "sent", "Sent")) continue;
    const to = text(field(draft, "to", "To"));
    const subject = text(field(draft, "subject", "Subject"));
    if (!to && !subject) continue;
    const draftId = text(field(draft, "id", "ID"));
    rest.push({
      id: `draft:${draftId || `${to}:${subject}`}`,
      kind: "draft",
      title: to || subject,
      detail: to ? subject : "",
      source: "mail",
      at: text(field(draft, "at", "At", "created_at")),
      draftId,
    });
  }
  for (const item of queue.inbox) {
    if (!field(item, "unread", "Unread")) continue;
    const id = text(field(item, "id", "ID"));
    const title = text(field(item, "title", "Title"));
    if (!id || !title) continue;
    rest.push({
      id: `inbox:${id}`,
      kind: "inbox",
      title,
      detail: text(field(item, "body", "Body")).slice(0, 200),
      source: "session",
      at: text(field(item, "at", "At", "created_at", "CreatedAt")),
      inboxId: id,
      sessionId: text(field(item, "session_id", "SessionID")),
    });
  }
  for (const proposal of queue.proposals) {
    const id = text(field(proposal, "id", "ID"));
    const title = text(field(proposal, "title", "Title"));
    if (!id || !title) continue;
    rest.push({
      id: `proposal:${id}`,
      kind: "proposal",
      title,
      detail: text(field(proposal, "account", "Account")),
      source: "personal",
      at: text(field(proposal, "created_at", "CreatedAt")),
      proposalId: id,
      hash: text(field(proposal, "hash", "Hash")),
    });
  }
  for (const task of queue.waiting) {
    const id = text(field(task, "id", "ID"));
    const title = text(field(task, "title", "Title"));
    if (!id || !title) continue;
    rest.push({
      id: `personal:${id}`,
      kind: "personal",
      title,
      detail: text(field(task, "question", "Question")),
      source: "personal",
      at: text(field(task, "updated_at", "UpdatedAt", "created_at")),
      personalId: id,
      status: text(field(task, "status", "Status")),
      sessionId: text(field(task, "session_id", "SessionID")),
    });
  }
  due.sort((a, b) => (dueDelta(a.due || "", now) ?? 0) - (dueDelta(b.due || "", now) ?? 0));
  rest.sort((a, b) => waited(a.at) - waited(b.at));
  return [...due, ...rest];
}

/** runningTickets 是正在做的工单数，给状态那一句用。排队的不算。 */
export function runningTickets(view: DeskView | null): number {
  return (view?.tickets ?? []).filter((ticket) => ticket.stage === "working" && ticket.action !== "queued").length;
}
