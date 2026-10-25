import { create } from "zustand";
import type { CutProject } from "./cut-desk";

type RemakeDeskState = {
  draft: CutProject | null;
  setDraft: (draft: CutProject | null) => void;
  clear: () => void;
};

export const useRemakeDesk = create<RemakeDeskState>((set) => ({
  draft: null,
  setDraft: (draft) => set({ draft: draft?.id ? draft : null }),
  clear: () => set({ draft: null }),
}));
