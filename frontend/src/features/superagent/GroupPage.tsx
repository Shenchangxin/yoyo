/**
 * 拥有：任务组页。分工时是按依赖排好的成员车道、最近的动态和共享目录；对比时是路由的比较和并排的成员列，标出路由推荐的那一份，可以采用其中一份。
 * 头部有停止全部、重试没成的和去主会话。在侧栏里打开，也能放到中间。
 * 不拥有：成员怎么排、动态怎么取、时长怎么算（work-model.ts），也不拥有单张工单的会话（TaskPage.tsx）。
 */

import { useEffect, useState } from "react";
import { Check, ChevronLeft, FileText, FolderOpen, Maximize2, MessageSquare, RotateCcw, Square } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { StatusDot } from "../../components/ui/status-dot";
import { Tooltip } from "../../components/ui/tooltip";
import { openFilePreview, orchestratorAct, orchestratorTranscript, revealPath } from "../../lib/client";
import { fileBase, joinedPath } from "../../lib/file-paths";
import { useCopy } from "../../lib/i18n";
import { errMessage } from "../../lib/normalize";
import { cn } from "../../lib/utils";
import { BuddyMark } from "../buddy/BuddyMark";
import { fill } from "../buddy/buddy-model";
import { useBuddy } from "../buddy/buddy-store";
import { ticketAgent, usageText } from "./agent-model";
import type { Group, Ticket } from "./desk-model";
import { groupStateText, memberHint } from "./GroupRow";
import { taskStatus } from "./TaskPage";
import { agoLabel } from "./TaskRow";
import { taskFiles } from "./task-session";
import {
  canPick,
  durationLabel,
  groupActivity,
  groupActive,
  groupProgress,
  groupTone,
  memberLabel,
  memberMinutes,
  memberTone,
  retryableMembers,
  ticketLine,
  topoOrder,
  type GroupEvent,
} from "./work-model";

/** 车道按依赖深度缩进，再深的和第三层对齐。 */
const DEPTH_INDENT = ["", "ml-4", "ml-8", "ml-12"];

/** 对比列上最多列出几个改过的文件。 */
const COLUMN_FILES = 5;

