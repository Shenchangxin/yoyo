import type { TokenMap } from "./schema.ts";

/**
 * 拥有：六套内置调色板的令牌，以及从背景色解析窗口 RGB。
 * 不拥有：把令牌写到文档，以及用户皮肤的校验。
 */

/** BuiltinPaletteId 内置调色板：三套暗色和三套亮色。 */
export type BuiltinPaletteId = "ink" | "dim" | "slate" | "neutral" | "paper" | "mist";

const ink: TokenMap = {
  "--background": "#13151a",
  "--foreground": "#f2f4f8",
  "--sidebar": "#0d0f13",
  "--panel": "#13151a",
  "--card": "#1a1d24",
  "--popover": "#1f232c",
  "--input-bar": "#1b1f28",
  "--lift": "#242934",
  "--border": "color-mix(in srgb, #f2f4f8 8%, transparent)",
  "--muted": "#8e95a5",
  "--accent": "#6e85ff",
  "--accent-fg": "#0d0f13",
  "--mark": "#f2965c",
  "--mark-deep": "#d47b44",
  "--danger": "#f27474",
  "--success": "#3dd68c",
  "--warning": "#f5b851",
  "--ring": "#6e85ff",
  "--media-surface": "#090b0f",
  "--media-foreground": "#f2f4f8",
  "--glass-blur": "20px",
  "--glass-bg": "color-mix(in srgb, var(--sidebar) 76%, transparent)",
  "--glass-stroke": "color-mix(in srgb, var(--foreground) 8%, transparent)",
  "--radius": "0.5rem",
  "--radius-lg": "0.75rem",
  "--radius-composer": "1.125rem",
  "--radius-window": "0.75rem",
  "--radius-pane": "0.75rem",
  "--radius-control": "0.5rem",
  "--radius-chip": "0.375rem",
  "--shadow-card": "inset 0 0.5px 0 color-mix(in srgb, white 6%, transparent)",
  "--shadow-composer": "0 14px 40px color-mix(in srgb, black 34%, transparent), inset 0 0.5px 0 color-mix(in srgb, white 7%, transparent)",
  "--shadow-popover": "0 20px 52px color-mix(in srgb, black 48%, transparent), 0 0 0 1px color-mix(in srgb, white 7%, transparent)",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 38%, #8e95a5)",
  "--ctx-tools": "color-mix(in srgb, var(--accent) 72%, var(--muted))",
  "--ctx-dynamic": "var(--warning)",
  "--ctx-chat": "var(--accent)",
  "--ctx-free": "color-mix(in srgb, var(--muted) 18%, transparent)",
};

const dimOver: TokenMap = {
  "--background": "#202226",
  "--foreground": "#f3f4f6",
  "--sidebar": "#181a1e",
  "--panel": "#202226",
  "--card": "#282b30",
  "--popover": "#30343b",
  "--input-bar": "#26292f",
  "--lift": "#343840",
  "--border": "color-mix(in srgb, white 8%, transparent)",
  "--muted": "#969ca8",
};

const slateOver: TokenMap = {
  "--background": "#121213",
  "--foreground": "#f5f5f7",
  "--sidebar": "#0b0b0c",
  "--panel": "#121213",
  "--card": "#1b1b1d",
  "--popover": "#232326",
  "--input-bar": "#19191b",
  "--lift": "#27272a",
  "--border": "color-mix(in srgb, white 8%, transparent)",
  "--muted": "#8f9096",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 32%, #8a9098)",
};

