/**
 * 拥有：「需要你」的那一份共享快照。编排快照和待审队列放在这里，今天页、底部坞上的数字、聚焦中的任务都读它。
 * 不拥有：条目怎么算，那在 needs-model.ts；也不拥有怎么处理一条，那由调用方做。
 */

import { create } from "zustand";
import { orchestratorView, reviewQueue } from "../../lib/client";
import type { DeskView } from "./desk-model";
import { EMPTY_QUEUE, needItems, normalizeQueue, type NeedItem, type ReviewQueue } from "./needs-model";

export type { NeedItem, NeedKind, ReviewQueue } from "./needs-model";

type NeedsState = {
  view: DeskView | null;
  queue: ReviewQueue;
  /** focusTask 是放到中间去聊的那张工单。空表示中间是主对话。 */
  focusTask: string;
  /** focusGroup 是放到中间去看的任务组。focusTask 也有值时先看那张工单，返回后回到任务组。 */
  focusGroup: string;
  /** paneMounted 为真时今天栏在场，对话里的「去看」才有地方落。 */
  paneMounted: boolean;
  /** pendingDesk 是今天栏不在场时要翻到的地方，等它挂上时接走。 */
  pendingDesk: DeskTarget | null;
  setView: (view: DeskView | null) => void;
  setQueue: (queue: ReviewQueue) => void;
  setFocusTask: (id: string) => void;
  setFocusGroup: (id: string) => void;
  setPaneMounted: (on: boolean) => void;
  setPendingDesk: (target: DeskTarget | null) => void;
};

/** useNeeds 是全窗口共用的一份快照。今天页每次刷新编排快照时也写进来。 */
export const useNeeds = create<NeedsState>((set) => ({
  view: null,
  queue: EMPTY_QUEUE,
  focusTask: "",
  focusGroup: "",
  paneMounted: false,
  pendingDesk: null,
  setView: (view) => set({ view }),
  setQueue: (queue) => set({ queue }),
  setFocusTask: (focusTask) => set({ focusTask }),
  setFocusGroup: (focusGroup) => set({ focusGroup }),
  setPaneMounted: (paneMounted) => set({ paneMounted }),
  setPendingDesk: (pendingDesk) => set({ pendingDesk }),
}));

const OPEN_DESK_EVENT = "yoyo-open-desk";

/** DeskTarget 是对话里点「去看」时今天栏要翻到的地方：一张工单、一个任务组，或记忆。 */
export type DeskTarget = { tab: "tasks"; ticket: string } | { tab: "tasks"; group: string } | { tab: "memory" };

/** openDesk 让今天栏翻到一张工单、一个任务组或记忆。今天栏不在场时记下来，等它挂上再翻，并返回假。 */
export function openDesk(target: DeskTarget): boolean {
  const handled = !window.dispatchEvent(new CustomEvent(OPEN_DESK_EVENT, { detail: target, cancelable: true }));
  if (!handled) useNeeds.getState().setPendingDesk(target);
  return handled;
}

/** onOpenDesk 订阅 openDesk。返回退订函数。 */
export function onOpenDesk(fn: (target: DeskTarget) => void): () => void {
  const handler = (event: Event) => {
    const target = (event as CustomEvent).detail as DeskTarget | null;
    if (!target) return;
    event.preventDefault();
    fn(target);
  };
  window.addEventListener(OPEN_DESK_EVENT, handler);
  return () => window.removeEventListener(OPEN_DESK_EVENT, handler);
}

const FOCUS_BRIEF_EVENT = "yoyo-focus-brief";

/** BriefTarget 是对话里那条简报消息带的内容。id 可能为空，按档和日子也能认出是哪一期。 */
export type BriefTarget = { id: string; slot: string; day: string; chip: string; brief: string };

/** focusBrief 让今天栏切到「今天」并翻到这一期简报。今天栏不在场时返回假，调用方再用弹层。 */
export function focusBrief(target: BriefTarget): boolean {
  return !window.dispatchEvent(new CustomEvent(FOCUS_BRIEF_EVENT, { detail: target, cancelable: true }));
}

/** onFocusBrief 订阅 focusBrief。返回退订函数。 */
export function onFocusBrief(fn: (target: BriefTarget) => void): () => void {
  const handler = (event: Event) => {
    const target = (event as CustomEvent).detail as BriefTarget | null;
    if (!target) return;
    event.preventDefault();
    fn(target);
  };
  window.addEventListener(FOCUS_BRIEF_EVENT, handler);
  return () => window.removeEventListener(FOCUS_BRIEF_EVENT, handler);
}

/** refreshNeeds 拉一次编排快照和待审队列。读不到的那一块保留上一次。 */
export async function refreshNeeds(): Promise<void> {
  const [view, queue] = await Promise.all([
    orchestratorView().catch(() => null),
    reviewQueue().catch(() => null),
  ]);
  const state = useNeeds.getState();
  if (view && typeof view === "object") state.setView(view as DeskView);
  if (queue && typeof queue === "object") state.setQueue(normalizeQueue(queue));
}

/** refreshQueue 只拉待审队列。审批或收件处理完之后用。 */
export async function refreshQueue(): Promise<void> {
  const queue = await reviewQueue().catch(() => null);
  if (queue && typeof queue === "object") useNeeds.getState().setQueue(normalizeQueue(queue));
}

/** useNeedItems 订阅快照，返回排好的条目。 */
export function useNeedItems(): NeedItem[] {
  const view = useNeeds((s) => s.view);
  const queue = useNeeds((s) => s.queue);
  return needItems(view, queue);
}

/** useNeedCount 是底部坞和标签上的数字，等于今天页列出的条数。 */
export function useNeedCount(): number {
  return useNeedItems().length;
}
