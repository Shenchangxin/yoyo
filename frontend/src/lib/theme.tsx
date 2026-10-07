import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { isCompanionSurface } from "./popout";
import { applySkinPaint, clearSkinTokens, paintWindowFromTokens, readPersistedSkinId, type SkinPaint } from "./skin/apply";
import { defaultMaterials, isUserSkinId, type SkinFontFace, type SkinMaterials, type TokenMap } from "./skin/schema";

export type ThemePref = "dark" | "light" | "system";
export type ResolvedTheme = "dark" | "light";
export type DarkPalette = "ink" | "dim" | "slate";
export type LightPalette = "neutral" | "paper" | "mist";
export type PaletteId = DarkPalette | LightPalette;

export const DARK_PALETTES: readonly DarkPalette[] = ["ink", "dim", "slate"];
export const LIGHT_PALETTES: readonly LightPalette[] = ["neutral", "paper", "mist"];

/** Native window chrome RGB — keep in lockstep with styles.css canvas tokens. */
export const PALETTE_WINDOW: Record<PaletteId, readonly [number, number, number]> = {
  ink: [28, 28, 26],
  dim: [36, 36, 34],
  slate: [18, 18, 18],
  neutral: [251, 251, 250],
  paper: [246, 243, 236],
  mist: [246, 248, 250],
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

function apply(resolved: ResolvedTheme, palette: PaletteId, skinId: string) {
  const root = document.documentElement;
  root.classList.remove("dark", "light");
  root.classList.add(resolved);
  root.setAttribute("data-palette", palette);
  if (isUserSkinId(skinId)) root.setAttribute("data-skin", skinId);
  else root.removeAttribute("data-skin");
  root.style.colorScheme = isCompanionSurface() ? "normal" : resolved;
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

export type SkinOverlay = {
  id: string;
  tokens: TokenMap;
  materials: SkinMaterials;
  wallpaperUrl?: string;
  wallpaperFile?: string;
  fonts?: SkinFontFace[];
  preserveEvidence?: boolean;
};

type ThemeCtxValue = {
  pref: ThemePref;
  resolved: ResolvedTheme;
  palette: PaletteId;
  darkPalette: DarkPalette;
  lightPalette: LightPalette;
  skinId: string;
  skin: SkinOverlay | null;
  setPref: (p: ThemePref) => void;
  setDarkPalette: (p: DarkPalette) => void;
  setLightPalette: (p: LightPalette) => void;
  setSkinOverlay: (next: SkinOverlay | null, opts?: { persist?: boolean }) => void;
  cycle: () => void;
};

const ThemeCtx = createContext<ThemeCtxValue>({
  pref: "system",
  resolved: "dark",
  palette: "ink",
  darkPalette: "ink",
  lightPalette: "neutral",
  skinId: "",
  skin: null,
  setPref: () => {},
  setDarkPalette: () => {},
  setLightPalette: () => {},
  setSkinOverlay: () => {},
  cycle: () => {},
});

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [pref, setPrefState] = useState<ThemePref>(readPref);
  const [darkPalette, setDarkState] = useState<DarkPalette>(readDark);
  const [lightPalette, setLightState] = useState<LightPalette>(readLight);
  const [skin, setSkin] = useState<SkinOverlay | null>(null);
  const resolved = useMemo(() => resolve(pref), [pref]);
  const palette = activePalette(resolved, darkPalette, lightPalette);
  const skinId = skin?.id || "";

  useEffect(() => {
    apply(resolved, palette, skinId);
    try {
      localStorage.setItem(KEY, pref);
      localStorage.setItem(KEY_DARK, darkPalette);
      localStorage.setItem(KEY_LIGHT, lightPalette);
    } catch { /* ignore */ }
    if (skin && isUserSkinId(skin.id)) {
      const paint: SkinPaint = {
        id: skin.id,
        tokens: skin.tokens,
        materials: skin.materials || defaultMaterials(),
        wallpaperUrl: skin.wallpaperUrl,
        wallpaperFile: skin.wallpaperFile,
        fonts: skin.fonts,
        preserveEvidence: skin.preserveEvidence,
        persist: true,
      };
      applySkinPaint(paint);
      paintWindowFromTokens(skin.tokens, PALETTE_WINDOW[palette]);
    } else {
      clearSkinTokens();
      paintWindowFromTokens(null, PALETTE_WINDOW[palette]);
    }
  }, [pref, resolved, darkPalette, lightPalette, palette, skin, skinId]);

  useEffect(() => {
    if (pref !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const on = () => {
      const next = resolve("system");
      apply(next, activePalette(next, darkPalette, lightPalette), skinId);
      paintWindowFromTokens(skin?.tokens || null, PALETTE_WINDOW[activePalette(next, darkPalette, lightPalette)]);
    };
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, [pref, darkPalette, lightPalette, skin, skinId]);

  const setPref = useCallback((p: ThemePref) => {
    setPrefState(p);
  }, []);
  const setDarkPalette = useCallback((p: DarkPalette) => {
    setDarkState(parseDarkPalette(p));
  }, []);
  const setLightPalette = useCallback((p: LightPalette) => {
    setLightState(parseLightPalette(p));
  }, []);
  const setSkinOverlay = useCallback((next: SkinOverlay | null) => {
    setSkin(next && isUserSkinId(next.id) ? next : null);
  }, []);
  const cycle = useCallback(() => {
    setPrefState((p) => (p === "system" ? "dark" : p === "dark" ? "light" : "system"));
  }, []);

  const value = useMemo<ThemeCtxValue>(
    () => ({
      pref,
      resolved,
      palette,
      darkPalette,
      lightPalette,
      skinId: skinId || readPersistedSkinId(),
      skin,
      setPref,
      setDarkPalette,
      setLightPalette,
      setSkinOverlay,
      cycle,
    }),
    [pref, resolved, palette, darkPalette, lightPalette, skin, skinId, setPref, setDarkPalette, setLightPalette, setSkinOverlay, cycle],
  );

  return <ThemeCtx.Provider value={value}>{children}</ThemeCtx.Provider>;
}

export function useTheme() {
  return useContext(ThemeCtx);
}
