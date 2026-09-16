import { useState } from "react";
import { FlaskConical, GitBranch, MessageSquarePlus, Pin, Search, Shield, SlidersHorizontal } from "lucide-react";
import { VList } from "virtua";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import type { Lab, Thread } from "../lib/protocol";

export function ThreadRail(props: {
  threads: Thread[];
  activeId: string;
  query: string;
  running: Record<string, boolean>;
  lab: Lab;
  showArchived?: boolean;
  onQuery: (q: string) => void;
  onSelect: (t: Thread) => void;
  onNew: () => void;
  onLab: (lab: Lab) => void;
  onControl: () => void;
  onPin?: (t: Thread, pinned: boolean) => void;
  onArchive?: (t: Thread, archived: boolean) => void;
  onDelete?: (t: Thread) => void;
  onRename?: (t: Thread, title: string) => void;
  onToggleArchived?: () => void;
}) {
  const copy = useCopy();
  const labs: { id: Lab; label: string; hint: string; icon: typeof Shield }[] = [
    { id: "harbor", label: copy.rail.harbor, hint: copy.rail.harborHint, icon: Shield },
    { id: "evolve", label: copy.rail.evolve, hint: copy.rail.evolveHint, icon: FlaskConical },
    { id: "harness", label: copy.rail.harness, hint: copy.rail.harnessHint, icon: GitBranch },
  ];
  const q = props.query.toLowerCase();
  const list = props.threads.filter((t) => {
    if (!props.showArchived && t.archived) return false;
    if (!q) return true;
    return (t.title + t.id + t.workspace).toLowerCase().includes(q);
  });
  const virtual = list.length > 24;
  return (
    <aside className="flex h-full min-h-0 flex-col overflow-hidden bg-sidebar">
      <div className="px-3 pt-3 pb-2">
        <Button className="w-full justify-start gap-2 rounded-xl" variant="lift" onClick={props.onNew}>
          <MessageSquarePlus className="size-4" aria-hidden />
          {copy.rail.newChat}
        </Button>
        <div className="relative mt-2">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
          <Input
            className="rounded-xl border-transparent bg-background pl-9"
            placeholder={copy.rail.search}
            aria-label={copy.rail.search}
            value={props.query}
            onChange={(e) => props.onQuery(e.target.value)}
          />
        </div>
      </div>
      <nav aria-label="Chats" className="min-h-0 flex-1 overflow-hidden px-2 pb-2">
        {list.length === 0 ? <div className="px-3 py-6 text-xs text-muted">{copy.rail.noChats}</div> : null}
        {virtual ? (
          <VList className="h-full">
            {list.map((t) => (
              <ThreadRow key={t.id} thread={t} {...props} />
            ))}
          </VList>
        ) : (
          <div className="h-full overflow-auto">
            {list.map((t) => (
              <ThreadRow key={t.id} thread={t} {...props} />
            ))}
          </div>
        )}
      </nav>
      <div className="border-t border-border p-2">
        <button type="button" className="mb-1 w-full px-3 text-left text-[11px] text-muted hover:text-foreground" onClick={props.onToggleArchived}>
          {copy.rail.showArchived}
        </button>
        <div className="px-3 pb-1 text-[11px] text-muted">{copy.rail.labs}</div>
        {labs.map((l) => {
          const Icon = l.icon;
          return (
            <button
              type="button"
              key={l.id}
              title={l.hint}
              aria-current={props.lab === l.id ? "page" : undefined}
              className={cn(
                "mb-0.5 flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-[13px] transition-colors duration-150",
                props.lab === l.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/60 hover:text-foreground",
              )}
              onClick={() => props.onLab(l.id)}
            >
              <Icon className="size-4" aria-hidden />
              {l.label}
            </button>
          );
        })}
        <button
          type="button"
          aria-current={props.lab === "control" ? "page" : undefined}
          className={cn(
            "mt-1 flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-[13px] transition-colors duration-150",
            props.lab === "control" ? "bg-lift text-foreground" : "text-muted hover:bg-lift/60 hover:text-foreground",
          )}
          onClick={props.onControl}
        >
          <SlidersHorizontal className="size-4" aria-hidden />
          {copy.rail.control}
        </button>
      </div>
    </aside>
  );
}

