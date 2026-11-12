/**
 * 拥有：从导师对话里的 learn_quiz 工具结果读出题卡要用的路径和题面。
 * 不拥有：题卡的界面和判分。这里只解析，不调用服务端。
 */

import type { PublicQuestion } from "./learn-api.ts";
import type { Item } from "./protocol.ts";
import { toolName, toolResultBody } from "./tool-summary.ts";

/** QuizPosted 是 learn_quiz 工具结果里题卡要用的部分。 */
export type QuizPosted = { pathId: string; question: PublicQuestion };

/** isLearnQuizItem 判断这条是不是出题工具的调用或结果。item 为空时返回假。 */
export function isLearnQuizItem(item?: Item | null): boolean {
  return !!item && toolName(item) === "learn_quiz";
}

/** parseQuizPosted 从出题工具的结果正文里读出路径和题面。正文不是 JSON、缺路径或缺题目 id 时返回 null。 */
export function parseQuizPosted(body: string): QuizPosted | null {
  try {
    const parsed = JSON.parse(body) as { path_id?: unknown; question?: PublicQuestion };
    const pathId = typeof parsed?.path_id === "string" ? parsed.path_id : "";
    const question = parsed?.question;
    if (!pathId || !question || typeof question.id !== "string" || !question.id) return null;
    return { pathId, question: { ...question, options: Array.isArray(question.options) ? question.options : [] } };
  } catch {
    return null;
  }
}

/** quizPostedOf 从工具结果条目里读出题卡要用的部分。条目为空或正文读不出题目时返回 null。 */
export function quizPostedOf(result?: Item | null): QuizPosted | null {
  if (!result) return null;
  return parseQuizPosted(toolResultBody(result));
}
