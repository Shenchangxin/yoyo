import { create } from "zustand";
import type { HarnessTab, Lab, Notice, SettingsTab, Surface, Thread, ThreadChannel, VideoMode } from "./protocol";
import { threadChannel } from "./protocol";
import { isHarnessTab, labFromTab, surfaceForLab, tabFromLab } from "./surface";

export type InspTab = "diff" | "files" | "trace" | "queue" | "memory";
export type DiffMode = "unified" | "split";

type UIState = {
  lab: Lab;
  surface: Surface;
  harnessTab: HarnessTab;
  settingsTab: SettingsTab;
  settingsSection: string;
  settingsNav: number;
  inspector: boolean;
  chatDock: boolean;
  palette: boolean;
  query: string;
  inspTab: InspTab;
  diffMode: DiffMode;
  plan: boolean;
  drafts: Record<string, string>;
  sidebarCollapsed: boolean;
  sidebarHover: boolean;
  notices: Notice[];
  noticesOpen: boolean;
  renameTick: number;
  setLab: (lab: Lab) => void;
  openHarness: (tab?: HarnessTab) => void;
  setHarnessTab: (tab: HarnessTab) => void;
  requestRename: () => void;
  openSettings: (tab?: SettingsTab, section?: string) => void;
  closeSettings: () => void;
  openSkills: () => void;
  closeSkills: () => void;
  openVideo: () => void;
  closeVideo: () => void;
  videoMode: VideoMode;
  setVideoMode: (m: VideoMode) => void;
  videoBoard: boolean;
  setVideoBoard: (v: boolean | ((p: boolean) => boolean)) => void;
  agentThreadId: string;
  videoThreadId: string;
  rememberThread: (t: Thread | null) => void;
  lastThreadId: (ch: ThreadChannel) => string;
  setSettingsTab: (tab: SettingsTab) => void;
  setInspector: (v: boolean | ((p: boolean) => boolean)) => void;
  setChatDock: (v: boolean | ((p: boolean) => boolean)) => void;
  setPalette: (v: boolean | ((p: boolean) => boolean)) => void;
  setQuery: (q: string) => void;
  setInspTab: (t: InspTab) => void;
  setDiffMode: (m: DiffMode) => void;
  setPlan: (v: boolean | ((p: boolean) => boolean)) => void;
  setDraft: (key: string, value: string) => void;
  patchDrafts: (patch: Record<string, string>) => void;
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

function readCollapsed(): boolean {
  try {
    return localStorage.getItem("yoyo-sidebar-collapsed") === "1";
  } catch {
    return false;
  }
}

function readChatDock(): boolean {
  try {
    return localStorage.getItem("yoyo-chat-dock") === "1";
  } catch {
    return false;
  }
}

function readVideoMode(): VideoMode {
  try {
    const v = localStorage.getItem("yoyo-video-mode");
    if (v === "drama" || v === "canvas" || v === "creative") return v;
  } catch {
    /* ignore */
  }
  return "drama";
}

function writeVideoMode(mode: VideoMode) {
  try {
    localStorage.setItem("yoyo-video-mode", mode);
  } catch {
    /* ignore */
  }
}

function readThreadId(key: string): string {
  try {
    return localStorage.getItem(key) || "";
  } catch {
    return "";
  }
}

function writeThreadId(key: string, id: string) {
  try {
    if (id) localStorage.setItem(key, id);
    else localStorage.removeItem(key);
  } catch {
    /* ignore */
  }
}

function readHarnessTab(): HarnessTab {
  try {
    const v = localStorage.getItem("yoyo-harness-tab");
    if (v && isHarnessTab(v)) return v;
  } catch {
    /* ignore */
  }
  return "overview";
}

function writeFlag(key: string, on: boolean) {
  try {
    localStorage.setItem(key, on ? "1" : "0");
  } catch {
    /* ignore */
  }
}

function writeHarnessTab(tab: HarnessTab) {
  try {
    localStorage.setItem("yoyo-harness-tab", tab);
  } catch {
    /* ignore */
  }
}

export const useUI = create<UIState>((set, get) => ({
  lab: "agent",
  surface: "agent",
  harnessTab: readHarnessTab(),
  settingsTab: "general",
  settingsSection: "",
  settingsNav: 0,
  inspector: true,
  chatDock: readChatDock(),
  palette: false,
  query: "",
  inspTab: "diff",
  diffMode: readDiffMode(),
  plan: false,
  drafts: {},
  sidebarCollapsed: readCollapsed(),
  sidebarHover: false,
  notices: [],
  noticesOpen: false,
  renameTick: 0,
  setLab: (lab) =>
    set((s) => {
      const harnessTab = lab === "agent" ? s.harnessTab : tabFromLab(lab);
      if (lab !== "agent") writeHarnessTab(harnessTab);
      return { lab, surface: surfaceForLab(lab), harnessTab };
    }),
  openHarness: (tab = "overview") => {
    writeHarnessTab(tab);
    set({ lab: labFromTab(tab), surface: "harness", harnessTab: tab });
  },
  setHarnessTab: (harnessTab) => {
    writeHarnessTab(harnessTab);
    set({ harnessTab, lab: labFromTab(harnessTab), surface: "harness" });
  },
  requestRename: () => set((s) => ({ renameTick: s.renameTick + 1 })),
  openSettings: (tab, section) =>
    set((s) => ({
      surface: "settings",
      settingsTab: tab || s.settingsTab || "general",
      settingsSection: section || "",
      settingsNav: Date.now(),
    })),
  closeSettings: () => set((s) => ({ surface: surfaceForLab(s.lab) })),
  openSkills: () => set({ surface: "skills" }),
  closeSkills: () => set((s) => ({ surface: surfaceForLab(s.lab) })),
  videoMode: readVideoMode(),
  agentThreadId: readThreadId("yoyo-agent-thread"),
  videoThreadId: readThreadId("yoyo-video-thread"),
  rememberThread: (t) => {
    if (!t?.id) return;
    const ch = threadChannel(t);
    const key = ch === "video" ? "yoyo-video-thread" : "yoyo-agent-thread";
    writeThreadId(key, t.id);
    set(ch === "video" ? { videoThreadId: t.id } : { agentThreadId: t.id });
  },
  lastThreadId: (ch) => (ch === "video" ? get().videoThreadId : get().agentThreadId),
  videoBoard: false,
  openVideo: () => set({ surface: "video", videoBoard: false }),
  closeVideo: () => set((s) => ({ surface: surfaceForLab(s.lab), videoBoard: false })),
  setVideoBoard: (v) => set((s) => ({ videoBoard: typeof v === "function" ? v(s.videoBoard) : v })),
  setVideoMode: (videoMode) => {
    writeVideoMode(videoMode);
    set((s) => ({ videoMode, videoBoard: videoMode === "drama" ? s.videoBoard : false }));
  },
  setSettingsTab: (settingsTab) => set({ settingsTab, settingsSection: "", settingsNav: Date.now() }),
  setInspector: (v) => set((s) => ({ inspector: typeof v === "function" ? v(s.inspector) : v })),
  setChatDock: (v) =>
    set((s) => {
      const chatDock = typeof v === "function" ? v(s.chatDock) : v;
      writeFlag("yoyo-chat-dock", chatDock);
      return { chatDock };
    }),
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
  setSidebarCollapsed: (v) =>
    set((s) => {
      const sidebarCollapsed = typeof v === "function" ? v(s.sidebarCollapsed) : v;
      writeFlag("yoyo-sidebar-collapsed", sidebarCollapsed);
      return { sidebarCollapsed, sidebarHover: sidebarCollapsed ? s.sidebarHover : false };
    }),
  setSidebarHover: (sidebarHover) => set({ sidebarHover }),
  pushNotice: (n) => set((s) => ({ notices: [n, ...s.notices].slice(0, 30) })),
  setNoticesOpen: (v) => set((s) => ({ noticesOpen: typeof v === "function" ? v(s.noticesOpen) : v })),
  clearNotices: () => set({ notices: [] }),
}));
