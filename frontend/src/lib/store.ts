import { create } from "zustand";
import type { LearnLane, LearnScopeSpec } from "./learn-scope";
import type { AuthMode, CanvasFocus, HarnessTab, Lab, Notice, SettingsTab, Surface, Thread, ThreadChannel, VideoMode, VideoPane } from "./protocol";
import { isCanvasWorldPane, isYingcePane, threadChannel } from "./protocol";
import { noticeExpired } from "./notices";
import { isHarnessTab, labFromTab, surfaceForLab, tabFromLab } from "./surface";
import type { VoicePhase } from "../features/presence/types";

/**
 * 拥有：壳层界面状态：当前表面、抽屉、草稿、通知、视频面板，以及还没有对话时选中的权限。
 * 不拥有：会话正文、网络请求，以及这些表面的渲染。
 */

/** InspTab 检查器的四个页签。 */
export type InspTab = "files" | "browser" | "trace" | "context";
/** SkillsTab 技能表面的三个页签。 */
export type SkillsTab = "installed" | "market" | "packs";

/** coerceInspTab 识别四个页签。旧的改动、差异、队列和记忆页都归到文件页，其余返回空。v 与 files、browser、trace、context、changes、diff 等比较是否相等，有的分支原样返回。 */
export function coerceInspTab(v: string | null | undefined): InspTab | null {
  if (v === "files" || v === "browser" || v === "trace" || v === "context") return v;
  if (v === "changes" || v === "diff" || v === "queue" || v === "memory") return "files";
  return null;
}
/** DiffMode 差异的单栏或左右分栏。 */
export type DiffMode = "unified" | "split";

type UIState = {
  lab: Lab;
  surface: Surface;
  harnessTab: HarnessTab;
  settingsTab: SettingsTab;
  settingsSection: string;
  /** 打开设置小节时写入的时间戳，用来再次触发滚动。 */
  settingsNav: number;
  inspector: boolean;
  chatDock: boolean;
  palette: boolean;
  query: string;
  inspTab: InspTab;
  reviewFile: string;
  diffMode: DiffMode;
  plan: boolean;
  drafts: Record<string, string>;
  sidebarCollapsed: boolean;
  sidebarHover: boolean;
  outlineCollapsed: boolean;
  notices: Notice[];
  noticesOpen: boolean;
  /** 还没有对话时选中的权限，套到下一次创建的会话。 */
  pendingAuthMode: AuthMode | null;
  /** 请求重命名时递增，让标题进入编辑。 */
  renameTick: number;
  setLab: (lab: Lab) => void;
  openHarness: (tab?: HarnessTab) => void;
  setHarnessTab: (tab: HarnessTab) => void;
  requestRename: () => void;
  showConversation: () => void;
  openSettings: (tab?: SettingsTab, section?: string) => void;
  closeSettings: () => void;
  openSkills: () => void;
  openPacks: () => void;
  closeSkills: () => void;
  openPages: () => void;
  closePages: () => void;
  openIdol: () => void;
  closeIdol: () => void;
  openOrchestrator: () => void;
  closeOrchestrator: () => void;
  skillsTab: SkillsTab;
  setSkillsTab: (t: SkillsTab) => void;
  openVideo: () => void;
  closeVideo: () => void;
  videoMode: VideoMode;
  setVideoMode: (m: VideoMode) => void;
  videoPane: VideoPane;
  canvasFocus: CanvasFocus;
  /** 交给岛内路由的路径和查询。空表示用面板派生路径。 */
  videoEntry: string;
  openVideoPane: (pane: VideoPane, opts?: { canvasFocus?: CanvasFocus; entry?: string }) => void;
  openStudioDesk: () => void;
  videoBoard: boolean;
  setVideoBoard: (v: boolean | ((p: boolean) => boolean)) => void;
  canvasStage: boolean;
  setCanvasStage: (v: boolean | ((p: boolean) => boolean)) => void;
  studioDesk: boolean;
  footageOpen: boolean;
  setFootageOpen: (v: boolean) => void;
  agentThreadId: string;
  videoThreadId: string;
  learnThreadId: string;
  learnLane: LearnLane;
  learnScope: LearnScopeSpec;
  setLearnLane: (lane: LearnLane) => void;
  setLearnScope: (spec: LearnScopeSpec) => void;
  rememberThread: (t: Thread | null) => void;
  lastThreadId: (ch: ThreadChannel) => string;
  setSettingsTab: (tab: SettingsTab) => void;
  setInspector: (v: boolean | ((p: boolean) => boolean)) => void;
  setChatDock: (v: boolean | ((p: boolean) => boolean)) => void;
  setPalette: (v: boolean | ((p: boolean) => boolean)) => void;
  setQuery: (q: string) => void;
  setInspTab: (t: InspTab) => void;
  setReviewFile: (path: string) => void;
  setDiffMode: (m: DiffMode) => void;
  setPlan: (v: boolean | ((p: boolean) => boolean)) => void;
  setDraft: (key: string, value: string) => void;
  patchDrafts: (patch: Record<string, string>) => void;
  setSidebarCollapsed: (v: boolean | ((p: boolean) => boolean)) => void;
  setSidebarHover: (v: boolean) => void;
  setOutlineCollapsed: (v: boolean | ((p: boolean) => boolean)) => void;
  pushNotice: (n: Notice) => void;
  dismissNotice: (id: string) => void;
  dismissNoticesFor: (sessionId: string) => void;
  pruneNotices: (now?: number) => void;
  setNoticesOpen: (v: boolean | ((p: boolean) => boolean)) => void;
  clearNotices: () => void;
  setPendingAuthMode: (mode: AuthMode | null) => void;
  voicePhase: VoicePhase;
  setVoicePhase: (v: VoicePhase) => void;
  moduleLoading: boolean;
  setModuleLoading: (v: boolean) => void;
  showHarnessLab: boolean;
  setShowHarnessLab: (v: boolean) => void;
};

