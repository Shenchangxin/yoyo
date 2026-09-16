import { useEffect, useState, type ComponentPropsWithoutRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { highlight } from "./highlight";

export function Markdown({ text }: { text: string }) {
  return (
    <div className="md-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          pre: ({ children }) => <>{children}</>,
          code: Code,
        }}
      >
        {text || ""}
      </ReactMarkdown>
    </div>
  );
}

function Code({ className, children, ...props }: ComponentPropsWithoutRef<"code">) {
  const lang = /language-([\w-]+)/.exec(className || "")?.[1];
  const value = String(children ?? "").replace(/\n$/, "");
  const fenced = Boolean(lang) || value.includes("\n");
  if (!fenced) {
    return (
      <code className={className} {...props}>
        {children}
      </code>
    );
  }
  return <Highlighted code={value} lang={lang || "text"} />;
}

function Highlighted({ code, lang }: { code: string; lang: string }) {
  const [html, setHtml] = useState("");
  useEffect(() => {
    let alive = true;
    highlight(code, lang)
      .then((out) => {
        if (alive) setHtml(out);
      })
      .catch(() => {
        if (alive) setHtml("");
      });
    return () => {
      alive = false;
    };
  }, [code, lang]);
  if (!html) {
    return (
      <pre>
        <code className={`language-${lang}`}>{code}</code>
      </pre>
    );
  }
  return <span className="md-shiki" dangerouslySetInnerHTML={{ __html: html }} />;
}
