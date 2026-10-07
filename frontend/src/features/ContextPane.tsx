import { useEffect, useState } from "react";
import { Pin, RefreshCw, Quote } from "lucide-react";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import * as api from "../lib/client";
import type { ContextInventory, ContextObject } from "../lib/protocol";

export function ContextPane(props: {
  sessionId?: string;
  running?: boolean;
  onQuote?: (text: string) => void;
}) {
  const copy = useCopy();
  const [inv, setInv] = useState<ContextInventory | null>(null);
  const [busy, setBusy] = useState(false);

  async function load() {
    if (!props.sessionId) {
      setInv(null);
      return;
    }
    setBusy(true);
    try {
      setInv(await api.contextInventory(props.sessionId));
    } catch {
      setInv(null);
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    void load();
  }, [props.sessionId, props.running]);

  async function togglePin(obj: ContextObject) {
    if (!props.sessionId) return;
    const kind = obj.kind === "hot" ? "file" : obj.kind;
    const key = obj.path || obj.title;
    if (kind !== "skill" && kind !== "file") return;
    try {
      await api.contextPin(props.sessionId, kind, key, !obj.pinned);
      await load();
    } catch {
      /* keep current */
    }
  }

  if (!props.sessionId) {
    return <Empty title={copy.review.contextEmpty} hint={copy.review.contextEmptyHint} />;
  }
  const objects = inv?.objects || [];
  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="context-pane">
      <div className="flex h-9 shrink-0 items-center justify-between border-b border-border/50 px-2.5">
        <span className="text-[11px] font-medium uppercase tracking-[0.06em] text-muted/70">{copy.review.contextTab}</span>
        <button
          type="button"
          className="inline-flex h-6 items-center gap-1 rounded-md px-1.5 text-[11px] text-muted hover:bg-lift hover:text-foreground"
          onClick={() => { void load(); }}
          disabled={busy}
          aria-label={copy.review.refreshContext}
        >
          <RefreshCw className={cn("size-3", busy && "animate-spin")} aria-hidden />
          {copy.review.refresh}
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-2 py-2">
        {!objects.length ? (
          <Empty title={copy.review.contextEmpty} hint={copy.review.contextEmptyHint} />
        ) : (
          <ul className="flex flex-col gap-0.5">
            {objects.map((obj) => (
              <li
                key={obj.id}
                className="group flex min-w-0 items-start gap-2 rounded-lg px-2 py-1.5 hover:bg-lift/60"
              >
                <span className="mt-0.5 w-12 shrink-0 text-[10px] uppercase tracking-[0.04em] text-muted/65">{obj.kind}</span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[13px] text-foreground/90">{obj.title}</span>
                  {obj.detail ? <span className="mt-0.5 block truncate text-[11px] text-muted">{obj.detail}</span> : null}
                  {obj.stale ? <span className="mt-0.5 block text-[10px] text-danger/80">{copy.review.stale}</span> : null}
                </span>
                <span className="flex shrink-0 items-center gap-0.5 opacity-0 group-hover:opacity-100">
                  {obj.path || obj.title ? (
                    <button
                      type="button"
                      className="grid size-6 place-items-center rounded-md text-muted hover:bg-background hover:text-foreground"
                      aria-label={copy.review.quoteContext}
                      onClick={() => {
                        const token = obj.path ? `@file:${obj.path}` : obj.title;
                        props.onQuote?.(token);
                      }}
                    >
                      <Quote className="size-3" aria-hidden />
                    </button>
                  ) : null}
                  {obj.kind === "skill" || obj.kind === "file" || obj.kind === "hot" ? (
                    <button
                      type="button"
                      className={cn(
                        "grid size-6 place-items-center rounded-md hover:bg-background",
                        obj.pinned ? "text-foreground" : "text-muted hover:text-foreground",
                      )}
                      aria-label={obj.pinned ? copy.review.unpin : copy.review.pin}
                      onClick={() => { void togglePin(obj); }}
                    >
                      <Pin className="size-3" aria-hidden />
                    </button>
                  ) : null}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function Empty({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="flex h-full min-h-[10rem] flex-col items-center justify-center px-6 text-center">
      <p className="text-[13px] font-medium text-foreground/85">{title}</p>
      <p className="mt-1 max-w-[22rem] text-[12px] leading-[1.55] text-muted">{hint}</p>
    </div>
  );
}
