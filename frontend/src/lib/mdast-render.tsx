/**
 * 拥有：把已定位的 mdast 块画成 React 元素。
 * 不拥有：增量解析，以及 Streamdown 的定稿渲染。
 */

import { createElement, type ReactNode } from "react";
import type { RootContent, PhrasingContent } from "mdast";
import type { PositionedBlock } from "./incremental-md";

/** 只留 http、https 和 mailto。相对地址和解析失败返回空串。 */
function sanitizeUrl(url: string): string {
  try {
    const protocol = new URL(url).protocol;
    if (protocol === "http:" || protocol === "https:" || protocol === "mailto:") return url;
  } catch {
    /* 相对地址 */
  }
  return "";
}

/** 把一行内的节点顺次画出来。没有节点则空数组。 */
function phrasing(nodes: readonly PhrasingContent[] | undefined): ReactNode[] {
  if (!nodes?.length) return [];
  return nodes.map((node, i) => inline(node, i));
}

/** 画一个行内节点。链接和图片的地址过不了校验时，链接退回文字，图片退回替代文本。其余有子节点的包进 span，有 value 的返回原文。 */
function inline(node: PhrasingContent, key: number): ReactNode {
  switch (node.type) {
    case "text":
      return node.value;
    case "inlineCode":
      return createElement("code", { key, "data-streamdown": "inline-code" }, node.value);
    case "strong":
      return createElement("strong", { key }, phrasing(node.children));
    case "emphasis":
      return createElement("em", { key }, phrasing(node.children));
    case "delete":
      return createElement("del", { key }, phrasing(node.children));
    case "break":
      return createElement("br", { key });
    case "link": {
      const href = sanitizeUrl(String(node.url || ""));
      if (!href) return createElement("span", { key }, phrasing(node.children));
      return createElement("a", { key, href, target: "_blank", rel: "noreferrer", className: "md-file-mention" }, phrasing(node.children));
    }
    case "image": {
      const src = sanitizeUrl(String(node.url || ""));
      if (!src) return node.alt || "";
      return createElement("img", { key, src, alt: node.alt || "", className: "max-w-full rounded" });
    }
    default:
      if ("children" in node && Array.isArray((node as { children?: PhrasingContent[] }).children)) {
        return createElement("span", { key }, phrasing((node as { children: PhrasingContent[] }).children));
      }
      if ("value" in node && typeof (node as { value?: string }).value === "string") {
        return (node as { value: string }).value;
      }
      return null;
  }
}

/** 画一个块。标题深度夹在 1 到 6。表格不少于四列时加宽并可以聚焦。html 块只显示原文。 */
function block(node: RootContent, key: number): ReactNode {
  switch (node.type) {
    case "paragraph":
      return createElement("p", { key }, phrasing(node.children));
    case "heading": {
      const tag = (`h${Math.min(6, Math.max(1, node.depth))}`) as "h1";
      return createElement(tag, { key, "data-streamdown": tag }, phrasing(node.children));
    }
    case "blockquote":
      return createElement("blockquote", { key }, node.children.map((c, i) => block(c, i)));
    case "list": {
      const Tag = node.ordered ? "ol" : "ul";
      return createElement(Tag, { key }, node.children.map((c, i) => block(c, i)));
    }
    case "listItem":
      return createElement("li", { key }, node.children.map((c, i) => block(c, i)));
    case "code":
      return createElement(
        "pre",
        { key, className: "overflow-auto rounded-md bg-lift/40 px-3 py-2 font-mono text-[12px] leading-[1.6]" },
        createElement("code", null, node.value),
      );
    case "thematicBreak":
      return createElement("hr", { key });
    case "table": {
      const rows = node.children;
      const cols = rows[0]?.children?.length ?? 0;
      const wide = cols >= 4;
      const head = rows[0];
      const body = rows.slice(1);
      return createElement(
        "div",
        {
          key,
          className: wide ? "md-table md-table-wide" : "md-table md-table-fill",
          "data-streamdown": "table-wrapper",
          tabIndex: wide ? 0 : undefined,
        },
        createElement(
          "div",
          { className: "md-table-scroll" },
          createElement(
            "table",
            { "data-streamdown": "table" },
            head
              ? createElement(
                  "thead",
                  { "data-streamdown": "table-header" },
                  createElement(
                    "tr",
                    { "data-streamdown": "table-row" },
                    head.children.map((cell, i) =>
                      createElement(
                        "th",
                        { key: i, "data-streamdown": "table-header-cell" },
                        phrasing((cell as { children?: PhrasingContent[] }).children),
                      ),
                    ),
                  ),
                )
              : null,
            body.length
              ? createElement(
                  "tbody",
                  { "data-streamdown": "table-body" },
                  body.map((row, i) =>
                    createElement(
                      "tr",
                      { key: i, "data-streamdown": "table-row" },
                      row.children.map((cell, j) =>
                        createElement(
                          "td",
                          { key: j, "data-streamdown": "table-cell" },
                          phrasing((cell as { children?: PhrasingContent[] }).children),
                        ),
                      ),
                    ),
                  ),
                )
              : null,
          ),
        ),
      );
    }
    case "html":
      return createElement("span", { key }, node.value);
    default:
      if ("children" in node && Array.isArray((node as { children?: RootContent[] }).children)) {
        return createElement("div", { key }, (node as { children: RootContent[] }).children.map((c, i) => block(c, i)));
      }
      return null;
  }
}

/** renderMdastBlocks 把冻结的 mdast 块画成元素。键是源码偏移，跨过冻结边界时 React 能对上。 */
export function renderMdastBlocks(blocks: readonly PositionedBlock[]): ReactNode[] {
  const out: ReactNode[] = [];
  for (const b of blocks) {
    out.push(block(b.node, b.key));
  }
  return out;
}
