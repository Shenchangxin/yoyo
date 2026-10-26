import type { CodeHighlighterPlugin, HighlightOptions, HighlightResult, ThemeInput } from "@streamdown/code";
import { bundledLanguages, bundledLanguagesInfo, createHighlighter, type BundledLanguage } from "shiki";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";

/**
 * 拥有：Streamdown 代码块和工具预览的语法高亮。缓存是按全文做键的有界 LRU，并共用一个按需加载语言的高亮器。
 * 不拥有：代码块的行渲染。替换 `@streamdown/code` 的 codePlugin：那个插件把每次高亮留在模块级 Map 里且从不逐出，长运行会按步增加代码块，WebView2 内存只增不减。它还用长度加首尾各 100 字做键，两段长度和边缘相同的不同代码可能显示成对方的 token。这里保持同一插件约定，调用处不用改。逐出只是再分词一次。聊天记录不动。主题固定为 THEMES，highlight 的 themes 参数被忽略。
 */

const THEME_LIGHT = "github-light";
const THEME_DARK = "github-dark-dimmed";
const THEMES: [ThemeInput, ThemeInput] = [THEME_LIGHT, THEME_DARK];
const PLAIN = "text";
const PLAIN_LANGS = new Set(["text", "plaintext", "txt", "plain"]);

/** MAX_HIGHLIGHT_CHARS 超过则渲染纯文本。分词兆字节会堵住界面线程。 */
export const MAX_HIGHLIGHT_CHARS = 120_000;
/** CODE_CACHE_MAX_ENTRIES LRU 条数上限。token 在 JS 堆上大约是源码的 10 到 20 倍。 */
export const CODE_CACHE_MAX_ENTRIES = 160;
/** CODE_CACHE_MAX_CHARS LRU 源码字符上限。 */
export const CODE_CACHE_MAX_CHARS = 400_000;

const ENGINE = createJavaScriptRegexEngine({ forgiving: true });
const ALIASES: Record<string, string> = Object.fromEntries(
  bundledLanguagesInfo.flatMap((info) => (info.aliases ?? []).map((alias) => [alias, info.id] as const)),
);
const KNOWN = new Set<string>(Object.keys(bundledLanguages));

type Highlighter = Awaited<ReturnType<typeof createHighlighter>>;

/** 去空白并小写。别名换成 shiki 的语言 id。raw 去掉首尾空白，转成小写后，用作下标，作为后备值。 */
function resolveLang(raw: string): string {
  const t = raw.trim().toLowerCase();
  return ALIASES[t] || t;
}

/** 在打包语言里，或是 text、plaintext、txt、plain。lang 传给 KNOWN.has，传给 PLAIN_LANGS.has。 */
function isKnownLang(lang: string): boolean {
  return KNOWN.has(lang) || PLAIN_LANGS.has(lang);
}

/** 已知语言用它，否则用 text。raw 传给 resolveLang。 */
function langOrPlain(raw: string): string {
  const lang = resolveLang(raw);
  return isKnownLang(lang) ? lang : PLAIN;
}

let shared: Promise<Highlighter> | null = null;
let ready: Highlighter | null = null;

/** 共用一个高亮器。创建失败则丢掉承诺，下次再试。 */
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

/**
 * 纯文本或已经加载则返回。同一语言的加载合并成一个承诺。
 * h 调用 getLoadedLanguages，调用 loadLanguage。
 * lang 传给 PLAIN_LANGS.has，传给 includes，传给 loadingLangs.get，传给 h.loadLanguage，传给 loadingLangs.delete，交给 loadingLangs.set。
 */
async function ensureLanguage(h: Highlighter, lang: string): Promise<void> {
  if (PLAIN_LANGS.has(lang) || h.getLoadedLanguages().includes(lang as BundledLanguage)) return;
  let p = loadingLangs.get(lang);
  if (!p) {
    p = h.loadLanguage(lang as BundledLanguage).finally(() => loadingLangs.delete(lang));
    loadingLangs.set(lang, p);
  }
  await p;
}

