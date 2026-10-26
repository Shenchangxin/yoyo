import { builtinTokens } from "./builtin.ts";
import { hslToHex, relativeLuminance, veilForReadability } from "./contrast.ts";
import type { TokenMap } from "./schema.ts";

/**
 * 拥有：从壁纸采样亮度、算出可读纱，以及按壁纸色相生成一套暗色和亮色令牌。
 * 不拥有：纱和令牌的上屏，以及对比度门槛本身。
 */

/** wallpaperVeil 壁纸的平均相对亮度，加上能让调色板文字盖在上面仍可读的纱。暗色和亮色用各自的背景、前景和次要文字亮度。file 传给 wallpaperLuminance。mode 与 dark 比较是否相等。 */
export async function wallpaperVeil(file: Blob, mode: "dark" | "light"): Promise<number> {
  const lum = await wallpaperLuminance(file);
  if (mode === "dark") return veilForReadability(lum, relativeLuminance(28, 28, 26), relativeLuminance(244, 244, 240), relativeLuminance(140, 139, 133));
  return veilForReadability(lum, relativeLuminance(251, 251, 250), relativeLuminance(28, 28, 26), relativeLuminance(111, 110, 104));
}

/** wallpaperLuminance 把图片缩采样后的不透明像素平均成相对亮度。没有画布或没有可计像素时返回 0.5。file 传给 createImageBitmap。 */
export async function wallpaperLuminance(file: Blob): Promise<number> {
  const bmp = await createImageBitmap(file);
  const size = 48;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) return 0.5;
  ctx.drawImage(bmp, 0, 0, size, size);
  const data = ctx.getImageData(0, 0, size, size).data;
  let sum = 0;
  let n = 0;
  for (let i = 0; i < data.length; i += 4) {
    if (data[i + 3] < 40) continue;
    sum += relativeLuminance(data[i], data[i + 1], data[i + 2]);
    n++;
  }
  return n ? sum / n : 0.5;
}

/** tokensFromImage 按壁纸色相生成暗色和亮色令牌。未要求重着色时，证据色和反馈色留在内置的 ink 与 neutral 上。file 传给 createImageBitmap。opts 读取 recolorEvidence，为空时不读。 */
export async function tokensFromImage(file: Blob, opts?: { recolorEvidence?: boolean }): Promise<{ dark: TokenMap; light: TokenMap; seed: string }> {
  const bmp = await createImageBitmap(file);
  const size = 48;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) throw new Error("no canvas");
  ctx.drawImage(bmp, 0, 0, size, size);
  const data = ctx.getImageData(0, 0, size, size).data;
  let hSum = 0;
  let sSum = 0;
  let wSum = 0;
  for (let i = 0; i < data.length; i += 4) {
    const a = data[i + 3] / 255;
    if (a < 0.4) continue;
    const hsl = rgbToHsl(data[i], data[i + 1], data[i + 2]);
    const w = a * (0.35 + hsl.s);
    hSum += hsl.h * w;
    sSum += hsl.s * w;
    wSum += w;
  }
  const h = wSum ? hSum / wSum : 32;
  const s = Math.min(0.22, Math.max(0.06, wSum ? sSum / wSum : 0.1));
  const dark = tonal("dark", h, s);
  const light = tonal("light", h, s);
  if (!opts?.recolorEvidence) {
    const ink = builtinTokens("ink");
    const neu = builtinTokens("neutral");
    for (const k of ["--accent", "--accent-fg", "--ring", "--mark", "--mark-deep", "--success", "--warning", "--danger", "--ctx-chat", "--ctx-tools", "--ctx-dynamic"] as const) {
      dark[k] = ink[k]!;
      light[k] = neu[k]!;
    }
  }
  return { dark, light, seed: `hsl(${Math.round(h)} ${Math.round(s * 100)}% 50%)` };
}

/** 用色相和饱和度铺一套暗色或亮色令牌。mode 与 dark 比较是否相等。h 四舍五入后，传给 hslToHex。s 乘以 100。 */
function tonal(mode: "dark" | "light", h: number, s: number): TokenMap {
  const hue = Math.round(h);
  const sat = Math.round(s * 100);
  /** hsl 调用 hslToHex。ss 传给 hslToHex。l 传给 hslToHex。 */
  const hsl = (ss: number, l: number) => hslToHex(hue, ss, l);
  if (mode === "dark") {
    const bg = hsl(Math.max(6, sat - 4), 11);
    const fg = hsl(Math.max(4, sat - 6), 96);
    return {
      "--background": bg,
      "--panel": bg,
      "--sidebar": hsl(Math.max(5, sat - 4), 14),
      "--card": hsl(Math.max(5, sat - 3), 16),
      "--popover": hsl(Math.max(5, sat - 3), 18),
      "--input-bar": hsl(Math.max(5, sat - 4), 14),
      "--lift": hsl(Math.max(4, sat - 2), 22),
      "--foreground": fg,
      "--muted": hsl(Math.max(4, sat - 8), 58),
      "--border": `color-mix(in srgb, ${fg} 9%, transparent)`,
      "--accent": hsl(Math.min(70, sat + 40), 62),
      "--accent-fg": hsl(sat, 10),
      "--ring": hsl(Math.min(70, sat + 40), 62),
      "--mark": hsl(18, 62),
      "--mark-deep": hsl(16, 48),
      "--danger": hsl(4, 62),
      "--success": hsl(145, 38),
      "--warning": hsl(42, 52),
      "--media-surface": hsl(Math.max(6, sat - 4), 6),
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
  }
  const bg = hsl(Math.max(4, sat - 8), 97);
  const fg = hsl(Math.max(6, sat - 4), 12);
  return {
    "--background": bg,
    "--panel": bg,
    "--sidebar": hsl(Math.max(4, sat - 8), 94),
    "--card": hsl(Math.max(2, sat - 10), 100),
    "--popover": hsl(Math.max(2, sat - 10), 100),
    "--input-bar": hsl(Math.max(2, sat - 10), 100),
    "--lift": hsl(Math.max(4, sat - 6), 90),
    "--foreground": fg,
    "--muted": hsl(Math.max(4, sat - 8), 42),
    "--border": `color-mix(in srgb, ${fg} 8%, transparent)`,
    "--accent": hsl(Math.min(75, sat + 45), 40),
    "--accent-fg": hsl(sat, 98),
    "--ring": hsl(Math.min(75, sat + 45), 40),
    "--mark": hsl(18, 52),
    "--mark-deep": hsl(16, 42),
    "--danger": hsl(4, 52),
    "--success": hsl(145, 48),
    "--warning": hsl(42, 48),
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
}

/** 把 0 到 255 的 RGB 收成色相、饱和度和亮度。r 减去 b，减去 g，传给 Math.max，传给 Math.min。g 减去 b，小于 b，传给 Math.max，传给 Math.min。b 减去 g，小于 g，传给 Math.max，传给 Math.min。 */
function rgbToHsl(r: number, g: number, b: number): { h: number; s: number; l: number } {
  r /= 255;
  g /= 255;
  b /= 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const l = (max + min) / 2;
  if (max === min) return { h: 0, s: 0, l };
  const d = max - min;
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
  let h = 0;
  switch (max) {
    case r:
      h = (g - b) / d + (g < b ? 6 : 0);
      break;
    case g:
      h = (b - r) / d + 2;
      break;
    default:
      h = (r - g) / d + 4;
  }
  return { h: h * 60, s, l };
}