/** 读出各会话草稿。损坏或缺失时为空对象。 */
function readDrafts(): Record<string, string> {
  try {
    const raw = localStorage.getItem("yoyo-drafts");
    if (!raw) return {};
    const o = JSON.parse(raw) as Record<string, unknown>;
    if (!o || typeof o !== "object") return {};
    const out: Record<string, string> = {};
    for (const [k, v] of Object.entries(o)) {
      if (typeof v === "string" && v) out[k] = v.slice(0, 80_000);
    }
    return out;
  } catch {
    return {};
  }
}

/** 把非空草稿写回本地。写入失败时忽略。drafts 传给 Object.entries。 */
function writeDrafts(drafts: Record<string, string>) {
  try {
    const slim: Record<string, string> = {};
    for (const [k, v] of Object.entries(drafts)) {
      if (v) slim[k] = v.slice(0, 80_000);
    }
    localStorage.setItem("yoyo-drafts", JSON.stringify(slim));
  } catch {
    /* 配额不足时忽略 */
  }
}

/** 忙时 Enter 的默认动作。queue 排到下一轮，steer 插进当前这一步。 */
export type BusyEnter = "queue" | "steer";

/** 读忙时 Enter 的偏好。缺省是排队。 */
export function readBusyEnter(): BusyEnter {
  try {
    return localStorage.getItem("yoyo-busy-enter") === "steer" ? "steer" : "queue";
  } catch {
    return "queue";
  }
}

/** 记下忙时 Enter 的偏好。mode 为 steer 时 Enter 插话，否则排队。 */
export function writeBusyEnter(mode: BusyEnter) {
  try {
    localStorage.setItem("yoyo-busy-enter", mode);
  } catch { /* 隐私模式写不进去时，这一次会话仍用内存里的选择。 */ }
}

/** 读差异模式。缺省为单栏。 */
function readDiffMode(): DiffMode {
  try {
    return localStorage.getItem("yoyo-diff-mode") === "split" ? "split" : "unified";
  } catch {
    return "unified";
  }
}