/** GroupPage 是任务组页，stage 时铺在中间。点成员打开它的工单；onFocus 把这一页放到中间，onOpenMain 回到发起它的主会话。 */
export function GroupPage(props: {
  group: Group | null;
  tickets: Ticket[];
  onBack: () => void;
  onOpenTask: (id: string) => void;
  onOpenMain?: (session: string) => void;
  onFocus?: (id: string) => void;
  stage?: boolean;
  onSent?: () => void;
}) {
  const copy = useCopy();
  const t = copy.tasks;
  const group = props.group;
  const members = group ? group.tickets.map((id) => props.tickets.find((ticket) => ticket.id === id)).filter((ticket): ticket is Ticket => !!ticket) : [];
  const files = useMemberFiles(members.map((member) => member.id));
  const [now, setNow] = useState(() => Date.now());
  const [stopArmed, setStopArmed] = useState(false);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, []);
  useEffect(() => {
    if (!stopArmed) return;
    const timer = window.setTimeout(() => setStopArmed(false), 3000);
    return () => window.clearTimeout(timer);
  }, [stopArmed]);
  if (!group) {
    return (
      <div data-testid={props.stage ? "group-stage" : "group-page"} className="px-3 pt-8">
        <p className="text-ui-body-l text-muted">{copy.work.noSubtasks}</p>
      </div>
    );
  }
  const progress = groupProgress(members);
  const mode = group.mode === "compare" ? t.groupCompare : t.groupSplit;
  const retryable = retryableMembers(members);
  async function act(kind: "pick" | "stopGroup" | "retry", ids: string[]) {
    setBusy(true);
    try {
      for (const id of ids) await orchestratorAct(kind, id, "", kind === "retry" ? "" : group!.id, "");
      props.onSent?.();
    } catch (err) {
      toast.error(errMessage(err));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div data-testid={props.stage ? "group-stage" : "group-page"} data-group={group.id} className={cn("flex h-full min-h-0 flex-col bg-background", props.stage && "mx-auto w-full max-w-[1200px]")}>
      <div className={cn("flex shrink-0 items-center gap-1 pt-1", props.stage ? "px-3" : "px-1")}>
        <button type="button" className="grid size-8 shrink-0 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground" onClick={props.onBack} aria-label={props.stage ? copy.desk.taskBackMain : copy.work.taskBack}>
          <ChevronLeft className="size-4" />
        </button>
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-ui-title font-semibold" data-testid="group-title">{group.title || group.id}</h1>
          <p className="flex min-w-0 items-center gap-1.5 text-ui-footnote text-muted" data-testid="group-status">
            <StatusDot tone={groupTone(group.state)} />
            <span className="truncate">{[mode, groupStateText(group.state, t.groupState), `${progress.done}/${progress.total}`].join(" · ")}</span>
          </p>
        </div>
        <div className="flex shrink-0 items-center">
          {props.onFocus && !props.stage ? (
            <Tooltip content={copy.desk.taskFocus}>
              <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" aria-label={copy.desk.taskFocus} data-testid="group-focus" onClick={() => props.onFocus?.(group.id)}>
                <Maximize2 className="size-3.5" />
              </button>
            </Tooltip>
          ) : null}
          {group.session && props.onOpenMain ? (
            <Tooltip content={t.toMain}>
              <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" aria-label={t.toMain} data-testid="group-main" onClick={() => props.onOpenMain?.(group.session || "")}>
                <MessageSquare className="size-3.5" />
              </button>
            </Tooltip>
          ) : null}
          {retryable.length ? (
            <Tooltip content={t.retryFailed}>
              <button type="button" disabled={busy} className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground disabled:opacity-50" aria-label={t.retryFailed} data-testid="group-retry" onClick={() => { void act("retry", retryable.map((member) => member.id)); }}>
                <RotateCcw className="size-3.5" />
              </button>
            </Tooltip>
          ) : null}
          {groupActive(group.state) ? (
            stopArmed ? (
              <button type="button" className="h-7 rounded-md bg-danger/10 px-2 text-ui-footnote font-medium text-danger" data-testid="group-stop-confirm" onClick={() => { setStopArmed(false); void act("stopGroup", [""]); }}>
                {t.confirmStopAll}
              </button>
            ) : (
              <Tooltip content={t.stopAll}>
                <button type="button" className="grid size-7 place-items-center rounded-md text-muted hover:bg-lift hover:text-danger" aria-label={t.stopAll} data-testid="group-stop" onClick={() => setStopArmed(true)}>
                  <Square className="size-3" />
                </button>
              </Tooltip>
            )
          ) : null}
        </div>
      </div>
      <div className={cn("min-h-0 flex-1 overflow-y-auto pb-6 pt-3", props.stage ? "px-3" : "px-1")}>
        {group.mode === "compare" ? (
          <CompareView group={group} members={members} files={files} now={now} busy={busy} onOpen={props.onOpenTask} onPick={(id) => { void act("pick", [id]); }} />
        ) : (
          <SplitView group={group} members={members} files={files} now={now} onOpen={props.onOpenTask} />
        )}
      </div>
    </div>
  );
}

