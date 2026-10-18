import type { Item } from "./protocol";
import { itemTurnKey } from "./turn-outline";
import { processGrouped, turnWorkFolded, type ChatPresentationPolicy } from "./presentation-policy";
import {
  isArtifactTool,
  isRichResult,
  isToolFailed,
  itemTimeMs,
  toolArgs,
  toolElapsedMs,
  toolKind,
  toolName,
  toolResultUnchanged,
  type ToolKind,
} from "./tool-summary";
import { artifactShouldShow, artifactView } from "./artifact-preview";
import { looksLikeHTMLFile } from "./html-preview";

export type ToolPair = { key: string; call?: Item; result?: Item; extra: Item[] };

export type LayoutOpts = {
  running?: boolean;
  showThinking?: boolean;
  policy?: ChatPresentationPolicy;
  nextUser?: boolean;
};

export type AgentPart =
  | { key: string; kind: "item"; item: Item }
  | { key: string; kind: "process"; items: Item[]; live: boolean; grouped?: boolean }
  | { key: string; kind: "artifact"; items: Item[] }
  | { key: string; kind: "subagent"; items: Item[]; child: string; live: boolean }
  | { key: string; kind: "stream"; item: Item }
  | { key: string; kind: "work"; parts: AgentPart[]; live: boolean; folded: boolean };

export type LayoutRow =
  | { key: string; kind: "user"; item: Item; turnKey: string }
  | { key: string; kind: "agent"; parts: AgentPart[]; copyText: string; turnKey: string }
  | { key: string; kind: "solo"; item: Item };

export function isToolish(it: Item): boolean {
  return it.type === "tool_call" || it.type === "tool_result";
}

export function isSteer(it: Item): boolean {
  return it.type === "user" && it.source === "steer";
}

/** @mention inject is folded into the user bubble; do not render it as a turn. */
export function isMentionInject(it: Item): boolean {
  return it.type === "context_injection" && it.source !== "skill";
}

export function liveAutoSkills(items: Item[] | undefined | null): { name: string; loaded: boolean }[] {
  if (!items?.length) return [];
  const out: { name: string; loaded: boolean }[] = [];
  const seen = new Set<string>();
  for (const it of items) {
    if (it.type !== "context_injection" || it.source !== "skill") continue;
    if (!it.payload?.auto) continue;
    const name = String(it.payload?.name || "").trim();
    if (!name || name === "load_skill" || seen.has(name)) continue;
    seen.add(name);
    out.push({ name, loaded: true });
  }
  return out;
}

/** Duplicate of a tool already in the turn — keep in the agent block, never render. */
export function isTranscriptDuplicate(it: Item): boolean {
  if (it.type === "plan" || it.type === "file_change" || it.type === "ask_user") return true;
  // Successful task calls are the SubagentCard; keep a failed result visible.
  if (isToolish(it) && toolName(it) === "task") {
    return !(it.type === "tool_result" && isToolFailed(it));
  }
  return false;
}

/** Artifacts the operator acts on. `update_plan` stays in the process rail. */
export function isOutcomeTool(it: Item): boolean {
  if (!isToolish(it)) return false;
  return pairShowsArtifact({
    key: String(it.payload?.id || it.key),
    call: it.type === "tool_call" ? it : undefined,
    result: it.type === "tool_result" ? it : undefined,
    extra: [],
  });
}