/** 读侧栏是否折叠。 */
function readCollapsed(): boolean {
  try {
    return localStorage.getItem("yoyo-sidebar-collapsed") === "1";
  } catch {
    return false;
  }
}

/** 读大纲是否折叠。 */
function readOutlineCollapsed(): boolean {
  try {
    return localStorage.getItem("yoyo-outline-collapsed") === "1";
  } catch {
    return false;
  }
}

/** 读对话坞是否打开。 */
function readChatDock(): boolean {
  try {
    return localStorage.getItem("yoyo-chat-dock") === "1";
  } catch {
    return false;
  }
}

/** 读是否显示 Harness 实验室。 */
function readShowHarnessLab(): boolean {
  try {
    return localStorage.getItem("yoyo-show-harness-lab") === "1";
  } catch {
    return false;
  }
}

/** 读视频模式。无法识别时为剧集。 */
function readVideoMode(): VideoMode {
  try {
    const v = localStorage.getItem("yoyo-video-mode");
    if (v === "drama" || v === "canvas" || v === "creative") return v;
  } catch {
    /* 失败时忽略 */
  }
  return "drama";
}

/** 记下视频模式。mode 传给 localStorage.setItem。 */
function writeVideoMode(mode: VideoMode) {
  try {
    localStorage.setItem("yoyo-video-mode", mode);
  } catch {
    /* 失败时忽略 */
  }
}

/** 读学习泳道。认不出时回到今日。 */
function readLearnLane(): LearnLane {
  try {
    const v = localStorage.getItem("yoyo-learn-lane");
    if (v === "today" || v === "study" || v === "library" || v === "practice") return v;
  } catch {
    /* 失败时忽略 */
  }
  return "today";
}

/** 读某个通道上次的对话 id。key 传给 localStorage.getItem。 */
function readThreadId(key: string): string {
  try {
    return localStorage.getItem(key) || "";
  } catch {
    return "";
  }
}

/** 记下或清除某个通道的对话 id。key 传给 localStorage.setItem，传给 localStorage.removeItem。 */
function writeThreadId(key: string, id: string) {
  try {
    if (id) localStorage.setItem(key, id);
    else localStorage.removeItem(key);
  } catch {
    /* 失败时忽略 */
  }
}

/** 读 Harness 页签。无法识别时为总览。 */
function readHarnessTab(): HarnessTab {
  try {
    const v = localStorage.getItem("yoyo-harness-tab");
    if (v && isHarnessTab(v)) return v;
  } catch {
    /* 失败时忽略 */
  }
  return "overview";
}

/** 把布尔写成 1 或 0。key 传给 localStorage.setItem。on 为真时才继续。 */
function writeFlag(key: string, on: boolean) {
  try {
    localStorage.setItem(key, on ? "1" : "0");
  } catch {
    /* 失败时忽略 */
  }
}

/** 记下 Harness 页签。tab 传给 localStorage.setItem。 */
function writeHarnessTab(tab: HarnessTab) {
  try {
    localStorage.setItem("yoyo-harness-tab", tab);
  } catch {
    /* 失败时忽略 */
  }
}

