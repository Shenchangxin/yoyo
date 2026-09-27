import { Fragment, useEffect, useState, type CSSProperties } from "react";
import { DiffBlock } from "../../lib/split-diff";
import { langFromPath, type ArtifactView } from "../../lib/artifact-preview";
import {
  asPreviewDocument,
  blobFromBase64,
  looksLikeHTML,
  looksLikeHTMLFile,
  looksLikeMarkdown,
  looksLikeOffice,
  looksLikePDF,
  needsBlobPreview,
  previewKindFromPath,
} from "../../lib/html-preview";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { codePlugin } from "../../lib/code-plugin";
import { Markdown } from "../../lib/markdown";
import { useUI } from "../../lib/store";
import { PresenceModuleLoading } from "../presence";
import { SandboxedFrame } from "./SandboxedFrame";

type HlToken = {
  content: string;
  color?: string;
  bgColor?: string;
  htmlStyle?: string | Record<string, string>;
  htmlAttrs?: Record<string, string>;
};

function tokenStyle(t: HlToken): CSSProperties | undefined {
  if (t.htmlStyle && typeof t.htmlStyle === "object" && !Array.isArray(t.htmlStyle)) {
    return t.htmlStyle as CSSProperties;
  }
  const style: CSSProperties = {};
  if (typeof t.htmlStyle === "string") {
    for (const part of t.htmlStyle.split(";")) {
      const i = part.indexOf(":");
      if (i < 0) continue;
      const key = part.slice(0, i).trim();
      const val = part.slice(i + 1).trim();
      if (key) (style as Record<string, string>)[key] = val;
    }
  }
  if (t.color && !style.color) style.color = t.color;
  if (t.bgColor && !style.backgroundColor) style.backgroundColor = t.bgColor;
  return Object.keys(style).length ? style : undefined;
}

function useHighlight(code: string, lang?: string): HlToken[][] | null {
  const [lines, setLines] = useState<HlToken[][] | null>(null);
  useEffect(() => {
    let live = true;
    setLines(null);
    const language = (lang && codePlugin.supportsLanguage(lang as Parameters<typeof codePlugin.supportsLanguage>[0]) ? lang : "text") as Parameters<typeof codePlugin.highlight>[0]["language"];
    const themes = codePlugin.getThemes();
    try {
      const sync = codePlugin.highlight({ code, language, themes }, (next) => {
        if (live && Array.isArray(next?.tokens)) setLines(next.tokens as HlToken[][]);
      });
      if (sync && live && Array.isArray(sync.tokens)) setLines(sync.tokens as HlToken[][]);
    } catch {
      if (live) setLines(null);
    }
    return () => {
      live = false;
    };
  }, [code, lang]);
  return lines;
}

/** One code surface — DESIGN.md recipe. No nested Streamdown card. */
export function CodePreview({
  lang,
  text,
  className,
  cap,
  dim,
  fill,
}: {
  lang?: string;
  text: string;
  className?: string;
  cap?: number;
  dim?: boolean;
  fill?: boolean;
}) {
  const body = cap && text.length > cap ? text.slice(0, cap) : text;
  const tokens = useHighlight(body, lang);
  if (!body.trim()) return null;
  return (
    <pre
      data-testid="artifact-code"
      data-lang={lang || undefined}
      tabIndex={0}
      className={cn(
        "code-hl min-w-0 overflow-auto bg-sidebar/70 px-3 py-2 font-mono text-[12px] leading-[1.6] text-foreground/90",
        fill ? "mt-0 h-full min-h-0 flex-1 rounded-none border-0" : "mt-2 max-h-[28rem] rounded-lg border border-border/60",
        dim && "opacity-70",
        className,
      )}
    >
      <code className="block min-h-full whitespace-pre">
        {tokens
          ? tokens.map((line, i) => (
              <Fragment key={i}>
                {line.map((t, j) => (
                  <span key={j} style={tokenStyle(t)} {...(t.htmlAttrs || {})}>
                    {t.content}
                  </span>
                ))}
                {i < tokens.length - 1 ? "\n" : null}
              </Fragment>
            ))
          : body}
      </code>
    </pre>
  );
}

