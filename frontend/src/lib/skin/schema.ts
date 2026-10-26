/**
 * 拥有：皮肤 CSS 变量的规格、材质、字体和已解析皮肤的类型，以及用户皮肤 id 的形状。
 * 不拥有：令牌值是否安全、对比度，以及上屏。
 */

/** TokenKind 令牌值的种类：颜色、长度或阴影。 */
export type TokenKind = "color" | "dimension" | "shadow";
/** TokenGroup 令牌分组：画布、壳层、文字、证据色或反馈色。 */
export type TokenGroup = "canvas" | "chrome" | "text" | "evidence" | "feedback";

/** TokenSpec 一个允许出现的 CSS 变量：名字、种类和分组。 */
export type TokenSpec = {
  name: string;
  kind: TokenKind;
  group: TokenGroup;
};

/** TOKEN_SPECS 皮肤可以覆盖的变量表。 */
export const TOKEN_SPECS: readonly TokenSpec[] = [
  { name: "--background", kind: "color", group: "canvas" },
  { name: "--panel", kind: "color", group: "canvas" },
  { name: "--media-surface", kind: "color", group: "canvas" },
  { name: "--media-foreground", kind: "color", group: "canvas" },
  { name: "--sidebar", kind: "color", group: "chrome" },
  { name: "--card", kind: "color", group: "chrome" },
  { name: "--popover", kind: "color", group: "chrome" },
  { name: "--input-bar", kind: "color", group: "chrome" },
  { name: "--lift", kind: "color", group: "chrome" },
  { name: "--border", kind: "color", group: "chrome" },
  { name: "--glass-bg", kind: "color", group: "chrome" },
  { name: "--glass-stroke", kind: "color", group: "chrome" },
  { name: "--glass-blur", kind: "dimension", group: "chrome" },
  { name: "--shadow-card", kind: "shadow", group: "chrome" },
  { name: "--shadow-composer", kind: "shadow", group: "chrome" },
  { name: "--shadow-popover", kind: "shadow", group: "chrome" },
  { name: "--foreground", kind: "color", group: "text" },
  { name: "--muted", kind: "color", group: "text" },
  { name: "--accent", kind: "color", group: "evidence" },
  { name: "--accent-fg", kind: "color", group: "evidence" },
  { name: "--ring", kind: "color", group: "evidence" },
  { name: "--mark", kind: "color", group: "evidence" },
  { name: "--mark-deep", kind: "color", group: "evidence" },
  { name: "--ctx-chat", kind: "color", group: "evidence" },
  { name: "--ctx-tools", kind: "color", group: "evidence" },
  { name: "--success", kind: "color", group: "feedback" },
  { name: "--warning", kind: "color", group: "feedback" },
  { name: "--danger", kind: "color", group: "feedback" },
  { name: "--ctx-dynamic", kind: "color", group: "feedback" },
  { name: "--ctx-system", kind: "color", group: "text" },
  { name: "--ctx-free", kind: "color", group: "chrome" },
  { name: "--radius", kind: "dimension", group: "chrome" },
  { name: "--radius-lg", kind: "dimension", group: "chrome" },
  { name: "--radius-composer", kind: "dimension", group: "chrome" },
  { name: "--radius-window", kind: "dimension", group: "chrome" },
  { name: "--radius-pane", kind: "dimension", group: "chrome" },
  { name: "--radius-control", kind: "dimension", group: "chrome" },
  { name: "--radius-chip", kind: "dimension", group: "chrome" },
];

/** EVIDENCE_TOKENS 证据色和反馈色。默认上屏时锁住，避免皮肤改掉状态语义。 */
export const EVIDENCE_TOKENS = new Set(
  TOKEN_SPECS.filter((s) => s.group === "evidence" || s.group === "feedback").map((s) => s.name),
);

/** TOKEN_NAMES 允许的变量名。 */
export const TOKEN_NAMES = new Set(TOKEN_SPECS.map((s) => s.name));

