import { useEffect, useState } from "react";
import { Globe } from "lucide-react";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import * as api from "../lib/client";
import type { BrowserView } from "../lib/client";
import { looksLikeHTMLFile } from "../lib/html-preview";
import { WorkspaceFileView } from "./transcript/FilePreview";

export function BrowserPane(props: {
  workspace?: string;
  previewPath?: string;
  running?: boolean;
}) {
  const copy = useCopy();
  const htmlPath = looksLikeHTMLFile(props.previewPath || "") ? props.previewPath : "";
  const [view, setView] = useState<BrowserView | null>(null);
  const [source, setSource] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    let alive = true;
    const pull = () => {
      void api.browserView().then((v) => {
        if (alive) {
          setView(v);
          setErr("");
        }
      }).catch((e) => {
        if (alive) setErr(e instanceof Error ? e.message : String(e));
      });
    };
    pull();
    const t = window.setInterval(pull, props.running ? 800 : 1200);
    return () => {
      alive = false;
      window.clearInterval(t);
    };
  }, [props.running]);

  const live = !!view?.live || !!view?.screenshot || !!view?.url;
  const showHtml = !!htmlPath && (!live || source);
  const lane = view?.headed ? copy.review.headedLane : view?.lane === "attached" ? copy.review.attachedLane : copy.review.isolatedLane;
  const address = view?.url || htmlPath || view?.title || copy.review.browser;
  const takeOver = async () => {
    setBusy(true);
    try {
      await api.browserTakeover();
      setView(await api.browserView());
      setErr("");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  if (!live && !htmlPath) {
    return (
      <div className="flex h-full min-h-[10rem] flex-col items-center justify-center px-6 text-center" data-testid="review-browser">
        <span className="mark-well mb-3 text-muted" aria-hidden><Globe className="size-4" /></span>
        <p className="text-[12.5px] font-medium text-foreground/85">{copy.review.browserIdle}</p>
        <p className="mt-1 max-w-[22rem] text-[11.5px] leading-[1.55] text-muted">{copy.review.browserIdleHint}</p>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="review-browser">
      <div className="flex h-9 shrink-0 items-center gap-1.5 border-b border-border/60 pl-2.5 pr-1.5 text-[11.5px] text-muted">
        {live ? <span className={cn("pulse-dot", view?.live && "is-accent")} aria-hidden /> : <Globe className="size-3.5 shrink-0" aria-hidden />}
        <span className="shrink-0 rounded-md bg-lift/80 px-1.5 py-0.5 text-[10.5px] uppercase tracking-[0.08em] text-muted">
          {live ? copy.review.liveLane : copy.review.page}
        </span>
        <span
          className="min-w-0 flex-1 truncate rounded-md bg-lift/70 px-2 py-0.5 font-mono text-[11px] text-foreground/85"
          title={address}
          data-testid="browser-url"
        >
          {address}
        </span>
        {live ? <span className="shrink-0 text-[10.5px] text-muted/80">{lane}</span> : null}
        {htmlPath && live ? (
          <button
            type="button"
            className="h-6 rounded-md px-2 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground"
            onClick={() => setSource((v) => !v)}
          >
            {source ? copy.review.liveLane : copy.transcript.preview}
          </button>
        ) : null}
        {live ? (
          <button
            type="button"
            className="h-6 rounded-md bg-foreground px-2 text-[11px] font-medium text-background hover:opacity-90 disabled:opacity-40"
            data-testid="browser-takeover"
            disabled={busy}
            onClick={() => { void takeOver(); }}
          >
            {busy ? copy.review.takingOver : copy.review.takeOver}
          </button>
        ) : null}
      </div>
      <div className="relative min-h-0 flex-1 bg-[var(--media-surface,#0b0d10)]">
        {showHtml && htmlPath ? (
          <WorkspaceFileView workspace={props.workspace} path={htmlPath} fill />
        ) : view?.screenshot && !source ? (
          <button
            type="button"
            className="block h-full w-full cursor-pointer"
            aria-label={copy.review.takeOver}
            data-testid="browser-frame-hit"
            onClick={() => { void takeOver(); }}
          >
            <img
              src={view.screenshot}
              alt={view.title || view.url}
              className="h-full w-full object-contain object-top"
              data-testid="browser-frame"
            />
          </button>
        ) : (
          <div className="grid h-full place-items-center px-6 text-center">
            <p className="text-[12px] text-muted">{view?.text || err || copy.review.waitingFrame}</p>
          </div>
        )}
        {err ? <p className="absolute bottom-2 left-3 right-3 text-[11px] text-danger">{err}</p> : null}
      </div>
      {view?.log?.length ? (
        <ul className="max-h-28 shrink-0 overflow-auto border-t border-border/50">
          {view.log.slice(-8).reverse().map((row, i) => (
            <li key={`${row.ts}-${i}`} className="flex h-7 items-center gap-2 border-b border-border/40 px-3 font-mono text-[11px] last:border-b-0">
              <span className="shrink-0 text-foreground/80">{row.op}</span>
              <span className={cn("min-w-0 truncate text-muted")}>{row.detail || row.url}</span>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
