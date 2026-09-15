import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

export type ThemePref = "dark" | "light" | "system";
export type ResolvedTheme = "dark" | "light";

const KEY = "yoyo-theme";

function systemDark(): boolean {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? true;
}

function resolve(pref: ThemePref): ResolvedTheme {
  if (pref === "system") return systemDark() ? "dark" : "light";
  return pref;
}

function apply(resolved: ResolvedTheme) {
  const root = document.documentElement;
  root.classList.remove("dark", "light");
  root.classList.add(resolved);
  root.style.colorScheme = resolved;
}

const ThemeCtx = createContext<{
  pref: ThemePref;
  resolved: ResolvedTheme;
  setPref: (p: ThemePref) => void;
  cycle: () => void;
}>({ pref: "system", resolved: "dark", setPref: () => {}, cycle: () => {} });

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [pref, setPref] = useState<ThemePref>(() => {
    try {
      const v = localStorage.getItem(KEY);
      if (v === "dark" || v === "light" || v === "system") return v;
    } catch { /* ignore */ }
    return "system";
  });
  const resolved = useMemo(() => resolve(pref), [pref]);

  useEffect(() => {
    apply(resolved);
    try { localStorage.setItem(KEY, pref); } catch { /* ignore */ }
    import("@wailsio/runtime").then(({ Window }) => {
      const rgb = resolved === "dark" ? [14, 14, 14] : [244, 244, 241];
      return Window.SetBackgroundColour(rgb[0], rgb[1], rgb[2], 255);
    }).catch(() => {});
  }, [pref, resolved]);

  useEffect(() => {
    if (pref !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const on = () => apply(resolve("system"));
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, [pref]);

  function cycle() {
    setPref((p) => (p === "system" ? "dark" : p === "dark" ? "light" : "system"));
  }

  return <ThemeCtx.Provider value={{ pref, resolved, setPref, cycle }}>{children}</ThemeCtx.Provider>;
}

export function useTheme() {
  return useContext(ThemeCtx);
}
