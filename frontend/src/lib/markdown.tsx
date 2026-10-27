/**
 * 拥有：把助手正文画成 markdown。流式时冻住已定稿的块，定稿后再交给 Streamdown 和高亮。
 * 不拥有：增量解析和 mdast 节点的绘制。
 */

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
 * 换成桌面尺寸的标题。Streamdown 自带的是 text-3xl 和 mt-6，列表是 list-inside。表格留它的复制菜单，去掉 bg-muted/80，因为 --muted 是墨色不是填充。不要覆盖 code，Streamdown 用它同时画行内和围栏。
 * tag 用作标签。
 * name 写到 Tag 的 data-streamdown。
 */
function heading(tag: "h1" | "h2" | "h3" | "h4" | "h5" | "h6", name: string) {
  /** MdHeading 根据参数得到结果。children 作为子节点渲染。className 写到 Tag 的 className。_node 函数体内没有读取。props 展开到 Tag。 */
  function MdHeading({ children, className, node: _node, ...props }: MdProps<HTMLHeadingElement>) {
    const Tag = tag;
    return (
      <Tag {...props} data-streamdown={name} className={className}>
        {children}
      </Tag>
    );
  }
  return MdHeading;
}

/** 取表格第一行的单元格数。没有则 0。node 读取 children，为空时不读。 */
function tableColumnCount(node: unknown): number {
  const row = (node as { children?: Array<{ children?: unknown[] }> } | undefined)?.children?.[0];
  return Array.isArray(row?.children) ? row.children.length : 0;
}

const mdComponents: Components = {
  /** h1 用桌面字号渲染一级标题。 */
  h1: heading("h1", "heading-1"),
  /** h2 用桌面字号渲染二级标题。 */
  h2: heading("h2", "heading-2"),
  /** h3 用桌面字号渲染三级标题。 */
  h3: heading("h3", "heading-3"),
  /** h4 用桌面字号渲染四级标题。 */
  h4: heading("h4", "heading-4"),
  /** h5 用桌面字号渲染五级标题。 */
  h5: heading("h5", "heading-5"),
  /** h6 用桌面字号渲染六级标题。 */
  h6: heading("h6", "heading-6"),
  /** 无序列表用外置圆点。嵌在列表项里时再缩进。children 作为子节点渲染。className 传给 cn。_node 函数体内没有读取。props 展开到 ul。 */
  ul: ({ children, className, node: _node, ...props }: MdProps<HTMLUListElement>) => (
    <ul
      {...props}
      data-streamdown="unordered-list"
      className={cn("list-outside list-disc whitespace-normal [li_&]:pl-5", className)}
    >
      {children}
    </ul>
  ),
  /** 有序列表用外置数字。嵌在列表项里时再缩进。children 作为子节点渲染。className 传给 cn。_node 函数体内没有读取。props 展开到 ol。 */
  ol: ({ children, className, node: _node, ...props }: MdProps<HTMLOListElement>) => (
    <ol
      {...props}
      data-streamdown="ordered-list"
      className={cn("list-outside list-decimal whitespace-normal [li_&]:pl-5", className)}
    >
      {children}
    </ol>
  ),
  /** 列表项里的段落保持行内。children 作为子节点渲染。className 传给 cn。_node 函数体内没有读取。props 展开到 li。 */
  li: ({ children, className, node: _node, ...props }: MdProps<HTMLLIElement>) => (
    <li {...props} data-streamdown="list-item" className={cn("[&>p]:inline", className)}>
      {children}
    </li>
  ),
  /** 引用块只去掉 node。children 作为子节点渲染。className 写到 blockquote 的 className。_node 函数体内没有读取。props 展开到 blockquote。 */
  blockquote: ({ children, className, node: _node, ...props }: MdProps<HTMLQuoteElement>) => (
    <blockquote {...props} data-streamdown="blockquote" className={className}>
      {children}
    </blockquote>
  ),
  /** 分隔线只去掉 node。className 写到 hr 的 className。_node 函数体内没有读取。props 展开到 hr。 */
  hr: ({ className, node: _node, ...props }: MdProps<HTMLHRElement>) => (
    <hr {...props} data-streamdown="horizontal-rule" className={className} />
  ),
  /** 链接加上文件提及的类名。children 作为子节点渲染。className 传给 cn。_node 函数体内没有读取。props 展开到 a。 */
  a: ({ children, className, node: _node, ...props }: MdProps<HTMLAnchorElement>) => (
    <a {...props} data-streamdown="link" className={cn("md-file-mention", className)}>
      {children}
    </a>
  ),
  /** 表格留复制菜单。不少于四列时加宽并可以聚焦。children 作为子节点渲染。className 写到 table 的 className。node 传给 tableColumnCount。props 展开到 table。 */
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
  /** 表头只去掉 node。children 作为子节点渲染。className 写到 thead 的 className。_node 函数体内没有读取。props 展开到 thead。 */
  thead: ({ children, className, node: _node, ...props }: MdProps<HTMLTableSectionElement>) => (
    <thead {...props} data-streamdown="table-header" className={className}>
      {children}
    </thead>
  ),
  /** 表体只去掉 node。children 作为子节点渲染。className 写到 tbody 的 className。_node 函数体内没有读取。props 展开到 tbody。 */
  tbody: ({ children, className, node: _node, ...props }: MdProps<HTMLTableSectionElement>) => (
    <tbody {...props} data-streamdown="table-body" className={className}>
      {children}
    </tbody>
  ),
  /** 表行只去掉 node。children 作为子节点渲染。className 写到 tr 的 className。_node 函数体内没有读取。props 展开到 tr。 */
  tr: ({ children, className, node: _node, ...props }: MdProps<HTMLTableRowElement>) => (
    <tr {...props} data-streamdown="table-row" className={className}>
      {children}
    </tr>
  ),
  /** 表头单元格只去掉 node。children 作为子节点渲染。className 写到 th 的 className。_node 函数体内没有读取。props 展开到 th。 */
  th: ({ children, className, node: _node, ...props }: MdProps<HTMLTableCellElement>) => (
    <th {...props} data-streamdown="table-header-cell" className={className}>
      {children}
    </th>
  ),
  /** 单元格只去掉 node。children 作为子节点渲染。className 写到 td 的 className。_node 函数体内没有读取。props 展开到 td。 */
  td: ({ children, className, node: _node, ...props }: MdProps<HTMLTableCellElement>) => (
    <td {...props} data-streamdown="table-cell" className={className}>
      {children}
    </td>
  ),
};

