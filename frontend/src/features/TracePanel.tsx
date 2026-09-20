import { useMemo, useState, type ReactNode } from "react";
import { Activity, Check, Copy, RefreshCw } from "lucide-react";
import { Tooltip } from "../components/ui/tooltip";
import { writeClipboard } from "../lib/clipboard";
import { useCopy } from "../lib/i18n";
import { formatTokens } from "../lib/models-dev";
import type { SessionTrace, SpillBlob, TraceArtifact, TraceEvent } from "../lib/protocol";
import { cn } from "../lib/utils";

type Filter = "all" | "exec" | "dialog" | "memory" | "error";

export function TracePanel(props: {
  sessionId: string;
  running?: boolean;
  trace: SessionTrace | null;
  onRefresh: () => void;
  onLoadSpill: (id: string) => Promise<SpillBlob>;
}) {
  const copy = useCopy();
  const [filter, setFilter] = useState<Filter>("all");
  const [open, setOpen] = useState<string | null>(null);
  const [blobs, setBlobs] = useState<Record<string, SpillBlob | "loading" | "error">>({});
  const [copied, setCopied] = useState("");
  const t = props.trace;
  const events = useMemo(() => {
    const rows = t?.events || [];
    if (filter === "all") return rows;
    if (filter === "error") return rows.filter((ev) => ev.lane === "error" || ev.error);
    return rows.filter((ev) => ev.lane === filter);
  }, [t, filter]);

  const markCopied = (id: string) => {
    setCopied(id);
    window.setTimeout(() => setCopied((cur) => (cur === id ? "" : cur)), 1200);
  };

  const loadSpill = async (id: string) => {
    if (!id || blobs[id] === "loading") return;
    setBlobs((m) => ({ ...m, [id]: "loading" }));
    try {
      const blob = await props.onLoadSpill(id);
      setBlobs((m) => ({ ...m, [id]: blob }));
    } catch {
      setBlobs((m) => ({ ...m, [id]: "error" }));
    }
  };

  if (!props.sessionId) {
    return <Empty title={copy.trace.noSession} />;
  }

  const stats = t?.stats;
  const filters: { id: Filter; label: string }[] = [
    { id: "all", label: copy.trace.filterAll },
    { id: "exec", label: copy.trace.filterExec },
    { id: "dialog", label: copy.trace.filterDialog },
    { id: "memory", label: copy.trace.filterMemory },
    { id: "error", label: copy.trace.filterError },
  ];
  const errors = stats?.errors || 0;

  return (
    <div className="flex min-h-0 flex-col" data-testid="trace-panel">
      <div className="flex h-9 shrink-0 items-center gap-1 border-b border-border/60 pl-3 pr-1.5">
        <dl className="flex min-w-0 flex-1 items-center gap-x-2.5 overflow-hidden whitespace-nowrap text-[11.5px] tabular-nums text-muted" data-testid="trace-stats">
          <Stat value={String(stats?.events || 0)} label={copy.trace.events} />
          <Dot />
          <Stat value={String(stats?.toolCalls || 0)} label={copy.trace.tools} />
          <Dot />
          <Stat value={String(errors)} label={copy.trace.errors} tone={errors > 0 ? "text-danger" : undefined} />
          <Dot />
          <Stat value={formatDuration(stats?.durationMs || 0)} />
          <Dot />
          <Stat value={formatTokens(stats?.tokens || 0)} label={copy.trace.tokens} />
          {stats?.spillBytes ? (
            <>
              <Dot />
              <Stat value={formatBytes(stats.spillBytes)} label={copy.trace.spilled} />
            </>
          ) : null}
        </dl>
        {props.running ? <span className="pulse-dot mr-1.5" aria-label={copy.rail.running} /> : null}
        {props.sessionId ? (
          <Tooltip content={copied === "session-id" ? copy.trace.copied : copy.trace.copyId}>
            <button
              type="button"
              aria-label={copy.trace.copyId}
              title={props.sessionId}
              className="mr-0.5 max-w-[9.5rem] shrink truncate rounded-md px-1 font-mono text-[11px] text-muted transition-colors hover:bg-lift hover:text-foreground"
              onClick={async () => {
                if (await writeClipboard(props.sessionId)) markCopied("session-id");
              }}
            >
              {props.sessionId}
            </button>
          </Tooltip>
        ) : null}
        <Tooltip content={copy.trace.refresh}>
          <button
            type="button"
            aria-label={copy.trace.refresh}
            className="grid size-6 shrink-0 place-items-center rounded-md text-muted transition-colors hover:bg-lift hover:text-foreground"
            onClick={props.onRefresh}
          >
            <RefreshCw className="size-3.5" aria-hidden />
          </button>
        </Tooltip>
      </div>

      {(t?.artifacts || []).length ? (
        <section>
          <div className="flex h-8 items-center px-3 text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80">
            {copy.trace.artifacts}
            <span className="ml-2 tabular-nums normal-case tracking-normal text-muted/60">{t!.artifacts.length}</span>
          </div>
          <ul className="divide-y divide-border/50 border-y border-border/50">
            {t!.artifacts.map((art) => (
              <ArtifactRow
                key={`${art.kind}:${art.id}`}
                art={art}
                blob={art.kind === "spill" ? blobs[art.id] : art.preview ? { id: art.id, bytes: art.bytes, text: art.preview, truncated: false } : undefined}
                onOpen={() => {
                  const key = `art:${art.kind}:${art.id}`;
                  if (open === key) {
                    setOpen(null);
                    return;
                  }
                  if (art.kind === "spill") void loadSpill(art.id);
                  setOpen(key);
                }}
                open={open === `art:${art.kind}:${art.id}`}
                onCopy={async (text) => {
                  if (await writeClipboard(text)) markCopied(`art:${art.id}`);
                }}
                copied={copied === `art:${art.id}`}
              />
            ))}
          </ul>
        </section>
      ) : null}

      <section className="min-h-0">
        <div className="flex h-9 items-center gap-1 px-1.5" role="tablist" aria-label={copy.trace.timeline}>
          {filters.map((f) => (
            <button
              type="button"
              key={f.id}
              role="tab"
              aria-selected={filter === f.id}
              className={cn(
                "h-6 rounded-md px-2 text-[11px] font-medium transition-colors",
                filter === f.id ? "bg-lift text-foreground" : "text-muted hover:text-foreground",
              )}
              onClick={() => setFilter(f.id)}
            >
              {f.label}
            </button>
          ))}
        </div>
        {!t || (t.events.length === 0 && filter === "all") ? (
          <Empty title={copy.trace.empty} icon={<Activity className="size-4" />} />
        ) : events.length === 0 ? (
          <Empty title={copy.trace.noMatch} />
        ) : (
          <ol className="divide-y divide-border/40 border-t border-border/50">
            {events.map((ev) => {
              const key = eventKey(ev);
              const expanded = open === key;
              return (
                <li key={key} data-testid="trace-event" data-type={ev.type}>
                  <button
                    type="button"
                    aria-expanded={expanded}
                    className={cn(
                      "grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-baseline gap-x-2.5 px-3 py-1.5 text-left transition-colors hover:bg-lift/40",
                      expanded && "bg-lift/30",
                    )}
                    onClick={() => setOpen(expanded ? null : key)}
                  >
                    <span className="font-mono text-[10.5px] tabular-nums text-muted/60">{formatTime(ev.ts)}</span>
                    <span className="flex min-w-0 items-baseline gap-2">
                      <span className={cn("shrink-0 font-mono text-[10.5px]", ev.error ? "text-danger" : "text-muted")}>
                        {ev.name || ev.type}
                      </span>
                      <span className="min-w-0 truncate text-[12px] text-foreground/85">{ev.summary}</span>
                    </span>
                    <span className="font-mono text-[10.5px] tabular-nums text-muted/60">
                      {ev.elapsedMs > 0 ? `${ev.elapsedMs}ms` : ""}
                    </span>
                  </button>
                  {expanded ? (
                    <EventDetail
                      ev={ev}
                      blob={ev.spillId ? blobs[ev.spillId] : undefined}
                      copied={copied === key}
                      onCopy={async () => {
                        const text = ev.detail || ev.summary;
                        if (await writeClipboard(text)) markCopied(key);
                      }}
                      onLoadSpill={ev.spillId ? () => loadSpill(ev.spillId) : undefined}
                    />
                  ) : null}
                </li>
              );
            })}
          </ol>
        )}
      </section>
    </div>
  );
}

