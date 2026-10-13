import type { CodeHighlighterPlugin, HighlightOptions, HighlightResult, ThemeInput } from "@streamdown/code";
import { bundledLanguages, bundledLanguagesInfo, createHighlighter, type BundledLanguage } from "shiki";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";

/**
 * Syntax highlighting for Streamdown code blocks and tool previews.
 *
 * Replaces `codePlugin` from `@streamdown/code`, which keeps every highlight
 * result in a module-level Map that is never evicted. A long agent run
 * highlights a new code block (or tool preview) per step, so the renderer
 * retained every token array for the life of the window: the WebView2 memory
 * that only grew. That plugin also keys results by length plus the first and
 * last 100 characters, so two different blocks with the same length and edges
 * could be shown with each other's tokens.
 *
 * This module keeps the same plugin contract (Streamdown and the call sites do
 * not change). It caches results in a bounded LRU keyed by the full text, and
 * shares one highlighter that loads languages on demand. Evicting a highlight
 * only costs a re-tokenize. Chat history is never touched.
 *
 * The themes are fixed (THEMES); the `themes` argument of `highlight` is ignored.
 */

const THEME_LIGHT = "github-light";
const THEME_DARK = "github-dark-dimmed";
const THEMES: [ThemeInput, ThemeInput] = [THEME_LIGHT, THEME_DARK];
const PLAIN = "text";
const PLAIN_LANGS = new Set(["text", "plaintext", "txt", "plain"]);

/** Past this size, render plain text: tokenizing megabytes stalls the UI thread. */
export const MAX_HIGHLIGHT_CHARS = 120_000;
/** LRU budget. Tokens take roughly 10-20x the source size on the JS heap. */
export const CODE_CACHE_MAX_ENTRIES = 160;
export const CODE_CACHE_MAX_CHARS = 400_000;

const ENGINE = createJavaScriptRegexEngine({ forgiving: true });
const ALIASES: Record<string, string> = Object.fromEntries(
  bundledLanguagesInfo.flatMap((info) => (info.aliases ?? []).map((alias) => [alias, info.id] as const)),
);
const KNOWN = new Set<string>(Object.keys(bundledLanguages));

type Highlighter = Awaited<ReturnType<typeof createHighlighter>>;

function resolveLang(raw: string): string {
  const t = raw.trim().toLowerCase();
  return ALIASES[t] || t;
}

function isKnownLang(lang: string): boolean {
  return KNOWN.has(lang) || PLAIN_LANGS.has(lang);
}

function langOrPlain(raw: string): string {
  const lang = resolveLang(raw);
  return isKnownLang(lang) ? lang : PLAIN;
}

let shared: Promise<Highlighter> | null = null;
let ready: Highlighter | null = null;

function getHighlighter(): Promise<Highlighter> {
  if (!shared) {
    shared = createHighlighter({ themes: THEMES, langs: [], engine: ENGINE }).then(
      (h) => {
        ready = h;
        return h;
      },
      (err) => {
        shared = null;
        throw err;
      },
    );
  }
  return shared;
}

const loadingLangs = new Map<string, Promise<void>>();

async function ensureLanguage(h: Highlighter, lang: string): Promise<void> {
  if (PLAIN_LANGS.has(lang) || h.getLoadedLanguages().includes(lang as BundledLanguage)) return;
  let p = loadingLangs.get(lang);
  if (!p) {
    p = h.loadLanguage(lang as BundledLanguage).finally(() => loadingLangs.delete(lang));
    loadingLangs.set(lang, p);
  }
  await p;
}

function render(h: Highlighter, code: string, lang: string): HighlightResult {
  return h.codeToTokens(code, {
    lang: lang as BundledLanguage,
    themes: { light: THEME_LIGHT, dark: THEME_DARK },
  });
}

/* ---- bounded LRU -------------------------------------------------------- */

type CacheEntry = { lang: string; code: string; tokens: HighlightResult };

/** Map iteration order is insertion order: first key = least recently used. */
const cache = new Map<string, CacheEntry>();
let cacheChars = 0;

/** 32-bit FNV-1a. Two seeds give a 64-bit key; hits still compare the full text. */
function fnv1a(s: string, seed: number): string {
  let h = seed >>> 0;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return (h >>> 0).toString(36);
}

function cacheKey(lang: string, code: string): string {
  return `${lang}\u0000${code.length}\u0000${fnv1a(code, 0x811c9dc5)}${fnv1a(code, 0x9e3779b9)}`;
}

function cacheGet(lang: string, code: string): HighlightResult | null {
  const key = cacheKey(lang, code);
  const hit = cache.get(key);
  if (!hit || hit.lang !== lang || hit.code !== code) return null;
  cache.delete(key);
  cache.set(key, hit);
  return hit.tokens;
}

function cachePut(lang: string, code: string, tokens: HighlightResult): void {
  const key = cacheKey(lang, code);
  const prev = cache.get(key);
  if (prev) {
    cacheChars -= prev.code.length;
    cache.delete(key);
  }
  cache.set(key, { lang, code, tokens });
  cacheChars += code.length;
  while (cache.size > CODE_CACHE_MAX_ENTRIES || cacheChars > CODE_CACHE_MAX_CHARS) {
    const oldest = cache.keys().next();
    if (oldest.done) break;
    const entry = cache.get(oldest.value);
    cache.delete(oldest.value);
    if (entry) cacheChars -= entry.code.length;
  }
}

/** Callers waiting on an in-flight highlight, keyed by exact language and text. */
const waiting = new Map<string, Set<(result: HighlightResult) => void>>();

function startLoad(lang: string, code: string, key: string): void {
  getHighlighter()
    .then(async (h) => {
      await ensureLanguage(h, lang);
      const tokens = render(h, code, lang);
      cachePut(lang, code, tokens);
      const subscribers = waiting.get(key);
      waiting.delete(key);
      subscribers?.forEach((cb) => cb(tokens));
    })
    .catch((err) => {
      waiting.delete(key);
      console.error("[code-plugin] highlight failed", lang, err);
    });
}

/* ---- plugin -------------------------------------------------------------- */

export const codePlugin: CodeHighlighterPlugin = {
  name: "shiki",
  type: "code-highlighter",
  supportsLanguage(language) {
    return isKnownLang(resolveLang(String(language)));
  },
  getSupportedLanguages() {
    return Array.from(KNOWN) as BundledLanguage[];
  },
  getThemes() {
    return THEMES;
  },
  highlight({ code, language }: HighlightOptions, callback?: (result: HighlightResult) => void) {
    if (code.length > MAX_HIGHLIGHT_CHARS) return null;
    const lang = langOrPlain(String(language));
    const hit = cacheGet(lang, code);
    if (hit) return hit;
    if (ready) {
      const direct = PLAIN_LANGS.has(lang) || ready.getLoadedLanguages().includes(lang as BundledLanguage);
      if (direct) {
        const tokens = render(ready, code, lang);
        cachePut(lang, code, tokens);
        return tokens;
      }
    }
    const key = `${lang}\u0000${code}`;
    let subscribers = waiting.get(key);
    if (!subscribers) {
      subscribers = new Set();
      waiting.set(key, subscribers);
      startLoad(lang, code, key);
    }
    if (callback) subscribers.add(callback);
    return null;
  },
};

/** Test and diagnostics hook: current cache occupancy. */
export function codeCacheStats(): { entries: number; chars: number } {
  return { entries: cache.size, chars: cacheChars };
}
