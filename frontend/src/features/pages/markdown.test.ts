import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { htmlToMd, mdToHtml } from "./markdown.ts";

describe("pages markdown", () => {
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
