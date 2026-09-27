import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  Activity,
  ExternalLink,
  Files,
  Globe,
  RefreshCw,
  TextQuote,
} from "lucide-react";
import { Tooltip } from "../components/ui/tooltip";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import type { DiffMode } from "../lib/split-diff";
import * as api from "../lib/client";
import type { FileHit, Hunk, SessionTrace, SpillBlob, Thread } from "../lib/protocol";
import { coerceInspTab, type InspTab } from "../lib/store";
import { ancestorPaths, FileTree, fileMarks, nestHits, normPath } from "./FileTree";
import { TracePanel } from "./TracePanel";
import { BrowserPane } from "./BrowserPane";
import { FilePeek } from "./FilePeek";

export function Inspector(props: {
  tab: InspTab;
  onTab: (t: InspTab) => void;
  diff: string;
  hunks: Hunk[];
  selected: Record<string, boolean>;
  mode: DiffMode;
  onMode: (m: DiffMode) => void;
  onToggle: (id: string) => void;
  onSelect?: (ids: string[], selected: boolean) => void;
  onApply: () => void;
  onRefreshDiff: () => void;
  onQuote?: (text: string) => void;
  sessionId?: string;
  running?: boolean;
  trace?: SessionTrace | null;
  onRefreshTrace?: () => void;
  onLoadSpill?: (id: string) => Promise<SpillBlob>;
  thread?: Thread | null;
  onOpenPath?: (path: string) => void;
  workspace?: string;
  focusFile?: string;
  peekInline?: boolean;
  onPreviewFile?: (path: string) => void;
}) {
  const copy = useCopy();
  const files = withFocus(fileNames(props.diff), props.focusFile);
  const active = coerceInspTab(props.tab) || "files";
  const dirty = files.length || props.hunks.length;
  const tabs: { id: InspTab; label: string; icon: typeof Files; count?: number }[] = [
    { id: "files", label: copy.review.files, icon: Files, count: dirty || undefined },
    { id: "browser", label: copy.review.browser, icon: Globe },
    { id: "trace", label: copy.trace.tab, icon: Activity },
  ];
  return (
    <aside className="@container flex h-full min-h-0 flex-col overflow-hidden bg-transparent">
      <div
        className="flex h-11 shrink-0 items-stretch gap-0.5 border-b border-border/50 px-2"
        role="tablist"
        aria-label={copy.review.tabs}
      >
        {tabs.map((t) => {
          const Icon = t.icon;
          const on = active === t.id;
          return (
            <button
              type="button"
              key={t.id}
              role="tab"
              aria-selected={on}
              aria-label={t.count ? `${t.label} ${t.count}` : t.label}
              title={t.label}
              className={cn(
                "relative flex min-w-0 flex-1 items-center justify-center gap-1.5 rounded-lg px-1.5 text-[12px] font-medium transition-colors duration-200 ease-[var(--ease-out)]",
                on ? "text-foreground" : "text-muted hover:text-foreground",
              )}
              onClick={() => props.onTab(t.id)}
            >
              <Icon className="size-3.5 shrink-0" aria-hidden />
              <span className="truncate">{t.label}</span>
              {t.count ? (
                <span className={cn("tabular-nums text-[11px]", on ? "text-muted" : "text-muted/70")}>{t.count}</span>
              ) : null}
              <span
                className={cn(
                  "absolute inset-x-2 -bottom-px h-[2px] rounded-full bg-accent transition-opacity duration-200 ease-[var(--ease-out)]",
                  on ? "opacity-100" : "opacity-0",
                )}
                aria-hidden
              />
            </button>
          );
        })}
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        {active === "trace" ? (
          <div className="h-full min-h-0 overflow-auto">
            <TracePanel
              sessionId={props.sessionId || ""}
              running={props.running}
              trace={props.trace || null}
              onRefresh={() => props.onRefreshTrace?.()}
              onLoadSpill={props.onLoadSpill || (async () => ({ id: "", bytes: 0, text: "", truncated: false }))}
            />
          </div>
        ) : active === "browser" ? (
          <BrowserPane workspace={props.workspace} previewPath={props.focusFile} running={props.running} />
        ) : (
          <FilesPane {...props} files={files} />
        )}
      </div>
    </aside>
  );
}

