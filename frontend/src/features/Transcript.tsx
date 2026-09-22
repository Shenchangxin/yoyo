import { memo, useState, type ReactNode } from "react";
import { ArrowDown, ArrowUpRight, Check, CircleDashed, Copy, FileText, RotateCcw, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { StickToBottom, useStickToBottomContext } from "use-stick-to-bottom";
import { Markdown } from "../lib/markdown";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import { writeClipboard } from "../lib/clipboard";
import { LONG_THREAD_TURNS, THREAD_COL, THREAD_GUTTER, THREAD_GUTTER_COMPACT } from "../lib/thread";
import {
  formatToolBody,
  itemTimeMs,
  patchFileCount,
  toolArgs,
  toolDetail,
  toolName,
} from "../lib/tool-summary";
import { artifactPreviewOpen, artifactShouldShow, artifactView } from "../lib/artifact-preview";
import { extractHTML, looksLikeHTML, looksLikeHTMLFile, looksLikePDF } from "../lib/html-preview";
import type { Approval, Item } from "../lib/protocol";
import { classifyItem, errorCopy } from "../lib/error";
import { layoutRows, pairShowsArtifact, pairTools, processGroupLive, type AgentPart, type LayoutRow } from "../lib/transcript-layout";
import { ProcessGroup, ToolLine, WorkingLine } from "./transcript/ProcessGroup";
import { ArtifactBody } from "./transcript/FilePreview";
import { SandboxedFrame } from "./transcript/SandboxedFrame";
import { MarkWell } from "./shell/YoyoMark";
import { displayWorkspace } from "../lib/display-title";

function lastRealUserIndex(items: Item[]): number {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "user" && items[i].source !== "steer") return i;
  }
  return -1;
}

function lastMeaningful(items: Item[]): Item | null {
  const start = lastRealUserIndex(items);
  for (let i = items.length - 1; i > start; i--) {
    const it = items[i];
    if (it.type === "error" || it.type === "compaction") continue;
    if (it.source === "steer") continue;
    return it;
  }
  return null;
}

/** Start of the live clock: the operator's last real message. */
function lastUserTimeMs(items: Item[]): number {
  const i = lastRealUserIndex(items);
  return i >= 0 ? itemTimeMs(items[i]) : 0;
}

/**
 * The tail part of the tail agent row owns the live indicator. A process
 * group there narrates itself ("Reading …" / "Thinking"); an artifact with an
 * open call shows its own pending state; anything else gets the working line.
 */
function tailOwnsActivity(rows: LayoutRow[]): boolean {
  const row = rows[rows.length - 1];
  if (!row || row.kind !== "agent") return false;
  const part = row.parts[row.parts.length - 1];
  if (!part) return false;
  if (part.kind === "process") return true;
  if (part.kind === "artifact") return processGroupLive(part.items);
  return false;
}

function lastErrorKey(items: Item[]): string {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "error") return items[i].key;
  }
  return "";
}

function lastAgentKey(rows: LayoutRow[]): string {
  for (let i = rows.length - 1; i >= 0; i--) {
    if (rows[i].kind === "agent") return rows[i].key;
  }
  return "";
}

function turnHasArtifact(parts: AgentPart[]): boolean {
  return parts.some((p) => p.kind === "artifact");
}

function isUserRow(row: LayoutRow): boolean {
  return row.kind === "user";
}

function rowSpace(rows: LayoutRow[], i: number): string {
  const row = rows[i];
  const prev = rows[i - 1];
  if (!prev) return "pt-0";
  if (isUserRow(row)) return "pt-10";
  if (row.kind === "agent" && prev.kind === "user") return "pt-4";
  if (row.kind === "agent") return "pt-5";
  return "pt-4";
}

function partSpace(parts: AgentPart[], i: number): string {
  if (i === 0) return "";
  const prev = parts[i - 1];
  const cur = parts[i];
  if (cur.kind === "process") return prev.kind === "item" ? "pt-2.5" : "pt-1.5";
  if (cur.kind === "artifact") return "pt-2";
  if (cur.kind === "item" && cur.item.type === "assistant") {
    if (prev.kind === "process" || prev.kind === "artifact") return "pt-3";
    return "pt-2";
  }
  return "pt-2";
}