function ThreadRow(props: {
  thread: Thread;
  activeId: string;
  running: Record<string, boolean>;
  lab: Lab;
  onSelect: (t: Thread) => void;
  onLab: (lab: Lab) => void;
  onPin?: (t: Thread, pinned: boolean) => void;
  onArchive?: (t: Thread, archived: boolean) => void;
  onDelete?: (t: Thread) => void;
  onRename?: (t: Thread, title: string) => void;
}) {
  const copy = useCopy();
  const t = props.thread;
  const run = !!props.running[t.id];
  const active = t.id === props.activeId && props.lab === "agent";
  const [menu, setMenu] = useState<{ x: number; y: number } | null>(null);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(t.title || "");

  function commit() {
    const title = draft.trim();
    setEditing(false);
    if (!title || title === t.title) return;
    props.onRename?.(t, title);
  }

  return (
    <div className="relative">
      <button
        type="button"
        aria-current={active ? "page" : undefined}
        className={cn(
          "mb-0.5 flex w-full items-start gap-2 rounded-xl px-3 py-2 text-left transition-colors duration-150",
          active ? "bg-lift text-foreground" : "text-muted hover:bg-lift/60 hover:text-foreground",
        )}
        onClick={() => {
          if (editing) return;
          props.onSelect(t);
          props.onLab("agent");
          setMenu(null);
        }}
        onDoubleClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          setDraft(t.title || "");
          setEditing(true);
        }}
        onContextMenu={(e) => {
          e.preventDefault();
          setMenu({ x: e.clientX, y: e.clientY });
        }}
      >
        <span className={cn("mt-1.5 size-1.5 shrink-0 rounded-full", run ? "animate-pulse bg-accent" : "bg-muted/40")} aria-hidden />
        <span className="min-w-0 flex-1">
          <span className="flex items-center gap-1">
            {t.pinned ? <Pin className="size-3 text-accent" aria-hidden /> : null}
            {editing ? (
              <input
                autoFocus
                aria-label={copy.rail.rename}
                className="w-full rounded-md bg-background px-1 text-[13px] font-medium text-foreground"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onClick={(e) => e.stopPropagation()}
                onBlur={commit}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    commit();
                  }
                  if (e.key === "Escape") {
                    e.preventDefault();
                    setDraft(t.title || "");
                    setEditing(false);
                  }
                }}
              />
            ) : (
              <span className="block truncate text-[13px] font-medium text-foreground">{t.title || t.id.slice(0, 8)}</span>
            )}
          </span>
          <span className="block truncate text-[11px] text-muted">
            {run ? copy.rail.running : t.archived ? copy.rail.archived : t.workspace ? short(t.workspace) : copy.rail.idle}
          </span>
        </span>
      </button>
      {menu ? (
        <div
          className="fixed z-50 min-w-[140px] rounded-xl border border-border bg-panel py-1 text-[12px] shadow-xl"
          style={{ left: menu.x, top: menu.y }}
          onMouseLeave={() => setMenu(null)}
        >
          <MenuItem label={copy.rail.rename} onClick={() => { setDraft(t.title || ""); setEditing(true); setMenu(null); }} />
          <MenuItem label={t.pinned ? copy.rail.unpin : copy.rail.pin} onClick={() => { props.onPin?.(t, !t.pinned); setMenu(null); }} />
          <MenuItem label={t.archived ? copy.rail.unarchive : copy.rail.archive} onClick={() => { props.onArchive?.(t, !t.archived); setMenu(null); }} />
          <MenuItem label={copy.rail.delete} onClick={() => { props.onDelete?.(t); setMenu(null); }} />
        </div>
      ) : null}
    </div>
  );
}

function MenuItem({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button type="button" className="block w-full px-3 py-1.5 text-left hover:bg-lift" onClick={onClick}>
      {label}
    </button>
  );
}

function short(p: string): string {
  const parts = p.replace(/\\/g, "/").split("/");
  return parts[parts.length - 1] || p;
}
