import type { Item } from "./protocol";
import {
  isSteer,
  isToolish,
  pairState,
  pairTools,
  type ToolPair,
} from "./transcript-layout";
import { itemTimeMs, toolDetail, toolKind, toolName, type ToolKind } from "./tool-summary";

/** What the composer status bar is allowed to say. `think` is composing, not a tool. */
export type StatusVerb = ToolKind | "think";

export type TurnPhase = "thinking" | "tool" | "streaming";

/**
 * Live projection of the in-flight turn. Derived from the transcript; never
 * stored. `live` means the trailing assistant is still growing. The composer
 * keeps the bar while the thread is running so a frozen letter cannot hide
 * that work is still in flight.
 */
export type TurnStatus = {
  phase: TurnPhase;
  verb: StatusVerb;
  toolName: string;
  detail: string;
  since: number;
  live: boolean;
  bytes: number;
};

export function lastRealUserIndex(items: Item[]): number {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "user" && !isSteer(items[i])) return i;
  }
  return -1;
}

export function lastUserTimeMs(items: Item[]): number {
  const i = lastRealUserIndex(items);
  if (i < 0) return 0;
  return itemTimeMs(items[i]) || 0;
}

function turnSlice(items: Item[]): Item[] {
  const start = lastRealUserIndex(items);
  return start >= 0 ? items.slice(start + 1) : items;
}

function pendingTool(pairs: ToolPair[], running: boolean): ToolPair | undefined {
  const pending = pairs.filter((p) => pairState(p, running) === "running");
  return pending[pending.length - 1];
}

function lastAssistant(items: Item[]): Item | undefined {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "assistant") return items[i];
  }
  return undefined;
}

function toolBytes(pair: ToolPair): number {
  const item = pair.call || pair.result;
  return Number(item?.payload?.bytes || 0) || 0;
}

/**
 * Composer-owned work status. Null when the thread is idle so the input
 * does not grow a ghost row.
 */
export function deriveTurnStatus(
  items: Item[],
  running: boolean,
): TurnStatus | null {
  if (!running) return null;
  const since = lastUserTimeMs(items);
  const slice = turnSlice(items);
  const current = pendingTool(pairTools(slice.filter(isToolish)), true);
  if (current?.call || current?.result) {
    const item = current.call || current.result!;
    const name = toolName(item);
    const kind = toolKind(name);
    return {
      phase: "tool",
      verb: kind,
      toolName: name,
      detail: toolDetail(item),
      since,
      live: false,
      bytes: toolBytes(current),
    };
  }
  const assistant = lastAssistant(slice);
  const growing = !!(assistant && (assistant.delta || assistant.payload?.streaming));
  const hasText = !!(assistant && (assistant.text.trim() || growing));
  if (hasText) {
    return {
      phase: "streaming",
      verb: "think",
      toolName: "",
      detail: "",
      since,
      live: growing,
      bytes: 0,
    };
  }
  return {
    phase: "thinking",
    verb: "think",
    toolName: "",
    detail: "",
    since,
    live: false,
    bytes: 0,
  };
}

/** Show the bar whenever the thread has a live status — including streaming. */
export function statusBarVisible(status: TurnStatus | null | undefined): status is TurnStatus {
  return status != null;
}
