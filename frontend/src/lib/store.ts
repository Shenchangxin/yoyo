import { create } from "zustand";
import type { Lab, Notice, SettingsTab, Surface } from "./protocol";

export type InspTab = "diff" | "files" | "context" | "approvals";
export type DiffMode = "unified" | "split";

type UIState = {
  lab: Lab;
  surface: Surface;
  settingsTab: SettingsTab;
  settingsSection: string;
  settingsNav: number;
  inspector: boolean;
  palette: boolean;
  query: string;
  inspTab: InspTab;
  diffMode: DiffMode;
  plan: boolean;
  drafts: Record<string, string>;
  setupDismissed: boolean;
  sidebarCollapsed: boolean;
  sidebarHover: boolean;
  notices: Notice[];
  noticesOpen: boolean;
  renameTick: number;
  setLab: (lab: Lab) => void;
  requestRename: () => void;
  openSettings: (tab?: SettingsTab, section?: string) => void;
  closeSettings: () => void;
  setSettingsTab: (tab: SettingsTab) => void;
  setInspector: (v: boolean | ((p: boolean) => boolean)) => void;
  setPalette: (v: boolean | ((p: boolean) => boolean)) => void;
  setQuery: (q: string) => void;
  setInspTab: (t: InspTab) => void;
  setDiffMode: (m: DiffMode) => void;
  setPlan: (v: boolean | ((p: boolean) => boolean)) => void;
  setDraft: (key: string, value: string) => void;
  patchDrafts: (patch: Record<string, string>) => void;
  setSetupDismissed: (v: boolean) => void;
  setSidebarCollapsed: (v: boolean | ((p: boolean) => boolean)) => void;
  setSidebarHover: (v: boolean) => void;
  pushNotice: (n: Notice) => void;
  setNoticesOpen: (v: boolean | ((p: boolean) => boolean)) => void;
  clearNotices: () => void;
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

function readCollapsed(): boolean {
  try {
    return localStorage.getItem("yoyo-sidebar-collapsed") === "1";
  } catch {
    return false;
  }
}

export const useUI = create<UIState>((set) => ({
  lab: "agent",
  surface: "agent",
  settingsTab: "general",
  settingsSection: "",
  settingsNav: 0,
  inspector: true,
  palette: false,
  query: "",
  inspTab: "diff",
  diffMode: readDiffMode(),
  plan: false,
  drafts: {},
  setupDismissed: readSetupDismissed(),
  sidebarCollapsed: readCollapsed(),
  sidebarHover: false,
  notices: [],
  noticesOpen: false,
  renameTick: 0,
  setLab: (lab) => set({ lab, surface: lab }),
  requestRename: () => set((s) => ({ renameTick: s.renameTick + 1 })),
  openSettings: (tab, section) =>
    set((s) => ({
      surface: "settings",
      settingsTab: tab || s.settingsTab || "general",
      settingsSection: section || "",
      settingsNav: Date.now(),
    })),
  closeSettings: () => set((s) => ({ surface: s.lab })),
  setSettingsTab: (settingsTab) => set({ settingsTab, settingsSection: "", settingsNav: Date.now() }),
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
  setSidebarCollapsed: (v) =>
    set((s) => {
      const sidebarCollapsed = typeof v === "function" ? v(s.sidebarCollapsed) : v;
      try {
        localStorage.setItem("yoyo-sidebar-collapsed", sidebarCollapsed ? "1" : "0");
      } catch {
        /* ignore */
      }
      return { sidebarCollapsed, sidebarHover: sidebarCollapsed ? s.sidebarHover : false };
    }),
  setSidebarHover: (sidebarHover) => set({ sidebarHover }),
  pushNotice: (n) => set((s) => ({ notices: [n, ...s.notices].slice(0, 30) })),
  setNoticesOpen: (v) => set((s) => ({ noticesOpen: typeof v === "function" ? v(s.noticesOpen) : v })),
  clearNotices: () => set({ notices: [] }),
}));
