/**
 * 拥有：练习各处共用的小件：题型、来源、阶段和间隔的说法，亮出答案后的选项列表，以及表单里的一行标签加控件。
 * 不拥有：任何调用和状态。
 */

import type { ReactNode } from "react";
import type { useCopy } from "../../../lib/i18n";
import type { QuizOption } from "../../../lib/learn-api";
import type { Question } from "../../../lib/learn-practice";
import { fill, whenLabel } from "../../../lib/learn-format";
import { answerLabels, intervalParts } from "../../../lib/learn-practice-util";
import { cn } from "../../../lib/utils";

/** DrillCopy 是练习这一块的文案表。 */
export type DrillCopy = ReturnType<typeof useCopy>["learn"]["drill"];

/** typeLabel 是题型的名字。认不出的题型原样返回。 */
export function typeLabel(L: DrillCopy, qtype: string): string {
  switch (qtype) {
    case "choice": return L.typeChoice;
    case "multi": return L.typeMulti;
    case "true_false": return L.typeTrueFalse;
    case "fill": return L.typeFill;
    case "short": return L.typeShort;
    case "open": return L.typeOpen;
    default: return qtype;
  }
}

/** sourceLabel 是题目来源的说法。认不出的来源原样返回。 */
export function sourceLabel(L: DrillCopy, source: string): string {
  switch (source) {
    case "manual": case "": return L.sourceManual;
    case "import": return L.sourceImport;
    case "mastery": return L.sourceMastery;
    case "generated": return L.sourceGenerated;
    case "book": return L.sourceBook;
    case "reading": return L.sourceReading;
    default: return source;
  }
}

/** stateLabel 是一道题所处阶段的名字。 */
export function stateLabel(L: DrillCopy, state: string): string {
  if (state === "review") return L.stateReview;
  if (state === "learning") return L.stateLearning;
  return L.stateNew;
}

/** intervalLabel 把间隔天数说成「10 分钟」「3 天」这样的话。 */
export function intervalLabel(L: DrillCopy, days: number): string {
  const { n, unit } = intervalParts(days);
  const template = unit === "minute" ? L.unitMinute : unit === "hour" ? L.unitHour : unit === "day" ? L.unitDay : unit === "month" ? L.unitMonth : L.unitYear;
  return fill(template, { n });
}

/** dueLabel 说出一道题什么时候到期。暂停的题说已暂停，新题说还没开始，已经到期的说到期。locale 是当前语言。 */
export function dueLabel(L: DrillCopy, q: Question, locale: string, now: Date = new Date()): string {
  if (q.suspended) return L.suspended;
  if (q.state === "new") return L.notScheduled;
  const at = new Date(q.due_at);
  if (Number.isNaN(at.getTime()) || at.getTime() <= now.getTime()) return L.dueToday;
  return fill(L.dueWhen, { when: whenLabel(q.due_at, locale, L.today, now) });
}

/**
 * OptionKey 画出亮了答案的选项。answer 是标准答案的字母，多选用逗号隔开。picked 是学习者选的，可空。
 * 正确项用成功色，学习者选错的用危险色，其余淡出。
 */
export function OptionKey(props: { options: QuizOption[]; answer: string; picked?: string; dense?: boolean }) {
  const right = answerLabels(props.answer);
  const mine = answerLabels(props.picked || "");
  return (
    <ul className="flex flex-col gap-1">
      {props.options.map((o) => {
        const ok = right.has(o.label);
        const wrong = mine.has(o.label) && !ok;
        return (
          <li
            key={o.label}
            className={cn(
              "flex items-start gap-2.5 rounded-lg border border-border/60 text-ui-body",
              props.dense ? "px-2.5 py-1.5" : "px-3 py-2",
              ok && "border-success/50 bg-success/[0.11]",
              wrong && "border-danger/50 bg-danger/[0.11]",
              !ok && !wrong && "opacity-70",
            )}
          >
            <span className="mt-px w-4 shrink-0 font-medium tabular-nums text-muted">{o.label}</span>
            <span className="min-w-0 flex-1 break-words">{o.body}</span>
          </li>
        );
      })}
    </ul>
  );
}

/** Field 是表单里的一行：上面是标签，下面是控件。hint 是标签旁的一句说明。 */
export function Field(props: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-ui-footnote text-muted">
        {props.label}
        {props.hint ? <span className="text-muted/70"> · {props.hint}</span> : null}
      </span>
      {props.children}
    </label>
  );
}

/** textareaClass 是练习里多行输入框共用的样式。 */
export const textareaClass = "w-full resize-y rounded-lg border border-border/60 bg-transparent px-3 py-2 text-ui-body outline-none focus-visible:border-accent/60";
