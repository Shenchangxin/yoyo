import { useEffect, useRef, useState, type ReactNode } from "react";
import { ChevronRight, Copy, Loader2, ShieldAlert, Wrench } from "lucide-react";
import { VList, type VListHandle } from "virtua";
import { Markdown } from "../lib/markdown";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { DiffBlock } from "../lib/split-diff";
import type { Approval, Item } from "../lib/protocol";

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
}) {
  const copy = useCopy();
  const scroller = useRef<HTMLDivElement>(null);
  const vlist = useRef<VListHandle>(null);
  const stick = useRef(true);
  const count = props.items.length + props.approvals.length + (props.running ? 1 : 0);

  useEffect(() => {
    if (!stick.current) return;
    if (vlist.current && count > 0) {
      vlist.current.scrollToIndex(count - 1, { align: "end" });
      return;
    }
    const el = scroller.current;
    if (el) el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
  }, [count, props.items, props.approvals, props.running]);

  const empty = props.items.length === 0 && !props.running && props.approvals.length === 0;
  let lastAssistant = -1;
  for (let i = props.items.length - 1; i >= 0; i--) {
    if (props.items[i].type === "assistant") {
      lastAssistant = i;
      break;
    }
  }

  const rows: { key: string; node: ReactNode }[] = [
    ...props.items.map((it, i) => ({
      key: it.key,
      node: <ItemRow item={it} streaming={props.running && it.type === "assistant" && i === lastAssistant} onOpenReview={props.onOpenReview} />,
    })),
    ...props.approvals.map((a) => ({
      key: `ask:${a.id}`,
      node: (
        <div className="rounded-xl border border-accent/30 bg-accent/10 p-4" role="status">
          <div className="mb-2 flex items-center gap-2 text-xs font-medium text-accent">
            <ShieldAlert className="size-3.5" aria-hidden />
            {copy.transcript.needsApproval}
          </div>
          <div className="text-sm font-medium">{a.action || "action"}</div>
          <div className="mt-1 font-mono text-xs text-muted">{a.command || a.path || a.level}</div>
          <div className="mt-3 flex flex-wrap gap-2">
            <Button size="sm" onClick={() => props.onResolve(a.id, "once")}>{copy.transcript.once}</Button>
            <Button size="sm" variant="lift" onClick={() => props.onResolve(a.id, "session")}>{copy.transcript.session}</Button>
            <Button size="sm" variant="lift" onClick={() => props.onResolve(a.id, "always")}>{copy.transcript.always}</Button>
            <Button size="sm" variant="danger" onClick={() => props.onResolve(a.id, "deny")}>{copy.transcript.deny}</Button>
          </div>
        </div>
      ),
    })),
  ];
  if (props.running) {
    rows.push({
      key: "working",
      node: (
        <div className="flex items-center gap-2 text-sm text-muted" role="status" aria-live="polite">
          <Loader2 className="size-4 animate-spin text-accent" aria-hidden />
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
      className={cn("prose-select min-h-0 flex-1 px-4 pb-3 pt-5", virtual ? "overflow-hidden" : "overflow-auto")}
      ref={scroller}
      onScroll={virtual ? undefined : () => {
        const el = scroller.current;
        if (el) onScrollNearBottom(el);
      }}
    >
      {empty && !props.compact ? (
        <div className="mx-auto flex h-full min-h-[200px] max-w-2xl flex-col justify-end px-1 pb-4">
          <h1 className="text-[22px] font-semibold tracking-[-0.02em]">
            {props.needsSetup ? copy.transcript.setupTitle : copy.transcript.ready}
          </h1>
          <p className="mt-1.5 max-w-md text-[13px] leading-[1.6] text-muted/80">
            {props.needsSetup ? copy.transcript.setupBody : copy.transcript.readyBody}
          </p>
          {props.needsSetup ? (
            <button
              type="button"
              className="mt-5 w-fit rounded-full bg-foreground px-4 py-2 text-[13px] font-medium text-background"
              onClick={props.onSetup}
            >
              {copy.transcript.openControl}
            </button>
          ) : (
            <div className="mt-5 flex flex-wrap gap-2">
              {copy.transcript.starters.map((s) => (
                <button
                  type="button"
                  key={s.label}
                  className="rounded-full border border-border/70 bg-input-bar px-3.5 py-1.5 text-[12px] text-foreground transition-colors hover:border-accent/35 hover:bg-lift/40"
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
          className="mx-auto h-full w-full max-w-2xl"
          onScroll={(offset) => {
            const handle = vlist.current;
            if (!handle) return;
            stick.current = handle.scrollSize - offset - handle.viewportSize < 80;
          }}
        >
          {rows.map((row) => (
            <div key={row.key} className="pb-5">
              {row.node}
            </div>
          ))}
        </VList>
      ) : (
        <div className="mx-auto flex h-full w-full max-w-2xl flex-col gap-5">
          {rows.map((row) => (
            <div key={row.key}>{row.node}</div>
          ))}
        </div>
      )}
    </div>
  );
}

function ItemRow({ item, streaming, onOpenReview }: { item: Item; streaming?: boolean; onOpenReview?: () => void }) {
  const copy = useCopy();
  if (item.type === "user") {
    return (
      <div className="flex justify-end">
        <div className="max-w-[80%] rounded-2xl bg-lift px-4 py-2.5 text-[15px] leading-6">
          <Markdown text={item.text} />
        </div>
      </div>
    );
  }
  if (item.type === "assistant") {
    return (
      <div className="group relative text-[15px] leading-7">
        <Markdown text={item.text} />
        {streaming ? <span className="caret-blink ml-0.5 inline-block h-[1em] w-[7px] translate-y-0.5 rounded-sm bg-accent align-text-bottom" aria-hidden /> : null}
        {item.text && !streaming ? (
          <button
            type="button"
            className="absolute -right-1 -top-1 hidden rounded-md p-1 text-muted opacity-0 transition-opacity hover:bg-lift hover:text-foreground group-hover:block group-hover:opacity-100"
            aria-label={copy.transcript.copy}
            onClick={() => navigator.clipboard.writeText(item.text)}
          >
            <Copy className="size-3.5" />
          </button>
        ) : null}
      </div>
    );
  }
  if (item.type === "error") {
    return (
      <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
        {item.text}
      </div>
    );
  }
  if (item.type === "reasoning") {
    return (
      <details className="rounded-xl border border-border bg-card px-3 py-2 text-xs text-muted">
        <summary className="cursor-pointer text-foreground">{copy.transcript.thinking}</summary>
        <div className="mt-2 text-sm text-muted"><Markdown text={item.text} /></div>
      </details>
    );
  }
  if (item.type === "compaction") {
    return <div className="text-center text-[11px] text-muted">{item.payload.note || item.text || copy.app.compacted}</div>;
  }
  if (item.type === "subagent") {
    return (
      <details className="rounded-xl border border-border bg-card px-3 py-2 text-xs text-muted">
        <summary className="cursor-pointer text-foreground">{copy.transcript.subagent}</summary>
        <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap font-mono">{item.text || JSON.stringify(item.payload).slice(0, 2000)}</pre>
      </details>
    );
  }
  if (item.type === "tool_call" || item.type === "tool_result") {
    return <ToolItem item={item} onOpenReview={onOpenReview} />;
  }
  if (item.type === "context_injection") {
    return (
      <details className="rounded-xl border border-border bg-card px-3 py-2 text-xs text-muted">
        <summary className="cursor-pointer text-foreground">{copy.transcript.mention}</summary>
        <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap font-mono">{item.text.slice(0, 1200)}</pre>
      </details>
    );
  }
  if (item.type === "turn_end" || item.type === "system") return null;
  return (
      <details className="rounded-xl border border-border bg-card px-3 py-2 text-xs text-muted">
      <summary className="cursor-pointer text-foreground">{item.type}</summary>
      <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap font-mono">
        {item.text || JSON.stringify(item.payload).slice(0, 400)}
      </pre>
    </details>
  );
}

function ToolItem({ item, onOpenReview }: { item: Item; onOpenReview?: () => void }) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const name = item.name || item.payload.name || "tool";
  const body =
    item.type === "tool_call"
      ? String(item.payload.arguments || item.text || "")
      : String(item.payload.content || item.text || "");
  if (name === "update_plan") {
    return (
      <div className="rounded-xl border border-border bg-card px-3 py-2">
        <div className="mb-1 text-[11px] text-muted">{copy.transcript.plan}</div>
        <pre className="whitespace-pre-wrap font-mono text-xs text-foreground">{body.slice(0, 4000)}</pre>
      </div>
    );
  }
  if (name === "apply_patch") {
    return (
      <details className="rounded-xl border border-border bg-card px-3 py-2 text-xs" open>
        <summary className="cursor-pointer text-foreground">{copy.transcript.patch}</summary>
        <div className="mt-2">
          <DiffBlock src={body.slice(0, 6000)} mode="unified" />
        </div>
        {onOpenReview ? (
          <button type="button" className="mt-2 text-[11px] text-accent hover:underline" onClick={onOpenReview}>
            {copy.review.openReview}
          </button>
        ) : null}
      </details>
    );
  }
  return (
    <div className="rounded-xl border border-border bg-card">
      <button type="button" className="flex w-full items-center gap-2 px-3 py-2 text-left text-xs text-muted" onClick={() => setOpen((v) => !v)}>
        <Wrench className="size-3.5" aria-hidden />
        <span className="rounded-full bg-lift px-1.5 py-0.5 text-[10px] tracking-wide">
          {item.type === "tool_call" ? "call" : "result"}
        </span>
        <span className="font-medium text-foreground">{name}</span>
        <ChevronRight className={cn("ml-auto size-3.5 transition-transform duration-150", open && "rotate-90")} />
      </button>
      {open ? <pre className="max-h-64 overflow-auto border-t border-border px-3 py-2 font-mono text-[11px] text-muted">{body.slice(0, 4000)}</pre> : null}
    </div>
  );
}
