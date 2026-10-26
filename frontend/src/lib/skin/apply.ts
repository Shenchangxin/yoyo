/**
 * 拥有：把皮肤令牌、壁纸、字体和材质写到文档根，并在需要时记入本地存储。
 * 不拥有：令牌是否合法、对比度，以及主题偏好本身。
 */

import { isCompanionSurface } from "../popout";
import { windowRGBFromTokens } from "./builtin.ts";
import {
  defaultMaterials,
  EVIDENCE_TOKENS,
  KEY_SKIN,
  KEY_SKIN_EVIDENCE,
  KEY_SKIN_MATERIALS,
  KEY_SKIN_TOKENS,
  KEY_SKIN_WALLPAPER,
  RADIUS_TOKENS,
  isUserSkinId,
  type SkinFontFace,
  type SkinMaterials,
  type TokenMap,
} from "./schema.ts";
import { sanitizeTokenMap } from "./validate.ts";

const applied = new Set<string>();
const FONT_STYLE_ID = "yoyo-skin-fonts";

/** SkinPaint 一次要上屏的皮肤。preserveEvidence 为假时才允许改证据色和反馈色。persist 为假时不写入本地。 */
export type SkinPaint = {
  id: string;
  tokens: TokenMap | null;
  materials?: SkinMaterials | null;
  wallpaperUrl?: string;
  wallpaperFile?: string;
  fonts?: SkinFontFace[];
  preserveEvidence?: boolean;
  persist?: boolean;
};

/** 文档根。没有 document 时返回空。 */
function rootEl(): HTMLElement | null {
  return typeof document === "undefined" ? null : document.documentElement;
}

/** clearSkinTokens 去掉本次以及上次写上的皮肤变量、壁纸、字体和 data-skin。 */
export function clearSkinTokens() {
  const root = rootEl();
  if (!root) return;
  for (const k of applied) root.style.removeProperty(k);
  applied.clear();
  root.removeAttribute("data-skin");
  root.removeAttribute("data-skin-wallpaper");
  root.style.removeProperty("--skin-wallpaper");
  root.style.removeProperty("--skin-wallpaper-fit");
  root.style.removeProperty("--skin-wallpaper-repeat");
  root.style.removeProperty("--skin-wallpaper-dim");
  root.style.removeProperty("--skin-grain");
  const tag = document.getElementById(FONT_STYLE_ID);
  tag?.remove();
  root.style.removeProperty("--font-sans");
  root.style.removeProperty("--font-mono");
}

/** applySkinPaint 先清掉旧皮肤，再写入令牌、颗粒、玻璃、壁纸、圆角和字体。伴随窗口不上壁纸。证据色默认锁住。 */
export function applySkinPaint(paint: SkinPaint) {
  const root = rootEl();
  if (!root) return;
  clearSkinTokens();
  const companion = isCompanionSurface();
  const tokens = sanitizeTokenMap(paint.tokens);
  const locked = paint.preserveEvidence !== false;
  for (const [k, v] of Object.entries(tokens)) {
    if (locked && EVIDENCE_TOKENS.has(k)) continue;
    root.style.setProperty(k, v);
    applied.add(k);
  }
  if (isUserSkinId(paint.id)) {
    root.setAttribute("data-skin", paint.id);
  }
  const mat = paint.materials || defaultMaterials();
  root.style.setProperty("--skin-grain", String(mat.grain ?? 0.04));
  applied.add("--skin-grain");
  if (mat.glassBlur) {
    root.style.setProperty("--glass-blur", mat.glassBlur);
    applied.add("--glass-blur");
  }
  if (!companion && paint.wallpaperUrl && /^\/yoyo-skin\/[a-z][a-z0-9._-]{2,63}\/assets\/wallpaper\.(png|jpe?g|webp)$/i.test(paint.wallpaperUrl)) {
    const fit = mat.wallpaperFit === "contain" ? "contain" : mat.wallpaperFit === "tile" ? "auto" : "cover";
    const repeat = mat.wallpaperFit === "tile" ? "repeat" : "no-repeat";
    root.style.setProperty("--skin-wallpaper", `url("${paint.wallpaperUrl}")`);
    root.style.setProperty("--skin-wallpaper-fit", fit);
    root.style.setProperty("--skin-wallpaper-repeat", repeat);
    root.style.setProperty("--skin-wallpaper-dim", String(mat.wallpaperDim ?? 0.45));
    root.setAttribute("data-skin-wallpaper", "1");
    applied.add("--skin-wallpaper");
    applied.add("--skin-wallpaper-fit");
    applied.add("--skin-wallpaper-repeat");
    applied.add("--skin-wallpaper-dim");
  }
  applyRadiusScale(tokens, mat.radiusScale);
  applyFonts(paint.fonts);
  if (paint.persist !== false) persistPaint(paint, tokens);
}

