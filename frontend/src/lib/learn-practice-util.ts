/**
 * 拥有：练习界面用的几个纯函数：题型能不能本地批改、多选的作答怎么拼、间隔怎么读、标签怎么拆、幂等键怎么生成。
 * 不拥有：任何调用和状态。这里不引入 client，好在 node 里直接测。
 */

/** LabeledOption 是带字母的选项，只用到字母。 */
type LabeledOption = { label: string };

/** objective 在题型能在本地批改时为真。 */
export function objective(qtype: string): boolean {
  return qtype === "choice" || qtype === "multi" || qtype === "true_false" || qtype === "fill";
}

/** answerLabels 把标准答案或作答拆成选项字母的集合。 */
export function answerLabels(answer: string): Set<string> {
  return new Set((answer || "").split(",").map((s) => s.trim()).filter(Boolean));
}

/** toggleLabel 在多选的作答里勾上或去掉一个选项，返回按选项顺序排好、用逗号连起来的字母。 */
export function toggleLabel(options: LabeledOption[], picked: string, label: string): string {
  const on = answerLabels(picked);
  if (on.has(label)) on.delete(label);
  else on.add(label);
  return options.map((o) => o.label).filter((l) => on.has(l)).join(",");
}

/** IntervalUnit 是间隔的单位。 */
export type IntervalUnit = "minute" | "hour" | "day" | "month" | "year";

/**
 * intervalParts 把间隔天数换成一个数和一个单位，界面再配上文字。
 * 不到一小时用分钟，不到一天用小时，不到 30 天用天，不到一年用月，其余用年。月和年留一位小数。
 */
export function intervalParts(days: number): { n: number; unit: IntervalUnit } {
  const minutes = Math.max(0, days) * 1440;
  if (minutes < 60) return { n: Math.max(1, Math.round(minutes)), unit: "minute" };
  if (days < 1) return { n: Math.round(minutes / 60), unit: "hour" };
  if (days < 30) return { n: Math.round(days), unit: "day" };
  if (days < 365) return { n: Math.round((days / 30) * 10) / 10, unit: "month" };
  return { n: Math.round((days / 365) * 10) / 10, unit: "year" };
}

/** splitTags 把一段用逗号、分号或空白隔开的文字拆成标签，去掉开头的井号和重复项。 */
export function splitTags(text: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of (text || "").split(/[,，;；\s]+/)) {
    const tag = raw.trim().replace(/^[#＃]+/, "");
    const key = tag.toLowerCase();
    if (!tag || seen.has(key)) continue;
    seen.add(key);
    out.push(tag);
  }
  return out;
}

/** newRequestId 生成一次评分的幂等键。 */
export function newRequestId(): string {
  const rand = typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  return `rev-${rand}`;
}

/** barHeights 把一列数换成 0 到 100 的柱高。全是 0 时全是 0；非 0 的值至少 4，免得小数看不见。 */
export function barHeights(values: number[]): number[] {
  const max = Math.max(0, ...values);
  if (max <= 0) return values.map(() => 0);
  return values.map((v) => (v <= 0 ? 0 : Math.max(4, Math.round((v / max) * 100))));
}
