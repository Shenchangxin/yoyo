import { create } from "zustand";

export type CreateMode = "clip" | "ranking" | "podcast" | "interview";

export const useCreateMode = create<{ mode: CreateMode; setMode: (mode: CreateMode) => void }>((set) => ({
  mode: "clip",
  setMode: (mode) => set({ mode }),
}));
