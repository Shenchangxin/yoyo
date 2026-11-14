/**
 * 拥有：任务栏里这一页怎么把一张工单读成一条会话。在等你时问题放在最上面，回答就在原处。
 * 输入框里的停只停这一轮，工单留着；顶上的停止才结束工单，要点两次。
 * 不拥有：身份、模型和权限。那些留在主会话。下一句仍交给这张工单自己的工人。谁在做、换人和审批条在 TaskAgent.tsx。
 */

import { useEffect, useRef, useState } from "react";
import { ChevronLeft, ExternalLink, FileText, FolderOpen, Maximize2, RotateCcw, Square } from "lucide-react";
import { toast } from "sonner";
import { Composer } from "../Composer";
import { Transcript } from "../Transcript";
import { Button } from "../../components/ui/button";
import { Tooltip } from "../../components/ui/tooltip";
import { openFilePreview, openURL, orchestratorAct, orchestratorTranscript, revealPath } from "../../lib/client";
import { fileBase, isAbsPath, joinedPath } from "../../lib/file-paths";
import { useCopy } from "../../lib/i18n";
import { errMessage } from "../../lib/normalize";
import { latestTaskPlan } from "../../lib/plan";
import type { Attachment, FileHit, SkillInfo } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { external, ticketAgent, ticketNeed } from "./agent-model";
import type { Ticket } from "./desk-model";
import { useNeeds } from "./needs";
import { TaskAgentRow, TaskApprovals } from "./TaskAgent";
import { taskFiles, taskSessionItems, type TaskLine } from "./task-session";
import { stageKey, ticketLine } from "./work-model";

