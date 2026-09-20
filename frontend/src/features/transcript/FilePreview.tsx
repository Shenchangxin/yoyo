import { useEffect, useState } from "react";
import { DiffBlock } from "../../lib/split-diff";
import { langFromPath, type ArtifactView } from "../../lib/artifact-preview";
import { looksLikeHTML, looksLikeHTMLFile } from "../../lib/html-preview";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { SandboxedFrame } from "./SandboxedFrame";

/** One code surface — DESIGN.md recipe. No nested Streamdown card. */
export function CodePreview({
  lang,
  text,
  className,
  cap,
  dim,
}: {
  lang?: string;
  text: string;
  className?: string;
  cap?: number;
  dim?: boolean;
}) {
  const body = cap && text.length > cap ? text.slice(0, cap) : text;
  if (!body.trim()) return null;
  return (
    <pre
      data-testid="artifact-code"
      data-lang={lang || undefined}
      tabIndex={0}
      className={cn(
        "mt-2 max-h-[28rem] min-w-0 overflow-auto rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[11.5px] leading-[1.6] text-foreground/90",
        dim && "opacity-70",
        className,
      )}
    >
      <code className="block whitespace-pre">{body}</code>
    </pre>
  );
}

export function WorkspaceFileView({
  workspace,
  path,
  source,
}: {
  workspace?: string;
  path: string;
  source?: boolean;
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
  if (!workspace || !path) return null;
  if (err) return <p className="mt-2 text-[12px] text-muted">{err}</p>;
  if (binary) return <p className="mt-2 text-[12px] text-muted">{copy.transcript.binaryFile}</p>;
  const asHtml = !source && (html || looksLikeHTML(text) || looksLikeHTMLFile(path));
  if (asHtml && looksLikeHTML(text)) return <SandboxedFrame html={text} title={path} />;
  if (text) return <CodePreview lang={lang || langFromPath(path)} text={text} />;
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
    </div>
  );
}
