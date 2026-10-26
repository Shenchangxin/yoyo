import { SPECS_BY_NAME, TOKEN_NAMES, type TokenKind, type TokenMap } from "./schema.ts";

/**
 * 拥有：皮肤令牌的名字和值是否能安全写进样式，以及壁纸文件名是否允许。
 * 不拥有：令牌表本身，以及把通过校验的值写到文档。
 */

const NAME = /^--[a-z0-9-]+$/;
const HEX = /^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/;
const VAR = /^var\(--[a-z0-9-]+\)$/;
const DIM = /^(?:0|[0-9]*\.?[0-9]+)(?:px|rem|em|%)$/;
const OKLCH = /^oklch\(\s*[0-9.%\s./+-]+\s*\)$/;
const RGB = /^(?:rgb|hsl)a?\(\s*[0-9.%\s,./+-]+\s*\)$/;
const MIX = /^color-mix\(\s*in\s+srgb\s*,.+\)$/;

/** isSafeTokenName 名字符合 CSS 变量形状，并且在允许表里。 */
export function isSafeTokenName(name: string): boolean {
  return NAME.test(name) && TOKEN_NAMES.has(name);
}

/** isSafeTokenValue 按种类检查值。拒绝分号、尖括号、url、expression、import 和 javascript，也拒绝非 ASCII。长度和阴影另有上限与放行。 */
export function isSafeTokenValue(kind: TokenKind, raw: string): boolean {
  const v = (raw || "").trim();
  if (!v || v.length > 240) return false;
  if (/[;{}<>`]/.test(v)) return false;
  const lower = v.toLowerCase();
  if (lower.includes("url(") || lower.includes("expression") || lower.includes("@import") || lower.includes("javascript")) {
    return false;
  }
  if (![...v].every((ch) => ch.charCodeAt(0) < 128)) return false;
  if (kind === "dimension") return DIM.test(v);
  if (kind === "shadow") return true;
  return v === "transparent" || v === "white" || v === "black" || HEX.test(v) || VAR.test(v) || OKLCH.test(v) || RGB.test(v) || MIX.test(v);
}

/** sanitizeTokenMap 只保留名字和值都安全的项。 */
export function sanitizeTokenMap(m: TokenMap | null | undefined): TokenMap {
  const out: TokenMap = {};
  if (!m) return out;
  for (const [k, raw] of Object.entries(m)) {
    if (!isSafeTokenName(k)) continue;
    const spec = SPECS_BY_NAME.get(k);
    if (!spec) continue;
    const v = String(raw ?? "").trim();
    if (!isSafeTokenValue(spec.kind, v)) continue;
    out[k] = v;
  }
  return out;
}

/** wallpaperFileOk 壁纸或预览图文件名是 wallpaper 或 preview，扩展名为 png、jpg、jpeg 或 webp。 */
export function wallpaperFileOk(name: string): boolean {
  return /^(?:wallpaper\.)(?:png|jpe?g|webp)$/i.test(name) || /^(?:preview\.)(?:png|jpe?g|webp)$/i.test(name);
}
