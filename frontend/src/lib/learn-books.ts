/**
 * 拥有：课本的类型和调用。字段名和 Go 侧的 JSON 一致。
 * 不拥有：写作、批改和块的校验，那些在引擎里。也不持有任何状态。
 */

import { learnCall } from "./client";
import type { QuizOption } from "./learn-api";

/** BookStatus 是课本的六种状态。draft 等确认提案，spine_ready 等确认章节，compiling 正在写。 */
export type BookStatus = "draft" | "spine_ready" | "compiling" | "paused" | "ready" | "error";

/** BookSummary 是列表里的一本课本。pause_reason 为 building_spine 时章节正在后台排。 */
export type BookSummary = {
  id: string;
  title: string;
  description: string;
  library_id: string;
  status: BookStatus;
  depth: "brief" | "standard" | "deep";
  language: string;
  chapters: number;
  ready_chapters: number;
  pause_reason: string;
  chapter_id: string;
  percent: number;
  updated_at: string;
};

/** BookProposal 是一本课本的提案。 */
export type BookProposal = {
  title: string;
  description: string;
  scope: string;
  target_level: string;
  estimated_chapters: number;
  rationale: string;
};

/** BookChapter 是书脊上的一章，不带块。 */
export type BookChapter = {
  id: string;
  order: number;
  title: string;
  summary: string;
  objectives: string[];
  content_type: string;
  status: "pending" | "planning" | "generating" | "ready" | "partial" | "error";
  error: string;
};

/** BlockKind 是块的八种种类。 */
export type BlockKind = "text" | "section" | "callout" | "code" | "timeline" | "flash_cards" | "quiz" | "user_note";

/** BookBlock 是章节里的一块。payload 的形状由 kind 决定；随堂题的 payload 里没有答案。 */
export type BookBlock = {
  id: string;
  chapter_id: string;
  order: number;
  kind: BlockKind;
  status: "pending" | "generating" | "ready" | "error";
  title: string;
  payload: any;
  error: string;
  edited: boolean;
};

/** BookChapterView 是打开的一章：章节加上它的块。 */
export type BookChapterView = BookChapter & { blocks: BookBlock[] };

/** BookView 是一本课本的全貌。 */
export type BookView = {
  book: BookSummary;
  proposal: BookProposal;
  chapters: BookChapter[];
  progress: { chapter_id: string; visited: string[]; percent: number };
};

/** ChapterInput 是改书脊时的一章。id 是现有章节，新加的留空。 */
export type ChapterInput = { id?: string; title: string; summary: string; objectives: string[]; content_type?: string };

/** QuizQuestion 是给界面的一道随堂题。 */
export type QuizQuestion = { id: string; question: string; qtype: "choice" | "short"; options: QuizOption[] };

/** QuizResult 是一道随堂题的批改结果。correct 为 null 表示没法判定。 */
export type QuizResult = { correct: boolean | null; answer: string; explanation: string; feedback?: string };

/** list 把可能是 null 的数组收成数组。 */
function list<T>(v: T[] | null | undefined): T[] {
  return Array.isArray(v) ? v : [];
}

/** normalizeView 给课本全貌里可能缺的数组补上空数组。 */
function normalizeView(v: BookView): BookView {
  return {
    ...v,
    chapters: list(v?.chapters).map((c) => ({ ...c, objectives: list(c.objectives) })),
    progress: { chapter_id: v?.progress?.chapter_id || "", visited: list(v?.progress?.visited), percent: Number(v?.progress?.percent || 0) },
  };
}

/** books 列出全部课本，最近动过的在前。 */
export async function books(): Promise<BookSummary[]> {
  return list(await learnCall("learn.books"));
}

/** createBook 请模型按意图写一份提案，存成草稿。libraryId 可空。session 决定用哪段对话的模型。 */
export async function createBook(req: { intent: string; libraryId?: string; depth?: string; session?: string }): Promise<{ book: BookSummary; proposal: BookProposal }> {
  return learnCall("learn.book.create", { intent: req.intent, library_id: req.libraryId || "", depth: req.depth || "standard", session: req.session || "" });
}

