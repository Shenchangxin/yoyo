/**
 * 拥有：当前语言、系统语言探测，以及金额和时长的格式。
 * 不拥有：文案句子本身。那是 copy 的事。
 */

import { useSyncExternalStore } from "react";
import { catalog, copy, en, setCopyCatalog, type Copy } from "./copy";
import { zhCN } from "./copy-zh";

/** Locale 英文或简体中文。 */
export type Locale = "en" | "zh-CN";

const catalogs: Record<Locale, Copy> = { en: en as Copy, "zh-CN": zhCN };

let locale: Locale = "en";
const listeners = new Set<() => void>();

/** 浏览器语言以 zh 开头则为简体中文，否则英文。读不到语言时为英文。 */
function systemLocale(): Locale {
  try {
    const lang = (navigator.language || "en").toLowerCase();
    if (lang.startsWith("zh")) return "zh-CN";
  } catch {
    /* 读不到浏览器语言 */
  }
  return "en";
}

/** 读出本地记下的语言。只认 zh-CN 和 en。 */
function readStored(): Locale | null {
  try {
    const saved = localStorage.getItem("yoyo-locale");
    if (saved === "zh-CN" || saved === "en") return saved;
  } catch {
    /* 读失败时忽略 */
  }
  return null;
}

/** detectLocale 端到端测试固定英文。否则用记下的语言，没有则跟系统。 */
export function detectLocale(): Locale {
  if (import.meta.env.VITE_E2E) return "en";
  return readStored() || systemLocale();
}

/** getLocale 当前语言。 */
export function getLocale(): Locale {
  return locale;
}

/** setLocale 切换语言、换上文案表，并通知订阅者。写入失败时忽略。 */
export function setLocale(next: Locale) {
  locale = next;
  setCopyCatalog(catalogs[next] || en);
  try {
    localStorage.setItem("yoyo-locale", next);
  } catch {
    /* 写入失败时忽略 */
  }
  listeners.forEach((l) => l());
}

/** applyLocale 只接受 zh-CN 和 en。其他值重新探测。 */
export function applyLocale(next?: string) {
  if (next === "zh-CN" || next === "en") {
    setLocale(next);
    return;
  }
  setLocale(detectLocale());
}

applyLocale(detectLocale());

/** useCopy 订阅语言变化，返回当前文案表。 */
export function useCopy(): Copy {
  useSyncExternalStore(
    (cb) => {
      listeners.add(cb);
      return () => listeners.delete(cb);
    },
    () => locale,
    () => locale,
  );
  return catalog();
}

/** formatMoney 按当前语言格式成美元。非法数字为破折号。大于 0 且小于 0.01 时最少四位小数，小于 1 时最多四位，否则最多两位。 */
export function formatMoney(n: number): string {
  if (!Number.isFinite(n)) return "—";
  try {
    return new Intl.NumberFormat(locale === "zh-CN" ? "zh-CN" : "en-US", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: n > 0 && n < 0.01 ? 4 : 2,
      maximumFractionDigits: n < 1 ? 4 : 2,
    }).format(n);
  } catch {
    return `$${n.toFixed(2)}`;
  }
}

/** formatDurationMs 非法或负数为破折号。不足一秒、不足一分钟和更长分别格式化。不足十秒保留一位小数。中文用「秒」和「分」。 */
export function formatDurationMs(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "—";
  const zh = locale === "zh-CN";
  if (ms < 1000) return zh ? "不到 1 秒" : "< 1s";
  const s = ms / 1000;
  if (s < 60) {
    const v = s < 10 ? s.toFixed(1) : String(Math.round(s));
    return zh ? `${v} 秒` : `${v}s`;
  }
  const m = Math.floor(s / 60);
  const rem = Math.round(s % 60);
  return zh ? `${m} 分 ${rem} 秒` : `${m}m ${rem}s`;
}

/** copy 当前文案表的代理。语言切换后读取走新表。 */
export { copy };
