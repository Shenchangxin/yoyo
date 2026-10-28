/**
 * 拥有：对话怎么排成行、虚拟滚动和卡片。
 * 不拥有：工具步骤的内部排版，那些在 ProcessGroup。只投影条目，复制走剪贴板，不调用 api。
 */

import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowDown, ArrowUpRight, BookOpen, Boxes, Check, CircleDashed, Copy, FileText, Folder, GitFork, MessageSquare, RotateCcw, ShieldAlert } from "lucide-react";
import { SendSling } from "../components/SendSling";
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
  isArtifactTool,
  patchFileCount,
  toolArgs,
  toolDetail,
  toolName,
} from "../lib/tool-summary";
import { artifactPreviewOpen, artifactShouldShow, artifactView, isCodeChangeTool } from "../lib/artifact-preview";
import { extractHTML, looksLikeHTML, looksLikeHTMLFile, looksLikeMCPUI, looksLikePDF } from "../lib/html-preview";
import type { Approval, Item } from "../lib/protocol";
import { isPageReviewItem, PageReviewCard } from "./pages/PageReviewCard";
import { isBrowserShotItem, BrowserShotCard } from "./pages/BrowserShotCard";
import { classifyItem, errorCopy } from "../lib/error";
import { ARTIFACT_PAGE, classifyToolPairs, flattenWorkProcess, hydrateLayout, layoutRows, pairTools, processGroupLive, tailProcessPairs, type AgentPart, type LayoutRow } from "../lib/transcript-layout";
import { structureSig } from "../lib/stream-live";
import { coerceCollapseTiming, coerceTranscriptView, presentationPolicyFor } from "../lib/presentation-policy";
import { ScrollFollow, scrollMetrics } from "../lib/scroll-follow";
import { ProcessGroup, ToolLine } from "./transcript/ProcessGroup";
import { SubagentCard } from "./transcript/SubagentCard";
import { ArtifactBody } from "./transcript/FilePreview";
import { SandboxedFrame } from "./transcript/SandboxedFrame";
import { PresenceStamp } from "./presence";
import { MarkWell } from "./shell/YoyoMark";
import { displayWorkspace } from "../lib/display-title";
import { formatMessageClock } from "../lib/session-stats";
import { mergeMentionPins, peelCompletedMentions, pinsFromAttachments, type MentionKind, type MentionPin } from "../lib/mentions";
import { itemTurnKey, pickActiveTurnKey, rowMatchesJump } from "../lib/turn-outline";
import { studioNextPrompt, type StudioNextKey } from "./video/studio-next";
import { useStudioSelection } from "./video/workshop-store";

/** 从最后一行往前找，返回最近一条流式片段的 key。没有就返回空字符串。rows 读取 length，按下标读取。 */
function lastStreamKey(rows: LayoutRow[]): string {
  for (let i = rows.length - 1; i >= 0; i--) {
    const row = rows[i];
    if (row.kind !== "agent") continue;
    const key = streamKeyInParts(row.parts);
    if (key) return key;
  }
  return "";
}

/** 在这一层和嵌套的 work 里，从后往前找 stream 的 key。parts 读取 length，按下标读取。 */
function streamKeyInParts(parts: AgentPart[]): string {
  for (let j = parts.length - 1; j >= 0; j--) {
    const part = parts[j];
    if (part.kind === "stream") return part.item.key;
    if (part.kind === "work") {
      const inner = streamKeyInParts(part.parts);
      if (inner) return inner;
    }
  }
  return "";
}

/** 返回最后一块过程所在的下标。嵌套在 work 里时返回这块 work 的下标。没有返回 -1。parts 读取 length，按下标读取。 */
function lastProcessIndex(parts: AgentPart[]): number {
  for (let i = parts.length - 1; i >= 0; i--) {
    const part = parts[i];
    if (part.kind === "process") return i;
    if (part.kind === "work") {
      const inner = lastProcessIndex(part.parts);
      if (inner >= 0) return i;
    }
  }
  return -1;
}

/** 最后一条 type 为 error 的 key。没有就返回空字符串。items 读取 length，按下标读取。 */
function lastErrorKey(items: Item[]): string {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "error") return items[i].key;
  }
  return "";
}

/** 最后一行 agent 的 key。没有就返回空字符串。rows 读取 length，按下标读取。 */
function lastAgentKey(rows: LayoutRow[]): string {
  for (let i = rows.length - 1; i >= 0; i--) {
    if (rows[i].kind === "agent") return rows[i].key;
  }
  return "";
}

/** 这些片段里有没有 artifact。parts 调用 some。 */
function turnHasArtifact(parts: AgentPart[]): boolean {
  return parts.some((p) => p.kind === "artifact");
}

/** 这一行是不是用户消息。row 读取 kind。 */
function isUserRow(row: LayoutRow): boolean {
  return row.kind === "user";
}

/** 第一行不加顶距。紧凑时用户行 pt-4、其余 pt-2。普通时用户行 pt-10，用户后面的 agent 行 pt-4，连续 agent 行 pt-5。rows 按下标读取。i 减去 1，用作下标。compact 为真时才继续。 */
function rowSpace(rows: LayoutRow[], i: number, compact?: boolean): string {
  const row = rows[i];
  const prev = rows[i - 1];
  if (!prev) return "pt-0";
  if (compact) {
    if (isUserRow(row)) return "pt-4";
    return "pt-2";
  }
  if (isUserRow(row)) return "pt-10";
  if (row.kind === "agent" && prev.kind === "user") return "pt-4";
  if (row.kind === "agent") return "pt-5";
  return "pt-4";
}

