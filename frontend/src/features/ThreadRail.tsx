import { useCallback, useEffect, useLayoutEffect, useRef, useState, type ReactNode, type RefObject } from "react";
import { createPortal } from "react-dom";
import {
  Archive,
  Bell,
  BookOpen,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Clapperboard,
  Copy as CopyIcon,
  Film,
  Folder,
  GitBranch,
  LayoutGrid,
  MessageSquare,
  MessageSquarePlus,
  MoreHorizontal,
  Pin,
  Plus,
  Search,
  Settings,
} from "lucide-react";
import { VList } from "virtua";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { EmptyState } from "../components/ui/empty-state";
import { IconSwap } from "../components/ui/icon-swap";
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
import { writeClipboard } from "../lib/clipboard";
import { isMac } from "../lib/chrome";
import { displayTitle, displayWorkspace } from "../lib/display-title";
import { canaryDirty, parseHarnessRefs, shortHash, stagingDirty } from "../lib/harness-refs";
import type { Lab, Notice, Surface, Thread, VideoProject } from "../lib/protocol";
import { threadChannel } from "../lib/protocol";
import { SidebarCard } from "./shell/AppFrame";
import { YoyoMark } from "./shell/YoyoMark";
import { VideoShellNav } from "./video/VideoShellNav";
import type { Copy } from "../lib/copy";

type RailEntry =
  | { kind: "label"; id: string; label: string }
  | { kind: "space"; id: string; label: string; path: string }
  | { kind: "thread"; thread: Thread }
  | { kind: "project"; project: VideoProject };

function workspaceKey(t: Thread): string {
  return (t.originWorkspace || t.workspace || "").replace(/\\/g, "/");
}

function railEntries(threads: Thread[], copy: Copy, opts: { groupSpaces: boolean; homeWorkspace?: string }): RailEntry[] {
  const pinned: Thread[] = [];
  const rest: Thread[] = [];
  for (const t of threads) {
    if (t.pinned) pinned.push(t);
    else rest.push(t);
  }
  const out: RailEntry[] = [];
  if (pinned.length) {
    out.push({ kind: "label", id: "pinned", label: copy.rail.pinned });
    for (const thread of pinned) out.push({ kind: "thread", thread });
  }
  if (!opts.groupSpaces) {
    for (const thread of rest) out.push({ kind: "thread", thread });
    return out;
  }
  const projects = new Map<string, { path: string; label: string; items: Thread[] }>();
  const addPath = (path: string) => {
    const key = path.replace(/\\/g, "/") || "__none__";
    if (projects.has(key)) return;
    projects.set(key, {
      path,
      label: displayWorkspace(path, copy.rail.project),
      items: [],
    });
  };
  if (opts.homeWorkspace) addPath(opts.homeWorkspace);
  for (const t of rest) {
    const path = t.originWorkspace || t.workspace || opts.homeWorkspace || "";
    addPath(path);
    const key = path.replace(/\\/g, "/") || "__none__";
    projects.get(key)!.items.push(t);
  }
  const home = (opts.homeWorkspace || "").replace(/\\/g, "/");
  const names = [...projects.values()].sort((a, b) => {
    const ap = a.path.replace(/\\/g, "/");
    const bp = b.path.replace(/\\/g, "/");
    if (ap === home) return -1;
    if (bp === home) return 1;
    return a.label.localeCompare(b.label);
  });
  for (const p of names) {
    const items = p.items.slice().sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt));
    out.push({ kind: "space", id: "p-" + (p.path || p.label), label: p.label, path: p.path });
    for (const thread of items) out.push({ kind: "thread", thread });
  }
  return out;
}

