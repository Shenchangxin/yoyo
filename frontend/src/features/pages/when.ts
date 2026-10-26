/**
 * 拥有：页面时间的相对写法和字数统计。
 * 不拥有：正文。本地格式化，不改服务端。
 */

/** formatPageWhen 把时刻写成相对时间。太远就写成月日。无法解析则空。iso 传给 Date.parse。locale 看开头。 */
export function formatPageWhen(iso: string, locale = "en"): string {
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return "";
  const diffSec = Math.round((t - Date.now()) / 1000);
  const abs = Math.abs(diffSec);
  const rtf = new Intl.RelativeTimeFormat(locale.startsWith("zh") ? "zh-CN" : "en", { numeric: "auto" });
  if (abs < 45) return rtf.format(0, "second");
  if (abs < 45 * 60) return rtf.format(Math.round(diffSec / 60), "minute");
  if (abs < 36 * 3600) return rtf.format(Math.round(diffSec / 3600), "hour");
  if (abs < 10 * 86400) return rtf.format(Math.round(diffSec / 86400), "day");
  return new Intl.DateTimeFormat(locale.startsWith("zh") ? "zh-CN" : "en", { month: "short", day: "numeric" }).format(t);
}

/** pageMeasure 中文按字数，英文按词数。text 去掉首尾空白后，做替换，切开，为假时走另一支。locale 看开头。 */
export function pageMeasure(text: string, locale = "en"): { n: number; kind: "words" | "chars" } {
  const t = text.trim();
  if (!t) return { n: 0, kind: locale.startsWith("zh") ? "chars" : "words" };
  if (locale.startsWith("zh")) return { n: Array.from(t.replace(/\s+/g, "")).length, kind: "chars" };
  return { n: t.split(/\s+/).filter(Boolean).length, kind: "words" };
}
