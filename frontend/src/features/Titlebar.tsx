import { useState } from "react";
import { Folder, GitFork, PanelRight, Settings } from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import type { ContextUsage, Health, Thread } from "../lib/protocol";
import { RunningHub } from "./RunningHub";

export function Titlebar(props: {
  health: Health;
  ctx: ContextUsage;
  workspace: string;
  onControl: () => void;
  onToggleInspector: () => void;
  inspector: boolean;
  title: string;
  onRename: (title: string) => void;
  onFork: () => void;
  onDiff: () => void;
  runningCount?: number;
  runningThreads?: Thread[];
  onSelectRunning?: (t: Thread) => void;
}) {
  const copy = useCopy();
  const h = props.health;
  const ctx = props.ctx;
  const pct = ctx.budget > 0 ? Math.min(100, Math.round((ctx.tokens / ctx.budget) * 100)) : 0;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(props.title);

  function commit() {
    const title = draft.trim();
    setEditing(false);
    if (!title || title === props.title) {
      setDraft(props.title);
      return;
    }
    props.onRename(title);
  }

  return (
    <div className="flex h-11 min-w-0 shrink-0 items-center gap-3 border-b border-border px-4">
      {editing ? (
        <input
          autoFocus
          aria-label={copy.titlebar.rename}
          className="min-w-0 flex-1 rounded-md bg-lift px-2 py-1 text-[13px] font-medium text-foreground"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              commit();
            }
            if (e.key === "Escape") {
              setDraft(props.title);
              setEditing(false);
            }
          }}
        />
      ) : (
        <button
          type="button"
          className="min-w-0 truncate text-[13px] font-medium text-foreground"
          onClick={() => {
            setDraft(props.title);
            setEditing(true);
          }}
          aria-label={copy.titlebar.rename}
        >
          {props.title}
        </button>
      )}
      <div className="ml-auto flex min-w-0 items-center gap-1.5">
        {props.workspace ? (
          <Badge className="hidden max-w-[140px] truncate sm:inline-flex" title={props.workspace}>
            <Folder className="mr-1 size-3" aria-hidden />
            {shortPath(props.workspace)}
          </Badge>
        ) : null}
        {ctx.budget > 0 ? (
          <span className="hidden items-center gap-2 text-[11px] tabular-nums text-muted md:flex" title={ctx.note || "context window"}>
            <span className="h-1.5 w-16 overflow-hidden rounded-full bg-lift">
              <span className="block h-full bg-accent" style={{ width: pct + "%" }} />
            </span>
            {fmt(ctx.tokens)}/{fmt(ctx.budget)}
          </span>
        ) : null}
        {props.runningThreads && props.onSelectRunning ? (
          <RunningHub threads={props.runningThreads} onSelect={props.onSelectRunning} />
        ) : props.runningCount ? (
          <Badge className="hidden sm:inline-flex">{props.runningCount} {copy.titlebar.live}</Badge>
        ) : null}
        {h.usageUsd ? <Badge className="hidden lg:inline-flex">${h.usageUsd.toFixed(4)}</Badge> : null}
        <Badge className="hidden sm:inline-flex">{h.model || "model"}</Badge>
        <Button className="hidden sm:inline-flex" variant="ghost" size="sm" onClick={props.onFork} aria-label={copy.titlebar.fork}>
          <GitFork />
          {copy.titlebar.fork}
        </Button>
        <Button className="hidden sm:inline-flex" variant="ghost" size="sm" onClick={props.onDiff}>{copy.titlebar.diff}</Button>
        <Button
          variant="ghost"
          size="icon"
          onClick={props.onToggleInspector}
          aria-label={copy.review.toggle}
          aria-pressed={props.inspector}
          className={cn("shrink-0", props.inspector && "text-foreground")}
        >
          <PanelRight />
        </Button>
        <Button variant="ghost" size="icon" className="shrink-0" onClick={props.onControl} aria-label={copy.app.openControl}>
          <Settings />
        </Button>
      </div>
    </div>
  );
}

function shortPath(p: string): string {
  if (!p) return "";
  const parts = p.replace(/\\/g, "/").split("/");
  return parts.slice(-2).join("/");
}

function fmt(n: number): string {
  if (n >= 1000) return (n / 1000).toFixed(n >= 10000 ? 0 : 1) + "k";
  return String(n);
}
