import { useEffect, useRef, useState } from "react";
import {
  Bell,
  ChevronLeft,
  FlaskConical,
  GitBranch,
  MessageSquarePlus,
  MoreHorizontal,
  Pin,
  Search,
  Settings,
  Shield,
} from "lucide-react";
import { VList } from "virtua";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { EmptyState } from "../components/ui/empty-state";
import { Tooltip } from "../components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../components/ui/dropdown-menu";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { isMac } from "../lib/chrome";
import { displayTitle, displayWorkspace } from "../lib/display-title";
import type { Lab, Notice, Surface, Thread } from "../lib/protocol";
import { SidebarCard } from "./shell/AppFrame";

export function ThreadRail(props: {
  threads: Thread[];
  activeId: string;
  query: string;
  running: Record<string, boolean>;
  surface: Surface;
  lab: Lab;
  showArchived?: boolean;
  notices: Notice[];
  noticesOpen: boolean;
  connected: boolean;
  isolated?: boolean;
  onQuery: (q: string) => void;
  onSelect: (t: Thread) => void;
  onNew: () => void;
  onLab: (lab: Lab) => void;
  onSettings: () => void;
  onCollapse: () => void;
  onPin?: (t: Thread, pinned: boolean) => void;
  onArchive?: (t: Thread, archived: boolean) => void;
  onDelete?: (t: Thread) => void;
  onRename?: (t: Thread, title: string) => void;
  onFork?: (t: Thread) => void;
  onToggleArchived?: () => void;
  onToggleNotices: () => void;
  onClearNotices: () => void;
  onNotice: (n: Notice) => void;
}) {
  const copy = useCopy();
  const mac = isMac();
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
    <SidebarCard>
      <div className={cn("chrome drag flex h-11 shrink-0 items-center gap-2 px-3", mac && "pl-[76px]")}>
        <span
          className={cn("size-2 rounded-full", props.connected ? "bg-accent" : "bg-muted")}
          title={props.connected ? copy.rail.connected : copy.rail.disconnected}
        />
        <span className="text-[13px] font-semibold tracking-tight">Yoyo</span>
        {props.isolated ? <span className="rounded-md bg-lift px-1.5 py-0.5 text-[10px] text-muted">{copy.rail.isolated}</span> : null}
        <Tooltip content={copy.rail.collapse} side="bottom">
          <button
            type="button"
            className="no-drag ml-auto grid size-7 place-items-center rounded-md text-muted transition-colors hover:bg-lift hover:text-foreground"
            aria-label={copy.rail.collapse}
            onClick={props.onCollapse}
          >
            <ChevronLeft className="size-4" />
          </button>
        </Tooltip>
      </div>
      <div className="px-2.5 pb-2">
        <Button className="h-9 w-full justify-start gap-2 rounded-xl text-[13px]" variant="outline" onClick={props.onNew}>
          <MessageSquarePlus className="size-4" aria-hidden />
          {copy.rail.newChat}
        </Button>
        <div className="mt-2 flex gap-0.5" aria-label={copy.rail.labs}>
          {labs.map((l) => {
            const Icon = l.icon;
            const active = props.surface === l.id;
            return (
              <button
                type="button"
                key={l.id}
                title={l.hint}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex flex-1 items-center justify-center gap-1 rounded-lg py-[6px] text-[11px] font-medium transition-colors duration-180",
                  active ? "bg-lift text-foreground" : "text-muted hover:bg-lift/60 hover:text-foreground",
                )}
                onClick={() => props.onLab(l.id)}
              >
                <Icon className="size-3.5" aria-hidden />
                <span className="hidden min-[220px]:inline">{l.label}</span>
              </button>
            );
          })}
        </div>
        <div className="relative mt-2">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
          <Input
            className="h-8 rounded-lg border-transparent bg-lift/40 pl-8 text-[13px]"
            placeholder={copy.rail.search}
            aria-label={copy.rail.search}
            value={props.query}
            onChange={(e) => props.onQuery(e.target.value)}
          />
        </div>
      </div>
      <nav aria-label={copy.rail.chats} className="min-h-0 flex-1 overflow-hidden px-1.5 pb-1">
        {list.length === 0 ? (
          <EmptyState title={copy.rail.noChats} />
        ) : virtual ? (
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
      <div className="flex items-center gap-1 border-t border-border/80 px-2 py-1.5">
        <button
          type="button"
          className="rounded-md px-2 py-1 text-[11px] text-muted transition-colors hover:bg-lift hover:text-foreground"
          onClick={props.onToggleArchived}
        >
          {props.showArchived ? copy.rail.hideArchived : copy.rail.showArchived}
        </button>
        <div className="ml-auto flex items-center gap-0.5">
          <DropdownMenu open={props.noticesOpen} onOpenChange={(v) => { if (v !== props.noticesOpen) props.onToggleNotices(); }}>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="relative grid size-8 place-items-center rounded-lg text-muted transition-colors hover:bg-lift hover:text-foreground"
                aria-label={copy.rail.notifications}
                title={copy.rail.notifications}
              >
                  <Bell className="size-4" />
                  {props.notices.length ? (
                    <span className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-accent" />
                  ) : null}
                </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-72">
              {props.notices.length === 0 ? (
                <div className="px-3 py-6 text-center text-[11px] text-muted">{copy.rail.noNotifications}</div>
              ) : (
                props.notices.slice(0, 8).map((n) => (
                  <DropdownMenuItem key={n.id} onSelect={() => props.onNotice(n)}>
                    <span className="min-w-0">
                      <span className="block truncate text-[12px] font-medium">{n.title}</span>
                      <span className="block truncate text-[11px] text-muted">{n.body}</span>
                    </span>
                  </DropdownMenuItem>
                ))
              )}
              {props.notices.length ? (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={props.onClearNotices}>{copy.rail.clearNotifications}</DropdownMenuItem>
                </>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
          <Tooltip content={copy.rail.settings} side="top">
            <button
              type="button"
              aria-current={props.surface === "settings" ? "page" : undefined}
              aria-label={copy.rail.settings}
              className={cn(
                "grid size-8 place-items-center rounded-lg transition-colors",
                props.surface === "settings" ? "bg-lift text-foreground" : "text-muted hover:bg-lift hover:text-foreground",
              )}
              onClick={props.onSettings}
            >
              <Settings className="size-4" />
            </button>
          </Tooltip>
        </div>
      </div>
    </SidebarCard>
  );
}

function ThreadRow(props: {
  thread: Thread;
  activeId: string;
  running: Record<string, boolean>;
  surface: Surface;
  onSelect: (t: Thread) => void;
  onLab: (lab: Lab) => void;
  onPin?: (t: Thread, pinned: boolean) => void;
  onArchive?: (t: Thread, archived: boolean) => void;
  onDelete?: (t: Thread) => void;
  onRename?: (t: Thread, title: string) => void;
  onFork?: (t: Thread) => void;
}) {
  const copy = useCopy();
  const t = props.thread;
  const run = !!props.running[t.id];
  const active = t.id === props.activeId && props.surface === "agent";
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(t.title || "");
  const [menuOpen, setMenuOpen] = useState(false);
  const skipBlur = useRef(false);

  useEffect(() => {
    if (!editing) setDraft(t.title || "");
  }, [t.title, editing]);

  function startRename() {
    skipBlur.current = false;
    setDraft(t.title || "");
    setEditing(true);
    setMenuOpen(false);
  }

  function commit() {
    if (skipBlur.current) {
      skipBlur.current = false;
      setDraft(t.title || "");
      setEditing(false);
      return;
    }
    const title = draft.trim();
    setEditing(false);
    if (!title || title === (t.title || "")) {
      setDraft(t.title || "");
      return;
    }
    props.onRename?.(t, title);
  }

  if (editing) {
    return (
      <div className="mb-px flex w-full items-center gap-2 rounded-lg bg-lift px-2.5 py-[7px]">
        <span className={cn("size-1.5 shrink-0 rounded-full", run ? "animate-pulse bg-accent" : "bg-muted/35")} aria-hidden />
        <input
          autoFocus
          aria-label={copy.rail.rename}
          className="w-full rounded-md bg-background px-1 text-[13px] font-medium text-foreground"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              commit();
            }
            if (e.key === "Escape") {
              e.preventDefault();
              skipBlur.current = true;
              setDraft(t.title || "");
              setEditing(false);
            }
          }}
        />
      </div>
    );
  }

  return (
    <DropdownMenu modal={false} open={menuOpen} onOpenChange={setMenuOpen}>
      <div
        className={cn(
          "group mb-px flex w-full items-start rounded-lg transition-colors duration-150",
          active ? "bg-lift text-foreground" : "text-muted hover:bg-lift/55 hover:text-foreground",
        )}
      >
        <button
          type="button"
          aria-current={active ? "page" : undefined}
          className="flex min-w-0 flex-1 items-center gap-2 px-2.5 py-[7px] text-left"
          onClick={() => {
            props.onSelect(t);
            props.onLab("agent");
            setMenuOpen(false);
          }}
          onDoubleClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            startRename();
          }}
          onContextMenu={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setMenuOpen(true);
          }}
        >
          <span className={cn("size-1.5 shrink-0 rounded-full", run ? "animate-pulse bg-accent" : "bg-muted/35")} aria-hidden />
          <span className="min-w-0 flex-1">
            <span className="flex items-center gap-1">
              {t.pinned ? <Pin className="size-3 text-accent" aria-hidden /> : null}
              <span className="block truncate text-[13px] font-medium text-foreground">{displayTitle(t.title, copy.rail.untitled)}</span>
            </span>
            <span className="block truncate text-[11px] text-muted">
              {run ? copy.rail.running : t.archived ? copy.rail.archived : displayWorkspace(t.workspace, copy.rail.idle)}
            </span>
          </span>
        </button>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="mt-1 mr-1 grid size-7 shrink-0 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground data-[state=open]:bg-lift data-[state=open]:text-foreground"
            aria-label={copy.rail.more}
            onClick={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="size-4" />
          </button>
        </DropdownMenuTrigger>
      </div>
      <DropdownMenuContent align="end" className="w-44" onCloseAutoFocus={(e) => e.preventDefault()}>
        <DropdownMenuItem onSelect={startRename}>{copy.rail.rename}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => props.onFork?.(t)}>{copy.rail.fork}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => props.onPin?.(t, !t.pinned)}>{t.pinned ? copy.rail.unpin : copy.rail.pin}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => props.onArchive?.(t, !t.archived)}>{t.archived ? copy.rail.unarchive : copy.rail.archive}</DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="text-danger"
          onPointerDown={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setMenuOpen(false);
            props.onDelete?.(t);
          }}
          onSelect={(e) => {
            e.preventDefault();
            props.onDelete?.(t);
          }}
        >
          {copy.rail.delete}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
