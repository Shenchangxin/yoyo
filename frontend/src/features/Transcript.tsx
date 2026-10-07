import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowDown, ArrowUpRight, BookOpen, Boxes, Check, CircleDashed, Copy, FileText, Folder, GitFork, RotateCcw, ShieldAlert } from "lucide-react";
import { IconSwap } from "../components/ui/icon-swap";
import { Tooltip } from "../components/ui/tooltip";
import { toast } from "sonner";
import { Markdown } from "../lib/markdown";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import { writeClipboard } from "../lib/clipboard";
import { THREAD_COL, THREAD_GUTTER, THREAD_GUTTER_COMPACT } from "../lib/thread";
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
import { structureSig, withLiveText } from "../lib/stream-live";
import { ProcessGroup, ToolLine, WorkingLine } from "./transcript/ProcessGroup";
import { SubagentCard } from "./transcript/SubagentCard";
import { ArtifactBody } from "./transcript/FilePreview";
import { SandboxedFrame } from "./transcript/SandboxedFrame";
import { PresenceStamp } from "./presence";
import { MarkWell } from "./shell/YoyoMark";
import { displayWorkspace } from "../lib/display-title";
import { peelCompletedMentions, type MentionKind, type MentionPin } from "../lib/mentions";
import { itemTurnKey, pickActiveTurnKey, rowMatchesJump } from "../lib/turn-outline";

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
  if (part.kind === "subagent") return part.live;
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
  if (cur.kind === "subagent") return "pt-1.5";
  if (cur.kind === "artifact") return "pt-2";
  if (cur.kind === "item" && cur.item.type === "assistant") {
    if (prev.kind === "process" || prev.kind === "artifact" || prev.kind === "subagent") return "pt-3";
    return "pt-2";
  }
  return "pt-2";
}

