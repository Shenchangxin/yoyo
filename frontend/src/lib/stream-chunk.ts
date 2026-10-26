/**
 * 拥有：与 harness 组装器对齐的封闭流片段，以及把增量收成内容块。
 * 不拥有：实时通知的游标，以及把块折进转录条目。
 */

/** ContentBlock 组装完成后的一块：正文、推理或工具调用。 */
export type ContentBlock =
  | { type: "text"; text: string }
  | { type: "reasoning"; text: string }
  | { type: "tool-call"; id: string; name: string; arguments: string };

/** StreamChunk 封闭的流片段：块开始、增量、块结束、用量和结束原因。 */
export type StreamChunk =
  | { type: "block-start"; index: number; blockType: string }
  | { type: "text-delta"; index: number; text: string }
  | { type: "reasoning-delta"; index: number; text: string }
  | { type: "tool-call-delta"; index: number; id: string; name?: string; argumentsDelta: string }
  | { type: "block-end"; index: number; block: ContentBlock }
  | { type: "usage"; usage: { inputTokens: number; outputTokens: number; cacheReadTokens?: number } }
  | { type: "finish"; reason: { kind: string } };

/** AssistantStreamRecord 落在条目上的打包流：按时间增量压紧的文本、推理、工具参数，或单独一块。 */
export type AssistantStreamRecord =
  | { type: "text-chunks"; time0: number; index: number; dt: number[]; texts: string[] }
  | { type: "reasoning-chunks"; time0: number; index: number; dt: number[]; texts: string[] }
  | { type: "tool-call-chunks"; time0: number; index: number; dt: number[]; id: string; name?: string; args: string[] }
  | { type: "chunk"; time: number; chunk: StreamChunk };

type PartialBlock = {
  blockType: string;
  text: string;
  toolCallId?: string;
  toolCallName?: string;
  toolCallArguments: string;
  block?: ContentBlock;
};

/** BlockAssembler 按序号把流片段收成内容块。结束原因为 max-tokens 时丢掉工具调用。 */
export class BlockAssembler {
  private partials = new Map<number, PartialBlock>();
  private order: number[] = [];
  private finishKind = "stop";

  /** push 按片段种类累积一块。已经收束的块不再接受增量。chunk 读取 type、index、blockType、text、id、name 等。 */
  push(chunk: StreamChunk): void {
    switch (chunk.type) {
      case "block-start":
        if (!this.partials.has(chunk.index)) {
          this.order.push(chunk.index);
          this.partials.set(chunk.index, { blockType: chunk.blockType, text: "", toolCallArguments: "" });
        }
        return;
      case "text-delta":
      case "reasoning-delta": {
        const p = this.ensure(chunk.index, chunk.type === "text-delta" ? "text" : "reasoning");
        if (p.block) return;
        p.text += chunk.text;
        return;
      }
      case "tool-call-delta": {
        const p = this.ensure(chunk.index, "tool-call");
        if (p.block) return;
        p.toolCallId = chunk.id || p.toolCallId;
        if (chunk.name) p.toolCallName = chunk.name;
        p.toolCallArguments += chunk.argumentsDelta || "";
        return;
      }
      case "block-end": {
        const p = this.ensure(chunk.index, chunk.block.type);
        if (p.block) return;
        p.block = chunk.block;
        return;
      }
      case "finish":
        this.finishKind = chunk.reason?.kind || "stop";
        return;
      default:
        return;
    }
  }

  /** 取出或建出该序号的半成品块。index 传给 partials.get，交给 partials.set，放进 order。blockType 写入字段 blockType。 */
  private ensure(index: number, blockType: string): PartialBlock {
    let p = this.partials.get(index);
    if (!p) {
      p = { blockType, text: "", toolCallArguments: "" };
      this.partials.set(index, p);
      this.order.push(index);
    }
    return p;
  }

  /** 半成品收成内容块。工具调用缺少 id 时用序号占位。p 读取 block、blockType、text、toolCallId、toolCallName、toolCallArguments。index 拼进字符串。 */
  private assemble(p: PartialBlock, index: number): ContentBlock {
    if (p.block) return p.block;
    if (p.blockType === "reasoning") return { type: "reasoning", text: p.text };
    if (p.blockType === "tool-call") {
      return {
        type: "tool-call",
        id: p.toolCallId || `call-${index}`,
        name: p.toolCallName || "",
        arguments: p.toolCallArguments,
      };
    }
    return { type: "text", text: p.text };
  }

  /** hydrateFromBlocks 用已经组装好的块恢复一次进行中的尝试。最后一块保持未收束，前面的块收束。blocks 为假时改用后备值，读取 length，逐项处理。 */
  hydrateFromBlocks(blocks: ContentBlock[]): void {
    this.partials.clear();
    this.order = [];
    this.finishKind = "stop";
    (blocks || []).forEach((b, i) => {
      this.push({ type: "block-start", index: i, blockType: b.type });
      if (b.type === "text") this.push({ type: "text-delta", index: i, text: b.text });
      else if (b.type === "reasoning") this.push({ type: "reasoning-delta", index: i, text: b.text });
      else this.push({ type: "tool-call-delta", index: i, id: b.id, name: b.name, argumentsDelta: b.arguments });
      if (i < blocks.length - 1) this.push({ type: "block-end", index: i, block: b });
    });
  }

