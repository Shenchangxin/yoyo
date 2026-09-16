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

export { copy };