export function Transcript(props: {
  items: Item[];
  approvals: Approval[];
  running: boolean;
  compact?: boolean;
  flush?: boolean;
  workspace?: string;
  onResolve: (id: string, decision: string) => void;
  onPrompt?: (text: string) => void;
  onOpenReview?: (path?: string) => void;
  onRetry?: () => void;
}) {
  const pad = props.flush ? "" : props.compact ? THREAD_GUTTER_COMPACT : THREAD_GUTTER;
  const col = props.flush ? "w-full min-w-0" : cn(THREAD_COL, pad);
  const layout = layoutRows(props.items);
  const empty = props.items.length === 0 && !props.running && props.approvals.length === 0;
  const live = lastMeaningful(props.items);
  const streamingKey = props.running && live?.type === "assistant" && live.delta ? live.key : "";
  const showWorking = props.running && !streamingKey && !tailOwnsActivity(layout);
  const since = props.running ? lastUserTimeMs(props.items) : 0;
  const retryKey = lastErrorKey(props.items);
  const pinnedTurn = lastAgentKey(layout);
  const longThread = layout.length > LONG_THREAD_TURNS;

  const rows: { key: string; space?: string; virtualize?: boolean; node: ReactNode }[] = [
    ...layout.map((row, i) => {
      const virtualize = longThread && i < layout.length - 2;
      if (row.kind === "user") {
        return {
          key: row.key,
          space: rowSpace(layout, i),
          virtualize,
          node: <ItemRow item={row.item} copyText={props.compact ? "" : row.item.text} />,
        };
      }
      if (row.kind === "agent") {
        const liveHere = row.parts.some((p) => p.kind === "item" && p.item.key === streamingKey);
        const last = row.key === pinnedTurn;
        const workingHere = last && showWorking;
        const tail = row.parts.length - 1;
        return {
          key: row.key,
          space: rowSpace(layout, i),
          virtualize: virtualize && !liveHere && !workingHere,
          node: (
            <article className="assistant-letter group/turn" data-testid="agent-turn">
              {row.parts.map((part, pi) => (
                <div
                  key={part.key}
                  className={cn(
                    partSpace(row.parts, pi),
                    part.kind === "item" && part.item.type === "assistant" && "assistant-block min-w-0",
                    (part.kind === "process" || part.kind === "artifact") && "u-chrome",
                  )}
                >
                  {part.kind === "process" ? (
                    <ProcessGroup
                      items={part.items}
                      live={props.running && last && pi === tail}
                      running={props.running}
                      compact={props.compact}
                    />
                  ) : part.kind === "artifact" ? (
                    <ArtifactTimeline items={part.items} running={props.running} workspace={props.workspace} onOpenReview={props.onOpenReview} />
                  ) : (
                    <ItemRow
                      item={part.item}
                      streaming={part.item.key === streamingKey}
                    />
                  )}
                </div>
              ))}
              {workingHere ? (
                <div className={row.parts.length ? "pt-1.5" : undefined}>
                  <WorkingLine since={since} />
                </div>
              ) : null}
              {row.copyText && !props.compact ? (
                <div className="u-chrome">
                  <TurnActions
                    text={row.copyText}
                    live={liveHere || (last && props.running)}
                    pinned={last && !props.running}
                    showReview={turnHasArtifact(row.parts)}
                    onOpenReview={props.onOpenReview}
                  />
                </div>
              ) : null}
            </article>
          ),
        };
      }
      return {
        key: row.key,
        space: rowSpace(layout, i),
        virtualize,
        node: (
          <ItemRow
            item={row.item}
            onRetry={row.item.key === retryKey ? props.onRetry : undefined}
          />
        ),
      };
    }),
    ...props.approvals.map((a) => ({
      key: `ask:${a.id}`,
      space: "pt-4",
      node: <ApprovalCard item={a} onResolve={props.onResolve} />,
    })),
  ];
  if (showWorking && !pinnedTurn) {
    rows.push({
      key: "working",
      space: "pt-3",
      node: <WorkingLine since={since} />,
    });
  }

  return (
    <StickToBottom
      className="prose-select relative flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      resize="instant"
      initial="instant"
      data-testid="conversation-column"
    >
      {empty && !props.compact ? (
        <StickToBottom.Content
          scrollClassName="transcript-scroll"
          className={cn(col, "flex min-h-full flex-col justify-end pb-7 pt-8")}
        >
          <EmptyTurn workspace={props.workspace} onPrompt={props.onPrompt} />
        </StickToBottom.Content>
      ) : (
        <StickToBottom.Content
          scrollClassName="transcript-scroll"
          className={cn(col, "flex flex-col pb-3 pt-4")}
        >
          {rows.map((row) => (
            <div
              key={row.key}
              className={row.space}
              style={row.virtualize ? { contentVisibility: "auto", containIntrinsicSize: "auto 160px" } : undefined}
            >
              {row.node}
            </div>
          ))}
        </StickToBottom.Content>
      )}
      {props.compact ? null : <JumpLatest />}
    </StickToBottom>
  );
}

