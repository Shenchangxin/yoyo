import type { TokenMap } from "./schema.ts";

/**
 * 拥有：深色与浅色两套内置令牌，以及从背景色解析窗口 RGB。
 * 不拥有：把令牌写到文档，以及用户皮肤的校验。
 * dim、slate 与 ink 相同，paper、mist 与 neutral 相同，旧 id 仍能解析。
 */

/** BuiltinPaletteId 仍接受旧的六套名字，视觉上只剩深色和浅色。 */
export type BuiltinPaletteId = "ink" | "dim" | "slate" | "neutral" | "paper" | "mist";

const ink: TokenMap = {
  "--background": "#161922",
  "--foreground": "#f4f5f7",
  "--sidebar": "#0c0e14",
  "--panel": "#161922",
  "--card": "rgba(255, 255, 255, 0.075)",
  "--popover": "rgba(28, 32, 42, 0.88)",
  "--input-bar": "rgba(22, 26, 34, 0.72)",
  "--lift": "rgba(255, 255, 255, 0.06)",
  "--border": "rgba(255, 255, 255, 0.08)",
  "--muted": "#a3a9b5",
  "--accent": "#8b9cff",
  "--accent-fg": "#12141c",
  "--mark": "#f2965c",
  "--mark-deep": "#d47b44",
  "--danger": "#f27474",
  "--success": "#3dd68c",
  "--warning": "#f5b851",
  "--ring": "#8b9cff",
  "--media-surface": "#090b10",
  "--media-foreground": "#f4f5f7",
  "--glass-blur": "22px",
  "--glass-bg": "rgba(22, 26, 34, 0.62)",
  "--glass-stroke": "rgba(255, 255, 255, 0.14)",
  "--radius": "0.625rem",
  "--radius-lg": "1rem",
  "--radius-composer": "1.625rem",
  "--radius-window": "1rem",
  "--radius-pane": "1.5rem",
  "--radius-control": "0.625rem",
  "--radius-chip": "999px",
  "--shadow-card": "0 14px 36px rgba(0, 0, 0, 0.32), inset 0 1px 0 rgba(255, 255, 255, 0.10)",
  "--shadow-composer": "0 16px 40px rgba(0, 0, 0, 0.38), inset 0 1px 0 rgba(255, 255, 255, 0.12)",
  "--shadow-popover": "0 24px 60px rgba(0, 0, 0, 0.42), inset 0 1px 0 rgba(255, 255, 255, 0.12)",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 38%, #a3a9b5)",
  "--ctx-tools": "color-mix(in srgb, var(--accent) 72%, var(--muted))",
  "--ctx-dynamic": "var(--warning)",
  "--ctx-chat": "var(--accent)",
  "--ctx-free": "color-mix(in srgb, var(--muted) 18%, transparent)",
};

const neutral: TokenMap = {
  "--background": "#f7f8fb",
  "--foreground": "#1c1f26",
  "--sidebar": "#e3e7ee",
  "--panel": "#f7f8fb",
  "--card": "rgba(255, 255, 255, 0.94)",
  "--popover": "rgba(255, 255, 255, 0.92)",
  "--input-bar": "rgba(255, 255, 255, 0.78)",
  "--lift": "rgba(28, 31, 38, 0.045)",
  "--border": "rgba(28, 31, 38, 0.08)",
  "--muted": "#5c6472",
  "--accent": "#3558f5",
  "--accent-fg": "#f7f8ff",
  "--mark": "#c2512a",
  "--mark-deep": "#9a3e20",
  "--danger": "#c4313b",
  "--success": "#0e7a4b",
  "--warning": "#a16207",
  "--ring": "#3558f5",
  "--media-surface": "#12141a",
  "--media-foreground": "#f4f5f7",
  "--glass-blur": "22px",
  "--glass-bg": "rgba(255, 255, 255, 0.74)",
  "--glass-stroke": "rgba(255, 255, 255, 0.78)",
  "--radius": "0.625rem",
  "--radius-lg": "1rem",
  "--radius-composer": "1.625rem",
  "--radius-window": "1rem",
  "--radius-pane": "1.5rem",
  "--radius-control": "0.625rem",
  "--radius-chip": "999px",
  "--shadow-card": "0 10px 28px rgba(36, 44, 68, 0.07), inset 0 1px 0 rgba(255, 255, 255, 0.85)",
  "--shadow-composer": "0 16px 40px rgba(36, 44, 68, 0.10), inset 0 1px 0 rgba(255, 255, 255, 0.9)",
  "--shadow-popover": "0 24px 60px rgba(28, 34, 52, 0.14), inset 0 1px 0 rgba(255, 255, 255, 0.9)",
  "--ctx-system": "color-mix(in srgb, var(--foreground) 22%, #5c6472)",
  "--ctx-tools": "color-mix(in srgb, var(--accent) 55%, var(--muted))",
  "--ctx-dynamic": "var(--warning)",
  "--ctx-chat": "var(--accent)",
  "--ctx-free": "color-mix(in srgb, var(--muted) 16%, transparent)",
};

/** BUILTIN_PALETTES 旧 id 折叠到深色或浅色。 */
export const BUILTIN_PALETTES: Record<BuiltinPaletteId, TokenMap> = {
  ink,
  dim: ink,
  slate: ink,
  neutral,
  paper: neutral,
  mist: neutral,
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
