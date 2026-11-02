/**
 * 拥有：右侧这一页怎么把一张工单的输入、报告和跟随读出来。
 * 不拥有：工单怎么执行。停止和重试交给编排器，返回后对话还在原来的会话里。
 */

import { useState } from "react";
import { ChevronLeft } from "lucide-react";
import { orchestratorAct } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import type { Assignment, Ticket } from "./desk-model";
import { manuscriptNames, stageKey } from "./work-model";

/** TaskRead 把工单读在侧栏里。主会话继续留在原来的位置，不跟着换页。 */
export function TaskRead(props: {
  ticket: Ticket | null;
  onBack: () => void;
  onLook: () => void;
  onContinue?: (text: string) => void;
}) {
  const copy = useCopy();
  return (
    <div data-testid="task-read" className="px-1 pb-6 pt-2">
      <div className="flex items-center gap-1">
        <button type="button" className="grid size-8 shrink-0 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground" onClick={props.onBack} aria-label={copy.work.taskBack}>
          <ChevronLeft className="size-4" />
        </button>
        <h1 className="min-w-0 truncate text-[16px] font-semibold">{props.ticket?.title || copy.work.taskEmpty}</h1>
      </div>
      {!props.ticket ? (
        <article className="px-2 pt-8">
          <p className="text-[14px] text-muted">{copy.work.noSubtasks}</p>
          <p className="mt-2 text-[13px] leading-6 text-muted">{copy.work.taskEmptyHint}</p>
        </article>
      ) : (
        <TaskDocument ticket={props.ticket} onLook={props.onLook} onContinue={props.onContinue} compact />
      )}
    </div>
  );
}

/** TaskPage 是任务被点开后的阅读面。没有交互时也把工单本身摆出来。 */
export function TaskPage(props: {
  ticket: Ticket | null;
  onBack: () => void;
  onLook: () => void;
}) {
  const copy = useCopy();
  const ticket = props.ticket;
  return (
    <div data-testid="task-page" className="flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-background text-foreground">
      <div className="flex shrink-0 items-center px-6 py-3">
        <button type="button" className="h-8 rounded-full bg-card px-3 text-[12px] text-muted shadow-[var(--shadow-card)] hover:text-foreground" onClick={props.onBack}>
          {copy.work.taskBack}
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
        {!ticket ? (
          <article className="mx-auto w-full max-w-3xl px-8 py-8">
            <h1 className="text-[22px] font-semibold tracking-[-0.03em]">{copy.work.taskEmpty}</h1>
            <p className="mt-2 text-[13px] leading-6 text-muted">{copy.work.taskEmptyHint}</p>
          </article>
        ) : (
          <TaskDocument ticket={ticket} onLook={props.onLook} />
        )}
      </div>
    </div>
  );
}

