import { Fragment, useEffect, useState, type CSSProperties } from "react";
import { DiffBlock } from "../../lib/split-diff";
import { langFromPath, type ArtifactView } from "../../lib/artifact-preview";
import { looksLikeHTML, looksLikeHTMLFile, looksLikePDF, asPreviewDocument } from "../../lib/html-preview";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { codePlugin } from "../../lib/code-plugin";
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
    const sync = codePlugin.highlight({ code, language, themes }, (next) => {
      if (live && Array.isArray(next?.tokens)) setLines(next.tokens as HlToken[][]);
    });
    if (sync && live && Array.isArray(sync.tokens)) setLines(sync.tokens as HlToken[][]);
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
        "code-hl min-w-0 overflow-auto bg-sidebar/70 px-3 py-2 font-mono text-[11.5px] leading-[1.6] text-foreground/90",
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
  const [text, setText] = useState("");
  const [html, setHtml] = useState(false);
  const [lang, setLang] = useState("");
  const [binary, setBinary] = useState(false);
  const [err, setErr] = useState("");
  useEffect(() => {
    if (!workspace || !path) return;
    let cancel = false;
    void api
      .previewWorkspaceFile(workspace, path)
      .then((p) => {
        if (cancel) return;
        setText(p.text || "");
        setHtml(!!p.html);
        setLang(p.lang || langFromPath(path));
        setBinary(!!p.binary);
        setErr("");
      })
      .catch((e) => {
        if (!cancel) setErr(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancel = true;
    };
  }, [workspace, path]);
  const frame = fill ? "flex h-full min-h-0 flex-col" : "";
  const note = fill ? "grid h-full place-items-center px-4 text-center text-[12px] text-muted" : "mt-2 text-[12px] text-muted";
  if (!workspace || !path) return null;
  if (err) return <p className={note}>{err}</p>;
  if (binary) return <p className={note}>{copy.transcript.binaryFile}</p>;
  const asHtml = !source && (html || looksLikeHTML(text) || looksLikeHTMLFile(path));
  if (asHtml && text) {
    return (
      <div className={cn(frame, fill && "min-h-0")}>
        <SandboxedFrame html={asPreviewDocument(text)} title={path} fill={fill} />
      </div>
    );
  }
  if (text) {
    return (
      <div className={cn(frame, fill && "min-h-0")}>
        <CodePreview lang={lang || langFromPath(path)} text={text} fill={fill} />
      </div>
    );
  }
  return null;
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
