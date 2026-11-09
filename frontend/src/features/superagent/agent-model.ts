/**
 * 拥有：工单上和 agent 有关的判断：谁在做、要不要先去登录、用量怎么写、换人能换给谁。
 * 不拥有：取数和画法。文案模板由调用方传进来，这里不读当前语言。
 */

import type { BuddyView } from "../../lib/client.ts";
import { fill, routable, sortBuddies } from "../buddy/buddy-model.ts";
import type { Ticket, Usage } from "./desk-model.ts";

/** YOYO 是 yoyo 自带工人的编号。 */
export const YOYO = "yoyo";

/** ticketAgent 是做这张工单的 agent。没写时是 yoyo。 */
export function ticketAgent(t: Pick<Ticket, "agent">): string {
  return t.agent || YOYO;
}

/** external 在工单交给外部 agent 时为真。 */
export function external(t: Pick<Ticket, "agent">): boolean {
  return ticketAgent(t) !== YOYO;
}

/** ticketNeed 是最近一次派工最后一条报告要人先做的事，和 Go 的 TicketNeed 一致。login 表示先去 Buddy 登录。 */
export function ticketNeed(t: Pick<Ticket, "assignments">): string {
  const steps = t.assignments ?? [];
  const lines = steps[steps.length - 1]?.lines ?? [];
  return lines[lines.length - 1]?.need || "";
}

/** compactTokens 把 token 数写短：999、53k、1.2M。 */
export function compactTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`;
  if (n >= 1_000) return `${Math.round(n / 1_000)}k`;
  return String(Math.max(0, Math.round(n)));
}

/** money 写出花费。美元用 $ 开头，其余币种写在后面；不到一分时多留两位。 */
export function money(cost: number, currency = ""): string {
  const cur = (currency || "USD").toUpperCase();
  const amount = cost < 0.01 ? cost.toFixed(4) : cost.toFixed(2);
  return cur === "USD" ? `$${amount}` : `${amount} ${cur}`;
}

/** usageText 是任务页的用量细字，例如「上下文 53k/200k · $0.05」。context 是带 {used} 和 {size} 的模板。没报过用量时为空。 */
export function usageText(u: Usage | undefined, context: string): string {
  if (!u) return "";
  const parts: string[] = [];
  if (u.size) parts.push(fill(context, { used: compactTokens(u.used ?? 0), size: compactTokens(u.size) }));
  if (u.cost) parts.push(money(u.cost, u.currency));
  return parts.join(" · ");
}

/** approvalKind 把审批的动作名换成文案里的说法。外部 agent 的动作名是 buddy.<种类>；不认识的用 other。 */
export function approvalKind(action: string, kinds: Record<string, string> & { other: string }): string {
  const kind = action.replace(/^buddy\./, "");
  return Object.prototype.hasOwnProperty.call(kinds, kind) ? kinds[kind] : kinds.other;
}

/** reassignTargets 是「换人」能换给的 agent：现在能派的，yoyo 在前，去掉正在做的这个。 */
export function reassignTargets(list: BuddyView[], current: string): BuddyView[] {
  return sortBuddies(list).filter((v) => routable(v) && v.id !== current);
}