/**
 * Empty thread. Sits just above the composer so the greeting, the hints and
 * the input read as one block instead of a headline floating in a void.
 */
function EmptyTurn({ workspace, onPrompt }: { workspace?: string; onPrompt?: (text: string) => void }) {
  const copy = useCopy();
  const surface = useUI((s) => s.surface);
  const videoMode = useUI((s) => s.videoMode);
  const videoing = surface === "video";
  const ready = !videoing
    ? copy.transcript.ready
    : videoMode === "canvas"
      ? copy.transcript.canvasReady
      : videoMode === "creative"
        ? copy.transcript.creativeReady
        : copy.transcript.videoReady;
  const starters = !videoing
    ? copy.transcript.starters
    : videoMode === "canvas"
      ? copy.transcript.canvasStarters
      : videoMode === "creative"
        ? copy.transcript.creativeStarters
        : copy.transcript.videoStarters;
  const ws = displayWorkspace(workspace, "");
  const hints = [
    ws ? { key: "ws", node: <><span className="text-muted">{copy.transcript.workingIn}</span><span className="ml-1.5 font-mono text-[12px] text-foreground/80">{ws}</span></> } : null,
    { key: "enter", node: <><kbd className="mr-1">↵</kbd>{copy.transcript.hintSend}</> },
    videoing
      ? { key: "mode", node: copy.transcript.hintVideoMode }
      : { key: "at", node: <><kbd className="mr-1">@</kbd>{copy.transcript.hintMention}</> },
    videoing ? null : { key: "plan", node: <><kbd className="mr-1">⇧⇥</kbd>{copy.transcript.hintPlan}</> },
  ].filter(Boolean) as { key: string; node: ReactNode }[];
  return (
    <div className="empty-rise" data-testid="empty-turn">
      <MarkWell />
      <h1 className="mt-5 max-w-[18ch] text-[21px] font-semibold tracking-[-0.03em] text-pretty text-foreground">
        {ready}
      </h1>
      <p className="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-[12.5px] leading-[1.6] text-muted">
        {hints.map((h, i) => (
          <span key={h.key} className="inline-flex items-center">
            {i > 0 ? <span className="mr-3 text-muted/40" aria-hidden>·</span> : null}
            {h.node}
          </span>
        ))}
      </p>
      <div className="mt-6 flex flex-wrap gap-1.5">
        {starters.map((s, i) => (
          <button
            type="button"
            key={s.label}
            className="inline-flex cursor-pointer items-center gap-1.5 rounded-full border border-border/80 bg-card/50 py-1.5 pl-3 pr-2.5 text-[12.5px] text-foreground/90 transition-[border-color,background-color,color] duration-150 hover:border-border hover:bg-lift/60 hover:text-foreground"
            style={{ animationDelay: `${60 + i * 40}ms` }}
            onClick={() => onPrompt?.(s.text)}
          >
            {s.label}
            <ArrowUpRight className="size-3 text-muted/60" aria-hidden />
          </button>
        ))}
      </div>
    </div>
  );
}

function JumpLatest() {
  const copy = useCopy();
  const ctx = useStickToBottomContext() as ReturnType<typeof useStickToBottomContext> & {
    isAtBottom: boolean;
    escapedFromLock: boolean;
  };
  if (ctx.isAtBottom && !ctx.escapedFromLock) return null;
  return (
    <button
      type="button"
      className="absolute bottom-3 left-1/2 z-10 inline-flex -translate-x-1/2 cursor-pointer items-center gap-1.5 rounded-full border border-border bg-popover/95 py-1.5 pl-2.5 pr-3 text-[12px] text-foreground shadow-[var(--shadow-popover)] backdrop-blur-md transition-colors hover:bg-lift"
      onClick={() => {
        void ctx.scrollToBottom();
      }}
    >
      <ArrowDown className="size-3 text-muted" aria-hidden />
      {copy.transcript.jumpLatest}
    </button>
  );
}

