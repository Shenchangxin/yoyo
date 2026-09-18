import type { Item } from "./protocol";

export type AgentPart =
  | { key: string; kind: "item"; item: Item }
  | { key: string; kind: "tools"; items: Item[] };

export type LayoutRow =
  | { key: string; kind: "user"; item: Item }
  | { key: string; kind: "agent"; parts: AgentPart[]; copyText: string }
  | { key: string; kind: "solo"; item: Item };

export function isToolish(it: Item): boolean {
  return it.type === "tool_call" || it.type === "tool_result";
}

export function isAgentItem(it: Item): boolean {
  return (
    it.type === "assistant" ||
    it.type === "reasoning" ||
    it.type === "subagent" ||
    isToolish(it) ||
    it.type === "context_injection"
  );
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
  let batch: Item[] = [];
  const flush = () => {
    if (!batch.length) return;
    parts.push({ key: `tools:${batch[0].key}`, kind: "tools", items: batch });
    batch = [];
  };
  for (const it of items) {
    if (isToolish(it) || (it.type === "context_injection" && batch.length > 0)) {
      batch.push(it);
      continue;
    }
    flush();
    parts.push({ key: it.key, kind: "item", item: it });
  }
  flush();
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
