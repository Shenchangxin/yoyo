/** Closed StreamChunk union + BlockAssembler, matching the harness assembler. */

export type ContentBlock =
  | { type: "text"; text: string }
  | { type: "reasoning"; text: string }
  | { type: "tool-call"; id: string; name: string; arguments: string };

export type StreamChunk =
  | { type: "block-start"; index: number; blockType: string }
  | { type: "text-delta"; index: number; text: string }
  | { type: "reasoning-delta"; index: number; text: string }
  | { type: "tool-call-delta"; index: number; id: string; name?: string; argumentsDelta: string }
  | { type: "block-end"; index: number; block: ContentBlock }
  | { type: "usage"; usage: { inputTokens: number; outputTokens: number; cacheReadTokens?: number } }
  | { type: "finish"; reason: { kind: string } };

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

export class BlockAssembler {
  private partials = new Map<number, PartialBlock>();
  private order: number[] = [];
  private finishKind = "stop";

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

  private ensure(index: number, blockType: string): PartialBlock {
    let p = this.partials.get(index);
    if (!p) {
      p = { blockType, text: "", toolCallArguments: "" };
      this.partials.set(index, p);
      this.order.push(index);
    }
    return p;
  }

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

  /** Rehydrate a live attempt from already-assembled blocks without closing the last one. */
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

  blocks(): ContentBlock[] {
    const all = this.order.map((i) => this.assemble(this.partials.get(i)!, i));
    if (this.finishKind !== "max-tokens") return all;
    return all.filter((b) => b.type !== "tool-call");
  }

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

export function textFromBlocks(blocks: ContentBlock[] | undefined, kind: "text" | "reasoning" = "text"): string {
  if (!blocks?.length) return "";
  let s = "";
  for (const b of blocks) {
    if (b.type === kind) s += b.text;
  }
  return s;
}

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

function parseContentBlock(raw: any): ContentBlock {
  const type = String(raw?.type || "text");
  if (type === "reasoning") return { type, text: String(raw?.text || "") };
  if (type === "tool-call") {
    return { type, id: String(raw?.id || ""), name: String(raw?.name || ""), arguments: String(raw?.arguments || "") };
  }
  return { type: "text", text: String(raw?.text || "") };
}

export function parseContentBlocks(raw: any): ContentBlock[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  return raw.map(parseContentBlock);
}
