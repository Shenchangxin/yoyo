/**
 * 拥有：任务列表里任务组的那一行：叠放的成员字形、标题、分工或对比、进度、组状态和更新时间；展开后是成员行和依赖提示。
 * 不拥有：任务组排在哪个分区（work-model 的 itemSections），也不拥有任务组页（GroupPage.tsx）。
 */

import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { StatusDot } from "../../components/ui/status-dot";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { BuddyMark } from "../buddy/BuddyMark";
import { fill } from "../buddy/buddy-model";
import { useBuddy } from "../buddy/buddy-store";
import { ticketAgent } from "./agent-model";
import type { Group, Ticket } from "./desk-model";
import { agoLabel, TaskRow } from "./TaskRow";
import { groupActive, groupProgress, groupTone, topoOrder, waitingOn } from "./work-model";

/** 任务组那一行最多叠放几个成员字形，再多的写成 +N。 */
const MARKS_SHOWN = 4;

/** GroupRow 是任务组一行。还在做或等人决定时默认展开成员，结束后默认收起。点标题打开任务组页，点箭头展开或收起。 */
export function GroupRow(props: { group: Group; members: Ticket[]; onOpen: (id: string) => void; onSelect: (id: string) => void }) {
  const copy = useCopy();
  const t = copy.tasks;
  const list = useBuddy((s) => s.list);
  const [open, setOpen] = useState(() => groupActive(props.group.state));
  const group = props.group;
  const progress = groupProgress(props.members);
  const marks = props.members.slice(0, MARKS_SHOWN);
  const more = props.members.length - marks.length;
  const when = group.updatedAt ? agoLabel(group.updatedAt, copy.work) : "";
  const mode = group.mode === "compare" ? t.groupCompare : t.groupSplit;
  return (
    <div data-testid="group-row" data-group={group.id}>
      <div className="flex w-full items-center gap-1 rounded-[20px] bg-card py-1.5 pl-1.5 pr-4 shadow-[var(--shadow-card)]">
        <button
          type="button"
          aria-expanded={open}
          aria-label={open ? t.hideMembers : t.showMembers}
          data-testid="group-toggle"
          className="grid size-8 shrink-0 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground"
          onClick={() => setOpen((on) => !on)}
        >
          <ChevronDown className={cn("size-3.5 transition-transform duration-200", !open && "-rotate-90")} />
        </button>
        <button type="button" className="flex min-w-0 flex-1 items-center gap-2.5 py-1.5 text-left" data-testid="group-open" onClick={() => props.onOpen(group.id)}>
          <span className="flex shrink-0 items-center -space-x-1.5" aria-hidden>
            {marks.map((member) => {
              const id = ticketAgent(member);
              const v = list.find((row) => row.id === id);
              return <BuddyMark key={member.id} v={v ?? { name: id }} className="bg-card ring-2 ring-card" />;
            })}
            {more > 0 ? <span className="grid size-5 shrink-0 place-items-center rounded-full bg-lift text-ui-badge text-muted ring-2 ring-card">+{more}</span> : null}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-ui-body-l font-medium">{group.title || group.id}</span>
            <span className="mt-0.5 flex min-w-0 items-center gap-1.5 text-ui-footnote leading-5 text-muted">
              <span className="shrink-0" data-testid="group-progress">{mode} · {progress.done}/{progress.total}</span>
              <StatusDot tone={groupTone(group.state)} />
              <span className="sr-only">{groupStateText(group.state, t.groupState)}</span>
              {when ? <span className="truncate">{when}</span> : null}
            </span>
          </span>
        </button>
      </div>
      {open ? (
        <ul className="ml-5 mt-2 space-y-2 border-l border-border pl-3" data-testid="group-members">
          {topoOrder(props.members).map(({ ticket }) => (
            <li key={ticket.id}>
              <TaskRow ticket={ticket} hint={memberHint(ticket, props.members, t)} onClick={() => props.onSelect(ticket.id)} />
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

/** memberHint 是成员的依赖提示：在等前置时写「等 后端」，前置没做成时写「前置没完成」，不挂着时为空。 */
export function memberHint(ticket: Ticket, members: Ticket[], t: { waitingOn: string; depsFailed: string }): string {
  if (ticket.held === "deps_failed") return t.depsFailed;
  if (ticket.held !== "deps") return "";
  const names = waitingOn(ticket, members);
  if (names.length === 0) return "";
  return fill(t.waitingOn, { role: names.join(/[\u4e00-\u9fff]/.test(t.waitingOn) ? "、" : ", ") });
}

/** groupStateText 把任务组状态换成文案。认不出的按在做说。 */
export function groupStateText(state: string, words: { running: string; attention: string; done: string; failed: string; stopped: string }): string {
  switch (state) {
    case "attention":
    case "done":
    case "failed":
    case "stopped":
      return words[state];
    default:
      return words.running;
  }
}