/** useUI 壳层界面状态。会话正文和网络不在这里。 */
export const useUI = create<UIState>((set, get) => ({
  lab: "agent",
  surface: "agent",
  harnessTab: readHarnessTab(),
  settingsTab: "general",
  settingsSection: "",
  settingsNav: 0,
  inspector: false,
  chatDock: readChatDock(),
  palette: false,
  query: "",
  inspTab: "files",
  reviewFile: "",
  diffMode: readDiffMode(),
  plan: false,
  drafts: readDrafts(),
  sidebarCollapsed: readCollapsed(),
  sidebarHover: false,
  outlineCollapsed: readOutlineCollapsed(),
  notices: [],
  noticesOpen: false,
  pendingAuthMode: null,
  renameTick: 0,
  /** 切换实验室，并让表面和页签跟着走。离开智能体时记下页签。lab 与 agent 比较是否相等，与 agent 比较是否不等，传给 tabFromLab，传给 surfaceForLab，写入字段 lab。 */
  setLab: (lab) =>
    set((s) => {
      const harnessTab = lab === "agent" ? s.harnessTab : tabFromLab(lab);
      if (lab !== "agent") writeHarnessTab(harnessTab);
      return { lab, surface: surfaceForLab(lab), harnessTab };
    }),
  /** 打开 Harness 表面并落到指定页签。tab 传给 writeHarnessTab，传给 labFromTab，写入字段 harnessTab。 */
  openHarness: (tab = "overview") => {
    writeHarnessTab(tab);
    set({ lab: labFromTab(tab), surface: "harness", harnessTab: tab });
  },
  /** 在 Harness 内换页签，实验室跟着页签变。harnessTab 传给 writeHarnessTab，传给 labFromTab，写入字段 harnessTab。 */
  setHarnessTab: (harnessTab) => {
    writeHarnessTab(harnessTab);
    set({ harnessTab, lab: labFromTab(harnessTab), surface: "harness" });
  },
  /** 递增重命名计数，让会话标题进入编辑。 */
  requestRename: () => set((s) => ({ renameTick: s.renameTick + 1 })),
  /** 回到智能体对话，并收起视频侧的画板、舞台和工作室。 */
  showConversation: () => set({ lab: "agent", surface: "agent", videoBoard: false, canvasStage: false, studioDesk: false, footageOpen: false, videoPane: "chat", videoEntry: "" }),
  /** 打开设置表面。section 非空时记下要滚到的小节，并用当前时间戳触发导航。tab 为假时改用后备值，为假时用 general，写入字段 settingsTab。 */
  openSettings: (tab, section) =>
    set((s) => ({
      surface: "settings",
      settingsTab: tab || s.settingsTab || "general",
      settingsSection: section || "",
      settingsNav: Date.now(),
    })),
  /** 关掉设置，回到对话。 */
  closeSettings: () => get().showConversation(),
  /** 打开技能表面。 */
  openSkills: () => set({ surface: "skills" }),
  /** 打开技能表面并落到技能包页。 */
  openPacks: () => set({ surface: "skills", skillsTab: "packs" }),
  /** 关掉技能表面，回到对话。 */
  closeSkills: () => get().showConversation(),
  /** 打开页面表面。 */
  openPages: () => set({ surface: "pages" }),
  /** 关掉页面表面，回到对话。 */
  closePages: () => get().showConversation(),
  /** 打开 idol 表面。 */
  openIdol: () => set({ surface: "idol" }),
  /** 关掉 idol 表面，回到对话。 */
  closeIdol: () => get().showConversation(),
  /** 打开编排表面。工单和来源在这一页，不进对话栏。 */
  openOrchestrator: () => set({ surface: "orchestrator" }),
  /** 关掉编排表面，回到对话。 */
  closeOrchestrator: () => get().showConversation(),
  skillsTab: "market",
  /** 切换技能页签。skillsTab 写入字段 skillsTab。 */
  setSkillsTab: (skillsTab) => set({ skillsTab }),
  videoMode: readVideoMode(),
  agentThreadId: readThreadId("yoyo-agent-thread"),
  videoThreadId: readThreadId("yoyo-video-thread"),
  learnThreadId: readThreadId("yoyo-learn-thread"),
  learnLane: readLearnLane(),
  learnScope: { scope: "today" },
  /** 记下学习泳道。lane 写入本地，并写入字段 learnLane。 */
  setLearnLane: (learnLane) => {
    try {
      localStorage.setItem("yoyo-learn-lane", learnLane);
    } catch {
      /* 失败时忽略 */
    }
    set({ learnLane });
  },
  /** 记下当前学习子模块的助手范围。spec 写入字段 learnScope。 */
  setLearnScope: (learnScope) => set({ learnScope }),
  /** 按通道记住这段对话的 id。没有 id 则不动。t 读取 id，为空时不读，传给 threadChannel。 */
  rememberThread: (t) => {
    if (!t?.id) return;
    const ch = threadChannel(t);
    const key = ch === "video" ? "yoyo-video-thread" : ch === "learn" ? "yoyo-learn-thread" : "yoyo-agent-thread";
    writeThreadId(key, t.id);
    set(ch === "video" ? { videoThreadId: t.id } : ch === "learn" ? { learnThreadId: t.id } : { agentThreadId: t.id });
  },
  /** 取出该通道上次记住的对话。ch 为 video 时读 videoThreadId，为 learn 时读 learnThreadId，否则读 agentThreadId。 */
  lastThreadId: (ch) => (ch === "video" ? get().videoThreadId : ch === "learn" ? get().learnThreadId : get().agentThreadId),
  videoBoard: false,
  canvasStage: false,
  studioDesk: false,
  footageOpen: false,
  /** 打开或关掉素材抽屉。v 写入字段 footageOpen。 */
  setFootageOpen: (v) => set({ footageOpen: v }),
  videoPane: "chat",
  canvasFocus: "library",
  videoEntry: "",
  /** 打开视频表面的创作台，并清掉画板和入口。 */
  openVideo: () => set({ surface: "video", videoBoard: false, canvasStage: false, studioDesk: false, footageOpen: false, videoPane: "create", videoEntry: "" }),
  /** 关掉视频表面，回到对话。 */
  closeVideo: () => get().showConversation(),
  /** 打开工作室面板。 */
  openStudioDesk: () => get().openVideoPane("studio"),
  /**
   * 按面板切换视频表面。游戏面板改走创作台。工作室记下创作模式并打开桌面。剧集和画布世界分别记下对应模式。
   * pane 与 chat、create、studio、game、drama、canvas 比较是否相等，写入字段 videoPane。
   * opts 读取 entry、canvasFocus，为空时不读，传给 openVideoPane。
   */
  openVideoPane: (pane, opts) => {
    const videoEntry = opts?.entry || "";
    if (pane === "chat") {
      set({ surface: "video", videoPane: "chat", videoBoard: false, canvasStage: false, studioDesk: false, footageOpen: false, videoEntry: "" });
      return;
    }
    if (pane === "create") {
      set({ surface: "video", videoPane: "create", videoBoard: false, canvasStage: false, studioDesk: false, videoEntry });
      return;
    }
    if (pane === "studio") {
      writeVideoMode("creative");
      set({
        surface: "video",
        videoPane: "studio",
        videoMode: "creative",
        videoBoard: false,
        canvasStage: false,
        studioDesk: true,
        footageOpen: false,
        videoEntry: "",
      });
      return;
    }
    if (pane === "game") {
      get().openVideoPane("create", opts);
      return;
    }
    if (pane === "drama") {
      writeVideoMode("drama");
      set({ surface: "video", videoPane: "drama", videoMode: "drama", videoBoard: false, canvasStage: true, studioDesk: false, footageOpen: false, videoEntry });
      return;
    }
    writeVideoMode("canvas");
    set((s) => ({
      surface: "video" as const,
      videoPane: pane,
      videoMode: "canvas" as const,
      videoBoard: false,
      canvasStage: true,
      studioDesk: false,
      footageOpen: false,
      videoEntry,
      canvasFocus: pane === "canvas" ? (opts?.canvasFocus ?? "library") : s.canvasFocus,
    }));
  },
  /** 打开或关掉剧集画板。打开时进入剧集模式。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.videoBoard。 */
  setVideoBoard: (v) =>
    set((s) => {
      const videoBoard = typeof v === "function" ? v(s.videoBoard) : v;
      if (videoBoard) {
        writeVideoMode("drama");
        return { videoBoard: true, canvasStage: true, studioDesk: false, videoPane: "drama" as const, videoMode: "drama" as const, videoEntry: "" };
      }
      return { videoBoard: false, canvasStage: true, studioDesk: false, videoPane: "drama" as const, videoMode: "drama" as const, videoEntry: "" };
    }),
  /** 打开或关掉画布舞台。打开时留在岛内面板。关掉时岛内面板回到对话，并清掉入口。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.canvasStage。 */
  setCanvasStage: (v) =>
    set((s) => {
      const canvasStage = typeof v === "function" ? v(s.canvasStage) : v;
      if (canvasStage) {
        const videoPane = isYingcePane(s.videoPane) ? s.videoPane : "canvas";
        if (videoPane === "drama") writeVideoMode("drama");
        else if (isCanvasWorldPane(videoPane)) writeVideoMode("canvas");
        return {
          canvasStage: true,
          videoBoard: false,
          studioDesk: false,
          videoPane,
          videoMode: videoPane === "drama" ? ("drama" as const) : isCanvasWorldPane(videoPane) ? ("canvas" as const) : s.videoMode,
          canvasFocus: videoPane === "canvas" ? s.canvasFocus || "library" : s.canvasFocus,
        };
      }
      return {
        canvasStage: false,
        videoBoard: false,
        studioDesk: false,
        videoPane: isYingcePane(s.videoPane) ? ("chat" as const) : s.videoPane,
        videoEntry: "",
      };
    }),
  /** 按模式打开剧集、画布或工作室。videoMode 与 drama、canvas 比较是否相等。 */
  setVideoMode: (videoMode) => {
    if (videoMode === "drama") {
      get().openVideoPane("drama");
      return;
    }
    if (videoMode === "canvas") {
      get().openVideoPane("canvas");
      return;
    }
    get().openVideoPane("studio");
  },
  /** 切换设置页签，清掉小节，并用当前时间触发导航。settingsTab 写入字段 settingsTab。 */
  setSettingsTab: (settingsTab) => set({ settingsTab, settingsSection: "", settingsNav: Date.now() }),
  /** 打开或关掉检查器。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.inspector。 */
  setInspector: (v) => set((s) => ({ inspector: typeof v === "function" ? v(s.inspector) : v })),
  /** 打开或关掉对话坞，并写入本地。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.chatDock。 */
  setChatDock: (v) =>
    set((s) => {
      const chatDock = typeof v === "function" ? v(s.chatDock) : v;
      writeFlag("yoyo-chat-dock", chatDock);
      return { chatDock };
    }),
  /** 打开或关掉命令面板。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.palette。 */
  setPalette: (v) => set((s) => ({ palette: typeof v === "function" ? v(s.palette) : v })),
  /** 设置命令面板的查询。query 写入字段 query。 */
  setQuery: (query) => set({ query }),
  /** 设置检查器页签。无法识别时落到文件页。inspTab 传给 coerceInspTab。 */
  setInspTab: (inspTab) => set({ inspTab: coerceInspTab(inspTab) || "files" }),
  /** 设置检查器里正在审阅的文件。reviewFile 为假时用空字符串，写入字段 reviewFile。 */
  setReviewFile: (reviewFile) => set({ reviewFile: reviewFile || "" }),
  /** 设置差异模式并写入本地。diffMode 传给 localStorage.setItem，写入字段 diffMode。 */
  setDiffMode: (diffMode) => {
    try {
      localStorage.setItem("yoyo-diff-mode", diffMode);
    } catch {
      /* 失败时忽略 */
    }
    set({ diffMode });
  },
  /** 打开或关掉计划面板。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.plan。 */
  setPlan: (v) => set((s) => ({ plan: typeof v === "function" ? v(s.plan) : v })),
  /** 按键写入草稿并持久化。key 用作下标。value 写入字段 [key]。 */
  setDraft: (key, value) =>
    set((s) => {
      const drafts = { ...s.drafts, [key]: value };
      writeDrafts(drafts);
      return { drafts };
    }),
  /** 合并一批草稿并持久化。patch 展开传递。 */
  patchDrafts: (patch) =>
    set((s) => {
      const drafts = { ...s.drafts, ...patch };
      writeDrafts(drafts);
      return { drafts };
    }),
  /** 折叠或展开侧栏。展开时清掉悬停展开。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.sidebarCollapsed。 */
  setSidebarCollapsed: (v) =>
    set((s) => {
      const sidebarCollapsed = typeof v === "function" ? v(s.sidebarCollapsed) : v;
      writeFlag("yoyo-sidebar-collapsed", sidebarCollapsed);
      return { sidebarCollapsed, sidebarHover: sidebarCollapsed ? s.sidebarHover : false };
    }),
  /** 设置折叠侧栏上的悬停展开。sidebarHover 写入字段 sidebarHover。 */
  setSidebarHover: (sidebarHover) => set({ sidebarHover }),
  /** 折叠或展开回合大纲。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.outlineCollapsed。 */
  setOutlineCollapsed: (v) =>
    set((s) => {
      const outlineCollapsed = typeof v === "function" ? v(s.outlineCollapsed) : v;
      writeFlag("yoyo-outline-collapsed", outlineCollapsed);
      return { outlineCollapsed };
    }),
  // 同一条通知可能从实时补齐和流各到一次，按 id 只留一条。
  /** pushNotice 比窗口更旧的重放在这里丢掉，其余交给 pruneNotices 变旧。列表保留最新的 30 条。n 读取 id，传给 noticeExpired。 */
  pushNotice: (n) => set((s) => (s.notices.some((x) => x.id === n.id) || noticeExpired(n)
    ? s
    : { notices: [n, ...s.notices.filter((x) => !noticeExpired(x))].slice(0, 30) })),
  /** 按 id 关掉一条通知。 */
  dismissNotice: (id) => set((s) => ({ notices: s.notices.filter((n) => n.id !== id) })),
  /** 关掉属于该会话的通知。没有匹配时保持原状态。sessionId 与 n.sessionId 比较是否相等，与 n.sessionId 比较是否不等。 */
  dismissNoticesFor: (sessionId) => set((s) => (s.notices.some((n) => n.sessionId === sessionId)
    ? { notices: s.notices.filter((n) => n.sessionId !== sessionId) }
    : s)),
  /** 丢掉已过期的通知。没有变化时保持原状态。now 传给 noticeExpired。 */
  pruneNotices: (now = Date.now()) => set((s) => {
    const kept = s.notices.filter((n) => !noticeExpired(n, now));
    return kept.length === s.notices.length ? s : { notices: kept };
  }),
  /** 打开或关掉通知列表。v 用 typeof 看类型，作为分支结果，被调用，并传入 s.noticesOpen。 */
  setNoticesOpen: (v) => set((s) => ({ noticesOpen: typeof v === "function" ? v(s.noticesOpen) : v })),
  /** 清空通知。 */
  clearNotices: () => set({ notices: [] }),
  /** 记下还没有对话时选中的权限模式。mode 写入字段 pendingAuthMode。 */
  setPendingAuthMode: (mode) => set({ pendingAuthMode: mode }),
  voicePhase: "off",
  /** 设置语音阶段。voicePhase 写入字段 voicePhase。 */
  setVoicePhase: (voicePhase) => set({ voicePhase }),
  moduleLoading: false,
  /** 设置模块是否仍在加载。moduleLoading 写入字段 moduleLoading。 */
  setModuleLoading: (moduleLoading) => set({ moduleLoading }),
  showHarnessLab: readShowHarnessLab(),
  /** 是否显示 Harness 实验室入口，并写入本地。showHarnessLab 传给 writeFlag，写入字段 showHarnessLab。 */
  setShowHarnessLab: (showHarnessLab) => {
    writeFlag("yoyo-show-harness-lab", showHarnessLab);
    set({ showHarnessLab });
  },
}));
