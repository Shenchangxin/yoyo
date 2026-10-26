// 本文件拥有一轮工具过程的折叠、进行中的标题和单条工具行。不拥有工具怎么配对，那一步在 transcript-layout。Transcript、SubagentCard 和 TurnStatusBar 使用它。

import { memo, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { AtSign, Bot, Brain, ChevronRight, CircleDashed, MessageSquare, X } from "lucide-react";
import { Markdown } from "../../lib/markdown";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import type { Copy } from "../../lib/copy";
import { DURATION_FAST, motionTransition, useMotionReduced } from "../../lib/motion";
import {
  formatElapsed,
  formatSpan,
  formatToolBody,
  isToolFailed,
  skillCallLabel,
  toolArgs,
  toolDetail,
  toolElapsedMs,
  toolKind,
  toolName,
  type ToolKind,
} from "../../lib/tool-summary";
import { langFromPath } from "../../lib/artifact-preview";
import { CodePreview } from "./FilePreview";
import type { Item } from "../../lib/protocol";
import * as api from "../../lib/client";
import { reasoningText } from "../../lib/transcript-layout";
import { writeClipboard } from "../../lib/clipboard";
import {
  pairState,
  pairTools,
  processFailedCount,
  processHasReasoning,
  processKindTally,
  processProgressCount,
  processSpanMs,
  processStartMs,
  tailProcessPairs,
  PROCESS_PAGE,
  clipPreview,
  type PairState,
  type ToolPair,
} from "../../lib/transcript-layout";

type TranscriptCopy = Copy["transcript"];

// 进行中的动词不能比 150 毫秒更快地换成下一个。
const PROCESS_TITLE_MINIMUM_MS = 150;

/** useStableLiveKey 不在进行中时立刻跟上 key。进行中时若 key 变了，先等到距上次稳住满 150 毫秒。live 为假时走另一支。 */
function useStableLiveKey(key: string, live: boolean): string {
  const [shown, setShown] = useState(key);
  const heldAt = useRef(Date.now());
  useEffect(() => {
    if (!live) {
      setShown(key);
      heldAt.current = Date.now();
      return;
    }
    if (shown === key) {
      heldAt.current = Date.now();
      return;
    }
    const wait = Math.max(0, PROCESS_TITLE_MINIMUM_MS - (Date.now() - heldAt.current));
    const id = window.setTimeout(() => {
      heldAt.current = Date.now();
      setShown(key);
    }, wait);
    return () => window.clearTimeout(id);
  }, [key, live, shown]);
  return live ? shown : key;
}

// useNow 只在 active 为真时每 1000 毫秒走一格。停下后不再计时。
export function useNow(active: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    setNow(Date.now());
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [active]);
  return now;
}

/**
 * WorkingLine 用在回合还活着、但没有工具在飞的时候。since 是上一条用户消息的时间，时钟不在每一批重新起算。没有 since，或 now 不大于 since，就不显示耗时。
 * label 为假时改用后备值，作为子节点渲染。
 * detail 为真时才继续，作为子节点渲染。
 * hint 为真时才继续，作为子节点渲染。
 */
export function WorkingLine({
  since,
  label,
  detail,
  hint,
}: {
  since?: number;
  label?: string;
  detail?: string;
  hint?: string;
}) {
  const copy = useCopy();
  const now = useNow(!!since);
  const elapsed = since && now > since ? formatSpan(now - since) : "";
  return (
    <div
      className="flex h-7 min-w-0 items-center gap-2.5 text-[13px]"
      role="status"
      aria-live="polite"
      data-testid="working-line"
    >
      <span className="grid size-[22px] shrink-0 place-items-center" aria-hidden>
        <span className="pulse-dot" />
      </span>
      <span className="shimmer-text shrink-0 font-medium">{label || copy.transcript.working}</span>
      {detail ? <span className="min-w-0 truncate font-mono text-[12px] text-muted/80">{detail}</span> : null}
      {elapsed ? <span className="shrink-0 tabular-nums text-[12px] text-muted/70">{elapsed}</span> : null}
      {hint ? <span className="ml-auto shrink-0 text-[11px] text-muted/55">{hint}</span> : null}
    </div>
  );
}

