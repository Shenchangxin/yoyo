import type { Item } from "./protocol";
import {
  isSteer,
  isToolish,
  pairState,
  pairTools,
  type ToolPair,
} from "./transcript-layout";
import { itemTimeMs, toolDetail, toolKind, toolName, type ToolKind } from "./tool-summary";

/**
 * 拥有：从进行中的转录推出作曲栏可以显示的工作状态。
 * 不拥有：状态栏的渲染，以及工具正文的摘要算法。
 */

/** StatusVerb 作曲栏允许说的动词。think 表示正在组织回答，不是工具。 */
export type StatusVerb = ToolKind | "think";

/** TurnPhase 这一轮处在思考、工具或正文流出。 */
export type TurnPhase = "thinking" | "tool" | "streaming";

/**
 * TurnStatus 进行中回合的实时投影，由转录推导，不另存一份。
 * live 表示末尾的助手正文还在长。线程仍在跑时作曲栏保持这一行，避免字已经停住时看不出工作还在进行。
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

/** lastRealUserIndex 最后一条非引导的用户消息下标。没有则返回 -1。items 读取 length，按下标读取。 */
export function lastRealUserIndex(items: Item[]): number {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "user" && !isSteer(items[i])) return i;
  }
  return -1;
}

/** lastUserTimeMs 最后一条真实用户消息的时间。没有则为零。items 按下标读取，传给 lastRealUserIndex。 */
export function lastUserTimeMs(items: Item[]): number {
  const i = lastRealUserIndex(items);
  if (i < 0) return 0;
  return itemTimeMs(items[i]) || 0;
}

/** 当前回合里，最后一条真实用户消息之后的条目。还没有用户消息时用整表。items 截取，作为分支结果，传给 lastRealUserIndex。 */
function turnSlice(items: Item[]): Item[] {
  const start = lastRealUserIndex(items);
  return start >= 0 ? items.slice(start + 1) : items;
}

/** 仍在跑的工具对里的最后一对。pairs 做过滤。running 传给 pairState。 */
function pendingTool(pairs: ToolPair[], running: boolean): ToolPair | undefined {
  const pending = pairs.filter((p) => pairState(p, running) === "running");
  return pending[pending.length - 1];
}

/** 切片里最后一条助手消息。items 读取 length，按下标读取。 */
function lastAssistant(items: Item[]): Item | undefined {
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].type === "assistant") return items[i];
  }
  return undefined;
}

/** 工具对上报的字节数。调用和结果都没有则为零。pair 读取 call、result。 */
function toolBytes(pair: ToolPair): number {
  const item = pair.call || pair.result;
  return Number(item?.payload?.bytes || 0) || 0;
}

/**
 * deriveTurnStatus 作曲栏拥有的工作状态。
 * 线程空闲时返回空，避免输入区多出一行幽灵状态。有进行中的工具时优先报工具，否则报正在流出或尚未出字的思考。
 * items 传给 lastUserTimeMs，传给 turnSlice。
 * running 为假时返回 null。
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

/** statusBarVisible 线程上有实时状态时就显示这一栏，正文流出时也显示。status 不是 null 时返回真。 */
export function statusBarVisible(status: TurnStatus | null | undefined): status is TurnStatus {
  return status != null;
}
