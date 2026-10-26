/**
 * 拥有：语言别名、全文做键、缓存条数上限，以及超长代码不进高亮。
 * 不拥有：高亮器的实现。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  codeCacheStats,
  codePlugin,
  CODE_CACHE_MAX_ENTRIES,
  MAX_HIGHLIGHT_CHARS,
} from "./code-plugin.ts";

type Result = NonNullable<ReturnType<typeof codePlugin.highlight>>;

/** 把 token 拼回源码。 */
function text(r: Result): string {
  return r.tokens.map((line) => line.map((t) => t.content).join("")).join("\n");
}

/** 同步命中直接取，第一次加载则等回调。 */
function highlightText(code: string, language: string): Promise<string> {
  return new Promise((resolve) => {
    const sync = codePlugin.highlight({ code, language: language as never, themes: codePlugin.getThemes() }, (r) =>
      resolve(text(r)),
    );
    if (sync) resolve(text(sync));
  });
}

describe("codePlugin language support", () => {
  /** ts 和 typescript 都支持。不认识的语言不支持。 */
  it("resolves aliases and rejects unknown languages", () => {
    assert.equal(codePlugin.supportsLanguage("ts" as never), true);
    assert.equal(codePlugin.supportsLanguage("typescript" as never), true);
    assert.equal(codePlugin.supportsLanguage("definitely-not-a-language" as never), false);
  });
});

describe("codePlugin highlighting", () => {
  /** 高亮后拼回的正文与源码一致。 */
  it("returns the full source text, highlighted", async () => {
    const code = "const answer = 42;\nconsole.log(answer);";
    assert.equal(await highlightText(code, "ts"), code);
  });

  /** 长度和首尾相同的两段代码不能共用 token。热缓存按全文取回。 */
  it("keys by the full text: same length and edges must not share tokens", async () => {
    const head = "// header ".repeat(20);
    const tail = "// footer ".repeat(20);
    const a = `${head}const value = 1111;\n${tail}`;
    const b = `${head}const value = 2222;\n${tail}`;
    assert.equal(a.length, b.length);
    assert.equal(await highlightText(a, "ts"), a);
    assert.equal(await highlightText(b, "ts"), b);
    const warm = codePlugin.highlight({ code: a, language: "ts" as never, themes: codePlugin.getThemes() });
    assert.ok(warm, "a warm result should come back synchronously");
    assert.equal(text(warm), a);
  });

  /** 超过条数上限之后，缓存条数仍不超过 CODE_CACHE_MAX_ENTRIES。 */
  it("keeps the cache bounded while many distinct blocks are highlighted", async () => {
    for (let i = 0; i < CODE_CACHE_MAX_ENTRIES + 40; i++) {
      await highlightText(`export const n${i} = ${i};`, "ts");
    }
    const stats = codeCacheStats();
    assert.ok(stats.entries <= CODE_CACHE_MAX_ENTRIES, `entries ${stats.entries}`);
  });

  /** 超过长度上限返回空，也不调用回调。 */
  it("renders oversized blocks as plain text (no highlight, no callback)", () => {
    let called = false;
    const out = codePlugin.highlight(
      { code: "x".repeat(MAX_HIGHLIGHT_CHARS + 1), language: "ts" as never, themes: codePlugin.getThemes() },
      () => {
        called = true;
      },
    );
    assert.equal(out, null);
    assert.equal(called, false);
  });
});