/** 第一块不加顶距。过程和 work 紧跟普通条目时用 pt-2.5，否则 pt-1.5。parts 按下标读取。i 为 0 时返回 空字符串；减去 1，用作下标。 */
function partSpace(parts: AgentPart[], i: number): string {
  if (i === 0) return "";
  const prev = parts[i - 1];
  const cur = parts[i];
  if (cur.kind === "process" || cur.kind === "work") return prev.kind === "item" ? "pt-2.5" : "pt-1.5";
  if (cur.kind === "subagent") return "pt-1.5";
  if (cur.kind === "artifact") return "pt-2";
  if (cur.kind === "stream") return prev.kind === "process" || prev.kind === "artifact" || prev.kind === "subagent" ? "pt-3" : "pt-2";
  if (cur.kind === "item" && cur.item.type === "assistant") {
    if (prev.kind === "process" || prev.kind === "artifact" || prev.kind === "subagent") return "pt-3";
    return "pt-2";
  }
  return "pt-2";
}

/**
 * Transcript 在没有可见条目、没在跑、也没有待批时显示空态。showThinking 为假时丢掉 reasoning。紧凑模式不给用户行复制、恢复和分叉。
 * props 读取 flush、compact、transcriptView、collapseTiming、showThinking、items 等。
 */
