import { useEffect, useState, type ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { AtSign, Bot, Brain, ChevronRight, CircleDashed, X } from "lucide-react";
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
  toolDetail,
  toolElapsedMs,
  toolKind,
  toolName,
  type ToolKind,
} from "../../lib/tool-summary";
import type { Item } from "../../lib/protocol";
import {
  pairState,
  pairTools,
  processFailedCount,
  processHasReasoning,
  processKindTally,
  processSpanMs,
  processStartMs,
  processToolPairs,
  type PairState,
  type ToolPair,
} from "../../lib/transcript-layout";

type TranscriptCopy = Copy["transcript"];

/** One-second clock that only ticks while something is live. */
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
 * The turn is alive but no tool is in flight: model is composing. `since`
 * is the last user message so the clock does not restart at every batch
 * (Codex falls back to the user timestamp the same way).
 */
export function WorkingLine({ since, label }: { since?: number; label?: string }) {
  const copy = useCopy();
  const now = useNow(!!since);
  const elapsed = since && now > since ? formatSpan(now - since) : "";
  return (
    <div
      className="flex h-7 items-center gap-2.5 text-[12.5px]"
      role="status"
      aria-live="polite"
      data-testid="working-line"
    >
      <span className="grid size-3.5 shrink-0 place-items-center" aria-hidden>
        <span className="pulse-dot" />
      </span>
      <span className="shimmer-text font-medium">{label || copy.transcript.working}</span>
      {elapsed ? <span className="tabular-nums text-[11.5px] text-muted/70">{elapsed}</span> : null}
    </div>
  );
}

