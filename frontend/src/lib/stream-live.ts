import type { Item } from "./protocol";

/** Token-shaped events: text may grow without changing transcript structure. */
export function isLiveDelta(it: Item): boolean {
  if (!it?.delta) return false;
  return it.type === "assistant" || it.type === "reasoning" || it.type === "tool_result";
}

/** Identity of the transcript graph without token text — layoutRows key. */
export function structureSig(list: Item[]): string {
  let s = "";
  for (const it of list) {
    s += `${it.key}\t${it.type}\t${it.delta ? "1" : "0"}\t${it.name}\n`;
  }
  return s;
}

export function liveBody(it: Item): string {
  if (it.type === "tool_result") {
    const c = it.payload?.content ?? it.text;
    return typeof c === "string" ? c : it.text || "";
  }
  return it.text || "";
}

export function withLiveText(it: Item, live?: Record<string, string>): Item {
  if (!live) return it;
  const t = live[it.key];
  if (t == null) return it;
  if (it.type === "tool_result") {
    return { ...it, text: t, payload: { ...it.payload, content: t } };
  }
  return { ...it, text: t, payload: { ...it.payload, text: t } };
}