const neutral: TokenMap = {
  "--background": "#ffffff",
  "--foreground": "#111318",
  "--sidebar": "#f3f4f6",
  "--panel": "#ffffff",
  "--card": "#ffffff",
  "--popover": "#ffffff",
  "--input-bar": "#ffffff",
  "--lift": "#e8ebef",
  "--border": "color-mix(in srgb, #111318 8%, transparent)",
  "--muted": "#646b78",
  "--accent": "#3b5bff",
  "--accent-fg": "#f8faff",
  "--mark": "#d25e2c",
  "--mark-deep": "#a3461e",
  "--danger": "#d93842",
  "--success": "#108956",
  "--warning": "#c87a0a",
  "--ring": "#3b5bff",
  "--media-surface": "#0b0d12",
  "--media-foreground": "#f2f4f8",
  "--glass-blur": "20px",
  "--glass-bg": "color-mix(in srgb, var(--sidebar) 78%, transparent)",
  "--glass-stroke": "color-mix(in srgb, var(--foreground) 7%, transparent)",
  "--radius": "0.5rem",
  "--radius-lg": "0.75rem",
  "--radius-composer": "1.125rem",
  "--radius-window": "0.75rem",
  "--radius-pane": "0.75rem",
  "--radius-control": "0.5rem",
  "--radius-chip": "0.375rem",
  "--shadow-card": "0 1px 2px color-mix(in srgb, #111318 4%, transparent)",
  "--shadow-composer": "0 14px 38px color-mix(in srgb, #111318 8%, transparent)",
  "--shadow-popover": "0 20px 48px color-mix(in srgb, #111318 11%, transparent), 0 0 0 1px color-mix(in srgb, #111318 6%, transparent)",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 22%, #646b78)",
  "--ctx-tools": "color-mix(in srgb, var(--accent) 55%, var(--muted))",
  "--ctx-dynamic": "var(--warning)",
  "--ctx-chat": "var(--accent)",
  "--ctx-free": "color-mix(in srgb, var(--muted) 16%, transparent)",
};

const paperOver: TokenMap = {
  "--background": "#f6f3ec",
  "--foreground": "#1a1916",
  "--sidebar": "#eeeae2",
  "--panel": "#f6f3ec",
  "--card": "#fffdf8",
  "--popover": "#fffdf8",
  "--input-bar": "#fffdf8",
  "--lift": "#e7e2d8",
  "--muted": "#6d6a62",
};

const mistOver: TokenMap = {
  "--background": "#f6f8fa",
  "--foreground": "#1b1f24",
  "--sidebar": "#eef1f4",
  "--panel": "#f6f8fa",
  "--card": "#ffffff",
  "--popover": "#ffffff",
  "--input-bar": "#ffffff",
  "--lift": "#e6eaee",
  "--border": "color-mix(in srgb, #1b1f24 7%, transparent)",
  "--muted": "#6a7178",
  "--shadow-composer": "0 12px 36px color-mix(in srgb, #1b1f24 6%, transparent)",
  "--shadow-popover": "0 18px 44px color-mix(in srgb, #1b1f24 9%, transparent), 0 0 0 1px color-mix(in srgb, #1b1f24 5%, transparent)",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 22%, #7a8aa0)",
};

/** 以一套完整调色为底，盖上少量覆盖项。base 展开传递。over 展开传递。 */
function merge(base: TokenMap, over: TokenMap): TokenMap {
  return { ...base, ...over };
}

/** BUILTIN_PALETTES 六套调色板。dim 与 slate 盖在 ink 上，paper 与 mist 盖在 neutral 上。 */
export const BUILTIN_PALETTES: Record<BuiltinPaletteId, TokenMap> = {
  ink,
  dim: merge(ink, dimOver),
  slate: merge(ink, slateOver),
  neutral,
  paper: merge(neutral, paperOver),
  mist: merge(neutral, mistOver),
};

/** builtinTokens 拷贝一套内置调色，避免调用方改到共享表。id 用作下标。 */
export function builtinTokens(id: BuiltinPaletteId): TokenMap {
  return { ...BUILTIN_PALETTES[id] };
}

/** windowRGBFromTokens 从 --sidebar（没有时回落 --background）的十六进制色取出窗口 RGB。不是 # 开头或长度不够时返回空。tokens 按下标读取。 */
export function windowRGBFromTokens(tokens: TokenMap): [number, number, number] | null {
  const hex = tokens["--sidebar"] || tokens["--background"];
  if (!hex || !hex.startsWith("#")) return null;
  const h = hex.slice(1);
  if (h.length === 3) {
    return [parseInt(h[0] + h[0], 16), parseInt(h[1] + h[1], 16), parseInt(h[2] + h[2], 16)];
  }
  if (h.length >= 6) {
    return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
  }
  return null;
}