/* ---------- shared bits ---------- */

export function PaneEmpty({ title, hint, icon }: { title: string; hint?: string; icon?: ReactNode }) {
  return (
    <div className="flex h-full min-h-[10rem] flex-col items-center justify-center px-6 text-center">
      {icon ? <span className="mark-well mb-3 text-muted" aria-hidden>{icon}</span> : null}
      <p className="text-[13px] font-medium text-foreground/85">{title}</p>
      {hint ? <p className="mt-1 max-w-[22rem] text-[12px] leading-[1.55] text-muted">{hint}</p> : null}
    </div>
  );
}

function IconAction({ label, onClick, children, className }: { label: string; onClick: () => void; children: ReactNode; className?: string }) {
  return (
    <Tooltip content={label}>
      <button
        type="button"
        aria-label={label}
        className={cn(
          "grid size-6 shrink-0 place-items-center rounded-md text-muted transition-colors hover:bg-lift hover:text-foreground",
          className,
        )}
        onClick={onClick}
      >
        {children}
      </button>
    </Tooltip>
  );
}

function TextAction({ children, onClick, tone }: { children: ReactNode; onClick: () => void; tone?: "primary" | "danger" | "ghost" }) {
  return (
    <button
      type="button"
      className={cn(
        "h-6 rounded-md px-2 text-[11px] font-medium transition-colors",
        tone === "primary" && "bg-foreground text-background hover:opacity-90",
        tone === "danger" && "text-muted hover:bg-danger/10 hover:text-danger",
        (!tone || tone === "ghost") && "text-muted hover:bg-lift hover:text-foreground",
      )}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

/* ---------- Files (workspace tree + diff) ---------- */

type FilesPaneProps = Parameters<typeof Inspector>[0] & { files: string[] };

function FilesPane(props: FilesPaneProps) {
  const copy = useCopy();
  const groups = useMemo(() => groupHunks(props.hunks), [props.hunks]);
  const filesKey = props.files.join("\0");
  const [hits, setHits] = useState<FileHit[]>([]);
  const [open, setOpen] = useState(props.focusFile || "");
  useEffect(() => {
    if (!props.workspace) {
      setHits([]);
      return;
    }
    let cancel = false;
    void api.workspaceTree(props.workspace).then((list) => {
      if (!cancel) setHits(list);
    }).catch(() => {
      if (!cancel) setHits([]);
    });
    return () => {
      cancel = true;
    };
  }, [props.workspace, props.running, filesKey]);
  useEffect(() => {
    if (!props.focusFile) return;
    setOpen(props.focusFile);
  }, [props.focusFile]);

  const selectedIds = Object.entries(props.selected).filter(([, v]) => v).map(([k]) => k);
  const selectedCount = selectedIds.filter((id) => props.hunks.some((h) => h.id === id)).length;
  const setMany = (ids: string[], on: boolean) => {
    if (props.onSelect) {
      props.onSelect(ids, on);
      return;
    }
    for (const id of ids) {
      if (!!props.selected[id] !== on) props.onToggle(id);
    }
  };

  const marks = useMemo(() => fileMarks(props.diff, [...props.files, ...props.hunks.map((h) => h.file)]), [props.diff, filesKey, props.hunks]);
  const treeHits = useMemo(() => mergeHits(hits, [...props.files, props.focusFile || "", ...props.hunks.map((h) => h.file)]), [hits, filesKey, props.focusFile, props.hunks]);
  const nodes = useMemo(() => nestHits(treeHits), [treeHits]);
  const expandPaths = useMemo(() => {
    const extra = [open, props.focusFile || "", ...Object.keys(marks)];
    return extra.flatMap((p) => ancestorPaths(p));
  }, [open, props.focusFile, marks]);

  const shown = open;
  const mark = shown ? marks[normPath(shown)] : undefined;

  const tree = nodes.length ? (
    <FileTree
      nodes={nodes}
      selected={open}
      marks={marks}
      expandPaths={expandPaths}
      onSelect={(path) => {
        setOpen(path);
        props.onPreviewFile?.(path);
      }}
    />
  ) : null;

  if (!nodes.length && !props.hunks.length && !shown) {
    return (
      <div className="flex h-full min-h-0 flex-col" data-testid="review-files">
        <PaneEmpty title={copy.review.noTree} hint={copy.review.noTreeHint} icon={<Files className="size-4" />} />
      </div>
    );
  }

  if (props.peekInline && shown) {
    return (
      <div className="flex h-full min-h-0 flex-col" data-testid="review-files">
        <FilePeek
          workspace={props.workspace}
          path={shown}
          hunks={props.hunks}
          mode={props.mode}
          onMode={props.onMode}
          onClose={() => {
            setOpen("");
            props.onPreviewFile?.("");
          }}
          onQuote={props.onQuote}
          onOpenPath={props.onOpenPath}
        />
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="review-files">
      <div className="flex h-9 shrink-0 items-center gap-1 border-b border-border/60 pl-3 pr-1.5 text-[12px] text-muted">
        <span className="min-w-0 flex-1 truncate tabular-nums">
          {shown
            ? shown.replace(/\\/g, "/").split("/").pop()
            : props.hunks.length
              ? copy.review.changes.replace("{files}", String(groups.length || props.files.length || 1)).replace("{hunks}", String(props.hunks.length))
              : copy.review.files}
        </span>
        {mark ? (
          <span className={cn("shrink-0 font-mono text-[10px] font-semibold", mark === "A" ? "text-emerald-500" : mark === "D" ? "text-red-400" : "text-amber-500")}>
            {mark}
          </span>
        ) : null}
        {shown && props.onQuote ? (
          <IconAction label={copy.review.quote} onClick={() => props.onQuote?.(`@file:${shown}`)}>
            <TextQuote className="size-3.5" aria-hidden />
          </IconAction>
        ) : null}
        {shown && props.onOpenPath ? (
          <IconAction label={copy.review.open} onClick={() => props.onOpenPath?.(shown)}>
            <ExternalLink className="size-3.5" aria-hidden />
          </IconAction>
        ) : null}
        <IconAction label={copy.review.refresh} onClick={() => {
          props.onRefreshDiff();
          if (!props.workspace) return;
          void api.workspaceTree(props.workspace).then(setHits).catch(() => setHits([]));
        }}>
          <RefreshCw className="size-3.5" aria-hidden />
        </IconAction>
      </div>
      {tree ? (
        <div className="min-h-0 flex-1 overflow-hidden">
          {tree}
        </div>
      ) : (
        <PaneEmpty title={copy.review.noTree} hint={copy.review.noTreeHint} icon={<Files className="size-4" />} />
      )}
      {selectedCount > 0 ? (
        <div className="flex h-11 shrink-0 items-center gap-1.5 border-t border-border/70 px-2.5">
          <TextAction tone="primary" onClick={props.onApply}>
            {copy.review.applyN.replace("{n}", String(selectedCount))}
          </TextAction>
          <TextAction onClick={() => setMany(selectedIds, false)}>{copy.review.clear}</TextAction>
        </div>
      ) : null}
    </div>
  );
}

function mergeHits(hits: FileHit[], extra: string[]): FileHit[] {
  const have = new Set(hits.map((h) => normPath(h.path)));
  const out = [...hits];
  for (const raw of extra) {
    const n = normPath(raw);
    if (!n || have.has(n)) continue;
    out.push({ path: n, kind: "file" });
    have.add(n);
  }
  return out;
}

function groupHunks(hunks: Hunk[]): { file: string; hunks: Hunk[] }[] {
  const order: string[] = [];
  const map = new Map<string, Hunk[]>();
  for (const h of hunks) {
    const key = h.file || "";
    if (!map.has(key)) {
      map.set(key, []);
      order.push(key);
    }
    map.get(key)!.push(h);
  }
  return order.map((file) => ({ file, hunks: map.get(file)! }));
}

function fileNames(diff: string): string[] {
  const out = new Set<string>();
  for (const line of (diff || "").split("\n")) {
    if (line.startsWith("+++ b/")) out.add(line.slice(6));
    else if (line.startsWith("diff --git ")) {
      const m = line.match(/ b\/(.+)$/);
      if (m) out.add(m[1]);
    }
  }
  return [...out];
}

function withFocus(files: string[], focus?: string): string[] {
  const f = (focus || "").trim();
  if (!f) return files;
  const n = normPath(f);
  if (files.some((x) => normPath(x) === n)) return files;
  return [f, ...files];
}
