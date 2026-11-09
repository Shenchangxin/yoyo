/**
 * 拥有：主会话里任务组的那张卡片。头部是任务组 · 分工或对比、标题、进度和「打开」，下面是进度细线和按依赖排的成员行；
 * 有成员在问人时底部写出问题和「去回答」；任务组结束后收成一行，对比写上路由推荐或已采用的那一份。
 * 不拥有：卡片排在转录哪里（transcript-layout 的 isGroupCard 和 layoutRows），也不拥有任务组页（superagent/GroupPage.tsx）。
 * 卡片在转录里只写一次作为锚点，成员和状态实时读 useNeeds 里的编排快照；快照里没有这个任务组时只列锚点里记下的成员。
 */

import { useEffect } from "react";
import { Users } from "lucide-react";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { StatusDot } from "../../components/ui/status-dot";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { BuddyMark } from "../buddy/BuddyMark";
import { fill } from "../buddy/buddy-model";
import { useBuddy } from "../buddy/buddy-store";
import { ticketAgent } from "../superagent/agent-model";
import type { Group, Ticket } from "../superagent/desk-model";
import { memberHint } from "../superagent/GroupRow";
import { refreshNeeds, showGroup, showTask, useNeeds } from "../superagent/needs";
import {
  askingMember,
  durationLabel,
  groupActive,
  groupProgress,
  groupTone,
  memberDone,
  memberStatus,
  memberTone,
  ticketLine,
  topoOrder,
  type MemberStatus,
} from "../superagent/work-model";

/** 今天栏不在场时，还在做的任务组的卡片多久拉一次编排快照。今天栏在场时它自己两秒拉一次。 */
const POLL_MS = 3000;

const LINK = "shrink-0 text-ui-caption text-foreground/85 underline decoration-dotted underline-offset-[3px] hover:text-foreground";

/** AnchorMember 是锚点里记下的一个成员。 */
type AnchorMember = { ticket: string; agent: string; agentName: string; role: string; title: string; dependsOn: string[] };

/** anchorMembers 读出锚点里记下的成员。没有编号的项跳过。 */
function anchorMembers(raw: unknown): AnchorMember[] {
  if (!Array.isArray(raw)) return [];
  const out: AnchorMember[] = [];
  for (const row of raw) {
    if (!row || typeof row !== "object") continue;
    const m = row as Record<string, unknown>;
    const ticket = typeof m.ticket === "string" ? m.ticket : "";
    if (!ticket) continue;
    out.push({
      ticket,
      agent: typeof m.agent === "string" ? m.agent : "",
      agentName: typeof m.agentName === "string" ? m.agentName : "",
      role: typeof m.role === "string" ? m.role : "",
      title: typeof m.title === "string" ? m.title : "",
      dependsOn: Array.isArray(m.dependsOn) ? m.dependsOn.filter((id): id is string => typeof id === "string") : [],
    });
  }
  return out;
}