/** SplitView 是分工的样子：成员车道按依赖排，后面跟着最近的动态；放到中间且够宽时两栏并排。 */
function SplitView(props: { group: Group; members: Ticket[]; files: Record<string, string[]>; now: number; onOpen: (id: string) => void }) {
  const t = useCopy().tasks;
  const shared = props.group.sharedDir || "";
  return (
    <div className="@container">
      <div className="grid grid-cols-1 gap-5 @[900px]:grid-cols-[minmax(0,1fr)_340px]">
        <section data-testid="group-lanes">
          <h2 className="px-1 text-ui-caption font-medium text-muted">{t.members}</h2>
          <ul className="mt-1.5 space-y-2">
            {topoOrder(props.members).map(({ ticket, depth }) => (
              <li key={ticket.id} className={DEPTH_INDENT[Math.min(depth, DEPTH_INDENT.length - 1)]}>
                <Lane ticket={ticket} members={props.members} files={props.files[ticket.id]} now={props.now} onOpen={props.onOpen} />
              </li>
            ))}
          </ul>
          {shared ? (
            <button
              type="button"
              data-testid="group-shared"
              title={shared}
              className="mt-3 flex w-full min-w-0 items-center gap-2 rounded-lg px-2 py-1.5 text-left text-ui-footnote text-muted hover:bg-lift hover:text-foreground"
              onClick={() => { void revealPath(shared); }}
            >
              <FolderOpen className="size-3.5 shrink-0" aria-hidden />
              <span className="shrink-0">{t.sharedDir}</span>
              <span className="truncate font-mono text-ui-caption">{shared}</span>
            </button>
          ) : null}
        </section>
        <Activity members={props.members} onOpen={props.onOpen} />
      </div>
    </div>
  );
}

/** Lane 是分工里的一个成员：谁、做哪一份、现在怎样、最近一句，以及改了几个文件、做了多久、用了多少。 */
function Lane(props: { ticket: Ticket; members: Ticket[]; files: string[] | undefined; now: number; onOpen: (id: string) => void }) {
  const copy = useCopy();
  const t = copy.tasks;
  const list = useBuddy((s) => s.list);
  const ticket = props.ticket;
  const agent = ticketAgent(ticket);
  const v = list.find((row) => row.id === agent);
  const hint = memberHint(ticket, props.members, t);
  const line = ticketLine(ticket);
  const facts = memberFacts(ticket, props.members, props.files, props.now, t);
  return (
    <button
      type="button"
      data-testid="group-lane"
      data-ticket={ticket.id}
      className="flex w-full items-start gap-2.5 rounded-xl bg-card px-3 py-2.5 text-left shadow-[var(--shadow-card)]"
      onClick={() => props.onOpen(ticket.id)}
    >
      <BuddyMark v={v ?? { name: agent }} className="mt-px" />
      <span className="min-w-0 flex-1">
        <span className="flex min-w-0 items-center gap-1.5 text-ui-body">
          <span className="truncate font-medium">{ticket.role || ticket.title || ticket.id}</span>
          <span className="shrink-0 text-muted">· {v?.name || agent}</span>
          <StatusDot tone={memberTone(ticket)} className="ml-auto" />
        </span>
        <span className={cn("mt-0.5 block truncate text-ui-footnote leading-5", hint || (ticket.person && ticket.stage !== "ended") ? "text-warning" : "text-muted")} data-testid="group-lane-status">
          {hint || taskStatus(ticket, copy.work)}
        </span>
        {line ? <span className="mt-0.5 line-clamp-2 text-ui-footnote leading-5 text-foreground/80">{line}</span> : null}
        {facts ? <span className="mt-1 block truncate text-ui-caption text-muted">{facts}</span> : null}
      </span>
    </button>
  );
}

/** Activity 是任务组的动态：成员的报告按时间合成一条，新的在前，每条带着成员的字形和分工。还在等人回答的问题旁边给「去回答」。 */
function Activity(props: { members: Ticket[]; onOpen: (id: string) => void }) {
  const t = useCopy().tasks;
  const events = groupActivity(props.members);
  return (
    <section data-testid="group-activity">
      <h2 className="px-1 text-ui-caption font-medium text-muted">{t.activity}</h2>
      {events.length === 0 ? (
        <p className="px-1 py-2 text-ui-footnote text-muted">{t.activityEmpty}</p>
      ) : (
        <ul className="mt-1 divide-y divide-border/60">
          {events.map((event, i) => (
            <EventRow key={`${event.ticket.id}:${event.at}:${i}`} event={event} onOpen={props.onOpen} />
          ))}
        </ul>
      )}
    </section>
  );
}

