import { useEffect, useRef, useState, type ReactNode } from "react";
import { Check, ChevronRight, Copy, Loader2, ShieldAlert, Wrench } from "lucide-react";
import { toast } from "sonner";
import { VList, type VListHandle } from "virtua";
import { Markdown } from "../lib/markdown";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { writeClipboard } from "../lib/clipboard";
import { DiffBlock } from "../lib/split-diff";
import type { Approval, Item } from "../lib/protocol";
import { classifyItem, errorCopy } from "../lib/error";

type AgentPart =
  | { key: string; kind: "item"; item: Item }
  | { key: string; kind: "tools"; items: Item[] };

type LayoutRow =
  | { key: string; kind: "user"; item: Item }
  | { key: string; kind: "agent"; parts: AgentPart[]; copyText: string }
  | { key: string; kind: "solo"; item: Item };

const INNER = "w-full min-w-0 px-5 sm:px-8 lg:px-10";

function isToolish(it: Item): boolean {
  return it.type === "tool_call" || it.type === "tool_result";
}

function isAgentItem(it: Item): boolean {
  return (
    it.type === "assistant" ||
    it.type === "reasoning" ||
    it.type === "subagent" ||
    isToolish(it) ||
    it.type === "context_injection"
  );
}

function agentCopyText(items: Item[]): string {
  return items
    .filter((it) => it.type === "assistant")
    .map((it) => it.text.trim())
    .filter(Boolean)
    .join("\n\n");
}

function layoutAgentParts(items: Item[]): AgentPart[] {
  const parts: AgentPart[] = [];
  let batch: Item[] = [];
  const flush = () => {
    if (!batch.length) return;
    parts.push({ key: `tools:${batch[0].key}`, kind: "tools", items: batch });
    batch = [];
  };
  for (const it of items) {
    if (isToolish(it) || (it.type === "context_injection" && batch.length > 0)) {
      batch.push(it);
      continue;
    }
    flush();
    parts.push({ key: it.key, kind: "item", item: it });
  }
  flush();
  return parts;
}

function layoutRows(items: Item[]): LayoutRow[] {
  const rows: LayoutRow[] = [];
  let agent: Item[] = [];
  const flushAgent = () => {
    if (!agent.length) return;
    rows.push({
      key: `turn:${agent[0].key}`,
      kind: "agent",
      parts: layoutAgentParts(agent),
      copyText: agentCopyText(agent),
    });
    agent = [];
  };
  for (const it of items) {
    if (it.type === "user" && it.source !== "steer") {
      flushAgent();
      rows.push({ key: it.key, kind: "user", item: it });
      continue;
    }
    if (isAgentItem(it)) {
      agent.push(it);
      continue;
    }
    flushAgent();
    rows.push({ key: it.key, kind: "solo", item: it });
  }
  flushAgent();
  return rows;
}

function pairTools(items: Item[]): { key: string; call?: Item; result?: Item; extra: Item[] }[] {
  const order: string[] = [];
  const map = new Map<string, { key: string; call?: Item; result?: Item; extra: Item[] }>();
  for (const it of items) {
    const id = String(it.payload?.id || (it.type === "context_injection" ? it.key : "") || it.key);
    if (!map.has(id)) {
      map.set(id, { key: id, extra: [] });
      order.push(id);
    }
    const row = map.get(id)!;
    if (it.type === "tool_call") row.call = it;
    else if (it.type === "tool_result") row.result = it;
    else row.extra.push(it);
  }
  return order.map((id) => map.get(id)!);
}

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

function isUserRow(row: LayoutRow): boolean {
  return row.kind === "user";
}

function rowSpace(rows: LayoutRow[], i: number): string {
  const row = rows[i];
  const prev = rows[i - 1];
  if (!prev) return "pt-0";
  if (isUserRow(row)) return "pt-8";
  if (row.kind === "agent" && prev.kind === "user") return "pt-3";
  if (row.kind === "agent") return "pt-4";
  return "pt-4";
}

function partSpace(parts: AgentPart[], i: number): string {
  if (i === 0) return "";
  const prev = parts[i - 1];
  const cur = parts[i];
  if (cur.kind === "tools") return "pt-2";
  if (cur.kind === "item" && cur.item.type === "assistant" && prev.kind === "tools") return "pt-3";
  if (cur.kind === "item" && cur.item.type === "assistant") return "pt-1.5";
  return "pt-3";
}

