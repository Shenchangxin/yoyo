import { memo, type HTMLAttributes, type ReactNode } from "react";
import { Streamdown, type Components, type ExtraProps } from "streamdown";
import { useCopy } from "./i18n";
import { cn } from "./utils";
import { codePlugin } from "./code-plugin";

type MdProps<T extends HTMLElement> = HTMLAttributes<T> & ExtraProps & { children?: ReactNode };

/**
 * Streamdown ships ChatGPT-scale headings (`text-3xl` / `mt-6`) and
 * `list-inside`. Replace those nodes so the letter follows DESIGN.md.
 * Do not override `code` — Streamdown uses it for both inline and fences.
 */
function heading(tag: "h1" | "h2" | "h3" | "h4" | "h5" | "h6", name: string) {
  return function MdHeading({ children, className, node: _node, ...props }: MdProps<HTMLHeadingElement>) {
    const Tag = tag;
    return (
      <Tag {...props} data-streamdown={name} className={className}>
        {children}
      </Tag>
    );
  };
}

const mdComponents: Components = {
  h1: heading("h1", "heading-1"),
  h2: heading("h2", "heading-2"),
  h3: heading("h3", "heading-3"),
  h4: heading("h4", "heading-4"),
  h5: heading("h5", "heading-5"),
  h6: heading("h6", "heading-6"),
  ul: ({ children, className, node: _node, ...props }: MdProps<HTMLUListElement>) => (
    <ul
      {...props}
      data-streamdown="unordered-list"
      className={cn("list-outside list-disc whitespace-normal [li_&]:pl-5", className)}
    >
      {children}
    </ul>
  ),
  ol: ({ children, className, node: _node, ...props }: MdProps<HTMLOListElement>) => (
    <ol
      {...props}
      data-streamdown="ordered-list"
      className={cn("list-outside list-decimal whitespace-normal [li_&]:pl-5", className)}
    >
      {children}
    </ol>
  ),
  li: ({ children, className, node: _node, ...props }: MdProps<HTMLLIElement>) => (
    <li {...props} data-streamdown="list-item" className={cn("[&>p]:inline", className)}>
      {children}
    </li>
  ),
  blockquote: ({ children, className, node: _node, ...props }: MdProps<HTMLQuoteElement>) => (
    <blockquote {...props} data-streamdown="blockquote" className={className}>
      {children}
    </blockquote>
  ),
  hr: ({ className, node: _node, ...props }: MdProps<HTMLHRElement>) => (
    <hr {...props} data-streamdown="horizontal-rule" className={className} />
  ),
};

export const Markdown = memo(function Markdown({
  text,
  streaming,
  quiet,
}: {
  text: string;
  streaming?: boolean;
  quiet?: boolean;
}) {
  const copy = useCopy();
  if (!text && !streaming) return null;
  return (
    <Streamdown
      className={cn("md-body space-y-2", quiet && "md-body-quiet")}
      mode={streaming ? "streaming" : "static"}
      isAnimating={!!streaming}
      caret={streaming ? "circle" : undefined}
      plugins={{ code: codePlugin }}
      animated={false}
      lineNumbers={false}
      codeBlockMaxHeight={Infinity}
      components={mdComponents}
      translations={{
        copyCode: copy.transcript.copyCode,
        copyTable: copy.transcript.copyTable,
      }}
      controls={{
        code: { copy: true, download: false },
        table: { copy: true, download: false, fullscreen: false },
        mermaid: false,
        image: { download: false },
      }}
    >
      {text || ""}
    </Streamdown>
  );
});
