/**
 * 拥有：一条实时通知相对全局序号游标该丢弃、跳过、应用还是回拉。
 * 不拥有：真正去拉环缓冲，或把事件折进转录。
 *
 * 枢纽序号跨会话全局递增，渲染器只为整条流保留一个游标，别的会话的通知也会推动它。
 * 游标只在直到该序号的内容都已应用或有意跳过之后才向前。通知跳过下一号，说明更早的序号还没到，事件也可能乱序到达。唯一的修补是回拉，因为环缓冲按序号顺序读回。
 */

/**
 * LiveStep 一条实时通知相对游标的处置。
 * stale：游标已经覆盖，来自种子、回拉或更早的通知。
 * advance：序号连续，但属于别的会话，只推动游标。
 * apply：序号连续、属于本会话，且通知里带了事件，应用并推动游标。
 * pull：单靠这条通知无法按序应用，回拉环缓冲尾部。
 */
export type LiveStep =
  | "stale"
  | "advance"
  | "apply"
  | "pull";

/** classifyLive 按游标、序号、是否本会话、事件是否内联决定这一步。序号不大于游标为过期；跳过下一号则回拉；非本会话只前进；本会话却没有内联事件则回拉。没有序号的通知按普通事件处理。 */
export function classifyLive(cursor: number, seq: number, own: boolean, inline: boolean): LiveStep {
  if (seq > 0 && seq <= cursor) return "stale";
  if (seq > 0 && seq > cursor + 1) return "pull";
  if (!own) return "advance";
  if (!inline) return "pull";
  return "apply";
}
