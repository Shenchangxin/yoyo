/**
 * 本文件拥有翻拍草稿在浏览器里的指针。不拥有剪辑工程，也不保存字幕或成片。
 * 它只改本地状态，不改服务端。
 */
import { create } from "zustand";
import type { CutProject } from "./cut-desk";

type RemakeDeskState = {
  draft: CutProject | null;
  setDraft: (draft: CutProject | null) => void;
  clear: () => void;
};

/** useRemakeDesk 保存当前翻拍草稿。没有 id 的对象会被记成空。 */
export const useRemakeDesk = create<RemakeDeskState>((set) => ({
  draft: null,
  /** 写入草稿；没有 id 时记成空。 */
  setDraft: (draft) => set({ draft: draft?.id ? draft : null }),
  /** 丢掉本地草稿指针。 */
  clear: () => set({ draft: null }),
}));
