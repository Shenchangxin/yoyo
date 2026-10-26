/**
 * 本文件拥有选题筛选、打分和能否开拍的纯判断。不拥有选题列表界面。
 * 它不请求服务端，也不改服务端状态。
 */
/** TOPIC_FILTERS 是选题列表可用的状态筛，第一项是空字符串。 */
export const TOPIC_FILTERS = ["", "inbox", "ready", "scheduled", "in_production", "done"] as const;
/** TopicFilter 是 TOPIC_FILTERS 里的一项。 */
export type TopicFilter = (typeof TOPIC_FILTERS)[number];

/** SCORE_KEYS 是选题卡上的七个分数键。 */
export const SCORE_KEYS = ["heat", "timing", "uniqueness", "feasibility", "persona", "conversion", "risk"] as const;
/** ScoreKey 是 SCORE_KEYS 里的一项。 */
export type ScoreKey = (typeof SCORE_KEYS)[number];

/** TopicCard 是判断选题时用到的标题、状态、分数和已绑定短片。 */
export type TopicCard = {
  id: string;
  title: string;
  status?: string;
  studio_project_id?: string;
  scores?: Record<string, number>;
  angle?: string;
  persona_page_id?: string;
};

/** topicScoreAvg 只平均大于 0 的有限分数，并保留一位小数。scores 取出各项分数，没有就当空对象。 */
export function topicScoreAvg(scores?: Record<string, number>): number {
  const vals = Object.values(scores || {}).filter(/** n 只留下有限且大于 0 的分数。 */ (n) => Number.isFinite(n) && n > 0);
  if (!vals.length) return 0;
  return Math.round((vals.reduce(/** a 是累加和。b 加进 a。 */ (a, b) => a + b, 0) / vals.length) * 10) / 10;
}

/** topicHasShort 看选题是否已经写了短片 id。t 看 studio_project_id 去掉空白后是否还有内容。 */
export function topicHasShort(t: Pick<TopicCard, "studio_project_id">): boolean {
  return !!String(t.studio_project_id || "").trim();
}

/** topicPrimaryAction 有短片时返回 open，否则返回 produce。t 交给 topicHasShort。 */
export function topicPrimaryAction(t: Pick<TopicCard, "studio_project_id">): "open" | "produce" {
  return topicHasShort(t) ? "open" : "produce";
}

/** filmInFlight 判断自动阶段是不是排队、剧本、声音、画面或合成。phase 和 queued、script、voice、visuals、compose 比较。 */
export function filmInFlight(phase?: string): boolean {
  return phase === "queued" || phase === "script" || phase === "voice" || phase === "visuals" || phase === "compose";
}

/** topicCanFilm 没短片时可以拍；已有短片只在失败且没有成片哈希时可以再拍。t 交给 topicHasShort。film 没有就不能再拍，否则看 auto_phase 是不是 failed，以及有没有 result_hash。 */
export function topicCanFilm(t: Pick<TopicCard, "studio_project_id">, film?: { auto_phase?: string; result_hash?: string }): boolean {
  if (!topicHasShort(t)) return true;
  if (!film) return false;
  return film.auto_phase === "failed" && !film.result_hash;
}

/** topicIsScheduled 只在状态是 scheduled 且还没有短片时为真。t 看 status 是不是 scheduled，并交给 topicHasShort。 */
export function topicIsScheduled(t: Pick<TopicCard, "status" | "studio_project_id">): boolean {
  return t.status === "scheduled" && !topicHasShort(t);
}

/** cycleScore 把分数限制在 0 到 5，5 再点回到 0。n 收成 0 到 5 的整数，满 5 回到 0，否则加 1。 */
export function cycleScore(n: number): number {
  const cur = Number.isFinite(n) ? Math.max(0, Math.min(5, Math.round(n))) : 0;
  return cur >= 5 ? 0 : cur + 1;
}

/** setScore 写入 1 到 5 的分数；再点同一个数或写成 0 就删掉这一键。scores 读出 key 上的旧分并复制一份。key 用来读写这一项。n 收成 0 到 5，0 或和旧分相同就删掉这一键。 */
export function setScore(scores: Record<string, number> | undefined, key: string, n: number): Record<string, number> {
  const cur = Number(scores?.[key] || 0);
  const v = Number.isFinite(n) ? Math.max(0, Math.min(5, Math.round(n))) : 0;
  const next = { ...(scores || {}) };
  if (!v || v === cur) delete next[key];
  else next[key] = v;
  return next;
}

/** 把若干字段拼成小写检索串。parts 每项转成字符串后用空格拼起来再转小写。 */
function hay(parts: Array<string | undefined>) {
  return parts.map(/** x 转成字符串，空值当空字符串。 */ (x) => String(x || "")).join(" ").toLowerCase();
}

/** topicMatchesQuery 在标题、角度、备注和来源里找这段文字。空查询算匹配。t 用 title、angle、note、source_title、source_platform 拼检索串。q 去掉空白并转成小写，空的算匹配。 */
export function topicMatchesQuery(t: { title?: string; angle?: string; note?: string; source_title?: string; source_platform?: string }, q: string) {
  const needle = q.trim().toLowerCase();
  if (!needle) return true;
  return hay([t.title, t.angle, t.note, t.source_title, t.source_platform]).includes(needle);
}

/** trendMatchesQuery 在热榜标题和热度里找这段文字。空查询算匹配。item 用 title 和 hot 拼检索串。q 去掉空白并转成小写，空的算匹配。 */
export function trendMatchesQuery(item: { title?: string; hot?: string }, q: string) {
  const needle = q.trim().toLowerCase();
  if (!needle) return true;
  return hay([item.title, item.hot]).includes(needle);
}
