import { useEffect, useRef, useState } from "react";
import { ChevronRight, Copy, Loader2, ShieldAlert, Wrench } from "lucide-react";
import { Markdown } from "../lib/markdown";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import type { Approval, Item } from "../lib/protocol";

const STARTERS = [
  { label: "Summarize the workspace", text: "Summarize this workspace and list the files that matter." },
  { label: "Pin the harness", text: "@harness\nWhat is the active harness and what can I safely change?" },
  { label: "Plan a change", text: "Propose a plan to add a README section describing how to run Yoyo desktop." },
];

export function Transcript(props: {
  items: Item[];
  approvals: Approval[];
  running: boolean;
  needsSetup?: boolean;
  compact?: boolean;
  onResolve: (id: string, decision: string) => void;
  onPrompt?: (text: string) => void;
  onSetup?: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const stick = useRef(true);
  useEffect(() => {
    const el = ref.current;
    if (!el || !stick.current) return;
    el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
  }, [props.items, props.approvals, props.running]);

  const empty = props.items.length === 0 && !props.running && props.approvals.length === 0;
  let lastAssistant = -1;
  for (let i = props.items.length - 1; i >= 0; i--) {
    if (props.items[i].type === "assistant") {
      lastAssistant = i;
      break;
    }
  }

  return (
    <div
      className="prose-select min-h-0 flex-1 overflow-auto px-4 pb-4 pt-6"
      ref={ref}
      onScroll={() => {
        const el = ref.current;
        if (!el) return;
        stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
      }}
    >
      {empty && !props.compact ? (
        <div className="mx-auto flex h-full min-h-[240px] max-w-3xl flex-col justify-center">
          <h1 className="text-2xl font-semibold tracking-tight">Ready for a turn</h1>
          <p className="mt-2 max-w-lg text-sm text-muted">
            Pin extra context with @file:path, @folder:dir, or @harness. Plan mode proposes without writing.
          </p>
          {props.needsSetup ? (
            <button
              type="button"
              className="mt-4 w-fit rounded-full bg-foreground px-4 py-2 text-sm text-background"
              onClick={props.onSetup}
            >
              Open Control to set workspace
            </button>
          ) : (
            <div className="mt-6 flex flex-wrap gap-2">
              {STARTERS.map((s) => (
                <button
                  type="button"
                  key={s.label}
                  className="rounded-xl border border-border bg-panel px-3 py-2 text-left text-sm hover:bg-lift"
                  onClick={() => props.onPrompt?.(s.text)}
                >
                  {s.label}
                </button>
              ))}
            </div>
          )}
        </div>
      ) : null}
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-5">
        {props.items.map((it, i) => (
          <ItemRow key={it.key} item={it} streaming={props.running && it.type === "assistant" && i === lastAssistant} />
        ))}
        {props.approvals.map((a) => (
          <div key={a.id} className="rounded-xl border border-accent/30 bg-accent/10 p-4" role="status">
            <div className="mb-2 flex items-center gap-2 text-xs font-medium text-accent">
              <ShieldAlert className="size-3.5" aria-hidden />
              Needs approval
            </div>
            <div className="text-sm font-medium">{a.action || "action"}</div>
            <div className="mt-1 font-mono text-xs text-muted">{a.command || a.path || a.level}</div>
            <div className="mt-3 flex flex-wrap gap-2">
              <Button size="sm" onClick={() => props.onResolve(a.id, "once")}>Once</Button>
              <Button size="sm" variant="lift" onClick={() => props.onResolve(a.id, "session")}>Session</Button>
              <Button size="sm" variant="lift" onClick={() => props.onResolve(a.id, "always")}>Always</Button>
              <Button size="sm" variant="danger" onClick={() => props.onResolve(a.id, "deny")}>Deny</Button>
            </div>
          </div>
        ))}
        {props.running ? (
          <div className="flex items-center gap-2 text-sm text-muted" role="status" aria-live="polite">
            <Loader2 className="size-4 animate-spin text-accent" aria-hidden />
            Working…
          </div>
        ) : null}
      </div>
    </div>
  );
}

function ItemRow({ item, streaming }: { item: Item; streaming?: boolean }) {
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
            className="absolute -right-1 -top-1 hidden rounded-md p-1 text-muted hover:bg-lift group-hover:block"
            aria-label="Copy"
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
  if (item.type === "tool_call" || item.type === "tool_result") {
    return <ToolItem item={item} />;
  }
  if (item.type === "context_injection") {
    return (
      <details className="rounded-xl border border-border bg-panel px-3 py-2 text-xs text-muted">
        <summary className="cursor-pointer text-foreground">Mention</summary>
        <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap font-mono">{item.text.slice(0, 1200)}</pre>
      </details>
    );
  }
  if (item.type === "turn_end" || item.type === "system") return null;
  return (
    <details className="rounded-xl border border-border bg-panel px-3 py-2 text-xs text-muted">
      <summary className="cursor-pointer text-foreground">{item.type}</summary>
      <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap font-mono">
        {item.text || JSON.stringify(item.payload).slice(0, 400)}
      </pre>
    </details>
  );
}

function ToolItem({ item }: { item: Item }) {
  const [open, setOpen] = useState(false);
  const name = item.name || item.payload.name || "tool";
  const body =
    item.type === "tool_call"
      ? String(item.payload.arguments || item.text || "")
      : String(item.payload.content || item.text || "");
  return (
    <div className="rounded-xl border border-border bg-panel">
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