export function Transcript(props: {
  items: Item[];
  approvals: Approval[];
  running: boolean;
  needsSetup?: boolean;
  compact?: boolean;
  onResolve: (id: string, decision: string) => void;
  onPrompt?: (text: string) => void;
  onSetup?: () => void;
  onOpenReview?: () => void;
  onRetry?: () => void;
}) {
  const copy = useCopy();
  const pad = props.compact ? "w-full min-w-0 px-3" : INNER;
  const scroller = useRef<HTMLDivElement>(null);
  const vlist = useRef<VListHandle>(null);
  const stick = useRef(true);
  const layout = layoutRows(props.items);
  const empty = props.items.length === 0 && !props.running && props.approvals.length === 0;
  const live = lastMeaningful(props.items);
  const streamingKey = props.running && live?.type === "assistant" ? live.key : "";
  const showWorking = props.running && !streamingKey && !toolsPending(props.items);
  const retryKey = lastErrorKey(props.items);
  const count = layout.length + props.approvals.length + (showWorking ? 1 : 0);

  useEffect(() => {
    if (!stick.current) return;
    if (vlist.current && count > 0) {
      vlist.current.scrollToIndex(count - 1, { align: "end" });
      return;
    }
    const el = scroller.current;
    if (el) el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
  }, [count, props.items, props.approvals, props.running]);

  useEffect(() => {
    const last = props.items[props.items.length - 1];
    if (last?.type === "user") stick.current = true;
  }, [props.items]);

  const rows: { key: string; space?: string; node: ReactNode }[] = [
    ...layout.map((row, i) => {
      if (row.kind === "user") {
        return {
          key: row.key,
          space: rowSpace(layout, i),
          node: <ItemRow item={row.item} compact={props.compact} copyText={props.compact ? "" : row.item.text} />,
        };
      }
      if (row.kind === "agent") {
        const liveHere = row.parts.some((p) => p.kind === "item" && p.item.key === streamingKey);
        const toolsLast = row.parts[row.parts.length - 1]?.kind === "tools";
        return {
          key: row.key,
          space: rowSpace(layout, i),
          node: (
            <div className="group/turn" data-testid="agent-turn">
              {row.parts.map((part, pi) => (
                <div key={part.key} className={partSpace(row.parts, pi)}>
                  {part.kind === "tools" ? (
                    <ToolGroup items={part.items} defaultOpen={!!props.running && toolsLast} onOpenReview={props.onOpenReview} />
                  ) : (
                    <ItemRow
                      item={part.item}
                      streaming={part.item.key === streamingKey}
                      compact={props.compact}
                      onOpenReview={props.onOpenReview}
                    />
                  )}
                </div>
              ))}
              {row.copyText && !props.compact
                ? (liveHere ? <div className="h-6" aria-hidden /> : <CopyAction text={row.copyText} />)
                : null}
            </div>
          ),
        };
      }
      return {
        key: row.key,
        space: rowSpace(layout, i),
        node: (
          <ItemRow
            item={row.item}
            compact={props.compact}
            onOpenReview={props.onOpenReview}
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
  if (showWorking) {
    rows.push({
      key: "working",
      space: "pt-3",
      node: (
        <div className="flex items-center gap-2 text-[13px] text-muted" role="status" aria-live="polite">
          <span className="thinking-dots" aria-hidden>
            <i /><i /><i />
          </span>
          {copy.transcript.working}
        </div>
      ),
    });
  }

  const virtual = !props.compact && rows.length > 24;

  function onScrollNearBottom(el: { scrollHeight: number; scrollTop: number; clientHeight: number }) {
    stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
  }

  return (
    <div
      className={cn(
        "transcript-scroll prose-select min-h-0 min-w-0 flex-1 overflow-x-hidden pb-2 pt-3",
        virtual ? "overflow-hidden" : "overflow-y-auto",
      )}
      ref={scroller}
      onScroll={virtual ? undefined : () => {
        const el = scroller.current;
        if (el) onScrollNearBottom(el);
      }}
    >
      {empty && !props.compact ? (
        <div className={cn(pad, "flex min-h-full flex-col justify-end pb-6")}>
          <h1 className="text-[26px] font-semibold tracking-[-0.038em] text-foreground">
            {props.needsSetup ? copy.transcript.setupTitle : copy.transcript.ready}
          </h1>
          <p className="mt-2 max-w-md text-[14px] leading-[1.55] text-muted">
            {props.needsSetup ? copy.transcript.setupBody : copy.transcript.readyBody}
          </p>
          {props.needsSetup ? (
            <button
              type="button"
              className="mt-6 w-fit rounded-full bg-foreground px-4 py-2 text-[13px] font-medium text-background"
              onClick={props.onSetup}
            >
              {copy.transcript.openControl}
            </button>
          ) : (
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
          )}
        </div>
      ) : virtual ? (
        <VList
          ref={vlist}
          className="transcript-scroll h-full w-full"
          onScroll={(offset) => {
            const handle = vlist.current;
            if (!handle) return;
            stick.current = handle.scrollSize - offset - handle.viewportSize < 80;
          }}
        >
          {rows.map((row) => (
            <div key={row.key} className={cn(pad, "pb-1", row.space)}>
              {row.node}
            </div>
          ))}
        </VList>
      ) : (
        <div className={cn(pad, "flex flex-col pb-2")}>
          {rows.map((row) => (
            <div key={row.key} className={row.space}>{row.node}</div>
          ))}
        </div>
      )}
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
          className="rounded-full bg-lift px-3 py-1.5 text-[12px] text-foreground hover:bg-lift/80"
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

function CopyAction({ text, align = "start" }: { text: string; align?: "start" | "end" }) {
  const copy = useCopy();
  const [done, setDone] = useState(false);
  if (!text) return null;
  return (
    <div
      className={cn(
        "flex h-6 items-center opacity-0 transition-opacity duration-150 pointer-events-none group-hover/msg:pointer-events-auto group-hover/msg:opacity-100 group-focus-within/msg:pointer-events-auto group-focus-within/msg:opacity-100 group-hover/turn:pointer-events-auto group-hover/turn:opacity-100 group-focus-within/turn:pointer-events-auto group-focus-within/turn:opacity-100",
        align === "end" ? "justify-end" : "justify-start",
      )}
    >
      <button
        type="button"
        className="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] text-muted hover:bg-lift hover:text-foreground"
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
    </div>
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
          className="mt-3 rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background"
          onClick={onRetry}
        >
          {copy.transcript.retry}
        </button>
      ) : null}
    </div>
  );
}

function ItemRow({
  item,
  streaming,
  compact,
  copyText,
  onOpenReview,
  onRetry,
}: {
  item: Item;
  streaming?: boolean;
  compact?: boolean;
  copyText?: string;
  onOpenReview?: () => void;
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
        <div className="group/msg max-w-[min(85%,28rem)]">
          <div className="whitespace-pre-wrap break-words rounded-[18px] bg-lift px-3.5 py-[9px] text-[14.5px] leading-[1.55] tracking-[-0.012em]">
            {item.text}
          </div>
          {copyText ? <CopyAction text={copyText} align="end" /> : null}
        </div>
      </div>
    );
  }
  if (item.type === "assistant") {
    return (
      <div className="w-full min-w-0">
        <div className={cn("text-[15px] leading-[1.65] tracking-[-0.011em] text-foreground/95", streaming && "assistant-live")}>
          {item.text ? <Markdown text={item.text} /> : null}
          {streaming ? (
            <span
              className="caret-blink ml-0.5 inline-block h-[1em] w-[7px] translate-y-0.5 rounded-sm bg-accent align-text-bottom"
              aria-hidden
            />
          ) : null}
        </div>
      </div>
    );
  }
  if (item.type === "error") {
    return <ErrorCard item={item} onRetry={onRetry} />;
  }
  if (item.type === "reasoning") {
    return (
      <details className="text-[12px] text-muted">
        <summary className="cursor-pointer text-[12px] text-muted hover:text-foreground">{copy.transcript.thinking}</summary>
        <div className="mt-1.5 text-[13px] text-muted"><Markdown text={item.text} /></div>
      </details>
    );
  }
  if (item.type === "compaction") {
    const kind = String(item.payload?.kind || "");
    const label = kind === "checkpoint"
      ? copy.transcript.checkpoint
      : (item.payload.note || item.text || copy.app.compacted);
    return <div className="text-center text-[11px] text-muted/80">{String(label)}</div>;
  }
  if (item.type === "subagent") {
    return (
      <details className="text-[12px] text-muted">
        <summary className="cursor-pointer hover:text-foreground">{copy.transcript.subagent}</summary>
        <pre className="mt-1.5 max-h-48 overflow-auto whitespace-pre-wrap font-mono text-[11px]">{item.text || JSON.stringify(item.payload).slice(0, 2000)}</pre>
      </details>
    );
  }
  if (item.type === "tool_call" || item.type === "tool_result") {
    return <ToolItem item={item} onOpenReview={onOpenReview} />;
  }
  if (item.type === "context_injection") {
    return (
      <details className="text-[12px] text-muted">
        <summary className="cursor-pointer hover:text-foreground">{copy.transcript.mention}</summary>
        <pre className="mt-1.5 max-h-48 overflow-auto whitespace-pre-wrap font-mono text-[11px]">{item.text.slice(0, 1200)}</pre>
      </details>
    );
  }
  if (item.type === "turn_end" || item.type === "system") return null;
  return (
    <details className="text-[12px] text-muted">
      <summary className="cursor-pointer hover:text-foreground">{item.type}</summary>
      <pre className="mt-1.5 max-h-48 overflow-auto whitespace-pre-wrap font-mono text-[11px]">
        {item.text || JSON.stringify(item.payload).slice(0, 400)}
      </pre>
    </details>
  );
}

function ToolGroup({ items, defaultOpen, onOpenReview }: { items: Item[]; defaultOpen?: boolean; onOpenReview?: () => void }) {
  const copy = useCopy();
  const pairs = pairTools(items);
  const pending = pairs.some((p) => p.call && !p.result);
  const [open, setOpen] = useState(!!pending || !!defaultOpen);
  useEffect(() => {
    if (pending) setOpen(true);
  }, [pending]);
  const n = Math.max(pairs.filter((p) => p.call || p.result).length, 1);
  const label = copy.transcript.toolsUsed.replace("{n}", String(n));
  const names = pairs.map((p) => p.call?.name || p.result?.name || p.call?.payload?.name).filter(Boolean);
  return (
    <div className="min-w-0">
      <button
        type="button"
        className="flex max-w-full items-center gap-1.5 py-0.5 text-left text-[12px] text-muted hover:text-foreground"
        onClick={() => setOpen((v) => !v)}
      >
        {pending ? <Loader2 className="size-3 animate-spin text-accent" aria-hidden /> : <Wrench className="size-3" aria-hidden />}
        <span>{label}</span>
        <span className="min-w-0 truncate text-[11px] text-muted/70">{names.join(" · ")}</span>
        <ChevronRight className={cn("size-3 shrink-0 transition-transform duration-150", open && "rotate-90")} />
      </button>
      {open ? (
        <div className="mt-1 space-y-0.5 border-l border-border/70 pl-3">
          {pairs.map((p) => (
            <div key={p.key} className="space-y-0.5">
              {p.call ? <ToolItem item={p.call} onOpenReview={onOpenReview} pending={!p.result} /> : null}
              {p.result ? <ToolItem item={p.result} onOpenReview={onOpenReview} /> : null}
              {p.extra.map((it) => (
                <ToolItem key={it.key} item={it} onOpenReview={onOpenReview} />
              ))}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function ToolItem({ item, onOpenReview, pending }: { item: Item; onOpenReview?: () => void; pending?: boolean }) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const name = item.name || item.payload.name || "tool";
  const body =
    item.type === "tool_call"
      ? String(item.payload.arguments || item.text || "")
      : String(item.payload.content || item.text || "");
  if (name === "update_plan") {
    return (
      <div className="py-1">
        <div className="mb-1 text-[11px] text-muted">{copy.transcript.plan}</div>
        <pre className="whitespace-pre-wrap font-mono text-[12px] text-foreground/90">{body.slice(0, 4000)}</pre>
      </div>
    );
  }
  if (name === "apply_patch") {
    return (
      <details className="py-1 text-[12px]" open>
        <summary className="cursor-pointer text-muted hover:text-foreground">{copy.transcript.patch}</summary>
        <div className="mt-1.5">
          <DiffBlock src={body.slice(0, 6000)} mode="unified" />
        </div>
        {onOpenReview ? (
          <button type="button" className="mt-1.5 text-[11px] text-accent hover:underline" onClick={onOpenReview}>
            {copy.review.openReview}
          </button>
        ) : null}
      </details>
    );
  }
  return (
    <div>
      <button type="button" className="flex w-full items-center gap-1.5 py-0.5 text-left text-[12px] text-muted hover:text-foreground" onClick={() => setOpen((v) => !v)}>
        {pending ? <Loader2 className="size-3 animate-spin text-accent" aria-hidden /> : null}
        <span className="font-medium text-foreground/85">{name}</span>
        <span className="text-[10px] tracking-wide text-muted/70">{pending ? "…" : item.type === "tool_call" ? "call" : "result"}</span>
        <ChevronRight className={cn("ml-auto size-3 shrink-0 transition-transform duration-150", open && "rotate-90")} />
      </button>
      {open ? <pre className="mt-0.5 max-h-56 overflow-auto font-mono text-[11px] text-muted">{body.slice(0, 4000)}</pre> : null}
    </div>
  );
}
