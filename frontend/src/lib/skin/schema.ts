export type TokenKind = "color" | "dimension" | "shadow";
export type TokenGroup = "canvas" | "chrome" | "text" | "evidence" | "feedback";

export type TokenSpec = {
  name: string;
  kind: TokenKind;
  group: TokenGroup;
};

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

export const EVIDENCE_TOKENS = new Set(
  TOKEN_SPECS.filter((s) => s.group === "evidence" || s.group === "feedback").map((s) => s.name),
);

export const TOKEN_NAMES = new Set(TOKEN_SPECS.map((s) => s.name));

export const SPECS_BY_NAME = new Map(TOKEN_SPECS.map((s) => [s.name, s]));

export type TokenMap = Record<string, string>;

export type SkinMaterials = {
  grain: number;
  glassBlur?: string;
  wallpaperFit: "cover" | "contain" | "tile";
  wallpaperDim: number;
  radiusScale: number;
};

export type SkinFontFace = {
  family: string;
  url: string;
  role: "sans" | "mono" | string;
};

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

export const KEY_SKIN = "yoyo-skin";
export const KEY_SKIN_TOKENS = "yoyo-skin-tokens";
export const KEY_SKIN_WALLPAPER = "yoyo-skin-wallpaper";
export const KEY_SKIN_MATERIALS = "yoyo-skin-materials";
export const KEY_SKIN_EVIDENCE = "yoyo-skin-evidence";

export const RADIUS_TOKENS: readonly { name: string; fallback: string }[] = [
  { name: "--radius", fallback: "0.5rem" },
  { name: "--radius-lg", fallback: "0.875rem" },
  { name: "--radius-composer", fallback: "1.25rem" },
  { name: "--radius-window", fallback: "1rem" },
  { name: "--radius-pane", fallback: "0.75rem" },
  { name: "--radius-control", fallback: "0.5rem" },
  { name: "--radius-chip", fallback: "0.375rem" },
];

export function isUserSkinId(id: string | undefined | null): boolean {
  return !!id && /^[a-z][a-z0-9._-]{2,63}$/.test(id) && !id.startsWith("builtin.");
}

export function defaultMaterials(): SkinMaterials {
  return { grain: 0.04, wallpaperFit: "cover", wallpaperDim: 0.45, radiusScale: 1 };
}