function Empty({ title, icon }: { title: string; icon?: ReactNode }) {
  return (
    <div className="flex min-h-[8rem] flex-col items-center justify-center px-6 text-center">
      {icon ? <span className="mb-3 grid size-8 place-items-center rounded-lg bg-lift/60 text-muted" aria-hidden>{icon}</span> : null}
      <p className="text-[12px] text-muted">{title}</p>
    </div>
  );
}

function Dot() {
  return <span className="text-muted/40" aria-hidden>·</span>;
}

function Stat(props: { value: string; label?: string; tone?: string }) {
  return (
    <div className="flex items-baseline gap-1">
      <dd className={cn("font-medium text-foreground/85", props.tone)}>{props.value}</dd>
      {props.label ? <dt className="text-muted/80">{props.label.toLowerCase()}</dt> : null}
    </div>
  );
}

function CopyButton({ copied, onClick }: { copied: boolean; onClick: () => void }) {
  const copy = useCopy();
  return (
    <button
      type="button"
      className="inline-flex h-6 items-center gap-1 rounded-md px-1.5 text-[11px] text-muted transition-colors hover:bg-lift hover:text-foreground"
      onClick={onClick}
    >
      {copied ? <Check className="size-3" aria-hidden /> : <Copy className="size-3" aria-hidden />}
      {copied ? copy.trace.copied : copy.trace.copy}
    </button>
  );
}