function PreviewLoading({ fill, label }: { fill?: boolean; label: string }) {
  useEffect(() => {
    useUI.getState().setModuleLoading(true);
    return () => useUI.getState().setModuleLoading(false);
  }, []);
  return (
    <PresenceModuleLoading
      label={label}
      size={fill ? 160 : 96}
      className={cn("w-full", fill ? "h-full min-h-0" : "min-h-40")}
    />
  );
}

export function WorkspaceFileView({
  workspace,
  path,
  source,
  fill,
}: {
  workspace?: string;
  path: string;
  source?: boolean;
  fill?: boolean;
}) {
  const copy = useCopy();
  const blobKind = !source && needsBlobPreview(path);
  const [text, setText] = useState("");
  const [html, setHtml] = useState(false);
  const [lang, setLang] = useState("");
  const [kind, setKind] = useState("");
  const [binary, setBinary] = useState(false);
  const [truncated, setTruncated] = useState(false);
  const [objectUrl, setObjectUrl] = useState("");
  const [mime, setMime] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);
  const [mediaReady, setMediaReady] = useState(false);
  useEffect(() => {
    if (!workspace || !path) return;
    let cancel = false;
    let url = "";
    setErr("");
    setText("");
    setHtml(false);
    setLang(langFromPath(path));
    setKind(previewKindFromPath(path));
    setBinary(blobKind);
    setTruncated(false);
    setObjectUrl("");
    setMime("");
    setLoading(true);
    setMediaReady(false);
    void (async () => {
      try {
        if (blobKind) {
          const blob = await api.readWorkspaceBlob(workspace, path);
          if (cancel) return;
          setTruncated(!!blob.truncated);
          setMime(blob.mime || "");
          setKind(blob.kind || previewKindFromPath(path, blob.mime));
          const file = await blobFromBase64(blob.base64, blob.mime || "application/octet-stream");
          if (cancel) return;
          if (/\.docx$/i.test(path)) {
            try {
              const mammoth = await import("mammoth");
              const converted = await mammoth.convertToHtml({ arrayBuffer: await file.arrayBuffer() });
              if (cancel) return;
              setText(converted.value || "");
              setHtml(true);
              setKind("office");
              setLoading(false);
              setMediaReady(true);
              return;
            } catch {
              if (!cancel) {
                setLoading(false);
                setMediaReady(true);
              }
              return;
            }
          }
          url = URL.createObjectURL(file);
          if (cancel) {
            URL.revokeObjectURL(url);
            url = "";
            return;
          }
          setObjectUrl(url);
          setLoading(false);
          return;
        }
        const p = await api.previewWorkspaceFile(workspace, path);
        if (cancel) return;
        setText(p.text || "");
        setHtml(!!p.html);
        setLang(p.lang || langFromPath(path));
        setKind(p.kind || previewKindFromPath(path, p.mime));
        setBinary(!!p.binary);
        setTruncated(!!p.truncated);
        setMime(p.mime || "");
        setLoading(false);
        setMediaReady(true);
      } catch (e) {
        if (!cancel) {
          setErr(e instanceof Error ? e.message : String(e));
          setLoading(false);
          setMediaReady(true);
        }
      }
    })();
    return () => {
      cancel = true;
      if (url) URL.revokeObjectURL(url);
    };
  }, [workspace, path, blobKind]);

  useEffect(() => {
    if (!objectUrl || mediaReady) return;
    const t = window.setTimeout(() => setMediaReady(true), 2_000);
    return () => window.clearTimeout(t);
  }, [objectUrl, mediaReady]);

  const frame = fill ? "flex h-full min-h-0 flex-col" : "";
  const note = fill ? "grid h-full place-items-center px-4 text-center text-[12px] text-muted" : "mt-2 text-[12px] text-muted";
  if (!workspace || !path) return null;
  if (err) return <p className={note}>{err}</p>;
  const resolved = kind || previewKindFromPath(path, mime);
  const cap = truncated ? (
    <p className={cn("shrink-0 px-3 py-1.5 text-[11px] text-muted", fill ? "border-t border-border/50" : "mt-1")}>
      {copy.review.previewTruncated}
    </p>
  ) : null;
  const waiting = blobKind && !mediaReady;
  const reveal = waiting ? "invisible" : "";
  const markReady = () => setMediaReady(true);

  if (blobKind) {
    return (
      <div className={cn(frame, "relative", fill && "min-h-0")} data-kind={resolved || "binary"}>
        {waiting ? (
          <div className={cn("z-[1] flex items-center justify-center bg-sidebar/70", objectUrl || text ? "absolute inset-0" : fill ? "h-full min-h-0 flex-1" : "")}>
            <PreviewLoading fill={fill} label={copy.review.openingFile} />
          </div>
        ) : null}
        {objectUrl && (resolved === "pdf" || looksLikePDF(path, mime)) ? (
          <iframe
            title={path}
            src={objectUrl}
            onLoad={markReady}
            onError={markReady}
            className={cn("w-full min-w-0 bg-media-surface", fill ? "h-full min-h-0 flex-1 border-0" : "mt-2 h-[28rem] rounded-lg border border-border/70", reveal)}
          />
        ) : null}
        {objectUrl && resolved === "image" ? (
          <div className={cn("grid place-items-center overflow-auto", fill ? "h-full min-h-0 flex-1 bg-media-surface p-3" : "mt-2 max-h-[28rem] rounded-lg border border-border/70 bg-media-surface p-3", reveal)}>
            <img src={objectUrl} alt={path} onLoad={markReady} onError={markReady} className="max-h-full max-w-full object-contain" />
          </div>
        ) : null}
        {objectUrl && resolved === "video" ? (
          <video controls src={objectUrl} onLoadedData={markReady} onError={markReady} className={cn("w-full bg-media-surface", fill ? "h-full min-h-0 flex-1" : "mt-2 max-h-[28rem] rounded-lg", reveal)} />
        ) : null}
        {objectUrl && resolved === "audio" ? (
          <audio controls src={objectUrl} onLoadedMetadata={markReady} onError={markReady} className={cn("w-full max-w-lg", fill ? "px-4" : "mt-2", reveal)} />
        ) : null}
        {!waiting && html && text ? <SandboxedFrame html={asPreviewDocument(text)} title={path} fill={fill} /> : null}
        {!waiting && !objectUrl && !text ? <p className={note}>{copy.review.cannotPreview}</p> : null}
        {cap}
      </div>
    );
  }

  if (loading) {
    return fill ? <div className={cn(frame, "min-h-0")} /> : null;
  }
  if (binary && !text && !objectUrl) return <p className={note}>{copy.transcript.binaryFile}</p>;
  const asHtml = !source && (html || looksLikeHTML(text) || looksLikeHTMLFile(path));
  if (asHtml && text) {
    return (
      <div className={cn(frame, fill && "min-h-0")}>
        <SandboxedFrame html={asPreviewDocument(text)} title={path} fill={fill} />
        {cap}
      </div>
    );
  }
  if (!source && (resolved === "markdown" || looksLikeMarkdown(path)) && text) {
    return (
      <div className={cn(frame, fill && "min-h-0")}>
        <div className={cn("min-h-0 overflow-auto px-4 py-3", fill ? "h-full flex-1" : "mt-2 max-h-[28rem] rounded-lg border border-border/60")}>
          <Markdown text={text} quiet />
        </div>
        {cap}
      </div>
    );
  }
  if (!source && (resolved === "office" || looksLikeOffice(path)) && text && !/\.pdf$/i.test(path)) {
    return (
      <div className={cn(frame, fill && "min-h-0")}>
        <OfficeText text={text} fill={fill} />
        {cap}
      </div>
    );
  }
  if (text) {
    return (
      <div className={cn(frame, fill && "min-h-0")}>
        <CodePreview lang={lang || langFromPath(path)} text={text} fill={fill} />
        {cap}
      </div>
    );
  }
  return <p className={note}>{copy.review.cannotPreview}</p>;
}

