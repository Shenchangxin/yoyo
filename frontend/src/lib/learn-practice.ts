/**
 * 拥有：练习和题库的类型和调用。字段名和 Go 侧的 JSON 一致。
 * 不拥有：日程、判分和校验，那些在引擎里。不碰网络的小工具在 learn-practice-util.ts。也不持有任何状态。
 */

import { learnCall } from "./client";
import type { QuizOption } from "./learn-api";

/** QuestionType 是题库里的六种题型。前四种在本地批改，后两种自评或请模型判。 */
export type QuestionType = "choice" | "multi" | "true_false" | "fill" | "short" | "open";

/** Rating 是一次复习的四种评分。 */
export type Rating = "again" | "hard" | "good" | "easy";

/** RATINGS 是评分按钮从左到右的顺序。 */
export const RATINGS: Rating[] = ["again", "hard", "good", "easy"];

/** Question 是题库里的一道题和它的日程。没有标准答案。 */
export type Question = {
  id: string;
  stem: string;
  qtype: QuestionType;
  options: QuizOption[];
  difficulty: string;
  tags: string[];
  source: string;
  source_id: string;
  path_id: string;
  point_id: string;
  library_id: string;
  bookmarked: boolean;
  mistake: boolean;
  suspended: boolean;
  created_at: string;
  updated_at: string;
  state: "new" | "learning" | "review";
  due_at: string;
  reps: number;
  lapses: number;
  stability: number;
  retrievability: number;
  last_review_at: string;
};

/** Reveal 是作答之后才给的标准答案和解析。 */
export type Reveal = { answer: string; explanation: string };

/** CheckResult 是核对一次作答的结果。correct 为 null 表示没有结论，由学习者自评。intervals 是四种评分各自排出的天数。 */
export type CheckResult = {
  correct: boolean | null;
  feedback?: string;
  reveal: Reveal;
  suggested: Rating | null;
  intervals: Record<Rating, number>;
};

/** ReviewResult 是一次评分的结果。 */
export type ReviewResult = { question: Question; interval_days: number; reveal: Reveal };

/** PracticeSummary 是练习页顶部的数字。 */
export type PracticeSummary = {
  due: number;
  new: number;
  reviewed_today: number;
  total: number;
  mistakes: number;
  next_due_at: string;
  retention: number;
  new_per_day: number;
  streak_days: number;
};

/** PracticeSettings 是练习的两项设置。 */
export type PracticeSettings = { retention: number; new_per_day: number };

/** Insights 是练习的统计。 */
export type Insights = {
  days: number;
  daily: { date: string; reviews: number; again: number; new_cards: number; mistakes: number }[];
  retention: { rate: number; reviews: number };
  forecast: { date: string; due: number }[];
  sources: { source: string; count: number }[];
};

/** QueueFilter 是取队列时的筛选。ids 非空时只练点名的题。 */
export type QueueFilter = { limit?: number; tag?: string; source?: string; mistakesOnly?: boolean; ids?: string[] };

/** BankFilter 是题库列表的筛选、排序和分页。 */
export type BankFilter = {
  query?: string;
  tag?: string;
  source?: string;
  mistakesOnly?: boolean;
  bookmarked?: boolean;
  suspended?: boolean;
  sort?: "recent" | "oldest" | "due";
  limit?: number;
  offset?: number;
};

/** QuestionEvent 是一条复习记录。 */
export type QuestionEvent = { at: string; rating: Rating; correct: boolean | null; answer: string };

/** QuestionDetail 是一道题的详情，带标准答案和复习记录。 */
export type QuestionDetail = { question: Question; reveal: Reveal; events: QuestionEvent[] };

/** DraftQuestion 是一道还没进题库的题，带标准答案。 */
export type DraftQuestion = {
  stem: string;
  qtype: QuestionType;
  options: QuizOption[];
  answer: string;
  explanation: string;
  difficulty: string;
  tags?: string[];
};

/** ImportRow 是导入预览里的一行。question 和 error 只有一个有值。 */
export type ImportRow = { line: number; question?: DraftQuestion; error?: string; duplicate?: boolean };

/** ImportPreview 是一次导入预览。token 交给提交。 */
export type ImportPreview = { token: string; rows: ImportRow[]; valid: number; invalid: number; duplicates: number };

/** list 把可能是 null 的数组收成数组。 */
function list<T>(v: T[] | null | undefined): T[] {
  return Array.isArray(v) ? v : [];
}

/** normalizeQuestion 给题目里可能缺的数组补上空数组。 */
function normalizeQuestion(q: Question): Question {
  return { ...q, options: list(q?.options), tags: list(q?.tags) };
}

/** summary 读取练习页顶部的数字。 */
export async function summary(): Promise<PracticeSummary> {
  return learnCall("learn.practice.summary");
}

/** queue 取现在该练的题：先到期的，再是每天上限之内的新题。 */
export async function queue(filter: QueueFilter = {}): Promise<Question[]> {
  const rows = await learnCall("learn.practice.queue", {
    limit: filter.limit || 0, tag: filter.tag || "", source: filter.source || "", mistakes_only: !!filter.mistakesOnly, ids: filter.ids || [],
  });
  return list<Question>(rows).map(normalizeQuestion);
}

