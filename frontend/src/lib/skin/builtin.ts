import type { TokenMap } from "./schema.ts";

/**
 * 拥有：六套内置调色板的令牌，以及从背景色解析窗口 RGB。
 * 不拥有：把令牌写到文档，以及用户皮肤的校验。
 */

/** BuiltinPaletteId 内置调色板：三套暗色和三套亮色。 */
export type BuiltinPaletteId = "ink" | "dim" | "slate" | "neutral" | "paper" | "mist";

const ink: TokenMap = {
  "--background": "#12141c",
  "--foreground": "#f6f3ec",
  "--sidebar": "#191d2b",
  "--panel": "#12141c",
  "--card": "#232a3d",
  "--popover": "#2c3450",
  "--input-bar": "#1a1f30",
  "--lift": "#353d58",
  "--border": "color-mix(in srgb, #f6f3ec 10%, transparent)",
  "--muted": "#b0a89c",
  "--accent": "#8096ff",
  "--accent-fg": "#12141c",
  "--mark": "#f0a06a",
  "--mark-deep": "#d4834a",
  "--danger": "#f09590",
  "--success": "#6ed4a4",
  "--warning": "#f0c36a",
  "--ring": "#8096ff",
  "--media-surface": "#0c0e14",
  "--media-foreground": "#f6f3ec",
  "--glass-blur": "20px",
  "--glass-bg": "color-mix(in srgb, var(--sidebar) 72%, transparent)",
  "--glass-stroke": "color-mix(in srgb, var(--foreground) 8%, transparent)",
  "--radius": "0.5rem",
  "--radius-lg": "0.875rem",
  "--radius-composer": "1.25rem",
  "--radius-window": "1rem",
  "--radius-pane": "0.75rem",
  "--radius-control": "0.5rem",
  "--radius-chip": "0.375rem",
  "--shadow-card": "none",
  "--shadow-composer": "0 12px 36px color-mix(in srgb, black 28%, transparent)",
  "--shadow-popover": "0 18px 48px color-mix(in srgb, black 42%, transparent), 0 0 0 1px color-mix(in srgb, white 6%, transparent)",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 38%, #9a958c)",
  "--ctx-tools": "color-mix(in srgb, var(--accent) 72%, var(--muted))",
  "--ctx-dynamic": "var(--warning)",
  "--ctx-chat": "var(--accent)",
  "--ctx-free": "color-mix(in srgb, var(--muted) 18%, transparent)",
};

const dimOver: TokenMap = {
  "--background": "#242422",
  "--foreground": "#f5f5f1",
  "--sidebar": "#2a2a27",
  "--panel": "#242422",
  "--card": "#2e2e2b",
  "--popover": "#363632",
  "--input-bar": "#2c2c29",
  "--lift": "#3c3c37",
  "--border": "color-mix(in srgb, white 9%, transparent)",
  "--muted": "#999891",
};

const slateOver: TokenMap = {
  "--background": "#121212",
  "--foreground": "#f5f5f5",
  "--sidebar": "#1a1a1a",
  "--panel": "#121212",
  "--card": "#1e1e1e",
  "--popover": "#262626",
  "--input-bar": "#1c1c1c",
  "--lift": "#2a2a2a",
  "--border": "color-mix(in srgb, white 9%, transparent)",
  "--muted": "#8e8e8e",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 32%, #8a9098)",
};

const neutral: TokenMap = {
  "--background": "#f7f4ee",
  "--foreground": "#1c1917",
  "--sidebar": "#efe9df",
  "--panel": "#f7f4ee",
  "--card": "#ffffff",
  "--popover": "#ffffff",
  "--input-bar": "#ffffff",
  "--lift": "#e7e0d4",
  "--border": "color-mix(in srgb, #1c1917 9%, transparent)",
  "--muted": "#6b645b",
  "--accent": "#3a4ff0",
  "--accent-fg": "#f6f7ff",
  "--mark": "#d25e2c",
  "--mark-deep": "#a3461e",
  "--danger": "#d23b42",
  "--success": "#0e8a5b",
  "--warning": "#c47a10",
  "--ring": "#3a4ff0",
  "--media-surface": "#0c0c0b",
  "--media-foreground": "#f4f4f0",
  "--glass-blur": "20px",
  "--glass-bg": "color-mix(in srgb, var(--sidebar) 72%, transparent)",
  "--glass-stroke": "color-mix(in srgb, var(--foreground) 8%, transparent)",
  "--radius": "0.5rem",
  "--radius-lg": "0.875rem",
  "--radius-composer": "1.25rem",
  "--radius-window": "1rem",
  "--radius-pane": "0.75rem",
  "--radius-control": "0.5rem",
  "--radius-chip": "0.375rem",
  "--shadow-card": "none",
  "--shadow-composer": "0 12px 36px color-mix(in srgb, #1c1c1a 7%, transparent)",
  "--shadow-popover": "0 18px 44px color-mix(in srgb, #1c1c1a 10%, transparent), 0 0 0 1px color-mix(in srgb, #1c1c1a 5%, transparent)",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 22%, #9a958c)",
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

/** windowRGBFromTokens 从 --background 的十六进制色取出窗口 RGB。不是 # 开头或长度不够时返回空。tokens 按下标读取。 */
export function windowRGBFromTokens(tokens: TokenMap): [number, number, number] | null {
  const hex = tokens["--background"];
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
