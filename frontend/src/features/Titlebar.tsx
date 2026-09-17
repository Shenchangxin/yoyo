import { useEffect, useRef, useState } from "react";
import { Folder, PanelRight } from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Tooltip } from "../components/ui/tooltip";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import type { Thread } from "../lib/protocol";
import { RunningHub } from "./RunningHub";

export function Titlebar(props: {
  workspace: string;
  onToggleInspector: () => void;
  inspector: boolean;
  title: string;
  onRename: (title: string) => void;
  runningCount?: number;
  runningThreads?: Thread[];
  onSelectRunning?: (t: Thread) => void;
  renameTick?: number;
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
      ) : (
        <button
          type="button"
          className="min-w-0 max-w-[46%] shrink truncate rounded-md px-1 py-0.5 text-left text-[13px] font-medium text-foreground hover:bg-lift/70"
          onClick={() => {
            setDraft(props.title);
            setEditing(true);
          }}
          aria-label={copy.titlebar.rename}
        >
          {props.title}
        </button>
      )}
      <div className="h-full min-w-4 flex-1" aria-hidden />
      <div className="no-drag flex shrink-0 items-center gap-1">
        {props.workspace ? (
          <Badge className="hidden max-w-[140px] truncate lg:inline-flex" title={props.workspace}>
            <Folder className="mr-1 size-3" aria-hidden />
            {shortPath(props.workspace)}
          </Badge>
        ) : null}
        {props.runningThreads && props.onSelectRunning ? (
          <RunningHub threads={props.runningThreads} onSelect={props.onSelectRunning} />
        ) : props.runningCount ? (
          <Badge className="hidden sm:inline-flex">{props.runningCount} {copy.titlebar.live}</Badge>
        ) : null}
        <Tooltip content={copy.review.toggle}>
          <Button
            variant="ghost"
            size="icon"
            onClick={props.onToggleInspector}
            aria-label={copy.review.toggle}
            aria-pressed={props.inspector}
            className={cn("shrink-0", props.inspector && "bg-lift text-foreground")}
          >
            <PanelRight />
          </Button>
        </Tooltip>
      </div>
    </div>
  );
}

function shortPath(p: string): string {
  if (!p) return "";
  const parts = p.replace(/\\/g, "/").split("/");
  return parts.slice(-2).join("/");
}
