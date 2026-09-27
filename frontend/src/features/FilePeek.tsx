import { useEffect, useState, type ReactNode } from "react";
import { Columns2, ExternalLink, Rows3, TextQuote, X } from "lucide-react";
import { Tooltip } from "../components/ui/tooltip";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { DiffBlock, type DiffMode } from "../lib/split-diff";
import type { Hunk } from "../lib/protocol";
import { normPath } from "./FileTree";
import { WorkspaceFileView } from "./transcript/FilePreview";

export function FilePeek(props: {
  workspace?: string;
  path: string;
  hunks: Hunk[];
  mode: DiffMode;
  onMode: (m: DiffMode) => void;
  onClose: () => void;
  onQuote?: (text: string) => void;
  onOpenPath?: (path: string) => void;
}) {
  const copy = useCopy();
  const shown = props.path;
  const fileHunks = props.hunks.filter((h) => normPath(h.file) === normPath(shown));
  const canDiff = fileHunks.length > 0;
  const [view, setView] = useState<"file" | "diff">("file");
  useEffect(() => {
    setView("file");
  }, [shown]);
  const showFile = view === "file" || !canDiff;
  const name = shown.replace(/\\/g, "/").split("/").pop() || shown;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        props.onClose();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [props.onClose]);

  return (
    <div className="glass-chrome flex h-full min-h-0 flex-col overflow-hidden" data-testid="review-file-preview">
      <div className="flex h-11 shrink-0 items-center gap-1 border-b border-border/50 pl-3 pr-1.5">
        <span className="min-w-0 flex-1 truncate font-mono text-[12px] text-foreground" title={shown}>
          {name}
        </span>
        {canDiff ? (
          <div className="flex rounded-md bg-lift/70 p-0.5" role="group" aria-label={copy.review.files}>
            {([["diff", copy.review.diff], ["file", copy.review.file]] as const).map(([id, label]) => (
              <button
                key={id}
                type="button"
                aria-pressed={view === id}
                className={cn(
                  "h-6 rounded-[5px] px-2 text-[11px] font-medium transition-colors",
                  view === id ? "bg-background text-foreground" : "text-muted hover:text-foreground",
                )}
                onClick={() => setView(id)}
              >
                {label}
              </button>
            ))}
          </div>
        ) : null}
        {!showFile ? (
          <div className="flex rounded-md bg-lift/70 p-0.5" role="group" aria-label={copy.review.diffLayout}>
            {([["unified", Rows3, copy.review.unified], ["split", Columns2, copy.review.split]] as const).map(([m, Icon, label]) => (
              <Tooltip key={m} content={label}>
                <button
                  type="button"
                  aria-pressed={props.mode === m}
                  aria-label={label}
                  className={cn(
                    "grid size-6 place-items-center rounded-[5px] transition-colors",
                    props.mode === m ? "bg-background text-foreground" : "text-muted hover:text-foreground",
                  )}
                  onClick={() => props.onMode(m)}
                >
                  <Icon className="size-3.5" aria-hidden />
                </button>
              </Tooltip>
            ))}
          </div>
        ) : null}
        {props.onQuote ? (
          <PeekAction label={copy.review.quote} onClick={() => props.onQuote?.(`@file:${shown}`)}>
            <TextQuote className="size-3.5" aria-hidden />
          </PeekAction>
        ) : null}
        {props.onOpenPath ? (
          <PeekAction label={copy.review.open} onClick={() => props.onOpenPath?.(shown)}>
            <ExternalLink className="size-3.5" aria-hidden />
          </PeekAction>
        ) : null}
        <PeekAction label={copy.review.closePreview} onClick={props.onClose}>
          <X className="size-3.5" aria-hidden />
        </PeekAction>
      </div>
      <div className="min-h-0 flex-1 overflow-hidden bg-sidebar/40">
        {showFile ? (
          <div className="h-full min-h-0 overflow-hidden">
            <WorkspaceFileView workspace={props.workspace} path={shown} fill />
          </div>
        ) : (
          <div className="h-full overflow-auto">
            {fileHunks.map((h) => (
              <div key={h.id} className="border-b border-border/40 last:border-b-0">
                <div className="px-3 py-1.5 font-mono text-[11px] text-muted/75">{h.header || h.id}</div>
                <DiffBlock
                  src={h.body.slice(0, 8000)}
                  mode={props.mode}
                  onLineClick={props.onQuote ? (text) => props.onQuote?.(`In ${h.file || "file"}: ${text}`) : undefined}
                />
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function PeekAction({ label, onClick, children }: { label: string; onClick: () => void; children: ReactNode }) {
  return (
    <Tooltip content={label}>
      <button
        type="button"
        aria-label={label}
        className="grid size-7 shrink-0 place-items-center rounded-md text-muted transition-colors hover:bg-lift hover:text-foreground"
        onClick={onClick}
      >
        {children}
      </button>
    </Tooltip>
  );
}
