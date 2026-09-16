import { FileText, GitCompare, Layers, ShieldAlert } from "lucide-react";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { DiffBlock, type DiffMode } from "../lib/split-diff";
import type { Approval, ContextUsage, Hunk } from "../lib/protocol";

type Tab = "diff" | "files" | "context" | "approvals";

export function Inspector(props: {
  tab: Tab;
  onTab: (t: Tab) => void;
  diff: string;
  hunks: Hunk[];
  selected: Record<string, boolean>;
  mode: DiffMode;
  onMode: (m: DiffMode) => void;
  onToggle: (id: string) => void;
  onApply: () => void;
  onRefreshDiff: () => void;
  ctx: ContextUsage;
  approvals: Approval[];
  onResolve: (id: string, decision: string) => void;
  onQuote?: (text: string) => void;
}) {
  const copy = useCopy();
  const files = fileNames(props.diff);
  const tabs: { id: Tab; label: string; icon: typeof GitCompare }[] = [
    { id: "diff", label: props.hunks.length ? `${copy.review.diff} ${props.hunks.length}` : copy.review.diff, icon: GitCompare },
    { id: "files", label: copy.review.files, icon: FileText },
    { id: "context", label: copy.review.context, icon: Layers },
    { id: "approvals", label: props.approvals.length ? `${copy.review.ask} ${props.approvals.length}` : copy.review.ask, icon: ShieldAlert },
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
              aria-selected={props.tab === t.id}
              className={cn(
                "flex flex-1 items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[11px] font-medium transition-colors",
                props.tab === t.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
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
        {props.tab === "diff" ? (
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
        ) : null}
        {props.tab === "files" ? (
          files.length ? (
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
          ) : <p className="text-xs text-muted">{copy.review.noFiles}</p>
        ) : null}
        {props.tab === "context" ? (
          <div className="space-y-3 text-sm">
            <Row k={copy.review.elided} v={String(props.ctx.elided || 0)} />
            {props.ctx.window ? <Row k="window" v={String(props.ctx.window)} /> : null}
            {props.ctx.prefixTokens ? <Row k="prefix" v={String(props.ctx.prefixTokens)} /> : null}
            {props.ctx.schemaTokens ? <Row k="schemas" v={String(props.ctx.schemaTokens)} /> : null}
            {props.ctx.layers?.length ? (
              <div className="flex flex-wrap gap-1">
                {props.ctx.layers.map((l) => (
                  <span key={l} className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">{l}</span>
                ))}
              </div>
            ) : null}
            <p className="text-xs text-muted">
              {props.ctx.note || copy.review.contextNote}
            </p>
            <p className="text-xs text-muted">{copy.review.contextMoved}</p>
          </div>
        ) : null}
        {props.tab === "approvals" ? (
          props.approvals.length === 0 ? (
            <p className="text-xs text-muted">{copy.review.noAsk}</p>
          ) : (
            props.approvals.map((a) => (
              <div key={a.id} className="mb-2 rounded-xl border border-border bg-card p-3">
                <div className="text-sm font-medium">{a.action}</div>
                <div className="mt-1 font-mono text-xs text-muted">{a.command || a.path}</div>
                <div className="mt-2 flex flex-wrap gap-2">
                  <Button size="sm" onClick={() => props.onResolve(a.id, "once")}>{copy.review.once}</Button>
                  <Button size="sm" variant="lift" onClick={() => props.onResolve(a.id, "session")}>{copy.review.session}</Button>
                  <Button size="sm" variant="lift" onClick={() => props.onResolve(a.id, "always")}>{copy.review.always}</Button>
                  <Button size="sm" variant="danger" onClick={() => props.onResolve(a.id, "deny")}>{copy.review.deny}</Button>
                </div>
              </div>
            ))
          )
        ) : null}
      </div>
    </aside>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-center justify-between rounded-lg bg-card px-3 py-2">
      <span className="text-muted">{k}</span>
      <strong className="tabular-nums">{v}</strong>
    </div>
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
