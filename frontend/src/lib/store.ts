import { create } from "zustand";
import type { Lab } from "./protocol";

export type InspTab = "diff" | "files" | "context" | "approvals";
export type DiffMode = "unified" | "split";

type UIState = {
  lab: Lab;
  inspector: boolean;
  palette: boolean;
  query: string;
  inspTab: InspTab;
  diffMode: DiffMode;
  plan: boolean;
  drafts: Record<string, string>;
  setupDismissed: boolean;
  setLab: (lab: Lab) => void;
  setInspector: (v: boolean | ((p: boolean) => boolean)) => void;
  setPalette: (v: boolean | ((p: boolean) => boolean)) => void;
  setQuery: (q: string) => void;
  setInspTab: (t: InspTab) => void;
  setDiffMode: (m: DiffMode) => void;
  setPlan: (v: boolean | ((p: boolean) => boolean)) => void;
  setDraft: (key: string, value: string) => void;
  patchDrafts: (patch: Record<string, string>) => void;
  setSetupDismissed: (v: boolean) => void;
};

function readDiffMode(): DiffMode {
  try {
    return localStorage.getItem("yoyo-diff-mode") === "split" ? "split" : "unified";
  } catch {
    return "unified";
  }
}

function readSetupDismissed(): boolean {
  try {
    return localStorage.getItem("yoyo-setup-dismissed") === "1";
  } catch {
    return false;
  }
}

export const useUI = create<UIState>((set) => ({
  lab: "agent",
  inspector: true,
  palette: false,
  query: "",
  inspTab: "diff",
  diffMode: readDiffMode(),
  plan: false,
  drafts: {},
  setupDismissed: readSetupDismissed(),
  setLab: (lab) => set({ lab }),
  setInspector: (v) => set((s) => ({ inspector: typeof v === "function" ? v(s.inspector) : v })),
  setPalette: (v) => set((s) => ({ palette: typeof v === "function" ? v(s.palette) : v })),
  setQuery: (query) => set({ query }),
  setInspTab: (inspTab) => set({ inspTab }),
  setDiffMode: (diffMode) => {
    try {
      localStorage.setItem("yoyo-diff-mode", diffMode);
    } catch {
      /* ignore */
    }
    set({ diffMode });
  },
  setPlan: (v) => set((s) => ({ plan: typeof v === "function" ? v(s.plan) : v })),
  setDraft: (key, value) => set((s) => ({ drafts: { ...s.drafts, [key]: value } })),
  patchDrafts: (patch) => set((s) => ({ drafts: { ...s.drafts, ...patch } })),
  setSetupDismissed: (setupDismissed) => {
    try {
      localStorage.setItem("yoyo-setup-dismissed", setupDismissed ? "1" : "0");
    } catch {
      /* ignore */
    }
    set({ setupDismissed });
  },
}));
