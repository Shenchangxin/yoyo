export const TOPIC_FILTERS = ["", "inbox", "ready", "scheduled", "in_production", "done"] as const;
export type TopicFilter = (typeof TOPIC_FILTERS)[number];

export const SCORE_KEYS = ["heat", "timing", "uniqueness", "feasibility", "persona", "conversion", "risk"] as const;
export type ScoreKey = (typeof SCORE_KEYS)[number];

export type TopicCard = {
  id: string;
  title: string;
  status?: string;
  studio_project_id?: string;
  scores?: Record<string, number>;
  angle?: string;
  persona_page_id?: string;
};

export function topicScoreAvg(scores?: Record<string, number>): number {
  const vals = Object.values(scores || {}).filter((n) => Number.isFinite(n) && n > 0);
  if (!vals.length) return 0;
  return Math.round((vals.reduce((a, b) => a + b, 0) / vals.length) * 10) / 10;
}

export function topicHasShort(t: Pick<TopicCard, "studio_project_id">): boolean {
  return !!String(t.studio_project_id || "").trim();
}

export function topicPrimaryAction(t: Pick<TopicCard, "studio_project_id">): "open" | "produce" {
  return topicHasShort(t) ? "open" : "produce";
}

export function filmInFlight(phase?: string): boolean {
  return phase === "queued" || phase === "script" || phase === "voice" || phase === "visuals" || phase === "compose";
}

export function topicCanFilm(t: Pick<TopicCard, "studio_project_id">, film?: { auto_phase?: string; result_hash?: string }): boolean {
  if (!topicHasShort(t)) return true;
  if (!film) return false;
  return film.auto_phase === "failed" && !film.result_hash;
}

export function topicIsScheduled(t: Pick<TopicCard, "status" | "studio_project_id">): boolean {
  return t.status === "scheduled" && !topicHasShort(t);
}

export function cycleScore(n: number): number {
  const cur = Number.isFinite(n) ? Math.max(0, Math.min(5, Math.round(n))) : 0;
  return cur >= 5 ? 0 : cur + 1;
}

export function setScore(scores: Record<string, number> | undefined, key: string, n: number): Record<string, number> {
  const cur = Number(scores?.[key] || 0);
  const v = Number.isFinite(n) ? Math.max(0, Math.min(5, Math.round(n))) : 0;
  const next = { ...(scores || {}) };
  if (!v || v === cur) delete next[key];
  else next[key] = v;
  return next;
}

function hay(parts: Array<string | undefined>) {
  return parts.map((x) => String(x || "")).join(" ").toLowerCase();
}

export function topicMatchesQuery(t: { title?: string; angle?: string; note?: string; source_title?: string; source_platform?: string }, q: string) {
  const needle = q.trim().toLowerCase();
  if (!needle) return true;
  return hay([t.title, t.angle, t.note, t.source_title, t.source_platform]).includes(needle);
}

export function trendMatchesQuery(item: { title?: string; hot?: string }, q: string) {
  const needle = q.trim().toLowerCase();
  if (!needle) return true;
  return hay([item.title, item.hot]).includes(needle);
}
