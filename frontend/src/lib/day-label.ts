/**
 * 拥有：主会话里按天分隔用的两样东西：一个时刻落在哪个本地日历日，以及这一天怎么称呼。
 * 不拥有：分隔线画在哪、长什么样，那是 Transcript 的事。
 */

/** dayKey 是 time 所在的本地日历日，形如 2026-11-12。无效时间返回空串。 */
export function dayKey(time: number): string {
  if (!Number.isFinite(time) || time <= 0) return "";
  const d = new Date(time);
  const pad = (v: number) => String(v).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/**
 * dayLabel 把一天写成人话：今天、昨天；同一年写月日和星期；更早带上年份。无效时间返回空串。
 * words 是「今天」「昨天」两个词，locale 是日期用的地区码，now 只为测试能定住时间。
 */
export function dayLabel(time: number, words: { today: string; yesterday: string }, locale: string, now = Date.now()): string {
  const key = dayKey(time);
  if (!key) return "";
  const today = new Date(now);
  if (key === dayKey(now)) return words.today;
  const before = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1).getTime();
  if (key === dayKey(before)) return words.yesterday;
  const d = new Date(time);
  const sameYear = d.getFullYear() === today.getFullYear();
  try {
    return new Intl.DateTimeFormat(locale, {
      year: sameYear ? undefined : "numeric",
      month: "short",
      day: "numeric",
      weekday: "short",
    }).format(d);
  } catch {
    return key;
  }
}
