// 本文件拥有检查器里的浏览器面：地址、截图、工作区 HTML 和接管。不拥有浏览器怎么开。调用方传入 workspace 和 previewPath。

import { useEffect, useState } from "react";
import { ExternalLink, Globe } from "lucide-react";
import { Tooltip } from "../components/ui/tooltip";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import * as api from "../lib/client";
import type { BrowserView } from "../lib/client";
import { looksLikeHTMLFile } from "../lib/html-preview";
import { fileBase, isBrowserPreviewPath, joinedPath, previewTarget } from "../lib/file-paths";
import { joinWorkspace } from "../lib/workspace";
import { WorkspaceFileView } from "./transcript/FilePreview";

/** parseCsv 按逗号和引号切开。引号里的逗号和换行留在格子里。 */
function parseCsv(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let quoted = false;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (quoted) {
      if (ch === "\"" && text[i + 1] === "\"") {
        cell += "\"";
        i += 1;
      } else if (ch === "\"") quoted = false;
      else cell += ch;
      continue;
    }
    if (ch === "\"") quoted = true;
    else if (ch === ",") {
      row.push(cell);
      cell = "";
    } else if (ch === "\n") {
      row.push(cell);
      rows.push(row);
      row = [];
      cell = "";
    } else if (ch !== "\r") cell += ch;
  }
  if (cell || row.length) {
    row.push(cell);
    rows.push(row);
  }
  return rows.filter((line) => line.some((part) => part.trim()));
}