export function Transcript(props: {
  items: Item[];
  showThinking?: boolean;
  transcriptView?: string;
  collapseTiming?: string;
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
  const policy = useMemo(
    () => presentationPolicyFor(coerceTranscriptView(props.transcriptView), coerceCollapseTiming(props.collapseTiming)),
    [props.transcriptView, props.collapseTiming],
  );
  const visible = useMemo(
    () => (props.showThinking ? props.items : props.items.filter((it) => it.type !== "reasoning")),
    [props.items, props.showThinking],
  );
  const layout = useStableLayout(visible, props.running, policy, props.showThinking !== false);
  const empty = visible.length === 0 && !props.running && props.approvals.length === 0;
  const streamingKey = props.running ? lastStreamKey(layout) : "";
  const liveLen = useMemo(() => {
    let n = 0;
    for (const it of visible) {
      if (it.delta || it.payload?.streaming || it.payload?.delta) n += (it.text || "").length;
    }
    return n;
  }, [visible]);
  const retryKey = lastErrorKey(visible);
  const pinnedTurn = lastAgentKey(layout);

  const rows: TranscriptRow[] = [
    ...layout.map((row, i) => {
      if (row.kind === "user") {
        return {
          key: row.key,
          kind: "user" as const,
          turnKey: row.turnKey || itemTurnKey(row.item),
          space: rowSpace(layout, i, props.compact),
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
        const liveHere = row.parts.some((p) => (p.kind === "item" || p.kind === "stream") && p.item.key === streamingKey);
        const last = row.key === pinnedTurn;
        const lastProcessAt = lastProcessIndex(row.parts);
        const hasStream = row.parts.some((p) => p.kind === "stream");
        return {
          key: row.key,
          kind: "agent" as const,
          turnKey: row.turnKey,
          space: rowSpace(layout, i, props.compact),
          estimate: agentEstimate(row.copyText),
          render: () => (
            <article className={cn("assistant-letter group/turn flex min-w-0 gap-2.5", props.compact && "gap-2")} data-testid="agent-turn">
              <PresenceStamp size={props.compact ? 32 : 36} live={liveHere || (props.running && last)} className="mt-0.5 shrink-0" />
              <div className="min-w-0 flex-1">
              {row.parts.map((part, pi) => (
                <div
                  key={part.key}
                  className={cn(
                    partSpace(row.parts, pi),
                    part.kind === "item" && part.item.type === "assistant" && "assistant-block min-w-0",
                    part.kind === "stream" && "assistant-stream min-w-0",
                    (part.kind === "process" || part.kind === "artifact" || part.kind === "subagent" || part.kind === "work") && "u-chrome",
                  )}
                >
                  {part.kind === "work" ? (
                    <TurnWork
                      part={part}
                      running={props.running}
                      last={last}
                      compact={props.compact}
                      liveDetail={policy.liveProcessDetail}
                      streamingKey={streamingKey}
                      workspace={props.workspace}
                      onOpenReview={props.onOpenReview}
                    />
                  ) : part.kind === "process" ? (
                    <ProcessGroup
                      items={part.items}
                      live={props.running && last && (
                        processGroupLive(part.items)
                        || (pi === lastProcessAt && !hasStream)
                      )}
                      running={props.running}
                      compact={props.compact}
                      grouped={part.grouped !== false}
                      liveDetail={policy.liveProcessDetail}
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
                  ) : part.kind === "stream" ? (
                    <ItemRow
                      item={part.item}
                      streaming={!!part.item.delta || !!part.item.payload?.streaming}
                      rich={false}
                    />
                  ) : (
                    <ItemRow
                      item={part.item}
                      streaming={false}
                      rich={part.item.type === "assistant" && !props.running}
                    />
                  )}
                </div>
              ))}
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
        space: rowSpace(layout, i, props.compact),
        estimate: 96,
        render: () => (
          <ItemRow
            item={row.item}
            onRetry={row.item.key === retryKey ? props.onRetry : undefined}
          />
        ),
      };
    }),
    ...props.approvals.filter((a) => a.action !== "ask_user").map((a) => ({
      key: `ask:${a.id}`,
      kind: "ask" as const,
      space: "pt-4",
      estimate: 160,
      render: () => <ApprovalCard item={a} onResolve={props.onResolve} />,
    })),
  ];
  return (
    <div
      className="prose-select relative flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      data-testid="conversation-column"
    >
      {empty && !props.compact ? (
        <div className={cn(col, "transcript-scroll flex min-h-full flex-col justify-end overflow-y-auto pb-8 pt-10")}>
          <EmptyTurn workspace={props.workspace} onPrompt={props.onPrompt} />
        </div>
      ) : visible.length === 0 && props.approvals.length === 0 && props.compact ? (
        <CompactCopilotEmpty className={col} />
      ) : (
        <TranscriptLane
          className={cn(col, props.compact ? "min-h-0 flex-1 py-2" : "min-h-0 flex-1 pb-4 pt-5")}
          compact={props.compact}
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

/**
 * 折起来时过程不再标成进行中。展开时，只有最后一轮、且这块过程自己在动或它是最后一块且还没有流式正文时，才标成进行中。
 * part 读取 parts、folded。
 * running 为假时走另一支，写到 ArtifactTimeline 的 running。
 * last 为真时才继续。
 * compact 写到 ProcessGroup 的 compact。
 * liveDetail 写到 ProcessGroup 的 liveDetail。
 * workspace 写到 ArtifactTimeline 的 workspace。
 * onOpenReview 写到 ArtifactTimeline 的 onOpenReview。
 * streamingKey 为假时走另一支。
 */
function TurnWork({
  part,
  running,
  last,
  compact,
  liveDetail,
  workspace,
  onOpenReview,
  streamingKey,
}: {
  part: Extract<AgentPart, { kind: "work" }>;
  running: boolean;
  last: boolean;
  compact?: boolean;
  liveDetail: boolean;
  workspace?: string;
  onOpenReview?: (path?: string) => void;
  streamingKey: string;
}) {
  const inner = part.parts;
  const lastProcessAt = lastProcessIndex(inner);
  const hasStream = inner.some((p) => p.kind === "stream") || !!streamingKey;
  if (part.folded) {
    const processItems = flattenWorkProcess(inner);
    return (
      <div data-testid="turn-work" data-folded="true">
        {processItems.length ? (
          <ProcessGroup items={processItems} live={false} running={false} compact={compact} grouped liveDetail={liveDetail} />
        ) : null}
        {inner.map((p) => p.kind === "artifact" ? (
          <div key={p.key} className="pt-2">
            <ArtifactTimeline items={p.items} running={running} workspace={workspace} onOpenReview={onOpenReview} />
          </div>
        ) : null)}
      </div>
    );
  }
  return (
    <div data-testid="turn-work" data-folded="false">
      {inner.map((p, pi) => (
        <div
          key={p.key}
          className={cn(
            partSpace(inner, pi),
            p.kind === "item" && p.item.type === "assistant" && "assistant-block min-w-0",
            p.kind === "stream" && "assistant-stream min-w-0",
            (p.kind === "process" || p.kind === "artifact" || p.kind === "subagent") && "u-chrome",
          )}
        >
          {p.kind === "process" ? (
            <ProcessGroup
              items={p.items}
              live={running && last && (processGroupLive(p.items) || (pi === lastProcessAt && !hasStream))}
              running={running}
              compact={compact}
              grouped={p.grouped !== false}
              liveDetail={liveDetail}
            />
          ) : p.kind === "subagent" ? (
            <SubagentCard items={p.items} child={p.child} live={running && p.live} running={running} compact={compact} />
          ) : p.kind === "artifact" ? (
            <ArtifactTimeline items={p.items} running={running} workspace={workspace} onOpenReview={onOpenReview} />
          ) : p.kind === "stream" ? (
            <ItemRow item={p.item} streaming={!!p.item.delta || !!p.item.payload?.streaming} rich={false} />
          ) : p.kind === "item" ? (
            <ItemRow item={p.item} streaming={false} rich={p.item.type === "assistant" && !running} />
          ) : null}
        </div>
      ))}
    </div>
  );
}

/**
 * 结构、策略和是否显示思考没变时，沿用上次排好的行，再用当前条目把内容填回去。
 * items 传给 structureSig，传给 layoutRows，传给 hydrateLayout。
 * running 传给 structureSig，写入字段 running。
 * policy 读取 mode、collapseTiming，写入字段 policy。
 * showThinking 为真时才继续，写入字段 showThinking。
 */
function useStableLayout(items: Item[], running: boolean, policy: ReturnType<typeof presentationPolicyFor>, showThinking: boolean): LayoutRow[] {
  const sig = `${structureSig(items, running)}\n${policy.mode}\t${policy.collapseTiming}\t${showThinking ? 1 : 0}`;
  const ref = useRef({ sig: "", rows: [] as LayoutRow[] });
  if (ref.current.sig !== sig) {
    ref.current = { sig, rows: layoutRows(items, { running, showThinking, policy }) };
  }
  return hydrateLayout(ref.current.rows, items);
}

/** 陪伴侧空列表只占住剩余高度，不画问候。className 传给 cn。 */
function CompactCopilotEmpty({ className }: { className: string }) {
  return <div className={cn(className, "min-h-0 flex-1")} />;
}

/** 空线程贴在输入框上方，问候、提示和输入读成一块。workspace 传给 displayWorkspace。onPrompt 被调用，并传入 s.text。 */
function EmptyTurn({ workspace, onPrompt }: { workspace?: string; onPrompt?: (text: string) => void }) {
  const copy = useCopy();
  const surface = useUI((s) => s.surface);
  const videoMode = useUI((s) => s.videoMode);
  const studioDesk = useUI((s) => s.studioDesk);
  const videoPane = useUI((s) => s.videoPane);
  const videoing = surface === "video";
  const createDesk = videoing && videoPane === "create";
  const openStudioDesk = useUI((s) => s.openStudioDesk);
  const boundTitle = useStudioSelection((s) => s.title);
  const nextKey = useStudioSelection((s) => s.nextKey);
  const ready = studioDesk
    ? (boundTitle
      ? copy.transcript.studioDeskReadyNamed.replace("{title}", boundTitle)
      : copy.transcript.studioDeskReady)
    : createDesk
      ? copy.transcript.conductorReady
    : !videoing
      ? copy.transcript.ready
    : videoMode === "canvas"
      ? copy.transcript.canvasReady
      : videoMode === "creative"
        ? copy.transcript.creativeReady
        : copy.transcript.videoReady;
  const nextStarter = studioDesk && ["collect", "produce", "setup", "script", "audio", "visuals", "compose", "pack", "open"].includes(nextKey)
    ? { label: copy.video.studio.nextDo, text: studioNextPrompt(copy.video.studio, nextKey as StudioNextKey) }
    : null;
  const starters = studioDesk
    ? [nextStarter, ...copy.transcript.studioDeskStarters].filter(Boolean) as { label: string; text: string }[]
    : createDesk
      ? copy.transcript.conductorStarters
    : !videoing
      ? copy.transcript.starters
    : videoMode === "canvas"
      ? copy.transcript.canvasStarters
      : videoMode === "creative"
        ? copy.transcript.creativeStarters
        : copy.transcript.videoStarters;
  const ws = displayWorkspace(workspace, "");
  const hints = (studioDesk
    ? [
        { key: "enter", node: <><kbd className="mr-1">↵</kbd>{copy.transcript.hintSend}</> },
        { key: "desk", node: copy.transcript.hintStudioDesk },
      ]
    : createDesk
    ? [
        { key: "enter", node: <><kbd className="mr-1">↵</kbd>{copy.transcript.hintSend}</> },
        { key: "desk", node: copy.transcript.hintConductor },
      ]
    : [
        ws ? { key: "ws", node: <><span className="text-muted">{copy.transcript.workingIn}</span><span className="ml-1.5 font-mono text-[12px] text-foreground/80">{ws}</span></> } : null,
        { key: "enter", node: <><kbd className="mr-1">↵</kbd>{copy.transcript.hintSend}</> },
        videoing
          ? { key: "mode", node: copy.transcript.hintVideoMode }
          : { key: "at", node: <><kbd className="mr-1">@</kbd>{copy.transcript.hintMention}</> },
        videoing ? null : { key: "plan", node: <><kbd className="mr-1">⇧⇥</kbd>{copy.transcript.hintPlan}</> },
      ]
  ).filter(Boolean) as { key: string; node: ReactNode }[];
  return (
    <div className="empty-rise" data-testid="empty-turn">
      {studioDesk ? null : <MarkWell />}
      <h1 className={cn(
        "font-semibold leading-[1.15] tracking-[-0.042em] text-pretty text-foreground",
        studioDesk ? "text-[16px]" : "mt-6 max-w-[16ch] text-[21px]",
      )}>
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
      <div className={cn("flex flex-wrap gap-2", studioDesk ? "mt-4" : "mt-7")}>
        {videoing && videoMode === "creative" && !studioDesk && !createDesk ? (
          <button
            type="button"
            data-testid="studio-open-desk-empty"
            className="inline-flex cursor-pointer items-center gap-1.5 rounded-full border border-border/80 bg-transparent py-1.5 pl-3 pr-2.5 text-[13px] text-muted transition-[background-color,color,transform,border-color] duration-200 ease-[var(--ease-out)] hover:border-border hover:bg-lift hover:text-foreground active:scale-[0.98]"
            onClick={() => openStudioDesk()}
          >
            {copy.video.shellStudio}
            <ArrowUpRight className="size-3 text-muted/60" aria-hidden />
          </button>
        ) : null}
        {starters.map((s, i) => (
          <button
            type="button"
            key={`${s.label}-${i}`}
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

/** 虚拟列表里的一行：估算高度和怎么画。 */
type TranscriptRow = {
  key: string;
  kind?: "user" | "agent" | "solo" | "ask" | "working";
  turnKey?: string;
  space?: string;
  estimate: number;
  render: () => ReactNode;
};

/** 按正文字数估算高度，夹在 520 和 2400 之间，底数是 160 加长度乘 0.48。copyText 读取 length，为空时不读。 */
function agentEstimate(copyText: string): number {
  return Math.max(520, Math.min(2400, 160 + Math.round((copyText?.length || 0) * 0.48)));
}

/**
 * 跟底阈值是 80。跳转序号变了就先停止跟随。锚在列表末尾。
 * className 传给 cn。
 * compact 为真时才继续。
 * rows 取第一段，读取 length、key，为空时不读，调用 findIndex，按下标读取，展开传递。
 * older 为真时才继续。
 * loadingOlder 写到 button 的 disabled。
 * overscan 写入字段 overscan。
 * onLoadOlder 为真时才继续，被调用。
 * follow 为假时走另一支。
 * followNonce 放进数组。
 * jumpTo 读取 nonce、key、aliases，为空时不读。
 * latestNonce 与 appliedLatest.current 比较是否相等，为假时走另一支，赋给 appliedLatest.current。
 * onActiveTurn 为假时走另一支，被调用，并传入 key。
 * onJumpLatest 为真时才继续，被调用。
 */
function TranscriptLane({
  className,
  compact,
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
  compact?: boolean;
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
  const followCtl = useRef<ScrollFollow | null>(null);
  if (!followCtl.current) followCtl.current = new ScrollFollow(true, 80);
  const [escaped, setEscaped] = useState(false);
  const loadingRef = useRef(false);
  const appliedJump = useRef(0);
  const appliedLatest = useRef(0);
  const lastActive = useRef('');
  const spyRaf = useRef(0);
  const jumpHeld = useRef(false);
  const jumpNonce = jumpTo?.nonce || 0;
  if (jumpNonce && jumpNonce !== appliedJump.current) {
    followCtl.current.setFollowing(false);
  }
  const listRef = useRef<HTMLDivElement>(null);
  const [scrollMargin, setScrollMargin] = useState(0);
  const following = followCtl.current.active;
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    /** estimateSize 根据参数得到结果。i 用作下标。 */
    estimateSize: (i) => rows[i]?.estimate ?? 96,
    overscan,
    /** getItemKey 根据参数得到结果。i 用作下标，作为后备值。 */
    getItemKey: (i) => rows[i]?.key ?? i,
    anchorTo: "end",
    followOnAppend: follow && following,
    scrollEndThreshold: 80,
    scrollMargin,
  });
  /** 没在跟随且这次是瞬间滚动、调用方也不要求跟随时，直接返回。behavior 与 instant 比较是否相等，传给 ctl.toBottom。 */
  const followEnd = useCallback((behavior: "instant" | "smooth" = "instant") => {
    const el = scrollRef.current;
    const ctl = followCtl.current;
    if (!el || !ctl) return;
    if (!ctl.active && behavior === "instant" && !follow) return;
    ctl.toBottom(el, scrollMetrics(el), behavior);
    setEscaped(!ctl.active);
  }, [follow]);
  /** 滚轮、按下或触摸开始后，不再按跳转目标锁住当前回合。 */
  const releaseJump = useCallback(() => {
    jumpHeld.current = false;
  }, []);
  useLayoutEffect(() => {
    const list = listRef.current;
    const scroller = scrollRef.current;
    if (!list || !scroller) return;
    const margin = list.offsetTop - scroller.offsetTop;
    if (margin !== scrollMargin) setScrollMargin(margin);
  }, [older, rows.length, scrollMargin]);
  useLayoutEffect(() => {
    const el = scrollRef.current;
    const ctl = followCtl.current;
    if (!el || !ctl) return;
    return ctl.bind(el);
  }, []);
  useLayoutEffect(() => {
    const el = scrollRef.current;
    const ctl = followCtl.current;
    if (!el || !ctl || !ctl.active) return;
    ctl.toBottom(el, scrollMetrics(el), follow ? "smooth" : "instant");
    setEscaped(!ctl.active);
  }, [rows.length, follow, followNonce]);
  useLayoutEffect(() => {
    if (!latestNonce || latestNonce === appliedLatest.current) return;
    appliedLatest.current = latestNonce;
    jumpHeld.current = false;
    followCtl.current?.setFollowing(true);
    followEnd("smooth");
  }, [latestNonce, followEnd]);
  useLayoutEffect(() => {
    const key = jumpTo?.key;
    const nonce = jumpTo?.nonce || 0;
    if (!key || nonce === appliedJump.current) return;
    const aliases = jumpTo?.aliases?.length ? jumpTo.aliases : [key];
    const idx = rows.findIndex((r) => rowMatchesJump(r.turnKey, aliases) || rowMatchesJump(r.key, aliases));
    followCtl.current?.setFollowing(false);
    setEscaped(true);
    lastActive.current = key;
    if (idx < 0) return;
    appliedJump.current = nonce;
    jumpHeld.current = true;
    /** 跳转已经被松开就不再滚动。否则把这一行对齐到列表开头。 */
    const align = () => {
      if (!jumpHeld.current) return;
      virtualizer.scrollToIndex(idx, { align: "start" });
    };
    align();
    requestAnimationFrame(align);
  }, [jumpTo, rows.length, rows[0]?.key, rows[rows.length - 1]?.key, virtualizer]);
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    /** 滚动停稳后按当前位置决定还跟不跟底。 */
    const onEnd = () => {
      const ctl = followCtl.current;
      if (!ctl) return;
      ctl.settle(scrollMetrics(el));
      setEscaped(!ctl.active);
    };
    el.addEventListener('wheel', releaseJump, { passive: true });
    el.addEventListener('pointerdown', releaseJump);
    el.addEventListener('touchstart', releaseJump, { passive: true });
    el.addEventListener('scrollend', onEnd);
    return () => {
      el.removeEventListener('wheel', releaseJump);
      el.removeEventListener('pointerdown', releaseJump);
      el.removeEventListener('touchstart', releaseJump);
      el.removeEventListener('scrollend', onEnd);
    };
  }, [releaseJump]);
  /** 同一个 key 不重复上报。 */
  const reportActive = useCallback((key: string) => {
    if (!key || key === lastActive.current) return;
    lastActive.current = key;
    onActiveTurn?.(key);
  }, [onActiveTurn]);
  /** 跳转还按着时上报跳转目标。离底部不到 80 像素时上报最后一条带 turnKey 的行。否则按视口顶往下最多 64 像素或高度 16% 的那条线选。 */
  const spyTurns = useCallback(() => {
    const root = scrollRef.current;
    if (!root || !onActiveTurn) return;
    if (jumpHeld.current) {
      const k = jumpTo?.key;
      if (k) reportActive(k);
      return;
    }
    const gap = root.scrollHeight - root.scrollTop - root.clientHeight;
    if (gap < 80) {
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
  /** 同一帧只排一次。 */
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
  /** 滚到顶不足 48 像素且还有更早内容时加载，加载完按高度差把滚动位置补回去。 */
  const onScroll = () => {
    const el = scrollRef.current;
    const ctl = followCtl.current;
    if (!el || !ctl) return;
    if (!jumpHeld.current) {
      ctl.sample(scrollMetrics(el));
      setEscaped(!ctl.active);
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
        className={cn('transcript-scroll relative min-h-0 flex-1', compact ? 'overflow-y-scroll overflow-x-hidden' : 'overflow-y-auto', className)}
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
        <div ref={listRef} className="relative w-full shrink-0 overflow-hidden bg-background" style={{ height: virtualizer.getTotalSize() }}>
          {virtualizer.getVirtualItems().map((v) => {
            const row = rows[v.index];
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
                  transform: `translateY(${v.start - scrollMargin}px)`,
                }}
              >
                {row.render()}
              </div>
            );
          })}
        </div>
      </div>
      {escaped ? (
        <button
          type='button'
          className='absolute bottom-4 left-1/2 z-10 inline-flex -translate-x-1/2 cursor-pointer items-center gap-1.5 rounded-full border border-border bg-popover py-1.5 pl-2.5 pr-3 text-[12px] text-foreground shadow-[var(--shadow-popover)] transition-[background-color,transform] duration-200 ease-[var(--ease-out)] hover:bg-lift active:scale-[0.98]'
          onClick={() => {
            followCtl.current?.setFollowing(true);
            setEscaped(false);
            if (onJumpLatest) void onJumpLatest();
            else followEnd("smooth");
          }}
        >
          <ArrowDown className='size-3 text-muted' aria-hidden />
          {copy.transcript.jumpLatest}
        </button>
      ) : null}
    </>
  );
}

/** 还在生成时只留一行空白。否则悬停或焦点才显示复制，以及可选的打开审查。text 写到 CopyAction 的 text。live 为真时才继续。pinned 为真时才继续。showReview 为真时才继续。onOpenReview 为真时才继续，被调用。 */
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

/** ask_user 要先有非空回答才能以 once 提交。其余审批用条目上的命令、路径或级别当标题。item 读取 action、command、path、level、id。onResolve 被调用，并传入 item.id、"once"。 */
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
          <span className="inline-flex items-center gap-2">
            <span className="text-[12px] font-medium text-foreground">{copy.transcript.sendAnswer}</span>
            <SendSling
              disabled={!answer.trim()}
              ariaLabel={copy.transcript.sendAnswer}
              title={copy.transcript.sendAnswer}
              padColor="var(--foreground)"
              iconColor="var(--background)"
              accentColor="var(--foreground)"
              onSend={() => onResolve(item.id, "once", answer.trim())}
            />
          </span>
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

/** 回合操作按钮的共用样式。 */
const turnActionBtn =
  "inline-flex h-7 items-center gap-1 rounded-md px-1.5 text-[11px] text-muted hover:bg-lift hover:text-foreground disabled:opacity-40";

/** 没有文字就不渲染。写入成功后勾选保持 1400 毫秒。失败弹出复制失败。text 为假时返回 null；传给 writeClipboard，写到 button 的 data-copy-text。 */
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

/** canceled、max_turns、plan_open 用柔和样式。细节和标题相同就不展开。可重试且给了 onRetry 才显示重试。item 读取 payload，传给 classifyItem。 */
function ErrorCard({ item, onRetry }: { item: Item; onRetry?: () => void }) {
  const copy = useCopy();
  const classified = classifyItem(item);
  const labels = errorCopy(copy.transcript, classified.kind);
  const hint = classified.payloadHint || labels.hint;
  const detail = classified.detail;
  const showDetail = !!detail && detail !== labels.title && detail !== classified.payloadTitle;
  const soft = classified.kind === "canceled" || classified.kind === "max_turns" || classified.kind === "plan_open";
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

/** 没有正文、也没有恢复或分叉回调时不渲染。正在跑其中一个时不再接受第二次。text 为假时走另一支，写到 CopyAction 的 text。from 为假时走另一支，传给 fn。onRestore 为假时走另一支，传给 run。onFork 为假时走另一支，传给 run。 */
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
  /** 没有来源、没有回调或已经在跑时直接返回。结束后清掉忙碌标记。kind 传给 setBusy。fn 为假时走另一支，被调用，并传入 from。 */
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

/** 优先用 note，否则拼保留、省略、补回的数量。都没有时用压缩文案。item 读取 payload、text。 */
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

/** 文件夹、技能、harness、会话各用自己的图标，其余用文件图标。kind 与 folder、skill、harness、session 比较是否相等。 */
function MentionGlyph({ kind }: { kind: MentionKind }) {
  const cls = "size-3 shrink-0 opacity-80";
  if (kind === "folder") return <Folder className={cls} />;
  if (kind === "skill") return <BookOpen className={cls} />;
  if (kind === "harness") return <Boxes className={cls} />;
  if (kind === "session") return <MessageSquare className={cls} />;
  return <FileText className={cls} />;
}

/** 把完成的提及和附件收成芯片，剩下的正文照原文显示。图片只取带 image_url 且类型是图片的部分。text 作为后备值，传给 peelCompletedMentions。parts 为假时改用后备值，做过滤。attachments 传给 pinsFromAttachments。 */
function UserPrompt({
  text,
  parts,
  attachments,
}: {
  text: string;
  parts?: { type?: string; image_url?: string; mime?: string; text?: string }[];
  attachments?: { name?: string; path?: string; mime?: string }[];
}) {
  const peeled = peelCompletedMentions(text);
  const chips: MentionPin[] = mergeMentionPins(peeled.chips, pinsFromAttachments(attachments));
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
              data-testid={c.kind === "file" ? "user-attach-chip" : "user-mention-chip"}
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

/**
 * 用户的改道消息去掉固定前缀后居中。turn_end、system、plan、file_change、ask_user、approval 不在这里画。
 * item 作为分支结果，展开传递。
 * streaming 为假时走另一支，写到 Markdown 的 streaming。
 * liveText 与 null 比较是否不等，写入字段 text。
 * copyText 为假时用 onRestore，为假时用 onFork，写到 UserMsgActions 的 text。
 * from 写到 UserMsgActions 的 from。
 * onRestore 作为后备值，写到 UserMsgActions 的 onRestore。
 * onFork 作为后备值，写到 UserMsgActions 的 onFork。
 * rich 为假时走另一支。
 * onRetry 写到 ErrorCard 的 onRetry。
 */
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
    const clock = formatMessageClock(Date.parse(shown.ts), { md: copy.transcript.clockMd, ymd: copy.transcript.clockYmd });
    return (
      <div className="flex justify-end">
        <div className="group/msg w-fit max-w-[80%]">
          <UserPrompt
            text={shown.text}
            parts={Array.isArray(shown.payload?.parts) ? shown.payload.parts : undefined}
            attachments={Array.isArray(shown.payload?.attachments) ? shown.payload.attachments : undefined}
          />
          <div className="mt-0.5 flex min-h-5 items-center justify-end gap-1.5">
            {clock ? <span className="text-[11px] tabular-nums text-muted/65">{clock}</span> : null}
            {copyText || onRestore || onFork ? (
              <UserMsgActions text={copyText} from={from} onRestore={onRestore} onFork={onFork} />
            ) : null}
          </div>
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

/**
 * 没有产物对就返回 null。一次多显示 ARTIFACT_PAGE 条，更早的用按钮展开。
 * items 传给 pairTools。
 * running 写到 ArtifactCard 的 running。
 * workspace 写到 ArtifactCard 的 workspace。
 * onOpenReview 写到 ArtifactCard 的 onOpenReview。
 */
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
  const [shown, setShown] = useState(ARTIFACT_PAGE);
  const classified = classifyToolPairs(pairTools(items));
  const all = pairTools(items).filter((p) => classified.get(p.key) === "artifact");
  if (!all.length) return null;
  const { visible: pairs, hidden } = tailProcessPairs(all, shown);
  return (
    <div className="u-chrome min-w-0 space-y-1" data-testid="tool-timeline">
      {hidden > 0 ? (
        <button
          type="button"
          className="h-7 w-full rounded-md px-1 text-left text-[12px] text-muted transition-colors hover:bg-lift/50 hover:text-foreground"
          data-testid="artifact-earlier"
          onClick={() => setShown((n) => n + ARTIFACT_PAGE)}
        >
          {copy.transcript.earlierSteps.replace("{n}", String(hidden))}
        </button>
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

/**
 * 页面审查和浏览器截图交给各自的卡片。代码改动或还在等待结果时不自动展开。
 * call 为假时用 item，为假时用 result，作为后备值，传给 artifactView，传给 formatToolBody，传给 toolArgs，传给 isPageReviewItem，传给 isBrowserShotItem，写到 PageReviewCard 的 item。
 * result 为假时改用后备值，为假时用 call，为假时用 item，作为后备值，传给 artifactView，传给 toolDetail，传给 formatToolBody，传给 isPageReviewItem，传给 isBrowserShotItem，写到 PageReviewCard 的 result。
 * running 为假时走另一支。
 * workspace 为假时走另一支，写到 ArtifactBody 的 workspace。
 * onOpenReview 为真时才继续，被调用，并传入 path || undefined。
 */
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
  const openCall = !!call && !result;
  const pending = openCall && running;
  const interrupted = openCall && !running;
  /** 代码改动卡片先折着，等操作者打开差异。其余预览（网页、MCP 应用、文档）自己展开。 */
  const codeChange = isCodeChangeTool(name);
  const [open, setOpen] = useState(() => !codeChange && !pending && artifactPreviewOpen(view));
  const detail = toolDetail(result || call || item);
  const body = result ? formatToolBody(result) : formatToolBody(call || item);
  const files = name === "apply_patch" ? patchFileCount(body) : 0;
  const args = toolArgs(call || result || item);
  const path = view.path || String(args.path || args.file || "");
  const mcpHtml = result ? extractHTML(body) : "";
  const pdf = looksLikePDF(path);
  const office = name.startsWith("office_") && /\.(docx|xlsx|pptx|pdf)$/i.test(path);
  const mcp = looksLikeMCPUI(mcpHtml, name);
  const htmlish = view.kind === "html" || looksLikeHTMLFile(path) || looksLikeHTML(view.html) || mcp;
  const hasPreview = artifactShouldShow(view) || mcp || ((office || pdf) && !!path && !!workspace) || (htmlish && !!workspace && looksLikeHTMLFile(path)) || !!view.diff;
  const fileChip = isArtifactTool(name) && !!path;
  useEffect(() => {
    if (codeChange || pending) return;
    if (artifactPreviewOpen(view) || !!view.diff) setOpen(true);
  }, [codeChange, pending, view.kind, view.path, view.html, view.diff]);
  if (isPageReviewItem(call) || isPageReviewItem(result)) {
    return <PageReviewCard item={call} result={result} />;
  }
  if (isBrowserShotItem(call) || isBrowserShotItem(result)) {
    return <BrowserShotCard call={call} result={result} />;
  }
  if (!office && !pdf && name !== "cite_sources" && !hasPreview && !fileChip) return null;
  const title = view.path
    ? view.path.replace(/\\/g, "/").split("/").pop() || view.path
    : name === "apply_patch"
      ? copy.transcript.patch
      : name === "cite_sources"
        ? copy.transcript.citations
        : mcp
          ? copy.transcript.mcpApp
          : htmlish
            ? copy.transcript.preview
            : copy.transcript.artifact;
  const added = view.added || 0;
  const removed = view.removed || 0;
  const counts = [
    view.created ? copy.transcript.created : "",
    name === "apply_patch" && files > 0 ? copy.transcript.filesCount.replace("{n}", String(files)) : "",
  ].filter(Boolean);
  return (
    <div
      className={cn("file-bubble u-card-hover", pending && "is-pending")}
      data-testid="artifact-card"
    >
      <div className="file-bubble-header">
        <span className="file-bubble-tile" aria-hidden>
          {pending ? <span className="pulse-dot" /> : interrupted ? <CircleDashed className="size-3.5 text-warning" /> : <FileText className="size-5" />}
        </span>
        <div className="file-bubble-titles">
          <div className="file-bubble-title">
            <span className={cn("truncate", pending && "shimmer-text")}>{title}</span>
            {interrupted ? (
              <span className="ml-2 rounded-md bg-warning/12 px-1.5 py-px text-[11px] font-medium text-warning">{copy.transcript.toolInterrupted}</span>
            ) : null}
          </div>
          <div className="file-bubble-stat">
            <span className="file-bubble-stat-counts">
              {counts.length ? <span>{counts.join(" · ")}</span> : null}
              {added ? <span className="file-bubble-added">+{added}</span> : null}
              {removed ? <span className="file-bubble-deleted">−{removed}</span> : null}
              {!added && !removed && !counts.length ? <span className="truncate">{path || detail || name}</span> : null}
            </span>
            <span className="file-bubble-hint">{copy.transcript.preview}</span>
          </div>
        </div>
        <div className="file-bubble-actions">
          {hasPreview ? (
            <button
              type="button"
              aria-expanded={open}
              onClick={() => setOpen((v) => !v)}
            >
              {open ? copy.transcript.hidePreview : copy.transcript.preview}
            </button>
          ) : null}
          {htmlish && open ? (
            <button type="button" onClick={() => setSource((v) => !v)}>
              {source ? copy.transcript.preview : copy.transcript.source}
            </button>
          ) : null}
          {onOpenReview ? (
            <button type="button" onClick={() => onOpenReview(path || undefined)}>
              {copy.review.openReview}
            </button>
          ) : null}
        </div>
      </div>
      {open ? (
        <>
          {mcp && mcpHtml && !view.diff && !view.code && !view.html ? (
            <SandboxedFrame html={mcpHtml} title={title} allowScripts />
          ) : (
            <ArtifactBody view={{ ...view, path: view.path || path, kind: view.kind || ((office || pdf) ? "text" : view.kind) }} workspace={workspace} source={source} />
          )}
        </>
      ) : null}
    </div>
  );
}
