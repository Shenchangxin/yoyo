/**
 * 拥有：明暗偏好、内置调色板和用户皮肤覆盖层，以及把它们写到文档和原生窗口。
 * 不拥有：皮肤令牌的校验和壁纸取色。
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { isCompanionSurface } from "./popout";
import { applySkinPaint, clearSkinTokens, paintWindowFromTokens, readPersistedSkinId, type SkinPaint } from "./skin/apply";
import { defaultMaterials, isUserSkinId, type SkinFontFace, type SkinMaterials, type TokenMap } from "./skin/schema";

/** ThemePref 明暗偏好：暗色、亮色，或跟随系统。 */
export type ThemePref = "dark" | "light" | "system";
/** ResolvedTheme 已经落到暗色或亮色，不再含跟随系统。 */
export type ResolvedTheme = "dark" | "light";
/** DarkPalette 三套暗色调色板。 */
export type DarkPalette = "ink" | "dim" | "slate";
/** LightPalette 三套亮色调色板。 */
export type LightPalette = "neutral" | "paper" | "mist";
/** PaletteId 当前可用的调色板，暗色或亮色之一。 */
export type PaletteId = DarkPalette | LightPalette;

/** DARK_PALETTES 暗色调色板的固定顺序。 */
export const DARK_PALETTES: readonly DarkPalette[] = ["ink", "dim", "slate"];
/** LIGHT_PALETTES 亮色调色板的固定顺序。 */
export const LIGHT_PALETTES: readonly LightPalette[] = ["neutral", "paper", "mist"];

/** PALETTE_WINDOW 原生窗口背景的 RGB，与 styles.css 里的画布令牌保持一致。 */
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

/** isDarkPalette 是不是三套暗色调色板之一。v 是 ink、dim 或 slate 时返回真。 */
export function isDarkPalette(v: string | undefined | null): v is DarkPalette {
  return v === "ink" || v === "dim" || v === "slate";
}

/** isLightPalette 是不是三套亮色调色板之一。v 是 neutral、paper 或 mist 时返回真。 */
export function isLightPalette(v: string | undefined | null): v is LightPalette {
  return v === "neutral" || v === "paper" || v === "mist";
}

/** parseThemePref 只接受 dark、light、system，其余返回空。v 与 dark、light、system 比较是否相等，有的分支原样返回。 */
export function parseThemePref(v: string | undefined | null): ThemePref | null {
  if (v === "dark" || v === "light" || v === "system") return v;
  return null;
}

/** parseDarkPalette 收成暗色调色板，无法识别时用 ink。v 作为分支结果，传给 isDarkPalette。 */
export function parseDarkPalette(v: string | undefined | null): DarkPalette {
  return isDarkPalette(v) ? v : "ink";
}

/** parseLightPalette 收成亮色调色板，无法识别时用 neutral。v 作为分支结果，传给 isLightPalette。 */
export function parseLightPalette(v: string | undefined | null): LightPalette {
  return isLightPalette(v) ? v : "neutral";
}

/** 系统是否偏好暗色。没有 matchMedia 时视为暗色。 */
function systemDark(): boolean {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? true;
}

/** 跟随系统时看系统偏好，否则偏好本身就是结果。pref 与 system 比较是否相等，有的分支原样返回。 */
function resolve(pref: ThemePref): ResolvedTheme {
  if (pref === "system") return systemDark() ? "dark" : "light";
  return pref;
}

/** 暗色用暗色调色板，亮色用亮色调色板。resolved 为 dark 时返回 dark，否则返回 light。dark 作为分支结果。light 作为分支结果。 */
function activePalette(resolved: ResolvedTheme, dark: DarkPalette, light: LightPalette): PaletteId {
  return resolved === "dark" ? dark : light;
}

/**
 * 把明暗类、调色板和用户皮肤写到文档根。伴随窗口的 color-scheme 保持 normal。
 * resolved 作为分支结果，加进 root.classList。
 * palette 传给 root.setAttribute。
 * skinId 传给 isUserSkinId，传给 root.setAttribute。
 */
function apply(resolved: ResolvedTheme, palette: PaletteId, skinId: string) {
  const root = document.documentElement;
  root.classList.remove("dark", "light");
  root.classList.add(resolved);
  root.setAttribute("data-palette", palette);
  if (isUserSkinId(skinId)) root.setAttribute("data-skin", skinId);
  else root.removeAttribute("data-skin");
  root.style.colorScheme = isCompanionSurface() ? "normal" : resolved;
}

/** 读出明暗偏好。读失败或无法识别时跟随系统。 */
function readPref(): ThemePref {
  try {
    return parseThemePref(localStorage.getItem(KEY)) ?? "system";
  } catch {
    return "system";
  }
}

/** 读出暗色调色板。读失败时用 ink。 */
function readDark(): DarkPalette {
  try {
    return parseDarkPalette(localStorage.getItem(KEY_DARK));
  } catch {
    return "ink";
  }
}

/** 读出亮色调色板。读失败时用 neutral。 */
function readLight(): LightPalette {
  try {
    return parseLightPalette(localStorage.getItem(KEY_LIGHT));
  } catch {
    return "neutral";
  }
}

/** SkinOverlay 盖在内置调色板上的用户皮肤。id 不是用户皮肤时不会留下。 */
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

/** ThemeProvider 持有明暗、调色板和皮肤，并在变化时写到文档和窗口。children 作为子节点渲染。 */
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

  /** 设置明暗偏好。p 传给 setPrefState。 */
  const setPref = useCallback((p: ThemePref) => {
    setPrefState(p);
  }, []);
  /** 设置暗色调色板，无法识别时收成 ink。p 传给 parseDarkPalette。 */
  const setDarkPalette = useCallback((p: DarkPalette) => {
    setDarkState(parseDarkPalette(p));
  }, []);
  /** 设置亮色调色板，无法识别时收成 neutral。p 传给 parseLightPalette。 */
  const setLightPalette = useCallback((p: LightPalette) => {
    setLightState(parseLightPalette(p));
  }, []);
  /** 盖上用户皮肤。id 不合格时清掉覆盖层。next 读取 id，为真时才继续，作为分支结果。 */
  const setSkinOverlay = useCallback((next: SkinOverlay | null) => {
    setSkin(next && isUserSkinId(next.id) ? next : null);
  }, []);
  /** 在跟随系统、暗色、亮色之间循环。 */
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

/** useTheme 读取当前主题上下文。 */
export function useTheme() {
  return useContext(ThemeCtx);
}