/** GroupCard 是任务组卡片。item 是 task_split 时写下的锚点。 */
export function GroupCard({ item }: { item: Item }) {
  const copy = useCopy();
  const t = copy.transcript;
  const list = useBuddy((s) => s.list);
  const view = useNeeds((s) => s.view);
  const mounted = useNeeds((s) => s.paneMounted);
  const gid = String(item.payload?.group || "");
  const anchors = anchorMembers(item.payload?.members);
  const live = view?.groups?.find((row) => row.id === gid) ?? null;
  const active = !!live && groupActive(live.state);
  useEffect(() => {
    if (!active || mounted) return;
    const timer = window.setInterval(() => { void refreshNeeds(); }, POLL_MS);
    return () => window.clearInterval(timer);
  }, [active, mounted]);
  const byId = new Map((view?.tickets ?? []).map((row) => [row.id, row]));
  const members: Ticket[] = live
    ? live.tickets.flatMap((id) => byId.get(id) ?? [])
    : anchors.map((m) => ({ id: m.ticket, stage: "", action: "", person: false, origin: "", title: m.title, agent: m.agent, role: m.role, dependsOn: m.dependsOn }));
  const mode = (live?.mode || String(item.payload?.mode || "")) === "compare" ? copy.tasks.groupCompare : copy.tasks.groupSplit;
  const title = live?.title || String(item.payload?.title || "") || gid;
  const nameOf = (ticket: Ticket) => {
    const id = ticketAgent(ticket);
    return list.find((row) => row.id === id)?.name || anchors.find((m) => m.ticket === ticket.id)?.agentName || id;
  };
  const markOf = (ticket: Ticket) => {
    const id = ticketAgent(ticket);
    return list.find((row) => row.id === id) ?? { name: nameOf(ticket) };
  };
  const who = (ticket: Ticket) => [nameOf(ticket), ticket.role || ""].filter(Boolean).join(" · ");

  if (live && !groupActive(live.state)) {
    return (
      <button
        type="button"
        className="flex w-full max-w-[40rem] min-w-0 items-center gap-2 rounded-lg border border-border bg-card px-3.5 py-2.5 text-left text-ui-footnote shadow-[var(--shadow-card)] hover:bg-lift/40"
        data-testid="group-card"
        data-group={gid}
        data-state={live.state}
        data-collapsed=""
        onClick={() => showGroup(gid)}
      >
        <Users className="size-3.5 shrink-0 text-muted" aria-hidden />
        <StatusDot tone={groupTone(live.state)} still />
        <span className="shrink-0 font-medium" data-testid="group-card-summary">{endedLine(live, members, (id) => {
          const member = members.find((row) => row.id === id);
          return member ? nameOf(member) : id;
        }, t)}</span>
        <span className="min-w-0 flex-1 truncate text-muted">{title}</span>
        <span className={LINK}>{t.groupView}</span>
      </button>
    );
  }

  const { done, total } = groupProgress(members);
  const asking = live ? askingMember(members) : undefined;
  const now = Date.now();
  return (
    <div
      className="w-full max-w-[40rem] rounded-lg border border-border bg-card px-3.5 py-3 shadow-[var(--shadow-card)]"
      data-testid="group-card"
      data-group={gid}
      data-state={live?.state || ""}
    >
      <div className="flex min-w-0 items-center gap-2">
        <Users className="size-3.5 shrink-0 text-muted" aria-hidden />
        <span className="shrink-0 text-ui-caption text-muted">{fill(t.groupHead, { mode })}</span>
        <span className="min-w-0 flex-1 truncate text-ui-body font-medium" data-testid="group-card-title">{title}</span>
        {live ? <span className="shrink-0 text-ui-caption tabular-nums text-muted" data-testid="group-card-progress">{done}/{total}</span> : null}
        <button type="button" className={LINK} data-testid="group-card-open" onClick={() => showGroup(gid)}>{t.groupOpen}</button>
      </div>
      {live ? <ProgressHairline className="mt-2.5" value={total ? (done / total) * 100 : 0} indeterminate={done === 0} /> : null}
      <ul className="mt-2.5 space-y-1.5" data-testid="group-card-members">
        {topoOrder(members).map(({ ticket }) => (
          <li key={ticket.id} className="flex min-w-0 items-center gap-2 text-ui-footnote" data-testid="group-card-member" data-ticket={ticket.id}>
            <BuddyMark v={markOf(ticket)} className="size-4 text-ui-badge" />
            <span className="min-w-0 shrink truncate text-foreground/85">{who(ticket)}</span>
            {live ? (
              <>
                <StatusDot tone={memberTone(ticket)} />
                <span className="min-w-0 flex-1 truncate text-muted" data-testid="group-card-status">
                  {statusText(memberStatus(ticket, members, now), ticket, members, copy)}
                </span>
              </>
            ) : null}
          </li>
        ))}
      </ul>
      {asking ? (
        <div className="mt-2.5 flex min-w-0 items-center gap-2 border-t border-border/70 pt-2.5 text-ui-footnote" data-testid="group-card-ask">
          <StatusDot tone="needs" still />
          <span className="min-w-0 flex-1 truncate">{fill(t.groupAskLine, { name: who(asking), question: ticketLine(asking) })}</span>
          <button type="button" className={LINK} data-testid="group-card-answer" onClick={() => showTask(asking.id)}>{copy.tasks.goAnswer}</button>
        </div>
      ) : live?.state === "attention" ? (
        <div className="mt-2.5 flex min-w-0 items-center gap-2 border-t border-border/70 pt-2.5 text-ui-footnote" data-testid="group-card-decide">
          <StatusDot tone="needs" still />
          <span className="min-w-0 flex-1 truncate font-medium">{t.groupAttention}</span>
          <button type="button" className={LINK} onClick={() => showGroup(gid)}>{t.groupOpen}</button>
        </div>
      ) : null}
    </div>
  );
}

/** statusText 把成员状态写成卡片上的短句。等前置的和前置没完成的，说法和任务列表一致。 */
function statusText(status: MemberStatus, ticket: Ticket, members: Ticket[], copy: ReturnType<typeof useCopy>): string {
  const t = copy.transcript;
  switch (status.kind) {
    case "asks":
      return t.memberAsks;
    case "needs":
      return t.needsYou;
    case "working":
      return status.minutes == null ? t.memberWorkingNow : fill(t.memberWorking, { time: durationLabel(status.minutes, copy.tasks) });
    case "waiting":
    case "depsFailed":
      return memberHint(ticket, members, copy.tasks) || t.memberQueued;
    case "queued":
      return t.memberQueued;
    case "done":
      return t.memberDone;
    case "stopped":
      return t.memberStopped;
    case "failed":
      return status.reason ? fill(t.memberFailed, { reason: status.reason }) : t.memberFailedBare;
  }
}

/** endedLine 是结束后收起的那一句：完成或停下带上进度，有没成的写几份没成；对比再写上采用了谁，没采用时写路由推荐谁。 */
function endedLine(group: Group, members: Ticket[], nameOf: (ticket: string) => string, t: ReturnType<typeof useCopy>["transcript"]): string {
  const { done, total } = groupProgress(members);
  if (group.state === "failed") {
    const n = members.filter((m) => m.stage === "ended" && !memberDone(m) && m.outcome !== "stopped").length || total - done;
    const [one, many] = t.groupFailed;
    return n === 1 ? one : fill(many, { n: String(n) });
  }
  const bits = [group.state === "stopped" ? t.groupStopped : t.groupDone, `${done}/${total}`];
  if (group.mode === "compare" && group.winner) bits.push(fill(t.groupPicked, { name: nameOf(group.winner) }));
  else if (group.mode === "compare" && group.recommended) bits.push(fill(t.groupRecommend, { name: nameOf(group.recommended) }));
  return bits.join(" · ");
}