/** EventRow 是动态里的一条：字形、分工、多久以前，下面是报告的话。 */
function EventRow(props: { event: GroupEvent; onOpen: (id: string) => void }) {
  const copy = useCopy();
  const list = useBuddy((s) => s.list);
  const event = props.event;
  const agent = ticketAgent(event.ticket);
  return (
    <li className="flex items-start gap-2 px-1 py-2" data-testid="group-event">
      <BuddyMark v={list.find((row) => row.id === agent) ?? { name: agent }} className="mt-px" />
      <span className="min-w-0 flex-1 text-ui-footnote leading-5">
        <span className="font-medium text-foreground/90">{memberLabel(event.ticket)}</span>
        <span className="text-muted"> · {agoLabel(event.at, copy.work)}</span>
        <span className={cn("mt-0.5 block line-clamp-3 whitespace-pre-line", event.ask ? "text-warning" : "text-foreground/80")}>{event.text}</span>
      </span>
      {event.ask ? (
        <Button size="sm" variant="outline" className="shrink-0" data-testid="group-answer" onClick={() => props.onOpen(event.ticket.id)}>
          {copy.tasks.goAnswer}
        </Button>
      ) : null}
    </li>
  );
}

/** CompareView 是对比的样子：最上面是路由的比较，下面每个成员一列，够宽时并排。采用了一份后，其余几列淡下去。 */
function CompareView(props: {
  group: Group;
  members: Ticket[];
  files: Record<string, string[]>;
  now: number;
  busy: boolean;
  onOpen: (id: string) => void;
  onPick: (id: string) => void;
}) {
  const t = useCopy().tasks;
  const verdict = props.group.verdict?.trim() || "";
  return (
    <div className="space-y-4">
      <section className="rounded-xl border-l-2 border-accent/60 bg-card px-3.5 py-3" data-testid="group-verdict">
        <h2 className="text-ui-caption font-medium text-muted">{t.verdictTitle}</h2>
        <p className={cn("mt-1 whitespace-pre-line text-ui-body", verdict ? "text-foreground/90" : "text-muted")}>{verdict || t.verdictPending}</p>
      </section>
      <div className="@container">
        <div className={cn("grid grid-cols-1 gap-3", props.members.length >= 3 ? "@[900px]:grid-cols-3" : "@[900px]:grid-cols-2")}>
          {props.members.map((ticket) => (
            <Column key={ticket.id} group={props.group} ticket={ticket} members={props.members} files={props.files[ticket.id]} now={props.now} busy={props.busy} onOpen={props.onOpen} onPick={props.onPick} />
          ))}
        </div>
      </div>
    </div>
  );
}

