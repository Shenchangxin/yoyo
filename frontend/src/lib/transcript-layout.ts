/**
 * 拥有：把转录条目排成操作者回合、过程轨、产出卡片和子代理。
 * 不拥有：详略策略本身，那在 presentation-policy；条目如何从事件折出来，那在 stream-fold。
 */

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

/** ToolPair 同一次工具的调用、结果，以及挂在同一 id 上的其余条目。 */
export type ToolPair = { key: string; call?: Item; result?: Item; extra: Item[] };

/** LayoutOpts 排版时线程是否仍在跑、是否显示思考，以及详略策略和下一条用户消息是否已经出现。 */
export type LayoutOpts = {
  running?: boolean;
  showThinking?: boolean;
  policy?: ChatPresentationPolicy;
  nextUser?: boolean;
};

/** AgentPart 智能体行里的一块：普通条目、过程轨、产出、子代理、流，或包住它们的工作块。 */
export type AgentPart =
  | { key: string; kind: "item"; item: Item }
  | { key: string; kind: "process"; items: Item[]; live: boolean; grouped?: boolean }
  | { key: string; kind: "artifact"; items: Item[] }
  | { key: string; kind: "subagent"; items: Item[]; child: string; live: boolean }
  | { key: string; kind: "stream"; item: Item }
  | { key: string; kind: "work"; parts: AgentPart[]; live: boolean; folded: boolean };

/** LayoutRow 一条视觉行：用户、智能体回合，或单独的分隔。 */
export type LayoutRow =
  | { key: string; kind: "user"; item: Item; turnKey: string }
  | { key: string; kind: "agent"; parts: AgentPart[]; copyText: string; turnKey: string }
  | { key: string; kind: "solo"; item: Item };

/** isToolish 工具调用或工具结果。 */
export function isToolish(it: Item): boolean {
  return it.type === "tool_call" || it.type === "tool_result";
}

/** isSteer 来源为引导的用户消息。 */
export function isSteer(it: Item): boolean {
  return it.type === "user" && it.source === "steer";
}

/** isMentionInject 提及注入折进用户气泡，不单独成一轮。技能注入除外。 */
export function isMentionInject(it: Item): boolean {
  return it.type === "context_injection" && it.source !== "skill";
}

/** liveAutoSkills 这一段里自动载入的技能名。同名只留一次。 */
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

/** isTranscriptDuplicate 已经在回合里有对应物的条目。留在智能体块里，不再单独渲染。成功的 task 调用由子代理卡片承担，失败结果仍要看见。 */
export function isTranscriptDuplicate(it: Item): boolean {
  if (it.type === "plan" || it.type === "file_change" || it.type === "ask_user") return true;
  // 成功的 task 调用由子代理卡片承担，失败结果仍要看见。
  if (isToolish(it) && toolName(it) === "task") {
    return !(it.type === "tool_result" && isToolFailed(it));
  }
  return false;
}

/** isOutcomeTool 操作者要打开的产出。update_plan 留在过程轨。 */
export function isOutcomeTool(it: Item): boolean {
  if (!isToolish(it)) return false;
  return pairShowsArtifact({
    key: String(it.payload?.id || it.key),
    call: it.type === "tool_call" ? it : undefined,
    result: it.type === "tool_result" ? it : undefined,
    extra: [],
  });
}