function visibleEntries(entries: RailEntry[], collapsed: Record<string, boolean>): RailEntry[] {
  const out: RailEntry[] = [];
  let hidePath: string | null = null;
  for (const e of entries) {
    if (e.kind === "space") {
      hidePath = collapsed[e.path] ? e.path.replace(/\\/g, "/") : null;
      out.push(e);
      continue;
    }
    if (hidePath) {
      if (e.kind === "thread" && (workspaceKey(e.thread) === hidePath || hidePath === "chats")) continue;
      if (e.kind === "project" && e.project.kind === hidePath) continue;
    }
    out.push(e);
  }
  return out;
}

function videoRailEntries(threads: Thread[], projects: VideoProject[], copy: Copy, q: string): RailEntry[] {
  const query = q.toLowerCase();
  const match = (title: string, id: string) => !query || `${title} ${id}`.toLowerCase().includes(query);
  const bound = new Set(projects.map((p) => p.sessionId).filter(Boolean));
  const canvases = projects.filter((p) => p.kind === "canvas" && match(p.title || "", p.id));
  const dramas = projects.filter((p) => p.kind === "drama" && match(p.title || "", p.id));
  const chats = threads.filter((t) => !bound.has(t.id) && match(t.title || "", t.id));
  const out: RailEntry[] = [];
  const push = (path: string, label: string, items: RailEntry[]) => {
    if (!items.length) return;
    out.push({ kind: "space", id: "g-" + path, label, path });
    for (const item of items) out.push(item);
  };
  push("canvas", copy.video.canvas, canvases.map((project) => ({ kind: "project", project })));
  push("drama", copy.video.drama, dramas.map((project) => ({ kind: "project", project })));
  push("chats", copy.rail.chats, chats.map((thread) => ({ kind: "thread", thread })));
  return out;
}

