/**
 * 拥有：冻结只留最后两块、相同输入命中缓存，以及非追加输入换代。
 * 不拥有：解析器的实现。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { IncrementalMarkdownParser } from "./incremental-md.ts";
import { parseGfm } from "./parse-gfm.ts";

describe("IncrementalMarkdownParser", () => {
  /** 超过两块时冻住前面的，尾部留两块。再追加时已冻结的前缀保持原样，代不变。 */
  it("freezes all but the trailing two blocks", () => {
    const parser = new IncrementalMarkdownParser(parseGfm);
    const first = parser.update("a\n\nb\n\nc\n\nd\n\ne");
    assert.deepEqual(first.frozen.map((b) => b.node.type), ["paragraph", "paragraph", "paragraph"]);
    assert.equal(first.tail.length, 2);
    assert.ok(first.tailStart > 0);
    const frozenSrc = "a\n\nb\n\nc\n\nd\n\ne".slice(0, first.tailStart);
    assert.match(frozenSrc, /a/);
    const second = parser.update("a\n\nb\n\nc\n\nd\n\ne\n\nf\n\ng");
    assert.equal(second.frozen.length, 5);
    assert.equal(second.tail.length, 2);
    assert.deepEqual(second.frozen.slice(0, 3), first.frozen);
    assert.equal(second.generation, first.generation);
  });

  /** 只有两块时全部留在尾部，冻结偏移为 0。 */
  it("holds every block in the tail until more than two exist", () => {
    const parser = new IncrementalMarkdownParser(parseGfm);
    const result = parser.update("only\n\ntwo blocks");
    assert.equal(result.frozen.length, 0);
    assert.equal(result.tail.length, 2);
    assert.equal(result.tailStart, 0);
  });

  /** 相同输入返回同一次结果。 */
  it("returns the cached result for identical input", () => {
    const parser = new IncrementalMarkdownParser(parseGfm);
    const first = parser.update("a\n\nb\n\nc");
    assert.equal(parser.update("a\n\nb\n\nc"), first);
  });

  /** 不是追加时代加一，已冻结的块丢掉。 */
  it("bumps generation and discards frozen blocks on non-append input", () => {
    const parser = new IncrementalMarkdownParser(parseGfm);
    const before = parser.update("a\n\nb\n\nc\n\nd");
    assert.ok(before.frozen.length > 0);
    const after = parser.update("different");
    assert.equal(after.generation, before.generation + 1);
    assert.equal(after.frozen.length, 0);
    assert.deepEqual(after.tail.map((b) => b.node.type), ["paragraph"]);
  });
});