/** SPECS_BY_NAME 按变量名查规格。 */
export const SPECS_BY_NAME = new Map(TOKEN_SPECS.map((s) => [s.name, s]));

/** TokenMap CSS 变量名到值。 */
export type TokenMap = Record<string, string>;

/** SkinMaterials 颗粒、玻璃模糊、壁纸铺法和圆角倍率。 */
export type SkinMaterials = {
  grain: number;
  glassBlur?: string;
  wallpaperFit: "cover" | "contain" | "tile";
  wallpaperDim: number;
  radiusScale: number;
};

/** SkinFontFace 皮肤自带的一张字体。role 区分无衬线和等宽。 */
export type SkinFontFace = {
  family: string;
  url: string;
  role: "sans" | "mono" | string;
};

/** SkinInfo 目录里的一条皮肤。dark 与 light 是两套令牌文件，预览图和预览令牌给选择器。 */
export type SkinInfo = {
  id: string;
  name: string;
  author?: string;
  description?: string;
  builtin?: boolean;
  preserveEvidence?: boolean;
  capabilities?: string[];
  hasWallpaper?: boolean;
  hasFonts?: boolean;
  dark?: string;
  light?: string;
  materials?: SkinMaterials;
  wallpaperUrl?: string;
  previewUrl?: string;
  previewDark?: TokenMap;
  previewLight?: TokenMap;
};

/** SkinResolved 按当前明暗解析好、可以上屏的皮肤。windowRgb 给原生窗口背景。 */
export type SkinResolved = {
  id: string;
  name: string;
  builtin?: boolean;
  mode: "dark" | "light";
  preserveEvidence: boolean;
  tokens: TokenMap;
  windowRgb: number[];
  wallpaperUrl?: string;
  wallpaperFile?: string;
  fonts?: SkinFontFace[];
  materials: SkinMaterials;
  contrast: { foreground: number; muted: number; pass: boolean; reason?: string };
};

/** KEY_SKIN 本地存储里的皮肤 id。 */
export const KEY_SKIN = "yoyo-skin";
/** KEY_SKIN_TOKENS 本地存储里的令牌 JSON。 */
export const KEY_SKIN_TOKENS = "yoyo-skin-tokens";
/** KEY_SKIN_WALLPAPER 本地存储里的壁纸文件名。 */
export const KEY_SKIN_WALLPAPER = "yoyo-skin-wallpaper";
/** KEY_SKIN_MATERIALS 本地存储里的材质 JSON。 */
export const KEY_SKIN_MATERIALS = "yoyo-skin-materials";
/** KEY_SKIN_EVIDENCE 本地存储里是否锁住证据色。 */
export const KEY_SKIN_EVIDENCE = "yoyo-skin-evidence";

/** RADIUS_TOKENS 可按倍率缩放的圆角变量及其缺省长度。 */
export const RADIUS_TOKENS: readonly { name: string; fallback: string }[] = [
  { name: "--radius", fallback: "0.5rem" },
  { name: "--radius-lg", fallback: "0.875rem" },
  { name: "--radius-composer", fallback: "1.25rem" },
  { name: "--radius-window", fallback: "1rem" },
  { name: "--radius-pane", fallback: "0.75rem" },
  { name: "--radius-control", fallback: "0.5rem" },
  { name: "--radius-chip", fallback: "0.375rem" },
];

/** isUserSkinId 小写开头、长度符合目录名，且不是 builtin. 前缀，才是用户皮肤。id 看开头，为假时走另一支，传给 test。 */
export function isUserSkinId(id: string | undefined | null): boolean {
  return !!id && /^[a-z][a-z0-9._-]{2,63}$/.test(id) && !id.startsWith("builtin.");
}

/** defaultMaterials 颗粒、铺满、暗化和圆角倍率的缺省材质。 */
export function defaultMaterials(): SkinMaterials {
  return { grain: 0.04, wallpaperFit: "cover", wallpaperDim: 0.45, radiusScale: 1 };
}
