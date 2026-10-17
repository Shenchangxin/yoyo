export type GameCamera = "tps" | "fps" | "top";

export type GameRow = {
  id: string;
  title?: string;
  intent?: string;
  stage?: string;
  status?: string;
  camera?: string;
  genre?: string;
  preview_html?: string;
  playable?: string;
  ir?: Record<string, unknown>;
  playable_spec?: unknown;
  job_ids?: string[];
  error?: string;
  need_collect?: number;
  verification?: { status?: string; checks?: Record<string, string> };
};

export type GameQA = "pass" | "fail" | "pending" | "";

export type GameStep = "intent" | "build" | "play" | "check";

export type GameNext = "make" | "finish" | "play" | "repair";

export const GAME_CAMERAS: GameCamera[] = ["tps", "fps", "top"];

const STAGE_RANK: Record<string, number> = {
  draft: 0,
  scene: 1,
  rules: 2,
  candidates: 3,
  placed: 4,
  assets: 5,
  compiled: 6,
  verified: 7,
  failed: 0,
};

export function stageRank(stage?: string): number {
  return STAGE_RANK[stage || ""] ?? 0;
}

export function qaStatus(row: GameRow | null | undefined): GameQA {
  if (!row) return "";
  const s = String(row.verification?.status || "").trim().toUpperCase();
  if (s === "PASS" || row.stage === "verified") return "pass";
  if (s === "FAIL" || row.status === "qa_failed") return "fail";
  if (hasPlayable(row) || s === "NOT_RUN") return "pending";
  return "";
}

export function gameStepOf(stage?: string, row?: GameRow | null): GameStep {
  const qa = row ? qaStatus(row) : "";
  if (qa === "pass" || stage === "verified") return "check";
  if (stage === "compiled" || qa === "fail" || qa === "pending") return "play";
  if (!stage || stage === "draft" || stage === "failed") return "intent";
  return "build";
}

export function stepDone(stage: string | undefined, step: GameStep, row?: GameRow | null): boolean {
  const order: GameStep[] = ["intent", "build", "play", "check"];
  if (stage === "failed") return step === "intent";
  return order.indexOf(gameStepOf(stage, row)) > order.indexOf(step);
}

export function hasPlayable(row: GameRow | null): boolean {
  if (!row) return false;
  return !!row.preview_html || !!row.playable || row.stage === "compiled" || row.stage === "verified";
}

export function gameNext(row: GameRow | null): GameNext {
  if (!row) return "make";
  if (row.stage === "failed" || row.status === "failed" || row.error) return "repair";
  if (qaStatus(row) === "fail") return "repair";
  if (hasPlayable(row)) return "play";
  if (row.stage && row.stage !== "draft") return "finish";
  return "make";
}

export function titleFromIntent(intent: string, fallback: string): string {
  const t = intent.trim().replace(/\s+/g, " ");
  if (!t) return fallback;
  return t.length > 32 ? `${t.slice(0, 31)}…` : t;
}

export function normalizeCamera(v?: string): GameCamera {
  if (v === "fps" || v === "top") return v;
  return "tps";
}

export function gameStageLabel(stage: string | undefined, labels: {
  statusDraft: string;
  statusScene: string;
  statusRules: string;
  statusPlaced: string;
  statusAssets: string;
  statusCompiled: string;
  statusVerified: string;
  statusFailed: string;
}): string {
  if (stage === "scene") return labels.statusScene;
  if (stage === "rules") return labels.statusRules;
  if (stage === "placed" || stage === "candidates") return labels.statusPlaced;
  if (stage === "assets") return labels.statusAssets;
  if (stage === "compiled") return labels.statusCompiled;
  if (stage === "verified") return labels.statusVerified;
  if (stage === "failed") return labels.statusFailed;
  return labels.statusDraft;
}

export type GameInspect = {
  objective: string;
  win: string;
  fail: string;
  camera: string;
  need: number;
  elements: { id: string; type: string }[];
  interactions: { id: string; trigger: string }[];
  jobs: string[];
};

export function inspectGame(row: GameRow): GameInspect {
  const spec = asObj(row.playable_spec);
  const ir = asObj(row.ir);
  const rules = asObj(ir.rules);
  const elements = asArr(spec.elements).map((el) => ({
    id: str(el.id),
    type: str(el.type),
  })).filter((el) => el.id || el.type);
  const interactions = asArr(spec.interactions || ir.interactions).map((it) => ({
    id: str(it.id),
    trigger: str(it.trigger),
  })).filter((it) => it.id || it.trigger);
  return {
    objective: str(spec.objective || rules.objective),
    win: str(spec.win || spec.win_condition || rules.win || rules.win_condition),
    fail: str(spec.fail || spec.fail_condition || rules.fail || rules.fail_condition),
    camera: str(spec.camera || row.camera || "tps"),
    need: num(spec.need_collect ?? row.need_collect),
    elements,
    interactions,
    jobs: row.job_ids || [],
  };
}

function asObj(v: unknown): Record<string, unknown> {
  return v && typeof v === "object" && !Array.isArray(v) ? v as Record<string, unknown> : {};
}

function asArr(v: unknown): Record<string, unknown>[] {
  return Array.isArray(v) ? v.filter((x) => x && typeof x === "object") as Record<string, unknown>[] : [];
}

function str(v: unknown): string {
  return v == null ? "" : String(v).trim();
}

function num(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}
