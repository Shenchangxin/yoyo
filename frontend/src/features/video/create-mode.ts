import { create } from "zustand";

export type CreateMode = "clip" | "ranking" | "podcast" | "interview";

export const useCreateMode = create<{
  mode: CreateMode;
  /** The operator sent a chat from this desk. History on the welcome thread does not count. */
  engaged: boolean;
  setMode: (mode: CreateMode) => void;
  setEngaged: (engaged: boolean) => void;
}>((set) => ({
  mode: "clip",
  engaged: false,
  setMode: (mode) => set({ mode }),
  setEngaged: (engaged) => set({ engaged }),
}));
