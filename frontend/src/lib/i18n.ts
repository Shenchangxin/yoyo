import { useSyncExternalStore } from "react";
import { catalog, copy, en, setCopyCatalog, type Copy } from "./copy";
import { zhCN } from "./copy-zh";

export type Locale = "en" | "zh-CN";

const catalogs: Record<Locale, Copy> = { en: en as Copy, "zh-CN": zhCN };

let locale: Locale = "en";
const listeners = new Set<() => void>();

function systemLocale(): Locale {
  try {
    const lang = (navigator.language || "en").toLowerCase();
    if (lang.startsWith("zh")) return "zh-CN";
  } catch {
    /* node */
  }
  return "en";
}

function readStored(): Locale | null {
  try {
    const saved = localStorage.getItem("yoyo-locale");
    if (saved === "zh-CN" || saved === "en") return saved;
  } catch {
    /* ignore */
  }
  return null;
}

export function detectLocale(): Locale {
  if (import.meta.env.VITE_E2E) return "en";
  return readStored() || systemLocale();
}

export function getLocale(): Locale {
  return locale;
}

export function setLocale(next: Locale) {
  locale = next;
  setCopyCatalog(catalogs[next] || en);
  try {
    localStorage.setItem("yoyo-locale", next);
  } catch {
    /* ignore */
  }
  listeners.forEach((l) => l());
}

export function applyLocale(next?: string) {
  if (next === "zh-CN" || next === "en") {
    setLocale(next);
    return;
  }
  setLocale(detectLocale());
}

applyLocale(detectLocale());

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

export { copy };