function OfficeText({ text, fill }: { text: string; fill?: boolean }) {
  const rows = parseSheet(text);
  if (rows) {
    return (
      <div className={cn("min-h-0 overflow-auto", fill ? "h-full flex-1" : "mt-2 max-h-[28rem] rounded-lg border border-border/60")}>
        <table className="w-full border-collapse text-left text-[12px]">
          <tbody>
            {rows.map((row, i) => (
              <tr key={i} className="border-b border-border/40">
                {row.map((cell, j) => (
                  <td key={j} className="px-2.5 py-1.5 font-mono text-foreground/90">
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }
  return (
    <pre className={cn("overflow-auto whitespace-pre-wrap px-3 py-2 font-mono text-[12px] leading-[1.6]", fill ? "h-full flex-1" : "mt-2 max-h-[28rem] rounded-lg border border-border/60")}>
      {text}
    </pre>
  );
}

function parseSheet(text: string): string[][] | null {
  const lines = text.split("\n").map((l) => l.trim()).filter(Boolean);
  if (lines.length < 2) return null;
  const cells: { r: number; c: number; v: string }[] = [];
  for (const line of lines) {
    const m = line.match(/^([A-Z]+)(\d+)=(.*)$/);
    if (!m) return null;
    cells.push({ c: colIndex(m[1]), r: Number(m[2]), v: m[3] });
  }
  const maxR = Math.max(...cells.map((c) => c.r));
  const maxC = Math.max(...cells.map((c) => c.c));
  if (maxR > 80 || maxC > 26) return null;
  const grid = Array.from({ length: maxR }, () => Array.from({ length: maxC }, () => ""));
  for (const c of cells) {
    if (c.r > 0 && c.c > 0) grid[c.r - 1][c.c - 1] = c.v;
  }
  return grid;
}

function colIndex(col: string): number {
  let n = 0;
  for (const ch of col) n = n * 26 + (ch.charCodeAt(0) - 64);
  return n;
}

export function ArtifactBody({
  view,
  workspace,
  source,
}: {
  view: ArtifactView;
  workspace?: string;
  source?: boolean;
}) {
  const localHtml = view.html && looksLikeHTML(view.html) ? view.html : "";
  const wantHtml = looksLikeHTMLFile(view.path) || !!localHtml;
  return (
    <div className="min-w-0 overflow-hidden" data-testid="artifact-body">
      {view.diff ? (
        <div className="mt-2 min-w-0 overflow-hidden rounded-lg border border-border/70 bg-sidebar/50" data-testid="artifact-diff">
          <DiffBlock src={view.diff} mode="unified" className="max-h-[28rem]" />
        </div>
      ) : null}
      {wantHtml && !source && localHtml ? <SandboxedFrame html={localHtml} title={view.title || view.path} /> : null}
      {wantHtml && !source && !localHtml && workspace && view.path ? (
        <WorkspaceFileView workspace={workspace} path={view.path} />
      ) : null}
      {source || (!wantHtml && view.code) ? <CodePreview lang={view.lang || langFromPath(view.path)} text={view.code} /> : null}
      {view.kind === "text" && view.path && workspace && !view.diff && !view.code ? (
        <WorkspaceFileView workspace={workspace} path={view.path} source={source} />
      ) : null}
      {view.path && workspace && !view.diff && !view.code && !wantHtml && view.kind !== "text" && (looksLikePDF(view.path) || /\.(docx|xlsx|pptx)$/i.test(view.path)) ? (
        <WorkspaceFileView workspace={workspace} path={view.path} source={source} />
      ) : null}
    </div>
  );
}