function TurnActions({
  text,
  live,
  pinned,
  showReview,
  onOpenReview,
}: {
  text: string;
  live?: boolean;
  pinned?: boolean;
  showReview?: boolean;
  onOpenReview?: (path?: string) => void;
}) {
  const copy = useCopy();
  if (live) return <div className="h-8" aria-hidden />;
  return (
    <div
      className={cn(
        "mt-1 flex h-8 items-center gap-0.5 transition-opacity duration-150",
        pinned
          ? "opacity-100"
          : "pointer-events-none opacity-0 group-hover/turn:pointer-events-auto group-hover/turn:opacity-100 group-focus-within/turn:pointer-events-auto group-focus-within/turn:opacity-100",
      )}
    >
      <CopyAction text={text} />
      {showReview && onOpenReview ? (
        <button
          type="button"
          className="inline-flex h-7 items-center gap-1 rounded-md px-1.5 text-[11px] text-muted hover:bg-lift hover:text-foreground"
          onClick={() => onOpenReview()}
        >
          <FileText className="size-3.5" aria-hidden />
          {copy.review.openReview}
        </button>
      ) : null}
    </div>
  );
}

function ApprovalCard({ item, onResolve }: { item: Approval; onResolve: (id: string, decision: string) => void }) {
  const copy = useCopy();
  const subject = item.command || item.path || item.level;
  return (
    <div
      className="surface-inset relative overflow-hidden rounded-2xl border border-border bg-card/90 px-4 py-3.5"
      role="status"
      data-testid="approval-card"
    >
      <span className="pointer-events-none absolute inset-y-3 left-0 w-[2px] rounded-full bg-warning/80" aria-hidden />
      <div className="flex items-start gap-3">
        <span className="mt-px grid size-7 shrink-0 place-items-center rounded-lg bg-warning/12 text-warning" aria-hidden>
          <ShieldAlert className="size-3.5" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 text-[11px] font-medium uppercase tracking-[0.08em] text-muted">
            {copy.transcript.needsApproval}
          </div>
          <div className="mt-0.5 text-[13.5px] font-medium tracking-[-0.015em] text-foreground">{item.action || "action"}</div>
          {subject ? (
            <pre className="mt-2 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[11.5px] leading-[1.5] text-foreground/85">
              {subject}
            </pre>
          ) : null}
          <div className="mt-3 flex flex-wrap items-center gap-1.5">
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background transition-opacity hover:opacity-90"
              onClick={() => onResolve(item.id, "once")}
            >
              {copy.transcript.allow}
              <kbd className="border-background/20 bg-background/15 text-background/80" aria-hidden>1</kbd>
            </button>
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-[12px] text-foreground transition-colors hover:bg-lift"
              onClick={() => onResolve(item.id, "session")}
            >
              {copy.transcript.allowSession}
              <kbd aria-hidden>2</kbd>
            </button>
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[12px] text-muted transition-colors hover:bg-lift hover:text-foreground"
              onClick={() => onResolve(item.id, "always")}
            >
              {copy.transcript.allowAlways}
              <kbd aria-hidden>3</kbd>
            </button>
            <span className="flex-1" />
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[12px] text-muted transition-colors hover:bg-danger/10 hover:text-danger"
              onClick={() => onResolve(item.id, "deny")}
            >
              {copy.transcript.reject}
              <kbd aria-hidden>Esc</kbd>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function CopyAction({ text }: { text: string }) {
  const copy = useCopy();
  const [done, setDone] = useState(false);
  if (!text) return null;
  return (
    <button
      type="button"
      className="inline-flex h-7 items-center gap-1 rounded-md px-1.5 text-[11px] text-muted hover:bg-lift hover:text-foreground"
      aria-label={copy.transcript.copy}
      data-copy-text={text}
      onClick={async (e) => {
        e.stopPropagation();
        const ok = await writeClipboard(text);
        if (ok) {
          setDone(true);
          window.setTimeout(() => setDone(false), 1400);
        } else {
          toast.error(copy.transcript.copyFailed);
        }
      }}
    >
      {done ? <Check className="size-3.5" aria-hidden /> : <Copy className="size-3.5" aria-hidden />}
      <span>{done ? copy.transcript.copied : copy.transcript.copy}</span>
    </button>
  );
}

function ErrorCard({ item, onRetry }: { item: Item; onRetry?: () => void }) {
  const copy = useCopy();
  const classified = classifyItem(item);
  const labels = errorCopy(copy.transcript, classified.kind);
  const hint = classified.payloadHint || labels.hint;
  const detail = classified.detail;
  const showDetail = !!detail && detail !== labels.title && detail !== classified.payloadTitle;
  const soft = classified.kind === "canceled" || classified.kind === "max_turns";
  const detailsLabel = soft ? copy.transcript.details : copy.transcript.errorDetails;
  return (
    <div
      className={cn(
        "surface-inset relative overflow-hidden rounded-2xl border px-4 py-3.5",
        soft ? "border-border bg-card/80" : "border-danger/25 bg-danger/8",
      )}
      role="alert"
    >
      <span
        className={cn("pointer-events-none absolute inset-y-3 left-0 w-[2px] rounded-full", soft ? "bg-muted/60" : "bg-danger/80")}
        aria-hidden
      />
      <div className={cn("text-[13.5px] font-medium tracking-[-0.01em]", soft ? "text-foreground" : "text-danger")}>{labels.title}</div>
      {hint ? <p className={cn("mt-1 text-[13px] leading-5", soft ? "text-muted" : "text-danger/80")}>{hint}</p> : null}
      {showDetail ? (
        <details className="mt-2 text-[12px] text-muted">
          <summary className={cn("cursor-pointer select-none", soft ? "text-muted hover:text-foreground" : "text-danger/70")}>{detailsLabel}</summary>
          <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[11px] leading-[1.5]">{detail}</pre>
        </details>
      ) : null}
      {classified.retryable && onRetry ? (
        <button
          type="button"
          className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background transition-opacity hover:opacity-90"
          onClick={onRetry}
        >
          <RotateCcw className="size-3" aria-hidden />
          {copy.transcript.retry}
        </button>
      ) : null}
      {item.payload?.crash_path ? (
        <p className="mt-2 font-mono text-[11px] text-muted">
          {copy.settings.crashFile}: {String(item.payload.crash_path)}
        </p>
      ) : null}
    </div>
  );
}

const ItemRow = memo(function ItemRow({
  item,
  streaming,
  copyText,
  onRetry,
}: {
  item: Item;
  streaming?: boolean;
  copyText?: string;
  onRetry?: () => void;
}) {
  const copy = useCopy();
  if (item.type === "user") {
    if (item.source === "steer") {
      return (
        <div className="text-center text-[11px] text-muted">
          {copy.transcript.steer}: {item.text.replace(/^User steering \(apply now\):\s*/i, "")}
        </div>
      );
    }
    return (
      <div className="flex justify-end">
        <div className="group/msg w-fit max-w-[80%]">
          <div className="whitespace-pre-wrap break-words rounded-[18px] bg-lift px-3.5 py-[9px] text-[13px] leading-[1.5] tracking-[-0.011em]">
            {item.text}
          </div>
          {copyText ? (
            <div className="flex h-8 items-center justify-end opacity-0 transition-opacity duration-150 pointer-events-none group-hover/msg:pointer-events-auto group-hover/msg:opacity-100 group-focus-within/msg:pointer-events-auto group-focus-within/msg:opacity-100">
              <CopyAction text={copyText} />
            </div>
          ) : null}
        </div>
      </div>
    );
  }
  if (item.type === "assistant") {
    return (
      <div className="assistant-prose w-full min-w-0">
        <div className={cn(streaming && "assistant-live")}>
          <Markdown text={item.text} streaming={streaming} />
        </div>
      </div>
    );
  }
  if (item.type === "error") {
    return <ErrorCard item={item} onRetry={onRetry} />;
  }
  if (item.type === "compaction") {
    const kind = String(item.payload?.kind || "");
    if (kind === "checkpoint_start") return null;
    const note = String(item.payload?.note || item.text || "");
    const label = note || (kind === "checkpoint" ? copy.transcript.checkpoint : copy.app.compacted);
    return <div className="text-center text-[11px] text-muted/80">{label}</div>;
  }
  if (item.type === "tool_call" || item.type === "tool_result") {
    return <ToolLine item={item} />;
  }
  if (item.type === "turn_end" || item.type === "system") return null;
  if (item.type === "plan" || item.type === "file_change" || item.type === "ask_user") return null;
  if (item.type === "approval") return null;
  return null;
});

function ArtifactTimeline({
  items,
  running,
  workspace,
  onOpenReview,
}: {
  items: Item[];
  running: boolean;
  workspace?: string;
  onOpenReview?: (path?: string) => void;
}) {
  const pairs = pairTools(items).filter(pairShowsArtifact);
  if (!pairs.length) return null;
  return (
    <div className="u-chrome min-w-0 space-y-1" data-testid="tool-timeline">
      {pairs.map((p) => (
        <ArtifactCard
          key={p.key}
          call={p.call}
          result={p.result}
          running={running}
          workspace={workspace}
          onOpenReview={onOpenReview}
        />
      ))}
    </div>
  );
}

function ArtifactCard({
  call,
  result,
  running,
  workspace,
  onOpenReview,
}: {
  call?: Item;
  result?: Item;
  running: boolean;
  workspace?: string;
  onOpenReview?: (path?: string) => void;
}) {
  const copy = useCopy();
  const [source, setSource] = useState(false);
  const item = result || call!;
  const name = toolName(item);
  const view = artifactView(call, result);
  const [open, setOpen] = useState(() => artifactPreviewOpen(view));
  const openCall = !!call && !result;
  const pending = openCall && running;
  const interrupted = openCall && !running;
  const detail = toolDetail(result || call || item);
  const body = result ? formatToolBody(result) : formatToolBody(call || item);
  const files = name === "apply_patch" ? patchFileCount(body) : 0;
  const args = toolArgs(call || result || item);
  const path = view.path || String(args.path || args.file || "");
  const mcpHtml = result ? extractHTML(body) : "";
  const pdf = looksLikePDF(path);
  const office = name.startsWith("office_");
  const htmlish = view.kind === "html" || looksLikeHTMLFile(path) || looksLikeHTML(view.html) || looksLikeHTML(mcpHtml);
  const hasPreview = !office && !pdf && (artifactShouldShow(view) || !!mcpHtml);
  if (!office && !pdf && name !== "cite_sources" && !hasPreview) return null;
  const title = view.path
    ? view.path.replace(/\\/g, "/").split("/").pop() || view.path
    : name === "apply_patch"
      ? copy.transcript.patch
      : name === "cite_sources"
        ? copy.transcript.citations
        : htmlish
          ? copy.transcript.mcpApp
          : copy.transcript.artifact;
  const meta = name === "apply_patch" && files > 0
    ? copy.transcript.filesCount.replace("{n}", String(files))
    : (path || detail || name);
  return (
    <div
      className={cn(
        "surface-inset min-w-0 overflow-hidden rounded-xl border bg-card/60 px-3.5 py-2.5 transition-colors duration-200",
        pending ? "border-foreground/15" : "border-border/80",
      )}
      data-testid="artifact-card"
    >
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="grid size-7 shrink-0 place-items-center rounded-lg bg-lift/70 text-muted" aria-hidden>
          {pending ? <span className="pulse-dot" /> : interrupted ? <CircleDashed className="size-3.5 text-warning" /> : <FileText className="size-3.5" />}
        </span>
        <div className="min-w-0 flex-1 basis-40">
          <div className="flex items-center gap-2 text-[12.5px] font-medium text-foreground">
            <span className={cn("truncate", pending && "shimmer-text")}>{title}</span>
            {interrupted ? (
              <span className="rounded-md bg-warning/12 px-1.5 py-px text-[10.5px] font-medium text-warning">{copy.transcript.toolInterrupted}</span>
            ) : null}
          </div>
          <div className="truncate font-mono text-[11px] text-muted">{meta}</div>
        </div>
        <div className="ml-auto flex shrink-0 flex-wrap justify-end gap-1">
          {hasPreview ? (
            <button
              type="button"
              className="rounded-full border border-border px-2.5 py-1 text-[11px] text-foreground transition-colors hover:bg-lift"
              aria-expanded={open}
              onClick={() => setOpen((v) => !v)}
            >
              {open ? copy.transcript.hidePreview : copy.transcript.preview}
            </button>
          ) : null}
          {htmlish && open ? (
            <button
              type="button"
              className="rounded-full border border-border px-2.5 py-1 text-[11px] text-foreground transition-colors hover:bg-lift"
              onClick={() => setSource((v) => !v)}
            >
              {source ? copy.transcript.preview : copy.transcript.source}
            </button>
          ) : null}
          {onOpenReview ? (
            <button
              type="button"
              className="rounded-full border border-border px-2.5 py-1 text-[11px] text-foreground transition-colors hover:bg-lift"
              onClick={() => onOpenReview(path || undefined)}
            >
              {copy.review.openReview}
            </button>
          ) : null}
        </div>
      </div>
      {open ? (
        <>
          {mcpHtml && looksLikeHTML(mcpHtml) && !view.diff && !view.code && !view.html ? (
            <SandboxedFrame html={mcpHtml} title={title} />
          ) : (
            <ArtifactBody view={view} workspace={workspace} source={source} />
          )}
        </>
      ) : null}
    </div>
  );
}
