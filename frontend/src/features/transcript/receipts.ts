/**
 * 拥有：一轮回复里哪些事要在末尾留一行回执。现在只有写进记忆的那几条。
 * 不拥有：回执怎么画，那在 TurnReceipts。
 */

import type { AgentPart } from "../../lib/transcript-layout.ts";
import { isToolFailed, toolArgs, toolName, toolResultBody } from "../../lib/tool-summary.ts";
import type { Item } from "../../lib/protocol.ts";

/** MEMORY_TOOLS 是会写下一条长期记忆的工具。 */
const MEMORY_TOOLS = new Set(["memory_write", "remember_fact"]);
/** REFUSED 是记忆工具没有记下时回给模型的开头，比如带了密钥。 */
const REFUSED = "没有记";

function flatten(parts: AgentPart[], out: Item[]) {
  for (const part of parts) {
    if (part.kind === "work") flatten(part.parts, out);
    else if (part.kind === "item" || part.kind === "stream") out.push(part.item);
    else out.push(...part.items);
  }
}

/** memoryReceipts 列出这一轮真正写进记忆的句子，按写下的先后。被拒绝、失败或还没有结果的不算。 */
export function memoryReceipts(parts: AgentPart[]): string[] {
  const items: Item[] = [];
  flatten(parts, items);
  const calls = new Map<string, Item>();
  const out: string[] = [];
  for (const it of items) {
    const id = String(it.payload?.id || "");
    if (!id) continue;
    if (it.type === "tool_call" && MEMORY_TOOLS.has(toolName(it))) calls.set(id, it);
    if (it.type !== "tool_result") continue;
    const call = calls.get(id);
    if (!call || isToolFailed(it) || toolResultBody(it).trim().startsWith(REFUSED)) continue;
    const text = String(toolArgs(call).text || "").trim();
    if (text && !out.includes(text)) out.push(text);
  }
  return out;
}
