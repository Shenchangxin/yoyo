import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  Activity,
  ArrowUpRight,
  Brain,
  Columns2,
  ExternalLink,
  FileText,
  GitCompare,
  Inbox,
  RefreshCw,
  Rows3,
  TextQuote,
} from "lucide-react";
import { Tooltip } from "../components/ui/tooltip";
import { Checkbox } from "../components/ui/checkbox";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { DiffBlock, type DiffMode } from "../lib/split-diff";
import type { Hunk, SessionTrace, SpillBlob, Thread } from "../lib/protocol";
import type { InspTab } from "../lib/store";
import { TracePanel } from "./TracePanel";
import { WorkspaceFileView } from "./transcript/FilePreview";
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
  onOpenThread?: (id: string) => void;
  onResolve?: (id: string, decision: string) => void;
  onOpenPath?: (path: string) => void;
  workspace?: string;
  focusFile?: string;
}) {
  const copy = useCopy();
  const files = withFocus(fileNames(props.diff), props.focusFile);
  const allowed: InspTab[] = ["diff", "files", "queue", "memory", "trace"];
  const active: InspTab = allowed.includes(props.tab) ? props.tab : "diff";
  const tabs: { id: InspTab; label: string; icon: typeof GitCompare; count?: number }[] = [
    { id: "diff", label: copy.review.diff, icon: GitCompare, count: props.hunks.length || undefined },
    { id: "files", label: copy.review.files, icon: FileText, count: files.length || undefined },
    { id: "queue", label: copy.review.queue, icon: Inbox },
    { id: "memory", label: copy.review.memory, icon: Brain },
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
              <Icon className="size-3.5 shrink-0 @[320px]:hidden" aria-hidden />
              <span className="hidden truncate @[320px]:inline">{t.label}</span>
              {t.count ? (
                <span className={cn("tabular-nums text-[10.5px]", on ? "text-muted" : "text-muted/70")}>{t.count}</span>
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
        ) : active === "queue" ? (
          <div className="h-full min-h-0 overflow-auto">
            <ReviewQueue onOpenThread={props.onOpenThread} onResolve={props.onResolve} />
          </div>
        ) : active === "memory" ? (
          <div className="h-full min-h-0 overflow-auto">
            <MemoryPanel />
          </div>
        ) : active === "diff" ? (
          <DiffPane {...props} files={files} />
        ) : (
          <FilesPane files={files} workspace={props.workspace} focusFile={props.focusFile} onQuote={props.onQuote} onOpenPath={props.onOpenPath} />
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
      <p className="text-[12.5px] font-medium text-foreground/85">{title}</p>
      {hint ? <p className="mt-1 max-w-[22rem] text-[11.5px] leading-[1.55] text-muted">{hint}</p> : null}
    </div>
  );
}

function SectionLabel({ children, count }: { children: ReactNode; count?: number }) {
  return (
    <div className="flex h-8 items-center gap-2 px-3 text-[11px] font-medium text-muted">
      <span>{children}</span>
      {count ? <span className="tabular-nums normal-case tracking-normal text-muted/60">{count}</span> : null}
    </div>
  );
}

function FilePath({ path, className }: { path: string; className?: string }) {
  const norm = path.replace(/\\/g, "/");
  const i = norm.lastIndexOf("/");
  const dir = i >= 0 ? norm.slice(0, i + 1) : "";
  const base = i >= 0 ? norm.slice(i + 1) : norm;
  return (
    <span className={cn("min-w-0 truncate text-left font-mono text-[11.5px]", className)} title={path} dir="rtl">
      <bdi>
        {dir ? <span className="text-muted/70">{dir}</span> : null}
        <span className="text-foreground/90">{base}</span>
      </bdi>
    </span>
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

/* ---------- Diff ---------- */

type DiffPaneProps = Parameters<typeof Inspector>[0] & { files: string[] };

function DiffPane(props: DiffPaneProps) {
  const copy = useCopy();
  const groups = useMemo(() => groupHunks(props.hunks), [props.hunks]);
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
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-9 shrink-0 items-center gap-1 border-b border-border/60 pl-3 pr-1.5 text-[11.5px] text-muted">
        <span className="min-w-0 flex-1 truncate tabular-nums">
          {props.hunks.length
            ? copy.review.changes.replace("{files}", String(groups.length)).replace("{hunks}", String(props.hunks.length))
            : copy.review.noHunks}
        </span>
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
        <IconAction label={copy.review.refresh} onClick={props.onRefreshDiff}>
          <RefreshCw className="size-3.5" aria-hidden />
        </IconAction>
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        {groups.length === 0 ? (
          <PaneEmpty title={copy.review.noHunks} hint={copy.review.noHunksHint} icon={<GitCompare className="size-4" />} />
        ) : (
          groups.map((g) => {
            const ids = g.hunks.map((h) => h.id);
            const on = ids.filter((id) => props.selected[id]).length;
            return (
              <section key={g.file} className="border-b border-border/50 last:border-b-0">
                <header className="sticky top-0 z-[1] flex h-8 items-center gap-2 bg-sidebar px-3">
                  <Checkbox
                    aria-label={copy.review.selectFile}
                    checked={on === ids.length}
                    indeterminate={on > 0 && on < ids.length}
                    onChange={() => setMany(ids, on !== ids.length)}
                  />
                  <FilePath path={g.file} className="flex-1 font-medium" />
                  <span className="shrink-0 tabular-nums text-[10.5px] text-muted/70">
                    {g.hunks.length === 1 ? copy.review.hunkOne : copy.review.hunksCount.replace("{n}", String(g.hunks.length))}
                  </span>
                </header>
                {g.hunks.map((h) => (
                  <div key={h.id} className="border-b border-border/40 last:border-b-0">
                    <label className="flex h-7 cursor-pointer items-center gap-2 px-3 text-[11px] transition-colors hover:bg-lift/40">
                      <Checkbox checked={!!props.selected[h.id]} onChange={() => props.onToggle(h.id)} />
                      <span className="min-w-0 flex-1 truncate font-mono text-muted/75">{h.header || h.id}</span>
                    </label>
                    <div className="pb-1">
                      <DiffBlock
                        src={h.body.slice(0, 1800)}
                        mode={props.mode}
                        onLineClick={props.onQuote ? (text) => props.onQuote?.(`In ${h.file || "file"}: ${text}`) : undefined}
                      />
                    </div>
                  </div>
                ))}
              </section>
            );
          })
        )}
      </div>
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

function normPath(p: string): string {
  return p.replace(/\\/g, "/").replace(/^\.\//, "");
}

function withFocus(files: string[], focus?: string): string[] {
  const f = (focus || "").trim();
  if (!f) return files;
  const n = normPath(f);
  if (files.some((x) => normPath(x) === n)) return files;
  return [f, ...files];
}

/* ---------- Files ---------- */

function FilesPane({
  files,
  workspace,
  focusFile,
  onQuote,
  onOpenPath,
}: {
  files: string[];
  workspace?: string;
  focusFile?: string;
  onQuote?: (t: string) => void;
  onOpenPath?: (p: string) => void;
}) {
  const copy = useCopy();
  const [open, setOpen] = useState(focusFile || "");
  useEffect(() => {
    if (focusFile) setOpen(focusFile);
  }, [focusFile]);
  if (!files.length) {
    return <PaneEmpty title={copy.review.noFiles} hint={copy.review.noHunksHint} icon={<FileText className="size-4" />} />;
  }
  const shown = files.find((f) => normPath(open) === normPath(f));
  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="review-files">
      <ul className={cn("overflow-auto", shown ? "max-h-36 shrink-0" : "min-h-0 flex-1")}>
        {files.map((f) => {
          const on = normPath(open) === normPath(f);
          return (
            <li key={f} className="border-b border-border/40 last:border-b-0">
              <div className={cn("group/file flex h-8 items-center gap-2 pl-3 pr-1.5 transition-colors hover:bg-lift/40", on && "bg-lift/50")}>
                <button
                  type="button"
                  className="flex min-w-0 flex-1 items-center gap-2 text-left"
                  aria-expanded={on}
                  onClick={() => setOpen(on ? "" : f)}
                >
                  <FileText className="size-3.5 shrink-0 text-muted/60" aria-hidden />
                  <FilePath path={f} className="flex-1" />
                </button>
                <span className="flex shrink-0 items-center opacity-0 transition-opacity group-hover/file:opacity-100 group-focus-within/file:opacity-100">
                  {onQuote ? (
                    <IconAction label={copy.review.quote} onClick={() => onQuote(`@file:${f}`)}>
                      <TextQuote className="size-3.5" aria-hidden />
                    </IconAction>
                  ) : null}
                  {onOpenPath ? (
                    <IconAction label={copy.review.open} onClick={() => onOpenPath(f)}>
                      <ExternalLink className="size-3.5" aria-hidden />
                    </IconAction>
                  ) : null}
                </span>
              </div>
            </li>
          );
        })}
      </ul>
      {shown ? (
        <div className="min-h-0 flex-1 overflow-hidden border-t border-border/50 bg-sidebar/40" data-testid="review-file-preview">
          <WorkspaceFileView workspace={workspace} path={shown} fill />
        </div>
      ) : null}
    </div>
  );
}

/* ---------- Queue ---------- */

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
    return <PaneEmpty title={copy.review.nothingWaiting} hint={copy.review.noQueue} icon={<Inbox className="size-4" />} />;
  }
  return (
    <div className="pb-2">
      {offers.length ? (
        <section>
          <SectionLabel count={offers.length}>{copy.review.approvals}</SectionLabel>
          <ul className="divide-y divide-border/50 border-y border-border/50">
            {offers.map((o: any) => {
              const id = str(o.id || o.ID);
              const req = o.request || o.Request || {};
              const session = str(req.session_id || req.SessionID);
              return (
                <li key={id} className="u-row-hover px-3 py-2.5">
                  <div className="flex items-start gap-2">
                    <div className="min-w-0 flex-1">
                      <div className="text-[12.5px] font-medium text-foreground">{str(req.action || req.Action, "approval")}</div>
                      <div className="mt-0.5 truncate font-mono text-[11px] text-muted">{str(req.command || req.Command || req.path || req.Path)}</div>
                    </div>
                    {session && props.onOpenThread ? (
                      <IconAction label={copy.review.openThread} onClick={() => props.onOpenThread?.(session)}>
                        <ArrowUpRight className="size-3.5" aria-hidden />
                      </IconAction>
                    ) : null}
                  </div>
                  <div className="mt-2 flex items-center gap-1">
                    <TextAction tone="primary" onClick={() => { void props.onResolve?.(id, "once"); refresh(); }}>{copy.review.once}</TextAction>
                    <TextAction tone="danger" onClick={() => { void props.onResolve?.(id, "deny"); refresh(); }}>{copy.review.deny}</TextAction>
                  </div>
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
      {inbox.length ? (
        <section>
          <SectionLabel count={inbox.length}>{copy.review.inbox}</SectionLabel>
          <ul className="divide-y divide-border/50 border-y border-border/50">
            {inbox.map((it: any) => {
              const id = str(it.id || it.ID);
              const session = str(it.session_id || it.SessionID);
              const offer = offers.find((o: any) => str((o.request || o.Request || {}).session_id || (o.request || o.Request || {}).SessionID) === session);
              const offerId = str(offer?.id || offer?.ID);
              return (
                <li key={id} className="u-row-hover px-3 py-2.5">
                  <div className="flex items-start gap-2">
                    <div className="min-w-0 flex-1">
                      <div className="text-[12.5px] font-medium text-foreground">{str(it.title || it.Title)}</div>
                      <div className="mt-0.5 text-[11.5px] leading-[1.5] text-muted">{str(it.body || it.Body).slice(0, 200)}</div>
                    </div>
                    {session && props.onOpenThread ? (
                      <IconAction label={copy.review.openThread} onClick={() => { void api.inboxRead(id); props.onOpenThread?.(session); }}>
                        <ArrowUpRight className="size-3.5" aria-hidden />
                      </IconAction>
                    ) : null}
                  </div>
                  <div className="mt-2 flex items-center gap-1">
                    {offerId ? (
                      <>
                        <TextAction tone="primary" onClick={() => { void props.onResolve?.(offerId, "once"); void api.inboxRead(id); refresh(); }}>{copy.review.once}</TextAction>
                        <TextAction tone="danger" onClick={() => { void props.onResolve?.(offerId, "deny"); void api.inboxDismiss(id); refresh(); }}>{copy.review.deny}</TextAction>
                      </>
                    ) : (
                      <TextAction onClick={() => { void api.inboxDismiss(id); refresh(); }}>{copy.review.dismiss}</TextAction>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
      {drafts.length ? (
        <section>
          <SectionLabel count={drafts.length}>{copy.review.drafts}</SectionLabel>
          <ul className="divide-y divide-border/50 border-y border-border/50">
            {drafts.map((d: any, i: number) => (
              <li key={str(d.id || i)} className="u-row-hover px-3 py-2.5">
                <div className="text-[12.5px] font-medium text-foreground">{str(d.to)}</div>
                <div className="mt-0.5 text-[11.5px] text-muted">{str(d.subject)}</div>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {browser.length ? (
        <section>
          <SectionLabel count={browser.length}>{copy.review.browserActions}</SectionLabel>
          <ul className="divide-y divide-border/50 border-y border-border/50">
            {browser.map((b: any, i: number) => (
              <li key={i} className="u-row-hover px-3 py-2 font-mono text-[11px] text-muted">
                <span className="text-foreground/80">{str(b.op)}</span> {str(b.detail)}
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}

/* ---------- Memory ---------- */

function MemoryPanel() {
  const copy = useCopy();
  const [items, setItems] = useState<any[]>([]);
  const refresh = () => { void api.memoryList().then(setItems).catch(() => {}); };
  useEffect(() => { refresh(); }, []);
  if (!items.length) {
    return <PaneEmpty title={copy.review.noMemory} hint={copy.review.noMemoryHint} icon={<Brain className="size-4" />} />;
  }
  const staging = items.filter((it) => asBool(it.staging ?? it.Staging));
  const trusted = items.filter((it) => !asBool(it.staging ?? it.Staging));
  return (
    <div className="pb-2">
      <MemoryGroup title={copy.review.staging} items={staging} staging refresh={refresh} />
      <MemoryGroup title={copy.review.trusted} items={trusted} refresh={refresh} />
    </div>
  );
}

function MemoryGroup(props: { title: string; items: any[]; staging?: boolean; refresh: () => void }) {
  const copy = useCopy();
  if (!props.items.length) return null;
  return (
    <section>
      <SectionLabel count={props.items.length}>{props.title}</SectionLabel>
      <ul className="divide-y divide-border/50 border-y border-border/50">
        {props.items.map((it) => {
          const id = str(it.id || it.ID);
          return (
            <li key={id} className="group/mem px-3 py-2.5">
              <div className="text-[10.5px] font-medium uppercase tracking-[0.06em] text-muted/70">{str(it.kind || it.Kind)}</div>
              <div className="mt-0.5 whitespace-pre-wrap text-[12.5px] leading-[1.55] text-foreground/90">{str(it.text || it.Text)}</div>
              <div className="mt-1.5 flex items-center gap-1 opacity-70 transition-opacity group-hover/mem:opacity-100">
                {props.staging ? (
                  <TextAction tone="primary" onClick={() => { void api.memoryPromote(id).then(props.refresh); }}>{copy.review.promote}</TextAction>
                ) : null}
                <TextAction tone="danger" onClick={() => { void api.memoryForget(id).then(props.refresh); }}>{copy.review.forget}</TextAction>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
