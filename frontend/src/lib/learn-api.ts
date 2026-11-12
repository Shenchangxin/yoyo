/**
 * 拥有：学习模块里掌握路径这一块的类型和调用。字段名和 Go 侧的 JSON 一致。
 * 不拥有：文库、练习和课本的调用，那几块还直接用 learnCall。也不持有任何状态。
 */

import { learnCall } from "./client";

/** PointKind 是知识点的四种类型。记忆和步骤靠答题过门，概念和设计靠讲解过门。 */
export type PointKind = "memory" | "procedure" | "concept" | "design";

/** PathMode 是路径的三种模式。 */
export type PathMode = "outline" | "study" | "review";

/** NextAction 是引擎给出的下一步。 */
export type NextAction = "outline" | "answer_pending" | "review" | "probe" | "teach" | "practice" | "assess" | "complete";

/** QuizOption 是选择题的一个选项。 */
export type QuizOption = { label: string; body: string };

/** PublicQuestion 是一道待答的题。没有标准答案。 */
export type PublicQuestion = {
  id: string;
  point_id: string;
  question_type: "choice" | "short" | "open";
  prompt: string;
  options?: QuizOption[];
  difficulty?: string;
  purpose?: string;
  created_at?: string;
};

/** NextStep 是下一步。question 只在 answer_pending 时有。 */
export type NextStep = {
  action: NextAction;
  point_id?: string;
  name?: string;
  kind?: PointKind;
  module_id?: string;
  reason?: string;
  question?: PublicQuestion;
};

/** PathCounts 是各状态的知识点数。 */
export type PathCounts = { total: number; mastered: number; learning: number; new: number };

/** PathSummary 是列表里的一条路径。 */
export type PathSummary = {
  id: string;
  name: string;
  goal?: string;
  description?: string;
  library_id?: string;
  mode: PathMode;
  status: string;
  counts: PathCounts;
  due_reviews: number;
  pending_question: boolean;
  complete: boolean;
  updated_at: string;
};

/** PointView 是地图上的一个知识点。mastery_source 为 learner 表示学习者自己声明会了。 */
export type PointView = {
  id: string;
  module_id: string;
  name: string;
  kind: PointKind;
  status: "new" | "learning" | "mastered";
  mastery: number;
  mastery_source?: "system" | "learner" | "";
  gate: "quantitative" | "qualitative";
  prereqs?: string[];
  blocked?: boolean;
  deferred?: boolean;
  attempts: number;
  due_at?: string;
  retrievability?: number;
};

/** ModuleView 是地图上的一章。 */
export type ModuleView = {
  id: string;
  name: string;
  objective?: string;
  order: number;
  mastered: number;
  total: number;
  points: PointView[];
};

/** ReviewItem 是一条复习。due 为真表示已经到期。 */
export type ReviewItem = { point_id: string; name: string; kind: PointKind; due_at: string; due: boolean; retrievability: number };

/** PathProfile 是这条路径上学习者的情况。 */
export type PathProfile = {
  teaching: "probe_first" | "teach_first";
  prior_knowledge: string;
  target_level: string;
  time_budget: string;
  preferences: string;
};

/** PathView 是一条路径的完整地图。 */
export type PathView = PathSummary & {
  profile: PathProfile;
  intake_needed: boolean;
  desired_retention: number;
  revision: number;
  sources: { id: string; kind: string; ref: string; label: string }[];
  modules: ModuleView[];
  next?: NextStep;
  reviews: ReviewItem[];
};

/** GradeResult 是一次作答的判定。final 为假表示没法判定，题目还在等。 */
export type GradeResult = {
  question_id: string;
  point_id: string;
  result: "correct" | "partial" | "incorrect" | "ungraded" | "voided";
  correct: boolean;
  final: boolean;
  replayed?: boolean;
  feedback?: string;
  expected?: string;
  explanation?: string;
  mastery: number;
  mastered: boolean;
  status: string;
  gate: string;
  next?: NextStep;
};

/** QuestionState 是一道题现在的样子。标准答案和解析只在判完之后才有。 */
export type QuestionState = {
  question: PublicQuestion;
  status: "pending" | "graded" | "skipped" | "voided";
  answer?: string;
  result?: GradeResult["result"];
  feedback?: string;
  expected?: string;
  explanation?: string;
};

/** OutlinePoint 是大纲里的一个知识点。id 指向已有的点，prereqs 可以写 id 或名称。 */
export type OutlinePoint = { id?: string; ref?: string; name: string; kind: PointKind; prereqs?: string[] };

/** OutlineModule 是大纲里的一章。 */
export type OutlineModule = { id?: string; name: string; objective?: string; points: OutlinePoint[] };

/** OutlineDraft 是模型起草的大纲，还没写入。 */
export type OutlineDraft = { description: string; modules: OutlineModule[] };

/** PointEvidence 是知识点详情里的一条记录。 */
export type PointEvidence = {
  at: string;
  kind: "quiz" | "qualitative" | "claim";
  result: string;
  independent: boolean;
  answer?: string;
  feedback?: string;
  prompt?: string;
  expected?: string;
};