/** Column 是对比里的一份：谁做的、现在怎样、它的总结、改过的文件和花费。可以去看过程，也可以采用它。 */
function Column(props: {
  group: Group;
  ticket: Ticket;
  members: Ticket[];
  files: string[] | undefined;
  now: number;
  busy: boolean;
  onOpen: (id: string) => void;
  onPick: (id: string) => void;
}) {
  const copy = useCopy();
  const t = copy.tasks;
  const list = useBuddy((s) => s.list);
  const ticket = props.ticket;
  const agent = ticketAgent(ticket);
  const v = list.find((row) => row.id === agent);
  const picked = props.group.winner === ticket.id;
  const faded = !!props.group.winner && !picked;
  const recommended = !props.group.winner && props.group.recommended === ticket.id;
  const line = ticketLine(ticket);
  const dir = ticket.directory || "";
  const shown = (props.files ?? []).slice(0, COLUMN_FILES);
  const facts = memberFacts(ticket, props.members, props.files, props.now, t);
  return (
    <article
      data-testid="group-column"
      data-ticket={ticket.id}
      className={cn("flex min-w-0 flex-col rounded-xl bg-card p-3.5 shadow-[var(--shadow-card)] transition-opacity", faded && "opacity-55", picked && "ring-1 ring-accent/45")}
    >
      <header className="flex min-w-0 items-center gap-2">
        <BuddyMark v={v ?? { name: agent }} />
        <span className="truncate text-ui-body-l font-medium">{v?.name || agent}</span>
        {picked ? (
          <span className="ml-auto inline-flex shrink-0 items-center gap-1 rounded-full bg-accent/10 px-2 py-0.5 text-ui-caption font-medium text-accent" data-testid="group-picked">
            <Check className="size-3" aria-hidden />
            {t.picked}
          </span>
        ) : recommended ? (
          <span className="ml-auto inline-flex shrink-0 items-center gap-1.5 rounded-full bg-lift px-2 py-0.5 text-ui-caption font-medium text-foreground/85" data-testid="group-recommended">
            <StatusDot tone={memberTone(ticket)} />
            {t.recommended}
          </span>
        ) : (
          <StatusDot tone={memberTone(ticket)} className="ml-auto" />
        )}
      </header>
      <p className={cn("mt-0.5 truncate text-ui-footnote", ticket.person && ticket.stage !== "ended" ? "text-warning" : "text-muted")}>{taskStatus(ticket, copy.work)}</p>
      {line ? <p className="mt-2 line-clamp-6 whitespace-pre-line text-ui-body text-foreground/85">{line}</p> : null}
      {shown.length ? (
        <ul className="mt-2 divide-y divide-border/60">
          {shown.map((path) => (
            <li key={path}>
              <button type="button" className="flex w-full min-w-0 items-center gap-2 py-1.5 text-left text-ui-body text-foreground/85 hover:text-foreground" title={joinedPath(dir, path)} onClick={() => { void openFilePreview(dir, path); }}>
                <FileText className="size-3.5 shrink-0 text-muted" aria-hidden />
                <span className="truncate">{fileBase(path)}</span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      {facts ? <p className="mt-2 truncate text-ui-caption text-muted">{facts}</p> : null}
      <div className="mt-auto flex flex-wrap gap-2 pt-3">
        <Button size="sm" variant="outline" data-testid="group-view" onClick={() => props.onOpen(ticket.id)}>
          {t.viewWork}
        </Button>
        {canPick(props.group, ticket) ? (
          <Button size="sm" disabled={props.busy} data-testid="group-pick" onClick={() => props.onPick(ticket.id)}>
            {t.pick}
          </Button>
        ) : null}
      </div>
    </article>
  );
}

/** memberFacts 是成员的一行小字：改了几个文件、做了多久、用了多少。都没有时为空。filesChanged 是一个文件和多个文件的两种说法。 */
function memberFacts(
  ticket: Ticket,
  members: Ticket[],
  files: string[] | undefined,
  now: number,
  t: { filesChanged: readonly string[]; minutes: string; hours: string; usageContext: string },
): string {
  const minutes = memberMinutes(ticket, members, now);
  const count = files?.length ?? 0;
  return [
    count ? fill(count === 1 ? t.filesChanged[0] : t.filesChanged[1], { n: count }) : "",
    minutes === null ? "" : durationLabel(minutes, t),
    usageText(ticket.usage, t.usageContext),
  ]
    .filter(Boolean)
    .join(" · ");
}

/** useMemberFiles 每五秒读一遍成员的会话，收出各自改过的文件，最近改的在前。读不到的那一个保留上一次的结果。 */
function useMemberFiles(ids: string[]): Record<string, string[]> {
  const [files, setFiles] = useState<Record<string, string[]>>({});
  const key = ids.join(",");
  useEffect(() => {
    const list = key ? key.split(",") : [];
    if (list.length === 0) return;
    let stop = false;
    const pull = () => {
      void Promise.all(
        list.map((id) =>
          orchestratorTranscript(id)
            .then((rows) => [id, Array.isArray(rows) ? taskFiles(rows) : null] as const)
            .catch(() => [id, null] as const),
        ),
      ).then((pairs) => {
        if (stop) return;
        setFiles((prev) => {
          const next = { ...prev };
          for (const [id, found] of pairs) if (found) next[id] = found;
          return next;
        });
      });
    };
    pull();
    const timer = window.setInterval(pull, 5000);
    return () => {
      stop = true;
      window.clearInterval(timer);
    };
  }, [key]);
  return files;
}
