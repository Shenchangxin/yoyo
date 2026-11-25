/**
 * 拥有：明暗偏好和用户皮肤覆盖层，以及把它们写到文档和原生窗口。
 * 不拥有：皮肤令牌的校验和壁纸取色。产品只有深色和浅色两套外观。
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { isCompanionSurface } from "./popout";
import { applySkinPaint, clearSkinTokens, paintWindowFromTokens, readPersistedSkinId, type SkinPaint } from "./skin/apply";
import { defaultMaterials, isUserSkinId, type SkinFontFace, type SkinMaterials, type TokenMap } from "./skin/schema";

/** ThemePref 明暗偏好：暗色、亮色，或跟随系统。 */
export type ThemePref = "dark" | "light" | "system";
/** ResolvedTheme 已经落到暗色或亮色，不再含跟随系统。 */
export type ResolvedTheme = "dark" | "light";

/** WINDOW_RGB 原生窗口背景，与气氛层的 --canvas 一致，避免圆角窗外露出一条异色边。 */
export const WINDOW_RGB: Record<ResolvedTheme, readonly [number, number, number]> = {
  dark: [16, 19, 26],
  light: [231, 235, 242],
};

const KEY = "yoyo-theme";
const KEY_DARK = "yoyo-palette-dark";
const KEY_LIGHT = "yoyo-palette-light";

/** parseThemePref 只接受 dark、light、system，其余返回空。 */
export function parseThemePref(v: string | undefined | null): ThemePref | null {
  if (v === "dark" || v === "light" || v === "system") return v;
  return null;
}

/** 系统是否偏好暗色。没有 matchMedia 时视为暗色。 */
function systemDark(): boolean {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? true;
}

/** 跟随系统时看系统偏好，否则偏好本身就是结果。 */
function resolve(pref: ThemePref): ResolvedTheme {
  if (pref === "system") return systemDark() ? "dark" : "light";
  return pref;
}

/**
 * 把明暗类和用户皮肤写到文档根。伴随窗口的 color-scheme 保持 normal。
 * 旧的 data-palette 不再参与外观。
 */
function apply(resolved: ResolvedTheme, skinId: string) {
  const root = document.documentElement;
  root.classList.remove("dark", "light");
  root.classList.add(resolved);
  root.removeAttribute("data-palette");
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

/** 清掉已经不再使用的色板本地记录。 */
function forgetPalettes() {
  try {
    localStorage.removeItem(KEY_DARK);
    localStorage.removeItem(KEY_LIGHT);
  } catch { /* ignore */ }
}

/** SkinOverlay 盖在当前明暗主题上的用户皮肤。id 不是用户皮肤时不会留下。 */
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
  skinId: string;
  skin: SkinOverlay | null;
  setPref: (p: ThemePref) => void;
  setSkinOverlay: (next: SkinOverlay | null, opts?: { persist?: boolean }) => void;
  cycle: () => void;
};

const ThemeCtx = createContext<ThemeCtxValue>({
  pref: "system",
  resolved: "dark",
  skinId: "",
  skin: null,
  setPref: () => {},
  setSkinOverlay: () => {},
  cycle: () => {},
});

/** ThemeProvider 持有明暗和皮肤，并在变化时写到文档和窗口。children 作为子节点渲染。 */
export function ThemeProvider({ children }: { children: ReactNode }) {
  const [pref, setPrefState] = useState<ThemePref>(readPref);
  const [skin, setSkin] = useState<SkinOverlay | null>(null);
  const resolved = useMemo(() => resolve(pref), [pref]);
  const skinId = skin?.id || "";
  if (typeof document !== "undefined") apply(resolved, skinId);

  useEffect(() => {
    apply(resolved, skinId);
    forgetPalettes();
    try {
      localStorage.setItem(KEY, pref);
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
      paintWindowFromTokens(skin.tokens, WINDOW_RGB[resolved]);
    } else {
      clearSkinTokens();
      paintWindowFromTokens(null, WINDOW_RGB[resolved]);
    }
  }, [pref, resolved, skin, skinId]);

  useEffect(() => {
    if (pref !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const on = () => {
      const next = resolve("system");
      apply(next, skinId);
      paintWindowFromTokens(skin?.tokens || null, WINDOW_RGB[next]);
    };
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, [pref, skin, skinId]);

  /** 设置明暗偏好。p 传给 setPrefState。 */
  const setPref = useCallback((p: ThemePref) => {
    setPrefState(p);
  }, []);
  /** 盖上用户皮肤。id 不合格时清掉覆盖层。next 读取 id，为真时才继续。 */
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
      skinId: skinId || readPersistedSkinId(),
      skin,
      setPref,
      setSkinOverlay,
      cycle,
    }),
    [pref, resolved, skin, skinId, setPref, setSkinOverlay, cycle],
  );

  return <ThemeCtx.Provider value={value}>{children}</ThemeCtx.Provider>;
}

/** useTheme 读取当前主题上下文。 */
export function useTheme() {
  return useContext(ThemeCtx);
}
