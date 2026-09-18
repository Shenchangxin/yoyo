import { Activity, Brain, FileText, GitCompare, Inbox } from "lucide-react";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { DiffBlock, type DiffMode } from "../lib/split-diff";
import type { Hunk, SessionTrace, SpillBlob, Thread } from "../lib/protocol";
import type { InspTab } from "../lib/store";
import { TracePanel } from "./TracePanel";
import { useEffect, useState } from "react";
import * as api from "../lib/client";
import { asArray, asBool, str } from "../lib/normalize";

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
  thread?: Thread | null;
  onOpenThread?: (id: string) => void;
  onResolve?: (id: string, decision: string) => void;
  onOpenPath?: (path: string) => void;
}) {
  const copy = useCopy();
  const files = fileNames(props.diff);
  const allowed: InspTab[] = ["diff", "files", "queue", "memory", "trace"];
  const active: InspTab = allowed.includes(props.tab) ? props.tab : "diff";
  const tabs: { id: InspTab; label: string; icon: typeof GitCompare }[] = [
    { id: "diff", label: props.hunks.length ? `${copy.review.diff} ${props.hunks.length}` : copy.review.diff, icon: GitCompare },
    { id: "files", label: copy.review.files, icon: FileText },
    { id: "queue", label: copy.review.queue, icon: Inbox },
    { id: "memory", label: copy.review.memory, icon: Brain },
    { id: "trace", label: copy.trace.tab, icon: Activity },
  ];
  return (
    <aside className="flex h-full min-h-0 flex-col overflow-hidden bg-transparent">
      <div className="flex flex-wrap gap-0.5 border-b border-border/80 p-1.5" role="tablist" aria-label={copy.review.tabs}>
        {tabs.map((t) => {
          const Icon = t.icon;
          return (
            <button
              type="button"
              key={t.id}
              role="tab"
              aria-selected={active === t.id}
              className={cn(
                "flex items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[11px] font-medium transition-colors",
                active === t.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
              )}
              onClick={() => props.onTab(t.id)}
            >
              <Icon className="size-3" aria-hidden />
              <span className="hidden min-[320px]:inline">{t.label}</span>
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
        ) : active === "queue" ? (
          <ReviewQueue onOpenThread={props.onOpenThread} onResolve={props.onResolve} />
        ) : active === "memory" ? (
          <MemoryPanel />
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
              <li key={f} className="flex items-center gap-1">
                <button
                  type="button"
                  className="min-w-0 flex-1 truncate rounded-md px-2 py-1 text-left hover:bg-lift"
                  title={copy.review.quote}
                  onClick={() => props.onQuote?.(`@file:${f}`)}
                >
                  {f}
                </button>
                {props.onOpenPath ? (
                  <button
                    type="button"
                    className="shrink-0 rounded-md px-2 py-1 text-[11px] hover:bg-lift hover:text-foreground"
                    onClick={() => props.onOpenPath?.(f)}
                  >
                    {copy.transcript.openFile}
                  </button>
                ) : null}
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

function ReviewQueue(props: {
  onOpenThread?: (id: string) => void;
  onResolve?: (id: string, decision: string) => void;
}) {
  const copy = useCopy();
  const [q, setQ] = useState<any>({});
  const refresh = () => { void api.reviewQueue().then(setQ).catch(() => {}); };
  useEffect(() => { refresh(); }, []);
  const drafts = asArray(q.drafts);
  const browser = asArray(q.browser);
  const inbox = asArray(q.inbox);
  const offers = asArray(q.offers);
  if (!drafts.length && !browser.length && !inbox.length && !offers.length) {
    return <p className="text-xs text-muted">{copy.review.noQueue}</p>;
  }
  return (
    <div className="space-y-3 text-[12px]">
      {offers.map((o: any) => {
        const id = str(o.id || o.ID);
        const req = o.request || o.Request || {};
        const session = str(req.session_id || req.SessionID);
        return (
          <div key={id} className="rounded-md border border-border p-2">
            <div className="font-medium">{str(req.action || req.Action, "approval")}</div>
            <div className="font-mono text-[11px] text-muted">{str(req.command || req.Command || req.path || req.Path)}</div>
            <div className="mt-2 flex flex-wrap gap-1.5">
              <Button size="sm" onClick={() => { void props.onResolve?.(id, "once"); refresh(); }}>{copy.review.once}</Button>
              <Button size="sm" variant="ghost" onClick={() => { void props.onResolve?.(id, "deny"); refresh(); }}>{copy.review.deny}</Button>
              {session ? (
                <Button size="sm" variant="lift" onClick={() => props.onOpenThread?.(session)}>{copy.review.openThread}</Button>
              ) : null}
            </div>
          </div>
        );
      })}
      {inbox.map((it: any) => {
        const id = str(it.id || it.ID);
        const session = str(it.session_id || it.SessionID);
        const offer = offers.find((o: any) => str((o.request || o.Request || {}).session_id || (o.request || o.Request || {}).SessionID) === session);
        const offerId = str(offer?.id || offer?.ID);
        return (
          <div key={id} className="rounded-md border border-border p-2">
            <div className="font-medium">{str(it.title || it.Title)}</div>
            <div className="text-muted">{str(it.body || it.Body).slice(0, 200)}</div>
            <div className="mt-2 flex flex-wrap gap-1.5">
              {offerId ? (
                <>
                  <Button size="sm" onClick={() => { void props.onResolve?.(offerId, "once"); void api.inboxRead(id); refresh(); }}>{copy.review.once}</Button>
                  <Button size="sm" variant="ghost" onClick={() => { void props.onResolve?.(offerId, "deny"); void api.inboxDismiss(id); refresh(); }}>{copy.review.deny}</Button>
                </>
              ) : (
                <Button size="sm" variant="ghost" onClick={() => { void api.inboxDismiss(id); refresh(); }}>{copy.review.dismiss}</Button>
              )}
              {session ? (
                <Button size="sm" variant="lift" onClick={() => { void api.inboxRead(id); props.onOpenThread?.(session); }}>{copy.review.openThread}</Button>
              ) : null}
            </div>
          </div>
        );
      })}
      {drafts.map((d: any, i: number) => (
        <div key={str(d.id || i)} className="rounded-md border border-border p-2">
          <div className="font-medium">mail draft {str(d.to)}</div>
          <div className="text-muted">{str(d.subject)}</div>
        </div>
      ))}
      {browser.map((b: any, i: number) => (
        <div key={i} className="rounded-md border border-border p-2 text-muted">
          {str(b.op)} {str(b.detail)}
        </div>
      ))}
    </div>
  );
}

function MemoryPanel() {
  const copy = useCopy();
  const [items, setItems] = useState<any[]>([]);
  const refresh = () => { void api.memoryList().then(setItems).catch(() => {}); };
  useEffect(() => { refresh(); }, []);
  if (!items.length) {
    return <p className="text-xs text-muted">{copy.review.noMemory}</p>;
  }
  const staging = items.filter((it) => asBool(it.staging ?? it.Staging));
  const trusted = items.filter((it) => !asBool(it.staging ?? it.Staging));
  return (
    <div className="space-y-4 text-[12px]">
      <MemoryGroup title={copy.review.staging} items={staging} staging refresh={refresh} />
      <MemoryGroup title={copy.review.trusted} items={trusted} refresh={refresh} />
    </div>
  );
}

function MemoryGroup(props: { title: string; items: any[]; staging?: boolean; refresh: () => void }) {
  const copy = useCopy();
  if (!props.items.length) return null;
  return (
    <div className="space-y-2">
      <div className="text-[11px] font-medium text-muted">{props.title}</div>
      {props.items.map((it) => {
        const id = str(it.id || it.ID);
        return (
          <div key={id} className="rounded-md border border-border p-2">
            <div className="text-muted">{str(it.kind || it.Kind)}</div>
            <div className="mt-0.5 whitespace-pre-wrap">{str(it.text || it.Text)}</div>
            <div className="mt-2 flex flex-wrap gap-1.5">
              {props.staging ? (
                <Button size="sm" onClick={() => { void api.memoryPromote(id).then(props.refresh); }}>{copy.review.promote}</Button>
              ) : null}
              <Button size="sm" variant="ghost" onClick={() => { void api.memoryForget(id).then(props.refresh); }}>{copy.review.forget}</Button>
            </div>
          </div>
        );
      })}
    </div>
  );
}
