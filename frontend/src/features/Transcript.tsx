import { memo, useState, type ReactNode } from "react";
import { Check, Copy, FileText, Loader2, RotateCcw, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { StickToBottom, useStickToBottomContext } from "use-stick-to-bottom";
import { Markdown } from "../lib/markdown";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { writeClipboard } from "../lib/clipboard";
import { LONG_THREAD_TURNS, THREAD_COL, THREAD_GUTTER, THREAD_GUTTER_COMPACT } from "../lib/thread";
import {
  formatToolBody,
  isArtifactTool,
  patchFileCount,
  toolDetail,
  toolName,
} from "../lib/tool-summary";
import type { Approval, Item } from "../lib/protocol";
import { classifyItem, errorCopy } from "../lib/error";
import { layoutRows, pairTools, type AgentPart, type LayoutRow } from "../lib/transcript-layout";
import { ProcessGroup, ToolLine, WorkingLine } from "./transcript/ProcessGroup";

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

function toolsPending(items: Item[]): boolean {
  const start = lastRealUserIndex(items);
  const from = start < 0 ? 0 : start + 1;
  const open = new Set<string>();
  for (let i = from; i < items.length; i++) {
    const it = items[i];
    const id = String(it.payload?.id || it.key);
    if (it.type === "tool_call") open.add(id);
    if (it.type === "tool_result") open.delete(id);
  }
  return open.size > 0;
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
  if (isUserRow(row)) return "border-t border-border/40 pt-8";
  if (row.kind === "agent" && prev.kind === "user") return "pt-3";
  if (row.kind === "agent") return "pt-5";
  return "pt-4";
}

function partSpace(parts: AgentPart[], i: number): string {
  if (i === 0) return "";
  const prev = parts[i - 1];
  const cur = parts[i];
  if (cur.kind === "process") return "pt-1.5";
  if (cur.kind === "artifact") return "pt-2";
  if (cur.kind === "item" && cur.item.type === "assistant") {
    if (prev.kind === "process") return "pt-1.5";
    if (prev.kind === "artifact") return "pt-2.5";
    return "pt-1";
  }
  return "pt-2";
}

export function Transcript(props: {
  items: Item[];
  approvals: Approval[];
  running: boolean;
  compact?: boolean;
  flush?: boolean;
  onResolve: (id: string, decision: string) => void;
  onPrompt?: (text: string) => void;
  onOpenReview?: () => void;
  onRetry?: () => void;
}) {
  const copy = useCopy();
  const pad = props.flush ? "" : props.compact ? THREAD_GUTTER_COMPACT : THREAD_GUTTER;
  const col = props.flush ? "w-full min-w-0" : cn(THREAD_COL, pad);
  const layout = layoutRows(props.items);
  const empty = props.items.length === 0 && !props.running && props.approvals.length === 0;
  const live = lastMeaningful(props.items);
  const streamingKey = props.running && live?.type === "assistant" && live.delta ? live.key : "";
  const showWorking = props.running && !streamingKey && !toolsPending(props.items);
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
                    part.kind === "item" && part.item.type === "assistant" && "assistant-block",
                    (part.kind === "process" || part.kind === "artifact") && "u-chrome",
                  )}
                >
                  {part.kind === "process" ? (
                    <ProcessGroup items={part.items} live={part.live && props.running && last} compact={props.compact} />
                  ) : part.kind === "artifact" ? (
                    <ArtifactTimeline items={part.items} onOpenReview={props.onOpenReview} />
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
                  <WorkingLine />
                </div>
              ) : null}
              {row.copyText && !props.compact ? (
                <div className="u-chrome">
                  <TurnActions
                    text={row.copyText}
                    live={liveHere}
                    pinned={last && !liveHere && !workingHere}
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
      node: <WorkingLine />,
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
          className={cn(col, "flex min-h-full flex-col justify-center pb-10 pt-8")}
        >
          <h1 className="text-[26px] font-semibold tracking-[-0.038em] text-foreground">
            {copy.transcript.ready}
          </h1>
          <p className="mt-2 max-w-[36rem] text-[14.5px] leading-[1.6] text-muted">
            {copy.transcript.readyBody}
          </p>
          <div className="mt-6 flex flex-wrap gap-2">
            {copy.transcript.starters.map((s) => (
              <button
                type="button"
                key={s.label}
                className="rounded-full border border-border/80 bg-transparent px-3.5 py-1.5 text-[12px] text-muted transition-colors hover:border-border hover:bg-lift/50 hover:text-foreground"
                onClick={() => props.onPrompt?.(s.text)}
              >
                {s.label}
              </button>
            ))}
          </div>
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
      className="absolute bottom-3 left-1/2 z-10 -translate-x-1/2 rounded-full border border-border bg-card/95 px-3 py-1.5 text-[12px] text-foreground shadow-[var(--shadow-popover)] backdrop-blur-sm"
      onClick={() => {
        void ctx.scrollToBottom();
      }}
    >
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
  onOpenReview?: () => void;
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
          onClick={onOpenReview}
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
  return (
    <div className="rounded-2xl border border-border bg-card/90 px-4 py-3.5" role="status">
      <div className="flex items-center gap-2 text-[12px] font-medium text-foreground">
        <ShieldAlert className="size-3.5 text-accent" aria-hidden />
        {copy.transcript.needsApproval}
      </div>
      <div className="mt-1.5 text-[13.5px] font-medium tracking-[-0.015em]">{item.action || "action"}</div>
      <div className="mt-0.5 truncate font-mono text-[11px] text-muted">{item.command || item.path || item.level}</div>
      <div className="mt-3 flex flex-wrap gap-1.5">
        <button
          type="button"
          className="rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background"
          onClick={() => onResolve(item.id, "once")}
        >
          {copy.transcript.allow}
        </button>
        <button
          type="button"
          className="rounded-full border border-border px-3 py-1.5 text-[12px] text-foreground hover:bg-lift"
          onClick={() => onResolve(item.id, "always")}
        >
          {copy.transcript.allowAlways}
        </button>
        <button
          type="button"
          className="rounded-full px-3 py-1.5 text-[12px] text-muted hover:bg-lift hover:text-foreground"
          onClick={() => onResolve(item.id, "session")}
        >
          {copy.transcript.allowSession}
        </button>
        <button
          type="button"
          className="rounded-full px-3 py-1.5 text-[12px] text-muted hover:bg-danger/10 hover:text-danger"
          onClick={() => onResolve(item.id, "deny")}
        >
          {copy.transcript.reject}
        </button>
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
  return (
    <div className="rounded-2xl border border-danger/25 bg-danger/8 px-4 py-3" role="alert">
      <div className="text-[13.5px] font-medium text-danger">{labels.title}</div>
      {hint ? <p className="mt-1 text-[13px] leading-5 text-danger/80">{hint}</p> : null}
      {showDetail ? (
        <details className="mt-2 text-[12px] text-muted">
          <summary className="cursor-pointer text-danger/70">{copy.transcript.errorDetails}</summary>
          <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap font-mono text-[11px]">{detail}</pre>
        </details>
      ) : null}
      {classified.retryable && onRetry ? (
        <button
          type="button"
          className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background"
          onClick={onRetry}
        >
          <RotateCcw className="size-3" aria-hidden />
          {copy.transcript.retry}
        </button>
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
        <div className="group/msg max-w-[min(85%,36rem)]">
          <div className="whitespace-pre-wrap break-words rounded-[18px] bg-lift px-3.5 py-[9px] text-[14.5px] leading-[1.55] tracking-[-0.012em]">
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
      <div className="assistant-prose w-full min-w-0 text-[15.5px] leading-[1.7] tracking-[-0.011em] text-foreground/95">
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
    const label = kind === "checkpoint"
      ? copy.transcript.checkpoint
      : (item.payload.note || item.text || copy.app.compacted);
    return <div className="text-center text-[11px] text-muted/80">{String(label)}</div>;
  }
  if (item.type === "tool_call" || item.type === "tool_result") {
    return <ToolLine item={item} />;
  }
  if (item.type === "turn_end" || item.type === "system") return null;
  if (item.type === "plan" || item.type === "file_change" || item.type === "ask_user") return null;
  if (item.type === "approval") return null;
  return null;
});

function ArtifactTimeline({ items, onOpenReview }: { items: Item[]; onOpenReview?: () => void }) {
  const pairs = pairTools(items);
  return (
    <div className="u-chrome min-w-0 space-y-0.5" data-testid="tool-timeline">
      {pairs.map((p) => {
        const name = toolName(p.call || p.result || p.extra[0] || ({ name: "tool", payload: {} } as Item));
        if (name === "update_plan" && (p.call || p.result)) {
          return <PlanBlock key={p.key} item={p.result || p.call!} />;
        }
        if (isArtifactTool(name)) {
          return (
            <ArtifactCard
              key={p.key}
              call={p.call}
              result={p.result}
              onOpenReview={onOpenReview}
            />
          );
        }
        return null;
      })}
    </div>
  );
}

function PlanBlock({ item }: { item: Item }) {
  const copy = useCopy();
  const body = formatToolBody(item);
  return (
    <div className="rounded-xl border border-border/70 bg-lift/30 px-3 py-2">
      <div className="text-[11px] font-medium uppercase tracking-[0.04em] text-muted">{copy.transcript.plan}</div>
      <pre className="mt-1 whitespace-pre-wrap font-sans text-[13px] leading-5 text-foreground/90">{body.slice(0, 4000)}</pre>
    </div>
  );
}

function ArtifactCard({
  call,
  result,
  onOpenReview,
}: {
  call?: Item;
  result?: Item;
  onOpenReview?: () => void;
}) {
  const copy = useCopy();
  const item = result || call!;
  const name = toolName(item);
  const pending = !!call && !result;
  const detail = toolDetail(result || call || item);
  const body = result ? formatToolBody(result) : formatToolBody(call || item);
  const files = name === "apply_patch" ? patchFileCount(body) : 0;
  const title = name === "apply_patch"
    ? copy.transcript.patch
    : name === "cite_sources"
      ? copy.transcript.citations
      : copy.transcript.artifact;
  const meta = name === "apply_patch" && files > 0
    ? copy.transcript.filesCount.replace("{n}", String(files))
    : (detail || name);
  return (
    <div
      className="flex items-center gap-3 rounded-xl border border-border/80 bg-lift/25 px-3 py-2.5"
      data-testid="artifact-card"
    >
      <FileText className="size-4 shrink-0 text-accent" aria-hidden />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2 text-[12.5px] font-medium text-foreground">
          {pending ? <Loader2 className="size-3 animate-spin text-accent" aria-hidden /> : null}
          <span>{title}</span>
        </div>
        <div className="truncate font-mono text-[11px] text-muted">{meta}</div>
      </div>
      {onOpenReview ? (
        <button
          type="button"
          className="shrink-0 rounded-full border border-border px-2.5 py-1 text-[11px] text-foreground hover:bg-lift"
          onClick={onOpenReview}
        >
          {copy.review.openReview}
        </button>
      ) : null}
    </div>
  );
}
