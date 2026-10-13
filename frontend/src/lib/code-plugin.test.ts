import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  codeCacheStats,
  codePlugin,
  CODE_CACHE_MAX_ENTRIES,
  MAX_HIGHLIGHT_CHARS,
} from "./code-plugin.ts";

type Result = NonNullable<ReturnType<typeof codePlugin.highlight>>;

function text(r: Result): string {
  return r.tokens.map((line) => line.map((t) => t.content).join("")).join("\n");
}

/** Resolve the highlight whether it is sync (warm) or async (first load). */
function highlightText(code: string, language: string): Promise<string> {
  return new Promise((resolve) => {
    const sync = codePlugin.highlight({ code, language: language as never, themes: codePlugin.getThemes() }, (r) =>
      resolve(text(r)),
    );
    if (sync) resolve(text(sync));
  });
}

describe("codePlugin language support", () => {
  it("resolves aliases and rejects unknown languages", () => {
    assert.equal(codePlugin.supportsLanguage("ts" as never), true);
    assert.equal(codePlugin.supportsLanguage("typescript" as never), true);
    assert.equal(codePlugin.supportsLanguage("definitely-not-a-language" as never), false);
  });
});

describe("codePlugin highlighting", () => {
  it("returns the full source text, highlighted", async () => {
    const code = "const answer = 42;\nconsole.log(answer);";
    assert.equal(await highlightText(code, "ts"), code);
  });

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

  it("keeps the cache bounded while many distinct blocks are highlighted", async () => {
    for (let i = 0; i < CODE_CACHE_MAX_ENTRIES + 40; i++) {
      await highlightText(`export const n${i} = ${i};`, "ts");
    }
    const stats = codeCacheStats();
    assert.ok(stats.entries <= CODE_CACHE_MAX_ENTRIES, `entries ${stats.entries}`);
  });

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
