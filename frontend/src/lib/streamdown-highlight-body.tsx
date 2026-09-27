import { Fragment, useEffect, useState, type CSSProperties } from "react";
import { codePlugin } from "./code-plugin";

type HlToken = {
  content: string;
  color?: string;
  bgColor?: string;
  htmlStyle?: string | Record<string, string>;
  htmlAttrs?: Record<string, string>;
};

type Props = {
  code: string;
  language?: string;
  className?: string;
  raw?: { tokens?: HlToken[][] };
  maxHeight?: number | string;
  startLine?: number;
  lineNumbers?: boolean;
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

/**
 * Streamdown lazy-loads ./highlighted-body-*.js. Vite's dep optimizer rewrites
 * that to a hashed file under .vite/deps that WebView2 never receives.
 * This module is the same highlighter (via @streamdown/code) served from /src.
 */
export function HighlightedCodeBlockBody({
  code,
  language,
  className,
  raw,
  maxHeight,
  lineNumbers,
  startLine = 1,
}: Props) {
  const [lines, setLines] = useState<HlToken[][] | null>(raw?.tokens ?? null);
  useEffect(() => {
    let live = true;
    const lang = (
      language && codePlugin.supportsLanguage(language as Parameters<typeof codePlugin.supportsLanguage>[0])
        ? language
        : "text"
    ) as Parameters<typeof codePlugin.highlight>[0]["language"];
    const themes = codePlugin.getThemes();
    try {
      const sync = codePlugin.highlight({ code, language: lang, themes }, (next) => {
        if (live && Array.isArray(next?.tokens)) setLines(next.tokens as HlToken[][]);
      });
      if (sync && live && Array.isArray(sync.tokens)) setLines(sync.tokens as HlToken[][]);
    } catch {
      if (live) setLines(raw?.tokens ?? null);
    }
    return () => {
      live = false;
    };
  }, [code, language, raw]);
  const tokens = lines;
  const style = maxHeight != null && maxHeight !== Infinity ? { maxHeight } : undefined;
  return (
    <pre className={className} style={style} tabIndex={0}>
      <code className="code-hl block min-h-full whitespace-pre font-mono text-[12px] leading-[1.55]">
        {tokens
          ? tokens.map((line, i) => (
              <Fragment key={i}>
                {lineNumbers ? (
                  <span className="inline-block w-8 pr-2 text-right text-muted select-none">{startLine + i}</span>
                ) : null}
                {line.map((t, j) => (
                  <span key={j} style={tokenStyle(t)} {...(t.htmlAttrs || {})}>
                    {t.content}
                  </span>
                ))}
                {i < tokens.length - 1 ? "\n" : null}
              </Fragment>
            ))
          : code}
      </code>
    </pre>
  );
}
