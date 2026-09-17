import { useMemo, useState } from "react";
import { Button } from "../components/ui/button";
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
    return <p className="text-xs text-muted">{copy.trace.noSession}</p>;
  }

  const stats = t?.stats;
  const filters: { id: Filter; label: string }[] = [
    { id: "all", label: copy.trace.filterAll },
    { id: "exec", label: copy.trace.filterExec },
    { id: "dialog", label: copy.trace.filterDialog },
    { id: "memory", label: copy.trace.filterMemory },
    { id: "error", label: copy.trace.filterError },
  ];

  return (
    <div className="flex min-h-0 flex-col gap-3" data-testid="trace-panel">
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="lift" size="sm" onClick={props.onRefresh}>{copy.trace.refresh}</Button>
        {props.running ? <span className="text-[11px] text-muted">{copy.rail.running}</span> : null}
      </div>
      <dl className="grid grid-cols-2 gap-x-3 gap-y-1 text-[11px] tabular-nums sm:grid-cols-3" data-testid="trace-stats">
        <Stat label={copy.trace.events} value={String(stats?.events || 0)} />
        <Stat label={copy.trace.tools} value={String(stats?.toolCalls || 0)} />
        <Stat label={copy.trace.errors} value={String(stats?.errors || 0)} />
        <Stat label={copy.trace.duration} value={formatDuration(stats?.durationMs || 0)} />
        <Stat label={copy.trace.tokens} value={formatTokens(stats?.tokens || 0)} />
        <Stat label={copy.trace.spilled} value={formatBytes(stats?.spillBytes || 0)} />
      </dl>
      {(t?.artifacts || []).length ? (
        <section>
          <h3 className="mb-1.5 text-[11px] font-medium text-muted">{copy.trace.artifacts}</h3>
          <ul className="flex flex-col gap-1">
            {t!.artifacts.map((art) => (
              <ArtifactRow
                key={`${art.kind}:${art.id}`}
                art={art}
                blob={art.kind === "spill" ? blobs[art.id] : art.preview ? { id: art.id, bytes: art.bytes, text: art.preview, truncated: false } : undefined}
                onOpen={() => {
                  if (art.kind === "spill") void loadSpill(art.id);
                  setOpen(`art:${art.kind}:${art.id}`);
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
      <section>
        <h3 className="mb-1.5 text-[11px] font-medium text-muted">{copy.trace.timeline}</h3>
        <div className="mb-2 flex flex-wrap gap-1" role="tablist" aria-label={copy.trace.timeline}>
          {filters.map((f) => (
            <button
              type="button"
              key={f.id}
              role="tab"
              aria-selected={filter === f.id}
              className={cn(
                "rounded-md px-2 py-1 text-[11px]",
                filter === f.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
              )}
              onClick={() => setFilter(f.id)}
            >
              {f.label}
            </button>
          ))}
        </div>
        {!t || (t.events.length === 0 && filter === "all") ? (
          <p className="text-xs text-muted">{copy.trace.empty}</p>
        ) : events.length === 0 ? (
          <p className="text-xs text-muted">{copy.trace.noMatch}</p>
        ) : (
          <ol className="space-y-1">
            {events.map((ev) => {
              const key = eventKey(ev);
              const expanded = open === key;
              return (
                <li key={key} data-testid="trace-event" data-type={ev.type}>
                  <button
                    type="button"
                    className={cn(
                      "w-full rounded-lg border border-border bg-card px-2 py-1.5 text-left hover:bg-lift/40",
                      ev.error && "border-danger/40",
                    )}
                    onClick={() => setOpen(expanded ? null : key)}
                  >
                    <div className="flex items-baseline gap-2">
                      <span className="shrink-0 font-mono text-[10px] tabular-nums text-muted">{formatTime(ev.ts)}</span>
                      <span className={cn("shrink-0 text-[10px] font-medium uppercase tracking-wide", ev.error ? "text-danger" : "text-muted")}>
                        {ev.name || ev.type}
                      </span>
                      <span className="min-w-0 flex-1 truncate text-[12px] text-foreground">{ev.summary}</span>
                      {ev.elapsedMs > 0 ? <span className="shrink-0 font-mono text-[10px] text-muted">{ev.elapsedMs}ms</span> : null}
                    </div>
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

function Stat(props: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <dt className="text-muted">{props.label}</dt>
      <dd className="font-mono text-foreground">{props.value}</dd>
    </div>
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
    <li className="rounded-lg border border-border bg-card">
      <button type="button" className="flex w-full items-baseline gap-2 px-2 py-1.5 text-left text-[11px]" onClick={props.onOpen}>
        <span className="shrink-0 font-medium uppercase tracking-wide text-muted">{art.kind}</span>
        <span className="min-w-0 flex-1 truncate font-mono text-foreground">{art.label}</span>
        {art.bytes > 0 ? <span className="shrink-0 tabular-nums text-muted">{formatBytes(art.bytes)}</span> : null}
      </button>
      {props.open ? (
        <div className="border-t border-border px-2 py-2">
          {props.blob === "loading" ? <p className="text-[11px] text-muted">…</p> : null}
          {props.blob === "error" ? <p className="text-[11px] text-danger">—</p> : null}
          {props.blob && typeof props.blob === "object" ? (
            <>
              <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-muted">{props.blob.text}</pre>
              {props.blob.truncated ? <p className="mt-1 text-[10px] text-muted">{copy.trace.truncated}</p> : null}
              <Button variant="ghost" size="sm" className="mt-1 h-7 px-2" onClick={() => props.onCopy(props.blob && typeof props.blob === "object" ? props.blob.text : art.preview)}>
                {props.copied ? copy.trace.copied : copy.trace.copy}
              </Button>
            </>
          ) : art.preview ? (
            <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-muted">{art.preview}</pre>
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
    <div className="mt-1 rounded-lg border border-border/80 bg-panel/40 px-2 py-2">
      {ev.detail ? (
        <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-muted">{ev.detail}</pre>
      ) : null}
      <div className="mt-1.5 flex flex-wrap items-center gap-2">
        <Button variant="ghost" size="sm" className="h-7 px-2" onClick={props.onCopy}>
          {props.copied ? copy.trace.copied : copy.trace.copy}
        </Button>
        {ev.spillId && props.onLoadSpill ? (
          <Button variant="lift" size="sm" className="h-7 px-2" onClick={props.onLoadSpill}>
            {copy.trace.loadSpill}
          </Button>
        ) : null}
        {ev.bytes > 0 ? <span className="text-[10px] tabular-nums text-muted">{formatBytes(ev.bytes)}</span> : null}
        {ev.spillId ? <span className="font-mono text-[10px] text-muted">{ev.spillId}</span> : null}
      </div>
      {props.blob === "loading" ? <p className="mt-1 text-[11px] text-muted">…</p> : null}
      {props.blob && typeof props.blob === "object" ? (
        <>
          <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-foreground/90">{props.blob.text}</pre>
          {props.blob.truncated ? <p className="mt-1 text-[10px] text-muted">{copy.trace.truncated}</p> : null}
        </>
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
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
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
