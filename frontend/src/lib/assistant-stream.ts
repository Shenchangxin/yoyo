/**
 * 拥有：客户端对密集助手帧的折叠。合并帧带 index_from 到 index，nextIndex 推进到 index 加一。瞬时状态留到步骤或结束。
 * 不拥有：流片段的解析，以及把块折进转录。
 */

import type { Item } from "./protocol";
import { parseStreamChunk, type StreamChunk } from "./stream-chunk";

/** AssistantStreamFrame 一帧助手流。kind 为开始、块或结束。合并帧的下标从 indexFrom 到 index。 */
export type AssistantStreamFrame = {
  kind: "start" | "chunk" | "end";
  attemptId: string;
  revision: number;
  index: number;
  indexFrom: number;
  turn?: number;
  step?: number;
  outcome?: string;
  chunk?: StreamChunk;
  committedSeq?: number;
};

type ActiveAttempt = {
  attemptId: string;
  turn: number;
  step: number;
  nextIndex: number;
};

/** ClientStreamResult 折叠一帧后的结果：开始、一块、结束、需要重新对齐，或没有变化。 */
export type ClientStreamResult =
  | { type: "start"; attemptId: string; turn: number; step: number }
  | { type: "chunk"; attemptId: string; turn: number; step: number; chunk: StreamChunk }
  | { type: "end"; attemptId: string; outcome: string }
  | { type: "rebaseline" }
  | undefined;

/** ClientAssistantStream 记住当前尝试。结果为 committed 的尝试留在 retained，直到被替换或退休。 */
export class ClientAssistantStream {
  private active: ActiveAttempt | undefined;
  retained: Pick<ActiveAttempt, "attemptId" | "turn" | "step"> | undefined;

  /** reset 清掉当前尝试和留下的尝试。 */
  reset(): void {
    this.active = undefined;
    this.retained = undefined;
  }

  /** hydrate 装入一个已在进行的尝试，并清掉留下的尝试。active 展开传递。 */
  hydrate(active: ActiveAttempt): void {
    this.active = { ...active };
    this.retained = undefined;
  }

  /**
   * acceptFrame 开始时若已有别的尝试则要求重新对齐。同一尝试的重复开始忽略。块的起点高于 nextIndex 则重新对齐，低于则忽略。结束时清掉当前尝试；结果为 committed 才留下。对不上当前尝试的结束，用帧上的结果，没有则是 abandoned。
   * frame 读取 kind、attemptId、turn、step、indexFrom、index 等。
   */
  acceptFrame(frame: AssistantStreamFrame): ClientStreamResult {
    switch (frame.kind) {
      case "start":
        if (this.active?.attemptId === frame.attemptId) return undefined;
        if (this.retained?.attemptId === frame.attemptId) this.retained = undefined;
        if (this.active || this.retained) return { type: "rebaseline" };
        this.active = {
          attemptId: frame.attemptId,
          turn: frame.turn || 0,
          step: frame.step || 0,
          nextIndex: 0,
        };
        return { type: "start", attemptId: frame.attemptId, turn: this.active.turn, step: this.active.step };
      case "chunk": {
        const attempt = this.active;
        if (!attempt || attempt.attemptId !== frame.attemptId) return undefined;
        const from = Number.isFinite(frame.indexFrom) ? frame.indexFrom : frame.index;
        const to = Number.isFinite(frame.index) ? frame.index : from;
        if (from > attempt.nextIndex) return { type: "rebaseline" };
        if (from < attempt.nextIndex) return undefined;
        attempt.nextIndex = Math.max(attempt.nextIndex, to + 1);
        if (!frame.chunk) return undefined;
        return { type: "chunk", attemptId: attempt.attemptId, turn: attempt.turn, step: attempt.step, chunk: frame.chunk };
      }
      case "end": {
        const attempt = this.active;
        if (!attempt || attempt.attemptId !== frame.attemptId) {
          this.active = undefined;
          this.retained = undefined;
          return { type: "end", attemptId: frame.attemptId, outcome: frame.outcome || "abandoned" };
        }
        const outcome = frame.outcome || "committed";
        if (outcome === "committed") {
          this.retained = { attemptId: attempt.attemptId, turn: attempt.turn, step: attempt.step };
        } else {
          this.retained = undefined;
        }
        this.active = undefined;
        return { type: "end", attemptId: attempt.attemptId, outcome };
      }
    }
  }

  /** retire 清掉留下的尝试。给了 attemptId 且当前尝试就是它时，也清掉当前尝试。没给 attemptId 时只清留下的。 */
  retire(attemptId?: string): void {
    if (!attemptId || this.retained?.attemptId === attemptId) this.retained = undefined;
    if (this.active?.attemptId === attemptId) this.active = undefined;
  }

  /** activeAttempt 当前尝试。 */
  get activeAttempt(): ActiveAttempt | undefined {
    return this.active;
  }
}

/** frameFromItem 从 assistant_stream 条目取出一帧。kind 不是开始、块或结束则没有帧。块用 parseStreamChunk。下标缺省为 0。item 读取 type、payload、seq。 */
export function frameFromItem(item: Item): AssistantStreamFrame | undefined {
  if (item.type !== "assistant_stream") return undefined;
  const p = item.payload || {};
  const kind = String(p.kind || "");
  if (kind !== "start" && kind !== "chunk" && kind !== "end") return undefined;
  const chunk = kind === "chunk" ? parseStreamChunk(p.chunk) : undefined;
  const index = p.index != null ? Number(p.index) : 0;
  const fromRaw = p.index_from ?? p.indexFrom;
  const indexFrom = fromRaw != null ? Number(fromRaw) : index;
  return {
    kind,
    attemptId: String(p.attempt_id || p.attemptId || p.id || p.round || ""),
    revision: Number(p.revision || 0),
    index: Number.isFinite(index) ? index : 0,
    indexFrom: Number.isFinite(indexFrom) ? indexFrom : 0,
    turn: Number(p.turn || 0) || undefined,
    step: Number(p.step || 0) || undefined,
    outcome: p.outcome ? String(p.outcome) : undefined,
    chunk,
    committedSeq: item.seq,
  };
}
