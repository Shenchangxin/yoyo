/**
 * 本文件拥有创作入口当前选中的模式，以及这张桌子是否已经从这里发过聊天。
 * 不拥有线程历史。它只改本地状态，不改服务端。
 */
import { create } from "zustand";

/** CreateMode 是创作入口的四种模式：成片、榜单、播客、访谈。 */
export type CreateMode = "clip" | "ranking" | "podcast" | "interview";

/** useCreateMode 记住当前模式，以及操作者是否从这张桌子发过聊天。 */
export const useCreateMode = create<{
  mode: CreateMode;
  /** 操作者从这张桌子发过聊天。欢迎线程上的历史不算。 */
  engaged: boolean;
  setMode: (mode: CreateMode) => void;
  setEngaged: (engaged: boolean) => void;
}>(/** set 写入模式或是否已发过聊天。 */ (set) => ({
  mode: "clip",
  engaged: false,
  /** 改当前创作模式。mode 写进状态。 */
  setMode: (mode) => set({ mode }),
  /** 记下这张桌子是否已经发过聊天。engaged 写进状态。 */
  setEngaged: (engaged) => set({ engaged }),
}));
