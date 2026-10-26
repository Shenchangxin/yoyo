import type { Item } from "./protocol";

/**
 * 拥有：进行中的转录里，哪些条目只是文本在长，以及把实时正文贴回条目。
 * 不拥有：流的订阅、折叠和视觉行排版。
 */

/** isLiveDelta 文本还可以继续长、结构不变的条目：带 streaming 标记，或增量中的助手、推理和工具结果。it 读取 payload、type、delta，为空时不读。 */
export function isLiveDelta(it: Item): boolean {
  if (it?.payload?.streaming) return it.type === "assistant" || it.type === "reasoning" || it.type === "tool_result";
  if (!it?.delta) return false;
  return it.type === "assistant" || it.type === "reasoning" || it.type === "tool_result";
}

/** structureSig 不含 token 正文的转录图身份，给 layoutRows 当缓存键。进行中与否算在里面。list 用来遍历。running 为真时才继续。 */
export function structureSig(list: Item[], running = false): string {
  let s = running ? "1\n" : "0\n";
  for (const it of list) {
    const live = it.delta || it.payload?.streaming || it.payload?.delta ? "d" : "s";
    s += `${it.key}\t${it.type}\t${it.name}\t${live}\n`;
  }
  return s;
}

/** liveBody 工具结果优先取 payload 里的正文，其余取条目文本。it 读取 type、payload、text。 */
export function liveBody(it: Item): string {
  if (it.type === "tool_result") {
    const c = it.payload?.content ?? it.text;
    return typeof c === "string" ? c : it.text || "";
  }
  return it.text || "";
}

/** withLiveText 用实时表里的正文替换条目。工具结果同时写回 content。表里没有这一条则原样返回。it 读取 key、type、payload，展开传递。live 为假时返回 it；按下标读取。 */
export function withLiveText(it: Item, live?: Record<string, string>): Item {
  if (!live) return it;
  const t = live[it.key];
  if (t == null) return it;
  if (it.type === "tool_result") {
    return { ...it, text: t, payload: { ...it.payload, content: t } };
  }
  return { ...it, text: t, payload: { ...it.payload, text: t } };
}
