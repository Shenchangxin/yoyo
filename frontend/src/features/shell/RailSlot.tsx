/**
 * 拥有：侧栏中间留给当前模块填的那一块。
 * 不拥有：填进去的名单。模块自己用 portal 放进来。
 */

import { createContext, useContext } from "react";

export const RailSlotContext = createContext<HTMLElement | null>(null);

/** 取出侧栏槽。没有槽时模块把名单画在自己的内容里。 */
export function useRailSlot() {
  return useContext(RailSlotContext);
}
