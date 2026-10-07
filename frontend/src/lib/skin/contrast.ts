import type { TokenMap } from "./schema.ts";

export type ContrastReport = {
  foreground: number;
  muted: number;
  pass: boolean;
  reason?: string;
};

function parseRGB(s: string | undefined): [number, number, number] | null {
  if (!s || !s.startsWith("#")) return null;
  const h = s.slice(1);
  const n = (a: string) => parseInt(a, 16) / 255;
  if (h.length === 3) return [n(h[0] + h[0]), n(h[1] + h[1]), n(h[2] + h[2])];
  if (h.length >= 6) return [n(h.slice(0, 2)), n(h.slice(2, 4)), n(h.slice(4, 6))];
  return null;
}

function lin(v: number): number {
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

function lum(c: [number, number, number]): number {
  return 0.2126 * lin(c[0]) + 0.7152 * lin(c[1]) + 0.0722 * lin(c[2]);
}

function ratio(a: [number, number, number], b: [number, number, number]): number {
  const l1 = lum(a);
  const l2 = lum(b);
  const hi = Math.max(l1, l2);
  const lo = Math.min(l1, l2);
  return (hi + 0.05) / (lo + 0.05);
}

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
  const hex = (v: number) => Math.round((v + m) * 255).toString(16).padStart(2, "0");
  return `#${hex(r)}${hex(g)}${hex(b)}`;
}

function contrastLum(a: number, b: number): number {
  const hi = Math.max(a, b);
  const lo = Math.min(a, b);
  return (hi + 0.05) / (lo + 0.05);
}

/** Smallest veil (0 = raw wallpaper, 1 = solid palette) so fg/muted stay readable over a wallpaper of `wallLum`. */
export function veilForReadability(wallLum: number, bgLum: number, fgLum: number, mutedLum?: number): number {
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
  // Cap Auto so a bright photo stays visible in dark mode; the slider can still go higher.
  return Math.min(0.58, Math.max(0, v));
}

export function relativeLuminance(r: number, g: number, b: number): number {
  return lum([r / 255, g / 255, b / 255]);
}

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
