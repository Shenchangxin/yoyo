import type { Item } from "./protocol";
import { isPlanTool, isToolFailed, toolArgs, toolName, toolResultBody } from "./tool-summary";

export type PlanStatus = "pending" | "in_progress" | "complete";

export type PlanStep = {
  step: string;
  status: PlanStatus;
};

export type TaskPlan = {
  id: string;
  explanation: string;
  steps: PlanStep[];
};

const STEP_MARKED = /^\s*\d+\.\s*\[([^\]]+)\]\s+(.+?)\s*$/;
const STEP_NUMBERED = /^\s*\d+\.\s+(.+?)\s*$/;

export function normalizePlanStatus(raw: string): PlanStatus {
  const s = raw.trim().toLowerCase().replace(/[\s-]+/g, "_");
  if (s === "complete" || s === "completed" || s === "done" || s === "finished") return "complete";
  if (s === "in_progress" || s === "inprogress" || s === "active" || s === "running") return "in_progress";
  return "pending";
}

export function parsePlanArgs(args: Record<string, unknown>): TaskPlan | null {
  const rows = args.plan;
  if (!Array.isArray(rows) || rows.length === 0) return null;
  const steps: PlanStep[] = [];
  for (const row of rows) {
    if (!row || typeof row !== "object") continue;
    const rec = row as Record<string, unknown>;
    const step = String(rec.step || "").trim();
    if (!step) continue;
    steps.push({ step, status: normalizePlanStatus(String(rec.status || "pending")) });
  }
  if (!steps.length) return null;
  return { id: "", explanation: String(args.explanation || "").trim(), steps };
}

export function parsePlanText(text: string): TaskPlan | null {
  const raw = text.trim();
  if (!raw) return null;
  const lines = raw.split(/\r?\n/);
  const marked: PlanStep[] = [];
  const numbered: PlanStep[] = [];
  let firstStep = -1;
  for (let i = 0; i < lines.length; i++) {
    const markedHit = lines[i].match(STEP_MARKED);
    if (markedHit) {
      if (firstStep < 0) firstStep = i;
      marked.push({ status: normalizePlanStatus(markedHit[1]), step: markedHit[2].trim() });
      continue;
    }
    const numberedHit = lines[i].match(STEP_NUMBERED);
    if (numberedHit) {
      if (firstStep < 0) firstStep = i;
      numbered.push({ status: "pending", step: numberedHit[1].trim() });
    }
  }
  const steps = marked.length ? marked : numbered;
  if (!steps.length) return null;
  const explanation = (firstStep >= 0 ? lines.slice(0, firstStep) : []).join("\n").trim();
  return { id: "", explanation, steps };
}

export function planDoneCount(plan: TaskPlan): number {
  return plan.steps.filter((s) => s.status === "complete").length;
}

/** The step the operator should look at: current work, else the next pending. */
export function planFocus(plan: TaskPlan): PlanStep | undefined {
  return plan.steps.find((s) => s.status === "in_progress") || plan.steps.find((s) => s.status === "pending");
}

function planTextOf(it: Item): string {
  if (it.type === "plan") return (it.text || String(it.payload?.text || "")).trim();
  if (it.type === "tool_result") return toolResultBody(it).trim();
  return "";
}

function isPlanEvent(it: Item): boolean {
  return it.type === "plan" || (isPlanTool(toolName(it)) && (it.type === "tool_call" || it.type === "tool_result"));
}

function lastOperatorIndex(items: Item[]): number {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "user" && items[i].source !== "steer") return i;
  }
  return -1;
}

/**
 * Current-turn plan: last successful update_plan (or in-flight call) after
 * the latest operator message. Steer notes do not start a turn.
 * Prefers structured tool arguments; falls back to the formatted tool body.
 */
export function latestTaskPlan(items: Item[]): TaskPlan | null {
  const start = lastOperatorIndex(items) + 1;
  const failed = new Set<string>();
  for (let i = start; i < items.length; i++) {
    const it = items[i];
    if (it.type === "tool_result" && isPlanTool(toolName(it)) && isToolFailed(it)) {
      failed.add(String(it.payload?.id || it.key));
    }
  }
  let hit: Item | undefined;
  for (let i = items.length - 1; i >= start; i--) {
    const it = items[i];
    if (!isPlanEvent(it)) continue;
    const id = String(it.payload?.id || it.key);
    if (failed.has(id) || (it.type === "tool_result" && isToolFailed(it))) continue;
    hit = it;
    break;
  }
  if (!hit) return null;
  const id = String(hit.payload?.id || hit.key);
  let args: Record<string, unknown> = {};
  let text = "";
  for (let i = start; i < items.length; i++) {
    const it = items[i];
    if (String(it.payload?.id || it.key) !== id && String(it.payload?.id || "") !== id) continue;
    if (it.type === "tool_call" && isPlanTool(toolName(it))) args = toolArgs(it);
    const body = planTextOf(it);
    if (body) text = body;
  }
  const parsed = parsePlanArgs(args) || parsePlanText(text);
  if (!parsed) return null;
  return { ...parsed, id };
}
