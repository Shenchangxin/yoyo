import type { TokenMap } from "./schema.ts";

/**
 * 拥有：前景和次要文字相对背景的对比度，以及壁纸上要加多厚的纱才能读清字。
 * 不拥有：把纱写到文档，以及从图片采样亮度。
 */

/** ContrastReport 前景和次要文字的对比度。pass 为假时 reason 说明哪一条没过。 */
export type ContrastReport = {
  foreground: number;
  muted: number;
  pass: boolean;
  reason?: string;
};

/** 解析 #rgb 或 #rrggbb，通道换成 0 到 1。其他写法返回空。s 看开头，截取，为假时走另一支。 */
function parseRGB(s: string | undefined): [number, number, number] | null {
  if (!s || !s.startsWith("#")) return null;
  const h = s.slice(1);
  /** n 调用 parseInt。a 传给 parseInt。 */
  const n = (a: string) => parseInt(a, 16) / 255;
  if (h.length === 3) return [n(h[0] + h[0]), n(h[1] + h[1]), n(h[2] + h[2])];
  if (h.length >= 6) return [n(h.slice(0, 2)), n(h.slice(2, 4)), n(h.slice(4, 6))];
  return null;
}

/** sRGB 通道转线性光。v 小于等于 0.04045，除以 12.92。 */
function lin(v: number): number {
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

/** 相对亮度。c 取第一段，按下标读取，传给 lin。 */
function lum(c: [number, number, number]): number {
  return 0.2126 * lin(c[0]) + 0.7152 * lin(c[1]) + 0.0722 * lin(c[2]);
}

/** 两个颜色的对比度。a 传给 lum。b 传给 lum。 */
function ratio(a: [number, number, number], b: [number, number, number]): number {
  const l1 = lum(a);
  const l2 = lum(b);
  const hi = Math.max(l1, l2);
  const lo = Math.min(l1, l2);
  return (hi + 0.05) / (lo + 0.05);
}

/** hslToHex 把色相和百分比饱和度、亮度收成 #rrggbb。h 取余 360。sPct 除以 100。lPct 除以 100。 */
export function hslToHex(h: number, sPct: number, lPct: number): string {
  const s = Math.min(1, Math.max(0, sPct / 100));
  const l = Math.min(1, Math.max(0, lPct / 100));
  const c = (1 - Math.abs(2 * l - 1)) * s;
  const hp = (((h % 360) + 360) % 360) / 60;
  const x = c * (1 - Math.abs((hp % 2) - 1));
  let r = 0;
  let g = 0;
  let b = 0;
  if (hp < 1) [r, g, b] = [c, x, 0];
  else if (hp < 2) [r, g, b] = [x, c, 0];
  else if (hp < 3) [r, g, b] = [0, c, x];
  else if (hp < 4) [r, g, b] = [0, x, c];
  else if (hp < 5) [r, g, b] = [x, 0, c];
  else [r, g, b] = [c, 0, x];
  const m = l - c / 2;
  /** hex 调用 padStart。v 加上 m。 */
  const hex = (v: number) => Math.round((v + m) * 255).toString(16).padStart(2, "0");
  return `#${hex(r)}${hex(g)}${hex(b)}`;
}

/** 两个相对亮度的对比度。a 传给 Math.max，传给 Math.min。b 传给 Math.max，传给 Math.min。 */
function contrastLum(a: number, b: number): number {
  const hi = Math.max(a, b);
  const lo = Math.min(a, b);
  return (hi + 0.05) / (lo + 0.05);
}

/**
 * veilForReadability 最小纱厚度。0 是裸壁纸，1 是纯调色板。前景至少 4.5，次要文字至少 3。自动结果封顶，亮图在暗色里仍能看见，滑条仍可再加厚。
 * wallLum 乘以 (1 - v)，传给 contrastLum。
 * bgLum 乘以 v。
 * fgLum 传给 needed。
 * mutedLum 与 null 比较是否不等，传给 needed。
 */
export function veilForReadability(wallLum: number, bgLum: number, fgLum: number, mutedLum?: number): number {
  /** needed 调用 contrastLum。ink 传给 contrastLum。minRatio 大于等于 contrastLum(ink, wallLum)，大于等于 contrastLum(ink, mix)。 */
  const needed = (ink: number, minRatio: number) => {
    if (contrastLum(ink, wallLum) >= minRatio) return 0;
    let lo = 0;
    let hi = 1;
    let best = 1;
    for (let i = 0; i < 20; i++) {
      const v = (lo + hi) / 2;
      const mix = v * bgLum + (1 - v) * wallLum;
      if (contrastLum(ink, mix) >= minRatio) {
        best = v;
        hi = v;
      } else {
        lo = v;
      }
    }
    return best;
  };
  let v = needed(fgLum, 4.5);
  if (mutedLum != null) v = Math.max(v, needed(mutedLum, 3));
  // 自动纱封顶，亮图在暗色里仍能看见；滑条仍可再加厚。
  return Math.min(0.58, Math.max(0, v));
}

/** relativeLuminance 把 0 到 255 的通道换成相对亮度。r 除以 255。g 除以 255。b 除以 255。 */
export function relativeLuminance(r: number, g: number, b: number): number {
  return lum([r / 255, g / 255, b / 255]);
}

/** contrastOf 用背景对照前景和次要文字。前景低于 4.5 或次要文字低于 3 时不通过。解析不出的颜色不参与。tokens 按下标读取。 */
export function contrastOf(tokens: TokenMap): ContrastReport {
  const bg = parseRGB(tokens["--background"]);
  const fg = parseRGB(tokens["--foreground"]);
  const mu = parseRGB(tokens["--muted"]);
  const rep: ContrastReport = { foreground: 0, muted: 0, pass: true };
  if (bg && fg) {
    rep.foreground = ratio(bg, fg);
    if (rep.foreground < 4.5) {
      rep.pass = false;
      rep.reason = "Foreground contrast must be at least 4.5:1";
    }
  }
  if (bg && mu) {
    rep.muted = ratio(bg, mu);
    if (rep.muted < 3) {
      rep.pass = false;
      if (!rep.reason) rep.reason = "Muted contrast must be at least 3:1";
    }
  }
  return rep;
}