export function ThreadRail(props: {
  threads: Thread[];
  activeId: string;
  query: string;
  running: Record<string, boolean>;
  surface: Surface;
  lab: Lab;
  harness?: unknown;
  fallbackActive?: string;
  showArchived?: boolean;
  notices: Notice[];
  noticesOpen: boolean;
  connected: boolean;
  isolated?: boolean;
  isolationKind?: string;
  onQuery: (q: string) => void;
  onSelect: (t: Thread) => void;
  onSelectProject?: (p: VideoProject) => void;
  videoProjects?: VideoProject[];
  canvasProjectId?: string;
  dramaId?: string;
  onNew: () => void;
  onNewIn?: (path: string) => void;
  workspace?: string;
  onLab: (lab: Lab) => void;
  onChats: () => void;
  onHarness: () => void;
  onSkills: () => void;
  onVideo: () => void;
  onSettings: () => void;
  onCollapse: () => void;
  onPin?: (t: Thread, pinned: boolean) => void;
  onArchive?: (t: Thread, archived: boolean) => void;
  onDelete?: (t: Thread) => void;
  onRename?: (t: Thread, title: string) => void;
  onFork?: (t: Thread) => void;
  onPopOut?: (t: Thread) => void;
  onOpenEditor?: (t: Thread) => void;
  onOpenTerminal?: (t: Thread) => void;
  onToggleArchived?: () => void;
  onToggleNotices: () => void;
  onClearNotices: () => void;
  onNotice: (n: Notice) => void;
}) {
  const copy = useCopy();
  const mac = isMac();
  const refs = parseHarnessRefs(props.harness, props.fallbackActive);
  const activeShort = shortHash(refs.active || props.fallbackActive || "");
  const dirty = stagingDirty(refs);
  const canary = canaryDirty(refs);
  const q = props.query.toLowerCase();
  const list = props.threads.filter((t) => {
    if (!props.showArchived && t.archived) return false;
    if (threadChannel(t) !== (props.surface === "video" ? "video" : "agent")) return false;
    if (!q) return true;
    return (t.title + t.id + t.workspace).toLowerCase().includes(q);
  });
  const harnessOn = props.surface === "harness";
  const skillsOn = props.surface === "skills";
  const videoOn = props.surface === "video";
  const virtual = list.length + (videoOn ? (props.videoProjects || []).length : 0) > 24;
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const entries = visibleEntries(
    videoOn
      ? videoRailEntries(list, props.videoProjects || [], copy, q)
      : railEntries(list, copy, { groupSpaces: !q, homeWorkspace: props.workspace }),
    collapsed,
  );
  const empty = videoOn
    ? list.length === 0 && !(props.videoProjects || []).length
    : list.length === 0 && !props.workspace;
  function renderEntry(e: RailEntry) {
    if (e.kind === "label") return <RailLabel key={e.id} label={e.label} />;
    if (e.kind === "space") {
      return (
        <SpaceHeader
          key={e.id}
          label={e.label}
          path={e.path}
          collapsed={!!collapsed[e.path]}
          onToggle={() => setCollapsed((c) => ({ ...c, [e.path]: !c[e.path] }))}
          onNew={!videoOn && props.onNewIn ? () => props.onNewIn?.(e.path) : undefined}
        />
      );
    }
    if (e.kind === "project") {
      const p = e.project;
      const active = (p.sessionId && p.sessionId === props.activeId)
        || (p.kind === "canvas" && p.id === props.canvasProjectId)
        || (p.kind === "drama" && p.id === props.dramaId);
      return (
        <ProjectRow
          key={`${p.kind}-${p.id}`}
          project={p}
          active={!!active}
          onSelect={() => props.onSelectProject?.(p)}
        />
      );
    }
    return <ThreadRow key={e.thread.id} thread={e.thread} nested={!q && !e.thread.pinned && !videoOn} {...props} />;
  }
  return (
    <SidebarCard>
      <div className={cn("chrome drag flex h-12 shrink-0 items-center gap-2 px-3", mac && "pl-[76px]")}>
        <YoyoMark compact />
        {props.isolated ? <span className="rounded-full bg-lift px-1.5 py-0.5 text-[10px] text-muted">{copy.rail.isolated}</span> : props.isolationKind && props.isolationKind !== "none" ? (
          <span className="rounded-full bg-lift px-1.5 py-0.5 text-[10px] text-muted" title={copy.rail.isolator}>{props.isolationKind.replace(/_/g, " ")}</span>
        ) : null}
        <Tooltip content={copy.rail.collapse} side="bottom">
          <button
            type="button"
            className="no-drag ml-auto grid size-7 cursor-pointer place-items-center rounded-lg text-muted transition-[background-color,color,transform] duration-200 ease-[var(--ease-out)] hover:bg-lift hover:text-foreground active:scale-[0.96]"
            aria-label={copy.rail.collapse}
            onClick={props.onCollapse}
          >
            <ChevronLeft className="size-4" />
          </button>
        </Tooltip>
      </div>
      {videoOn ? <VideoShellNav /> : (
      <div className="px-2.5 pb-2">
        <Button className="h-9 w-full justify-start gap-2 rounded-xl text-[13px]" onClick={props.onNew}>
          <MessageSquarePlus className="size-4" aria-hidden />
          {copy.rail.newChat}
        </Button>
        <div className="relative mt-2.5">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
          <Input
            className="h-8 rounded-xl border-transparent bg-lift/80 pl-8 text-[13px]"
            placeholder={copy.rail.search}
            aria-label={copy.rail.search}
            value={props.query}
            onChange={(e) => props.onQuery(e.target.value)}
            autoComplete="off"
            name="thread-search"
            spellCheck={false}
          />
        </div>
      </div>
      )}
      {videoOn ? null : (
      <nav aria-label={copy.rail.chats} className="min-h-0 flex-1 overflow-hidden px-1.5 pb-1">
        {empty ? (
          <EmptyState icon={<YoyoMark compact />} title={copy.rail.noChats} />
        ) : virtual ? (
          <VList className="h-full">
            {entries.map(renderEntry)}
          </VList>
        ) : (
          <div className="h-full overflow-auto">
            {entries.map(renderEntry)}
          </div>
        )}
      </nav>
      )}
      {videoOn ? <div className="min-h-0 flex-1" /> : null}
      <div className="rail-dock">
        <DockIcon
          label={copy.rail.sessions}
          hint={copy.rail.sessionsHint}
          on={props.surface === "agent"}
          onClick={props.onChats}
        >
          <MessageSquare className="size-4" aria-hidden />
        </DockIcon>
        <DockIcon
          label={copy.rail.skills}
          hint={copy.rail.skillsHint}
          on={skillsOn}
          onClick={props.onSkills}
        >
          <BookOpen className="size-4" aria-hidden />
        </DockIcon>
        <DockIcon
          label={copy.rail.video}
          hint={copy.rail.videoHint}
          on={videoOn}
          onClick={props.onVideo}
        >
          <Clapperboard className="size-4" aria-hidden />
        </DockIcon>
        <DockIcon
          label={copy.rail.harness}
          hint={activeShort ? `${copy.rail.harness} · ${activeShort}` : copy.rail.harnessHint}
          on={harnessOn}
          onClick={props.onHarness}
          mark={dirty || canary}
        >
          <GitBranch className="size-4" aria-hidden />
        </DockIcon>
        <div className="ml-auto flex items-center gap-0.5">
          <Tooltip content={props.showArchived ? copy.rail.hideArchived : copy.rail.showArchived}>
            <button
              type="button"
              className={cn(
                "dock-hit grid size-8 place-items-center rounded-xl",
                props.showArchived ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
              )}
              aria-label={props.showArchived ? copy.rail.hideArchived : copy.rail.showArchived}
              aria-pressed={!!props.showArchived}
              onClick={props.onToggleArchived}
            >
              <Archive className="size-4" />
            </button>
          </Tooltip>
          <DropdownMenu open={props.noticesOpen} onOpenChange={(v) => { if (v !== props.noticesOpen) props.onToggleNotices(); }}>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="dock-hit relative grid size-8 place-items-center rounded-xl text-muted hover:bg-lift hover:text-foreground"
                aria-label={copy.rail.notifications}
                title={copy.rail.notifications}
              >
                  <Bell className="size-4" />
                  {props.notices.length ? (
                    <span className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-foreground" />
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
                "dock-hit grid size-8 place-items-center rounded-xl",
                props.surface === "settings" ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
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

function RailLabel({ label }: { label: string }) {
  return (
    <div className="px-2.5 pb-1 pt-3 text-[11px] font-medium text-muted first:pt-1.5">
      {label}
    </div>
  );
}

function ProjectRow(props: { project: VideoProject; active: boolean; onSelect: () => void }) {
  const copy = useCopy();
  const p = props.project;
  const Icon = p.kind === "drama" ? Film : LayoutGrid;
  const fallback = p.kind === "drama" ? copy.video.untitled : copy.video.canvas;
  return (
    <button
      type="button"
      data-testid={`video-project-${p.kind}-${p.id}`}
      aria-current={props.active ? "page" : undefined}
      className={cn(
        "mb-px ml-2 flex min-h-8 w-[calc(100%-0.5rem)] items-center gap-2 rounded-lg px-2.5 py-[6px] text-left text-[13px] transition-[background-color,color] duration-200 ease-[var(--ease-out)]",
        props.active ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
      )}
      onClick={props.onSelect}
    >
      <Icon className="size-3.5 shrink-0 opacity-70" aria-hidden />
      <span className="truncate">{displayTitle(p.title, fallback)}</span>
    </button>
  );
}

function SpaceHeader(props: {
  label: string;
  path: string;
  collapsed: boolean;
  onToggle: () => void;
  onNew?: () => void;
}) {
  const copy = useCopy();
  return (
    <div className="group/space flex items-center gap-0.5 px-1.5 pb-0.5 pt-3 first:pt-1.5">
      <button
        type="button"
        className="flex min-w-0 flex-1 items-center gap-1.5 rounded-lg px-1.5 py-1 text-left text-[12px] font-medium tracking-[-0.01em] text-foreground/80 transition-colors hover:bg-lift/55 hover:text-foreground"
        onClick={props.onToggle}
        aria-expanded={!props.collapsed}
        title={props.path}
      >
        {props.collapsed ? <ChevronRight className="size-3.5 shrink-0 text-muted" aria-hidden /> : <ChevronDown className="size-3.5 shrink-0 text-muted" aria-hidden />}
        <Folder className="size-3.5 shrink-0 text-muted" aria-hidden />
        <span className="truncate">{props.label}</span>
      </button>
      {props.onNew ? (
        <Tooltip content={copy.rail.newInSpace}>
          <button
            type="button"
            className="grid size-6 place-items-center rounded-md text-muted opacity-0 transition-opacity hover:bg-lift hover:text-foreground group-hover/space:opacity-100 focus-visible:opacity-100"
            aria-label={copy.rail.newInSpace}
            onClick={(e) => {
              e.stopPropagation();
              props.onNew?.();
            }}
          >
            <Plus className="size-3.5" />
          </button>
        </Tooltip>
      ) : null}
    </div>
  );
}

function DockIcon(props: {
  label: string;
  hint?: string;
  on?: boolean;
  mark?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip content={props.hint || props.label} side="top">
      <button
        type="button"
        title={props.hint || props.label}
        aria-label={props.label}
        aria-current={props.on ? "page" : undefined}
        className={cn(
          "dock-hit relative grid size-8 place-items-center rounded-xl",
          props.on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
        )}
        onClick={props.onClick}
      >
        {props.children}
        {props.mark ? <span className="absolute right-1.5 top-1.5 size-1.5 rounded-full bg-accent" /> : null}
      </button>
    </Tooltip>
  );
}

function ThreadRow(props: {
  thread: Thread;
  activeId: string;
  running: Record<string, boolean>;
  surface: Surface;
  nested?: boolean;
  onSelect: (t: Thread) => void;
  onLab: (lab: Lab) => void;
  onPin?: (t: Thread, pinned: boolean) => void;
  onArchive?: (t: Thread, archived: boolean) => void;
  onDelete?: (t: Thread) => void;
  onRename?: (t: Thread, title: string) => void;
  onFork?: (t: Thread) => void;
  onPopOut?: (t: Thread) => void;
  onOpenEditor?: (t: Thread) => void;
  onOpenTerminal?: (t: Thread) => void;
}) {
  const copy = useCopy();
  const t = props.thread;
  const run = !!props.running[t.id];
  const onChat = props.surface === "agent" || props.surface === "video";
  const active = onChat && t.id === props.activeId && threadChannel(t) === (props.surface === "video" ? "video" : "agent");
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(t.title || "");
  const [menuOpen, setMenuOpen] = useState(false);
  const [metaOpen, setMetaOpen] = useState(false);
  const skipBlur = useRef(false);
  const rowRef = useRef<HTMLDivElement>(null);
  const openMetaT = useRef(0);
  const closeMetaT = useRef(0);

  useEffect(() => {
    if (!editing) setDraft(t.title || "");
  }, [t.title, editing]);

  useEffect(() => () => {
    window.clearTimeout(openMetaT.current);
    window.clearTimeout(closeMetaT.current);
  }, []);

  useEffect(() => {
    if (!menuOpen) return;
    window.clearTimeout(openMetaT.current);
    window.clearTimeout(closeMetaT.current);
    setMetaOpen(false);
  }, [menuOpen]);

  function scheduleMetaOpen() {
    if (menuOpen) return;
    window.clearTimeout(closeMetaT.current);
    window.clearTimeout(openMetaT.current);
    openMetaT.current = window.setTimeout(() => setMetaOpen(true), 280);
  }

  function scheduleMetaClose() {
    window.clearTimeout(openMetaT.current);
    closeMetaT.current = window.setTimeout(() => setMetaOpen(false), 160);
  }

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
      <div className={cn("mb-px flex w-full items-center gap-2 rounded-lg bg-lift px-2.5 py-[6px]", props.nested && "ml-2 w-[calc(100%-0.5rem)]")}>
        {run ? <span className="pulse-dot is-accent shrink-0" aria-hidden /> : null}
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
        ref={rowRef}
        className={cn(
          "group relative mb-px flex min-h-8 w-full items-center rounded-lg transition-[background-color,color] duration-[var(--duration-fast)] ease-[var(--ease-out)]",
          props.nested && "ml-2 w-[calc(100%-0.5rem)]",
          active ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
        )}
        onMouseEnter={scheduleMetaOpen}
        onMouseLeave={scheduleMetaClose}
      >
        <button
          type="button"
          aria-current={active ? "page" : undefined}
          className="flex min-w-0 flex-1 items-center gap-2 px-2.5 py-[6px] text-left"
          onClick={() => {
            props.onSelect(t);
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
          {active ? <span className="absolute left-1 top-2 bottom-2 w-[2px] rounded-full bg-accent" aria-hidden /> : null}
          {run ? <span className="pulse-dot is-accent shrink-0" aria-hidden /> : null}
          <span className="min-w-0 flex-1">
            <span className="flex items-center gap-1">
              {t.pinned ? <Pin className="size-3 text-muted" aria-hidden /> : null}
              <span className={cn("block truncate text-[13px]", active ? "font-medium text-foreground" : "font-normal")}>{displayTitle(t.title, copy.rail.untitled)}</span>
            </span>
            {run || t.archived || t.isolate || t.interrupted || (t.queued || 0) > 0 ? (
              <span className="block truncate text-[11px] text-muted">
                {run ? copy.rail.running : t.interrupted ? copy.rail.interrupted : (t.queued || 0) > 0 ? copy.rail.queued : t.archived ? copy.rail.archived : copy.rail.isolated}
              </span>
            ) : null}
          </span>
        </button>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="mt-1 mr-1 grid size-7 shrink-0 place-items-center rounded-md text-muted opacity-0 hover:bg-lift hover:text-foreground group-hover:opacity-100 group-focus-within:opacity-100 data-[state=open]:bg-lift data-[state=open]:text-foreground data-[state=open]:opacity-100 max-sm:opacity-100"
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
        <DropdownMenuItem onSelect={() => props.onPopOut?.(t)}>{copy.rail.popOut}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => props.onOpenEditor?.(t)}>{copy.rail.openEditor}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => props.onOpenTerminal?.(t)}>{copy.rail.openTerminal}</DropdownMenuItem>
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
      {metaOpen && !menuOpen ? (
        <SessionMetaCard
          thread={t}
          running={run}
          anchorRef={rowRef}
          onMouseEnter={scheduleMetaOpen}
          onMouseLeave={scheduleMetaClose}
        />
      ) : null}
    </DropdownMenu>
  );
}

function formatCreated(iso: string): string {
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return "";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(t);
}

function SessionMetaCard(props: {
  thread: Thread;
  running: boolean;
  anchorRef: RefObject<HTMLDivElement | null>;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}) {
  const copy = useCopy();
  const t = props.thread;
  const cardRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ left: 0, top: 0, ready: false });
  const [copied, setCopied] = useState(false);
  const title = displayTitle(t.title, copy.rail.untitled);
  const created = formatCreated(t.createdAt);
  const harness = shortHash(t.harness);
  const workspace = t.originWorkspace || t.workspace;
  const chips = [
    props.running ? copy.rail.running : "",
    t.pinned ? copy.rail.pinned : "",
    t.archived ? copy.rail.archived : "",
    t.isolate ? copy.rail.isolated : "",
  ].filter(Boolean);

  const place = useCallback(() => {
    const anchor = props.anchorRef.current;
    const card = cardRef.current;
    if (!anchor) return;
    const r = anchor.getBoundingClientRect();
    if (r.bottom < 8 || r.top > window.innerHeight - 8) return;
    const w = card?.offsetWidth || 272;
    const h = card?.offsetHeight || 200;
    let left = r.right + 8;
    if (left + w > window.innerWidth - 8) left = Math.max(8, r.left - 8 - w);
    let top = r.top;
    if (top + h > window.innerHeight - 8) top = Math.max(8, window.innerHeight - 8 - h);
    setPos({ left, top, ready: true });
  }, [props.anchorRef]);

  useLayoutEffect(() => {
    place();
  }, [place, copied, chips.length]);

  useEffect(() => {
    window.addEventListener("scroll", place, true);
    window.addEventListener("resize", place);
    return () => {
      window.removeEventListener("scroll", place, true);
      window.removeEventListener("resize", place);
    };
  }, [place]);

  if (typeof document === "undefined") return null;

  return createPortal(
    <div
      ref={cardRef}
      role="complementary"
      aria-label={copy.rail.meta}
      data-testid="session-meta-card"
      className="fixed z-[60] w-[272px] overflow-hidden rounded-2xl border border-border bg-card shadow-[var(--shadow-popover)]"
      style={{ left: pos.left, top: pos.top, visibility: pos.ready ? "visible" : "hidden" }}
      onMouseEnter={props.onMouseEnter}
      onMouseLeave={props.onMouseLeave}
    >
      <div className="px-3 py-2.5">
        <div className="text-[11px] font-medium text-muted">{copy.rail.meta}</div>
        <div className="mt-1 truncate text-[13px] font-medium text-foreground">{title}</div>
        {chips.length ? (
          <div className="mt-1.5 flex flex-wrap gap-1">
            {chips.map((c) => (
              <span key={c} className="rounded-md bg-lift px-1.5 py-0.5 text-[10px] text-muted">{c}</span>
            ))}
          </div>
        ) : null}
      </div>
      <div className="border-t border-border/50 px-3 py-2">
        <div className="text-[11px] font-medium text-muted">{copy.rail.sessionId}</div>
        <div className="mt-1 flex items-center gap-1">
          <span className="min-w-0 flex-1 truncate font-mono text-[12px] text-foreground" title={t.id}>{t.id}</span>
          <button
            type="button"
            className="inline-flex h-6 shrink-0 items-center gap-1 rounded-md px-2 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground"
            aria-label={copy.rail.copyId}
            onClick={async () => {
              if (!(await writeClipboard(t.id))) return;
              setCopied(true);
              window.setTimeout(() => setCopied(false), 1200);
            }}
          >
            <IconSwap
              on={copied}
              className="size-3"
              off={<CopyIcon className="size-3" aria-hidden />}
              live={<Check className="size-3" aria-hidden />}
            />
            {copied ? copy.rail.copied : copy.rail.copy}
          </button>
        </div>
      </div>
      {workspace ? (
        <MetaLine label={copy.rail.workspace} value={workspace} title={workspace} />
      ) : null}
      {created ? <MetaLine label={copy.rail.created} value={created} /> : null}
      {t.model ? <MetaLine label={copy.rail.model} value={t.model} /> : null}
      {harness ? <MetaLine label={copy.rail.harness} value={harness} mono title={t.harness} /> : null}
    </div>,
    document.body,
  );
}

function MetaLine(props: { label: string; value: string; title?: string; mono?: boolean }) {
  return (
    <div className="border-t border-border/50 px-3 py-2">
      <div className="text-[11px] font-medium text-muted">{props.label}</div>
      <div
        className={cn("mt-1 truncate text-[12px] text-foreground", props.mono && "font-mono text-[12px]")}
        title={props.title || props.value}
      >
        {props.value}
      </div>
    </div>
  );
}
