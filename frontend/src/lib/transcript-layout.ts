import type { Item } from "./protocol";
import {
  isOutcomeToolName,
  isToolFailed,
  itemTimeMs,
  toolArgs,
  toolElapsedMs,
  toolKind,
  toolName,
  type ToolKind,
} from "./tool-summary";

export type ToolPair = { key: string; call?: Item; result?: Item; extra: Item[] };

export type AgentPart =
  | { key: string; kind: "item"; item: Item }
  | { key: string; kind: "process"; items: Item[]; live: boolean }
  | { key: string; kind: "artifact"; items: Item[] };

export type LayoutRow =
  | { key: string; kind: "user"; item: Item }
  | { key: string; kind: "agent"; parts: AgentPart[]; copyText: string }
  | { key: string; kind: "solo"; item: Item };

export function isToolish(it: Item): boolean {
  return it.type === "tool_call" || it.type === "tool_result";
}

export function isSteer(it: Item): boolean {
  return it.type === "user" && it.source === "steer";
}

/** Duplicate of a tool already in the turn — keep in the agent block, never render. */
export function isTranscriptDuplicate(it: Item): boolean {
  return it.type === "plan" || it.type === "file_change" || it.type === "ask_user";
}

export function isOutcomeTool(it: Item): boolean {
  if (!isToolish(it)) return false;
  const name = toolName(it);
  const body = it.type === "tool_result" ? String(it.payload?.content || it.text || "") : "";
  const path = String(toolArgs(it).path || "");
  return isOutcomeToolName(name, body, path);
}

export function isProcessItem(it: Item): boolean {
  if (it.type === "reasoning" || it.type === "subagent" || it.type === "context_injection") return true;
  return isToolish(it) && !isOutcomeTool(it);
}

export function isAgentItem(it: Item): boolean {
  return (
    it.type === "assistant" ||
    it.type === "reasoning" ||
    it.type === "subagent" ||
    isToolish(it) ||
    it.type === "context_injection" ||
    it.type === "plan" ||
    it.type === "file_change" ||
    it.type === "ask_user" ||
    isSteer(it)
  );
}

export function pairTools(items: Item[]): ToolPair[] {
  const order: string[] = [];
  const map = new Map<string, ToolPair>();
  for (const it of items) {
    const id = String(it.payload?.id || (it.type === "context_injection" ? it.key : "") || it.key);
    if (!map.has(id)) {
      map.set(id, { key: id, extra: [] });
      order.push(id);
    }
    const row = map.get(id)!;
    if (it.type === "tool_call") row.call = it;
    else if (it.type === "tool_result") row.result = it;
    else row.extra.push(it);
  }
  return order.map((id) => map.get(id)!);
}

export function processGroupLive(items: Item[]): boolean {
  const open = new Set<string>();
  for (const it of items) {
    const id = String(it.payload?.id || it.key);
    if (it.type === "tool_call") open.add(id);
    if (it.type === "tool_result") open.delete(id);
  }
  return open.size > 0;
}

export function processToolPairs(items: Item[]): ToolPair[] {
  return pairTools(items).filter((p) => p.call || p.result);
}

export function processElapsedMs(items: Item[]): number {
  return processToolPairs(items).reduce((sum, p) => sum + (p.result ? toolElapsedMs(p.result) : 0), 0);
}

/** Earliest timestamp in the group — the "Worked for" clock starts here. */
export function processStartMs(items: Item[]): number {
  let min = 0;
  for (const it of items) {
    const t = itemTimeMs(it);
    if (t && (!min || t < min)) min = t;
  }
  return min;
}

/**
 * Wall-clock span of a settled group. Model thinking between tool batches is
 * part of the work, so this beats summing per-tool elapsed_ms; fall back to
 * the sum when timestamps are missing (old JSONL, mocks).
 */
export function processSpanMs(items: Item[]): number {
  const start = processStartMs(items);
  let end = 0;
  for (const it of items) {
    const t = itemTimeMs(it);
    if (t > end) end = t;
  }
  const span = start && end > start ? end - start : 0;
  return span || processElapsedMs(items);
}

export function processFailedCount(items: Item[]): number {
  return processToolPairs(items).filter((p) => isToolFailed(p.result)).length;
}

export type PairState = "running" | "done" | "failed" | "interrupted";

/**
 * A call without a result is only "running" while the thread is. Once the
 * turn is over the honest state is "interrupted" — never a spinner on a
 * finished turn (the bug WorkBuddy shipped a fix for twice, and Codex once).
 */
export function pairState(pair: ToolPair, running: boolean): PairState {
  if (pair.call && !pair.result) return running ? "running" : "interrupted";
  if (isToolFailed(pair.result)) return "failed";
  return "done";
}

export type KindTally = Partial<Record<ToolKind, number>>;

export function processKindTally(items: Item[]): KindTally {
  const tally: KindTally = {};
  for (const p of processToolPairs(items)) {
    const k = toolKind(toolName(p.call || p.result!));
    tally[k] = (tally[k] || 0) + 1;
  }
  return tally;
}

export function processHasReasoning(items: Item[]): boolean {
  return items.some((it) => it.type === "reasoning");
}

function agentCopyText(items: Item[]): string {
  return items
    .filter((it) => it.type === "assistant")
    .map((it) => it.text.trim())
    .filter(Boolean)
    .join("\n\n");
}

export function layoutAgentParts(items: Item[]): AgentPart[] {
  const parts: AgentPart[] = [];
  let process: Item[] = [];
  let artifacts: Item[] = [];
  const flushProcess = () => {
    if (!process.length) return;
    parts.push({
      key: `process:${process[0].key}`,
      kind: "process",
      items: process,
      live: processGroupLive(process),
    });
    process = [];
  };
  const flushArtifacts = () => {
    if (!artifacts.length) return;
    parts.push({ key: `artifact:${artifacts[0].key}`, kind: "artifact", items: artifacts });
    artifacts = [];
  };
  for (const it of items) {
    if (isTranscriptDuplicate(it)) continue;
    if (isProcessItem(it)) {
      flushArtifacts();
      process.push(it);
      continue;
    }
    if (isOutcomeTool(it)) {
      flushProcess();
      artifacts.push(it);
      continue;
    }
    flushProcess();
    flushArtifacts();
    parts.push({ key: it.key, kind: "item", item: it });
  }
  flushProcess();
  flushArtifacts();
  return parts;
}

/** One visual row per operator turn. Agent text/tools between users stay together. */
export function layoutRows(items: Item[]): LayoutRow[] {
  const rows: LayoutRow[] = [];
  let agent: Item[] = [];
  const flushAgent = () => {
    if (!agent.length) return;
    rows.push({
      key: `turn:${agent[0].key}`,
      kind: "agent",
      parts: layoutAgentParts(agent),
      copyText: agentCopyText(agent),
    });
    agent = [];
  };
  for (const it of items) {
    if (it.type === "user" && it.source !== "steer") {
      flushAgent();
      rows.push({ key: it.key, kind: "user", item: it });
      continue;
    }
    if (isAgentItem(it)) {
      agent.push(it);
      continue;
    }
    flushAgent();
    rows.push({ key: it.key, kind: "solo", item: it });
  }
  flushAgent();
  return rows;
}