  /** blocks 按出现顺序交出内容块。结束原因为 max-tokens 时去掉工具调用。 */
  blocks(): ContentBlock[] {
    const all = this.order.map((i) => this.assemble(this.partials.get(i)!, i));
    if (this.finishKind !== "max-tokens") return all;
    return all.filter((b) => b.type !== "tool-call");
  }

  /** interruptedBlocks 只留下中断时已经有正文的文本和推理块。 */
  interruptedBlocks(): ContentBlock[] {
    return this.order
      .map((i) => {
        const p = this.partials.get(i);
        if (!p) return undefined;
        const type = p.block?.type ?? p.blockType;
        if (type !== "text" && type !== "reasoning") return undefined;
        return this.assemble(p, i);
      })
      .filter((b): b is ContentBlock => !!b && (b.type === "text" || b.type === "reasoning") && b.text.trim() !== "");
  }
}

/** expandAssistantStream 把打包记录展开成带时间的流片段。dt 是后一块相对前一块的时间增量。records 为假时改用后备值，用来遍历。 */
export function expandAssistantStream(records: AssistantStreamRecord[]): { time: number; chunk: StreamChunk }[] {
  const out: { time: number; chunk: StreamChunk }[] = [];
  for (const rec of records || []) {
    if (rec.type === "text-chunks" || rec.type === "reasoning-chunks") {
      const kind = rec.type === "text-chunks" ? "text-delta" : "reasoning-delta";
      let t = rec.time0;
      rec.texts.forEach((text, i) => {
        if (i > 0) t += rec.dt[i - 1] || 0;
        out.push({ time: t, chunk: { type: kind, index: rec.index, text } });
      });
    } else if (rec.type === "tool-call-chunks") {
      let t = rec.time0;
      rec.args.forEach((arg, i) => {
        if (i > 0) t += rec.dt[i - 1] || 0;
        out.push({
          time: t,
          chunk: { type: "tool-call-delta", index: rec.index, id: rec.id, name: rec.name, argumentsDelta: arg },
        });
      });
    } else if (rec.type === "chunk" && rec.chunk) {
      out.push({ time: rec.time, chunk: rec.chunk });
    }
  }
  return out;
}

/** textFromBlocks 把指定种类的块正文串起来。缺省串正文块。blocks 读取 length，为空时不读，用来遍历。kind 与 b.type 比较是否相等。 */
export function textFromBlocks(blocks: ContentBlock[] | undefined, kind: "text" | "reasoning" = "text"): string {
  if (!blocks?.length) return "";
  let s = "";
  for (const b of blocks) {
    if (b.type === kind) s += b.text;
  }
  return s;
}

/** parseStreamChunk 从原始对象读出流片段。不认识的种类返回空。raw 读取 type、index、blockType、block_type、text、id 等，为假时走另一支，用 typeof 看类型。 */
export function parseStreamChunk(raw: any): StreamChunk | undefined {
  if (!raw || typeof raw !== "object") return undefined;
  const type = String(raw.type || "");
  const index = Number(raw.index || 0);
  switch (type) {
    case "block-start":
      return { type, index, blockType: String(raw.blockType || raw.block_type || "") };
    case "text-delta":
    case "reasoning-delta":
      return { type, index, text: String(raw.text || "") };
    case "tool-call-delta":
      return {
        type,
        index,
        id: String(raw.id || ""),
        name: raw.name ? String(raw.name) : undefined,
        argumentsDelta: String(raw.argumentsDelta || raw.arguments_delta || ""),
      };
    case "block-end":
      return { type, index, block: parseContentBlock(raw.block) };
    case "usage":
      return { type, usage: raw.usage || { inputTokens: 0, outputTokens: 0 } };
    case "finish":
      return { type, reason: raw.reason || { kind: "stop" } };
    default:
      return undefined;
  }
}

/** 把原始对象收成内容块。缺省当正文。raw 读取 type、text、id、name、arguments，为空时不读。 */
function parseContentBlock(raw: any): ContentBlock {
  const type = String(raw?.type || "text");
  if (type === "reasoning") return { type, text: String(raw?.text || "") };
  if (type === "tool-call") {
    return { type, id: String(raw?.id || ""), name: String(raw?.name || ""), arguments: String(raw?.arguments || "") };
  }
  return { type: "text", text: String(raw?.text || "") };
}

/** parseContentBlocks 把数组收成内容块。不是数组则返回空。raw 做映射，传给 Array.isArray。 */
export function parseContentBlocks(raw: any): ContentBlock[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  return raw.map(parseContentBlock);
}