/** pairToolPath 这对工具的路径，反斜杠换成斜杠，并去掉开头的 ./。 */
export function pairToolPath(pair: ToolPair): string {
  const args = {
    ...(pair.result ? toolArgs(pair.result) : {}),
    ...(pair.call ? toolArgs(pair.call) : {}),
  };
  return String(args.path || args.file || "").replace(/\\/g, "/").replace(/^\.\//, "");
}

/** pairShowsArtifact 这对工具是否该做成产出卡片。失败、未改动，以及浏览器打开、接管和读文件都不算。 */
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

/** classifyToolPairs 给每一对标上产出或过程。 */
export function classifyToolPairs(pairs: ToolPair[]): Map<string, "artifact" | "process"> {
  const kind = new Map<string, "artifact" | "process">();
  for (const p of pairs) {
    kind.set(p.key, pairShowsArtifact(p) ? "artifact" : "process");
  }
  return kind;
}

/** isProcessItem 进过程轨的条目：推理、上下文注入，以及非产出的工具。提及注入和子代理不算。 */
export function isProcessItem(it: Item): boolean {
  if (isMentionInject(it)) return false;
  if (it.type === "reasoning" || it.type === "context_injection") return true;
  if (it.type === "subagent") return false;
  return isToolish(it) && !isOutcomeTool(it);
}

/** subagentChildId 子代理条目指向的子会话。 */
export function subagentChildId(it: Item): string {
  return String(it.payload?.child || "").trim();
}

/** subagentLive 最后一条的阶段还不是 complete 时，子代理仍在进行。 */
export function subagentLive(items: Item[]): boolean {
  if (!items.length) return false;
  const last = items[items.length - 1];
  const phase = String(last.payload?.phase || "").trim();
  return phase !== "complete";
}

/** liveSubagentCrumb 进行中子代理的提示，给会话头的面包屑。取第一行非空文字。 */
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

/** isAgentItem 属于智能体回合的条目。提及注入不算。 */
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

/** pairTools 按 id 把调用、结果和其余条目配成对，保持出现顺序。 */
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

/** processGroupLive 过程组里还有未结束的工具，或助手、推理仍在流。增量中的结果不算结束。 */
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

/** processToolPairs 只留带调用或结果的工具对。 */
export function processToolPairs(items: Item[]): ToolPair[] {
  return pairTools(items).filter((p) => p.call || p.result);
}

/** PROCESS_PAGE 过程轨从尾部展开的行数，避免上百步的回合一次挂上每一行。 */
export const PROCESS_PAGE = 60;
/** ARTIFACT_PAGE 产出卡片从尾部挂载的张数，更早的文件留待一次点击。 */
export const ARTIFACT_PAGE = 12;
/** TOOL_PREVIEW_LINES 过程轨里工具输出的紧凑行数。完整正文留在磁盘或卸出块。 */
export const TOOL_PREVIEW_LINES = 5;

/** clipPreview 头尾预览，避免很长的命令输出整段挂成一堵字。隐藏行数一并返回。 */
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

/** tailProcessPairs 只留尾部 shown 对。shown 不大于零或已经不长于它时全部可见。 */
export function tailProcessPairs(pairs: ToolPair[], shown: number): { visible: ToolPair[]; hidden: number } {
  if (shown <= 0 || pairs.length <= shown) return { visible: pairs, hidden: 0 };
  return { visible: pairs.slice(pairs.length - shown), hidden: pairs.length - shown };
}

/** processElapsedMs 组内各工具结果上报耗时之和。 */
export function processElapsedMs(items: Item[]): number {
  return processToolPairs(items).reduce((sum, p) => sum + (p.result ? toolElapsedMs(p.result) : 0), 0);
}

/** processStartMs 组里最早的时间戳。「工作了多久」从这里起算。 */
export function processStartMs(items: Item[]): number {
  let min = 0;
  for (const it of items) {
    const t = itemTimeMs(it);
    if (t && (!min || t < min)) min = t;
  }
  return min;
}

/**
 * processSpanMs 已收束组的墙钟。工具批次之间的模型思考也算工作，所以用起止时间，而不是把各工具的 elapsed_ms 相加。
 * 没有时间戳时退回各工具耗时之和，旧 JSONL 和测试替身会走到这里。
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

/** processFailedCount 组内失败的工具对数。 */
export function processFailedCount(items: Item[]): number {
  return processToolPairs(items).filter((p) => isToolFailed(p.result)).length;
}

/** PairState 一对工具的状态：进行中、完成、失败或中断。 */
export type PairState = "running" | "done" | "failed" | "interrupted";

/**
 * pairState 没有结果的调用只在线程仍在跑时算进行中。回合结束后如实标成中断，已结束的回合上不留转圈。
 */
export function pairState(pair: ToolPair, running: boolean): PairState {
  if (pair.call && !pair.result) return running ? "running" : "interrupted";
  if (pair.result && (pair.result.delta || pair.result.payload?.delta)) {
    return running ? "running" : "interrupted";
  }
  if (isToolFailed(pair.result)) return "failed";
  return "done";
}

/** KindTally 过程组里各动词族的次数。 */
export type KindTally = Partial<Record<ToolKind, number>>;

/** processKindTally 按工具动词族计数。 */
export function processKindTally(items: Item[]): KindTally {
  const tally: KindTally = {};
  for (const p of processToolPairs(items)) {
    const k = toolKind(toolName(p.call || p.result!));
    tally[k] = (tally[k] || 0) + 1;
  }
  return tally;
}

/** processHasReasoning 组里有没有推理条目。 */
export function processHasReasoning(items: Item[]): boolean {
  return items.some((it) => it.type === "reasoning");
}

/** processProgressCount 组里的助手条目数。 */
export function processProgressCount(items: Item[]): number {
  let n = 0;
  for (const it of items) {
    if (it.type === "assistant") n++;
  }
  return n;
}

/** 复制用的正文：从后往前找第一条非空助手文本。 */
function agentCopyText(items: Item[]): string {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "assistant" && items[i].text.trim()) return items[i].text.trim();
  }
  return "";
}

