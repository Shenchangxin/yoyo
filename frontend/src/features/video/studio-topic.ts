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

export function topicIsScheduled(t: Pick<TopicCard, "status" | "studio_project_id">): boolean {
  return t.status === "scheduled" && !topicHasShort(t);
}

export function cycleScore(n: number): number {
  const cur = Number.isFinite(n) ? Math.max(0, Math.min(5, Math.round(n))) : 0;
  return cur >= 5 ? 0 : cur + 1;
}
