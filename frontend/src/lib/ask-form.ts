/**
 * 拥有：把 ask_user 的命令收成问题和选项，并在操作者作答或空闲时拼出答复。
 * 不拥有：提问卡片的渲染，以及批准请求的发送。
 */

import type { Approval } from "./protocol";

/** ASK_IDLE_MS 未碰过的提问卡片在采用建议前等待的毫秒数。 */
export const ASK_IDLE_MS = 10_000;

/** AskOption 一个选项。 */
export type AskOption = { id: string; label: string };
/** AskQuestion 一道题。recommended 是建议选项的 id。没有选项时建议为空。 */
export type AskQuestion = {
  id: string;
  prompt: string;
  options: AskOption[];
  recommended: string;
};
/** AskForm 一张提问卡的标题和题目。 */
export type AskForm = { title: string; questions: AskQuestion[] };

/** pendingAsk 找出 ask_user。给了 sessionId 时，只留没有会话或会话相同的那条。 */
export function pendingAsk(approvals: Approval[], sessionId?: string): Approval | undefined {
  return approvals.find((a) => a.action === "ask_user" && (!sessionId || !a.sessionId || a.sessionId === sessionId));
}

/** parseAskCommand 以 `{` 开头且能解析出题目时当表单。否则整段当成一道没有选项的题。JSON 解析失败也走这一道题。 */
export function parseAskCommand(command: string): AskForm {
  const text = command.trim();
  if (text.startsWith("{")) {
    try {
      const raw = JSON.parse(text) as { title?: unknown; questions?: unknown };
      const questions = normalizeQuestions(raw.questions);
      if (questions.length) return { title: textOf(raw.title), questions };
    } catch {
      /* 当成一道普通题，不当表单 */
    }
  }
  return { title: "", questions: [{ id: "q1", prompt: text, options: [], recommended: "" }] };
}

/** askCanIdle 有题目，且每道题都已有选项时，卡片才能自己继续。 */
export function askCanIdle(form: AskForm): boolean {
  return form.questions.length > 0 && form.questions.every((q) => q.options.length > 0);
}

/** recommendedPicks 每题取建议项，没有则取第一项。 */
export function recommendedPicks(form: AskForm): Record<string, string> {
  const picks: Record<string, string> = {};
  for (const q of form.questions) {
    if (q.recommended) picks[q.id] = q.recommended;
    else if (q.options[0]) picks[q.id] = q.options[0].id;
  }
  return picks;
}

/** idleAnswer 用建议项拼出空闲答复，并注明操作者空闲。 */
export function idleAnswer(form: AskForm): string {
  return formatAsk(form, recommendedPicks(form), {}, true);
}

/** chosenAnswer 用操作者的选择和自由输入拼出答复。 */
export function chosenAnswer(form: AskForm, picks: Record<string, string>, free: Record<string, string>): string {
  return formatAsk(form, picks, free, false);
}

/** 有选项时写成选项 id 和标签。没有选项时用自由输入，空白则写 continue with your best judgment。空闲与作答各加一行 note。 */
function formatAsk(form: AskForm, picks: Record<string, string>, free: Record<string, string>, idle: boolean): string {
  const lines = form.questions.map((q) => {
    if (!q.options.length) {
      const typed = (free[q.id] || "").trim() || "continue with your best judgment";
      return `${q.id}: ${typed}`;
    }
    const id = picks[q.id] || q.recommended || q.options[0].id;
    const opt = q.options.find((o) => o.id === id) || q.options[0];
    return `${q.id}: ${opt.id} — ${opt.label}`;
  });
  lines.push(idle ? "note: operator idle; used recommended" : "note: operator answered");
  return lines.join("\n");
}

/** 只保留有题干的题目。建议必须是某个选项，否则改成第一项。没有选项则建议为空。最多六题，每题最多八个选项。 */
function normalizeQuestions(raw: unknown): AskQuestion[] {
  if (!Array.isArray(raw)) return [];
  const out: AskQuestion[] = [];
  for (const item of raw) {
    if (!item || typeof item !== "object") continue;
    const rec = item as Record<string, unknown>;
    const prompt = textOf(rec.prompt || rec.question);
    if (!prompt) continue;
    const options = normalizeOptions(rec.options).slice(0, 8);
    let recommended = textOf(rec.recommended);
    if (options.length && !options.some((o) => o.id === recommended)) recommended = options[0].id;
    if (!options.length) recommended = "";
    out.push({
      id: textOf(rec.id) || `q${out.length + 1}`,
      prompt,
      options,
      recommended,
    });
    if (out.length >= 6) break;
  }
  return out;
}

/** 字符串选项按出现顺序编号。对象选项要有标签或 id。 */
function normalizeOptions(raw: unknown): AskOption[] {
  if (!Array.isArray(raw)) return [];
  const out: AskOption[] = [];
  for (const item of raw) {
    if (typeof item === "string") {
      const label = item.trim();
      if (label) out.push({ id: String(out.length + 1), label });
      continue;
    }
    if (!item || typeof item !== "object") continue;
    const rec = item as Record<string, unknown>;
    const label = textOf(rec.label) || textOf(rec.id);
    if (!label) continue;
    out.push({ id: textOf(rec.id) || String(out.length + 1), label });
  }
  return out;
}

/** 收成去掉两端空白的字符串。 */
function textOf(v: unknown): string {
  return String(v ?? "").trim();
}
