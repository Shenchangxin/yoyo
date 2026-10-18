import { Component, memo, useEffect, useRef, useState, type ErrorInfo, type HTMLAttributes, type ReactNode } from "react";
import { Streamdown, TableCopyDropdown, type Components, type ExtraProps } from "streamdown";
import { useCopy } from "./i18n";
import { cn } from "./utils";
import { codePlugin } from "./code-plugin";
import { IncrementalMarkdownParser, type PositionedBlock } from "./incremental-md";
import { parseGfm } from "./parse-gfm";
import { renderMdastBlocks } from "./mdast-render";

type MdProps<T extends HTMLElement> = HTMLAttributes<T> & ExtraProps & { children?: ReactNode };

const SHIKI_SETTLE_MS = 400;

/**
 * Streamdown ships ChatGPT-scale headings (`text-3xl` / `mt-6`) and
 * `list-inside`. Replace those nodes so the letter stays desk-sized.
 * Tables keep Streamdown's copy dropdown but drop its `bg-muted/80`
 * chrome (`--muted` is ink, not a fill). Do not override `code` —
 * Streamdown uses it for both inline and fences.
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

function tableColumnCount(node: unknown): number {
  const row = (node as { children?: Array<{ children?: unknown[] }> } | undefined)?.children?.[0];
  return Array.isArray(row?.children) ? row.children.length : 0;
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
  a: ({ children, className, node: _node, ...props }: MdProps<HTMLAnchorElement>) => (
    <a {...props} data-streamdown="link" className={cn("md-file-mention", className)}>
      {children}
    </a>
  ),
  table: ({ children, className, node, ...props }: MdProps<HTMLTableElement>) => {
    const cols = tableColumnCount(node);
    const wide = cols >= 4;
    return (
      <div
        className={cn("md-table", wide ? "md-table-wide" : "md-table-fill")}
        data-streamdown="table-wrapper"
        tabIndex={wide ? 0 : undefined}
      >
        <div className="md-table-tools">
          <TableCopyDropdown />
        </div>
        <div className="md-table-scroll">
          <table {...props} data-streamdown="table" className={className}>
            {children}
          </table>
        </div>
      </div>
    );
  },
  thead: ({ children, className, node: _node, ...props }: MdProps<HTMLTableSectionElement>) => (
    <thead {...props} data-streamdown="table-header" className={className}>
      {children}
    </thead>
  ),
  tbody: ({ children, className, node: _node, ...props }: MdProps<HTMLTableSectionElement>) => (
    <tbody {...props} data-streamdown="table-body" className={className}>
      {children}
    </tbody>
  ),
  tr: ({ children, className, node: _node, ...props }: MdProps<HTMLTableRowElement>) => (
    <tr {...props} data-streamdown="table-row" className={className}>
      {children}
    </tr>
  ),
  th: ({ children, className, node: _node, ...props }: MdProps<HTMLTableCellElement>) => (
    <th {...props} data-streamdown="table-header-cell" className={className}>
      {children}
    </th>
  ),
  td: ({ children, className, node: _node, ...props }: MdProps<HTMLTableCellElement>) => (
    <td {...props} data-streamdown="table-cell" className={className}>
      {children}
    </td>
  ),
};

class MarkdownBoundary extends Component<{ text: string; fallback: string; children: ReactNode }, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError(): { failed: boolean } {
    return { failed: true };
  }

  componentDidCatch(err: Error, info: ErrorInfo) {
    console.warn("[markdown]", err.message, info.componentStack);
  }

  render() {
    if (this.state.failed) {
      return (
        <div data-testid="markdown-fallback">
          <p className="mb-2 text-[11px] text-muted">{this.props.fallback}</p>
          <pre className="overflow-auto whitespace-pre-wrap px-3 py-2 font-mono text-[12px] leading-[1.6] text-foreground/90">
            {this.props.text}
          </pre>
        </div>
      );
    }
    return this.props.children;
  }
}

const UI_MARKDOWN_CAP = 24_000;

export const Markdown = memo(function Markdown({
  text,
  streaming,
  quiet,
  rich,
}: {
  text: string;
  streaming?: boolean;
  quiet?: boolean;
  /** Streamdown+Shiki. Historical turns pass false so WebView2 does not keep a highlighter per letter. */
  rich?: boolean;
}) {
  const copy = useCopy();
  const live = !!streaming;
  const decorate = (rich ?? !live) && !live;
  const [highlight, setHighlight] = useState(false);
  useEffect(() => {
    if (!decorate) {
      setHighlight(false);
      return;
    }
    const id = window.setTimeout(() => setHighlight(true), SHIKI_SETTLE_MS);
    return () => window.clearTimeout(id);
  }, [decorate]);
  const capped = text && text.length > UI_MARKDOWN_CAP ? `${text.slice(0, UI_MARKDOWN_CAP)}\n…` : text;
  if (!capped && !streaming) return null;
  if (live) {
    return (
      <MarkdownBoundary text={capped || ""} fallback={copy.review.previewFailed}>
        <StreamingMarkdown text={capped || ""} quiet={quiet} />
      </MarkdownBoundary>
    );
  }
  const settled = highlight && !live;
  return (
    <MarkdownBoundary text={capped || ""} fallback={copy.review.previewFailed}>
      <Streamdown
        className={cn("md-body", quiet && "md-body-quiet")}
        mode={settled ? "static" : "streaming"}
        parseIncompleteMarkdown={live}
        remend={live ? {} : undefined}
        isAnimating={false}
        caret={live ? "block" : undefined}
        plugins={settled ? { code: codePlugin } : undefined}
        animated={false}
        lineNumbers={false}
        tableMaxHeight="none"
        components={mdComponents}
        translations={{
          copyCode: copy.transcript.copyCode,
          copyTable: copy.transcript.copyTable,
          copyTableAsCsv: copy.transcript.copyTableAsCsv,
          copyTableAsMarkdown: copy.transcript.copyTableAsMarkdown,
          copyTableAsTsv: copy.transcript.copyTableAsTsv,
          tableFormatCsv: copy.transcript.tableFormatCsv,
          tableFormatMarkdown: copy.transcript.tableFormatMarkdown,
          tableFormatTsv: copy.transcript.tableFormatTsv,
        }}
        controls={{
          code: { copy: true, download: false },
          table: { copy: true, download: false, fullscreen: false },
          mermaid: false,
          image: { download: false },
        }}
      >
        {capped || ""}
      </Streamdown>
    </MarkdownBoundary>
  );
});

