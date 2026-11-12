/**
 * 拥有：学习界面里几处共用的文字拼法：填模板、百分比、到期日的说法。
 * 不拥有：文案本身，那在 copy-learn。这里没有状态。
 */

/** fill 把模板里的 {名字} 换成 vars 里的值。没给的名字原样留着。 */
export function fill(template: string, vars: Record<string, string | number>): string {
  return Object.entries(vars).reduce((text, [key, value]) => text.split(`{${key}}`).join(String(value)), template);
}

/** percent 把 0 到 1 的数写成整数百分比。不是数时按 0。 */
export function percent(n: number | undefined): number {
  if (typeof n !== "number" || Number.isNaN(n)) return 0;
  return Math.round(Math.min(1, Math.max(0, n)) * 100);
}

/** dayDistance 返回 iso 这一刻距离 now 所在那天隔了几天。同一天是 0，明天是 1。解析不了时返回 null。 */
export function dayDistance(iso: string | undefined, now: Date = new Date()): number | null {
  if (!iso) return null;
  const at = new Date(iso);
  if (Number.isNaN(at.getTime())) return null;
  const a = new Date(at.getFullYear(), at.getMonth(), at.getDate()).getTime();
  const b = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  return Math.round((a - b) / 86_400_000);
}

/** whenLabel 把到期时刻说成「今天」「明天」「3 天后」这样的话。超过 60 天写日期。today 是「今天」在当前语言里的说法。 */
export function whenLabel(iso: string | undefined, locale: string, today: string, now: Date = new Date()): string {
  const days = dayDistance(iso, now);
  if (days === null) return "";
  if (days <= 0) return today;
  if (days > 60) {
    return new Intl.DateTimeFormat(locale, { month: "short", day: "numeric" }).format(new Date(iso as string));
  }
  return new Intl.RelativeTimeFormat(locale, { numeric: "auto" }).format(days, "day");
}
