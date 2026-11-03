/**
 * 拥有：任务栏里这一页怎么把一张工单读成一条会话。
 * 不拥有：身份、模型和权限。那些留在主会话。下一句仍交给这张工单自己的工人。
 */

import { useEffect, useState } from "react";
import { ChevronLeft } from "lucide-react";
import { Composer } from "../Composer";
import { Transcript } from "../Transcript";
import { orchestratorAct, orchestratorTranscript } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import type { Assignment, ReportLine, Ticket } from "./desk-model";
import { stageKey } from "./work-model";

type TaskLine = { id: string; role: string; text: string; name?: string };

/** TaskRead 把工单读在侧栏里。底下是这条会话的输入，不带身份设置。 */
export function TaskRead(props: {
  ticket: Ticket | null;
  onBack: () => void;
  onLook: () => void;
  onSent?: () => void;
}) {
  const copy = useCopy();
  const ticket = props.ticket;
  const [lines, setLines] = useState<TaskLine[]>([]);
  useEffect(() => {
    if (!ticket?.id) return;
    let stop = false;
    const pull = () => {
      orchestratorTranscript(ticket.id).then((rows) => {
        if (!stop && Array.isArray(rows)) setLines(rows);
      }).catch(() => {});
    };
    pull();
    const timer = window.setInterval(pull, 2000);
    return () => {
      stop = true;
      window.clearInterval(timer);
    };
  }, [ticket?.id]);
  if (!ticket) {
    return (
      <div data-testid="task-read" className="px-3 pt-8">
        <p className="text-[14px] text-muted">{copy.work.noSubtasks}</p>
      </div>
    );
  }
  const running = ticket.stage === "working";
  return (
    <div data-testid="task-read" className="flex h-full min-h-0 flex-col bg-background">
      <div className="flex shrink-0 items-center gap-1 px-1 pt-1">
        <button type="button" className="grid size-8 shrink-0 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground" onClick={props.onBack} aria-label={copy.work.taskBack}>
          <ChevronLeft className="size-4" />
        </button>
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-[16px] font-semibold">{ticket.title || copy.work.taskEmpty}</h1>
          <p className="truncate text-[12px] text-muted">{taskStatus(ticket, copy.work)}{ticket.directory ? ` · ${ticket.directory}` : ""}</p>
        </div>
      </div>
      <Transcript compact flush items={sessionItems(ticket, lines)} approvals={[]} running={running} onResolve={() => {}} />
      <div className="shrink-0 px-2 pb-2">
        <Composer
          plain
          compact
          flush
          placeholder={copy.work.taskSayPh}
          draftKey={`task:${ticket.id}`}
          running={running}
          disabled={false}
          onSend={(opts) => {
            const line = opts?.text?.trim() || "";
            if (!line) return;
            void orchestratorAct("say", ticket.id, "", line, "").then(() => props.onSent?.());
          }}
          onStop={() => { void orchestratorAct("stop", ticket.id, "", "", ""); }}
          sessionId={ticket.id}
        />
      </div>
    </div>
  );
}

/** TaskPage 是任务被点开后的整页会话。同样不带身份设置。 */
export function TaskPage(props: {
  ticket: Ticket | null;
  onBack: () => void;
  onLook: () => void;
  onSent?: () => void;
}) {
  return <TaskRead {...props} />;
}

function sessionItems(ticket: Ticket, lines: TaskLine[]): Item[] {
  const fromLog = lines.map((line) => lineItem(ticket.id, line));
  const seen = new Set(fromLog.filter((item) => item.type === "user").map((item) => item.text.trim()));
  const head: Item[] = [];
  for (const row of ticket.inputs ?? []) {
    const text = row.text?.trim();
    if (!text || seen.has(text)) continue;
    head.push({
      key: row.id,
      type: "user",
      sessionId: ticket.id,
      source: "user",
      ts: "",
      text,
      name: "",
      delta: false,
      payload: { id: row.id },
    });
  }
  const body = fromLog.some((item) => item.type === "assistant") ? fromLog : [...fromLog, ...reportItems(ticket)];
  return [...head, ...body];
}

function lineItem(sessionId: string, line: TaskLine): Item {
  const role = line.role === "user" || line.role === "tool_call" || line.role === "tool_result" ? line.role : "assistant";
  return {
    key: line.id,
    type: role,
    sessionId,
    source: role === "user" ? "user" : "worker",
    ts: "",
    text: line.text || "",
    name: line.name || "",
    delta: false,
    payload: { id: line.id, name: line.name || "", ...(role === "tool_call" ? { arguments: line.text || "" } : {}) },
  };
}

function reportItems(ticket: Ticket): Item[] {
  const items: Item[] = [];
  for (const step of ticket.assignments ?? []) {
    reportLines(step).forEach((line, index) => {
      if (line.state === "progress" || line.state === "waiting" || line.state === "pending" || line.state === "running") return;
      const text = [line.summary, line.question].map((part) => part?.trim() || "").filter((part) => part && part !== "pending" && part !== "running").join("\n\n");
      if (!text) return;
      items.push({
        key: `${step.id}:${index}`,
        type: "assistant",
        sessionId: ticket.id,
        source: "worker",
        ts: "",
        text,
        name: "",
        delta: false,
        payload: { id: `${step.id}:${index}` },
      });
    });
  }
  return items;
}

function reportLines(step: Assignment): ReportLine[] {
  if (step.lines && step.lines.length > 0) return step.lines;
  if (step.summary || step.report) return [{ state: step.report || step.state, summary: step.summary }];
  return [];
}

function taskStatus(ticket: Ticket, work: { taskWorking: string; taskDone: string; taskStopped: string; taskFailed: string; queued: string; running: string; waiting: string; ended: string; blocked: string; failed: string; answer: string; reviewWait: string; delivery: string; idle: string }): string {
  if (ticket.stage === "ended") {
    if (ticket.outcome === "stopped") return work.taskStopped;
    if (ticket.outcome === "failed" || ticket.outcome === "dropped") return work.taskFailed;
    return work.taskDone;
  }
  const key = stageKey(ticket);
  if (key === "queued" || key === "running") return work.taskWorking;
  return work[key];
}