const streamControls = {
  code: { copy: false, download: false },
  table: { copy: false, download: false, fullscreen: false },
  mermaid: false,
  image: { download: false },
} as const;

const TailMarkdown = memo(function TailMarkdown({ text }: { text: string }) {
  if (!text) return null;
  return (
    <Streamdown
      className="md-body"
      mode="streaming"
      parseIncompleteMarkdown
      remend={{}}
      isAnimating={false}
      animated={false}
      lineNumbers={false}
      tableMaxHeight="none"
      components={mdComponents}
      controls={streamControls}
    >
      {text}
    </Streamdown>
  );
});

class StreamingRenderer {
  private readonly parser = new IncrementalMarkdownParser(parseGfm);
  private generation = -1;
  private frozenCount = 0;
  private frozenElements: ReactNode[] = [];
  private lastText: string | null = null;
  private lastRendered: { frozen: ReactNode[]; tail: string; generation: number } | null = null;

  render(text: string): { frozen: ReactNode[]; tail: string; generation: number } {
    if (text === this.lastText && this.lastRendered) return this.lastRendered;
    let frozen: readonly PositionedBlock[] = [];
    let generation = this.generation < 0 ? 0 : this.generation;
    let tailStart = 0;
    try {
      const blocks = this.parser.update(text);
      frozen = blocks.frozen;
      generation = blocks.generation;
      tailStart = blocks.tailStart;
    } catch {
      frozen = [];
      tailStart = 0;
    }
    if (generation !== this.generation) {
      this.generation = generation;
      this.frozenCount = 0;
      this.frozenElements = [];
    }
    if (frozen.length > this.frozenCount) {
      const newly = frozen.slice(this.frozenCount);
      this.frozenElements = [...this.frozenElements, ...renderMdastBlocks(newly)];
      this.frozenCount = frozen.length;
    }
    const tail = text.slice(tailStart);
    this.lastText = text;
    this.lastRendered = { frozen: this.frozenElements, tail, generation };
    return this.lastRendered;
  }
}

function StreamingMarkdown({ text, quiet }: { text: string; quiet?: boolean }) {
  const renderer = useRef<StreamingRenderer | null>(null);
  if (!renderer.current) renderer.current = new StreamingRenderer();
  const { frozen, tail, generation } = renderer.current.render(text);
  return (
    <div className={cn("md-body md-stream", quiet && "md-body-quiet")} data-testid="markdown-stream">
      {frozen.length ? <div key={generation}>{frozen}</div> : null}
      {tail ? <TailMarkdown key={`tail-${generation}`} text={tail} /> : null}
      <span className="md-stream-caret ml-px inline-block h-[1.05em] w-[0.45em] translate-y-px bg-foreground/55" aria-hidden />
    </div>
  );
}