/** TaskRead 把工单读在侧栏里，stage 时铺在中间。底下是这条会话的输入，不带身份设置。任务组成员的标题上方有回到任务组的面包屑。 */
export function TaskRead(props: {
  ticket: Ticket | null;
  onBack: () => void;
  onFocus?: (id: string) => void;
  onOpenGroup?: (id: string) => void;
  stage?: boolean;
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
  const group = useNeeds((s) => (ticket?.group ? s.view?.groups?.find((row) => row.id === ticket.group) ?? null : null));
  const [lines, setLines] = useState<TaskLine[]>([]);
  const [now, setNow] = useState(() => Date.now());
  const [moved, setMoved] = useState(0);
  const seen = useRef("");
  const [stopArmed, setStopArmed] = useState(false);
  const running = !!ticket && ticket.stage === "working" && ticket.action !== "queued";
  const wasRunning = useRef(running);
  useEffect(() => {
    seen.current = "";
    setMoved(0);
  }, [ticket?.id]);
  useEffect(() => {
    if (running && !wasRunning.current) setMoved(Date.now());
    wasRunning.current = running;
  }, [running]);
  useEffect(() => {
    if (!ticket?.id) return;
    let stop = false;
    const pull = () => {
      orchestratorTranscript(ticket.id).then((rows) => {
        if (stop || !Array.isArray(rows)) return;
        const last = rows[rows.length - 1];
        const sig = `${rows.length}:${last?.id ?? ""}:${last?.text?.length ?? 0}`;
        if (seen.current && seen.current !== sig) setMoved(Date.now());
        seen.current = sig;
        setLines(rows);
      }).catch(() => {});
      setNow(Date.now());
    };
    pull();
    const timer = window.setInterval(pull, running ? 1500 : 4000);
    return () => {
      stop = true;
      window.clearInterval(timer);
    };
  }, [ticket?.id, running]);
  useEffect(() => {
    if (!stopArmed) return;
    const timer = window.setTimeout(() => setStopArmed(false), 3000);
    return () => window.clearTimeout(timer);
  }, [stopArmed]);
  if (!ticket) {
    return (
      <div data-testid="task-read" className="px-3 pt-8">
        <p className="text-ui-body-l text-muted">{copy.work.noSubtasks}</p>
      </div>
    );
  }
  const items = taskSessionItems(ticket, lines);
  const draftKey = `task:${ticket.id}`;
  const open = ticket.stage !== "ended";
  const canRetry = open && (ticket.action === "blocked" || ticket.action === "failed" || ticket.action === "delivery");
  const updated = Date.parse(ticket.updatedAt || "");
  const since = Math.max(Number.isFinite(updated) ? updated : 0, moved);
  const quiet = running && since > 0 ? Math.floor((now - since) / 60000) : 0;
  const status = [taskStatus(ticket, copy.work), quiet >= 2 ? copy.work.taskStale.replace("{n}", String(quiet)) : ""].filter(Boolean).join(" · ");
  const ask = taskAsk(ticket, copy.work);
  const crumb = group ? [copy.tasks.groupCrumb.replace("{title}", group.title || group.id), ticket.role || ""].filter(Boolean).join(" · ") : "";
  async function act(kind: "stop" | "retry" | "pause") {
    try {
      await orchestratorAct(kind, ticket!.id, "", "", "");
      props.onSent?.();
    } catch (err) {
      const text = errMessage(err);
      if (kind === "pause" && text.includes("notRunning")) return;
      toast.error(text);
    }
  }
  return (
    <div data-testid={props.stage ? "task-stage" : "task-read"} className={cn("flex h-full min-h-0 flex-col", props.stage ? "mx-auto w-full max-w-[880px] bg-background" : "bg-transparent")}>
      <div className={cn("flex shrink-0 items-center gap-1 pt-1", props.stage ? "px-3" : "px-1")}>
        <button type="button" className="grid size-8 shrink-0 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground" onClick={props.onBack} aria-label={props.stage ? copy.desk.taskBackMain : copy.work.taskBack}>
          <ChevronLeft className="size-4" />
        </button>
        <div className="min-w-0 flex-1">
          {crumb && group && props.onOpenGroup ? (
            <button type="button" data-testid="task-group-crumb" className="block max-w-full truncate text-left text-ui-caption text-muted hover:text-foreground" onClick={() => props.onOpenGroup?.(group.id)}>
              {crumb}
            </button>
          ) : crumb ? (
            <p data-testid="task-group-crumb" className="truncate text-ui-caption text-muted">{crumb}</p>
          ) : null}
          <h1 className="truncate text-ui-title-s font-semibold">{ticket.title || copy.work.taskEmpty}</h1>
          <p className={cn("truncate text-ui-footnote", ticket.person && open ? "text-warning" : "text-muted")} data-testid="task-status">{status}</p>
          <TaskAgentRow ticket={ticket} onSent={props.onSent} />
        </div>
        <div className="flex shrink-0 items-center">
          {props.onFocus && !props.stage ? (
            <Tooltip content={copy.desk.taskFocus}>
              <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" aria-label={copy.desk.taskFocus} data-testid="task-focus" onClick={() => props.onFocus?.(ticket.id)}>
                <Maximize2 className="size-3.5" />
              </button>
            </Tooltip>
          ) : null}
          {ticket.directory ? (
            <Tooltip content={copy.work.taskOpenDir}>
              <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" aria-label={copy.work.taskOpenDir} onClick={() => { void revealPath(ticket.directory || ""); }}>
                <FolderOpen className="size-3.5" />
              </button>
            </Tooltip>
          ) : null}
          {ticket.originUrl ? (
            <Tooltip content={copy.work.taskOrigin}>
              <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" aria-label={copy.work.taskOrigin} onClick={() => { void openURL(ticket.originUrl || ""); }}>
                <ExternalLink className="size-3.5" />
              </button>
            </Tooltip>
          ) : null}
          {canRetry ? (
            <Tooltip content={copy.work.confirmRetry}>
              <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" aria-label={copy.work.retry} onClick={() => { void act("retry"); }}>
                <RotateCcw className="size-3.5" />
              </button>
            </Tooltip>
          ) : null}
          {open ? (
            stopArmed ? (
              <button type="button" className="h-7 rounded-md bg-danger/10 px-2 text-ui-caption font-medium text-danger" onClick={() => { setStopArmed(false); void act("stop"); }}>
                {copy.work.confirmStop}
              </button>
            ) : (
              <Tooltip content={`${copy.work.stop} · ${copy.work.taskStopHint}`}>
                <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-danger" aria-label={copy.work.stop} onClick={() => setStopArmed(true)}>
                  <Square className="size-3" />
                </button>
              </Tooltip>
            )
          ) : null}
        </div>
      </div>
      <TaskApprovals ticket={ticket} className={props.stage ? "mx-3" : "mx-2"} />
      {ask ? (
        <div className={cn("mt-2 shrink-0 rounded-xl border border-border border-l-2 border-l-warning bg-card px-3 py-2.5 shadow-[var(--shadow-card)]", props.stage ? "mx-3" : "mx-2")} data-testid="task-ask">
          <p className="text-ui-footnote font-medium text-foreground/85">{ask.title}</p>
          {ask.detail ? <p className="mt-1 line-clamp-4 whitespace-pre-line text-ui-body leading-5">{ask.detail}</p> : null}
          {ticketNeed(ticket) === "login" ? (
            <Button size="sm" variant="outline" className="mt-2" data-testid="task-go-login" onClick={() => useUI.getState().openBuddy(ticketAgent(ticket))}>
              {copy.tasks.goLogin}
            </Button>
          ) : null}
          {ticket.action === "review" ? <TaskReview ticket={ticket} files={taskFiles(lines)} onDone={() => props.onSent?.()} /> : null}
        </div>
      ) : null}
      {items.length === 0 ? (
        <p className="min-h-0 flex-1 px-4 py-8 text-center text-ui-body leading-5 text-muted">{copy.work.taskEmptyHint}</p>
      ) : (
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
      )}
      <div className={cn("shrink-0 pb-2", props.stage ? "px-3" : "px-2")}>
        {props.stage ? (
          <p className="mb-1 truncate px-1 text-ui-caption text-muted" data-testid="task-target">{copy.desk.taskTarget.replace("{title}", ticket.title || copy.work.taskEmpty)}</p>
        ) : null}
        <Composer
          plain
          plus
          compact={!props.stage}
          flush
          placeholder={open && ticket.action === "answer" ? copy.work.taskAnswerPh : running && external(ticket) ? copy.tasks.afterStep : copy.work.taskSayPh}
          draftKey={draftKey}
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
            orchestratorAct("say", ticket.id, "", line, "")
              .then(() => props.onSent?.())
              .catch((err) => {
                // 输入框在发出时已经清空。没送到时把原话放回去，不让这句话丢掉。
                const ui = useUI.getState();
                if (!(ui.drafts[draftKey] || "").trim()) ui.setDraft(draftKey, opts?.text || line);
                toast.error(copy.work.taskSendFailed.replace("{error}", errMessage(err)));
              });
          }}
          onStop={() => { void act("pause"); }}
          sessionId={ticket.id}
        />
      </div>
    </div>
  );
}

