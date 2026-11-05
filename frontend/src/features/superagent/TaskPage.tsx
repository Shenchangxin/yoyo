/**
 * 拥有：任务栏里这一页怎么把一张工单读成一条会话。
 * 不拥有：身份、模型和权限。那些留在主会话。下一句仍交给这张工单自己的工人。
 */

import { useEffect, useState } from "react";
import { ChevronLeft } from "lucide-react";
import { Composer } from "../Composer";
import { Transcript } from "../Transcript";
import { orchestratorAct, orchestratorTranscript, revealPath } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { latestTaskPlan } from "../../lib/plan";
import type { Attachment, FileHit, SkillInfo } from "../../lib/protocol";
import type { Ticket } from "./desk-model";
import { taskSessionItems, type TaskLine } from "./task-session";
import { stageKey } from "./work-model";

/** TaskRead 把工单读在侧栏里。底下是这条会话的输入，不带身份设置。 */
export function TaskRead(props: {
  ticket: Ticket | null;
  onBack: () => void;
  onLook: () => void;
  onSent?: () => void;
  skills?: SkillInfo[];
  files?: FileHit[];
  workspace?: string;
  onSearchFiles?: (q: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
  onClipboard?: () => Promise<string>;
  onScreenshot?: () => Promise<string>;
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
  const items = taskSessionItems(ticket, lines);
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
      <Transcript
        compact
        flush
        linkFiles
        items={items}
        approvals={[]}
        running={running}
        workspace={ticket.directory || props.workspace}
        onRevealPath={(path) => { void revealPath(path); }}
        onResolve={() => {}}
      />
      <div className="shrink-0 px-2 pb-2">
        <Composer
          plain
          plus
          compact
          flush
          placeholder={copy.work.taskSayPh}
          draftKey={`task:${ticket.id}`}
          taskPlan={latestTaskPlan(items)}
          running={running}
          disabled={false}
          skills={props.skills}
          files={props.files}
          workspace={props.workspace}
          onSearchFiles={props.onSearchFiles}
          onPickFiles={props.onPickFiles}
          onClipboard={props.onClipboard}
          onScreenshot={props.onScreenshot}
          onSend={(opts) => {
            const line = workerLine(opts?.text, opts?.attachments);
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
  skills?: SkillInfo[];
  files?: FileHit[];
  workspace?: string;
  onSearchFiles?: (q: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
  onClipboard?: () => Promise<string>;
  onScreenshot?: () => Promise<string>;
}) {
  return <TaskRead {...props} />;
}

/** workerLine 把正文和附件路径收成工人能读的一句。技能标记已经在正文里。 */
function workerLine(text?: string, attachments?: Attachment[]): string {
  const body = (text || "").trim();
  const files = (attachments || [])
    .map((item) => (item.path || item.name || "").trim())
    .filter((path) => path && !body.includes(path));
  const stamp = files.map((path) => `@file:${path}`).join(" ");
  return [body, stamp].filter(Boolean).join("\n");
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