/** 用固定的亮色和暗色主题把代码分成 token。h 调用 codeToTokens。code 传给 h.codeToTokens。lang 写入字段 lang。 */
function render(h: Highlighter, code: string, lang: string): HighlightResult {
  return h.codeToTokens(code, {
    lang: lang as BundledLanguage,
    themes: { light: THEME_LIGHT, dark: THEME_DARK },
  });
}

type CacheEntry = { lang: string; code: string; tokens: HighlightResult };

/** Map 的迭代顺序是插入顺序，第一个键最久未用。 */
const cache = new Map<string, CacheEntry>();
let cacheChars = 0;

/** 32 位 FNV-1a。两个种子拼成键；命中后仍比较全文。s 读取 length，调用 charCodeAt。seed 无符号右移 0。 */
function fnv1a(s: string, seed: number): string {
  let h = seed >>> 0;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return (h >>> 0).toString(36);
}

/** 语言、长度和两段 FNV 拼成键。lang 拼进字符串。code 读取 length，传给 fnv1a。 */
function cacheKey(lang: string, code: string): string {
  return `${lang}\u0000${code.length}\u0000${fnv1a(code, 0x811c9dc5)}${fnv1a(code, 0x9e3779b9)}`;
}

/** 键命中且语言和全文一致才返回，并把该项移到最近使用。lang 与 hit.lang 比较是否不等，传给 cacheKey。code 与 hit.code 比较是否不等，传给 cacheKey。 */
function cacheGet(lang: string, code: string): HighlightResult | null {
  const key = cacheKey(lang, code);
  const hit = cache.get(key);
  if (!hit || hit.lang !== lang || hit.code !== code) return null;
  cache.delete(key);
  cache.set(key, hit);
  return hit.tokens;
}

/** 写入后按条数和字符数淘汰最久未用。已有同键则先减去旧长度。lang 传给 cacheKey，写入字段 lang。code 读取 length，传给 cacheKey，写入字段 code。tokens 写入字段 tokens。 */
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

/** 等着同一次高亮完成的回调，键是语言加全文。 */
const waiting = new Map<string, Set<(result: HighlightResult) => void>>();

/** 加载语言、分词、写入缓存，并通知等待同一键的回调。失败时打出错误并清掉等待。lang 传给 ensureLanguage，传给 render，传给 cachePut。code 传给 render，传给 cachePut。key 传给 waiting.get，传给 waiting.delete。 */
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

/** codePlugin 给 Streamdown 用的 shiki 高亮插件。 */
export const codePlugin: CodeHighlighterPlugin = {
  name: "shiki",
  type: "code-highlighter",
  /** supportsLanguage 解析别名后是否为已知语言。language 转成字符串，传给 resolveLang。 */
  supportsLanguage(language) {
    return isKnownLang(resolveLang(String(language)));
  },
  /** getSupportedLanguages 打包语言的 id，不含别名。 */
  getSupportedLanguages() {
    return Array.from(KNOWN) as BundledLanguage[];
  },
  /** getThemes 固定的亮色与暗色主题。 */
  getThemes() {
    return THEMES;
  },
  /**
   * highlight 超过 MAX_HIGHLIGHT_CHARS 返回空且不回调。缓存命中或语言已加载则同步返回。否则登记回调并开始加载，返回空。themes 参数忽略。
   * code 读取 length，拼进字符串，传给 cacheGet，传给 render，传给 cachePut，传给 startLoad。
   * language 转成字符串，传给 langOrPlain。
   * callback 为真时才继续，加进 subscribers。
   */
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

/** codeCacheStats 测试和诊断用的当前缓存条数和字符数。 */
export function codeCacheStats(): { entries: number; chars: number } {
  return { entries: cache.size, chars: cacheChars };
}
