import { useEffect, useMemo, useState } from "react";
import { useCopy } from "../lib/i18n";
import type { RunStatus, Thread } from "../lib/protocol";

export function RunningHub(props: {
  threads: Thread[];
  status?: RunStatus[];
  onSelect: (t: Thread) => void;
}) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") close(); };
    const tmr = window.setTimeout(() => {
      window.addEventListener("mousedown", close);
      window.addEventListener("keydown", onKey);
    }, 0);
    return () => {
      window.clearTimeout(tmr);
      window.removeEventListener("mousedown", close);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);
  useEffect(() => {
    if (!props.threads.length) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [props.threads.length]);
  const byId = useMemo(() => {
    const m = new Map<string, RunStatus>();
    for (const s of props.status || []) m.set(s.id, s);
    return m;
  }, [props.status]);
  if (!props.threads.length) return null;
  return (
    <div className="relative hidden sm:block">
      <button
        type="button"
        className="inline-flex items-center gap-1.5 rounded-full bg-lift px-2 py-0.5 text-[11px] tabular-nums text-foreground transition-[background-color,transform] duration-[var(--duration-fast)] ease-[var(--ease-out)] hover:bg-lift/80 active:scale-[0.98]"
        aria-expanded={open}
        aria-haspopup="listbox"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="pulse-dot" aria-hidden />
        {props.threads.length} {copy.titlebar.live}
      </button>
      {open ? (
        <div
          role="listbox"
          className="absolute right-0 z-30 mt-1 min-w-[240px] overflow-hidden rounded-xl border border-border bg-popover py-1 shadow-[var(--shadow-popover)]"
          onMouseDown={(e) => e.stopPropagation()}
        >
          {props.threads.map((t) => {
            const st = byId.get(t.id);
            const elapsed = formatElapsed(st?.startedAt, now);
            return (
              <button
                type="button"
                key={t.id}
                role="option"
                className="u-row-hover block w-full px-3 py-1.5 text-left"
                onClick={() => {
                  props.onSelect(t);
                  setOpen(false);
                }}
              >
                <span className="block truncate text-[12px]">{t.title || t.id.slice(0, 8)}</span>
                <span className="mt-0.5 flex items-center gap-2 text-[11px] tabular-nums text-muted">
                  {elapsed ? <span>{elapsed}</span> : null}
                  {st?.lastTool ? <span className="truncate">{st.lastTool}</span> : <span>{copy.titlebar.waiting}</span>}
                </span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}

function formatElapsed(iso: string | undefined, now: number): string {
  if (!iso) return "";
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return "";
  const sec = Math.max(0, Math.floor((now - t) / 1000));
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return m > 0 ? `${m}m ${s}s` : `${s}s`;
}
