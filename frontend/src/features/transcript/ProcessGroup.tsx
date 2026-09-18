import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { ChevronRight, Loader2 } from "lucide-react";
import { Markdown } from "../../lib/markdown";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { DURATION_FAST, motionTransition, useMotionReduced } from "../../lib/motion";
import {
  formatElapsed,
  formatToolBody,
  isToolFailed,
  toolDetail,
  toolElapsedMs,
  toolName,
} from "../../lib/tool-summary";
import type { Item } from "../../lib/protocol";
import {
  pairTools,
  processElapsedMs,
  processFailedCount,
  processToolPairs,
  type ToolPair,
} from "../../lib/transcript-layout";

export function WorkingLine() {
  const copy = useCopy();
  return (
    <div
      className="flex h-7 items-center gap-2 text-[12.5px] text-muted"
      role="status"
      aria-live="polite"
      data-testid="working-line"
    >
      <span className="thinking-dots" aria-hidden>
        <i /><i /><i />
      </span>
      {copy.transcript.working}
    </div>
  );
}

export function ProcessGroup({
  items,
  live,
  compact,
}: {
  items: Item[];
  live: boolean;
  compact?: boolean;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const transition = motionTransition(reduced, DURATION_FAST);
  const [open, setOpen] = useState(false);
  const pairs = pairTools(items);
  const tools = processToolPairs(items);
  const pending = tools.filter((p) => p.call && !p.result);
  const current = pending[pending.length - 1] || tools[tools.length - 1];
  const failed = processFailedCount(items);
  const elapsed = formatElapsed(processElapsedMs(items));
  const summary = processSummaryLabel(items, tools.length, copy.transcript);

  return (
    <div className="u-chrome min-w-0" data-testid="process-group" data-live={live ? "true" : "false"}>
      <div className="min-h-7">
        <AnimatePresence initial={false} mode="popLayout">
          {live ? (
            <motion.div
              key={current?.key || "live"}
              initial={reduced ? false : { opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={reduced ? undefined : { opacity: 0 }}
              transition={transition}
            >
              <LiveActivity pair={current} />
            </motion.div>
          ) : (
            <motion.div
              key="settled"
              initial={reduced ? false : { opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={reduced ? undefined : { opacity: 0 }}
              transition={transition}
            >
              <button
                type="button"
                className="flex h-7 w-full min-w-0 items-center gap-2 text-left text-[12.5px] text-muted hover:text-foreground"
                data-testid="process-summary"
                aria-expanded={open}
                onClick={() => setOpen((v) => !v)}
              >
                {failed > 0 ? (
                  <span className="size-1.5 shrink-0 rounded-full bg-danger" aria-label={copy.transcript.toolFailed} />
                ) : (
                  <span className="size-1.5 shrink-0 rounded-full bg-foreground/70" aria-hidden />
                )}
                <span className="min-w-0 truncate">
                  {summary}
                  {elapsed ? <span className="tabular-nums text-muted/70"> · {elapsed}</span> : null}
                </span>
                <ChevronRight className={cn("ml-auto size-3 shrink-0 transition-transform duration-150", open && "rotate-90")} />
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
                    <div className={cn("min-w-0 space-y-0.5 pb-0.5 pt-0.5", compact && "pt-1")}>
                      {pairs.map((p) => (
                        <ProcessPair key={p.key} pair={p} highlightFail />
                      ))}
                    </div>
                  </motion.div>
                ) : null}
              </AnimatePresence>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

function LiveActivity({ pair }: { pair?: ToolPair }) {
  if (!pair?.call && !pair?.result) return <WorkingLine />;
  const item = pair.call || pair.result!;
  const detail = toolDetail(pair.call || item);
  const name = toolName(item);
  return (
    <div
      className="flex h-7 min-w-0 items-center gap-2 text-[12.5px] text-muted"
      data-testid="tool-row"
      role="status"
      aria-live="polite"
    >
      <Loader2 className="size-3 shrink-0 animate-spin text-muted" aria-hidden />
      <span className="shrink-0 font-medium text-foreground/85">{name}</span>
      {detail ? <span className="min-w-0 truncate font-mono text-[11.5px] text-muted/80">{detail}</span> : null}
    </div>
  );
}

function ProcessPair({ pair, highlightFail }: { pair: ToolPair; highlightFail?: boolean }) {
  return (
    <>
      {pair.call || pair.result ? (
        <ToolLine
          item={pair.result || pair.call!}
          call={pair.call}
          result={pair.result}
          pending={!!pair.call && !pair.result}
          highlightFail={highlightFail}
        />
      ) : null}
      {pair.extra.map((it) => (
        <ProcessExtra key={it.key} item={it} />
      ))}
    </>
  );
}

function ProcessExtra({ item }: { item: Item }) {
  const copy = useCopy();
  if (item.type === "reasoning") {
    const ms = toolElapsedMs(item);
    const label = ms > 0
      ? copy.transcript.thoughtFor.replace("{n}", formatElapsed(ms))
      : copy.transcript.thinking;
    const body = readableExtra(item);
    if (!body) {
      return <div className="py-0.5 text-[12.5px] text-muted">{label}</div>;
    }
    return (
      <details className="text-[12.5px] text-muted">
        <summary className="cursor-pointer select-none hover:text-foreground">{label}</summary>
        <div className="mt-1.5 max-w-[72ch] text-[13px] leading-6 text-muted">
          <Markdown text={body} />
        </div>
      </details>
    );
  }
  if (item.type === "subagent") {
    const body = readableExtra(item);
    return (
      <details className="text-[12.5px] text-muted">
        <summary className="cursor-pointer select-none hover:text-foreground">{copy.transcript.subagent}</summary>
        {body ? (
          <div className="mt-1.5 max-w-[72ch] whitespace-pre-wrap text-[13px] leading-6 text-muted">{body}</div>
        ) : null}
      </details>
    );
  }
  if (item.type === "context_injection") {
    const body = readableExtra(item);
    return (
      <details className="text-[12.5px] text-muted">
        <summary className="cursor-pointer select-none hover:text-foreground">{copy.transcript.mention}</summary>
        {body ? (
          <pre className="mt-1.5 max-h-40 overflow-auto whitespace-pre-wrap font-mono text-[11px] leading-4">{body.slice(0, 1200)}</pre>
        ) : null}
      </details>
    );
  }
  return null;
}

function processSummaryLabel(
  items: Item[],
  toolCount: number,
  t: { toolsUsed: string; thinking: string; thoughtFor: string; subagent: string; mention: string },
): string {
  if (toolCount > 0) return t.toolsUsed.replace("{n}", String(toolCount));
  const reasoning = items.find((it) => it.type === "reasoning");
  if (reasoning) {
    const ms = toolElapsedMs(reasoning);
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

export function ToolLine({
  item,
  call,
  result,
  pending,
  highlightFail,
}: {
  item: Item;
  call?: Item;
  result?: Item;
  pending?: boolean;
  highlightFail?: boolean;
}) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const name = toolName(item);
  const source = result || call || item;
  const detail = toolDetail(call || source);
  const ms = toolElapsedMs(result || item);
  const elapsed = formatElapsed(ms);
  const failed = isToolFailed(result);
  const status = pending
    ? copy.transcript.toolRunning
    : failed
      ? copy.transcript.toolFailed
      : copy.transcript.toolDone;
  const body = formatToolBody(result || call || item);
  return (
    <div
      data-testid="tool-row"
      className={cn(highlightFail && failed && "rounded-md bg-danger/8 px-1")}
    >
      <button
        type="button"
        className="flex w-full min-w-0 items-center gap-2 py-0.5 text-left text-[12.5px] text-muted hover:text-foreground"
        onClick={() => setOpen((v) => !v)}
      >
        {pending ? (
          <Loader2 className="size-3 shrink-0 animate-spin text-muted" aria-hidden />
        ) : (
          <span className={cn("size-1.5 shrink-0 rounded-full", failed ? "bg-danger" : "bg-foreground/70")} aria-hidden />
        )}
        <span className={cn("shrink-0 font-medium", failed ? "text-danger" : "text-foreground/85")}>{name}</span>
        {detail ? <span className="min-w-0 truncate font-mono text-[11.5px] text-muted/80">{detail}</span> : null}
        <span className="ml-auto flex shrink-0 items-center gap-1.5 tabular-nums text-[11px] text-muted/70">
          {elapsed ? <span>{elapsed}</span> : null}
          <span>{status}</span>
          <ChevronRight className={cn("size-3 transition-transform duration-150", open && "rotate-90")} />
        </span>
      </button>
      {open ? (
        <pre className="mt-0.5 max-h-40 overflow-auto whitespace-pre-wrap rounded-lg bg-lift/40 px-2.5 py-2 font-mono text-[11px] leading-4 text-muted">
          {body.slice(0, 4000)}
        </pre>
      ) : null}
    </div>
  );
}
