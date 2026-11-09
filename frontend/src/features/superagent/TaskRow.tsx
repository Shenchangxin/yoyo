/**
 * 拥有：任务列表里的一行工单，以及列表上「几分钟前」的说法。外部 agent 做的、或任务组里的工单，这一行写明是谁在做。
 * 不拥有：分区和排序（work-model.ts），也不拥有任务组那一行（GroupRow.tsx）。
 */

import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { BuddyMark } from "../buddy/BuddyMark";
import { useBuddy } from "../buddy/buddy-store";
import { external, ticketAgent } from "./agent-model";
import type { Ticket } from "./desk-model";
import { taskStatus } from "./TaskPage";
import { taskSection, ticketLine } from "./work-model";

/**
 * TaskRow 说清这张工单在等谁、需要什么，以及最近一句。等你的那一张用 warning 标出来。
 * 交给外部 agent 的、或任务组里的工单，状态前面带上 agent 的字形和名字，任务组成员再带上分工。
 * hint 是任务组里的依赖提示，挂着等前置的成员用它代替状态。
 */
export function TaskRow(props: { ticket: Ticket; onClick: () => void; hint?: string }) {
  const copy = useCopy();
  const list = useBuddy((s) => s.list);
  const ticket = props.ticket;
  const section = taskSection(ticket);
  const status = ticket.held && props.hint ? "" : taskStatus(ticket, copy.work);
  const when = ticket.updatedAt ? agoLabel(ticket.updatedAt, copy.work) : "";
  const line = ticketLine(ticket);
  const failed = ticket.outcome === "failed" || ticket.outcome === "dropped" || (section === "person" && (ticket.action === "failed" || ticket.action === "blocked"));
  const agent = ticketAgent(ticket);
  const v = list.find((row) => row.id === agent);
  const name = v?.name || agent;
  const who = external(ticket) || ticket.group ? [ticket.group ? ticket.role : "", name].filter(Boolean).join(" · ") : "";
  const meta = [status, when].filter(Boolean).join(" · ");
  return (
    <button
      type="button"
      data-testid="task-row"
      className="flex w-full items-start gap-3 rounded-[20px] bg-card px-4 py-3 text-left shadow-[var(--shadow-card)]"
      onClick={props.onClick}
    >
      {section === "working" && !ticket.held ? (
        <span className="pulse-dot is-accent mt-[7px]" aria-hidden />
      ) : (
        <span className={cn("mt-[7px] size-1.5 shrink-0 rounded-full", failed ? "bg-danger" : section === "person" ? "bg-warning" : "bg-muted/50")} aria-hidden />
      )}
      <span className="min-w-0 flex-1">
        <span className="block truncate text-ui-body-l font-medium">{ticket.title || ticket.id}</span>
        <span className={cn("mt-0.5 flex min-w-0 items-center gap-1 text-ui-footnote leading-5", section === "person" ? "text-warning" : "text-muted")}>
          {who ? (
            <span className="inline-flex min-w-0 shrink items-center gap-1 text-foreground/80">
              <BuddyMark v={v ?? { name }} className="size-3.5 text-ui-badge" />
              <span className="truncate" data-testid="task-row-agent">{who}</span>
            </span>
          ) : null}
          {who && meta ? <span aria-hidden>·</span> : null}
          {meta ? <span className="truncate">{meta}</span> : null}
        </span>
        {props.hint ? <span className="mt-0.5 block truncate text-ui-footnote leading-5 text-warning" data-testid="task-hint">{props.hint}</span> : null}
        {line ? <span className="mt-0.5 block truncate text-ui-footnote leading-5 text-muted">{line}</span> : null}
      </span>
    </button>
  );
}

/** agoLabel 把一个时刻写成多久以前。工单、任务组和记忆用同一套说法。 */
export function agoLabel(iso: string, work: ReturnType<typeof useCopy>["work"]): string {
  const at = Date.parse(iso);
  if (Number.isNaN(at)) return work.memoryNotYet;
  const mins = Math.max(0, Math.round((Date.now() - at) / 60000));
  if (mins < 1) return work.memoryJustNow;
  if (mins < 60) return work.memoryMinutes.replace("{n}", String(mins));
  const hours = Math.round(mins / 60);
  if (hours < 24) return work.memoryHoursAgo.replace("{n}", String(hours));
  const days = Math.round(hours / 24);
  if (days < 14) return work.memoryDaysAgo.replace("{n}", String(days));
  return work.memoryWeeksAgo.replace("{n}", String(Math.round(days / 7)));
}
