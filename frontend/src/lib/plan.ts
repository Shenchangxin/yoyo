import type { Item } from "./protocol";
import { isPlanTool, isToolFailed, toolArgs, toolName, toolResultBody } from "./tool-summary";

/**
 * 拥有：从工具参数或正文解析任务计划，并在转录里取最近一次成功的计划。
 * 不拥有：计划卡片的渲染，以及 update_plan 的执行。
 */

/** PlanStatus 计划步骤的归一状态：未开始、进行中或已完成。 */
export type PlanStatus = "pending" | "in_progress" | "complete";

/** PlanStep 计划中的一步及其状态。 */
export type PlanStep = {
  step: string;
  status: PlanStatus;
};

/** TaskPlan 一次计划的说明和步骤。解析阶段 id 留空，由转录里的工具调用补上。 */
export type TaskPlan = {
  id: string;
  explanation: string;
  steps: PlanStep[];
};

/** 带方括号状态的编号行。 */
const STEP_MARKED = /^\s*\d+\.\s*\[([^\]]+)\]\s+(.+?)\s*$/;
/** 只有编号、没有状态的行。 */
const STEP_NUMBERED = /^\s*\d+\.\s+(.+?)\s*$/;

/** normalizePlanStatus 把完成、进行中的各种写法收成三种状态，其余视为未开始。 */
export function normalizePlanStatus(raw: string): PlanStatus {
  const s = raw.trim().toLowerCase().replace(/[\s-]+/g, "_");
  if (s === "complete" || s === "completed" || s === "done" || s === "finished") return "complete";
  if (s === "in_progress" || s === "inprogress" || s === "active" || s === "running") return "in_progress";
  return "pending";
}

/** parsePlanArgs 从结构化的 plan 参数里读出步骤。没有有效步骤时返回空。 */
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

/** parsePlanText 从带编号的正文里读出步骤。带状态的行优先于纯编号行，步骤之前的文字当作说明。 */
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

/** planDoneCount 统计已完成的步骤数。 */
export function planDoneCount(plan: TaskPlan): number {
  return plan.steps.filter((s) => s.status === "complete").length;
}

/** planFocus 操作者该看的一步：优先进行中，否则下一个未开始。 */
export function planFocus(plan: TaskPlan): PlanStep | undefined {
  return plan.steps.find((s) => s.status === "in_progress") || plan.steps.find((s) => s.status === "pending");
}

/** 取出计划事件或工具结果里的正文。 */
function planTextOf(it: Item): string {
  if (it.type === "plan") return (it.text || String(it.payload?.text || "")).trim();
  if (it.type === "tool_result") return toolResultBody(it).trim();
  return "";
}

/** 判断条目是计划事件，或名为计划工具的调用与结果。 */
function isPlanEvent(it: Item): boolean {
  return it.type === "plan" || (isPlanTool(toolName(it)) && (it.type === "tool_call" || it.type === "tool_result"));
}

/**
 * latestTaskPlan 取转录里最后一次成功的计划。
 * 后到的操作者消息仍保留它，只有更新的计划会替换。引导备注不开新回合。优先用结构化参数。
 */
export function latestTaskPlan(items: Item[]): TaskPlan | null {
  const failed = new Set<string>();
  for (let i = 0; i < items.length; i++) {
    const it = items[i];
    if (it.type === "tool_result" && isPlanTool(toolName(it)) && isToolFailed(it)) {
      failed.add(String(it.payload?.id || it.key));
    }
  }
  let hit: Item | undefined;
  for (let i = items.length - 1; i >= 0; i--) {
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
  for (let i = 0; i < items.length; i++) {
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