/** book 读出一本课本的全貌。 */
export async function book(id: string): Promise<BookView> {
  return normalizeView(await learnCall("learn.book", { id }));
}

/** confirmProposal 确认提案并在后台排章节。proposal 给了就先换上改过的提案。 */
export async function confirmProposal(id: string, proposal?: BookProposal, session = ""): Promise<void> {
  const params: Record<string, unknown> = { id, session };
  if (proposal) params.proposal = proposal;
  await learnCall("learn.book.confirm_proposal", params);
}

/** saveSpine 写入读者改过的章节列表。 */
export async function saveSpine(id: string, chapters: ChapterInput[]): Promise<BookChapter[]> {
  const res = await learnCall("learn.book.spine", { id, chapters });
  return list(res?.chapters);
}

/** confirmSpine 确认章节并开始逐章写。 */
export async function confirmSpine(id: string, session = ""): Promise<void> {
  await learnCall("learn.book.confirm_spine", { id, session });
}

/** chapter 读出一章和它的块，并记下读者打开过它。chapterId 为空时取上次读到的那一章。 */
export async function chapter(id: string, chapterId = ""): Promise<BookChapterView> {
  const res = await learnCall("learn.book.chapter", { id, chapter_id: chapterId });
  return { ...res, objectives: list(res?.objectives), blocks: list(res?.blocks) };
}

/** pauseBook 让正在写的课本停下。 */
export async function pauseBook(id: string): Promise<void> {
  await learnCall("learn.book.pause", { id });
}

/** resumeBook 接着写没写完的章节。 */
export async function resumeBook(id: string, session = ""): Promise<void> {
  await learnCall("learn.book.resume", { id, session });
}

/** deleteBook 删除一本课本。 */
export async function deleteBook(id: string): Promise<void> {
  await learnCall("learn.book.delete", { id });
}

/** renameBook 改课本的名字。 */
export async function renameBook(id: string, title: string): Promise<void> {
  await learnCall("learn.book.rename", { id, title });
}

/** regenerateChapter 重写一章。读者的笔记和改过的块留着。 */
export async function regenerateChapter(id: string, chapterId: string, session = ""): Promise<void> {
  await learnCall("learn.book.chapter.regenerate", { id, chapter_id: chapterId, session });
}

/** regenerateBlock 让模型把一块重写一遍。 */
export async function regenerateBlock(id: string, blockId: string, session = ""): Promise<void> {
  await learnCall("learn.book.block.regenerate", { id, block_id: blockId, session });
}

/** deleteBlock 删掉一块。 */
export async function deleteBlock(id: string, blockId: string): Promise<void> {
  await learnCall("learn.book.block.delete", { id, block_id: blockId });
}

/** updateBlock 改一块的正文。只有正文、提示框和读者笔记能改。 */
export async function updateBlock(id: string, blockId: string, body: string): Promise<BookBlock> {
  return learnCall("learn.book.block.update", { id, block_id: blockId, body });
}

/** insertNote 在一块后面插一条读者自己的笔记。afterBlockId 为空时放在这一章最前面。 */
export async function insertNote(id: string, chapterId: string, afterBlockId: string, body: string): Promise<BookBlock> {
  return learnCall("learn.book.block.insert", { id, chapter_id: chapterId, after_block_id: afterBlockId, kind: "user_note", body });
}

/** answerQuiz 作答一道随堂题，返回对错、答案和解析。 */
export async function answerQuiz(id: string, blockId: string, questionId: string, answer: string, session = ""): Promise<QuizResult> {
  return learnCall("learn.book.quiz", { id, block_id: blockId, question_id: questionId, answer, session });
}

/** exportBook 把整本书导成 Markdown。 */
export async function exportBook(id: string): Promise<{ filename: string; markdown: string }> {
  return learnCall("learn.book.export", { id });
}