/** 把带 px、rem 或 em 的长度乘上倍率。对不上单位时返回空。 */
function scaleDim(v: string, scale: number): string | null {
  const m = String(v || "").trim().match(/^([0-9]*\.?[0-9]+)(px|rem|em)$/);
  if (!m) return null;
  return `${Number(m[1]) * scale}${m[2]}`;
}

/** 按材质倍率改写圆角令牌。倍率为 1 或无效时不动。 */
function applyRadiusScale(tokens: TokenMap, scale: number) {
  const root = rootEl();
  if (!root || !scale || scale === 1) return;
  for (const spec of RADIUS_TOKENS) {
    const src = tokens[spec.name] || spec.fallback;
    const scaled = scaleDim(src, scale);
    if (!scaled) continue;
    root.style.setProperty(spec.name, scaled);
    applied.add(spec.name);
  }
}

/** 只接受皮肤目录里的 YoyoSkinSans 与 YoyoSkinMono，并写成字体面。 */
function applyFonts(fonts: SkinFontFace[] | undefined) {
  const root = rootEl();
  if (!root) return;
  const safe = (fonts || []).filter((f) => {
    if (!f?.family || !f.url) return false;
    if (!/^YoyoSkin(Sans|Mono)$/.test(f.family)) return false;
    return /^\/yoyo-skin\/[a-z][a-z0-9._-]{2,63}\/assets\/fonts\/(?:sans|mono)\.woff2$/.test(f.url);
  });
  if (!safe.length) return;
  let tag = document.getElementById(FONT_STYLE_ID) as HTMLStyleElement | null;
  if (!tag) {
    tag = document.createElement("style");
    tag.id = FONT_STYLE_ID;
    document.head.appendChild(tag);
  }
  tag.textContent = safe
    .map((f) => `@font-face{font-family:"${f.family}";src:url("${f.url}") format("woff2");font-display:swap;}`)
    .join("");
  const sans = safe.find((f) => f.role === "sans");
  const mono = safe.find((f) => f.role === "mono");
  if (sans) {
    root.style.setProperty("--font-sans", `"${sans.family}", "Noto Sans SC", "PingFang SC", "Microsoft YaHei UI", ui-sans-serif, system-ui, sans-serif`);
    applied.add("--font-sans");
  }
  if (mono) {
    root.style.setProperty("--font-mono", `"${mono.family}", ui-monospace, "SF Mono", "Cascadia Code", Consolas, monospace`);
    applied.add("--font-mono");
  }
}

/** 用户皮肤把令牌、壁纸文件和材质记入本地。证据色锁住时不存那些键。内置或空令牌则清掉这些键。 */
function persistPaint(paint: SkinPaint, tokens: TokenMap) {
  try {
    if (isUserSkinId(paint.id) && Object.keys(tokens).length) {
      const locked = paint.preserveEvidence !== false;
      const stored = { ...tokens };
      if (locked) {
        for (const k of EVIDENCE_TOKENS) delete stored[k];
      }
      localStorage.setItem(KEY_SKIN, paint.id);
      localStorage.setItem(KEY_SKIN_TOKENS, JSON.stringify(stored));
      localStorage.setItem(KEY_SKIN_WALLPAPER, paint.wallpaperFile || "");
      localStorage.setItem(KEY_SKIN_MATERIALS, JSON.stringify(paint.materials || defaultMaterials()));
      localStorage.setItem(KEY_SKIN_EVIDENCE, locked ? "1" : "0");
    } else {
      localStorage.removeItem(KEY_SKIN);
      localStorage.removeItem(KEY_SKIN_TOKENS);
      localStorage.removeItem(KEY_SKIN_WALLPAPER);
      localStorage.removeItem(KEY_SKIN_MATERIALS);
      localStorage.removeItem(KEY_SKIN_EVIDENCE);
    }
  } catch {
    /* 配额或隐私模式写入失败时忽略 */
  }
}

/** paintWindowFromTokens 把窗口背景设成令牌里的背景色。伴随窗口保持全透明。没有令牌时用调用方给的回退色。 */
export function paintWindowFromTokens(tokens: TokenMap | null, fallbackRGB: readonly [number, number, number]) {
  import("@wailsio/runtime")
    .then(({ Window }) => {
      if (isCompanionSurface()) {
        Window.SetBackgroundColour(0, 0, 0, 0);
        return;
      }
      const fromTokens = tokens ? windowRGBFromTokens(tokens) : null;
      const rgb = fromTokens || fallbackRGB;
      Window.SetBackgroundColour(rgb[0], rgb[1], rgb[2], 255);
    })
    .catch(() => {});
}

/** readPersistedSkinId 读出本地记下的用户皮肤 id。不是用户皮肤或读失败时返回空串。 */
export function readPersistedSkinId(): string {
  try {
    const id = localStorage.getItem(KEY_SKIN) || "";
    return isUserSkinId(id) ? id : "";
  } catch {
    return "";
  }
}