export function pairToolPath(pair: ToolPair): string {
  const args = {
    ...(pair.result ? toolArgs(pair.result) : {}),
    ...(pair.call ? toolArgs(pair.call) : {}),
  };
  return String(args.path || args.file || "").replace(/\\/g, "/").replace(/^\.\//, "");
}

export function pairShowsArtifact(pair: ToolPair): boolean {
  const item = pair.call || pair.result || pair.extra[0];
  if (!item) return false;
  const name = toolName(item);
  if (name === "browser_open" || name === "browser_takeover" || name === "read_file") return false;
  if (pair.result && isToolFailed(pair.result)) return false;
  if (toolResultUnchanged(pair.result)) return false;
  const path = pairToolPath(pair);
  if (name === "cite_sources" || name === "review_page" || name === "browser_screenshot" || name === "browser_snapshot" || name === "browser_verify") return true;
  if (name.startsWith("office_")) return /\.(docx|xlsx|pptx|pdf)$/i.test(path);
  if (name === "write_file" || name === "create_file" || name === "str_replace" || name === "edit_file" || name === "apply_patch") {
    return true;
  }
  if (artifactShouldShow(artifactView(pair.call, pair.result))) return true;
  const body = String((pair.result || item).payload?.content || (pair.result || item).text || "");
  if (name === "apply_patch" && (/\*\*\*\s+(Add|Update) File:/.test(body) || !!path)) return true;
  if (isArtifactTool(name) && looksLikeHTMLFile(path)) return true;
  return isRichResult(name, body, path);
}

export function classifyToolPairs(pairs: ToolPair[]): Map<string, "artifact" | "process"> {
  const kind = new Map<string, "artifact" | "process">();
  for (const p of pairs) {
    kind.set(p.key, pairShowsArtifact(p) ? "artifact" : "process");
  }
  return kind;
}

export function isProcessItem(it: Item): boolean {
  if (isMentionInject(it)) return false;
  if (it.type === "reasoning" || it.type === "context_injection") return true;
  if (it.type === "subagent") return false;
  return isToolish(it) && !isOutcomeTool(it);
}

export function subagentChildId(it: Item): string {
  return String(it.payload?.child || "").trim();
}

export function subagentLive(items: Item[]): boolean {
  if (!items.length) return false;
  const last = items[items.length - 1];
  const phase = String(last.payload?.phase || "").trim();
  return phase !== "complete";
}

/** Prompt of the live subagent, for the session header breadcrumb. */
export function liveSubagentCrumb(items: Item[] | undefined | null): string {
  if (!items?.length) return "";
  const groups = new Map<string, Item[]>();
  for (const it of items) {
    if (it.type !== "subagent") continue;
    const child = subagentChildId(it) || it.key;
    if (!child) continue;
    const list = groups.get(child) || [];
    list.push(it);
    groups.set(child, list);
  }
  let crumb = "";
  for (const list of groups.values()) {
    if (!subagentLive(list)) continue;
    for (const it of list) {
      const raw = String(it.payload?.prompt || it.payload?.summary || it.text || "").trim();
      if (!raw) continue;
      crumb = raw.split(/\r?\n/).find((l) => l.trim())?.trim() || raw;
      break;
    }
  }
  return crumb;
}

export function isAgentItem(it: Item): boolean {
  if (isMentionInject(it)) return false;
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
  let streaming = false;
  for (const it of items) {
    const id = String(it.payload?.id || it.key);
    if (it.type === "tool_call") open.add(id);
    if (it.type === "tool_result") {
      if (it.delta || it.payload?.delta) continue;
      open.delete(id);
    }
    if ((it.type === "assistant" || it.type === "reasoning") && (it.delta || it.payload?.streaming)) {
      streaming = true;
    }
  }
  return streaming || open.size > 0;
}

export function processToolPairs(items: Item[]): ToolPair[] {
  return pairTools(items).filter((p) => p.call || p.result);
}

/** Expand a process rail from the tail so a 200-step turn does not mount every row. */
export const PROCESS_PAGE = 60;
/** Artifact cards mount from the tail; earlier files stay one click away. */
export const ARTIFACT_PAGE = 12;
/** Compact tool output in the rail — Codex uses 5; full body stays on disk/spill. */
export const TOOL_PREVIEW_LINES = 5;

/** Head + tail preview so a 4k shell dump does not mount as a wall of text. */
export function clipPreview(text: string, maxLines = TOOL_PREVIEW_LINES): { text: string; hidden: number } {
  if (!text) return { text: "", hidden: 0 };
  const lines = text.split("\n");
  if (lines.length <= maxLines) return { text, hidden: 0 };
  const head = Math.ceil(maxLines / 2);
  const tail = maxLines - head;
  const hidden = lines.length - head - tail;
  return {
    text: `${lines.slice(0, head).join("\n")}\n…\n${lines.slice(-tail).join("\n")}`,
    hidden,
  };
}

export function tailProcessPairs(pairs: ToolPair[], shown: number): { visible: ToolPair[]; hidden: number } {
  if (shown <= 0 || pairs.length <= shown) return { visible: pairs, hidden: 0 };
  return { visible: pairs.slice(pairs.length - shown), hidden: pairs.length - shown };
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
  if (pair.result && (pair.result.delta || pair.result.payload?.delta)) {
    return running ? "running" : "interrupted";
  }
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

export function processProgressCount(items: Item[]): number {
  let n = 0;
  for (const it of items) {
    if (it.type === "assistant") n++;
  }
  return n;
}

function agentCopyText(items: Item[]): string {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "assistant" && items[i].text.trim()) return items[i].text.trim();
  }
  return "";
}

