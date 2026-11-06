/**
 * 拥有：简报看板上和你的标记有关的规则：哪张卡算要处理的、什么时候算处理完、勾掉的条目记在哪个键上、处理到第几张。
 * 卡上的内容写完就不变，这里只读 picked 里的标记。
 * 不拥有：卡怎么画，那在 WorkPane；也不拥有标记怎么存，那在今天栏的 choose。
 */

import type { BriefCard, BriefLayout, MorningBrief } from "./work-model.ts";

/** briefLayout 是这张卡实际用的版式。模型没写时按内容推断：两个以上选项是对比，有条目是清单，否则是一段话。 */
export function briefLayout(card: BriefCard): BriefLayout {
  return card.layout || (card.choices.length >= 2 ? "compare" : card.items.length > 0 ? "checks" : "plain");
}

/** actionLabel 用模型写的短字。没写短字、但写了点下去要放进输入框的请求时，用请求的第一句，最多十八个字。 */
export function actionLabel(label?: string, task?: string): string {
  const name = (label || "").trim();
  if (name) return name;
  const line = ((task || "").trim().split("\n")[0] || "").trim();
  if (!line) return "";
  const chars = Array.from(line);
  return chars.length > 18 ? chars.slice(0, 18).join("") : line;
}

/** itemKey 是清单里第 index 条的勾记在 picked 里的键。和按钮的键（卡的 id）不会撞。 */
export function itemKey(cardId: string, index: number): string {
  return `${cardId}#${index}`;
}

function checklist(card: BriefCard): boolean {
  return briefLayout(card) === "checks" && card.items.length > 0;
}

/** cardActionable 为真时这张卡有事可做：有按钮，或者是可以逐条勾掉的清单。只给你看的卡不算进进度。 */
export function cardActionable(card: BriefCard): boolean {
  return !!(actionLabel(card.primary, card.task) || actionLabel(card.secondary, card.alt)) || checklist(card);
}

/** checkedCount 是清单里已经勾掉的条数。 */
export function checkedCount(card: BriefCard, picked: Record<string, string>): number {
  if (!checklist(card)) return 0;
  return card.items.filter((_, index) => !!picked[itemKey(card.id, index)]).length;
}

/** cardHandled 为真时这张卡处理完了：点过按钮，或者清单全部勾掉。 */
export function cardHandled(card: BriefCard, picked: Record<string, string>): boolean {
  if (picked[card.id]) return true;
  return checklist(card) && checkedCount(card, picked) === card.items.length;
}

/** briefProgress 是一期简报里要处理的卡有几张、处理完了几张。 */
export function briefProgress(brief: MorningBrief): { done: number; total: number } {
  const cards = brief.cards.filter(cardActionable);
  return { done: cards.filter((card) => cardHandled(card, brief.picked)).length, total: cards.length };
}

/** shortKicker 是目录里那枚小标签上的字：眉题在「·」前的那一段，没有眉题时用标题的前几个字。 */
export function shortKicker(card: BriefCard): string {
  const head = (card.kicker || "").split(/\s*[·•|]\s*/)[0].trim();
  if (head) return head;
  const chars = Array.from(card.title.trim());
  return chars.length > 8 ? `${chars.slice(0, 8).join("")}…` : chars.join("");
}
