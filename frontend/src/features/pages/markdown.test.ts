/** 拥有页面 Markdown 往返的断言。不拥有编辑器。不改服务端。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { htmlToMd, mdToHtml } from "./markdown.ts";

describe("pages markdown", () => {
  /** 有序列表转过去再转回来仍是编号列表。 */
  it("round-trips ordered lists", () => {
    const src = "1. first\n2. second\n";
    const html = mdToHtml(src);
    assert.match(html, /<ol>/);
    const md = htmlToMd(html);
    assert.match(md, /^1\. first$/m);
    assert.match(md, /^2\. second$/m);
    assert.doesNotMatch(md, /^- first/m);
  });
});