/** TaskPage 是任务放到中间后的整页会话。同样不带身份设置；返回键回到主对话。 */
export function TaskPage(props: {
  ticket: Ticket | null;
  onBack: () => void;
  onOpenGroup?: (id: string) => void;
  onSent?: () => void;
  skills?: SkillInfo[];
  files?: FileHit[];
  workspace?: string;
  onSearchFiles?: (q: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
  onClipboard?: () => Promise<string>;
  onScreenshot?: () => Promise<string>;
}) {
  return <TaskRead {...props} stage />;
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

/** TaskReview 是等验收时的那一排：改过的文件点开就能看，通过就收下；不通过要写清哪里不对，退回给同一个工人接着改。 */
function TaskReview(props: { ticket: Ticket; files: string[]; onDone: () => void }) {
  const copy = useCopy();
  const [sending, setSending] = useState(false);
  const [back, setBack] = useState(false);
  const [reason, setReason] = useState("");
  const dir = props.ticket.directory || "";
  const shown = props.files.slice(0, 5);
  const latest = props.files[0] ? joinedPath(dir, props.files[0]) : "";
  const folder = latest && isAbsPath(latest) ? latest : dir;
  async function send(kind: "accept" | "say", text = "") {
    setSending(true);
    try {
      await orchestratorAct(kind, props.ticket.id, "", text, "");
      setBack(false);
      setReason("");
      props.onDone();
    } catch (err) {
      toast.error(errMessage(err));
    } finally {
      setSending(false);
    }
  }
  function sendBack() {
    const text = reason.trim();
    if (text && !sending) void send("say", copy.work.taskRejectSay.replace("{reason}", text));
  }
  return (
    <div className="mt-2" data-testid="task-review">
      {shown.length ? (
        <div className="mb-2">
          <p className="text-ui-caption text-muted">{copy.work.taskFiles}</p>
          <ul className="mt-0.5 divide-y divide-border/60">
            {shown.map((path) => (
              <li key={path}>
                <button type="button" className="flex w-full min-w-0 items-center gap-2 py-1.5 text-left text-ui-body text-foreground/85 hover:text-foreground" title={joinedPath(dir, path)} onClick={() => { void openFilePreview(dir, path); }}>
                  <FileText className="size-3.5 shrink-0 text-muted" aria-hidden />
                  <span className="truncate">{fileBase(path)}</span>
                </button>
              </li>
            ))}
          </ul>
          {props.files.length > shown.length ? <p className="mt-0.5 text-ui-caption text-muted">{copy.work.taskFilesMore.replace("{n}", String(props.files.length - shown.length))}</p> : null}
        </div>
      ) : null}
      {back ? (
        <form onSubmit={(event) => { event.preventDefault(); sendBack(); }}>
          <textarea
            autoFocus
            rows={3}
            value={reason}
            placeholder={copy.work.taskRejectPh}
            aria-label={copy.work.taskReject}
            onChange={(event) => setReason(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape") setBack(false);
              if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
                event.preventDefault();
                sendBack();
              }
            }}
            className="no-drag w-full resize-none rounded-lg border border-border bg-background px-2.5 py-2 text-ui-body text-foreground outline-none placeholder:text-muted/70 focus-visible:border-accent/45 focus-visible:ring-2 focus-visible:ring-accent/20"
          />
          <div className="mt-2 flex items-center gap-2">
            <Button type="submit" size="sm" disabled={sending || !reason.trim()}>{copy.work.taskRejectSend}</Button>
            <Button size="sm" variant="ghost" onClick={() => setBack(false)}>{copy.dialog.cancel}</Button>
          </div>
        </form>
      ) : (
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" disabled={sending} onClick={() => { void send("accept"); }}>{copy.work.taskAccept}</Button>
          <Button size="sm" variant="outline" disabled={sending} onClick={() => setBack(true)}>{copy.work.taskReject}</Button>
          {folder ? <Button size="sm" variant="ghost" onClick={() => { void revealPath(folder); }}>{copy.work.taskLookDir}</Button> : null}
        </div>
      )}
    </div>
  );
}

/** taskAsk 是这张工单在等你时放在最上面的那一块：在等什么，以及工人留下的那一句。不在等你时为空。 */
function taskAsk(ticket: Ticket, work: { taskAskAnswer: string; taskAskReview: string; taskAskBlocked: string; taskAskFailed: string; taskAskDelivery: string }): { title: string; detail: string } | null {
  if (!ticket.person || ticket.stage === "ended") return null;
  const titles: Record<string, string> = {
    answer: work.taskAskAnswer,
    review: work.taskAskReview,
    blocked: work.taskAskBlocked,
    failed: work.taskAskFailed,
    delivery: work.taskAskDelivery,
  };
  const title = titles[ticket.action] || work.taskAskBlocked;
  const detail = ticket.action === "answer" && ticket.attention?.trim()
    ? ticket.attention.trim()
    : ticket.action === "delivery" && ticket.deliveryError?.trim()
      ? ticket.deliveryError.trim()
      : ticketLine(ticket);
  return { title, detail };
}

/** taskStatus 用人看得懂的一句说这张工单在等谁、需要什么。列表和这一页用同一套说法。 */
export function taskStatus(ticket: Ticket, work: { taskWorking: string; taskDone: string; taskStopped: string; taskFailed: string; queued: string; running: string; waiting: string; ended: string; blocked: string; failed: string; answer: string; reviewWait: string; delivery: string; idle: string }): string {
  if (ticket.stage === "ended") {
    if (ticket.outcome === "stopped") return work.taskStopped;
    if (ticket.outcome === "failed" || ticket.outcome === "dropped") return work.taskFailed;
    return work.taskDone;
  }
  const key = stageKey(ticket);
  if (key === "running") return work.taskWorking;
  if (key === "queued") return work.queued;
  return work[key];
}
