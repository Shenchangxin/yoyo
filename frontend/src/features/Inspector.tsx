import { Activity, FileText, GitCompare } from "lucide-react";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { DiffBlock, type DiffMode } from "../lib/split-diff";
import type { Hunk, SessionTrace, SpillBlob } from "../lib/protocol";
import type { InspTab } from "../lib/store";
import { TracePanel } from "./TracePanel";

export function Inspector(props: {
  tab: InspTab;
  onTab: (t: InspTab) => void;
  diff: string;
  hunks: Hunk[];
  selected: Record<string, boolean>;
  mode: DiffMode;
  onMode: (m: DiffMode) => void;
  onToggle: (id: string) => void;
  onApply: () => void;
  onRefreshDiff: () => void;
  onQuote?: (text: string) => void;
  sessionId?: string;
  running?: boolean;
  trace?: SessionTrace | null;
  onRefreshTrace?: () => void;
  onLoadSpill?: (id: string) => Promise<SpillBlob>;
}) {
  const copy = useCopy();
  const files = fileNames(props.diff);
  const active: InspTab = props.tab === "files" || props.tab === "trace" ? props.tab : "diff";
  const tabs: { id: InspTab; label: string; icon: typeof GitCompare }[] = [
    { id: "diff", label: props.hunks.length ? `${copy.review.diff} ${props.hunks.length}` : copy.review.diff, icon: GitCompare },
    { id: "files", label: copy.review.files, icon: FileText },
    { id: "trace", label: copy.trace.tab, icon: Activity },
  ];
  return (
    <aside className="flex h-full min-h-0 flex-col overflow-hidden bg-transparent">
      <div className="flex gap-0.5 border-b border-border/80 p-1.5" role="tablist" aria-label={copy.review.tabs}>
        {tabs.map((t) => {
          const Icon = t.icon;
          return (
            <button
              type="button"
              key={t.id}
              role="tab"
              aria-selected={active === t.id}
              className={cn(
                "flex flex-1 items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[11px] font-medium transition-colors",
                active === t.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
              )}
              onClick={() => props.onTab(t.id)}
            >
              <Icon className="size-3" aria-hidden />
              <span className="hidden min-[280px]:inline">{t.label}</span>
            </button>
          );
        })}
      </div>
      <div className="min-h-0 flex-1 overflow-auto p-3">
        {active === "trace" ? (
          <TracePanel
            sessionId={props.sessionId || ""}
            running={props.running}
            trace={props.trace || null}
            onRefresh={() => props.onRefreshTrace?.()}
            onLoadSpill={props.onLoadSpill || (async () => ({ id: "", bytes: 0, text: "", truncated: false }))}
          />
        ) : active === "diff" ? (
          <>
            <div className="mb-3 flex flex-wrap gap-2">
              <Button variant="lift" size="sm" onClick={props.onRefreshDiff}>{copy.review.refresh}</Button>
              <Button size="sm" disabled={!Object.values(props.selected).some(Boolean)} onClick={props.onApply}>
                {copy.review.apply}
              </Button>
              <div className="ml-auto flex rounded-lg bg-lift p-0.5" role="group" aria-label={copy.review.diffLayout}>
                {(["unified", "split"] as DiffMode[]).map((m) => (
                  <button
                    type="button"
                    key={m}
                    aria-pressed={props.mode === m}
                    className={cn(
                      "rounded-md px-2 py-1 text-[11px]",
                      props.mode === m ? "bg-panel text-foreground" : "text-muted",
                    )}
                    onClick={() => props.onMode(m)}
                  >
                    {m === "unified" ? copy.review.unified : copy.review.split}
                  </button>
                ))}
              </div>
            </div>
            {props.hunks.length === 0 ? (
              <p className="text-xs text-muted">{copy.review.noHunks}</p>
            ) : (
              props.hunks.map((h) => (
                <label key={h.id} className="mb-2 block rounded-xl border border-border bg-card p-2">
                  <div className="mb-1 flex items-center gap-2 text-xs">
                    <input type="checkbox" checked={!!props.selected[h.id]} onChange={() => props.onToggle(h.id)} />
                    <span className="font-mono tabular-nums text-muted">{h.file ? `${h.file} · ${h.id}` : h.id}</span>
                  </div>
                  <DiffBlock
                    src={(h.header + "\n" + h.body).slice(0, 1800)}
                    mode={props.mode}
                    onLineClick={props.onQuote ? (text) => props.onQuote?.(`In ${h.file || "file"}: ${text}`) : undefined}
                  />
                </label>
              ))
            )}
          </>
        ) : files.length ? (
          <ul className="space-y-1 font-mono text-xs text-muted">
            {files.map((f) => (
              <li key={f}>
                <button
                  type="button"
                  className="w-full truncate rounded-md px-2 py-1 text-left hover:bg-lift"
                  title={copy.review.quote}
                  onClick={() => props.onQuote?.(`@file:${f}`)}
                >
                  {f}
                </button>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-xs text-muted">{copy.review.noFiles}</p>
        )}
      </div>
    </aside>
  );
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