/** PointDetail 是一个知识点的详情。 */
export type PointDetail = {
  point: PointView;
  module: string;
  stability: number;
  difficulty: number;
  reps: number;
  lapses: number;
  last_review_at?: string;
  evidence: PointEvidence[];
};

/** list 把可能是 null 的数组收成数组。 */
function list<T>(v: T[] | null | undefined): T[] {
  return Array.isArray(v) ? v : [];
}

/** normalizePath 给视图里可能缺的数组补上空数组，界面就不用到处判空。 */
function normalizePath(v: PathView): PathView {
  return {
    ...v,
    sources: list(v.sources),
    reviews: list(v.reviews),
    modules: list(v.modules).map((m) => ({ ...m, points: list(m.points) })),
  };
}

/** paths 列出全部路径。 */
export async function paths(): Promise<PathSummary[]> {
  return list(await learnCall("learn.paths"));
}

/** path 读取一条路径的地图。id 是路径 id。 */
export async function path(id: string): Promise<PathView> {
  return normalizePath(await learnCall("learn.path", { id }));
}

/** createPath 按学习目标新建路径。goal 是目标，libraryId 可空。 */
export async function createPath(goal: string, libraryId = ""): Promise<PathView> {
  return normalizePath(await learnCall("learn.path.create", { goal, library_id: libraryId }));
}

/** renamePath 改路径名。 */
export async function renamePath(id: string, name: string): Promise<void> {
  await learnCall("learn.path.rename", { id, name });
}

/** describePath 写下路径的一句话说明。 */
export async function describePath(id: string, description: string): Promise<void> {
  await learnCall("learn.path.describe", { id, description });
}

/** deletePath 删除路径和它的全部记录。 */
export async function deletePath(id: string): Promise<void> {
  await learnCall("learn.path.delete", { id });
}

/** resetPath 清掉这条路径上的作答和日程，大纲留着。 */
export async function resetPath(id: string): Promise<void> {
  await learnCall("learn.path.reset", { id });
}

/** outline 读取当前大纲，带 id。 */
export async function outline(id: string): Promise<OutlineModule[]> {
  return list<OutlineModule>(await learnCall("learn.path.outline.get", { id })).map((m) => ({ ...m, points: list(m.points) }));
}

/** saveOutline 写入大纲。留下来的知识点保住进度。返回写入后的地图。 */
export async function saveOutline(id: string, modules: OutlineModule[]): Promise<PathView> {
  const res = await learnCall("learn.path.outline", { id, modules });
  return normalizePath(res.path);
}

/** draftOutline 请模型起草大纲，不写入。session 决定用哪段对话的模型。 */
export async function draftOutline(id: string, session = "", mustCover = ""): Promise<OutlineDraft> {
  const res = await learnCall("learn.path.draft", { id, session, must_cover: mustCover });
  return { description: String(res?.description || ""), modules: list<OutlineModule>(res?.modules).map((m) => ({ ...m, points: list(m.points) })) };
}

/** setMode 切换路径模式。 */
export async function setMode(id: string, mode: PathMode): Promise<void> {
  await learnCall("learn.path.mode", { id, mode });
}

/** setProfile 改学习者档案。只传要改的字段。 */
export async function setProfile(id: string, patch: Partial<PathProfile>): Promise<void> {
  await learnCall("learn.path.profile", { id, ...patch });
}

/** setRetention 改目标记住率。 */
export async function setRetention(id: string, retention: number): Promise<void> {
  await learnCall("learn.path.retention", { id, retention });
}

/** claim 声明会了，或用 on 为假撤回。 */
export async function claim(id: string, pointId: string, on: boolean): Promise<void> {
  await learnCall("learn.path.claim", { id, point_id: pointId, on });
}

/** defer 暂缓一个知识点，或用 on 为假恢复。 */
export async function defer(id: string, pointId: string, on: boolean): Promise<void> {
  await learnCall("learn.path.defer", { id, point_id: pointId, on });
}

/** answer 提交对待答题的作答。giveUp 为真表示不会。session 决定判分用哪段对话的模型。 */
export async function answer(id: string, questionId: string, text: string, opts: { giveUp?: boolean; session?: string } = {}): Promise<GradeResult> {
  return learnCall("learn.path.answer", { id, question_id: questionId, answer: text, give_up: !!opts.giveUp, session: opts.session || "" });
}

/** skipQuestion 放弃待答的题，不计分。 */
export async function skipQuestion(id: string): Promise<void> {
  await learnCall("learn.path.skip", { id });
}

/** questionState 读取一道题现在的状态。 */
export async function questionState(id: string, questionId: string): Promise<QuestionState> {
  return learnCall("learn.path.question", { id, question_id: questionId });
}

/** pointDetail 读取一个知识点的详情。 */
export async function pointDetail(id: string, pointId: string): Promise<PointDetail> {
  const res = await learnCall("learn.path.point", { id, point_id: pointId });
  return { ...res, evidence: list(res?.evidence) };
}
