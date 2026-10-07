import type { TokenMap } from "./schema.ts";

export type BuiltinPaletteId = "ink" | "dim" | "slate" | "neutral" | "paper" | "mist";

const ink: TokenMap = {
  "--background": "#1c1c1a",
  "--foreground": "#f4f4f0",
  "--sidebar": "#222220",
  "--panel": "#1c1c1a",
  "--card": "#242422",
  "--popover": "#2c2c29",
  "--input-bar": "#222220",
  "--lift": "#32322e",
  "--border": "color-mix(in srgb, #f4f4f0 9%, transparent)",
  "--muted": "#8c8b85",
  "--accent": "#5c99d6",
  "--accent-fg": "#0c1924",
  "--mark": "#d4896a",
  "--mark-deep": "#b56545",
  "--danger": "#e06b66",
  "--success": "#74b48a",
  "--warning": "#d4b06a",
  "--ring": "#5c99d6",
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
  "--shadow-card": "inset 0 1px 0 color-mix(in srgb, white 6%, transparent)",
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
  "--background": "#fbfbfa",
  "--foreground": "#1c1c1a",
  "--sidebar": "#f1f1ee",
  "--panel": "#fbfbfa",
  "--card": "#ffffff",
  "--popover": "#ffffff",
  "--input-bar": "#ffffff",
  "--lift": "#e8e8e4",
  "--border": "color-mix(in srgb, #1c1c1a 8%, transparent)",
  "--muted": "#6f6e68",
  "--accent": "#0169cc",
  "--accent-fg": "#f5faff",
  "--mark": "#c2663d",
  "--mark-deep": "#9a4e2e",
  "--danger": "#c4453e",
  "--success": "#1f7a45",
  "--warning": "#a07a2a",
  "--ring": "#0169cc",
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
  "--shadow-card": "0 1px 0 color-mix(in srgb, #1c1c1a 4%, transparent)",
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

function merge(base: TokenMap, over: TokenMap): TokenMap {
  return { ...base, ...over };
}

export const BUILTIN_PALETTES: Record<BuiltinPaletteId, TokenMap> = {
  ink,
  dim: merge(ink, dimOver),
  slate: merge(ink, slateOver),
  neutral,
  paper: merge(neutral, paperOver),
  mist: merge(neutral, mistOver),
};

export function builtinTokens(id: BuiltinPaletteId): TokenMap {
  return { ...BUILTIN_PALETTES[id] };
}

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
