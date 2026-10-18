import type { Item } from "./protocol";
import { parseStreamChunk, type StreamChunk } from "./stream-chunk";

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

export type ClientStreamResult =
  | { type: "start"; attemptId: string; turn: number; step: number }
  | { type: "chunk"; attemptId: string; turn: number; step: number; chunk: StreamChunk }
  | { type: "end"; attemptId: string; outcome: string }
  | { type: "rebaseline" }
  | undefined;

/**
 * Client fold for dense Assistant frames. Coalesced frames carry index_from..index;
 * nextIndex advances to index+1. Transients stay until step/end.
 */
export class ClientAssistantStream {
  private active: ActiveAttempt | undefined;
  retained: Pick<ActiveAttempt, "attemptId" | "turn" | "step"> | undefined;

  reset(): void {
    this.active = undefined;
    this.retained = undefined;
  }

  hydrate(active: ActiveAttempt): void {
    this.active = { ...active };
    this.retained = undefined;
  }

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

  retire(attemptId?: string): void {
    if (!attemptId || this.retained?.attemptId === attemptId) this.retained = undefined;
    if (this.active?.attemptId === attemptId) this.active = undefined;
  }

  get activeAttempt(): ActiveAttempt | undefined {
    return this.active;
  }
}

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