function blockText(it: Item, kind: "text" | "reasoning"): string {
  const blocks = it.payload?.blocks;
  if (!Array.isArray(blocks)) return kind === "text" ? it.text : "";
  let s = "";
  for (const b of blocks) {
    if (b && b.type === kind) s += String(b.text || "");
  }
  return s;
}

export function reasoningText(it: Item): string {
  if (it.type === "reasoning") return it.text.trim();
  const tagged = String(it.payload?.reasoning || "").trim();
  if (tagged) return tagged;
  return blockText(it, "reasoning").trim();
}

export function flattenWorkProcess(parts: AgentPart[]): Item[] {
  const out: Item[] = [];
  for (const p of parts) {
    if (p.kind === "process") out.push(...p.items);
    else if (p.kind === "item" || p.kind === "stream") out.push(p.item);
    else if (p.kind === "subagent") out.push(...p.items);
    else if (p.kind === "work") out.push(...flattenWorkProcess(p.parts));
  }
  return out;
}

function lastCompactAnswerIndex(parts: AgentPart[]): number {
  for (let i = parts.length - 1; i >= 0; i--) {
    const p = parts[i];
    if ((p.kind === "item" || p.kind === "stream") && p.item.type === "assistant") {
      const workAfter = parts.slice(i + 1).some((x) => x.kind === "process" || x.kind === "artifact" || x.kind === "subagent");
      if (!workAfter) return i;
    }
  }
  return -1;
}

export function wrapTurnWork(parts: AgentPart[], opts?: LayoutOpts): AgentPart[] {
  if (!parts.length) return parts;
  const last = lastCompactAnswerIndex(parts);
  const before = last >= 0 ? parts.slice(0, last) : parts;
  const after = last >= 0 ? parts.slice(last) : [];
  if (!before.length) return parts;
  const grouped = opts?.policy ? processGrouped(opts.policy, !!opts.running, !!opts.nextUser) : true;
  const folded = opts?.policy ? turnWorkFolded(opts.policy, !!opts.running, !!opts.nextUser) : false;
  const tagged = before.map((p) => (p.kind === "process" ? { ...p, grouped } : p));
  return [
    { key: `work:${before[0].key}`, kind: "work", parts: tagged, live: !!opts?.running, folded },
    ...after,
  ];
}

export function hasAssistantReasoning(it: Item): boolean {
  if (it.type === "reasoning") return it.text.trim() !== "";
  if (it.type !== "assistant") return false;
  if (String(it.payload?.reasoning || "").trim()) return true;
  return blockText(it, "reasoning").trim() !== "";
}