/** CsvSheet 把表格预览限制在前 120 行、16 列。 */
function CsvSheet(props: { workspace?: string; path: string }) {
  const copy = useCopy();
  const [rows, setRows] = useState<string[][]>([]);
  const [err, setErr] = useState("");
  useEffect(() => {
    if (!props.workspace || !props.path) return;
    let stop = false;
    setErr("");
    void api.previewWorkspaceFile(props.workspace, props.path).then((file) => {
      if (!stop) setRows(parseCsv(file.text || "").slice(0, 120).map((line) => line.slice(0, 16)));
    }).catch((e) => {
      if (!stop) setErr(e instanceof Error ? e.message : String(e));
    });
    return () => { stop = true; };
  }, [props.workspace, props.path]);
  if (err) return <p className="grid h-full place-items-center px-6 text-[13px] text-muted">{err}</p>;
  if (!rows.length) return <p className="grid h-full place-items-center text-[13px] text-muted">{copy.review.openingFile}</p>;
  const head = rows[0];
  const body = rows.slice(1);
  return (
    <div className="overlay-scroll h-full overflow-auto" data-testid="csv-sheet">
      <table className="w-full border-collapse text-left text-[12px]">
        <thead className="sticky top-0 bg-card">
          <tr>
            {head.map((cell, i) => (
              <th key={i} className="border-b border-border/70 px-3 py-2 font-medium text-muted">{cell}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {body.map((line, r) => (
            <tr key={r} className="border-b border-border/40">
              {head.map((_, c) => (
                <td key={c} className="max-w-[16rem] truncate px-3 py-1.5 font-mono text-foreground/90">{line[c] || ""}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/**
 * FilePreviewPane 占满自己的窗口。HTML 用本机页面，CSV 用表格，其余用文件预览。
 * 这个窗口和主窗口分开，关掉它不会挡住主窗口。
 */
export function FilePreviewPane(props: { workspace?: string; path: string }) {
  const copy = useCopy();
  const target = previewTarget(props.workspace, props.path);
  const name = fileBase(props.path);
  const html = isBrowserPreviewPath(props.path);
  const csv = /\.csv$/i.test(props.path);
  const [pageURL, setPageURL] = useState("");
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    document.title = name || "Yoyo";
  }, [name]);
  useEffect(() => {
    if (!html || !target.workspace || !target.path) return;
    let stop = false;
    setPageURL("");
    setFailed(false);
    void api.previewPageURL(target.workspace, target.path).then((url) => {
      if (stop) return;
      if (!url) setFailed(true);
      else setPageURL(url);
    }).catch(() => {
      if (!stop) setFailed(true);
    });
    return () => { stop = true; };
  }, [html, target.workspace, target.path]);
  const abs = joinedPath(props.workspace, props.path);
  return (
    <div className="flex h-full min-h-0 flex-col bg-background" data-testid="file-preview-window">
      <div className="flex min-h-12 shrink-0 flex-wrap items-center gap-2 border-b border-border/60 py-2 pl-4 pr-3">
        <h1 className="min-w-0 flex-1 truncate text-[14px] font-medium">{name}</h1>
        <button
          type="button"
          className="h-7 rounded-full border border-border/70 px-3 text-[12px] text-foreground hover:bg-lift"
          onClick={() => { if (abs) void api.revealPath(abs); }}
        >
          {copy.review.openFolder}
        </button>
        <button
          type="button"
          className="h-7 rounded-full border border-border/70 px-3 text-[12px] text-foreground hover:bg-lift"
          onClick={() => { if (abs) void api.openPath(abs); }}
        >
          {copy.review.openWithApp}
        </button>
      </div>
      <div className="min-h-0 flex-1 bg-background">
        {html && pageURL ? (
          <iframe title={name} src={pageURL} className="h-full w-full border-0 bg-white" data-testid="file-browser-frame" />
        ) : html && !failed ? (
          <div className="grid h-full place-items-center text-[12px] text-muted">{copy.review.openingFile}</div>
        ) : csv && target.workspace && target.path ? (
          <CsvSheet workspace={target.workspace} path={target.path} />
        ) : target.workspace && target.path ? (
          <WorkspaceFileView workspace={target.workspace} path={target.path} fill />
        ) : (
          <div className="grid h-full place-items-center text-[12px] text-muted">{copy.review.cannotPreview}</div>
        )}
      </div>
    </div>
  );
}

/** isExternalURL http、https 或 file 开头算外部地址。raw 为假时用空字符串，去掉首尾空白，传给 test。 */
function isExternalURL(raw: string): boolean {
  return /^(https?:|file:)/i.test((raw || "").trim());
}

/** useRevocableURL blob: 原样用。data: 转成对象 URL，卸下时撤销。其余地址直接用。转换失败就退回原地址。空地址清掉。src 看开头，为假时走另一支，传给 setUrl，传给 fetch。 */
function useRevocableURL(src?: string): string {
  const [url, setUrl] = useState("");
  useEffect(() => {
    if (!src) {
      setUrl("");
      return;
    }
    if (src.startsWith("blob:")) {
      setUrl(src);
      return;
    }
    let objectUrl = "";
    let cancelled = false;
    void (async () => {
      try {
        const blob = src.startsWith("data:") ? await (await fetch(src)).blob() : null;
        if (cancelled) return;
        if (!blob) {
          setUrl(src);
          return;
        }
        objectUrl = URL.createObjectURL(blob);
        setUrl(objectUrl);
      } catch {
        if (!cancelled) setUrl(src);
      }
    })();
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [src]);
  return url;
}

/** BrowserPane 既没有活动页也没有工作区 HTML 时显示空闲。工作区 HTML 是给操作者看的页面。残留的隔离页不能盖住代理打开的这一页：source 且远程仍在时才改看截图。props 读取 previewPath、running、workspace。 */
export function BrowserPane(props: {
  workspace?: string;
  previewPath?: string;
  running?: boolean;
}) {
  const copy = useCopy();
  const [view, setView] = useState<BrowserView | null>(null);
  const [source, setSource] = useState(false);
  const [busy, setBusy] = useState(false);
  const [opening, setOpening] = useState(false);
  const [err, setErr] = useState("");
  const htmlPath = looksLikeHTMLFile(props.previewPath || "")
    ? (props.previewPath || "")
    : looksLikeHTMLFile(view?.preview || "")
      ? (view?.preview || "")
      : "";

  // 进行中每 800 毫秒拉一次，否则每 1200 毫秒。已经有工作区 HTML 且没有隔离活动页时不再排下一次。
  useEffect(() => {
    let alive = true;
    let timer = 0;
    const pull = () => {
      void api.browserView().then((v) => {
        if (!alive) return;
        setView(v);
        setErr("");
        const isolatedLive = !!(v?.live || v?.screenshot);
        if (htmlPath && !isolatedLive) return;
        timer = window.setTimeout(pull, props.running ? 800 : 1200);
      }).catch((e) => {
        if (!alive) return;
        setErr(e instanceof Error ? e.message : String(e));
        timer = window.setTimeout(pull, props.running ? 800 : 1200);
      });
    };
    pull();
    return () => {
      alive = false;
      if (timer) window.clearTimeout(timer);
    };
  }, [props.running, htmlPath]);

  const shot = useRevocableURL(view?.screenshot);
  const remoteLive = isExternalURL(view?.url || "") && (!!view?.live || !!view?.screenshot);
  const live = remoteLive || !!view?.live || !!view?.screenshot || isExternalURL(view?.url || "");
  // 工作区 HTML 是给操作者看的浏览器面。上一轮 http 或 https 留下的隔离页不能盖住代理打开的页面。
  const showHtml = !!htmlPath && !(source && remoteLive);
  const lane = view?.headed ? copy.review.headedLane : view?.lane === "attached" ? copy.review.attachedLane : copy.review.isolatedLane;
  const address = (remoteLive ? view?.url : "") || htmlPath || view?.url || view?.title || copy.review.browser;
  const canOpenExternal = isExternalURL(view?.url || "") || !!htmlPath;
  const canTakeOver = live || !!htmlPath;

  // 有工作区 HTML 时把它和 workspace 拼成目标再接管，否则不传目标。
  const takeOver = async () => {
    setBusy(true);
    try {
      const target = htmlPath ? joinWorkspace(props.workspace || "", htmlPath) : "";
      await api.browserTakeover(target || undefined);
      setView(await api.browserView());
      setErr("");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  // 正在看工作区 HTML 时用 openPath。否则外部地址用 openURL。再不行仍按工作区路径打开。
  const openExternal = async () => {
    const liveURL = (view?.url || "").trim();
    setOpening(true);
    try {
      if (showHtml && htmlPath) {
        await api.openPath(joinWorkspace(props.workspace || "", htmlPath));
      } else if (isExternalURL(liveURL)) {
        await api.openURL(liveURL);
      } else if (htmlPath) {
        await api.openPath(joinWorkspace(props.workspace || "", htmlPath));
      }
      setErr("");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setOpening(false);
    }
  };

  if (!live && !htmlPath) {
    return (
      <div className="flex h-full min-h-[10rem] flex-col items-center justify-center px-6 text-center" data-testid="review-browser">
        <span className="mark-well mb-3 text-muted" aria-hidden><Globe className="size-4" /></span>
        <p className="text-[13px] font-medium text-foreground/85">{copy.review.browserIdle}</p>
        <p className="mt-1 max-w-[22rem] text-[12px] leading-[1.55] text-muted">{copy.review.browserIdleHint}</p>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="review-browser">
      <div className="flex h-9 shrink-0 items-center gap-1.5 border-b border-border/60 pl-2.5 pr-1.5 text-[12px] text-muted">
        {live ? <span className={cn("pulse-dot", view?.live && "is-accent")} aria-hidden /> : <Globe className="size-3.5 shrink-0" aria-hidden />}
        <span className="shrink-0 rounded-md bg-lift/80 px-1.5 py-0.5 text-[11px] uppercase tracking-[0.08em] text-muted">
          {live ? copy.review.liveLane : copy.review.page}
        </span>
        <span
          className="min-w-0 flex-1 truncate rounded-md bg-lift/70 px-2 py-0.5 font-mono text-[11px] text-foreground/85"
          title={address}
          data-testid="browser-url"
        >
          {address}
        </span>
        {live ? <span className="shrink-0 text-[11px] text-muted/80">{lane}</span> : null}
        {live && (view?.tabCount ?? 0) > 1 ? (
          <span className="shrink-0 text-[11px] text-muted/80">
            {view?.tabCount} {copy.review.tabsLane}
          </span>
        ) : null}
        {view?.profileName && view.profileName !== "default" ? (
          <span className="shrink-0 rounded-md bg-lift/80 px-1.5 py-0.5 text-[11px] text-muted" title={view.profile}>
            {view.profileName}
          </span>
        ) : null}
        {view?.recording ? (
          <span className="shrink-0 rounded-md bg-lift/80 px-1.5 py-0.5 text-[11px] text-foreground/85">
            {copy.review.recordingLane}
          </span>
        ) : null}
        {htmlPath && live ? (
          <button
            type="button"
            className="h-6 rounded-md px-2 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground"
            onClick={() => setSource((v) => !v)}
          >
            {source ? copy.review.liveLane : copy.transcript.preview}
          </button>
        ) : null}
        <Tooltip content={copy.review.openExternalHint}>
          <button
            type="button"
            className="grid size-6 shrink-0 place-items-center rounded-md text-muted transition-colors hover:bg-lift hover:text-foreground disabled:opacity-40"
            aria-label={copy.review.openExternal}
            data-testid="browser-open-external"
            disabled={!canOpenExternal || opening}
            onClick={() => { void openExternal(); }}
          >
            <ExternalLink className="size-3.5" aria-hidden />
          </button>
        </Tooltip>
        {canTakeOver ? (
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
      <div className="relative min-h-0 flex-1 overflow-hidden bg-[var(--media-surface,#0b0d10)]">
        {showHtml && htmlPath ? (
          <div className="h-full min-h-0 overflow-hidden">
            <WorkspaceFileView workspace={props.workspace} path={htmlPath} fill />
          </div>
        ) : shot && !source ? (
          <button
            type="button"
            className="block h-full w-full min-h-full cursor-pointer"
            aria-label={copy.review.takeOver}
            data-testid="browser-frame-hit"
            onClick={() => { void takeOver(); }}
          >
            <img
              src={shot}
              alt={view?.title || view?.url}
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
        <ul className="overlay-scroll max-h-28 shrink-0 overflow-auto border-t border-border/50">
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
