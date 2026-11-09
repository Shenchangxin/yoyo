/**
 * 拥有：任务页上和 agent 有关的几块：谁在做（字形、名字、状态）、用量细字、换人，以及这张工单挂起的审批条。
 * 不拥有：工单怎么读成会话（TaskPage.tsx），也不判断能不能派。换人到了后端还会再查一遍。
 */

import { useEffect, useState } from "react";
import { ChevronDown } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { approvals, orchestratorAct, resolveApproval, type BuddyView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { errMessage } from "../../lib/normalize";
import type { Approval } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { BuddyMark } from "../buddy/BuddyMark";
import { agentStateText, fill, hasExternal } from "../buddy/buddy-model";
import { useBuddy } from "../buddy/buddy-store";
import { approvalKind, external, reassignTargets, ticketAgent, usageText } from "./agent-model";
import type { Ticket } from "./desk-model";

/** TaskAgentRow 是标题下的一行：谁在做、用量和换人。一个外部 agent 都没装、工单也是 yoyo 做的时候不画。 */
export function TaskAgentRow(props: { ticket: Ticket; onSent?: () => void }) {
  const copy = useCopy();
  const list = useBuddy((s) => s.list);
  const id = ticketAgent(props.ticket);
  if (!external(props.ticket) && !hasExternal(list)) return null;
  const v = list.find((x) => x.id === id);
  const name = v?.name || id;
  const state = v ? agentStateText(v, copy.buddy) : "";
  const usage = usageText(props.ticket.usage, copy.tasks.usageContext);
  return (
    <div className="mt-0.5 flex min-w-0 items-center gap-2 text-ui-caption text-muted" data-testid="task-agent">
      <button
        type="button"
        className="-ml-1 inline-flex min-w-0 items-center gap-1.5 rounded-md px-1 py-0.5 text-foreground/85 hover:bg-lift hover:text-foreground"
        title={v && !v.builtin ? fill(copy.tasks.via, { protocol: "ACP" }) : undefined}
        data-testid="task-agent-chip"
        onClick={() => useUI.getState().openBuddy(id)}
      >
        <BuddyMark v={v ?? { name }} className="size-3.5 text-ui-badge" />
        <span className="truncate">{name}</span>
        {state ? <span className="shrink-0 text-muted">· {state}</span> : null}
      </button>
      {usage ? <span className="truncate tabular-nums" data-testid="task-usage">{usage}</span> : null}
      {props.ticket.stage !== "ended" ? <TaskReassign ticket={props.ticket} current={id} list={list} onSent={props.onSent} /> : null}
    </div>
  );
}

/** TaskReassign 是「换人」：列出现在能派的 agent 和 yoyo。选了以后要再点一次确认，五秒不点就作罢。 */
function TaskReassign(props: { ticket: Ticket; current: string; list: BuddyView[]; onSent?: () => void }) {
  const copy = useCopy();
  const [armed, setArmed] = useState<BuddyView | null>(null);
  const [sending, setSending] = useState(false);
  useEffect(() => {
    if (!armed) return;
    const timer = window.setTimeout(() => setArmed(null), 5000);
    return () => window.clearTimeout(timer);
  }, [armed]);
  const targets = reassignTargets(props.list, props.current);
  async function confirm(v: BuddyView) {
    setSending(true);
    try {
      await orchestratorAct("reassign", props.ticket.id, "", v.id, "");
      toast(fill(copy.tasks.handover, { name: v.name }));
      props.onSent?.();
    } catch (err) {
      toast.error(errMessage(err));
    } finally {
      setSending(false);
      setArmed(null);
    }
  }
  if (armed) {
    return (
      <button
        type="button"
        className="ml-auto h-6 shrink-0 rounded-md bg-warning/10 px-2 font-medium text-warning disabled:opacity-60"
        disabled={sending}
        data-testid="task-reassign-confirm"
        onClick={() => { void confirm(armed); }}
      >
        {fill(copy.tasks.reassignConfirm, { name: armed.name })}
      </button>
    );
  }
  if (!targets.length) return null;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="ml-auto inline-flex h-6 shrink-0 items-center gap-0.5 rounded-md px-1.5 hover:bg-lift hover:text-foreground" data-testid="task-reassign">
          {copy.tasks.reassign}
          <ChevronDown className="size-3" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {targets.map((v) => (
          <DropdownMenuItem key={v.id} onSelect={() => setArmed(v)}>
            <BuddyMark v={v} className="size-4 text-ui-badge" />
            {fill(copy.tasks.reassignTo, { name: v.name })}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/**
 * TaskApprovals 是这张工单挂起的审批：每条写清谁想做什么，附上完整的路径或命令，下面是允许一次、这张任务都允许和拒绝。
 * 工单开着时每两秒看一次。问人的提问不在这里，那些走回答框。
 */
export function TaskApprovals(props: { ticket: Ticket; className?: string }) {
  const copy = useCopy();
  const list = useBuddy((s) => s.list);
  const [rows, setRows] = useState<Approval[]>([]);
  const [busy, setBusy] = useState("");
  const open = props.ticket.stage !== "ended";
  const session = `ticket-${props.ticket.id}`;
  useEffect(() => {
    setRows([]);
    if (!open) return;
    let stop = false;
    const pull = () => {
      approvals()
        .then((all) => {
          if (!stop) setRows(all.filter((a) => a.sessionId === session && a.action !== "ask_user"));
        })
        .catch(() => {});
    };
    pull();
    const timer = window.setInterval(pull, 2000);
    return () => {
      stop = true;
      window.clearInterval(timer);
    };
  }, [session, open]);
  if (!rows.length) return null;
  const id = ticketAgent(props.ticket);
  const name = list.find((v) => v.id === id)?.name || id;
  async function decide(approval: string, decision: "once" | "session" | "deny") {
    setBusy(approval);
    try {
      await resolveApproval(approval, decision);
      setRows((prev) => prev.filter((a) => a.id !== approval));
    } catch (err) {
      toast.error(errMessage(err));
    } finally {
      setBusy("");
    }
  }
  return (
    <div className={cn("mt-2 shrink-0 space-y-2", props.className)} data-testid="task-approvals">
      {rows.map((a) => (
        <div key={a.id} className="rounded-lg border-l-2 border-warning bg-card px-3 py-2.5" data-testid="task-approval">
          <p className="text-ui-footnote font-medium text-foreground/85">{fill(copy.tasks.approvalTitle, { name, action: approvalKind(a.action, copy.tasks.approvalKind) })}</p>
          {a.command || a.path ? <p className="mt-1 break-all font-mono text-ui-caption text-foreground/85">{a.command || a.path}</p> : null}
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Button size="sm" disabled={!!busy} onClick={() => { void decide(a.id, "once"); }}>{copy.tasks.allowOnce}</Button>
            <Button size="sm" variant="outline" disabled={!!busy} onClick={() => { void decide(a.id, "session"); }}>{copy.tasks.allowTask}</Button>
            <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => { void decide(a.id, "deny"); }}>{copy.tasks.deny}</Button>
          </div>
        </div>
      ))}
    </div>
  );
}