export function Transcript(props: {
  items: Item[];
  liveTexts?: Record<string, string>;
  showThinking?: boolean;
  approvals: Approval[];
  running: boolean;
  compact?: boolean;
  flush?: boolean;
  workspace?: string;
  older?: boolean;
  loadingOlder?: boolean;
  idle?: boolean;
  onLoadOlder?: () => void | Promise<void>;
  onResolve: (id: string, decision: string, answer?: string) => void;
  onPrompt?: (text: string) => void;
  onOpenReview?: (path?: string) => void;
  onRetry?: () => void;
  jumpTo?: { key: string; aliases?: string[]; nonce: number } | null;
  latestNonce?: number;
  onActiveTurn?: (key: string) => void;
  onJumpLatest?: () => void | Promise<void>;
  onRestoreFiles?: (from: string) => void | Promise<void>;
  onForkFrom?: (from: string) => void | Promise<void>;
}) {
  const pad = props.flush ? "" : props.compact ? THREAD_GUTTER_COMPACT : THREAD_GUTTER;
  const col = props.flush ? "w-full min-w-0" : cn(THREAD_COL, pad);
  const visible = useMemo(
    () => (props.showThinking ? props.items : props.items.filter((it) => it.type !== "reasoning")),
    [props.items, props.showThinking],
  );
  const layout = useStableLayout(visible);
  const empty = visible.length === 0 && !props.running && props.approvals.length === 0;
  const live = lastMeaningful(visible);
  const liveItem = live ? withLiveText(live, props.liveTexts) : null;
  const streamingKey = props.running && liveItem?.type === "assistant" && liveItem.delta ? liveItem.key : "";
  const liveLen = streamingKey ? (props.liveTexts?.[streamingKey]?.length || 0) : 0;
  const showWorking = props.running && !streamingKey && !tailOwnsActivity(layout);
  const since = props.running ? lastUserTimeMs(visible) : 0;
  const retryKey = lastErrorKey(visible);
  const pinnedTurn = lastAgentKey(layout);

  const rows: TranscriptRow[] = [
    ...layout.map((row, i) => {
      if (row.kind === "user") {
        return {
          key: row.key,
          kind: "user" as const,
          turnKey: row.turnKey || itemTurnKey(row.item),
          space: rowSpace(layout, i),
          estimate: 96,
          render: () => (
            <div className="group/user relative">
              <ItemRow
                item={row.item}
                copyText={props.compact ? "" : row.item.text}
                from={props.compact ? "" : (row.item.seq ? `seq:${row.item.seq}` : itemTurnKey(row.item))}
                onRestore={props.compact ? undefined : props.onRestoreFiles}
                onFork={props.compact ? undefined : props.onForkFrom}
              />
            </div>
          ),
        };
      }
      if (row.kind === "agent") {
        const liveHere = row.parts.some((p) => p.kind === "item" && p.item.key === streamingKey);
        const last = row.key === pinnedTurn;
        const workingHere = last && showWorking;
        const tail = row.parts.length - 1;
        return {
          key: row.key,
          kind: "agent" as const,
          turnKey: row.turnKey,
          space: rowSpace(layout, i),
          estimate: agentEstimate(row.copyText),
          render: () => (
            <article className="assistant-letter group/turn flex gap-2.5" data-testid="agent-turn">
              <PresenceStamp size={22} className="mt-0.5" />
              <div className="min-w-0 flex-1">
              {row.parts.map((part, pi) => (
                <div
                  key={part.key}
                  className={cn(
                    partSpace(row.parts, pi),
                    part.kind === "item" && part.item.type === "assistant" && "assistant-block min-w-0",
                    (part.kind === "process" || part.kind === "artifact" || part.kind === "subagent") && "u-chrome",
                  )}
                >
                  {part.kind === "process" ? (
                    <ProcessGroup
                      items={part.items}
                      live={props.running && last && pi === tail}
                      running={props.running}
                      compact={props.compact}
                      liveTexts={props.liveTexts}
                    />
                  ) : part.kind === "subagent" ? (
                    <SubagentCard
                      items={part.items}
                      child={part.child}
                      live={props.running && part.live}
                      running={props.running}
                      compact={props.compact}
                    />
                  ) : part.kind === "artifact" ? (
                    <ArtifactTimeline items={part.items} running={props.running} workspace={props.workspace} onOpenReview={props.onOpenReview} />
                  ) : (
                    <ItemRow
                      item={part.item}
                      streaming={part.item.key === streamingKey}
                      liveText={props.liveTexts?.[part.item.key]}
                      rich={part.item.type === "assistant" && part.item.key !== streamingKey}
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
              </div>
            </article>
          ),
        };
      }
      return {
        key: row.key,
        kind: "solo" as const,
        space: rowSpace(layout, i),
        estimate: 96,
        render: () => (
          <ItemRow
            item={row.item}
            onRetry={row.item.key === retryKey ? props.onRetry : undefined}
          />
        ),
      };
    }),
    ...props.approvals.map((a) => ({
      key: `ask:${a.id}`,
      kind: "ask" as const,
      space: "pt-4",
      estimate: 160,
      render: () => <ApprovalCard item={a} onResolve={props.onResolve} />,
    })),
  ];
  if (showWorking && !pinnedTurn) {
    rows.push({
      key: "working",
      kind: "working",
      space: "pt-3",
      estimate: 40,
      render: () => <WorkingLine since={since} />,
    });
  }

  return (
    <div
      className="prose-select relative flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      data-testid="conversation-column"
    >
      {empty && !props.compact ? (
        <div className={cn(col, "transcript-scroll flex min-h-full flex-col justify-end overflow-y-auto pb-8 pt-10")}>
          <EmptyTurn workspace={props.workspace} onPrompt={props.onPrompt} />
        </div>
      ) : (
        <TranscriptLane
          className={cn(col, "min-h-0 flex-1 pb-4 pt-5")}
          rows={rows}
          older={!!props.older}
          loadingOlder={!!props.loadingOlder}
          overscan={4}
          onLoadOlder={props.onLoadOlder}
          follow={props.running}
          followNonce={liveLen}
          jumpTo={props.jumpTo}
          latestNonce={props.latestNonce}
          onActiveTurn={props.onActiveTurn}
          onJumpLatest={props.onJumpLatest}
        />
      )}
    </div>
  );
}

function useStableLayout(items: Item[]): LayoutRow[] {
  const sig = structureSig(items);
  const ref = useRef({ sig: "", rows: [] as LayoutRow[] });
  if (ref.current.sig !== sig) {
    ref.current = { sig, rows: layoutRows(items) };
  }
  return ref.current.rows;
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
      <h1 className="mt-6 max-w-[16ch] text-[26px] font-semibold leading-[1.15] tracking-[-0.042em] text-pretty text-foreground">
        {ready}
      </h1>
      <p className="mt-3 flex max-w-[46ch] flex-wrap items-center gap-x-3 gap-y-1.5 text-[13px] leading-[1.6] text-muted">
        {hints.map((h, i) => (
          <span key={h.key} className="inline-flex items-center">
            {i > 0 ? <span className="mr-3 text-muted/35" aria-hidden>·</span> : null}
            {h.node}
          </span>
        ))}
      </p>
      <div className="mt-7 flex flex-wrap gap-2">
        {starters.map((s, i) => (
          <button
            type="button"
            key={s.label}
            className="inline-flex cursor-pointer items-center gap-1.5 rounded-full border border-border/80 bg-transparent py-1.5 pl-3 pr-2.5 text-[13px] text-muted transition-[background-color,color,transform,border-color] duration-200 ease-[var(--ease-out)] hover:border-border hover:bg-lift hover:text-foreground active:scale-[0.98]"
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

type TranscriptRow = {
  key: string;
  kind?: "user" | "agent" | "solo" | "ask" | "working";
  turnKey?: string;
  space?: string;
  estimate: number;
  render: () => ReactNode;
};

/** Live / current pack stays in document flow so streaming markdown is not virtualized. */
function lastPackStart(rows: TranscriptRow[]): number {
  for (let i = rows.length - 1; i >= 0; i--) {
    if (rows[i].kind === "user") return i;
  }
  for (let i = rows.length - 1; i >= 0; i--) {
    if (rows[i].kind === "agent") return i;
  }
  for (let i = 0; i < rows.length; i++) {
    if (rows[i].kind === "ask" || rows[i].kind === "working") return i;
  }
  return rows.length;
}

function cssEscape(s: string): string {
  if (typeof CSS !== "undefined" && typeof CSS.escape === "function") return CSS.escape(s);
  return s.replace(/["\\]/g, "\\$&");
}

function agentEstimate(copyText: string): number {
  return Math.max(520, Math.min(2400, 160 + Math.round((copyText?.length || 0) * 0.48)));
}

function TranscriptLane({
  className,
  rows,
  older,
  loadingOlder,
  overscan,
  onLoadOlder,
  follow,
  followNonce,
  jumpTo,
  latestNonce,
  onActiveTurn,
  onJumpLatest,
}: {
  className?: string;
  rows: TranscriptRow[];
  older: boolean;
  loadingOlder: boolean;
  overscan: number;
  onLoadOlder?: () => void | Promise<void>;
  follow: boolean;
  followNonce?: number;
  jumpTo?: { key: string; aliases?: string[]; nonce: number } | null;
  latestNonce?: number;
  onActiveTurn?: (key: string) => void;
  onJumpLatest?: () => void | Promise<void>;
}) {
  const copy = useCopy();
  const scrollRef = useRef<HTMLDivElement>(null);
  const stick = useRef(true);
  const [escaped, setEscaped] = useState(false);
  const loadingRef = useRef(false);
  const appliedJump = useRef(0);
  const appliedLatest = useRef(0);
  const lastActive = useRef('');
  const spyRaf = useRef(0);
  const jumpHeld = useRef(false);
  const jumpNonce = jumpTo?.nonce || 0;
  if (jumpNonce && jumpNonce !== appliedJump.current) {
    stick.current = false;
  }
  const packStart = lastPackStart(rows);
  const history = packStart > 0 ? rows.slice(0, packStart) : [];
  const pin = packStart >= 0 ? rows.slice(packStart) : rows;
  const virtualizer = useVirtualizer({
    count: history.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (i) => history[i]?.estimate ?? 96,
    overscan,
    getItemKey: (i) => history[i]?.key ?? i,
  });
  const followEnd = useCallback(() => {
    const el = scrollRef.current;
    if (!el || !stick.current) return;
    el.scrollTop = el.scrollHeight;
  }, []);
  const releaseJump = useCallback(() => {
    jumpHeld.current = false;
  }, []);
  useLayoutEffect(() => {
    followEnd();
  }, [followEnd, rows.length, follow, followNonce]);
  useLayoutEffect(() => {
    if (!latestNonce || latestNonce === appliedLatest.current) return;
    appliedLatest.current = latestNonce;
    jumpHeld.current = false;
    stick.current = true;
    setEscaped(false);
    followEnd();
  }, [latestNonce, followEnd]);
  useLayoutEffect(() => {
    const key = jumpTo?.key;
    const nonce = jumpTo?.nonce || 0;
    if (!key || nonce === appliedJump.current) return;
    const aliases = jumpTo?.aliases?.length ? jumpTo.aliases : [key];
    const idx = rows.findIndex((r) => rowMatchesJump(r.turnKey, aliases) || rowMatchesJump(r.key, aliases));
    stick.current = false;
    setEscaped(true);
    lastActive.current = key;
    if (idx < 0) return;
    appliedJump.current = nonce;
    jumpHeld.current = true;
    const align = () => {
      if (!jumpHeld.current) return;
      if (idx < history.length) {
        virtualizer.scrollToIndex(idx, { align: "start" });
        return;
      }
      const el = scrollRef.current?.querySelector(`[data-turn-key="${cssEscape(key)}"]`) as HTMLElement | null;
      el?.scrollIntoView({ block: "start" });
    };
    align();
    requestAnimationFrame(align);
  }, [jumpTo, rows.length, rows[0]?.key, rows[rows.length - 1]?.key, history.length]);
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    el.addEventListener('wheel', releaseJump, { passive: true });
    el.addEventListener('pointerdown', releaseJump);
    el.addEventListener('touchstart', releaseJump, { passive: true });
    return () => {
      el.removeEventListener('wheel', releaseJump);
      el.removeEventListener('pointerdown', releaseJump);
      el.removeEventListener('touchstart', releaseJump);
    };
  }, [releaseJump]);
  const reportActive = useCallback((key: string) => {
    if (!key || key === lastActive.current) return;
    lastActive.current = key;
    onActiveTurn?.(key);
  }, [onActiveTurn]);
  const spyTurns = useCallback(() => {
    const root = scrollRef.current;
    if (!root || !onActiveTurn) return;
    if (jumpHeld.current) {
      const k = jumpTo?.key;
      if (k) reportActive(k);
      return;
    }
    const gap = root.scrollHeight - root.scrollTop - root.clientHeight;
    if (gap < 72) {
      const last = [...rows].reverse().find((r) => r.turnKey)?.turnKey;
      if (last) reportActive(last);
      return;
    }
    const line = root.getBoundingClientRect().top + Math.min(64, root.clientHeight * 0.16);
    const nodes: { key: string; top: number }[] = [];
    root.querySelectorAll("[data-turn-key]").forEach((node) => {
      const k = (node as HTMLElement).dataset.turnKey || "";
      if (!k) return;
      nodes.push({ key: k, top: node.getBoundingClientRect().top });
    });
    const best = pickActiveTurnKey(nodes, line);
    if (best) reportActive(best);
  }, [rows, onActiveTurn, reportActive, jumpTo?.key]);
  const queueSpy = useCallback(() => {
    if (spyRaf.current) return;
    spyRaf.current = requestAnimationFrame(() => {
      spyRaf.current = 0;
      spyTurns();
    });
  }, [spyTurns]);
  useEffect(() => {
    queueSpy();
    return () => {
      if (spyRaf.current) cancelAnimationFrame(spyRaf.current);
      spyRaf.current = 0;
    };
  }, [queueSpy, rows.length]);
  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    const gap = el.scrollHeight - el.scrollTop - el.clientHeight;
    const atBottom = gap < 72;
    if (!jumpHeld.current) {
      stick.current = atBottom;
      setEscaped(!atBottom);
    }
    queueSpy();
    if (el.scrollTop < 48 && older && onLoadOlder && !loadingRef.current) {
      loadingRef.current = true;
      const prev = el.scrollHeight;
      Promise.resolve(onLoadOlder()).finally(() => {
        requestAnimationFrame(() => {
          const node = scrollRef.current;
          if (node) node.scrollTop += node.scrollHeight - prev;
          loadingRef.current = false;
        });
      });
    }
  };
  return (
    <>
      <div
        ref={scrollRef}
        className={cn('transcript-scroll min-h-0 flex-1 overflow-y-auto', className)}
        onScroll={onScroll}
      >
        {older ? (
          <button
            type="button"
            className="mb-3 w-full rounded-md py-1.5 text-center text-[12px] text-muted hover:bg-lift hover:text-foreground"
            disabled={loadingOlder}
            onClick={() => { void onLoadOlder?.(); }}
          >
            {copy.transcript.loadEarlier}
          </button>
        ) : null}
        {history.length ? (
          <div className="relative w-full shrink-0 overflow-clip bg-background" style={{ height: virtualizer.getTotalSize() }}>
            {virtualizer.getVirtualItems().map((v) => {
              const row = history[v.index];
              if (!row) return null;
              return (
                <div
                  key={row.key}
                  data-index={v.index}
                  {...(row.turnKey ? { "data-turn-key": row.turnKey } : {})}
                  ref={virtualizer.measureElement}
                  className={cn("bg-background", row.space)}
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    transform: `translateY(${v.start}px)`,
                  }}
                >
                  {row.render()}
                </div>
              );
            })}
          </div>
        ) : null}
        {pin.map((row) => (
          <div
            key={row.key}
            {...(row.turnKey ? { "data-turn-key": row.turnKey } : {})}
            className={cn("bg-background", row.space)}
          >
            {row.render()}
          </div>
        ))}
      </div>
      {escaped ? (
        <button
          type='button'
          className='absolute bottom-4 left-1/2 z-10 inline-flex -translate-x-1/2 cursor-pointer items-center gap-1.5 rounded-full border border-border bg-popover py-1.5 pl-2.5 pr-3 text-[12px] text-foreground shadow-[var(--shadow-popover)] transition-[background-color,transform] duration-200 ease-[var(--ease-out)] hover:bg-lift active:scale-[0.98]'
          onClick={() => {
            stick.current = true;
            setEscaped(false);
            if (onJumpLatest) void onJumpLatest();
            else followEnd();
          }}
        >
          <ArrowDown className='size-3 text-muted' aria-hidden />
          {copy.transcript.jumpLatest}
        </button>
      ) : null}
    </>
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
          className={turnActionBtn}
          onClick={() => onOpenReview()}
        >
          <FileText className="size-3.5" aria-hidden />
          {copy.review.openReview}
        </button>
      ) : null}
    </div>
  );
}

function ApprovalCard({ item, onResolve }: { item: Approval; onResolve: (id: string, decision: string, answer?: string) => void }) {
  const copy = useCopy();
  const ask = item.action === "ask_user";
  const [answer, setAnswer] = useState("");
  const subject = item.command || item.path || item.level;
  if (ask) {
    return (
      <div
        className="u-card-hover is-warn relative overflow-hidden rounded-2xl border border-border bg-card px-4 py-3.5 shadow-[var(--shadow-card)]"
        role="status"
        data-testid="ask-user-card"
      >
        <span className="pointer-events-none absolute inset-y-3 left-0 w-[2px] rounded-full bg-warning/80" aria-hidden />
        <div className="text-[11px] font-medium text-muted">{copy.transcript.askQuestion}</div>
        <div className="mt-0.5 text-[14px] font-medium tracking-tight text-foreground">{subject || copy.transcript.askQuestion}</div>
        <textarea
          className="mt-2 min-h-[4.5rem] w-full resize-y rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 text-[13px] leading-[1.5] text-foreground outline-none focus:border-foreground/30"
          value={answer}
          placeholder={copy.transcript.answerPlaceholder}
          autoComplete="off"
          spellCheck={false}
          onChange={(e) => setAnswer(e.target.value)}
        />
        <div className="mt-3 flex flex-wrap items-center gap-1.5">
          <button
            type="button"
            className="inline-flex items-center gap-1.5 rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background transition-[opacity,transform] duration-200 ease-[var(--ease-out)] hover:opacity-[0.92] active:scale-[0.98] disabled:opacity-40"
            disabled={!answer.trim()}
            onClick={() => onResolve(item.id, "once", answer.trim())}
          >
            {copy.transcript.sendAnswer}
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[12px] text-muted transition-colors hover:bg-danger/10 hover:text-danger"
            onClick={() => onResolve(item.id, "deny")}
          >
            {copy.transcript.reject}
          </button>
        </div>
      </div>
    );
  }
  return (
    <div
      className="u-card-hover is-warn relative overflow-hidden rounded-2xl border border-border bg-card px-4 py-3.5 shadow-[var(--shadow-card)]"
      role="status"
      data-testid="approval-card"
    >
      <span className="pointer-events-none absolute inset-y-3 left-0 w-[2px] rounded-full bg-warning/80" aria-hidden />
      <div className="flex items-start gap-3">
        <span className="mt-px grid size-7 shrink-0 place-items-center rounded-lg bg-warning/12 text-warning" aria-hidden>
          <ShieldAlert className="size-3.5" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 text-[11px] font-medium text-muted">
            {copy.transcript.needsApproval}
          </div>
          <div className="mt-0.5 text-[14px] font-medium tracking-tight text-foreground">{item.action || "action"}</div>
          {subject ? (
            <pre className="mt-2 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[12px] leading-[1.5] text-foreground/85">
              {subject}
            </pre>
          ) : null}
          <div className="mt-3 flex flex-wrap items-center gap-1.5">
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background transition-[opacity,transform] duration-200 ease-[var(--ease-out)] hover:opacity-[0.92] active:scale-[0.98] disabled:opacity-40"
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

const turnActionBtn =
  "inline-flex h-7 items-center gap-1 rounded-md px-1.5 text-[11px] text-muted hover:bg-lift hover:text-foreground disabled:opacity-40";

function CopyAction({ text }: { text: string }) {
  const copy = useCopy();
  const [done, setDone] = useState(false);
  if (!text) return null;
  return (
    <button
      type="button"
      className={turnActionBtn}
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
      <IconSwap
        on={done}
        className="size-3.5"
        off={<Copy className="size-3.5" aria-hidden />}
        live={<Check className="size-3.5" aria-hidden />}
      />
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
        "u-card-hover relative overflow-hidden rounded-2xl border px-4 py-3.5",
        soft ? "border-border bg-card/80" : "is-danger border-danger/25 bg-danger/8",
      )}
      role="alert"
    >
      <span
        className={cn("pointer-events-none absolute inset-y-3 left-0 w-[2px] rounded-full", soft ? "bg-muted/60" : "bg-danger/80")}
        aria-hidden
      />
      <div className={cn("text-[14px] font-medium tracking-[-0.01em]", soft ? "text-foreground" : "text-danger")}>{labels.title}</div>
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
          className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background transition-[opacity,transform] duration-200 ease-[var(--ease-out)] hover:opacity-[0.92] active:scale-[0.98] disabled:opacity-40"
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

function UserMsgActions({
  text,
  from,
  onRestore,
  onFork,
}: {
  text?: string;
  from?: string;
  onRestore?: (from: string) => void | Promise<void>;
  onFork?: (from: string) => void | Promise<void>;
}) {
  const copy = useCopy();
  const [busy, setBusy] = useState<"restore" | "fork" | "">("");
  const run = async (kind: "restore" | "fork", fn?: (from: string) => void | Promise<void>) => {
    if (!from || !fn || busy) return;
    setBusy(kind);
    try {
      await fn(from);
    } finally {
      setBusy("");
    }
  };
  if (!text && !onRestore && !onFork) return null;
  return (
    <div
      className="flex h-8 items-center justify-end gap-0.5 opacity-0 transition-opacity duration-150 pointer-events-none group-hover/msg:pointer-events-auto group-hover/msg:opacity-100 group-focus-within/msg:pointer-events-auto group-focus-within/msg:opacity-100"
      data-testid="user-turn-actions"
    >
      {text ? <CopyAction text={text} /> : null}
      {onRestore && from ? (
        <Tooltip content={copy.transcript.restoreFiles}>
          <button
            type="button"
            className={turnActionBtn}
            aria-label={copy.transcript.restoreFiles}
            data-testid="user-restore-files"
            disabled={!!busy}
            onClick={(e) => {
              e.stopPropagation();
              void run("restore", onRestore);
            }}
          >
            <RotateCcw className="size-3.5" aria-hidden />
            <span>{copy.transcript.restore}</span>
          </button>
        </Tooltip>
      ) : null}
      {onFork && from ? (
        <Tooltip content={copy.transcript.forkFrom}>
          <button
            type="button"
            className={turnActionBtn}
            aria-label={copy.transcript.forkFrom}
            data-testid="user-fork-from"
            disabled={!!busy}
            onClick={(e) => {
              e.stopPropagation();
              void run("fork", onFork);
            }}
          >
            <GitFork className="size-3.5" aria-hidden />
            <span>{copy.transcript.fork}</span>
          </button>
        </Tooltip>
      ) : null}
    </div>
  );
}

function CompactRow({ item }: { item: Item }) {
  const copy = useCopy();
  const p = item.payload || {};
  const note = String(p.note || item.text || "");
  const kept = Number(p.kept || 0);
  const elided = Number(p.elided || 0);
  const hydrated = Number(p.hydrated || 0);
  const bits = [
    copy.transcript.checkpoint,
    kept ? copy.transcript.checkpointKept.replace("{n}", String(kept)) : "",
    elided ? copy.transcript.checkpointElided.replace("{n}", String(elided)) : "",
    hydrated ? copy.transcript.checkpointHydrated.replace("{n}", String(hydrated)) : "",
  ].filter(Boolean);
  const label = note || bits.join(" · ") || copy.app.compacted;
  return (
    <div className="flex justify-center py-1" data-testid="compact-row">
      <div className="max-w-[36rem] rounded-full border border-border/50 bg-lift/40 px-3 py-1 text-center text-[11px] leading-[1.4] text-muted">
        {label}
      </div>
    </div>
  );
}

function MentionGlyph({ kind }: { kind: MentionKind }) {
  const cls = "size-3 shrink-0 opacity-80";
  if (kind === "folder") return <Folder className={cls} />;
  if (kind === "skill") return <BookOpen className={cls} />;
  if (kind === "harness") return <Boxes className={cls} />;
  return <FileText className={cls} />;
}

function UserPrompt({ text, parts }: { text: string; parts?: { type?: string; image_url?: string; mime?: string; text?: string }[] }) {
  const peeled = peelCompletedMentions(text);
  const chips: MentionPin[] = peeled.chips;
  const prose = peeled.text.trim();
  const images = (parts || []).filter((p) => p.image_url && (p.type === "image_url" || (p.mime || "").startsWith("image/")));
  return (
    <div className="user-bubble whitespace-pre-wrap break-words px-3.5 py-2.5 text-[14px] leading-[1.55] tracking-normal">
      {images.length ? (
        <div className={cn("flex flex-wrap gap-1.5", (chips.length || prose) && "mb-1.5")}>
          {images.map((p, i) => (
            <img
              key={`${p.image_url?.slice(-24)}-${i}`}
              src={p.image_url}
              alt={p.text || "attachment"}
              className="max-h-28 max-w-[9rem] rounded-lg object-cover"
            />
          ))}
        </div>
      ) : null}
      {chips.length ? (
        <div className={cn("flex flex-wrap items-center gap-1", prose && "mb-1.5")}>
          {chips.map((c, i) => (
            <span
              key={`${c.token}-${i}`}
              className="inline-flex max-w-[12rem] items-center gap-1 rounded-md bg-background/55 px-1.5 py-0.5 text-[11px]"
              title={c.detail || c.token}
            >
              <MentionGlyph kind={c.kind} />
              <span className="truncate">{c.label}</span>
            </span>
          ))}
        </div>
      ) : null}
      {prose || !chips.length ? prose || text : null}
    </div>
  );
}

const ItemRow = memo(function ItemRow({
  item,
  streaming,
  liveText,
  copyText,
  from,
  onRestore,
  onFork,
  rich,
  onRetry,
}: {
  item: Item;
  streaming?: boolean;
  liveText?: string;
  copyText?: string;
  from?: string;
  onRestore?: (from: string) => void | Promise<void>;
  onFork?: (from: string) => void | Promise<void>;
  rich?: boolean;
  onRetry?: () => void;
}) {
  const copy = useCopy();
  const shown = liveText != null ? { ...item, text: liveText } : item;
  if (shown.type === "user") {
    if (shown.source === "steer") {
      return (
        <div className="text-center text-[11px] text-muted">
          {copy.transcript.steer}: {shown.text.replace(/^User steering \(apply now\):\s*/i, "")}
        </div>
      );
    }
    return (
      <div className="flex justify-end">
        <div className="group/msg w-fit max-w-[80%]">
          <UserPrompt text={shown.text} parts={Array.isArray(shown.payload?.parts) ? shown.payload.parts : undefined} />
          {copyText || onRestore || onFork ? (
            <UserMsgActions text={copyText} from={from} onRestore={onRestore} onFork={onFork} />
          ) : null}
        </div>
      </div>
    );
  }
  if (shown.type === "assistant") {
    return (
      <div className="assistant-prose w-full min-w-0">
        <div className={cn(streaming && "assistant-live")}>
          <Markdown text={shown.text} streaming={streaming} rich={!!rich && !streaming} />
        </div>
      </div>
    );
  }
  if (shown.type === "error") {
    return <ErrorCard item={shown} onRetry={onRetry} />;
  }
  if (shown.type === "compaction") {
    const kind = String(shown.payload?.kind || "");
    if (kind === "checkpoint_start") return null;
    return <CompactRow item={shown} />;
  }
  if (shown.type === "tool_call" || shown.type === "tool_result") {
    return <ToolLine item={shown} />;
  }
  if (shown.type === "turn_end" || shown.type === "system") return null;
  if (shown.type === "plan" || shown.type === "file_change" || shown.type === "ask_user") return null;
  if (shown.type === "approval") return null;
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
  const copy = useCopy();
  const all = pairTools(items).filter(pairShowsArtifact);
  if (!all.length) return null;
  const cap = 12;
  const pairs = all.length > cap ? all.slice(-cap) : all;
  const hidden = all.length - pairs.length;
  return (
    <div className="u-chrome min-w-0 space-y-1" data-testid="tool-timeline">
      {hidden > 0 ? (
        <div className="px-1 py-0.5 text-[11px] text-muted">{copy.transcript.earlierSteps.replace("{n}", String(hidden))}</div>
      ) : null}
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
  const hasPreview = artifactShouldShow(view) || !!mcpHtml || ((office || pdf) && !!path && !!workspace);
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
        "u-card-hover min-w-0 overflow-hidden rounded-md border bg-sidebar/40 px-3.5 py-2.5",
        pending ? "border-foreground/15" : "border-border/80",
      )}
      data-testid="artifact-card"
    >
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="grid size-7 shrink-0 place-items-center rounded-lg bg-lift/70 text-muted" aria-hidden>
          {pending ? <span className="pulse-dot" /> : interrupted ? <CircleDashed className="size-3.5 text-warning" /> : <FileText className="size-3.5" />}
        </span>
        <div className="min-w-0 flex-1 basis-40">
          <div className="flex items-center gap-2 text-[13px] font-medium text-foreground">
            <span className={cn("truncate", pending && "shimmer-text")}>{title}</span>
            {interrupted ? (
              <span className="rounded-md bg-warning/12 px-1.5 py-px text-[11px] font-medium text-warning">{copy.transcript.toolInterrupted}</span>
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
            <ArtifactBody view={{ ...view, path: view.path || path, kind: view.kind || ((office || pdf) ? "text" : view.kind) }} workspace={workspace} source={source} />
          )}
        </>
      ) : null}
    </div>
  );
}
