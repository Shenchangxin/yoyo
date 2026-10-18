import { createElement, type ReactNode } from "react";
import type { RootContent, PhrasingContent } from "mdast";
import type { PositionedBlock } from "./incremental-md";

function sanitizeUrl(url: string): string {
  try {
    const protocol = new URL(url).protocol;
    if (protocol === "http:" || protocol === "https:" || protocol === "mailto:") return url;
  } catch {
    /* relative */
  }
  return "";
}

function phrasing(nodes: readonly PhrasingContent[] | undefined): ReactNode[] {
  if (!nodes?.length) return [];
  return nodes.map((node, i) => inline(node, i));
}

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

/** Frozen mdast blocks as React elements. Keys are source offsets so React reconciles across freeze. */
export function renderMdastBlocks(blocks: readonly PositionedBlock[]): ReactNode[] {
  const out: ReactNode[] = [];
  for (const b of blocks) {
    out.push(block(b.node, b.key));
  }
  return out;
}
