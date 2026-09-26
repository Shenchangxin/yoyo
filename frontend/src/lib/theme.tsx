import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { isCompanionSurface } from "./popout";

export type ThemePref = "dark" | "light" | "system";
export type ResolvedTheme = "dark" | "light";
export type DarkPalette = "ink" | "dim" | "slate";
export type LightPalette = "neutral" | "paper" | "mist";
export type PaletteId = DarkPalette | LightPalette;

export const DARK_PALETTES: readonly DarkPalette[] = ["ink", "dim", "slate"];
export const LIGHT_PALETTES: readonly LightPalette[] = ["neutral", "paper", "mist"];

/** Native window chrome RGB — keep in lockstep with styles.css canvas tokens. */
export const PALETTE_WINDOW: Record<PaletteId, readonly [number, number, number]> = {
  ink: [22, 19, 16],
  dim: [34, 30, 26],
  slate: [22, 24, 28],
  neutral: [245, 245, 243],
  paper: [244, 241, 235],
  mist: [247, 249, 250],
};

const KEY = "yoyo-theme";
const KEY_DARK = "yoyo-palette-dark";
const KEY_LIGHT = "yoyo-palette-light";

export function isDarkPalette(v: string | undefined | null): v is DarkPalette {
  return v === "ink" || v === "dim" || v === "slate";
}

export function isLightPalette(v: string | undefined | null): v is LightPalette {
  return v === "neutral" || v === "paper" || v === "mist";
}

export function parseThemePref(v: string | undefined | null): ThemePref | null {
  if (v === "dark" || v === "light" || v === "system") return v;
  return null;
}

export function parseDarkPalette(v: string | undefined | null): DarkPalette {
  return isDarkPalette(v) ? v : "ink";
}

export function parseLightPalette(v: string | undefined | null): LightPalette {
  return isLightPalette(v) ? v : "neutral";
}

function systemDark(): boolean {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? true;
}

function resolve(pref: ThemePref): ResolvedTheme {
  if (pref === "system") return systemDark() ? "dark" : "light";
  return pref;
}

function activePalette(resolved: ResolvedTheme, dark: DarkPalette, light: LightPalette): PaletteId {
  return resolved === "dark" ? dark : light;
}

function apply(resolved: ResolvedTheme, palette: PaletteId) {
  const root = document.documentElement;
  root.classList.remove("dark", "light");
  root.classList.add(resolved);
  root.setAttribute("data-palette", palette);
  root.style.colorScheme = isCompanionSurface() ? "normal" : resolved;
}

function paintWindow(palette: PaletteId) {
  import("@wailsio/runtime")
    .then(({ Window }) => {
      if (isCompanionSurface()) {
        Window.SetBackgroundColour(0, 0, 0, 0);
        return;
      }
      const rgb = PALETTE_WINDOW[palette];
      Window.SetBackgroundColour(rgb[0], rgb[1], rgb[2], 255);
    })
    .catch(() => {});
}

function readPref(): ThemePref {
  try {
    return parseThemePref(localStorage.getItem(KEY)) ?? "system";
  } catch {
    return "system";
  }
}

function readDark(): DarkPalette {
  try {
    return parseDarkPalette(localStorage.getItem(KEY_DARK));
  } catch {
    return "ink";
  }
}

function readLight(): LightPalette {
  try {
    return parseLightPalette(localStorage.getItem(KEY_LIGHT));
  } catch {
    return "neutral";
  }
}

type ThemeCtxValue = {
  pref: ThemePref;
  resolved: ResolvedTheme;
  palette: PaletteId;
  darkPalette: DarkPalette;
  lightPalette: LightPalette;
  setPref: (p: ThemePref) => void;
  setDarkPalette: (p: DarkPalette) => void;
  setLightPalette: (p: LightPalette) => void;
  cycle: () => void;
};

const ThemeCtx = createContext<ThemeCtxValue>({
  pref: "system",
  resolved: "dark",
  palette: "ink",
  darkPalette: "ink",
  lightPalette: "neutral",
  setPref: () => {},
  setDarkPalette: () => {},
  setLightPalette: () => {},
  cycle: () => {},
});

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [pref, setPrefState] = useState<ThemePref>(readPref);
  const [darkPalette, setDarkState] = useState<DarkPalette>(readDark);
  const [lightPalette, setLightState] = useState<LightPalette>(readLight);
  const resolved = useMemo(() => resolve(pref), [pref]);
  const palette = activePalette(resolved, darkPalette, lightPalette);

  useEffect(() => {
    apply(resolved, palette);
    try {
      localStorage.setItem(KEY, pref);
      localStorage.setItem(KEY_DARK, darkPalette);
      localStorage.setItem(KEY_LIGHT, lightPalette);
    } catch { /* ignore */ }
    paintWindow(palette);
  }, [pref, resolved, darkPalette, lightPalette, palette]);

  useEffect(() => {
    if (pref !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const on = () => {
      const next = resolve("system");
      apply(next, activePalette(next, darkPalette, lightPalette));
      paintWindow(activePalette(next, darkPalette, lightPalette));
    };
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, [pref, darkPalette, lightPalette]);

  const setPref = useCallback((p: ThemePref) => {
    setPrefState(p);
  }, []);
  const setDarkPalette = useCallback((p: DarkPalette) => {
    setDarkState(parseDarkPalette(p));
  }, []);
  const setLightPalette = useCallback((p: LightPalette) => {
    setLightState(parseLightPalette(p));
  }, []);
  const cycle = useCallback(() => {
    setPrefState((p) => (p === "system" ? "dark" : p === "dark" ? "light" : "system"));
  }, []);

  const value = useMemo<ThemeCtxValue>(
    () => ({ pref, resolved, palette, darkPalette, lightPalette, setPref, setDarkPalette, setLightPalette, cycle }),
    [pref, resolved, palette, darkPalette, lightPalette, setPref, setDarkPalette, setLightPalette, cycle],
  );

  return <ThemeCtx.Provider value={value}>{children}</ThemeCtx.Provider>;
}

export function useTheme() {
  return useContext(ThemeCtx);
}