/** 从块列表里串出指定种类的正文。没有块时，正文种类用条目文本，推理种类为空。 */
function blockText(it: Item, kind: "text" | "reasoning"): string {
  const blocks = it.payload?.blocks;
  if (!Array.isArray(blocks)) return kind === "text" ? it.text : "";
  let s = "";
  for (const b of blocks) {
    if (b && b.type === kind) s += String(b.text || "");
  }
  return s;
}

/** reasoningText 推理条目用自身文本，否则用载荷上的推理字段或推理块。 */
export function reasoningText(it: Item): string {
  if (it.type === "reasoning") return it.text.trim();
  const tagged = String(it.payload?.reasoning || "").trim();
  if (tagged) return tagged;
  return blockText(it, "reasoning").trim();
}

/** flattenWorkProcess 把工作块、过程轨和子代理摊平回条目。产出卡片不摊进去。 */
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

/** 最后一条后面不再跟着过程、产出或子代理的助手，才是紧凑回答。 */
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

/** wrapTurnWork 把紧凑回答之前的部件收进一个工作块，并按策略标上分组和折叠。 */
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

/** hasAssistantReasoning 条目上有非空推理，来自推理条目、载荷字段或推理块。 */
export function hasAssistantReasoning(it: Item): boolean {
  if (it.type === "reasoning") return it.text.trim() !== "";
  if (it.type !== "assistant") return false;
  if (String(it.payload?.reasoning || "").trim()) return true;
  return blockText(it, "reasoning").trim() !== "";
}

/** hasAssistantReply 看得见的回复。过程组以此为分界。正在流或有正文都算。 */
export function hasAssistantReply(it: Item): boolean {
  if (it.type !== "assistant") return false;
  if (it.delta || it.payload?.streaming) return true;
  if (it.text.trim()) return true;
  return blockText(it, "text").trim() !== "";
}

/**
 * isCompactAnswer 只有最后一条不再跟着工具的回复才是信。
 * 中间的叙述并进过程轨，并标成 response。
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

/** 给助手标上 groupPart，表示它在过程轨里是推理还是回复。 */
function taggedAssistant(it: Item, groupPart: "reasoning" | "response"): Item {
  return { ...it, payload: { ...it.payload, groupPart } };
}

/**
 * layoutAgentParts 过程组按助手回复切开，不按工具种类切开。
 * 推理并进当前步骤的轨，只有紧凑回答站在轨外。产出卡片保持独立，不摊进过程轨。
 */
export function layoutAgentParts(items: Item[], opts?: LayoutOpts): AgentPart[] {
  const parts: AgentPart[] = [];
  let process: Item[] = [];
  let artifacts: Item[] = [];
  const pairKind = classifyToolPairs(pairTools(items.filter(isToolish)));
  const grouped = opts?.policy ? processGrouped(opts.policy, !!opts.running, !!opts.nextUser) : true;
  /** 把积下的过程条目收成一块过程轨。 */
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
  /** 把积下的产出收成一张卡片列表。 */
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

/** 用当前条目替换缓存座位上的正文。推理和回复保留原来的 groupPart。 */
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

/** 递归把部件里的条目换成当前正文。没变的部件保持原引用。 */
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

/** hydrateLayout 用当前条目正文修补缓存座位，不重算结构。 */
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

/** layoutRows 一个操作者回合一行。用户之间的助手正文和工具留在同一行。 */
export function layoutRows(items: Item[], opts?: LayoutOpts): LayoutRow[] {
  const rows: LayoutRow[] = [];
  let agent: Item[] = [];
  let turnKey = "";
  /** 把积下的智能体条目收成一行。live 表示这行仍在进行，nextUser 表示后面已经有下一条用户消息。 */
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
    // 批准卡片从待办列表渲染。单独成行时会把回合从空档处拆成两截。
    if (it.type === "approval") {
      continue;
    }
    // 回合中的上下文检查点是回合内部的分隔，不是新回合。
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