function Surface({ children, tone }: { children: ReactNode; tone?: "fg" }) {
  return (
    <pre
      className={cn(
        "max-h-56 overflow-auto whitespace-pre-wrap break-all rounded-lg border border-border/60 bg-panel/70 px-3 py-2 font-mono text-[11px] leading-[1.55]",
        tone === "fg" ? "text-foreground/90" : "text-muted",
      )}
    >
      {children}
    </pre>
  );
}

function ArtifactRow(props: {
  art: TraceArtifact;
  blob?: SpillBlob | "loading" | "error";
  open: boolean;
  onOpen: () => void;
  onCopy: (text: string) => void;
  copied: boolean;
}) {
  const copy = useCopy();
  const art = props.art;
  return (
    <li>
      <button
        type="button"
        aria-expanded={props.open}
        className={cn(
          "grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-baseline gap-x-2.5 px-3 py-1.5 text-left text-[11.5px] transition-colors hover:bg-lift/40",
          props.open && "bg-lift/30",
        )}
        onClick={props.onOpen}
      >
        <span className="font-mono text-[10.5px] text-muted">{art.kind}</span>
        <span className="min-w-0 truncate font-mono text-foreground/85">{art.label}</span>
        <span className="font-mono text-[10.5px] tabular-nums text-muted/60">{art.bytes > 0 ? formatBytes(art.bytes) : ""}</span>
      </button>
      {props.open ? (
        <div className="px-3 pb-2.5 pt-0.5">
          {props.blob === "loading" ? <p className="text-[11px] text-muted">…</p> : null}
          {props.blob === "error" ? <p className="text-[11px] text-danger">—</p> : null}
          {props.blob && typeof props.blob === "object" ? (
            <>
              <Surface>{props.blob.text}</Surface>
              <div className="mt-1.5 flex items-center gap-2">
                <CopyButton copied={props.copied} onClick={() => props.onCopy(props.blob && typeof props.blob === "object" ? props.blob.text : art.preview)} />
                {props.blob.truncated ? <span className="text-[10.5px] text-muted">{copy.trace.truncated}</span> : null}
              </div>
            </>
          ) : art.preview ? (
            <Surface>{art.preview}</Surface>
          ) : art.kind === "spill" ? (
            <p className="text-[11px] text-muted">{copy.trace.loadSpill}</p>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}

function EventDetail(props: {
  ev: TraceEvent;
  blob?: SpillBlob | "loading" | "error";
  copied: boolean;
  onCopy: () => void;
  onLoadSpill?: () => void;
}) {
  const copy = useCopy();
  const ev = props.ev;
  return (
    <div className="px-3 pb-2.5 pt-0.5">
      {ev.detail ? <Surface>{ev.detail}</Surface> : null}
      <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
        <CopyButton copied={props.copied} onClick={props.onCopy} />
        {ev.spillId && props.onLoadSpill ? (
          <button
            type="button"
            className="h-6 rounded-md px-1.5 text-[11px] text-muted transition-colors hover:bg-lift hover:text-foreground"
            onClick={props.onLoadSpill}
          >
            {copy.trace.loadSpill}
          </button>
        ) : null}
        <span className="ml-auto flex items-center gap-2 font-mono text-[10.5px] tabular-nums text-muted/60">
          {ev.bytes > 0 ? <span>{formatBytes(ev.bytes)}</span> : null}
          {ev.spillId ? <span>{ev.spillId}</span> : null}
        </span>
      </div>
      {props.blob === "loading" ? <p className="mt-1 text-[11px] text-muted">…</p> : null}
      {props.blob && typeof props.blob === "object" ? (
        <div className="mt-2">
          <Surface tone="fg">{props.blob.text}</Surface>
          {props.blob.truncated ? <p className="mt-1 text-[10.5px] text-muted">{copy.trace.truncated}</p> : null}
        </div>
      ) : null}
    </div>
  );
}

function eventKey(ev: TraceEvent) {
  return `${ev.index}:${ev.type}:${ev.id}`;
}

function formatTime(ts: string) {
  if (!ts || ts.startsWith("0001-")) return "";
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
}

function formatDuration(ms: number) {
  if (ms <= 0) return "0ms";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(ms < 10_000 ? 1 : 0)}s`;
  const m = Math.floor(ms / 60_000);
  const s = Math.round((ms % 60_000) / 1000);
  return `${m}m ${s}s`;
}

function formatBytes(n: number) {
  if (!n) return "0 B";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(n < 10 * 1024 ? 1 : 0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}
