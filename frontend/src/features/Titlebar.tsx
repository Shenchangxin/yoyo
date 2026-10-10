import { useEffect, useRef, useState } from "react";
import { Bell, Pause, Play, PanelRight } from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Tooltip } from "../components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "../components/ui/dropdown-menu";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import type { Notice, RunStatus, Thread } from "../lib/protocol";
import { RunningHub } from "./RunningHub";
import * as api from "../lib/client";
import { InboxMenu, inboxBadgeCount } from "./InboxMenu";

export function Titlebar(props: {
  onToggleInspector: () => void;
  inspector: boolean;
  inspectLabel?: string;
  hideInspector?: boolean;
  hideInbox?: boolean;
  title: string;
  onRename: (title: string) => void;
  runningCount?: number;
  runningThreads?: Thread[];
  runningStatus?: RunStatus[];
  onSelectRunning?: (t: Thread) => void;
  renameTick?: number;
  onOpenThread?: (id: string) => void;
  onResolve?: (id: string, decision: string, answer?: string) => void;
  notices?: Notice[];
  onNotice?: (n: Notice) => void;
  onClearNotices?: () => void;
  paused?: boolean;
  onPause?: () => void;
}) {
  const copy = useCopy();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(props.title);
  const skipBlur = useRef(false);

  useEffect(() => {
    if (!editing) setDraft(props.title);
  }, [props.title, editing]);

  useEffect(() => {
    if (!props.renameTick) return;
    skipBlur.current = false;
    setDraft(props.title);
    setEditing(true);
  }, [props.renameTick]);

  function commit() {
    if (skipBlur.current) {
      skipBlur.current = false;
      setDraft(props.title);
      setEditing(false);
      return;
    }
    const title = draft.trim();
    setEditing(false);
    if (!title || title === props.title) {
      setDraft(props.title);
      return;
    }
    props.onRename(title);
  }

  return (
    <div className="flex min-w-0 flex-1 items-center gap-2">
      {editing ? (
        <input
          autoFocus
          aria-label={copy.titlebar.rename}
          autoComplete="off"
          spellCheck={false}
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
              e.preventDefault();
              skipBlur.current = true;
              setDraft(props.title);
              setEditing(false);
            }
          }}
        />
      ) : props.title ? (
        <button
          type="button"
          className="min-w-0 max-w-[46%] shrink truncate rounded-lg px-1.5 py-0.5 text-left text-[14px] font-medium tracking-[-0.02em] text-foreground hover:bg-lift/70"
          onClick={() => {
            setDraft(props.title);
            setEditing(true);
          }}
          aria-label={copy.titlebar.rename}
        >
          {props.title}
        </button>
      ) : null}
      <div className="h-full min-w-4 flex-1" aria-hidden />
      <div className="no-drag flex shrink-0 items-center gap-1">
        {props.onPause ? (
          <Tooltip content={props.paused ? copy.titlebar.resume : copy.titlebar.pause}>
            <Button
              variant="ghost"
              size="icon"
              aria-label={props.paused ? copy.titlebar.resume : copy.titlebar.pause}
              aria-pressed={!!props.paused}
              onClick={props.onPause}
              className={cn("shrink-0", props.paused && "bg-lift text-foreground")}
            >
              {props.paused ? <Play /> : <Pause />}
            </Button>
          </Tooltip>
        ) : null}
        {props.runningThreads && props.onSelectRunning ? (
          <RunningHub threads={props.runningThreads} status={props.runningStatus} onSelect={props.onSelectRunning} />
        ) : props.runningCount ? (
          <Badge className="hidden sm:inline-flex">{props.runningCount} {copy.titlebar.live}</Badge>
        ) : null}
        {props.hideInbox ? null : (
          <NoticeButton
            notices={props.notices || []}
            onNotice={props.onNotice}
            onClearNotices={props.onClearNotices}
            onOpenThread={props.onOpenThread}
            onResolve={props.onResolve}
          />
        )}
        {props.hideInspector ? null : (
          <Tooltip content={props.inspectLabel || copy.review.toggle}>
            <Button
              variant="ghost"
              size="icon"
              onClick={props.onToggleInspector}
              aria-label={props.inspectLabel || copy.review.toggle}
              aria-pressed={props.inspector}
              className={cn("shrink-0", props.inspector && "bg-lift text-foreground")}
            >
              <PanelRight />
            </Button>
          </Tooltip>
        )}
      </div>
    </div>
  );
}

function NoticeButton(props: {
  notices: Notice[];
  onNotice?: (n: Notice) => void;
  onClearNotices?: () => void;
  onOpenThread?: (id: string) => void;
  onResolve?: (id: string, decision: string, answer?: string) => void;
}) {
  const copy = useCopy();
  const [n, setN] = useState(0);
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const tick = () => {
      void Promise.all([api.inboxList().catch(() => []), api.reviewQueue().catch(() => ({}))]).then(([items, q]) => {
        setN(inboxBadgeCount(items, q, props.notices.length));
      });
    };
    tick();
    const id = window.setInterval(tick, 15000);
    return () => window.clearInterval(id);
  }, [open, props.notices.length]);
  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          title={copy.titlebar.inbox}
          aria-label={copy.titlebar.inbox}
          className={cn("relative shrink-0", n > 0 && "text-foreground")}
        >
          <Bell className="size-4" aria-hidden />
          {n > 0 ? (
            <span className="absolute -right-0.5 -top-0.5 min-w-3.5 rounded-full bg-foreground px-1 text-[9px] text-background">{n}</span>
          ) : null}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[22rem] p-0">
        <InboxMenu
          notices={props.notices}
          onNotice={(n) => {
            props.onNotice?.(n);
            setOpen(false);
          }}
          onClearNotices={props.onClearNotices}
          onOpenThread={(id) => {
            props.onOpenThread?.(id);
            setOpen(false);
          }}
          onResolve={props.onResolve}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