/** 渲染抛错时改显示原文和一句失败说明。 */
class MarkdownBoundary extends Component<{ text: string; fallback: string; children: ReactNode }, { failed: boolean }> {
  state = { failed: false };

  /** 出错后进入失败状态。 */
  static getDerivedStateFromError(): { failed: boolean } {
    return { failed: true };
  }

  /** 把错误和组件栈打到控制台。err 读取 message。info 读取 componentStack。 */
  componentDidCatch(err: Error, info: ErrorInfo) {
    console.warn("[markdown]", err.message, info.componentStack);
  }

  /** 失败时显示回退说明和原文，否则显示子节点。 */
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

/**
 * Markdown 一条助手正文。流式走增量渲染。定稿后等 400 毫秒再打开高亮。超过 24000 字截断并加省略号。
 * text 读取 length，截取，为真时才继续，作为分支结果。
 * streaming 为假时走另一支。
 * quiet 为真时才继续，写到 StreamingMarkdown 的 quiet。
 * rich 为空时改用后备值，为真时才继续。
 */
export const Markdown = memo(function Markdown({
  text,
  streaming,
  quiet,
  rich,
}: {
  text: string;
  streaming?: boolean;
  quiet?: boolean;
  /** 为真时用 Streamdown 加 Shiki。历史回合传假，避免 WebView2 按每封信留一个高亮器。 */
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

/** 不稳定尾部。空文本不渲染。走流式模式且不复制、不下载。text 为假时返回 null；作为子节点渲染。 */
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

/** 按代缓存已冻结块的元素。代变了就清空。解析失败时尾部从 0 开始。 */
class StreamingRenderer {
  private readonly parser = new IncrementalMarkdownParser(parseGfm);
  private generation = -1;
  private frozenCount = 0;
  private frozenElements: ReactNode[] = [];
  private lastText: string | null = null;
  private lastRendered: { frozen: ReactNode[]; tail: string; generation: number } | null = null;

  /** 相同文本返回上一次结果。冻结块变长时只补画新增的。尾部是 tailStart 之后的源码。text 与 this.lastText 比较是否相等，截取，记到 this.lastText，传给 parser.update。 */
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

/** 流式正文：冻结块按代挂上，尾部交给 TailMarkdown，末尾一个光标。text 传给 renderer.current.render。quiet 为真时才继续。 */
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