/** check 核对一次作答并亮出答案，不改日程。judge 为真时简答和开放题请模型判，session 决定用哪段对话的模型。 */
export async function check(id: string, answer: string, opts: { judge?: boolean; session?: string } = {}): Promise<CheckResult> {
  return learnCall("learn.practice.check", { id, answer, judge: !!opts.judge, session: opts.session || "" });
}

/** review 按评分排下一次复习。requestId 是幂等键，同一个键只排一次。correct 是核对出的对错，没有结论时不传。 */
export async function review(id: string, rating: Rating, opts: { answer?: string; correct?: boolean | null; requestId: string }): Promise<ReviewResult> {
  const params: Record<string, unknown> = { id, rating, answer: opts.answer || "", request_id: opts.requestId };
  if (typeof opts.correct === "boolean") params.correct = opts.correct;
  const res = await learnCall("learn.practice.review", params);
  return { ...res, question: normalizeQuestion(res.question) };
}

/** insights 读取最近 days 天的统计。 */
export async function insights(days: number): Promise<Insights> {
  const res = await learnCall("learn.practice.insights", { days });
  return { ...res, daily: list(res?.daily), forecast: list(res?.forecast), sources: list(res?.sources) };
}

/** saveSettings 改目标记住率或每天的新题上限，返回改完的值。 */
export async function saveSettings(patch: Partial<PracticeSettings>): Promise<PracticeSettings> {
  return learnCall("learn.practice.settings", patch);
}

/** questions 按筛选列出题库的一页。 */
export async function questions(filter: BankFilter = {}): Promise<{ items: Question[]; total: number }> {
  const params: Record<string, unknown> = {
    query: filter.query || "", tag: filter.tag || "", source: filter.source || "", mistakes_only: !!filter.mistakesOnly,
    sort: filter.sort || "recent", limit: filter.limit || 0, offset: filter.offset || 0,
  };
  if (typeof filter.bookmarked === "boolean") params.bookmarked = filter.bookmarked;
  if (typeof filter.suspended === "boolean") params.suspended = filter.suspended;
  const res = await learnCall("learn.questions", params);
  return { items: list<Question>(res?.items).map(normalizeQuestion), total: Number(res?.total || 0) };
}

/** question 读取一道题的详情，带答案和复习记录。 */
export async function question(id: string): Promise<QuestionDetail> {
  const res = await learnCall("learn.question", { id });
  return { ...res, question: normalizeQuestion(res.question), events: list(res?.events) };
}

/** saveQuestion 录入或修改一道题。id 为空是新建。 */
export async function saveQuestion(draft: DraftQuestion, id = ""): Promise<Question> {
  return normalizeQuestion(await learnCall("learn.questions.save", { id, ...draft, tags: draft.tags || [] }));
}

/** deleteQuestions 删除点名的题，连同它们的日程和记录。 */
export async function deleteQuestions(ids: string[]): Promise<void> {
  await learnCall("learn.questions.delete", { ids });
}

/** flagQuestions 改一批题的收藏、暂停或错题标记。只传要改的项。 */
export async function flagQuestions(ids: string[], flags: { bookmarked?: boolean; suspended?: boolean; mistake?: boolean }): Promise<void> {
  await learnCall("learn.questions.flag", { ids, ...flags });
}

/** tags 统计题库里每个标签的题数。 */
export async function tags(): Promise<{ tag: string; count: number }[]> {
  return list(await learnCall("learn.questions.tags"));
}

/** previewImport 预览一次导入。path 是本机文件，text 是粘贴的内容，给一个。 */
export async function previewImport(source: { path?: string; text?: string }): Promise<ImportPreview> {
  const res = await learnCall("learn.questions.import.preview", { path: source.path || "", text: source.text || "" });
  return { ...res, rows: list(res?.rows) };
}

/** commitImport 把预览过的题写进题库。 */
export async function commitImport(token: string, opts: { tags?: string[]; asMistakes?: boolean } = {}): Promise<{ imported: number; skipped: number }> {
  return learnCall("learn.questions.import.commit", { token, tags: opts.tags || [], as_mistakes: !!opts.asMistakes });
}

/** generate 请模型按主题出一组题，只返回草稿。 */
export async function generate(req: { topic: string; libraryId?: string; count?: number; qtypes?: QuestionType[]; difficulty?: string; session?: string }): Promise<DraftQuestion[]> {
  const res = await learnCall("learn.questions.generate", {
    topic: req.topic, library_id: req.libraryId || "", count: req.count || 0, qtypes: req.qtypes || [], difficulty: req.difficulty || "", session: req.session || "",
  });
  return list<DraftQuestion>(res?.items).map((d) => ({ ...d, options: list(d.options) }));
}

/** addQuestions 把挑中的草稿存进题库。 */
export async function addQuestions(items: DraftQuestion[], extraTags: string[] = [], session = ""): Promise<{ added: number; skipped: number }> {
  return learnCall("learn.questions.add", { items, tags: extraTags, session });
}
