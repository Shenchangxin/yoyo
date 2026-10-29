/**
 * 拥有：学习表面里一条助手对话的范围。今日、练习、每条路径、每份材料各自一把钥匙。
 * 不拥有：会话怎么创建。调用方拿钥匙去找或新建学习频道的会话。
 */

/** LearnLane 是学习表面的四条泳道。 */
export type LearnLane = "today" | "study" | "library" | "practice";

/** LearnScopeSpec 描述该由哪一段学习对话来接。scope 是持久化钥匙。 */
export type LearnScopeSpec = {
  scope: string;
  title?: string;
  pathId?: string;
  libraryId?: string;
  materialId?: string;
};

/** 读出范围到会话 id 的映射。坏掉的条目丢掉。raw 为空或解析失败时返回空对象。 */
export function readLearnScopeMap(raw: string | null): Record<string, string> {
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object") return {};
    const out: Record<string, string> = {};
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === "string" && value) out[key] = value;
    }
    return out;
  } catch {
    return {};
  }
}

/** learnScopeSpec 按泳道和当前选中项收成一把钥匙。学习泳道优先用路径，文库里的笔记、材料和书各用各的。lane 为 today 时返回 today，为 practice 时返回 practice，为 study 且有 pathId 时钥匙带上路径。 */
export function learnScopeSpec(input: {
  lane: LearnLane;
  title?: string;
  pathId?: string;
  pathName?: string;
  libraryId?: string;
  libraryName?: string;
  materialId?: string;
  materialTitle?: string;
  bookId?: string;
  bookTitle?: string;
  shelf?: "materials" | "books" | "notes";
}): LearnScopeSpec {
  if (input.lane === "today") return { scope: "today", title: input.title };
  if (input.lane === "practice") return { scope: "practice", title: input.title };
  if (input.lane === "study") {
    if (input.pathId) {
      return {
        scope: `path:${input.pathId}`,
        title: input.pathName || input.title,
        pathId: input.pathId,
        libraryId: input.libraryId || undefined,
      };
    }
    return { scope: "study", title: input.title };
  }
  if (input.shelf === "notes" || input.libraryId === "notes") return { scope: "notes", title: input.title || input.libraryName };
  if (input.materialId) {
    return {
      scope: `material:${input.materialId}`,
      title: input.materialTitle || input.title,
      materialId: input.materialId,
      libraryId: input.libraryId || undefined,
    };
  }
  if (input.bookId) return { scope: `book:${input.bookId}`, title: input.bookTitle || input.title };
  if (input.libraryId) return { scope: `library:${input.libraryId}`, title: input.libraryName || input.title, libraryId: input.libraryId };
  return { scope: "library", title: input.title };
}

/** 绑定对得上才算同一段对话。没有路径、材料和文库的范围，不能占用已经绑过的会话。thread 读取 pathId、materialId、libraryId。spec 读取 pathId、materialId、libraryId。 */
export function learnBindingMatches(
  thread: { pathId?: string; libraryId?: string; materialId?: string },
  spec: LearnScopeSpec,
): boolean {
  if (spec.pathId) return thread.pathId === spec.pathId;
  if (spec.materialId) return thread.materialId === spec.materialId;
  if (spec.libraryId) return thread.libraryId === spec.libraryId && !thread.pathId && !thread.materialId;
  return !thread.pathId && !thread.materialId && !thread.libraryId;
}

/** 先用记住的 id，对不上再按路径、材料或文库找。归档的跳过。list 做过滤。remembered 与 id 比较。 */
export function matchLearnThread<T extends { id: string; pathId?: string; libraryId?: string; materialId?: string; archived?: boolean }>(
  list: T[],
  spec: LearnScopeSpec,
  remembered: string,
): T | null {
  const live = list.filter((t) => !t.archived);
  if (remembered) {
    const hit = live.find((t) => t.id === remembered);
    if (hit && learnBindingMatches(hit, spec)) return hit;
  }
  if (spec.pathId) return live.find((t) => t.pathId === spec.pathId) || null;
  if (spec.materialId) return live.find((t) => t.materialId === spec.materialId) || null;
  if (spec.libraryId) return live.find((t) => t.libraryId === spec.libraryId && !t.pathId && !t.materialId) || null;
  return null;
}