function TaskDocument(props: { ticket: Ticket; onLook: () => void; onContinue?: (text: string) => void; compact?: boolean }) {
  const copy = useCopy();
  const ticket = props.ticket;
  const key = stageKey(ticket);
  const inputs = ticket.inputs ?? [];
  const steps = ticket.assignments ?? [];
  const follows = ticket.follows ?? [];
  const latest = steps.slice(-1)[0];
  const earlier = steps.slice(0, -1);
  const body = latest ? assignmentBody(latest) : "";
  const names = manuscriptNames([body, ...inputs.map((row) => row.text), ...follows.map((row) => row.thread)].join("\n"));
  const empty = !body && inputs.length === 0 && follows.length === 0 && !ticket.deliveryError;
  return (
    <article className={cn("mx-auto w-full", props.compact ? "px-2 pb-8 pt-4" : "max-w-3xl px-8 pb-16 pt-2")}>
      {props.compact ? null : <h1 className="text-[28px] font-semibold leading-9 tracking-[-0.03em]">{ticket.title || ticket.id}</h1>}
      <p className="text-[12px] text-muted">{copy.work[key]}{ticket.sender ? ` · ${ticket.sender}` : ""}</p>
      {ticket.attention ? <p className="mt-4 text-[15px] leading-7">{ticket.attention}</p> : null}
      {empty ? <p className="mt-10 text-center text-[14px] text-muted">{copy.work.noSubtasks}</p> : null}
      {inputs.length > 0 ? (
        <div className="mt-4 space-y-3">
          {inputs.map((row) => (
            <p key={row.id} className="whitespace-pre-wrap text-[14px] leading-7 text-muted">{row.text}</p>
          ))}
        </div>
      ) : null}
      {body ? <div className="mt-4 whitespace-pre-wrap text-[14px] leading-7">{body}</div> : null}
      {earlier.length > 0 ? (
        <section className="mt-6">
          <h2 className="text-[13px] text-muted">{copy.work.earlier}</h2>
          <div className="mt-2 space-y-3">
            {earlier.map((step) => (
              <p key={step.id} className="whitespace-pre-wrap text-[13px] leading-6 text-muted">{assignmentBody(step)}</p>
            ))}
          </div>
        </section>
      ) : null}
      {names.length > 0 ? (
        <section className="mt-6">
          <h2 className="text-[13px] text-muted">{copy.work.taskManuscript}</h2>
          <ul className="mt-2">
            {names.map((name) => (
              <li key={name} className="border-b border-border/40">
                <button type="button" className="w-full py-2.5 text-left text-[13px] hover:text-foreground" onClick={props.onLook}>{name}</button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {follows.length > 0 ? (
        <section className="mt-6">
          <h2 className="text-[13px] text-muted">{copy.work.taskFollows}</h2>
          <ul className="mt-2">
            {follows.map((row) => (
              <li key={row.thread} className="border-b border-border/40 py-2.5 text-[13px]">
                {row.thread}{row.closed ? ` · ${row.closed}` : ""}
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {ticket.deliveryError ? <p className="mt-6 text-[13px] leading-6 text-danger">{ticket.deliveryError}</p> : null}
      <TaskActions ticket={ticket} onLook={props.onLook} onContinue={props.onContinue} />
    </article>
  );
}

function assignmentBody(step: Assignment): string {
  const summary = step.summary?.trim() || "";
  const report = step.report?.trim() || "";
  if (report && report !== summary) return report;
  return summary || report || step.state;
}

function TaskActions(props: { ticket: Ticket; onLook: () => void; onContinue?: (text: string) => void }) {
  const copy = useCopy();
  const key = stageKey(props.ticket);
  const [confirm, setConfirm] = useState<"" | "stop" | "retry">("");
  const canRetry = key === "blocked" || key === "failed" || key === "delivery";
  const canStop = props.ticket.stage !== "ended";
  async function act(kind: "stop" | "retry") {
    await orchestratorAct(kind, props.ticket.id, "", "", kind === "stop" ? "stopped" : "");
    setConfirm("");
  }
  return (
    <div className="mt-8 flex flex-wrap gap-2">
      {props.onContinue ? (
        <button
          type="button"
          className="h-9 rounded-full bg-card px-3.5 text-[13px] font-medium shadow-[var(--shadow-card)]"
          onClick={() => props.onContinue?.(`${copy.work.taskContinue}：${props.ticket.title || props.ticket.id}`)}
        >
          {copy.work.taskContinue}
        </button>
      ) : null}
      <button type="button" className="h-9 rounded-full bg-card px-3.5 text-[13px] font-medium shadow-[var(--shadow-card)]" onClick={props.onLook}>
        {copy.work.look}
      </button>
      {canStop ? (
        <button
          type="button"
          className={cn("h-9 rounded-full px-3.5 text-[13px] font-medium", confirm === "stop" ? "bg-foreground text-background" : "bg-card shadow-[var(--shadow-card)]")}
          onClick={() => {
            if (confirm === "stop") {
              void act("stop");
              return;
            }
            setConfirm("stop");
          }}
        >
          {confirm === "stop" ? copy.work.confirmStop : copy.work.stop}
        </button>
      ) : null}
      {canRetry ? (
        <button
          type="button"
          className={cn("h-9 rounded-full px-3.5 text-[13px] font-medium", confirm === "retry" ? "bg-foreground text-background" : "bg-card shadow-[var(--shadow-card)]")}
          onClick={() => {
            if (confirm === "retry") {
              void act("retry");
              return;
            }
            setConfirm("retry");
          }}
        >
          {confirm === "retry" ? copy.work.confirmRetry : copy.work.retry}
        </button>
      ) : null}
    </div>
  );
}
