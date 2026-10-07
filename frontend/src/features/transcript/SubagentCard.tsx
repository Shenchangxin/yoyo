import { memo, useEffect, useMemo, useState } from "react";
import { Bot, ChevronRight, CircleDashed, X } from "lucide-react";
import { Markdown } from "../../lib/markdown";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { formatSpan, itemTimeMs } from "../../lib/tool-summary";
import type { Item } from "../../lib/protocol";
import { mergeItem, subscribeSession } from "../../lib/stream";
import { processStartMs } from "../../lib/transcript-layout";
import { ProcessGroup, useNow } from "./ProcessGroup";

function profileLabel(profile: string, t: { subagentExplore: string; subagentImplement: string; subagentQA: string }): string {
  switch (profile.toLowerCase()) {
    case "explore":
      return t.subagentExplore;
    case "qa":
      return t.subagentQA;
    case "implement":
      return t.subagentImplement;
    default:
      return "";
  }
}

function latestPayload(items: Item[]): Record<string, unknown> {
  const last = items[items.length - 1];
  return (last?.payload || {}) as Record<string, unknown>;
}

function promptTitle(items: Item[]): string {
  for (const it of items) {
    const p = it.payload || {};
    const raw = String(p.prompt || p.summary || it.text || "").trim();
    if (raw) return raw.split(/\r?\n/).find((l) => l.trim())?.trim() || raw;
  }
  return "";
}

function summaryText(items: Item[]): string {
  for (let i = items.length - 1; i >= 0; i--) {
    const p = items[i].payload || {};
    const s = String(p.summary || "").trim();
    if (s) return s;
    const err = String(p.error || "").trim();
    if (err) return err;
  }
  return "";
}

function subagentFailed(items: Item[]): boolean {
  const last = items[items.length - 1];
  if (!last) return false;
  if (last.payload?.error) return true;
  if (String(last.payload?.phase || "") === "complete" && last.payload?.ok === false) return true;
  return false;
}

export const SubagentCard = memo(function SubagentCard({
  items,
  child,
  live,
  running,
  compact,
}: {
  items: Item[];
  child: string;
  live: boolean;
  running: boolean;
  compact?: boolean;
}) {
  const copy = useCopy();
  const [userOpen, setUserOpen] = useState<boolean | null>(null);
  const open = userOpen ?? live;
  const [childItems, setChildItems] = useState<Item[]>([]);
  const payload = latestPayload(items);
  const profile = String(payload.profile || items[0]?.payload?.profile || "").trim();
  const title = promptTitle(items) || copy.transcript.subagent;
  const summary = summaryText(items);
  const failed = subagentFailed(items);
  const start = processStartMs(items);
  const now = useNow(live);
  const lastMs = itemTimeMs(items[items.length - 1]);
  const elapsed = formatSpan(live ? (start ? Math.max(0, now - start) : 0) : (start && lastMs > start ? lastMs - start : 0));
  const badge = profileLabel(profile, copy.transcript) || copy.transcript.subagent;
  const status = live
    ? copy.transcript.subagentRunning
    : failed
      ? copy.transcript.subagentFailed
      : copy.transcript.subagentDone;
  const work = useMemo(
    () => childItems.filter((it) => it.type === "reasoning" || it.type === "tool_call" || it.type === "tool_result" || it.type === "subagent"),
    [childItems],
  );
  const letters = useMemo(
    () => childItems.filter((it) => it.type === "assistant" && it.text.trim()),
    [childItems],
  );

  useEffect(() => {
    if (!open || !child) return;
    let alive = true;
    const stop = subscribeSession(
      child,
      (item) => {
        if (!alive) return;
        setChildItems((prev) => mergeItem(prev, item));
      },
      (seed) => {
        if (!alive) return;
        setChildItems(seed);
      },
    );
    return () => {
      alive = false;
      stop();
    };
  }, [open, child]);

  return (
    <div className="u-chrome min-w-0" data-testid="subagent-card" data-live={live ? "true" : "false"} data-child={child}>
      <button
        type="button"
        className="group/head flex w-full min-w-0 items-center gap-2.5 rounded-md py-0.5 text-left text-[13px]"
        aria-expanded={open}
        title={open ? copy.transcript.subagentHideWork : copy.transcript.subagentShowWork}
        onClick={() => setUserOpen((v) => !(v ?? live))}
      >
        <span className="grid size-[22px] shrink-0 place-items-center" aria-hidden>
          {live ? (
            <span className="pulse-dot" />
          ) : failed ? (
            <X className="size-3 text-danger" strokeWidth={2.4} />
          ) : (
            <Bot className="size-3.5 text-muted" />
          )}
        </span>
        <span className="min-w-0 flex-1 truncate">
          <span className={cn("font-medium", live && "shimmer-text")}>{badge}</span>
          <span className="ml-2 font-normal text-muted">{title}</span>
        </span>
        <span className="ml-auto flex shrink-0 items-center gap-2 tabular-nums text-[11px] text-muted/70">
          {elapsed ? <span>{elapsed}</span> : null}
          <span className={cn(failed && "text-danger/90", live && "text-foreground/70")}>{status}</span>
          <ChevronRight
            className={cn(
              "size-3 text-muted/50 transition-[transform,color] duration-150 group-hover/head:text-muted",
              open && "rotate-90",
            )}
            aria-hidden
          />
        </span>
      </button>
      {open ? (
        <div className="step-rail mb-1 ml-1 min-w-0 pb-1 pt-0.5">
          {summary && !live ? (
            <div className="mb-1.5 min-w-0 whitespace-pre-wrap break-words text-[13px] leading-6 text-foreground/85">
              {summary.replace(/^SUBAGENT_SUMMARY:\s*/i, "")}
            </div>
          ) : null}
          {work.length ? (
            <ProcessGroup items={work} live={live} running={running || live} compact={compact} />
          ) : live ? (
            <div className="flex h-7 items-center gap-2.5 text-[13px] text-muted">
              <CircleDashed className="size-3.5" aria-hidden />
              {copy.transcript.subagentRunning}
            </div>
          ) : null}
          {letters.map((it) => (
            <div key={it.key} className="assistant-prose mt-1 min-w-0 text-muted">
              <Markdown text={it.text} quiet />
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
});