export function ProcessGroup({
  items,
  live,
  running,
  compact,
}: {
  items: Item[];
  live: boolean;
  running: boolean;
  compact?: boolean;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const swap = motionTransition(reduced, DURATION_FAST);
  const [open, setOpen] = useState(false);
  const pairs = pairTools(items);
  const tools = processToolPairs(items);
  const pending = tools.filter((p) => pairState(p, running) === "running");
  const current = live ? pending[pending.length - 1] : undefined;
  const failed = processFailedCount(items);
  const interrupted = !running && tools.some((p) => pairState(p, false) === "interrupted");
  const start = processStartMs(items);
  const now = useNow(live);
  const spanMs = live ? (start ? Math.max(0, now - start) : 0) : processSpanMs(items);
  const span = formatSpan(spanMs);
  const summary = summarize(items, copy.transcript);
  const settledLabel = span ? copy.transcript.workedFor.replace("{n}", span) : "";

  return (
    <div className="u-chrome min-w-0" data-testid="process-group" data-live={live ? "true" : "false"}>
      <button
        type="button"
        className="group/head flex h-7 w-full min-w-0 items-center gap-2.5 rounded-md text-left text-[12.5px]"
        data-testid={live ? undefined : "process-summary"}
        aria-expanded={open}
        title={open ? copy.transcript.hideSteps : copy.transcript.showSteps}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="grid size-3.5 shrink-0 place-items-center" aria-hidden>
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
                key={current?.key || "composing"}
                className="flex min-w-0 items-center gap-2 truncate"
                initial={reduced ? false : { opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={reduced ? undefined : { opacity: 0, y: -4 }}
                transition={swap}
                data-testid={current ? "tool-row" : undefined}
              >
                <LiveLabel pair={current} t={copy.transcript} />
              </motion.span>
            ) : (
              <motion.span
                key="settled"
                className="flex min-w-0 items-center gap-1.5 truncate text-muted"
                initial={reduced ? false : { opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={swap}
              >
                {settledLabel ? <span className="text-foreground/85">{settledLabel}</span> : null}
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
          <span className="shrink-0 tabular-nums text-[11.5px] text-muted/70">{span}</span>
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
            <div className={cn("step-rail min-w-0 pb-1 pt-0.5", compact && "pt-1")}>
              {pairs.map((p) => (
                <ProcessPair key={p.key} pair={p} running={running} compact={compact} highlightFail />
              ))}
            </div>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

function LiveLabel({ pair, t }: { pair?: ToolPair; t: TranscriptCopy }) {
  if (!pair?.call && !pair?.result) {
    return <span className="shimmer-text font-medium">{t.live.think}</span>;
  }
  const item = pair.call || pair.result!;
  const name = toolName(item);
  const kind = toolKind(name);
  const detail = toolDetail(pair.call || item);
  const verb = kind === "other" ? t.live.other.replace("{name}", name) : t.live[kind];
  return (
    <>
      <span className="shrink-0 shimmer-text font-medium">{verb}</span>
      <span className="min-w-0 truncate font-mono text-[11.5px] text-muted/80">{detail || (kind === "other" ? "" : name)}</span>
    </>
  );
}

function ProcessPair({
  pair,
  running,
  compact,
  highlightFail,
}: {
  pair: ToolPair;
  running: boolean;
  compact?: boolean;
  highlightFail?: boolean;
}) {
  return (
    <>
      {pair.call || pair.result ? (
        <ToolLine
          item={pair.result || pair.call!}
          call={pair.call}
          result={pair.result}
          pending={!!pair.call && !pair.result}
          running={running}
          compact={compact}
          highlightFail={highlightFail}
        />
      ) : null}
      {pair.extra.map((it) => (
        <ProcessExtra key={it.key} item={it} compact={compact} />
      ))}
    </>
  );
}

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
        className="group/row flex w-full min-w-0 items-center gap-2.5 py-1 pr-0.5 text-left text-[12.5px]"
        aria-expanded={open}
        onClick={onToggle}
      >
        {glyph}
        <span className={cn("shrink-0 font-mono text-[12px] font-medium", labelTone || "text-foreground/85")}>{label}</span>
        {detail ? <span className="min-w-0 truncate font-mono text-[11.5px] text-muted/75">{detail}</span> : null}
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

function CodeSurface({ text, cap = 4000, dim }: { text: string; cap?: number; dim?: boolean }) {
  if (!text.trim()) return null;
  return (
    <pre
      className={cn(
        "max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[11px] leading-[1.55]",
        dim ? "text-muted/70" : "text-muted",
      )}
    >
      {text.slice(0, cap)}
    </pre>
  );
}

function ProcessExtra({ item, compact }: { item: Item; compact?: boolean }) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const body = readableExtra(item);
  if (item.type === "reasoning") {
    const ms = toolElapsedMs(item);
    const label = ms > 0
      ? copy.transcript.thoughtFor.replace("{n}", formatElapsed(ms))
      : copy.transcript.thinking;
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<Brain className="size-3 text-muted/70" />} />}
        label={label}
        labelTone="font-sans text-[12.5px] text-muted"
        open={open && !!body}
        onToggle={() => setOpen((v) => !v)}
      >
        {body ? (
          <div className="max-w-[72ch] text-[13px] leading-6 text-muted">
            <Markdown text={body} />
          </div>
        ) : null}
      </StepRow>
    );
  }
  if (item.type === "subagent") {
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<Bot className="size-3 text-muted/70" />} />}
        label={copy.transcript.subagent}
        labelTone="font-sans text-[12.5px] text-muted"
        detail={body.split(/\r?\n/)[0]?.slice(0, 120)}
        open={open && !!body}
        onToggle={() => setOpen((v) => !v)}
      >
        {body ? <div className="max-w-[72ch] whitespace-pre-wrap text-[13px] leading-6 text-muted">{body}</div> : null}
      </StepRow>
    );
  }
  if (item.type === "context_injection") {
    return (
      <StepRow
        glyph={<StepGlyph state="note" compact={compact} icon={<AtSign className="size-3 text-muted/70" />} />}
        label={copy.transcript.mention}
        labelTone="font-sans text-[12.5px] text-muted"
        open={open && !!body}
        onToggle={() => setOpen((v) => !v)}
      >
        <CodeSurface text={body} cap={1200} />
      </StepRow>
    );
  }
  return null;
}

const TALLY_ORDER: ToolKind[] = ["edit", "run", "read", "search", "web", "list", "plan", "skill", "other"];

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
  if (items.some((it) => it.type === "subagent")) return t.subagent;
  return t.mention;
}

function readableExtra(item: Item): string {
  const p = item.payload || {};
  const bits = [item.text, p.summary, p.prompt, p.note, p.question];
  for (const b of bits) {
    const s = String(b || "").trim();
    if (s) return s;
  }
  return "";
}

function argsPreview(call?: Item): string {
  if (!call) return "";
  const raw = formatToolBody(call).trim();
  if (!raw || raw === "{}" || raw === "null") return "";
  return raw;
}

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
  const name = toolName(item);
  const source = result || call || item;
  const detail = toolDetail(call || source);
  const ms = toolElapsedMs(result || item);
  const elapsed = formatElapsed(ms);
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
  const output = result ? formatToolBody(result) : "";
  const hasBody = !!(input || output);
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
        {input ? <CodeSurface text={input} cap={800} dim={!!output} /> : null}
        {output ? <CodeSurface text={output} /> : null}
      </div>
    </StepRow>
  );
}