/**
 * ProcessGroup 默认收成一组，liveDetail 默认打开。用户没点过时，分组的默认是合上。展开后先显示一页，再按 PROCESS_PAGE 往前补。进行中且没有当前工具时标题停在 composing。
 * rawItems 传给 pairTools，传给 processFailedCount，传给 processStartMs，传给 processSpanMs，传给 summarize。
 * live 为真时才继续，传给 useStableLiveKey，传给 useNow。
 * running 为假时走另一支，传给 pairState，写到 ProcessPair 的 running。
 * compact 为真时才继续，写到 ProcessPair 的 compact。
 * grouped 为假时走另一支。
 * onOpenChild 写到 ProcessPair 的 onOpenChild。
 */
export const ProcessGroup = memo(function ProcessGroup({
  items: rawItems,
  live,
  running,
  compact,
  grouped = true,
  liveDetail = true,
  onOpenChild,
}: {
  items: Item[];
  live: boolean;
  running: boolean;
  compact?: boolean;
  grouped?: boolean;
  liveDetail?: boolean;
  onOpenChild?: (id: string) => void;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const swap = motionTransition(reduced, DURATION_FAST);
  const [userOpen, setUserOpen] = useState<boolean | null>(null);
  const open = userOpen ?? !grouped;
  const [shown, setShown] = useState(PROCESS_PAGE);
  const items = rawItems;
  const pairs = useMemo(() => pairTools(items), [items]);
  const tools = useMemo(() => pairs.filter((p) => p.call || p.result), [pairs]);
  const pending = tools.filter((p) => pairState(p, running) === "running");
  const current = live ? pending[pending.length - 1] : undefined;
  const shownKey = useStableLiveKey(current?.key || (live ? "composing" : ""), live);
  const livePair = shownKey === "composing" || !shownKey
    ? undefined
    : tools.find((p) => p.key === shownKey) || current;
  const failed = processFailedCount(items);
  const interrupted = !running && tools.some((p) => pairState(p, false) === "interrupted");
  const start = processStartMs(items);
  const now = useNow(live);
  const spanMs = live ? (start ? Math.max(0, now - start) : 0) : processSpanMs(items);
  const span = formatSpan(spanMs);
  const summary = summarize(items, copy.transcript);
  const settledLabel = span ? copy.transcript.workedFor.replace("{n}", span) : "";
  const { visible: openPairs, hidden } = open ? tailProcessPairs(pairs, shown) : { visible: [] as typeof pairs, hidden: 0 };

  return (
    <div className="u-chrome min-w-0" data-testid="process-group" data-live={live ? "true" : "false"}>
      <button
        type="button"
        className="group/head flex h-7 w-full min-w-0 items-center gap-2.5 rounded-md text-left text-[13px]"
        data-testid={live ? undefined : "process-summary"}
        aria-expanded={open}
        title={open ? copy.transcript.hideSteps : copy.transcript.showSteps}
        onClick={() => setUserOpen(!(userOpen ?? !grouped))}
      >
        <span className="grid size-[22px] shrink-0 place-items-center" aria-hidden>
          {live ? (
            <span className="pulse-dot" />
          ) : failed > 0 ? (
            <X className="size-3 text-danger" strokeWidth={2.4} />
          ) : interrupted ? (
            <CircleDashed className="size-3 text-warning" strokeWidth={2.2} />
          ) : (
            <span className="size-[5px] rounded-full bg-foreground/55" />
          )}
        </span>
        <span className="relative min-w-0 flex-1 overflow-hidden" aria-live={live ? "polite" : undefined}>
          <AnimatePresence initial={false} mode="popLayout">
            {live ? (
              <motion.span
                key={livePair?.key || "composing"}
                className="flex min-w-0 items-center gap-2 truncate"
                initial={reduced ? false : { opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={reduced ? undefined : { opacity: 0, y: -4 }}
                transition={swap}
                data-testid={livePair ? "tool-row" : undefined}
              >
                <LiveLabel pair={livePair} t={copy.transcript} detail={liveDetail} />
              </motion.span>
            ) : (
              <motion.span
                key="settled"
                className="flex min-w-0 items-center gap-1.5 truncate text-muted"
                initial={reduced ? false : { opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={swap}
              >
                {settledLabel ? <span>{settledLabel}</span> : null}
                {settledLabel && summary ? <span className="text-muted/50">·</span> : null}
                {summary ? <span className="truncate">{summary}</span> : null}
                {failed > 0 ? (
                  <>
                    <span className="text-muted/50">·</span>
                    <span className="text-danger/90">{copy.transcript.failedCount.replace("{n}", String(failed))}</span>
                  </>
                ) : interrupted ? (
                  <>
                    <span className="text-muted/50">·</span>
                    <span className="text-warning/90">{copy.transcript.toolInterrupted}</span>
                  </>
                ) : null}
              </motion.span>
            )}
          </AnimatePresence>
        </span>
        {live && span ? (
          <span className="shrink-0 tabular-nums text-[12px] text-muted/70">{span}</span>
        ) : null}
        <ChevronRight
          className={cn(
            "size-3 shrink-0 text-muted/50 transition-[transform,color] duration-150 group-hover/head:text-muted",
            open && "rotate-90",
          )}
          aria-hidden
        />
      </button>
      <AnimatePresence initial={false}>
        {open ? (
          <motion.div
            key="body"
            initial={reduced ? false : { height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={reduced ? undefined : { height: 0, opacity: 0 }}
            transition={motionTransition(reduced)}
            className="overflow-hidden"
          >
            <div className={cn("step-rail min-w-0 pb-1 pt-0.5", compact && "pt-1", live && "step-rail-live")}>
              {hidden > 0 ? (
                <button
                  type="button"
                  className="mb-1 h-7 w-full rounded-md px-2 text-left text-[12px] text-muted transition-colors hover:bg-lift/50 hover:text-foreground"
                  data-testid="process-earlier"
                  onClick={() => setShown((n) => n + PROCESS_PAGE)}
                >
                  {copy.transcript.earlierSteps.replace("{n}", String(hidden))}
                </button>
              ) : null}
              {openPairs.map((p) => (
                <ProcessPair key={p.key} pair={p} running={running} compact={compact} highlightFail onOpenChild={onOpenChild} />
              ))}
            </div>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}, (a, b) => (
  // 这些字段和每条 item 的引用都没变时跳过这次渲染。
  a.live === b.live
  && a.running === b.running
  && a.compact === b.compact
  && a.grouped === b.grouped
  && a.liveDetail === b.liveDetail
  && a.onOpenChild === b.onOpenChild
  && a.items.length === b.items.length
  && a.items.every((it, i) => it === b.items[i])
));

/** LiveLabel 没有调用也没有结果时显示思考。kind 为 other 时用工具名填 other。detail 不为 false 时，字节超过 2048 且带 progress 才显示 KB。pair 读取 call、result，为空时不读。t 读取 live。 */
function LiveLabel({ pair, t, detail }: { pair?: ToolPair; t: TranscriptCopy; detail?: boolean }) {
  if (!pair?.call && !pair?.result) {
    return <span className="shimmer-text font-medium">{t.live.think}</span>;
  }
  const item = pair.call || pair.result!;
  const name = toolName(item);
  const kind = toolKind(name);
  const skill = skillCallLabel(pair.call || item);
  const path = skill || toolDetail(pair.call || item) || String(item.payload?.path || "");
  const verb = kind === "other" ? t.live.other.replace("{name}", name) : t.live[kind];
  const bytes = Number(item.payload?.bytes || 0);
  const size = item.payload?.progress && bytes > 2048 ? `${Math.round(bytes / 1024)}KB` : "";
  const extra = detail !== false ? [path || (kind === "other" ? "" : name), size].filter(Boolean).join(" · ") : "";
  return (
    <>
      <span className="shrink-0 shimmer-text font-medium">{verb}</span>
      {extra ? <span className="min-w-0 truncate font-mono text-[12px] text-muted/80">{extra}</span> : null}
    </>
  );
}

/**
 * ProcessPair 有调用或结果就画一条 ToolLine，其余条目交给 ProcessExtra。
 * pair 读取 call、result、extra，传给 pairState。
 * running 传给 pairState，写到 ToolLine 的 running。
 * compact 写到 ToolLine 的 compact。
 * highlightFail 写到 ToolLine 的 highlightFail。
 * onOpenChild 写到 ProcessExtra 的 onOpenChild。
 */
function ProcessPair({
  pair,
  running,
  compact,
  highlightFail,
  onOpenChild,
}: {
  pair: ToolPair;
  running: boolean;
  compact?: boolean;
  highlightFail?: boolean;
  onOpenChild?: (id: string) => void;
}) {
  return (
    <>
      {pair.call || pair.result ? (
        <ToolLine
          item={pair.result || pair.call!}
          call={pair.call}
          result={pair.result}
          pending={pairState(pair, running) === "running"}
          running={running}
          compact={compact}
          highlightFail={highlightFail}
        />
      ) : null}
      {pair.extra.map((it, i) => (
        <ProcessExtra key={`${it.key}:${it.payload?.groupPart || i}`} item={it} compact={compact} onOpenChild={onOpenChild} />
      ))}
    </>
  );
}

/** StepGlyph 传入 icon 就用它。否则 running 用圆点，failed 用叉，interrupted 用虚线圈，其余用实心点。compact 时底色用侧栏。state 与 running、failed、interrupted 比较是否相等。 */
function StepGlyph({ state, compact, icon }: { state: PairState | "note"; compact?: boolean; icon?: ReactNode }) {
  return (
    <span
      className={cn(
        "relative z-[1] grid size-3.5 shrink-0 place-items-center rounded-full",
        compact ? "bg-sidebar" : "bg-background",
      )}
      aria-hidden
    >
      {icon ? icon
        : state === "running" ? <span className="pulse-dot" />
        : state === "failed" ? <X className="size-3 text-danger" strokeWidth={2.4} />
        : state === "interrupted" ? <CircleDashed className="size-3 text-warning" strokeWidth={2.2} />
        : <span className="size-[5px] rounded-full bg-foreground/45" />}
    </span>
  );
}

/**
 * StepRow 点整行切换展开。open 为真且有 children 才放出正文。减少动效时不做高度动画。
 * glyph 作为子节点渲染。
 * label 作为子节点渲染。
 * labelTone 为假时用 text-foreground/85，传给 cn。
 * detail 为真时才继续，作为子节点渲染。
 * meta 作为子节点渲染。
 * onToggle 写到 button 的 onClick。
 * testId 写到 div 的 data-testid。
 * className 传给 cn。
 */
function StepRow({
  glyph,
  label,
  labelTone,
  detail,
  meta,
  open,
  onToggle,
  testId,
  className,
  children,
}: {
  glyph: ReactNode;
  label: string;
  labelTone?: string;
  detail?: string;
  meta?: ReactNode;
  open: boolean;
  onToggle: () => void;
  testId?: string;
  className?: string;
  children?: ReactNode;
}) {
  const reduced = useMotionReduced();
  return (
    <div data-testid={testId} className={cn("step-row -mx-1.5 rounded-md px-1.5", className)}>
      <button
        type="button"
        className="group/row flex w-full min-w-0 items-center gap-2.5 py-1 pr-0.5 text-left text-[13px]"
        aria-expanded={open}
        onClick={onToggle}
      >
        {glyph}
        <span className={cn("shrink-0 font-mono text-[12px] font-medium", labelTone || "text-foreground/85")}>{label}</span>
        {detail ? <span className="min-w-0 truncate font-mono text-[12px] text-muted/75">{detail}</span> : null}
        <span className="ml-auto flex shrink-0 items-center gap-2 tabular-nums text-[11px] text-muted/70">
          {meta}
          <ChevronRight
            className={cn(
              "size-3 text-muted/40 transition-[transform,color] duration-150 group-hover/row:text-muted",
              open && "rotate-90",
            )}
            aria-hidden
          />
        </span>
      </button>
      <AnimatePresence initial={false}>
        {open && children ? (
          <motion.div
            key="body"
            initial={reduced ? false : { height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={reduced ? undefined : { height: 0, opacity: 0 }}
            transition={motionTransition(reduced, DURATION_FAST)}
            className="overflow-hidden"
          >
            <div className="mb-1.5 ml-6 mt-0.5 min-w-0">{children}</div>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

/**
 * CodeSurface 缺省截到 4000。语言猜不出来时交给 guessLang。
 * text 传给 guessLang，写到 CodePreview 的 text。
 * cap 写到 CodePreview 的 cap。
 * dim 写到 CodePreview 的 dim。
 * lang 为假时改用后备值，写到 CodePreview 的 lang。
 */
function CodeSurface({ text, cap = 4000, dim, lang }: { text: string; cap?: number; dim?: boolean; lang?: string }) {
  return (
    <CodePreview
      lang={lang || guessLang(text)}
      text={text}
      cap={cap}
      dim={dim}
      className="mt-0 max-h-56"
    />
  );
}

/** ShellTerminal live 为真时在末尾留一个光标块。text 作为子节点渲染。 */
function ShellTerminal({ text, live }: { text: string; live?: boolean }) {
  return (
    <pre
      data-testid="shell-terminal"
      className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-md border border-border/50 bg-background px-2.5 py-2 font-mono text-[11px] leading-[1.55] text-foreground/90"
    >
      {text}
      {live ? <span className="ml-px inline-block h-[1.1em] w-[0.45em] translate-y-px bg-foreground/70" aria-hidden /> : null}
    </pre>
  );
}

/** TypedToolBody read 用代码面，search 只取前 40 条非空行，web 把地址和正文分开。props 读取 kind、path、text、lang、query、url。 */
function TypedToolBody(props: {
  kind: "read" | "search" | "web";
  text: string;
  path?: string;
  lang?: string;
  query?: string;
  url?: string;
}) {
  if (props.kind === "read") {
    return (
      <div data-testid="tool-card-read">
        {props.path ? <div className="mb-1 truncate font-mono text-[11px] text-muted">{props.path}</div> : null}
        <CodePreview text={props.text} lang={props.lang} />
      </div>
    );
  }
  if (props.kind === "search") {
    const hits = props.text.split(/\r?\n/).filter((l) => l.trim()).slice(0, 40);
    return (
      <div data-testid="tool-card-search" className="space-y-0.5">
        {props.query ? <div className="mb-1 font-mono text-[11px] text-muted">{props.query}</div> : null}
        {hits.map((line, i) => (
          <div key={i} className="truncate font-mono text-[11px] leading-[1.5] text-foreground/85">{line}</div>
        ))}
      </div>
    );
  }
  return (
    <div data-testid="tool-card-web" className="space-y-1">
      {props.url ? <div className="truncate font-mono text-[11px] text-accent">{props.url}</div> : null}
      <div className="whitespace-pre-wrap break-words text-[12px] leading-[1.55] text-foreground/85">{props.text}</div>
    </div>
  );
}

// 以 { 或 [ 开头当 json，以 diff、--- 或 *** Begin Patch 开头当 diff，其余当 text。
function guessLang(text: string): string {
  const t = text.trim();
  if (t.startsWith("{") || t.startsWith("[")) return "json";
  if (t.startsWith("diff ") || t.startsWith("--- ") || t.startsWith("*** Begin Patch")) return "diff";
  return "text";
}

/**
 * ProcessExtra 流式时强制展开。推理、进度、子代理和上下文注入各走一行；对不上就什么都不画。没有 onOpenChild 时，打开子会话会先 dump，再把路径或子会话 id 写入剪贴板。
 * item 读取 delta、payload、type、source，传给 readableExtra，传给 reasoningText，传给 toolElapsedMs。
 * compact 写到 StepGlyph 的 compact。
 */
function ProcessExtra({ item, compact, onOpenChild }: { item: Item; compact?: boolean; onOpenChild?: (id: string) => void }) {
  const copy = useCopy();
  const streaming = !!item.delta || !!item.payload?.streaming;
  const [open, setOpen] = useState(streaming);
  useEffect(() => {
    if (streaming) setOpen(true);
  }, [streaming]);
  const body = readableExtra(item);
  if (item.type === "assistant" && item.payload?.groupPart === "reasoning") {
    const thought = reasoningText(item);
    const ms = toolElapsedMs(item);
    const label = ms > 0
      ? copy.transcript.thoughtFor.replace("{n}", formatElapsed(ms))
      : copy.transcript.thinking;
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<Brain className="size-3 text-muted/70" />} />}
        label={label}
        labelTone="font-sans text-[13px] italic text-muted"
        open={open && !!thought}
        onToggle={() => setOpen((v) => !v)}
        testId="reasoning-block"
      >
        {thought ? (
          <div className="md-reasoning min-w-0 max-w-full text-muted">
            <Markdown text={thought} quiet streaming={streaming} />
          </div>
        ) : null}
      </StepRow>
    );
  }
  if (item.type === "assistant") {
    if (!body && !streaming) return null;
    const preview = body.split(/\r?\n/).find((line) => line.trim())?.slice(0, 80) || "";
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<MessageSquare className="size-3 text-muted/70" />} />}
        label={copy.transcript.progressNote}
        labelTone="font-sans text-[13px] text-muted"
        detail={preview}
        open={open && !!body}
        onToggle={() => setOpen((v) => !v)}
        testId="progress-note"
      >
        {body ? (
          <div className="md-reasoning min-w-0 max-w-full text-muted">
            <Markdown text={body} quiet streaming={streaming} />
          </div>
        ) : null}
      </StepRow>
    );
  }
  if (item.type === "reasoning") {
    const ms = toolElapsedMs(item);
    const label = ms > 0
      ? copy.transcript.thoughtFor.replace("{n}", formatElapsed(ms))
      : copy.transcript.thinking;
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<Brain className="size-3 text-muted/70" />} />}
        label={label}
        labelTone="font-sans text-[13px] italic text-muted"
        open={open && !!body}
        onToggle={() => setOpen((v) => !v)}
        testId="reasoning-block"
      >
        {body ? (
          <div className="md-reasoning min-w-0 max-w-full text-muted">
            <Markdown text={body} quiet streaming={streaming} />
          </div>
        ) : null}
      </StepRow>
    );
  }
  if (item.type === "subagent") {
    const child = String(item.payload?.child || "").trim();
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<Bot className="size-3 text-muted/70" />} />}
        label={copy.transcript.subagent}
        labelTone="font-sans text-[13px] text-muted"
        detail={body.split(/\r?\n/)[0]?.slice(0, 120)}
        open={open && !!(body || child)}
        onToggle={() => setOpen((v) => !v)}
      >
        {body ? <div className="min-w-0 max-w-full whitespace-pre-wrap break-words text-[13px] leading-6 text-muted">{body}</div> : null}
        {child ? (
          <button
            type="button"
            className="mt-1 text-[12px] text-muted underline decoration-muted/40 underline-offset-2 hover:text-foreground"
            data-testid="open-child"
            onClick={() => {
              if (onOpenChild) {
                onOpenChild(child);
                return;
              }
              void api.dumpSession(child).then(async (dump) => {
                const text = dump.path || child;
                await writeClipboard(text);
              });
            }}
          >
            {copy.transcript.openChild}
            <span className="ml-1 font-mono text-[11px] text-muted">{child}</span>
          </button>
        ) : null}
      </StepRow>
    );
  }
  if (item.type === "context_injection") {
    const skill = item.source === "skill" || String(item.payload?.name || "") === "load_skill";
    const name = String(item.payload?.name || "").trim();
    const label = skill
      ? (name && name !== "load_skill" ? name : copy.transcript.live.skill)
      : copy.transcript.live.skill;
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<AtSign className="size-3 text-muted/70" />} />}
        label={label}
        labelTone="font-sans text-[13px] text-muted"
        open={open && !!body}
        onToggle={() => setOpen((v) => !v)}
      >
        <CodeSurface text={body} cap={1200} />
      </StepRow>
    );
  }
  return null;
}

// 摘要里种类的先后。最多只用前三种。
const TALLY_ORDER: ToolKind[] = ["edit", "run", "read", "search", "web", "list", "plan", "skill", "other"];

/**
 * summarize 有计数就取前三种拼起来。没有计数时依次退到推理耗时、进度条数、子代理、技能，最后是进行中文案。一条用单数文案。
 * items 调用 find，调用 some，传给 processKindTally，传给 processHasReasoning，传给 processProgressCount。
 * t 读取 tally、thoughtFor、thinking、progressNote、progressNotes、subagent 等。
 */
function summarize(items: Item[], t: TranscriptCopy): string {
  const tally = processKindTally(items);
  const parts: string[] = [];
  for (const k of TALLY_ORDER) {
    const n = tally[k];
    if (!n) continue;
    const [one, many] = t.tally[k];
    parts.push(n === 1 ? one : many.replace("{n}", String(n)));
  }
  if (parts.length) return parts.slice(0, 3).join(" · ");
  if (processHasReasoning(items)) {
    const reasoning = items.find((it) => it.type === "reasoning");
    const ms = reasoning ? toolElapsedMs(reasoning) : 0;
    return ms > 0 ? t.thoughtFor.replace("{n}", formatElapsed(ms)) : t.thinking;
  }
  const notes = processProgressCount(items);
  if (notes) return notes === 1 ? t.progressNote : t.progressNotes.replace("{n}", String(notes));
  if (items.some((it) => it.type === "subagent")) return t.subagent;
  if (items.some((it) => it.type === "context_injection")) return t.live.skill;
  return t.working;
}

/** readableExtra 推理取 reasoningText。其余按 text、summary、prompt、note、question 取第一条非空。item 读取 payload、type、text，传给 reasoningText。 */
function readableExtra(item: Item): string {
  if (item.payload?.groupPart === "reasoning" || item.type === "reasoning") {
    return reasoningText(item);
  }
  const p = item.payload || {};
  const bits = [item.text, p.summary, p.prompt, p.note, p.question];
  for (const b of bits) {
    const s = String(b || "").trim();
    if (s) return s;
  }
  return "";
}

/** argsPreview 没有调用，或正文是空、{}、null，就返回空字符串。call 为假时返回空字符串；传给 formatToolBody。 */
function argsPreview(call?: Item): string {
  if (!call) return "";
  const raw = formatToolBody(call).trim();
  if (!raw || raw === "{}" || raw === "null") return "";
  return raw;
}

/**
 * ToolLine 在 pending 且 running 时自动展开。pending 但不在跑算中断，结果失败算失败，否则算完成。read、search、web 不画输入。shell 用终端，补丁类输出用 diff。
 * item 作为后备值，传给 toolName。
 * call 为假时用 source，为假时用 item，作为后备值，传给 toolDetail，传给 argsPreview，传给 toolArgs。
 * result 为假时用 call，为假时用 item，读取 delta、payload，为空时不读，传给 toolElapsedMs，传给 isToolFailed，传给 formatToolBody，传给 toolArgs。
 * compact 写到 StepGlyph 的 compact。
 * highlightFail 为真时才继续。
 */
export function ToolLine({
  item,
  call,
  result,
  pending,
  running = false,
  compact,
  highlightFail,
}: {
  item: Item;
  call?: Item;
  result?: Item;
  pending?: boolean;
  running?: boolean;
  compact?: boolean;
  highlightFail?: boolean;
}) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const [fullOut, setFullOut] = useState(false);
  const name = toolName(item);
  const source = result || call || item;
  const detail = toolDetail(call || source);
  const ms = toolElapsedMs(result || item);
  const elapsed = formatElapsed(ms);
  const streamingOut = !!(result?.delta || result?.payload?.delta);
  useEffect(() => {
    if (pending && running) setOpen(true);
  }, [pending, running]);
  const state: PairState = pending
    ? (running ? "running" : "interrupted")
    : isToolFailed(result) ? "failed" : "done";
  const status = state === "running"
    ? copy.transcript.toolRunning
    : state === "failed"
      ? copy.transcript.toolFailed
      : state === "interrupted"
        ? copy.transcript.toolInterrupted
        : "";
  const input = argsPreview(call);
  const rawOut = result ? formatToolBody(result) : "";
  const clipped = fullOut ? { text: rawOut, hidden: 0 } : clipPreview(rawOut);
  const output = clipped.text;
  const hasBody = !!(input || output);
  const path = String(toolArgs(call || item).path || toolArgs(result || item).path || "");
  const pathLang = langFromPath(path);
  const kind = toolKind(name);
  const outLang = name === "read_file" || name === "write_file" ? pathLang : guessLang(output);
  const inLang = name === "write_file" || name === "str_replace" || name === "apply_patch" ? (name === "apply_patch" ? "diff" : pathLang || "json") : "json";
  const query = String(toolArgs(call || item).pattern || toolArgs(call || item).query || toolArgs(call || item).q || "");
  const url = String(toolArgs(call || item).url || toolArgs(call || item).uri || "");
  return (
    <StepRow
      testId="tool-row"
      className={cn(highlightFail && state === "failed" && "bg-danger/8")}
      glyph={<StepGlyph state={state} compact={compact} />}
      label={name}
      labelTone={
        state === "failed" ? "text-danger"
          : state === "interrupted" ? "text-muted"
          : "text-foreground/85"
      }
      detail={detail}
      meta={(
        <>
          {elapsed ? <span>{elapsed}</span> : null}
          {status ? (
            <span
              className={cn(
                state === "failed" && "text-danger/85",
                state === "interrupted" && "text-warning/90",
                state === "running" && "text-foreground/70",
              )}
            >
              {status}
            </span>
          ) : null}
        </>
      )}
      open={open && hasBody}
      onToggle={() => setOpen((v) => !v)}
    >
      <div className="space-y-1">
        {input && kind !== "read" && kind !== "search" && kind !== "web" ? (
          <CodeSurface text={input} cap={800} dim={!!output} lang={inLang} />
        ) : null}
        {output ? (
          kind === "run" || name === "shell" ? (
            <ShellTerminal text={output} live={streamingOut || !!pending} />
          ) : kind === "read" ? (
            <TypedToolBody kind="read" path={path} text={output} lang={outLang} />
          ) : kind === "search" ? (
            <TypedToolBody kind="search" query={query} path={path} text={output} />
          ) : kind === "web" ? (
            <TypedToolBody kind="web" url={url} text={output} />
          ) : kind === "edit" && (name === "str_replace" || name === "apply_patch" || guessLang(output) === "diff") ? (
            <CodeSurface text={output} lang="diff" />
          ) : (
            <CodeSurface text={output} lang={outLang} />
          )
        ) : null}
        {clipped.hidden > 0 ? (
          <button
            type="button"
            className="h-6 text-left text-[11px] text-muted hover:text-foreground"
            data-testid="tool-more-lines"
            onClick={() => setFullOut(true)}
          >
            {copy.transcript.moreLines.replace("{n}", String(clipped.hidden))}
          </button>
        ) : null}
      </div>
    </StepRow>
  );
}