/** Visible reply content — the process-groups split boundary. */
export function hasAssistantReply(it: Item): boolean {
  if (it.type !== "assistant") return false;
  if (it.delta || it.payload?.streaming) return true;
  if (it.text.trim()) return true;
  return blockText(it, "text").trim() !== "";
}

/**
 * Harness compact-answer: only the latest tool-free reply is the letter.
 * Intermediate narration joins the process rail (`groupPart: response`).
 */
export function isCompactAnswer(items: Item[], index: number): boolean {
  const it = items[index];
  if (!hasAssistantReply(it)) return false;
  for (let i = index + 1; i < items.length; i++) {
    const later = items[i];
    if (hasAssistantReply(later) || isToolish(later)) return false;
  }
  return true;
}

function taggedAssistant(it: Item, groupPart: "reasoning" | "response"): Item {
  return { ...it, payload: { ...it.payload, groupPart } };
}

/**
 * Process groups split on an assistant reply, not on tool category. Reasoning
 * joins the in-step rail; only the compact answer stands outside it. Artifact
 * cards stay first-class (yoyo IA) rather than flattening into the rail.
 */
export function layoutAgentParts(items: Item[], opts?: LayoutOpts): AgentPart[] {
  const parts: AgentPart[] = [];
  let process: Item[] = [];
  let artifacts: Item[] = [];
  const pairKind = classifyToolPairs(pairTools(items.filter(isToolish)));
  const grouped = opts?.policy ? processGrouped(opts.policy, !!opts.running, !!opts.nextUser) : true;
  const flushProcess = () => {
    if (!process.length) return;
    parts.push({
      key: `process:${process[0].key}`,
      kind: "process",
      items: process,
      live: processGroupLive(process),
      grouped,
    });
    process = [];
  };
  const flushArtifacts = () => {
    if (!artifacts.length) return;
    parts.push({ key: `artifact:${artifacts[0].key}`, kind: "artifact", items: artifacts });
    artifacts = [];
  };
  for (let i = 0; i < items.length; i++) {
    const it = items[i];
    if (it.type === "assistant_stream" || it.type === "turn_start" || it.type === "step_start" || it.type === "step_end" || it.type === "inbox") continue;
    if (it.type === "reasoning" && opts?.showThinking === false) continue;
    if (isMentionInject(it) || isTranscriptDuplicate(it)) continue;
    if (it.type === "subagent") {
      flushProcess();
      flushArtifacts();
      const child = subagentChildId(it) || it.key;
      const existing = parts.find((p) => p.kind === "subagent" && p.child === child);
      if (existing && existing.kind === "subagent") {
        existing.items = [...existing.items, it];
        existing.live = subagentLive(existing.items);
      } else {
        parts.push({ key: `subagent:${child}`, kind: "subagent", items: [it], child, live: subagentLive([it]) });
      }
      continue;
    }
    if (isToolish(it)) {
      const id = String(it.payload?.id || it.key);
      if (pairKind.get(id) === "artifact") {
        flushProcess();
        artifacts.push(it);
      } else {
        flushArtifacts();
        process.push(it);
      }
      continue;
    }
    if (it.type === "assistant") {
      if (opts?.showThinking !== false && hasAssistantReasoning(it)) {
        process.push(taggedAssistant(it, "reasoning"));
      }
      if (hasAssistantReply(it)) {
        if (!isCompactAnswer(items, i)) {
          process.push(taggedAssistant(it, "response"));
          continue;
        }
        flushProcess();
        flushArtifacts();
        const toolsFollow = items.slice(i + 1).some(isToolish);
        const toolsYet = items.some(isToolish);
        const streaming = !!(opts?.running && (it.delta || it.payload?.streaming || it.payload?.delta || (!toolsFollow && !toolsYet)));
        const tagged = taggedAssistant(it, "response");
        if (streaming) {
          parts.push({ key: tagged.key, kind: "stream", item: tagged });
        } else {
          parts.push({ key: tagged.key, kind: "item", item: tagged });
        }
      }
      continue;
    }
    if (isProcessItem(it)) {
      process.push(it);
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

function hydrateItem(it: Item, byKey: Map<string, Item>): Item {
  const fresh = byKey.get(it.key);
  if (!fresh || fresh === it) return it;
  const groupPart = it.payload?.groupPart;
  if (groupPart === "reasoning" || groupPart === "response") {
    return {
      ...it,
      delta: fresh.delta,
      payload: { ...fresh.payload, ...it.payload, groupPart },
      text: groupPart === "reasoning" ? (reasoningText(fresh) || it.text) : (fresh.text || it.text),
    };
  }
  return fresh;
}

function hydrateParts(parts: AgentPart[], byKey: Map<string, Item>): AgentPart[] {
  return parts.map((p) => {
    if (p.kind === "item" || p.kind === "stream") {
      const item = hydrateItem(p.item, byKey);
      return item === p.item ? p : { ...p, item };
    }
    if (p.kind === "process" || p.kind === "artifact" || p.kind === "subagent") {
      const items = p.items.map((it) => hydrateItem(it, byKey));
      const changed = items.some((it, i) => it !== p.items[i]);
      return changed ? { ...p, items } : p;
    }
    if (p.kind === "work") {
      const inner = hydrateParts(p.parts, byKey);
      const changed = inner.some((x, i) => x !== p.parts[i]);
      return changed ? { ...p, parts: inner } : p;
    }
    return p;
  });
}

/** Patch cached layout seats with the current item bodies without recomputing structure. */
export function hydrateLayout(rows: LayoutRow[], items: Item[]): LayoutRow[] {
  const byKey = new Map<string, Item>();
  for (const it of items) byKey.set(it.key, it);
  return rows.map((row) => {
    if (row.kind === "user" || row.kind === "solo") {
      const item = hydrateItem(row.item, byKey);
      return item === row.item ? row : { ...row, item };
    }
    if (row.kind === "agent") {
      const parts = hydrateParts(row.parts, byKey);
      const changed = parts.some((p, i) => p !== row.parts[i]);
      return changed ? { ...row, parts } : row;
    }
    return row;
  });
}

/** One visual row per operator turn. Agent text/tools between users stay together. */
export function layoutRows(items: Item[], opts?: LayoutOpts): LayoutRow[] {
  const rows: LayoutRow[] = [];
  let agent: Item[] = [];
  let turnKey = "";
  const flushAgent = (live: boolean, nextUser: boolean) => {
    if (!agent.length) return;
    const partOpts: LayoutOpts = { running: live, showThinking: opts?.showThinking, policy: opts?.policy, nextUser };
    rows.push({
      key: `turn:${agent[0].key}`,
      kind: "agent",
      parts: wrapTurnWork(layoutAgentParts(agent, partOpts), partOpts),
      copyText: agentCopyText(agent),
      turnKey,
    });
    agent = [];
  };
  for (const it of items) {
    if (it.type === "user" && it.source !== "steer") {
      flushAgent(false, true);
      turnKey = itemTurnKey(it);
      rows.push({ key: it.key, kind: "user", item: it, turnKey });
      continue;
    }
    if (isMentionInject(it)) {
      continue;
    }
    // Neutral: the approval card renders from the pending list. As a solo row it
    // used to split the turn in two around an empty gap.
    if (it.type === "approval") {
      continue;
    }
    // A mid-turn context checkpoint is a divider inside the turn, not a new turn.
    if (it.type === "compaction" && agent.length) {
      agent.push(it);
      continue;
    }
    if (isAgentItem(it)) {
      agent.push(it);
      continue;
    }
    flushAgent(false, false);
    rows.push({ key: it.key, kind: "solo", item: it });
  }
  flushAgent(!!opts?.running, false);
  return rows;
}

