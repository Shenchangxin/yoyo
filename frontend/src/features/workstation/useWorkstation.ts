/**
 * 拥有：会话列表、对话折叠、发送、视频车道和 harness 操作。
 * 不拥有：面板怎么画。会调用 api 读投影，并改会话、发送、停止、配置和视频绑定。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import * as api from "../../lib/client";
import { bannerError } from "../../lib/error";
import { noticeFromItem } from "../../lib/notices";
import { BROWSE_TRANSCRIPT_TURNS, HOT_TRANSCRIPT_TURNS, catchUpSession, dropTurnErrors, eventText, foldLiveIntoSeed, foldTurnErrors, lastUserTurns, replayEvents, subscribeItems, subscribeSession, subscribeSessions, subscribeState, subscribeStopped, unwrapEvent, userTurnCount, withLiveTail } from "../../lib/stream";
import { ConversationFold } from "../../lib/conversation-fold";
import { deriveTurnStatus } from "../../lib/turn-status";
import { pendingOutline, neighborTurn, itemMatchesTurn, itemTurnKey, outlineStatus, outlineTurnKey, turnJumpAliases, type OutlineTurn } from "../../lib/turn-outline";
import { deriveSessionStats } from "../../lib/session-stats";
import { asArray, num, str } from "../../lib/normalize";
import { pathReady, workspaceReady } from "../../lib/workspace";
import { applyLocale, useCopy } from "../../lib/i18n";
import { mergeKeymap, matchKey } from "../../lib/keymap";
import { coerceInspTab, useUI } from "../../lib/store";
import { adoptPendingAuthMode } from "../../lib/pending-auth";
import { workspacePreviewPath } from "../../lib/html-preview";
import { pathFromPatch } from "../../lib/artifact-preview";
import { toolArgs, toolName } from "../../lib/tool-summary";
import { HARNESS_TABS } from "../../lib/surface";
import { applyUiScale } from "../../lib/scale";
import { loadModelCatalog } from "../../lib/model-catalog";
import { parseDarkPalette, parseLightPalette, parseThemePref, useTheme } from "../../lib/theme";
import { isUserSkinId } from "../../lib/skin/schema";
import { readPopoutId } from "../../lib/popout";
import type { AppConfig, Approval, Attachment, ContextUsage, FileHit, HarborKind, Health, Hunk, Item, PackStatus, RunStatus, SessionTrace, SkillInfo, SpillBlob, Thread, ThreadChannel, VideoProject } from "../../lib/protocol";
import { channelForSurface, isSubagentThread, threadChannel } from "../../lib/protocol";
import type { LearnScopeSpec } from "../../lib/learn-scope";
import { learnBindingMatches, matchLearnThread, readLearnScopeMap } from "../../lib/learn-scope";
import { selectSurfaceThread } from "../../lib/pick-channel";
import { bindVideoThread } from "../video/bind";
import { cutPrompt, ingestClipUrl, installSendToCut, resourceIdFromSend, type SendToCutRequest } from "../video/open-cut";
import { ensureCanvasProject, useCanvasHost } from "../video/canvas-host/session";
import { asStudioProject, studioSessionId } from "../video/studio-session";
import { laneAcceptsSession, lastLaneThread, rememberLaneThread, sessionLane, type VideoLane } from "../video/video-lanes";
import { cutThreadId, isCutThread, rememberCutThread } from "../video/cut-threads";
import { defaultDramaStyle, useDramaSelection, useStudioSelection } from "../video/workshop-store";
import { useCutDesk, type CutProject } from "../video/cut-desk";
import { useCreateMode } from "../video/create-mode";
import { ingestVideoPath, isVideoPath, nativeFilePath } from "../video/ingest-clip";
import { useFilmDesk } from "../video/film-desk";
import { filmCommission, readCast, useFilmCast, type FilmMode } from "../video/film-cast";

/** 蛇形或驼峰键存在时用 num，缺省按 0。两个键都没有就沿用 prev。p 为真时才继续，按下标读取。snake 用作下标。camel 用作下标。 */
function sliceNum(p: Record<string, any>, snake: string, camel: string, prev?: number): number | undefined {
  if (p && (snake in p || camel in p)) return num(p[snake] ?? p[camel], 0);
  return prev;
}

/** 还没读到健康信息时的空值。ok 为假，金额为 0。 */
const emptyHealth: Health = { ok: false, harness: "", model: "", version: "", isolated: false, isolationKind: "", budgetUsd: 0, usageUsd: 0, workspaceReady: false };
/** emptyCfg 是还没读到配置时的本地默认值。provider 是 openai，uiScale 是 1，theme 是 system。 */
export const emptyCfg: AppConfig = {
  provider: "openai",
  model: "",
  baseUrl: "",
  workspace: "",
  videoWorkspace: "",
  autoAllow: false,
  maxBudgetUsd: 0,
  usdPerMtok: 0,
  models: [],
  closeToTray: false,
  companionEnabled: true,
  updateUrl: "",
  updateAutoCheck: true,
  keymap: {},
  locale: "",
  alwaysOnTop: false,
  startAtLogin: false,
  notificationsEnabled: true,
  notifyWhenUnfocusedOnly: false,
  uiScale: 1,
  showThinking: false,
  transcriptView: "standard",
  collapseTiming: "completion",
  updateChannel: "latest",
  theme: "system",
  paletteDark: "ink",
  paletteLight: "neutral",
  skin: "",
  gateMode: "manual",
  crashResume: true,
  searchUrl: "",
  searchKey: "",
};
/** 还没有上下文用量时的空值。各项计数为 0，layers 是空数组。 */
const emptyCtx: ContextUsage = { tokens: 0, budget: 0, window: 0, prefixTokens: 0, dynamicTokens: 0, schemaTokens: 0, providerPrompt: 0, note: "", layers: [], elided: 0 };

/** 从后往前留住最后一个工具调用或结果里能预览的工作区路径。没有就返回空字符串。items 用来遍历。 */
function latestWorkspaceHTMLPath(items: Item[]): string {
  let found = "";
  for (const it of items) {
    if (it.type !== "tool_call" && it.type !== "tool_result") continue;
    const name = toolName(it);
    const args = toolArgs(it);
    const preview = workspacePreviewPath(name, { ...args, url: args.url || it.payload?.path });
    if (preview) found = preview;
  }
  return found;
}

const LEARN_SCOPE_KEY = "yoyo-learn-scopes";

/** 读出学习子模块各自记住的会话。 */
function readLearnScopes(): Record<string, string> {
  try {
    return readLearnScopeMap(localStorage.getItem(LEARN_SCOPE_KEY));
  } catch {
    return {};
  }
}

/** 记下某个学习子模块的会话。scope 和 id 写进本地映射。 */
function writeLearnScope(scope: string, id: string) {
  if (!scope || !id) return;
  const all = readLearnScopes();
  all[scope] = id;
  try {
    localStorage.setItem(LEARN_SCOPE_KEY, JSON.stringify(all));
  } catch {
    /* 配额不足时忽略 */
  }
}

/** 没有 id 返回 null。kind 只认 drama、studio、game，其余写成 canvas。v 读取 id、kind、title、session_id、sessionId、episode_id 等，为空时不读。 */
function videoProjectOf(v: any): VideoProject | null {
  const id = str(v?.id);
  if (!id) return null;
  const raw = str(v?.kind);
  const kind: VideoProject["kind"] = raw === "drama" ? "drama" : raw === "studio" ? "studio" : raw === "game" ? "game" : "canvas";
  return {
    kind,
    id,
    title: str(v?.title),
    sessionId: str(v?.session_id || v?.sessionId),
    episodeId: str(v?.episode_id || v?.episodeId),
    updatedAt: str(v?.updated_at || v?.updatedAt),
    status: str(v?.status),
    gate: str(v?.gate),
  };
}

/** 上次有过的 id 先写成假，这次给出的 id 写成真。ids 用来遍历。prev 传给 Object.keys。 */
function runningMap(ids: string[], prev: Record<string, boolean> = {}): Record<string, boolean> {
  const next: Record<string, boolean> = {};
  for (const id of Object.keys(prev)) next[id] = false;
  for (const id of ids) next[id] = true;
  return next;
}

/** 流已经说回合结束之后，轮询仍可能回答 running。这 6000 毫秒内以结束事件为准，避免工作行和转圈又亮起来。 */
const SETTLE_GRACE_MS = 6000;

/** 没有结束时间返回假。距离结束不到 SETTLE_GRACE_MS 返回真。endedAt 按下标读取。id 用作下标。 */
function settled(endedAt: Record<string, number>, id: string): boolean {
  const t = endedAt[id];
  return !!t && Date.now() - t < SETTLE_GRACE_MS;
}

/** 到点拒绝，错误文字是标签加 timed out。原承诺先结束就清掉计时器。p 调用 then。ms 传给 window.setTimeout。label 拼进字符串。 */
function withTimeout<T>(p: Promise<T>, ms: number, label: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const t = window.setTimeout(() => reject(new Error(`${label} timed out`)), ms);
    p.then(
      (v) => { window.clearTimeout(t); resolve(v); },
      (e) => { window.clearTimeout(t); reject(e); },
    );
  });
}

/** 有 randomUUID 就用它。失败或没有时用 req、当前时间和一段随机十六进制。 */
function newRequestId(): string {
  try {
    if (typeof crypto !== "undefined" && crypto.randomUUID) return crypto.randomUUID();
  } catch {
    /* 忽略 */
  }
  return `req-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

/** 先折进回合错误，再只留最近 HOT_TRANSCRIPT_TURNS 个用户回合。变短了才 trim 折叠器。fold 去掉首尾空白。list 传给 foldTurnErrors。 */
function commitHot(fold: ConversationFold, list: Item[]): Item[] {
  const next = foldTurnErrors(list);
  const trimmed = lastUserTurns(next, HOT_TRANSCRIPT_TURNS);
  if (trimmed.length !== next.length) fold.trim(trimmed);
  return trimmed;
}

/**
 * 本地先放一条 source 为 ui 的用户消息。附件只留有名字或路径的 name、path、mime。key 以 ui: 开头。
 * sessionId 拼进字符串，写入字段 sessionId。
 * text 写入字段 text。
 * attachments 为假时改用后备值，做映射。
 * requestId 为假时改用后备值后，拼进字符串，写入字段 request_id。
 */
function localUser(sessionId: string, text: string, attachments?: Attachment[], requestId?: string): Item {
  const atts = (attachments || [])
    .map((a) => ({ name: a.name, path: a.path, mime: a.mime }))
    .filter((a) => a.name || a.path);
  const id = requestId || newRequestId();
  return {
    key: `ui:${sessionId}:${id}`,
    type: "user",
    sessionId,
    source: "ui",
    ts: new Date().toISOString(),
    text,
    name: "",
    delta: false,
    payload: { text, request_id: id, ...(atts.length ? { attachments: atts } : {}) },
  };
}

/** 读 yoyo-insp- 加会话 id。读失败或没有记录返回 null。 */
function readInspTab(id: string) {
  try {
    return coerceInspTab(localStorage.getItem(`yoyo-insp-${id}`));
  } catch {
    /* 忽略 */
  }
  return null;
}

/** useWorkstation 保住当前会话、对话、队列和实验室动作，并把它们交给 App。 */
export function useWorkstation() {
  const copy = useCopy();
  const { setPref, setDarkPalette, setLightPalette, setSkinOverlay, resolved } = useTheme();
  const lab = useUI((s) => s.lab);
  const setLab = useUI((s) => s.setLab);
  const surface = useUI((s) => s.surface);
  const harnessTab = useUI((s) => s.harnessTab);
  const setHarnessTab = useUI((s) => s.setHarnessTab);
  const openHarness = useUI((s) => s.openHarness);
  const openSettings = useUI((s) => s.openSettings);
  const closeSettings = useUI((s) => s.closeSettings);
  const openSkills = useUI((s) => s.openSkills);
  const closeSkills = useUI((s) => s.closeSkills);
  const openPages = useUI((s) => s.openPages);
  const closePages = useUI((s) => s.closePages);
  const openIdol = useUI((s) => s.openIdol);
  const closeIdol = useUI((s) => s.closeIdol);
  const openVideo = useUI((s) => s.openVideo);
  const closeVideo = useUI((s) => s.closeVideo);
  const showConversation = useUI((s) => s.showConversation);
  const videoBoard = useUI((s) => s.videoBoard);
  const setVideoBoard = useUI((s) => s.setVideoBoard);
  const canvasStage = useUI((s) => s.canvasStage);
  const setCanvasStage = useUI((s) => s.setCanvasStage);
  const inspector = useUI((s) => s.inspector);
  const setInspector = useUI((s) => s.setInspector);
  const chatDock = useUI((s) => s.chatDock);
  const setChatDock = useUI((s) => s.setChatDock);
  const palette = useUI((s) => s.palette);
  const setPalette = useUI((s) => s.setPalette);
  const query = useUI((s) => s.query);
  const setQuery = useUI((s) => s.setQuery);
  const inspTab = useUI((s) => s.inspTab);
  const setInspTab = useUI((s) => s.setInspTab);
  const setReviewFile = useUI((s) => s.setReviewFile);
  const diffMode = useUI((s) => s.diffMode);
  const setDiffMode = useUI((s) => s.setDiffMode);
  const sidebarCollapsed = useUI((s) => s.sidebarCollapsed);
  const setSidebarCollapsed = useUI((s) => s.setSidebarCollapsed);
  const sidebarHover = useUI((s) => s.sidebarHover);
  const setSidebarHover = useUI((s) => s.setSidebarHover);
  const notices = useUI((s) => s.notices);
  const pushNotice = useUI((s) => s.pushNotice);
  const dismissNotice = useUI((s) => s.dismissNotice);
  const clearNotices = useUI((s) => s.clearNotices);
  const pruneNotices = useUI((s) => s.pruneNotices);
  const renameTick = useUI((s) => s.renameTick);
  /** 通知自己过期。窗口按天算，每 60 秒清一次就够。 */
  useEffect(() => {
    const id = window.setInterval(() => pruneNotices(), 60_000);
    return () => window.clearInterval(id);
  }, [pruneNotices]);

  const [health, setHealth] = useState<Health>(emptyHealth);
  const [savedCfg, setSavedCfg] = useState<AppConfig>(emptyCfg);
  const [threads, setThreads] = useState<Thread[]>([]);
  const [videoProjects, setVideoProjects] = useState<VideoProject[]>([]);
  const canvasProjectId = useCanvasHost((s) => s.projectId);
  const dramaId = useDramaSelection((s) => s.dramaId);
  const videoProjectsRef = useRef(videoProjects);
  videoProjectsRef.current = videoProjects;
  const threadsRef = useRef(threads);
  threadsRef.current = threads;
  const learnScopeWanted = useRef("");
  const learnScopeJobs = useRef(new Map<string, Promise<Thread>>());
  /** 当前学习子模块钉住的会话。切表面和刷新都让它优先，避免被会话模块盖掉。 */
  const learnHold = useRef("");
  const studioThreadJobs = useRef<Record<string, Promise<Thread>>>({});
  const laneThreadJobs = useRef<Record<string, Promise<Thread>>>({});
  const cutThreadJobs = useRef<Record<string, Promise<Thread>>>({});
  const prepareWelcomeRef = useRef<() => Promise<Thread>>(async () => { throw new Error("welcome"); });
  const videoPane = useUI((s) => s.videoPane);
  const [active, setActive] = useState<Thread | null>(null);
  const activeRef = useRef(active);
  activeRef.current = active;
  const sendToCutRef = useRef<(detail: SendToCutRequest) => Promise<void>>(async () => {});
  const [items, setItems] = useState<Item[]>([]);
  const itemsAcc = useRef<Item[]>([]);
  const hotItemsRef = useRef<Item[]>([]);
  const foldRef = useRef(new ConversationFold());
  const itemsRaf = useRef(0);
  const headSeqRef = useRef(0);
  const olderRef = useRef(false);
  const browsingRef = useRef(false);
  const [outline, setOutline] = useState<OutlineTurn[]>([]);
  const outlineRef = useRef<OutlineTurn[]>([]);
  const [activeTurn, setActiveTurn] = useState("");
  const activeTurnRef = useRef("");
  const [jumpTo, setJumpTo] = useState<{ key: string; aliases?: string[]; nonce: number } | null>(null);
  const [latestNonce, setLatestNonce] = useState(0);
  const [older, setOlder] = useState(false);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const [idle, setIdle] = useState(false);
  const [queued, setQueued] = useState(0);
  const [queueItems, setQueueItems] = useState<{ id?: string; text?: string; plan?: boolean }[]>([]);
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [running, setRunning] = useState<Record<string, boolean>>({});
  const endedAt = useRef<Record<string, number>>({});
  const [runStatus, setRunStatus] = useState<RunStatus[]>([]);
  const [ctx, setCtx] = useState<ContextUsage>(emptyCtx);
  const [trace, setTrace] = useState<SessionTrace | null>(null);
  const [err, setErr] = useState("");
  const [diff, setDiff] = useState("");
  const [hunks, setHunks] = useState<Hunk[]>([]);
  const [hunkSel, setHunkSel] = useState<Record<string, boolean>>({});
  const lastApply = useRef<{ workspace: string; ids: string[]; snapshot: string } | null>(null);
  const [harness, setHarness] = useState<any>({});
  const [plugins, setPlugins] = useState<any>({});
  const [evalReport, setEvalReport] = useState<any>(null);
  const [bestReport, setBestReport] = useState<any>(null);
  const [harborErr, setHarborErr] = useState("");
  const [harborKind, setHarborKind] = useState<HarborKind>("suite");
  const [evolve, setEvolve] = useState<any>(null);
  const [overlay, setOverlay] = useState<any>(null);
  const [overlayK, setOverlayK] = useState(1);
  const [overlayMode, setOverlayMode] = useState("agg");
  const [overlayLibrary, setOverlayLibrary] = useState<any>(null);
  const [overlayReuseTurn, setOverlayReuseTurn] = useState(true);
  const [playbook, setPlaybook] = useState<any>(null);
  const [tree, setTree] = useState<any[]>([]);
  const [labBusy, setLabBusy] = useState<string | null>(null);
  const [evolveK, setEvolveK] = useState(3);
  const [evolveRounds, setEvolveRounds] = useState(1);
  const [evolveSealed, setEvolveSealed] = useState(false);
  const [evolveBehavior, setEvolveBehavior] = useState(false);
  const [evolveIndex, setEvolveIndex] = useState(false);
  const [evolveBaselines, setEvolveBaselines] = useState(false);
  const [evolveMaxUsd, setEvolveMaxUsd] = useState(0);
  const [bonModels, setBonModels] = useState("");
  const [diffA, setDiffA] = useState("");
  const [diffB, setDiffB] = useState("");
  const [diffOut, setDiffOut] = useState<any>(null);
  const [booted, setBooted] = useState(false);
  const bootedRef = useRef(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [aboutInfo, setAboutInfo] = useState<Record<string, any>>({});
  const [pendingDelete, setPendingDelete] = useState<Thread | null>(null);
  const [files, setFiles] = useState<FileHit[]>([]);
  const [searchHits, setSearchHits] = useState<Thread[] | null>(null);
  const [showArchived, setShowArchived] = useState(false);
  const [skills, setSkills] = useState<SkillInfo[]>([]);
  const [packs, setPacks] = useState<PackStatus[]>([]);
  const activeIdRef = useRef("");
  const [logs, setLogs] = useState<any>(null);
  const [journal, setJournal] = useState<any>(null);
  const [doctor, setDoctor] = useState<any>(null);
  const [vault, setVault] = useState<any>({});
  const [pendingQuit, setPendingQuit] = useState(false);
  const handlers = useRef({
    /** slash 函数体是空的。_cmd 函数体内没有读取。_rest 函数体内没有读取。 */
    slash: async (_cmd: string, _rest: string) => {},
    /** command 函数体是空的。_c 函数体内没有读取。 */
    command: (_c: string) => {},
    quit: () => {},
    /** focus 函数体是空的。_id 函数体内没有读取。 */
    focus: (_id: string) => {},
    /** route 函数体是空的。_payload 函数体内没有读取。 */
    route: (_payload: Record<string, unknown>) => {},
    /** cut 函数体是空的。_payload 函数体内没有读取。 */
    cut: (_payload: Record<string, unknown>) => {},
  });

  const activeId = active?.id || "";
  activeIdRef.current = activeId;
  const draftKey = activeId || "_new";
  const threadRunning = !!running[activeId];
  const anyRun = Object.values(running).some(Boolean);
  const needsSetup = !workspaceReady(health.workspaceReady, savedCfg.workspace);
  const outlineTurns = useMemo(() => pendingOutline(outline, items), [outline, items]);
  const sessionStats = useMemo(
    () => deriveSessionStats({ outlineTurns, items, trace }),
    [outlineTurns, items, trace],
  );
  const outlineState = useMemo(
    () => outlineStatus({ running: threadRunning, waiting: approvals.length > 0 }),
    [threadRunning, approvals.length],
  );
  const turnStatus = useMemo(
    () => deriveTurnStatus(items, threadRunning),
    [items, threadRunning],
  );
  outlineRef.current = outlineTurns;
  activeTurnRef.current = activeTurn;

  /** 没有当前会话就写成 emptyCtx。请求失败就留着上次的值。 */
  const refreshCtx = useCallback(() => {
    if (!activeId) {
      setCtx(emptyCtx);
      return;
    }
    api.contextUsage(activeId).then(setCtx).catch(() => {});
  }, [activeId]);

  /** 没有当前会话就清空。请求失败留着上次的轨迹。 */
  const refreshTrace = useCallback(async () => {
    if (!activeId) {
      setTrace(null);
      return;
    }
    try {
      setTrace(await api.sessionTrace(activeId));
    } catch {
      /* 失败就留着上次的投影 */
    }
  }, [activeId]);

  /** 没有当前会话时返回这个 id、0 字节、空文本、未截断。blobID 传给 api.spillBlob，写入字段 id。 */
  const loadSpill = useCallback(async (blobID: string): Promise<SpillBlob> => {
    if (!activeId) return { id: blobID, bytes: 0, text: "", truncated: false };
    return api.spillBlob(activeId, blobID);
  }, [activeId]);

  /** 没有当前会话就清空目录。回来时会话已经换了就丢掉结果。失败留着上次的目录。 */
  const refreshOutline = useCallback(async () => {
    if (!activeId) {
      setOutline([]);
      return;
    }
    try {
      const rows = await api.trajectoryOutline(activeId);
      if (activeIdRef.current !== activeId) return;
      setOutline(rows);
    } catch {
      /* 失败就留着上次的目录 */
    }
  }, [activeId]);

  /** 没有当前会话就返回。正在回看且热列表是空的才拉一页。然后把跟随序号设成当前时间。 */
  const jumpToLatest = useCallback(async () => {
    if (!activeId) return;
    if (browsingRef.current) {
      browsingRef.current = false;
      if (!hotItemsRef.current.length) {
        const page = await api.trajectoryPage(activeId);
        if (activeIdRef.current !== activeId) return;
        hotItemsRef.current = commitHot(foldRef.current, replayEvents(page.events));
        foldRef.current.replace(hotItemsRef.current);
        headSeqRef.current = page.headSeq || 0;
        olderRef.current = !!page.older;
        setOlder(!!page.older);
      }
      itemsAcc.current = hotItemsRef.current;
      setItems(hotItemsRef.current.slice());
    }
    setLatestNonce(Date.now());
  }, [activeId]);

  /** 没有会话或没有回合就返回。这条还没挂上且 seq 大于 0 时，按该 seq 取一页并进入回看。turn 读取 seq，为假时走另一支，传给 outlineTurnKey，传给 turnJumpAliases，传给 itemMatchesTurn。 */
  const jumpToTurn = useCallback(async (turn: OutlineTurn) => {
    if (!activeId || !turn) return;
    const key = outlineTurnKey(turn);
    const aliases = turnJumpAliases(turn);
    setActiveTurn(key);
    const mounted = itemsAcc.current.some((it) => itemMatchesTurn(it, turn) || aliases.includes(itemTurnKey(it)));
    if (!mounted && turn.seq > 0) {
      const page = await api.trajectoryAround(activeId, turn.seq, HOT_TRANSCRIPT_TURNS);
      if (activeIdRef.current !== activeId) return;
      browsingRef.current = true;
      itemsAcc.current = lastUserTurns(replayEvents(page.events), BROWSE_TRANSCRIPT_TURNS);
      headSeqRef.current = page.headSeq || 0;
      olderRef.current = !!page.older;
      setOlder(!!page.older);
      setItems(withLiveTail(itemsAcc.current, hotItemsRef.current));
    }
    setJumpTo({ key, aliases, nonce: Date.now() });
  }, [activeId]);

  /** 用横幅文案同时写入 err 并弹出提示。e 传给 api.errMessage。 */
  const fail = (e: unknown) => {
    const msg = bannerError(api.errMessage(e), copy.transcript);
    setErr(msg);
    toast.error(msg);
  };

  /** 空 id 直接返回。记下现在，并把这个会话的 running 从真改成假。 */
  const markEnded = useCallback((id: string) => {
    if (!id) return;
    endedAt.current[id] = Date.now();
    setRunning((m) => (m[id] ? { ...m, [id]: false } : m));
  }, []);

  /** 空 id 直接返回。清掉结束时间，并把这个会话标成正在跑。 */
  const markStarted = useCallback((id: string) => {
    if (!id) return;
    delete endedAt.current[id];
    setRunning((m) => ({ ...m, [id]: true }));
  }, []);

  /** 仍在宽限期内的 id 不记成正在跑。请求失败就忽略。 */
  const syncRunning = useCallback(async () => {
    try {
      const [h, ids, status] = await Promise.all([
        api.health().catch(() => null),
        api.runningIDs(),
        api.runningStatus().catch(() => [] as RunStatus[]),
      ]);
      if (h) setHealth((cur) => ({ ...cur, ...h }));
      const live = ids.filter((id) => !settled(endedAt.current, id));
      setRunning((prev) => runningMap(live, prev));
      setRunStatus(status.filter((s) => !settled(endedAt.current, s.id)));
    } catch {
      /* 忽略 */
    }
  }, []);

  /** 第一次启动限时 8000 毫秒，之后 20000。弹出窗口只留那个会话。第一次失败才提示。结束时总会标成已启动。手册、归档、密钥和上次评测失败就跳过。 */
  const refresh = useCallback(async () => {
    const first = !bootedRef.current;
    try {
      const [h, c, list, hs, pl] = await withTimeout(Promise.all([
        api.health().catch(() => null),
        api.getConfig().catch(() => null),
        api.listSessions(),
        api.harness().catch(() => ({})),
        api.plugins().catch(() => ({})),
      ]), first ? 8000 : 20000, first ? "boot" : "refresh");
      if (h) setHealth(h);
      if (c) setSavedCfg(c);
      setThreads(list);
      setHarness(hs);
      setPlugins(pl);
      setActive((cur) => {
        const pop = readPopoutId();
        if (pop) return list.find((t) => t.id === pop) || cur || list[0] || null;
        const ch = channelForSurface(useUI.getState().surface);
        return selectSurfaceThread(list, ch, useUI.getState().lastThreadId(ch), cur, learnHold.current);
      });
      setBooted(true);
      try { setPlaybook(await api.playbook()); } catch { /* 可选，失败就跳过 */ }
      try { setTree(await api.archive()); } catch { /* 可选，失败就跳过 */ }
      try { setVault(await api.keyStatus()); } catch { /* 可选，失败就跳过 */ }
      try {
        const last = await api.lastEval();
        if (last && (last.suite || last.Suite || last.results || last.Results)) setEvalReport(last);
      } catch { /* 可选，失败就跳过 */ }
      try {
        const last = await api.lastEvolve();
        if (last && (last.tried || last.Tried || last.compare || last.Compare || last.promoted || last.Promoted)) setEvolve(last);
      } catch { /* 可选，失败就跳过 */ }
      await syncRunning();
    } catch (e) {
      if (first) fail(e);
    } finally {
      bootedRef.current = true;
      setBooted(true);
    }
  }, [syncRunning]);

  useEffect(() => {
    if (savedCfg.locale) applyLocale(savedCfg.locale);
    applyUiScale(savedCfg.uiScale);
    const theme = parseThemePref(savedCfg.theme);
    if (theme) setPref(theme);
    setDarkPalette(parseDarkPalette(savedCfg.paletteDark));
    setLightPalette(parseLightPalette(savedCfg.paletteLight));
    const skin = savedCfg.skin || "";
    if (!isUserSkinId(skin)) {
      setSkinOverlay(null);
      return;
    }
    void api.resolveSkin(skin, resolved).then((r) => {
      setSkinOverlay({
        id: r.id,
        tokens: r.tokens,
        materials: r.materials,
        wallpaperUrl: r.wallpaperUrl,
        wallpaperFile: r.wallpaperFile,
        fonts: r.fonts,
        preserveEvidence: r.preserveEvidence,
      });
    }).catch(() => {
      setSkinOverlay(null);
      void api.activateSkin("");
      toast.error(copy.settings.skinBroken);
    });
  }, [savedCfg.locale, savedCfg.uiScale, savedCfg.theme, savedCfg.paletteDark, savedCfg.paletteLight, savedCfg.skin, resolved, setPref, setDarkPalette, setLightPalette, setSkinOverlay, copy.settings.skinBroken]);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => { void loadModelCatalog(); }, []);
  useEffect(() => subscribeSessions(refresh), [refresh]);
  useEffect(() => subscribeState(() => { void refresh(); }), [refresh]);
  useEffect(() => {
    if (active) useUI.getState().rememberThread(active);
  }, [active?.id]);
  useEffect(() => installSendToCut((detail) => sendToCutRef.current(detail)), []);
  useEffect(() => {
    if (surface !== "video" || videoPane !== "create") return;
    const id = activeId;
    if (!id) return;
    useCutDesk.getState().noteSession(id);
    let alive = true;
    void api.video.cutSession(id).then((row) => {
      if (!alive) return;
      const project = row && row.id ? row as CutProject : null;
      useCutDesk.getState().adoptServer(id, project?.source_hash ? project : null);
    }).catch(() => {});
    return () => { alive = false; };
  }, [surface, videoPane, activeId]);
  useEffect(() => {
    const ch = channelForSurface(surface);
    const preferred = useUI.getState().lastThreadId(ch);
    setActive((cur) => selectSurfaceThread(threadsRef.current, ch, preferred, cur, learnHold.current));
  }, [surface]);
  /** 失败时提示，并返回当前已经有的列表。 */
  const loadVideoHistory = useCallback(async () => {
    try {
      const raw = await api.video.history();
      const list = asArray(Array.isArray(raw) ? raw : raw?.items || raw?.projects || []).map(videoProjectOf).filter((p): p is VideoProject => !!p);
      videoProjectsRef.current = list;
      setVideoProjects(list);
      return list;
    } catch (e) {
      toast.error(api.errMessage(e));
      return videoProjectsRef.current;
    }
  }, []);
  useEffect(() => {
    if (surface !== "video") return;
    void loadVideoHistory();
  }, [surface, canvasProjectId, dramaId, loadVideoHistory]);
  useEffect(() => {
    if (surface !== "video" || !active?.id || threadChannel(active) !== "video") return;
    void restoreVideoProject(active.id);
  }, [surface, active?.id]);
  useEffect(() => {
    if (surface !== "video") return;
    void adoptVideoLane(videoPane);
  }, [surface, videoPane]);
  useEffect(() => {
    if (!booted) return;
    const path = active?.workspace || savedCfg.workspace;
    api.listSkills(path).then(setSkills).catch(() => {});
    api.listPacks(path).then(setPacks).catch(() => {});
  }, [booted, active?.workspace, savedCfg.workspace]);

  /** 按当前工作区重拉技能和包。失败就留着上次的列表。 */
  const reloadSkills = useCallback(() => {
    const path = active?.workspace || savedCfg.workspace;
    api.listSkills(path).then(setSkills).catch(() => {});
    api.listPacks(path).then(setPacks).catch(() => {});
  }, [active?.workspace, savedCfg.workspace]);
  useEffect(() => {
    if (surface !== "settings") return;
    api.logs().then(setLogs).catch(() => {});
    api.journal().then(setJournal).catch(() => {});
    api.doctor().then(setDoctor).catch(() => {});
  }, [surface]);

  useEffect(() => {
    if (!activeId) return;
    setReviewFile("");
    const saved = readInspTab(activeId);
    if (saved) setInspTab(saved);
  }, [activeId, setInspTab, setReviewFile]);

  useEffect(() => {
    if (!activeId) return;
    try {
      localStorage.setItem(`yoyo-insp-${activeId}`, inspTab);
    } catch {
      /* 忽略 */
    }
  }, [activeId, inspTab]);

  useEffect(() => {
    if (import.meta.env.VITE_E2E) return;
    const timer = window.setInterval(() => {
      void api.health().then((h) => setHealth((cur) => ({ ...cur, ...h }))).catch(() => {});
    }, 4000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    void refreshTrace();
  }, [refreshTrace]);

  useEffect(() => {
    if (!inspector) return;
    if (inspTab !== "trace" && inspTab !== "context") return;
    void refreshTrace();
    if (!threadRunning) return;
    const timer = window.setInterval(() => { void refreshTrace(); }, 4000);
    return () => window.clearInterval(timer);
  }, [inspector, inspTab, refreshTrace, threadRunning]);

  useEffect(() => {
    if (!activeId || !threadRunning) return;
    if (inspector && (inspTab === "trace" || inspTab === "context")) return;
    const timer = window.setInterval(() => { void refreshTrace(); }, 4000);
    return () => window.clearInterval(timer);
  }, [activeId, threadRunning, refreshTrace, inspector, inspTab]);

  useEffect(() => {
    if (import.meta.env.VITE_E2E) return;
    if (typeof window !== "undefined" && !(window as any)._wails) return;
    let offs: Array<() => void> = [];
    let alive = true;
    (async () => {
      try {
        const mod: any = await import("@wailsio/runtime");
        const Events = mod.Events;
        if (!alive || !Events?.On) return;
        offs.push(Events.On("yoyo:about", (info: any) => { setAboutInfo(unwrapEvent(info) || {}); setAboutOpen(true); }));
        offs.push(Events.On("yoyo:workspace", (path: any) => {
          const p = eventText(path);
          if (!p) return;
          setSavedCfg((cfg) => {
            const next = { ...cfg, workspace: p };
            void api.setConfig(next);
            return next;
          });
          const id = activeIdRef.current;
          if (!id) return;
          void api.setSessionWorkspace(id, p).then((t) => {
            setActive(t);
            setThreads((list) => list.map((x) => (x.id === t.id ? { ...x, ...t } : x)));
          }).catch(() => {});
        }));
        offs.push(Events.On("yoyo:command", (cmd: any) => handlers.current.command(eventText(cmd))));
        offs.push(Events.On("yoyo:quit", () => handlers.current.quit()));
        offs.push(Events.On("yoyo:focus", (session: any) => handlers.current.focus(eventText(session))));
        offs.push(Events.On("yoyo:doctor", (info: any) => { setDoctor(unwrapEvent(info) || {}); openSettings("advanced", "advanced-doctor"); }));
        offs.push(Events.On("yoyo:logs", (info: any) => { setLogs(unwrapEvent(info) || {}); openSettings("advanced", "advanced-logs"); }));
        offs.push(Events.On("yoyo:item", (ev: any) => {
          const kind = String(ev?.item_kind || ev?.ItemKind || "");
          const payload = (ev?.payload || ev?.Payload || {}) as Record<string, unknown>;
          if (kind === "video_route") handlers.current.route(payload);
          if (kind === "cut_state") handlers.current.cut(payload);
        }));
      } catch {
        /* 不在桌面壳里 */
      }
    })();
    return () => {
      alive = false;
      offs.forEach((o) => o());
    };
  }, [openSettings]);

  useEffect(() => subscribeItems((item) => {
    /** 一轮开始时循环正好发一条 system。turn_end 之后马上被踢起的排队回合会立刻取消宽限。 */
    if (item.type === "system" || (item.type === "user" && item.source === "user")) {
      markStarted(item.sessionId);
    }
    if (item.type === "turn_end" || item.type === "error") {
      markEnded(item.sessionId);
    }
    if (item.type === "approval") {
      api.approvals().then(setApprovals).catch(() => {});
    }
    if (item.type === "subagent") {
      void refresh();
    }
    const viewing = item.sessionId === activeIdRef.current && typeof document !== "undefined" && document.hasFocus();
    const notice = noticeFromItem(item, copy, viewing);
    if (notice) pushNotice(notice);
  }), [pushNotice, markEnded, markStarted, refresh, copy]);

  useEffect(() => subscribeStopped((id) => markEnded(id)), [markEnded]);

  useEffect(() => {
    /** 页面隐藏时标成空闲，并把渲染内存设成 low。可见时设成 normal。 */
    const onVis = () => {
      const hidden = typeof document !== "undefined" && document.hidden;
      setIdle(hidden);
      void api.setRendererMemory(hidden ? "low" : "normal");
    };
    document.addEventListener("visibilitychange", onVis);
    return () => document.removeEventListener("visibilitychange", onVis);
  }, []);

  useEffect(() => {
    if (!activeId) {
      itemsAcc.current = [];
      hotItemsRef.current = [];
      foldRef.current = new ConversationFold();
      browsingRef.current = false;
      setItems([]);
      setOlder(false);
      olderRef.current = false;
      headSeqRef.current = 0;
      setOutline([]);
      setActiveTurn("");
      setCtx(emptyCtx);
      setFiles([]);
      return;
    }
    itemsAcc.current = [];
    hotItemsRef.current = [];
    foldRef.current = new ConversationFold();
    browsingRef.current = false;
    setItems([]);
    setQueued(0);
    setQueueItems([]);
    setCtx(emptyCtx);
    setOlder(false);
    olderRef.current = false;
    headSeqRef.current = 0;
    setOutline([]);
    setActiveTurn("");
    setFiles([]);
    /** 正在回看时把热尾接到回看列表上。否则直接用热列表的副本。 */
    const flushItems = () => {
      itemsRaf.current = 0;
      if (browsingRef.current) {
        setItems(withLiveTail(itemsAcc.current, hotItemsRef.current));
      } else {
        setItems(hotItemsRef.current.slice());
      }
    };
    /** 立刻刷新会清掉已经排上的 50 毫秒。已经排过就不再排第二次。immediate 为真时才继续。 */
    const scheduleItems = (immediate: boolean) => {
      if (immediate) {
        if (itemsRaf.current) {
          window.clearTimeout(itemsRaf.current);
          itemsRaf.current = 0;
        }
        flushItems();
        return;
      }
      if (itemsRaf.current) return;
      itemsRaf.current = window.setTimeout(flushItems, 50);
    };
    const unsub = subscribeSession(
      activeId,
      (item) => {
        hotItemsRef.current = commitHot(foldRef.current, foldRef.current.accept(item));
        if (item.type === "user" && item.source !== "steer") {
          if (!browsingRef.current) itemsAcc.current = hotItemsRef.current;
          void refreshOutline();
        } else if (!browsingRef.current) {
          itemsAcc.current = hotItemsRef.current;
        }
        const delayed = item.type === "tool_result"
          || item.type === "tool_call"
          || item.type === "assistant_stream"
          || (item.type === "assistant" && !!(item.delta || item.payload?.streaming || item.payload?.delta));
        scheduleItems(!delayed);
        if (item.type === "compaction") {
          const p = item.payload || {};
          setCtx((prev) => ({
            ...prev,
            tokens: num(p.tokens, prev.tokens),
            budget: num(p.budget, prev.budget),
            window: num(p.window, prev.window || 0) || prev.window,
            prefixTokens: num(p.prefix_tokens ?? p.prefixTokens, prev.prefixTokens || 0) || prev.prefixTokens,
            dynamicTokens: num(p.dynamic_tokens ?? p.dynamicTokens, prev.dynamicTokens || 0) || prev.dynamicTokens,
            schemaTokens: num(p.schema_tokens ?? p.schemaTokens, prev.schemaTokens || 0) || prev.schemaTokens,
            identityTokens: sliceNum(p, "identity_tokens", "identityTokens", prev.identityTokens),
            rulesTokens: sliceNum(p, "rules_tokens", "rulesTokens", prev.rulesTokens),
            catalogTokens: sliceNum(p, "catalog_tokens", "catalogTokens", prev.catalogTokens),
            skillBodyTokens: sliceNum(p, "skill_body_tokens", "skillBodyTokens", prev.skillBodyTokens),
            memoryTokens: sliceNum(p, "memory_tokens", "memoryTokens", prev.memoryTokens),
            todayTokens: sliceNum(p, "today_tokens", "todayTokens", prev.todayTokens),
            notesTokens: sliceNum(p, "notes_tokens", "notesTokens", prev.notesTokens),
            chatTokens: sliceNum(p, "chat_tokens", "chatTokens", prev.chatTokens),
            slicesKnown: ("chat_tokens" in p || "chatTokens" in p || "identity_tokens" in p || "identityTokens" in p) || prev.slicesKnown,
            providerPrompt: num(p.provider_prompt ?? p.providerPrompt, prev.providerPrompt || 0) || prev.providerPrompt,
            cachedTokens: num(p.cached_tokens ?? p.cachedTokens, prev.cachedTokens || 0),
            cacheReported: !!(p.cache_reported ?? p.cacheReported ?? prev.cacheReported),
            cacheStable: !!(p.cache_stable ?? p.cacheStable ?? prev.cacheStable),
            trigger: str(p.trigger, prev.trigger || ""),
            hydrated: num(p.hydrated, prev.hydrated || 0),
            note: str(p.note, prev.note),
            layers: Array.isArray(p.layers) ? p.layers.map(String) : prev.layers,
            elided: num(p.elided, prev.elided),
          }));
        }
        if (item.type === "approval") {
          api.approvals().then(setApprovals).catch(() => {});
        }
        if (item.type === "tool_call") {
          const name = toolName(item);
          const args = toolArgs(item);
          const path = String(args.path || pathFromPatch(String(args.patch || args.diff || "")) || "");
          const preview = workspacePreviewPath(name, { ...args, url: args.url || item.payload?.path }) || path;
          if (name === "browser_open" || name === "browser_takeover") {
            if (preview) setReviewFile(preview);
            setInspector(true);
            setInspTab("browser");
          }
        }
        if (item.type === "tool_result") {
          const name = toolName(item);
          const args = toolArgs(item);
          const preview = workspacePreviewPath(name, { ...args, url: args.url || item.payload?.path });
          if ((name === "browser_open" || name === "browser_takeover") && preview) {
            setReviewFile(preview);
            setInspector(true);
            setInspTab("browser");
          }
        }
        if (item.type === "turn_end" || item.type === "error") {
          markEnded(activeId);
          api.contextUsage(activeId).then(setCtx).catch(() => {});
          api.queueList(activeId).then((q) => {
            setQueueItems(q);
            setQueued(q.length);
          }).catch(() => { setQueued(0); setQueueItems([]); });
          void refreshOutline();
          void refreshTrace();
        }
      },
      (seed, page, opts) => {
        if (opts.resync) {
          /** 实时尾上有洞时补一次。座位折叠里已经有流式正文，所以回看位置和审查面板不动。 */
          hotItemsRef.current = commitHot(foldRef.current, foldLiveIntoSeed(seed, hotItemsRef.current));
          foldRef.current.replace(hotItemsRef.current);
          if (!browsingRef.current) {
            itemsAcc.current = hotItemsRef.current;
            setItems(hotItemsRef.current.slice());
          }
          void refreshOutline();
          return;
        }
        browsingRef.current = false;
        hotItemsRef.current = commitHot(foldRef.current, foldLiveIntoSeed(seed, hotItemsRef.current));
        foldRef.current.replace(hotItemsRef.current);
        itemsAcc.current = hotItemsRef.current;
        setItems(hotItemsRef.current.slice());
        const html = latestWorkspaceHTMLPath(hotItemsRef.current);
        if (html) setReviewFile(html);
        if (page) {
          headSeqRef.current = page.headSeq || 0;
          olderRef.current = !!page.older;
          setOlder(!!page.older);
        }
        void refreshOutline();
      },
    );
    api.approvals().then(setApprovals).catch(() => {});
    api.contextUsage(activeId).then(setCtx).catch(() => {});
    api.running(activeId)
      .then((live) => setRunning((m) => ({ ...m, [activeId]: live && !settled(endedAt.current, activeId) })))
      .catch(() => {});
    return () => {
      unsub();
      if (itemsRaf.current) {
        window.clearTimeout(itemsRaf.current);
        itemsRaf.current = 0;
      }
    };
  }, [activeId, markEnded, refreshOutline, refreshTrace]);

  useEffect(() => {
    const q = query.trim();
    if (!q) {
      setSearchHits(null);
      return;
    }
    const t = window.setTimeout(() => {
      api.searchSessions(q, true).then(setSearchHits).catch(() => setSearchHits(null));
    }, 220);
    return () => window.clearTimeout(t);
  }, [query]);

  useEffect(() => {
    if (!anyRun) return;
    let alive = true;
    const t = window.setInterval(async () => {
      await syncRunning();
      if (!alive || !activeId) return;
      const [live, offers] = await Promise.all([
        api.running(activeId).catch(() => false),
        api.approvals().catch(() => [] as Approval[]),
      ]);
      if (!alive) return;
      setRunning((m) => ({ ...m, [activeId]: live && !settled(endedAt.current, activeId) }));
      setApprovals(offers);
      api.contextUsage(activeId).then(setCtx).catch(() => {});
      /** 进行中的回合可能丢了尾事件。正在跑时拉一次环，让对话在这一轮轮询里跟上，而不是干等通知。 */
      if (live) catchUpSession(activeId);
    }, 1500);
    return () => {
      alive = false;
      window.clearInterval(t);
    };
  }, [anyRun, activeId, syncRunning]);

  /** 按当前表面取通道。 */
  async function currentChannel(): Promise<ThreadChannel> {
    return channelForSurface(useUI.getState().surface);
  }

  /** 视频通道依次用配置里的视频工作区、健康信息里的、再退回普通工作区。ch 与 video 比较是否相等。 */
  function channelWorkspace(ch: ThreadChannel): string {
    if (ch === "video") return savedCfg.videoWorkspace || health.videoWorkspace || savedCfg.workspace;
    return savedCfg.workspace;
  }

  /** 记住这条通道的会话。已经是当前会话就返回。否则清空条目和队列，并保持当前窗格打开它。t 读取 id，传给 openThread。lane 传给 rememberLaneThread。 */
  function activateLaneThread(t: Thread, lane: VideoLane) {
    rememberLaneThread(lane, t.id);
    if (activeRef.current?.id === t.id) return;
    itemsAcc.current = [];
    setItems([]);
    setQueued(0);
    openThread(t, { keepPane: true });
  }

  /** 新建一条视频会话并套用待定权限。返回值没有 channel 时补成 video。 */
  async function mintVideoThread(): Promise<Thread> {
    const ch: ThreadChannel = "video";
    const created = await adoptPendingAuthMode(await api.createSession(channelWorkspace(ch), ch));
    const t = created.channel ? created : { ...created, channel: ch };
    setThreads((prev) => {
      const next = [t, ...prev.filter((x) => x.id !== t.id)];
      threadsRef.current = next;
      return next;
    });
    return t;
  }

  /** 空 id 返回 null。先在列表里找，找不到再看是不是当前会话。 */
  function findVideoThread(id: string): Thread | null {
    const sid = (id || "").trim();
    if (!sid) return null;
    return threadsRef.current.find((t) => t.id === sid) || (activeRef.current?.id === sid ? activeRef.current : null);
  }

  /** 空 id 返回假。已记下的剪辑会话直接为真。有源片哈希就记住并返回真。工程种类是 canvas、drama 或 studio 也返回真。 */
  async function threadOwnsCut(id: string): Promise<boolean> {
    const sid = (id || "").trim();
    if (!sid) return false;
    if (isCutThread(sid)) return true;
    const row = await api.video.cutSession(sid).catch(() => null);
    const cutId = str(row?.id);
    if (cutId && str(row?.source_hash)) {
      rememberCutThread(cutId, sid);
      return true;
    }
    const st = await api.video.sessionProject(sid).catch(() => null);
    const kind = str(st?.kind);
    return kind === "canvas" || kind === "drama" || kind === "studio";
  }

  /** 已经有一次创建在飞就等它。记住的会话不存在或已经属于剪辑时才新建。 */
  async function prepareWelcomeThread(): Promise<Thread> {
    const inflight = laneThreadJobs.current.create;
    if (inflight) return inflight;
    /** 记住的创作会话不能用时新建一条，并记到创作通道。 */
    const job = (async () => {
      const remembered = lastLaneThread("create");
      let t = findVideoThread(remembered);
      if (!t || await threadOwnsCut(t.id)) {
        t = await mintVideoThread();
        rememberLaneThread("create", t.id);
      }
      return t;
    })();
    laneThreadJobs.current.create = job;
    try {
      return await job;
    } finally {
      delete laneThreadJobs.current.create;
    }
  }

  /** 没有工程 id 就抛 cut project required。同一个工程已经在找会话时复用那一次。找到或新建后绑定。 */
  async function ensureCutThread(project: CutProject): Promise<Thread> {
    const projectId = str(project.id);
    if (!projectId) throw new Error("cut project required");
    const inflight = cutThreadJobs.current[projectId];
    if (inflight) return inflight;
    /** 先按剪辑工程上的会话找，没有再新建，然后绑定。 */
    const job = (async () => {
      let sid = cutThreadId(projectId) || str(project.session_id);
      if (!sid) {
        const fresh = await api.video.cutGet(projectId).catch(() => null) as CutProject | null;
        sid = str(fresh?.session_id);
      }
      let t = sid ? findVideoThread(sid) : null;
      if (sid && !t) {
        const fresh = await api.listSessions().catch(() => threadsRef.current);
        setThreads(fresh);
        threadsRef.current = fresh;
        t = fresh.find((x) => x.id === sid) || null;
      }
      if (!t) {
        t = await mintVideoThread();
        const title = str(project.title);
        if (title) {
          await api.renameSession(t.id, title).catch(() => {});
          t = { ...t, title };
          setThreads((prev) => prev.map((x) => (x.id === t!.id ? t! : x)));
        }
      }
      await api.video.cutBind(t.id, projectId);
      rememberCutThread(projectId, t.id);
      return t;
    })();
    cutThreadJobs.current[projectId] = job;
    try {
      return await job;
    } finally {
      delete cutThreadJobs.current[projectId];
    }
  }

  /** 记住剪辑和这条会话。标题不同才改名。再开一条欢迎会话给创作通道，并打开创作窗格。project 读取 id、title，传给 setProject。session 读取 id、title，传给 openThread，展开传递。 */
  async function claimCutThread(project: CutProject, session: Thread) {
    const id = str(project.id);
    if (id) rememberCutThread(id, session.id);
    const title = str(project.title);
    if (title && session.title !== title) {
      await api.renameSession(session.id, title).catch(() => {});
      session = { ...session, title };
      setThreads((prev) => prev.map((x) => (x.id === session.id ? { ...x, title } : x)));
    }
    const next = await mintVideoThread();
    rememberLaneThread("create", next.id);
    useCutDesk.getState().setProject(project, session.id);
    if (activeRef.current?.id !== session.id) {
      itemsAcc.current = [];
      setItems([]);
      setQueued(0);
      openThread(session, { keepPane: true });
    }
    useUI.getState().openVideoPane("create");
  }

  /** 创作通道走欢迎会话。其他通道已有一次在飞就等它。先用记住的、再看当前会话，都不合适才新建。lane 与 create 比较是否相等，用作下标，传给 activateLaneThread，传给 lastLaneThread，传给 laneAcceptsSession。 */
  async function ensureLaneThread(lane: VideoLane): Promise<Thread> {
    if (lane === "create") {
      const t = await prepareWelcomeThread();
      activateLaneThread(t, lane);
      return t;
    }
    const inflight = laneThreadJobs.current[lane];
    if (inflight) return inflight;
    /** 先用记住的会话，再看当前会话，这条通道不接受就新建。 */
    const job = (async () => {
      const projects = videoProjectsRef.current;
      const chats = threadsRef.current.filter((t) => !isSubagentThread(t) && threadChannel(t) === "video");
      const remembered = lastLaneThread(lane);
      /** 只在视频会话里挑，并且这条通道接受它。对不上返回 null。id 与 t.id、activeRef.current?.id 比较是否相等。 */
      const pick = (id: string) => {
        const hit = chats.find((t) => t.id === id) || (activeRef.current?.id === id ? activeRef.current : null);
        if (!hit || !laneAcceptsSession(lane, hit.id, projects)) return null;
        return hit;
      };
      let t = pick(remembered)
        || (activeRef.current && threadChannel(activeRef.current) === "video" && laneAcceptsSession(lane, activeRef.current.id, projects) ? activeRef.current : null);
      if (!t) t = await mintVideoThread();
      activateLaneThread(t, lane);
      return t;
    })();
    laneThreadJobs.current[lane] = job;
    try {
      return await job;
    } finally {
      delete laneThreadJobs.current[lane];
    }
  }

  prepareWelcomeRef.current = prepareWelcomeThread;
  /** 没有 id、桌子已收起或不是当前工程就返回。拉回来之后再核一次，避免中途换了工程。payload 读取 id。 */
  handlers.current.cut = (payload) => {
    const id = str(payload.id);
    const desk = useCutDesk.getState();
    if (!id || desk.parked || desk.project?.id !== id) return;
    void api.video.cutGet(id).then((row) => {
      const now = useCutDesk.getState();
      if (now.parked || now.project?.id !== id) return;
      now.setProject(row as CutProject, now.sessionId);
    }).catch(() => {});
  };

  /** studio、assets、skills、plugins、tasks、game 不切换会话。桌子上有源片且没收起时留在剪辑会话，失败就留在当前线程。否则回到创作通道。pane 与 studio、assets、skills、plugins、tasks、game 等比较是否相等。 */
  async function adoptVideoLane(pane: string) {
    if (pane === "studio" || pane === "assets" || pane === "skills" || pane === "plugins" || pane === "tasks" || pane === "game") return;
    if (pane === "drama") {
      await ensureLaneThread("drama");
      return;
    }
    if (pane === "canvas") {
      await ensureLaneThread("canvas");
      return;
    }
    const desk = useCutDesk.getState();
    if (desk.project?.id && desk.project.source_hash && !desk.parked) {
      try {
        const t = await ensureCutThread(desk.project);
        if (activeRef.current?.id !== t.id) {
          itemsAcc.current = [];
          setItems([]);
          setQueued(0);
          openThread(t, { keepPane: true });
        }
      } catch {
        /* 失败就留在当前线程 */
      }
      return;
    }
    await ensureLaneThread("create");
  }

  /** 空 id 返回。canvas、studio、drama 各自写回选择。查绑定失败就跳过。sessionId 为假时走另一支，传给 api.video.sessionProject。 */
  async function restoreVideoProject(sessionId: string) {
    if (!sessionId) return;
    try {
      const st = await api.video.sessionProject(sessionId);
      const kind = str(st?.kind);
      const title = str(st?.title || st?.drama_title);
      if (kind === "canvas") {
        const canvasId = str(st?.canvas_id || st?.id);
        if (canvasId) useCanvasHost.setState({ projectId: canvasId, title: title || copy.video.canvas });
        return;
      }
      if (kind === "studio") {
        const studioId = str(st?.studio_id || st?.id);
        if (studioId) {
          useStudioSelection.getState().setStudioId(studioId);
          if (title) useStudioSelection.getState().setStudioMeta({ title });
        }
        return;
      }
      if (kind === "drama") {
        const nextDrama = str(st?.drama_id || st?.id);
        const nextEpisode = str(st?.episode_id);
        if (nextDrama) {
          useDramaSelection.getState().setDramaId(nextDrama);
          if (nextEpisode) useDramaSelection.getState().setEpisodeId(nextEpisode);
        }
      }
    } catch {
      /* 查绑定失败就跳过 */
    }
  }

  /** 把这条工作室工程放到列表最前，并记下这次的会话 id。同 id 的旧工作室行去掉。p 读取 id，展开传递。sessionId 写入字段 sessionId。 */
  function rememberStudioProject(p: VideoProject, sessionId: string) {
    const row = asStudioProject({ ...p, sessionId });
    setVideoProjects((prev) => {
      const next = [row, ...prev.filter((x) => !(x.kind === "studio" && x.id === p.id))];
      videoProjectsRef.current = next;
      return next;
    });
  }

  /** 没有工程 id 抛 studio project required。同一个工程已经在绑就等它。没有待定权限且权限不是 full 时改成 full。p 读取 id、sessionId、title，传给 rememberStudioProject。 */
  async function ensureStudioThread(p: VideoProject): Promise<Thread> {
    const projectId = str(p.id);
    if (!projectId) {
      throw new Error("studio project required");
    }
    const inflight = studioThreadJobs.current[projectId];
    if (inflight) return inflight;
    /** 先找已经绑过的会话，没有再新建。没有待定权限且不是 full 时改成 full，然后绑定。 */
    const job = (async () => {
      const ch: ThreadChannel = "video";
      let boundId = studioSessionId(projectId, videoProjectsRef.current) || str(p.sessionId);
      if (!boundId) {
        const hist = await loadVideoHistory();
        boundId = studioSessionId(projectId, hist);
      }
      let t = boundId
        ? threadsRef.current.find((x) => x.id === boundId) || (activeRef.current?.id === boundId ? activeRef.current : null)
        : null;
      if (boundId && !t) {
        const fresh = await api.listSessions().catch(() => threadsRef.current);
        setThreads(fresh);
        threadsRef.current = fresh;
        t = fresh.find((x) => x.id === boundId) || null;
      }
      if (!t) {
        const pending = useUI.getState().pendingAuthMode;
        const created = await api.createSession(channelWorkspace(ch), ch);
        t = created.channel ? created : { ...created, channel: ch };
        if (pending) {
          t = await adoptPendingAuthMode(t);
        } else if (t.authMode !== "full") {
          t = await api.setSessionAuthMode(t.id, "full").catch(() => t!);
        }
        if (p.title) {
          await api.renameSession(t.id, p.title).catch(() => {});
          t = { ...t, title: p.title };
        }
        setThreads((prev) => {
          const next = [t!, ...prev.filter((x) => x.id !== t!.id)];
          threadsRef.current = next;
          return next;
        });
      }
      await api.video.studioBind(t.id, projectId);
      rememberStudioProject(p, t.id);
      useStudioSelection.getState().setStudioId(projectId);
      useStudioSelection.getState().setStudioMeta({ title: p.title || "" });
      useUI.getState().openStudioDesk();
      if (activeRef.current?.id !== t.id) {
        itemsAcc.current = [];
        setItems([]);
        setQueued(0);
        openThread(t, { keepPane: true });
      }
      void loadVideoHistory();
      return t;
    })();
    studioThreadJobs.current[projectId] = job;
    try {
      return await job;
    } finally {
      delete studioThreadJobs.current[projectId];
    }
  }

  /** studio 交给工作室会话。game 直接返回。没有会话就新建并绑定。然后清空条目并保持窗格打开它。p 读取 kind、sessionId、id、title、episodeId，传给 ensureStudioThread。 */
  async function openVideoProject(p: VideoProject) {
    if (p.kind === "studio") {
      await ensureStudioThread(p);
      return;
    }
    if (p.kind === "game") {
      return;
    }
    const ch: ThreadChannel = "video";
    const ui = useUI.getState();
    let t = p.sessionId ? threads.find((x) => x.id === p.sessionId) || null : null;
    if (!t) {
      const created = await adoptPendingAuthMode(await api.createSession(channelWorkspace(ch), ch));
      t = created.channel ? created : { ...created, channel: ch };
      setThreads((prev) => [t!, ...prev.filter((x) => x.id !== t!.id)]);
      if (p.kind === "canvas") {
        await api.video.canvasBind(t.id, p.id).catch(() => {});
        useCanvasHost.setState({ projectId: p.id, title: p.title || copy.video.canvas });
        ui.openVideoPane("canvas", { canvasFocus: "editor" });
      } else {
        if (p.episodeId) await api.video.bind(t.id, p.episodeId).catch(() => {});
        useDramaSelection.getState().setDramaId(p.id);
        if (p.episodeId) useDramaSelection.getState().setEpisodeId(p.episodeId);
        ui.openVideoPane("drama");
      }
      if (p.title) await api.renameSession(t.id, p.title).catch(() => {});
      t = { ...t, title: p.title || t.title };
      setThreads((prev) => prev.map((x) => (x.id === t!.id ? t! : x)));
    } else if (p.kind === "canvas") {
      useCanvasHost.setState({ projectId: p.id, title: p.title || copy.video.canvas });
      await api.video.canvasBind(t.id, p.id).catch(() => {});
      ui.openVideoPane("canvas", { canvasFocus: "editor" });
    } else {
      useDramaSelection.getState().setDramaId(p.id);
      if (p.episodeId) useDramaSelection.getState().setEpisodeId(p.episodeId);
      if (p.episodeId) await api.video.bind(t.id, p.episodeId).catch(() => {});
      ui.openVideoPane("drama");
    }
    itemsAcc.current = [];
    setItems([]);
    setQueued(0);
    openThread(t, { keepPane: true });
    void loadVideoHistory();
  }

  /** 学习子模块各自一段对话。已经对上就直接打开，否则新建学习频道的会话并绑上路径或材料。spec 读取 scope、title、pathId、libraryId、materialId。 */
  async function ensureLearnScope(spec: LearnScopeSpec): Promise<Thread> {
    const scope = spec.scope || "today";
    learnScopeWanted.current = scope;
    useUI.getState().setLearnScope({ ...spec, scope });
    const inflight = learnScopeJobs.current.get(scope);
    /** 同一范围已经在建时等它。范围已经换成别的就不再打开。 */
    const finish = async (job: Promise<Thread>) => {
      const ready = await job;
      if (learnScopeWanted.current === scope) activateLearnThread(ready);
      return ready;
    };
    if (inflight) return finish(inflight);
    const job = mintLearnScope(scope, spec);
    learnScopeJobs.current.set(scope, job);
    try {
      return await finish(job);
    } finally {
      if (learnScopeJobs.current.get(scope) === job) learnScopeJobs.current.delete(scope);
    }
  }

  /** 按范围找出或新建学习会话，并写上绑定。scope 用作本地钥匙。spec 读取 title、pathId、libraryId、materialId。 */
  async function mintLearnScope(scope: string, spec: LearnScopeSpec): Promise<Thread> {
    const remembered = readLearnScopes()[scope] || "";
    const learnThreads = threadsRef.current.filter((t) => !isSubagentThread(t) && threadChannel(t) === "learn");
    let t = matchLearnThread(learnThreads, spec, remembered);
    if (!t) {
      const created = await adoptPendingAuthMode(await api.createSession(channelWorkspace("learn"), "learn"));
      t = created.channel === "learn" ? created : { ...created, channel: "learn" };
      setThreads((prev) => {
        const next = [t!, ...prev.filter((x) => x.id !== t!.id)];
        threadsRef.current = next;
        return next;
      });
    }
    if (spec.title && (!t.title || t.title === "learn" || t.title === "session" || t.title === "thread")) {
      await api.renameSession(t.id, spec.title).catch(() => {});
      t = { ...t, title: spec.title };
    }
    if ((spec.pathId || spec.libraryId || spec.materialId) && !learnBindingMatches(t, spec)) {
      const raw = await api.learnCall("learn.bind", {
        session: t.id,
        library_id: spec.libraryId || "",
        path_id: spec.pathId || "",
        material_id: spec.materialId || "",
      });
      const bound = api.threadOf(raw);
      t = bound.channel === "learn" ? bound : { ...bound, channel: "learn" };
    }
    setThreads((prev) => {
      const next = prev.some((x) => x.id === t!.id) ? prev.map((x) => (x.id === t!.id ? t! : x)) : [t!, ...prev];
      threadsRef.current = next;
      return next;
    });
    writeLearnScope(scope, t.id);
    return t;
  }

  /** 切到这段学习对话，并清掉上一段的正文。已经是当前会话就不动。先钉住，避免随后的表面切换把它盖掉。t 读取 id，传给 openThread。 */
  function activateLearnThread(t: Thread) {
    learnHold.current = t.id;
    if (useUI.getState().surface !== "pages") return;
    if (activeRef.current?.id === t.id) return;
    itemsAcc.current = [];
    setItems([]);
    setQueued(0);
    openThread(t, { keepPane: true });
  }

  /** 当前会话已经是这个通道就直接用。否则新建并套用待定权限。 */
  async function ensureThread(): Promise<Thread> {
    const ch = await currentChannel();
    if (active && threadChannel(active) === ch) return active;
    const created = await adoptPendingAuthMode(await api.createSession(channelWorkspace(ch), ch));
    const t = created.channel ? created : { ...created, channel: ch };
    setThreads((prev) => [t, ...prev.filter((x) => x.id !== t.id)]);
    setActive(t);
    useUI.getState().rememberThread(t);
    return t;
  }

  /**
   * 正文和附件都空就返回。工作区没就绪时提示并打开常规设置。工作室桌子没有短片 id 就返回。正在跑且是改道时只调用 steer。创作窗格里 yoyo-cut 附件只绑定、不随消息发出。错误文字含 queued 或 already running 时只把队列加一。
   * opts 读取 text、attachments、steer，为空时不读。
   */
  async function onSend(opts?: { steer?: boolean; attachments?: Attachment[]; text?: string }) {
    const text = (opts?.text !== undefined ? opts.text : (useUI.getState().drafts[draftKey] || "")).trim();
    if (!text && !(opts?.attachments && opts.attachments.length)) return;
    const turnWs = active?.workspace || channelWorkspace(channelForSurface(useUI.getState().surface));
    const ch = channelForSurface(useUI.getState().surface);
    const ready = ch === "video"
      ? pathReady(turnWs) || workspaceReady(health.videoWorkspaceReady, savedCfg.videoWorkspace || "")
      : pathReady(turnWs) || workspaceReady(health.workspaceReady, savedCfg.workspace);
    if (!ready) {
      toast.message(copy.app.setupFirst);
      openSettings("general", "general-workspace");
      return;
    }
    setErr("");
    try {
      const ui = useUI.getState();
      let t: Thread;
      if (ui.studioDesk) {
        const studioId = useStudioSelection.getState().studioId;
        if (!studioId) {
          toast.message(copy.video.studio.needShortFirst);
          return;
        }
        t = await ensureStudioThread(asStudioProject({
          id: studioId,
          title: useStudioSelection.getState().title,
          sessionId: studioSessionId(studioId, videoProjectsRef.current),
        }));
      } else if (ui.surface === "video" && ui.videoPane === "create") {
        const desk = useCutDesk.getState();
        if (desk.project?.id && desk.project.source_hash && !desk.parked) {
          t = await ensureCutThread(desk.project);
        } else {
          t = await ensureLaneThread("create");
        }
      } else if (ui.surface === "video" && ui.videoPane === "drama") {
        t = await ensureLaneThread("drama");
        await bindVideoThread(t.id).catch(() => {});
      } else if (ui.surface === "video" && ui.videoPane === "canvas") {
        t = await ensureLaneThread("canvas");
        await ensureCanvasProject(t.id, copy.video.canvas).catch(() => {});
        void loadVideoHistory();
      } else if (ui.surface === "pages") {
        t = await ensureLearnScope(ui.learnScope?.scope ? ui.learnScope : { scope: "today" });
      } else {
        t = await ensureThread();
        if (ui.surface === "video" && ui.videoMode === "drama") {
          await bindVideoThread(t.id).catch(() => {});
        }
        if (ui.surface === "video" && ui.videoMode === "canvas") {
          await ensureCanvasProject(t.id, copy.video.canvas).catch(() => {});
          void loadVideoHistory();
        }
      }
      if (opts?.steer && running[t.id]) {
        await api.steer(t.id, text);
        useUI.getState().patchDrafts({ [t.id]: "", _new: "" });
        if (t.id === activeId) {
          hotItemsRef.current = commitHot(foldRef.current, foldRef.current.accept({
            ...localUser(t.id, text, opts?.attachments),
            source: "steer",
            key: `ui-steer:${t.id}:${Date.now()}`,
          }));
          if (!browsingRef.current) itemsAcc.current = hotItemsRef.current;
          setItems(browsingRef.current ? withLiveTail(itemsAcc.current, hotItemsRef.current) : hotItemsRef.current.slice());
        }
        toast.success(copy.app.steered);
        return;
      }
      const wasRunning = !!running[t.id];
      useUI.getState().patchDrafts({ [t.id]: "", _new: "" });
      markStarted(t.id);
      const requestId = newRequestId();
      if (t.id === activeId) {
        browsingRef.current = false;
        hotItemsRef.current = commitHot(foldRef.current, foldRef.current.accept(localUser(t.id, text, opts?.attachments, requestId)));
        itemsAcc.current = hotItemsRef.current;
        setItems(hotItemsRef.current.slice());
      }
      const rawAtts = opts?.attachments || [];
      const sendAtts: Attachment[] = [];
      if (useUI.getState().videoPane === "create") {
        for (const a of rawAtts) {
          const m = /^yoyo-cut:([^:]+):/.exec(String(a.path || ""));
          if (m) {
            await api.video.cutBind(t.id, m[1]).catch(() => {});
            continue;
          }
          sendAtts.push(a);
        }
      } else {
        sendAtts.push(...rawAtts);
      }
      const res = await api.send(t.id, text, { plan: useUI.getState().plan, attachments: sendAtts, requestId });
      if (wasRunning || res.queued) {
        const q = await api.queueList(t.id).catch(() => []);
        setQueueItems(q);
        setQueued(q.length || 1);
        toast.message(copy.app.queued);
      } else {
        setQueued(0);
        setQueueItems([]);
      }
      void refreshOutline();
    } catch (e) {
      const m = api.errMessage(e);
      if (m.includes("queued") || m.includes("already running")) {
        setQueued((n) => n + 1);
        toast.message(copy.app.queued);
        return;
      }
      if (activeId) markEnded(activeId);
      fail(e);
    }
  }

  /** 按当前表面找对应通道。先用记住的，再找中断或有队列的，再找同通道任意一条。已经在跑就只打开，不再重试。 */
  async function onContinueLast() {
    const ch = channelForSurface(surface);
    const last = useUI.getState().lastThreadId(ch);
    const hit = threads.find((t) => t.id === last && threadChannel(t) === ch)
      || threads.find((t) => threadChannel(t) === ch && (t.interrupted || (t.queued || 0) > 0))
      || threads.find((t) => threadChannel(t) === ch);
    if (!hit) return;
    openThread(hit);
    if (running[hit.id]) return;
    if (hit.id === activeId) {
      hotItemsRef.current = dropTurnErrors(hotItemsRef.current);
      foldRef.current.replace(hotItemsRef.current);
      itemsAcc.current = browsingRef.current ? dropTurnErrors(itemsAcc.current) : hotItemsRef.current;
      setItems(browsingRef.current ? withLiveTail(itemsAcc.current, hotItemsRef.current) : hotItemsRef.current.slice());
      markStarted(hit.id);
    }
    try {
      await api.retry(hit.id);
    } catch (e) {
      if (hit.id === activeId) markEnded(hit.id);
      fail(e);
    }
  }

  /** 没有当前会话或它正在跑就返回。没有非改道的用户消息、也没有助手消息就返回。 */
  async function onRetryLast() {
    if (!activeId || running[activeId]) return;
    const hasTurn = itemsAcc.current.some((it) => (it.type === "user" && it.source !== "steer") || it.type === "assistant");
    if (!hasTurn) return;
    setErr("");
    markStarted(activeId);
    hotItemsRef.current = dropTurnErrors(hotItemsRef.current);
    foldRef.current.replace(hotItemsRef.current);
    itemsAcc.current = browsingRef.current ? dropTurnErrors(itemsAcc.current) : hotItemsRef.current;
    setItems(browsingRef.current ? withLiveTail(itemsAcc.current, hotItemsRef.current) : hotItemsRef.current.slice());
    try {
      await api.retry(activeId);
    } catch (e) {
      markEnded(activeId);
      fail(e);
    }
  }

  /**
   * 去掉开头的斜杠后分派。schedule 用 heartbeat、30m、isolate。remember 以 profile 或 pin 开头时 kind 写成 profile。rewind 不带 files、both、conversation、chat、from 时拼成 both from。artifact 只把固定一句追加进草稿。
   * cmd 做替换。
   * rest 去掉首尾空白，为假时改用后备值，为假时用 project，做替换，传给 api.compactSession，传给 api.setSessionModel。
   */
  async function onSlash(cmd: string, rest: string) {
    const raw = cmd.replace(/^\//, "");
    if (raw === "plan") {
      useUI.getState().setPlan((v) => !v);
      return;
    }
    if (raw === "new") {
      await onNew();
      return;
    }
    if (raw === "quit") {
      requestQuit();
      return;
    }
    if (raw === "improve") {
      useUI.getState().openHarness("propose");
      return;
    }
    if (raw === "diff") {
      await refreshDiff();
      return;
    }
    if (raw === "stop") {
      await onStop();
      return;
    }
    if (raw === "steer") {
      const text = rest.trim() || (useUI.getState().drafts[draftKey] || "").trim();
      if (!text || !activeId) return;
      await api.steer(activeId, text);
      useUI.getState().patchDrafts({ [activeId]: "", _new: "" });
      toast.success(copy.app.steered);
      return;
    }
    if (raw === "schedule") {
      const prompt = rest.trim();
      if (!prompt) return;
      await api.scheduleCreate({ kind: "heartbeat", spec: "30m", prompt, isolate: true });
      toast.success(copy.settings.addJob);
      return;
    }
    if (raw === "remember") {
      let text = rest.trim();
      if (!text) return;
      let kind = "episodic";
      const low = text.toLowerCase();
      if (low.startsWith("profile ")) {
        kind = "profile";
        text = text.slice(8).trim();
      } else if (low.startsWith("pin ")) {
        kind = "profile";
        text = text.slice(4).trim();
      }
      if (!text) return;
      await api.memoryWrite(kind, text);
      toast.success(copy.settings.save);
      return;
    }
    if (raw === "forget") {
      const id = rest.trim();
      if (!id) return;
      await api.memoryForget(id);
      toast.success(copy.settings.forget);
      return;
    }
    if (raw === "project") {
      const name = rest.trim() || "project";
      await api.projectCreate({ name, root: savedCfg.workspace });
      toast.success(name);
      return;
    }
    if (!activeId && raw !== "new") return;
    if (raw === "rename") {
      const title = rest.trim();
      if (!activeId) return;
      if (!title) {
        useUI.getState().requestRename();
        return;
      }
      await api.renameSession(activeId, title);
      await refresh();
      return;
    }
    if (raw === "fork") {
      if (!activeId) return;
      const from = rest.replace(/^from\s+/i, "").trim();
      const t = await api.forkSession(activeId, from);
      setThreads((prev) => [t, ...prev]);
      setActive(t);
      toast.success(copy.app.forked);
      return;
    }
    if (raw === "archive") {
      if (!activeId) return;
      await api.archiveSession(activeId, true);
      await refresh();
      toast.success(copy.app.archived);
      return;
    }
    if (raw === "delete") {
      if (!active) return;
      setPendingDelete(active);
      return;
    }
    if (raw === "compact") {
      if (!activeId) return;
      const note = await api.compactSession(activeId, rest.trim());
      toast.success(note || copy.app.compacted);
      refreshCtx();
      return;
    }
    if (raw === "rewind") {
      if (!activeId) return;
      const r = rest.trim();
      let focus = "both";
      if (r) {
        const low = r.toLowerCase();
        if (low.startsWith("files") || low.startsWith("both") || low.startsWith("conversation") || low.startsWith("chat") || low.startsWith("from ")) {
          focus = r;
        } else {
          focus = "both from " + r;
        }
      }
      const note = await api.compactSession(activeId, focus);
      toast.success(note || copy.app.compacted);
      refreshCtx();
      return;
    }
    if (raw === "model") {
      if (!rest.trim()) return;
      const t = await api.setSessionModel(activeId, rest.trim());
      setActive(t);
      await refresh();
      refreshCtx();
      return;
    }
    if (raw === "export") {
      const md = await api.exportSession(activeId);
      const dump = await api.dumpSession(activeId).catch(() => ({ path: "" }));
      const blob = dump.path ? `${md}\n\nDump: ${dump.path}` : md;
      await navigator.clipboard.writeText(blob);
      toast.success(dump.path ? copy.app.dumped : copy.app.exported);
      return;
    }
    if (raw === "context") {
      if (!activeId) return;
      const u = await api.contextUsage(activeId);
      const layers = (u.layers || []).filter(Boolean).join(" · ");
      const line = `${u.tokens || 0}/${u.budget || 0}${u.window ? ` · window ${u.window}` : ""}${layers ? ` · ${layers}` : ""}${u.note ? ` · ${u.note}` : ""}`;
      toast.message(line.trim() || copy.review.context);
      refreshCtx();
      return;
    }
    if (raw === "artifact") {
      const cur = useUI.getState().drafts[draftKey] || "";
      const next = (cur ? cur + "\n" : "") + "Deliver a real .docx, formula .xlsx, and .pptx. Preview with office_render. Do not send mail.";
      useUI.getState().setDraft(draftKey, next);
    }
  }

  /** 还有会话在跑就先弹出确认。否则直接退出。 */
  function requestQuit() {
    if (anyRun) {
      setPendingQuit(true);
      return;
    }
    void api.quit();
  }

  /** 没有当前会话就返回。只发中断，不在这里把回合标成结束。 */
  async function onStop() {
    if (!activeId) return;
    /** 这里不把回合标成结束。停止后工具或模型流还可能再吐输出。回合等 turn_end、error、stopped，或等 runningIDs 轮询再关。 */
    void api.interrupt(activeId).catch(() => {});
  }

  /** 没有会话、正在加载或没有更早内容就返回。用户回合已经达到回看上限就关掉更早。按 key 去掉已经有的条目。 */
  const loadOlder = useCallback(async () => {
    if (!activeId || loadingOlder || !olderRef.current) return;
    if (userTurnCount(itemsAcc.current) >= BROWSE_TRANSCRIPT_TURNS) {
      olderRef.current = false;
      setOlder(false);
      return;
    }
    setLoadingOlder(true);
    try {
      const page = await api.trajectoryPage(activeId, { before: headSeqRef.current, turns: HOT_TRANSCRIPT_TURNS });
      const olderItems = replayEvents(page.events);
      const have = new Set(itemsAcc.current.map((x) => x.key));
      const add = olderItems.filter((x) => !have.has(x.key));
      browsingRef.current = true;
      itemsAcc.current = lastUserTurns([...add, ...itemsAcc.current], BROWSE_TRANSCRIPT_TURNS);
      headSeqRef.current = page.headSeq;
      const atCap = userTurnCount(itemsAcc.current) >= BROWSE_TRANSCRIPT_TURNS;
      olderRef.current = !!page.older && !atCap;
      setOlder(olderRef.current);
      setItems(withLiveTail(itemsAcc.current, hotItemsRef.current));
    } finally {
      setLoadingOlder(false);
    }
  }, [activeId, loadingOlder]);

  const presenceItems = useMemo(() => lastUserTurns(items, 2), [items]);

  /** 提交决定后从本地列表去掉这一条。id 与 a.id 比较是否不等，传给 api.resolveApproval。decision 传给 api.resolveApproval。answer 传给 api.resolveApproval。 */
  async function onResolve(id: string, decision: string, answer?: string) {
    await api.resolveApproval(id, decision, answer);
    setApprovals((prev) => prev.filter((a) => a.id !== id));
  }

  /** 拉取工作区差异并清空已选块。失败走 fail。 */
  async function refreshDiff() {
    try {
      const h = await api.workspaceHunks(active?.workspace || savedCfg.workspace || "");
      setDiff(h.diff);
      setHunks(h.hunks);
      setHunkSel({});
    } catch (e) {
      fail(e);
    }
  }

  /** 没有上一次应用就返回。按当时的工作区、块和快照撤销。 */
  async function undoLastApply() {
    const last = lastApply.current;
    if (!last) return;
    try {
      await api.reverseHunks(last.workspace, last.ids, last.snapshot);
      lastApply.current = null;
      await refreshDiff();
      toast.success(copy.app.undone);
    } catch (e) {
      fail(e);
    }
  }

  /** 只提交选中的块。成功后把这次记下来，提示里可以撤销。 */
  async function applySelected() {
    const ids = Object.entries(hunkSel).filter(([, v]) => v).map(([k]) => k);
    const workspace = active?.workspace || savedCfg.workspace || "";
    const snapshot = diff;
    try {
      await api.applyHunks(workspace, ids);
      lastApply.current = { workspace, ids, snapshot };
      await refreshDiff();
      toast.success(copy.app.applied, {
        action: { label: copy.app.undo, onClick: () => { void undoLastApply(); } },
      });
    } catch (e) {
      fail(e);
    }
  }

  /** 在给出的工作区新建会话，没给就用当前通道的工作区。清空条目和队列后打开它。workspace 为假时改用后备值，传给 api.createSession。 */
  async function onNewIn(workspace?: string) {
    const ch = channelForSurface(useUI.getState().surface);
    const created = await adoptPendingAuthMode(await api.createSession(workspace || channelWorkspace(ch), ch));
    const t = created.channel ? created : { ...created, channel: ch };
    setThreads((list) => [t, ...list.filter((x) => x.id !== t.id)]);
    itemsAcc.current = [];
    setItems([]);
    setQueued(0);
    openThread(t);
  }

  /** 视频表面只打开创作窗格。其他表面先回到会话模块，再新建一条会话对话。 */
  async function onNew() {
    const surfaceNow = useUI.getState().surface;
    if (surfaceNow === "video") {
      useUI.getState().openVideoPane("create");
      return;
    }
    if (surfaceNow !== "agent") useUI.getState().showConversation();
    await onNewIn();
  }

  /** 记住这条会话并清掉待定权限。视频会话在 keepPane 时不改窗格。否则按通道打开工作室、短剧、画布或创作。其他通道回到对话。t 读取 id，传给 ui.rememberThread，传给 threadChannel，传给 setActive。opts 读取 keepPane，为空时不读。 */
  function openThread(t: Thread, opts?: { keepPane?: boolean }) {
    const ui = useUI.getState();
    ui.rememberThread(t);
    /** 打开一段对话就读掉它的通知。待定的权限属于还没创建的新对话，不属于这一段。 */
    ui.dismissNoticesFor(t.id);
    ui.setPendingAuthMode(null);
    if (threadChannel(t) === "video") {
      if (opts?.keepPane) {
        if (ui.surface !== "video") useUI.setState({ surface: "video" });
      } else {
        const lane = sessionLane(t.id, videoProjectsRef.current);
        if (lane === "studio") ui.openVideoPane("studio");
        else if (lane === "drama") ui.openVideoPane("drama");
        else if (lane === "canvas") ui.openVideoPane("canvas", { canvasFocus: "editor" });
        else ui.openVideoPane("create");
      }
    } else if (threadChannel(t) === "learn") {
      learnHold.current = t.id;
      if (ui.surface !== "pages") useUI.setState({ surface: "pages" });
    } else {
      ui.showConversation();
    }
    setActive(t);
  }

  /** 没有哈希也没有资源 id 时，有地址就先入库。还是没有就提示需要视频。当前会话不是视频或已经属于剪辑时先准备欢迎会话。没有工程 id 抛 cut project was not created。detail 读取 hash、url、title，传给 resourceIdFromSend。 */
  sendToCutRef.current = async (detail) => {
    const c = copy.video.conductor;
    let hash = String(detail.hash || "").trim();
    const resourceId = resourceIdFromSend(detail);
    if (!hash && !resourceId && detail.url) {
      toast.message(c.sendingToCut);
      hash = await ingestClipUrl(String(detail.url));
    }
    if (!hash && !resourceId) {
      toast.message(c.sendNeedVideo);
      return;
    }
    leaveGen.current += 1;
    let t = activeRef.current;
    if (!t || threadChannel(t) !== "video" || await threadOwnsCut(t.id)) {
      t = await prepareWelcomeThread();
    }
    toast.message(c.sendingToCut);
    let p: { id?: string; source_hash?: string };
    try {
      p = await api.video.cutOpen({
        title: String(detail.title || "").trim() || c.cutTitle,
        hash,
        resource_id: resourceId,
        session_id: t.id,
      }) as { id?: string; source_hash?: string };
    } catch (err) {
      if (!hash && detail.url) {
        hash = await ingestClipUrl(String(detail.url));
        p = await api.video.cutOpen({
          title: String(detail.title || "").trim() || c.cutTitle,
          hash,
          session_id: t.id,
        }) as { id?: string; source_hash?: string };
      } else {
        throw err;
      }
    }
    const id = String(p?.id || "");
    const src = String(p?.source_hash || hash);
    if (!id) throw new Error("cut project was not created");
    await claimCutThread(p as CutProject, t);
    useUI.getState().setDraft(t.id, cutPrompt(copy, id, src));
    toast.success(c.sentToCut);
  };

  /** 空哈希返回。种类固定成 suite，并打开证明页。hash 为假时用空字符串，转成字符串，去掉首尾空白后，为假时走另一支，写入字段 snapshot。 */
  async function runHarborAt(hash: string) {
    const snapshot = String(hash || "").trim();
    if (!snapshot) return;
    setHarborKind("suite");
    setHarborErr("");
    setLabBusy(copy.labs.busyHarbor);
    openHarness("prove");
    try {
      setEvalReport(await api.runEval({ snapshot }));
      setBestReport(null);
    } catch (e) {
      setHarborErr(api.errMessage(e));
      fail(e);
    } finally {
      setLabBusy(null);
    }
  }

  /** 按种类调用对应评测。bon 取 3 次里的 best 或 Best。models 按逗号拆开并去掉空白。kind 与 suite、safety、sealed、transfer、index、behavior 等比较是否相等，传给 setHarborKind。 */
  async function runHarbor(kind: HarborKind) {
    setHarborKind(kind);
    setHarborErr("");
    setLabBusy(copy.labs.busyHarbor);
    openHarness("prove");
    try {
      if (kind === "suite") { setEvalReport(await api.runEval()); setBestReport(null); }
      if (kind === "safety") { setEvalReport(await api.runEvalSafety()); setBestReport(null); }
      if (kind === "sealed") { setEvalReport(await api.runEvalSealed()); setBestReport(null); }
      if (kind === "transfer") { setEvalReport(await api.runEvalTransfer()); setBestReport(null); }
      if (kind === "index") { setEvalReport(await api.runEvalIndex()); setBestReport(null); }
      if (kind === "behavior") { setEvalReport(await api.runEvalBehavior()); setBestReport(null); }
      if (kind === "tb") { setEvalReport(await api.runEvalTB()); setBestReport(null); }
      if (kind === "bon") {
        const r = await api.bestOfN(3);
        setBestReport(r);
        setEvalReport(r.best || r.Best);
      }
      if (kind === "models") {
        const r = await api.bestOfModels(bonModels.split(",").map((s) => s.trim()).filter(Boolean));
        setBestReport(r);
        setEvalReport(r.best || r.Best);
      }
    } catch (e) {
      setHarborErr(api.errMessage(e));
      fail(e);
    } finally {
      setLabBusy(null);
    }
  }

  /** 打开提议页。baselines 打开时传 3，否则传 0。结束后刷新归档和配置。 */
  async function runEvolve() {
    setLabBusy(copy.labs.busyEvolve);
    openHarness("propose");
    try {
      setEvolve(await api.evolve(evolveK, {
        rounds: evolveRounds,
        sealed: evolveSealed,
        behavior: evolveBehavior,
        index: evolveIndex,
        baselines: evolveBaselines ? 3 : 0,
        maxUsd: evolveMaxUsd,
      }));
      setTree(await api.archive());
      await refresh();
    } catch (e) {
      fail(e);
    } finally {
      setLabBusy(null);
    }
  }

  /** 没有当前会话或当前是视频通道就提示并返回。采纳时哈希只显示前 8 位，没有哈希用破折号。 */
  async function runOverlay() {
    if (!activeId) {
      toast.error(copy.labs.overlayNeedSession);
      return;
    }
    if (threadChannel(active) === "video") {
      toast.error(copy.labs.overlayVideo);
      return;
    }
    setLabBusy(copy.labs.busyOverlay);
    openHarness("propose");
    try {
      const res = await api.evolveOverlay({
        session: activeId,
        reuseTurn: overlayReuseTurn,
        maxUsd: evolveMaxUsd,
        overlayK,
        overlayMode,
      });
      setOverlay(res);
      try { setOverlayLibrary(await api.overlayLibrary(activeId)); } catch { /* 库是空的也行 */ }
      const adopted = !!(res && (res.adopted || res.Adopted));
      const hash = String(res?.hash || res?.Hash || "");
      toast.success(adopted
        ? copy.labs.overlayPin.replace("{hash}", hash ? hash.slice(0, 8) : "—")
        : copy.labs.overlayKeep);
      await refresh();
    } catch (e) {
      fail(e);
    } finally {
      setLabBusy(null);
    }
  }

  /** 左边没传时用 diffA，再没有用健康信息里的 harness。右边没传时用 diffB。传入的值会写回输入框。a 为假时用 diffA，为假时改用后备值，传给 setDiffA。b 为假时用 diffB，传给 setDiffB。 */
  async function compareHarness(a?: string, b?: string) {
    try {
      const left = a || diffA || health.harness;
      const right = b || diffB;
      if (a) setDiffA(a);
      if (b) setDiffB(b);
      setDiffOut(await api.diff(left, right));
    } catch (e) {
      fail(e);
    }
  }

  /** l3 缺省按假。成功后刷新并提示已检出。hash 传给 api.checkout。 */
  async function checkoutHarness(hash: string, l3?: boolean) {
    await api.checkout(hash, !!l3);
    await refresh();
    toast.success(copy.app.checkedOut);
  }

  /** 回滚后刷新并提示已回滚。 */
  async function rollbackHarness() {
    await api.rollback();
    await refresh();
    toast.success(copy.app.rolledBack);
  }

  /** 没传哈希就传空字符串。失败时先提示，再把错误抛回去。hash 为假时用空字符串，传给 api.revealHarness。 */
  async function revealHarness(hash?: string) {
    try {
      await api.revealHarness(hash || "");
    } catch (e) {
      fail(e);
      throw e;
    }
  }

  /** 先把合并后的配置写进本地，再提交整份配置。partial 展开传递。 */
  async function patchConfig(partial: Partial<AppConfig>) {
    const next = { ...savedCfg, ...partial };
    setSavedCfg(next);
    await api.setConfig(next);
  }

  const leaveGen = useRef(0);
  /** 先收起剪辑桌并取消创作参与。中途又触发了一次就不再切换。欢迎会话回来后如果桌子又绑上了源片，就留在原地。 */
  const leaveCut = useCallback(async () => {
    const gen = ++leaveGen.current;
    useCutDesk.getState().park();
    useCreateMode.getState().setEngaged(false);
    useUI.getState().openVideoPane("create");
    let welcome: Thread;
    try {
      welcome = await prepareWelcomeRef.current();
    } catch (e) {
      if (gen === leaveGen.current) toast.error(api.errMessage(e));
      return;
    }
    if (gen !== leaveGen.current) return;
    const now = useCutDesk.getState();
    if (!now.parked && now.project?.source_hash) return;
    activateLaneThread(welcome, "create");
  }, []);

  useEffect(() => {
    const km = mergeKeymap(savedCfg.keymap);
    /** 输入框里不响应新建、回合和审批快捷键。命令面板键始终切换。Escape 先让菜单、列表和模态对话框用掉。视频表面按当前窗格退一层。e 读取 target、metaKey、ctrlKey、shiftKey、altKey、key 等，调用 preventDefault，传给 matchKey。 */
    const onKey = (e: KeyboardEvent) => {
      const typing = (e.target as HTMLElement | null)?.closest("input, textarea, [contenteditable]");
      if (matchKey(e, km.palette)) {
        e.preventDefault();
        setPalette((v) => !v);
        return;
      }
      if (!typing && matchKey(e, km.newChat)) {
        e.preventDefault();
        void onNew();
      }
      if (matchKey(e, km.control)) {
        e.preventDefault();
        openSettings();
      }
      if (matchKey(e, km.toggleReview)) {
        e.preventDefault();
        const ui = useUI.getState();
        if (ui.surface === "video") return;
        if (ui.surface === "harness") setChatDock((v) => !v);
        else setInspector((v) => !v);
      }
      if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && !typing && useUI.getState().surface === "harness") {
        const idx = ["1", "2", "3", "4"].indexOf(e.key);
        if (idx >= 0) {
          e.preventDefault();
          setHarnessTab(HARNESS_TABS[idx]);
        }
      }
      if (e.key === "Tab" && e.shiftKey && !typing) {
        e.preventDefault();
        useUI.getState().setPlan((v) => !v);
      }
      if (e.key === "Escape") {
        if (e.defaultPrevented) return;
        if (useUI.getState().palette) {
          setPalette(false);
          return;
        }
        /** 菜单和确认层先用掉这一下 Escape。同一下就离开模块会让「更多检查」像个陷阱。 */
        if (document.querySelector('[role="menu"], [role="listbox"], [role="dialog"][aria-modal="true"]')) {
          return;
        }
        const ui = useUI.getState();
        if (ui.surface === "video") {
          if (ui.videoBoard) {
            ui.setVideoBoard(false);
            return;
          }
          if (ui.videoPane === "studio") {
            const sel = useStudioSelection.getState();
            if (sel.studioId) sel.setStudioId("");
            else ui.openVideoPane("create");
            return;
          }
          if (ui.videoPane === "canvas" && ui.canvasFocus === "editor") {
            ui.openVideoPane("canvas", { canvasFocus: "library" });
            return;
          }
          if (!typing && ui.videoPane === "create" && useCutDesk.getState().project?.source_hash) {
            void leaveCut();
            return;
          }
          if (ui.videoPane !== "create" && ui.videoPane !== "chat") {
            ui.openVideoPane("create");
            return;
          }
          return;
        }
        if (useUI.getState().surface === "settings" || useUI.getState().surface === "skills" || useUI.getState().surface === "harness" || useUI.getState().surface === "pages" || useUI.getState().surface === "idol") {
          showConversation();
          return;
        }
        const firstAsk = approvals[0];
        if (firstAsk && !typing && matchKey(e, km.deny)) {
          e.preventDefault();
          void onResolve(firstAsk.id, "deny");
        }
        return;
      }
      if (!typing && matchKey(e, km.prevTurn)) {
        e.preventDefault();
        const next = neighborTurn(outlineRef.current, activeTurnRef.current, -1);
        if (next) void jumpToTurn(next);
        return;
      }
      if (!typing && matchKey(e, km.nextTurn)) {
        e.preventDefault();
        const next = neighborTurn(outlineRef.current, activeTurnRef.current, 1);
        if (next) void jumpToTurn(next);
        return;
      }
      const first = approvals[0];
      if (first && !typing && !(e.ctrlKey || e.metaKey)) {
        if (matchKey(e, km.once)) {
          e.preventDefault();
          void onResolve(first.id, "once");
        }
        if (matchKey(e, km.session)) {
          e.preventDefault();
          void onResolve(first.id, "session");
        }
        if (matchKey(e, km.always)) {
          e.preventDefault();
          void onResolve(first.id, "always");
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [savedCfg.keymap, approvals, openSettings, showConversation, setInspector, setPalette, setLab, jumpToTurn, leaveCut]);

  /** 有文件但没有本地路径就提示并返回空字符串。都没有时打开选择框，选了却没有视频也返回空字符串。路径不是视频同样返回空字符串。source 读取 path、file。 */
  async function pickVideoFile(source: { path?: string; file?: File } = {}) {
    let path = String(source.path || "").trim();
    const file = source.file;
    if (!path && file) path = nativeFilePath(file);
    if (!path && file) {
      toast.error(copy.video.conductor.needFile);
      return "";
    }
    if (!path && !file) {
      const picked = await api.pickFiles();
      path = picked.find((item) => isVideoPath(item)) || "";
      if (!path) {
        if (picked.length) toast.error(copy.composer.importClipHint);
        return "";
      }
    }
    if (path && !isVideoPath(path)) {
      toast.error(copy.composer.importClipHint);
      return "";
    }
    return path;
  }

  /** 没有路径就返回。工程没有 id 或源片哈希时抛需要哈希。错误文字正好是 clip too large 时用过大文案。source 传给 pickVideoFile。 */
  async function ingestVideo(source: { path?: string; file?: File } = {}) {
    leaveGen.current += 1;
    const path = await pickVideoFile(source);
    if (!path) return;
    const toastId = toast.loading(copy.video.conductor.importing);
    try {
      const t = await prepareWelcomeThread();
      const project = await ingestVideoPath(path, t.id);
      if (!project?.id || !project.source_hash) throw new Error(copy.video.conductor.needHash);
      await claimCutThread(project, t);
      toast.success(copy.video.conductor.importedOk, { id: toastId });
    } catch (e) {
      const tooBig = e instanceof Error && e.message === "clip too large";
      toast.error(tooBig ? copy.composer.clipTooLarge : api.errMessage(e), { id: toastId });
    }
  }

  /** 没有路径就返回。没有哈希或没有工程 id 时抛需要哈希。标题去掉扩展名。 */
  async function startRemake() {
    const path = await pickVideoFile();
    if (!path) return;
    const toastId = toast.loading(copy.video.film.working);
    try {
      const ref = await api.video.putStudioMediaPath(path);
      const hash = String(ref?.cas_hash || ref?.hash || "");
      if (!hash) throw new Error(copy.video.conductor.needHash);
      const base = path.split(/[/\\]/).pop() || copy.video.conductor.remakeDraft;
      const project = await api.video.filmOpen({ hash, title: base.replace(/\.[^.]+$/, "") });
      if (!project?.id) throw new Error(copy.video.conductor.needHash);
      useFilmDesk.getState().setProject(project);
      toast.success(copy.video.film.ready, { id: toastId });
    } catch (e) {
      const tooBig = e instanceof Error && e.message === "clip too large";
      toast.error(tooBig ? copy.composer.clipTooLarge : api.errMessage(e), { id: toastId });
    }
  }

  /** 把这种片子的简报交给 Hypit。structure 选择关系。description 非空时只换题目。 */
  async function startFilm(structure: FilmMode, description = "") {
    const film = copy.video.film;
    const hints: Record<FilmMode, string> = {
      talking: film.talkingHint,
      drama: film.dramaHint,
      podcast: film.podcastHint,
      interview: film.interviewHint,
      ranking: film.rankingHint,
      explainer: film.explainerHint,
      narration: film.narrationHint,
    };
    const typed = description.trim();
    const cast = { ...readCast(structure, film), ...(typed ? { subject: typed } : {}) };
    if (typed) useFilmCast.getState().write(structure, cast);
    useCreateMode.getState().setEngaged(true);
    await onSend({ text: filmCommission(structure, cast, hints[structure]) });
  }

  /** 空 id 返回。打开创作窗格。不是当前会话才切换过去。 */
  async function openCut(id: string) {
    const cutId = String(id || "").trim();
    if (!cutId) return;
    leaveGen.current += 1;
    try {
      const project = await api.video.cutGet(cutId) as CutProject;
      const t = await ensureCutThread(project);
      useCutDesk.getState().setProject({ ...project, session_id: t.id }, t.id);
      useUI.getState().openVideoPane("create");
      if (activeRef.current?.id !== t.id) {
        itemsAcc.current = [];
        setItems([]);
        setQueued(0);
        openThread(t, { keepPane: true });
      }
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 失败时提示，不再往外抛。dest 传给 openCreateHandoffInner。 */
  async function openCreateHandoff(dest: "studio" | "drama" | "canvas") {
    try {
      await openCreateHandoffInner(dest);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 工作室比例只接受 16:9 和 1:1，其余用 9:16。没有工程或剧集 id 就返回。画布没有 id 时焦点落在库。dest 与 studio、drama 比较是否相等。 */
  async function openCreateHandoffInner(dest: "studio" | "drama" | "canvas") {
    if (dest === "studio") {
      const cfg = await api.video.settings().catch(() => ({} as Record<string, string>));
      const aspect = String(cfg?.studio_aspect || "9:16");
      const project = await api.video.createStudio({
        title: copy.video.shellStudio,
        aspect: aspect === "16:9" || aspect === "1:1" ? aspect : "9:16",
      });
      if (!project?.id) return;
      await ensureStudioThread(asStudioProject({
        id: project.id,
        title: project.title || copy.video.shellStudio,
        status: project.status,
        gate: project.gate,
      }));
      void loadVideoHistory();
      return;
    }
    if (dest === "drama") {
      let dramaId = useDramaSelection.getState().dramaId;
      let episodeId = useDramaSelection.getState().episodeId;
      if (!dramaId) {
        const drama = await api.video.createDrama({ title: "", style: defaultDramaStyle(), aspect_ratio: "9:16" });
        dramaId = String(drama?.id || "");
        episodeId = "";
      }
      if (!dramaId) return;
      if (!episodeId) {
        const episode = await api.video.createEpisode(dramaId, copy.video.untitledEp, "");
        episodeId = String(episode?.id || "");
      }
      if (!episodeId) return;
      useDramaSelection.getState().setDramaId(dramaId);
      useDramaSelection.getState().setEpisodeId(episodeId);
      const t = await ensureLaneThread("drama");
      await api.video.bind(t.id, episodeId, dramaId).catch(() => {});
      useUI.getState().openVideoPane("drama");
      if (activeRef.current?.id !== t.id) openThread(t, { keepPane: true });
      void loadVideoHistory();
      return;
    }
    const t = await ensureLaneThread("canvas");
    const canvasId = await ensureCanvasProject(t.id, copy.video.canvas);
    useUI.getState().openVideoPane("canvas", { canvasFocus: canvasId ? "editor" : "library" });
    if (activeRef.current?.id !== t.id) openThread(t, { keepPane: true });
    void loadVideoHistory();
  }

  /** studio 和 drama 先清掉剪辑桌。drama 把 id 当成剧集。canvas 没有 id 时用 session_id 或当前会话补工程。create 有 id 就打开那条剪辑。payload 读取 pane、id、title、drama_id、session_id。 */
  function applyVideoRoute(payload: Record<string, unknown>) {
    const pane = String(payload.pane || "");
    const id = String(payload.id || "");
    const title = String(payload.title || "");
    if (pane === "studio" && id) {
      useCutDesk.getState().clear();
      useStudioSelection.getState().setStudioId(id);
      if (title) useStudioSelection.getState().setStudioMeta({ title });
      useUI.getState().openVideoPane("studio");
      void loadVideoHistory();
      return;
    }
    if (pane === "drama" && id) {
      useCutDesk.getState().clear();
      const dramaId = String(payload.drama_id || "");
      if (dramaId) useDramaSelection.getState().setDramaId(dramaId);
      useDramaSelection.getState().setEpisodeId(id);
      useUI.getState().openVideoPane("drama");
      void loadVideoHistory();
      return;
    }
    if (pane === "canvas") {
      useCutDesk.getState().clear();
      if (id) useCanvasHost.setState({ projectId: id, title: title || copy.video.canvas });
      useUI.getState().openVideoPane("canvas", { canvasFocus: id ? "editor" : "library" });
      if (!id) {
        const sid = String(payload.session_id || activeIdRef.current || "");
        if (sid) void ensureCanvasProject(sid, title || copy.video.canvas);
      }
      void loadVideoHistory();
      return;
    }
    if (pane === "create" && id) {
      void openCut(id);
    }
  }

  handlers.current.slash = onSlash;
  /** review 在视频表面不做。认不出的命令交给 onSlash，并补上斜杠。c 与 review、sidebar、dock、harness、video、run-eval 等比较是否相等，加上 /。 */
  handlers.current.command = (c) => {
    if (c === "review") {
      if (useUI.getState().surface === "video") return;
      setInspector((v) => !v);
    }
    else if (c === "sidebar") setSidebarCollapsed((v) => !v);
    else if (c === "dock") setChatDock((v) => !v);
    else if (c === "harness") openHarness("overview");
    else if (c === "video") openVideo();
    else if (c === "run-eval") void runHarbor("suite");
    else if (c === "run-evolve") void runEvolve();
    else if (c === "palette") setPalette(true);
    else if (c === "control") openSettings();
    else if (c === "rename") useUI.getState().requestRename();
    else if (c === "new") {
      useUI.getState().showConversation();
      void onNewIn();
    }
    else void onSlash("/" + c, "");
  };
  handlers.current.quit = requestQuit;
  /** 空 id 返回。列表里有这条才打开。 */
  handlers.current.focus = (id) => {
    if (!id) return;
    const t = threads.find((x) => x.id === id);
    if (t) openThread(t);
  };
  /** 把路由载荷交给 applyVideoRoute。payload 传给 applyVideoRoute。 */
  handlers.current.route = (payload) => { applyVideoRoute(payload); };
  /** 只在 id 就是当前剪辑工程时刷新。会话 id 沿用桌子上的。payload 读取 id。 */
  handlers.current.cut = (payload) => {
    const id = String(payload.id || "");
    const cur = useCutDesk.getState();
    if (!id || cur.project?.id !== id) return;
    void api.video.cutGet(id).then((row) => {
      if (row?.id) useCutDesk.getState().setProject(row as CutProject, cur.sessionId);
    }).catch(() => {});
  };

  return {
    copy, lab, setLab, surface, harnessTab, setHarnessTab, openHarness, openSettings, closeSettings, openSkills, closeSkills, openPages, closePages, openIdol, closeIdol, openVideo, closeVideo, showConversation, videoBoard, setVideoBoard, canvasStage, setCanvasStage, inspector, setInspector,
    chatDock, setChatDock,
    palette, setPalette, query, setQuery, inspTab, setInspTab, diffMode, setDiffMode,
    sidebarCollapsed, setSidebarCollapsed, sidebarHover, setSidebarHover,
    notices, dismissNotice, clearNotices, renameTick,
    health, savedCfg, setSavedCfg, threads, videoProjects, canvasProjectId, dramaId, active, setActive, items, turnStatus, presenceItems, older, loadingOlder, loadOlder, idle, approvals, running, runStatus, queued, queueItems, ctx, trace, sessionStats, err, setErr,
    outlineTurns, outlineState, activeTurn, setActiveTurn, jumpToTurn, jumpToLatest, jumpTo, latestNonce,
    searchHits, showArchived, setShowArchived,
    diff, hunks, hunkSel, setHunkSel, harness, plugins, evalReport, setEvalReport, bestReport, setBestReport, harborErr, setHarborErr, harborKind, setHarborKind,
    evolve, setEvolve, overlay, overlayK, setOverlayK, overlayMode, setOverlayMode, overlayLibrary, overlayReuseTurn, setOverlayReuseTurn, playbook, setPlaybook, tree, setTree, labBusy, setLabBusy, evolveK, setEvolveK, evolveRounds, setEvolveRounds, evolveSealed, setEvolveSealed, evolveBehavior, setEvolveBehavior, evolveIndex, setEvolveIndex, evolveBaselines, setEvolveBaselines, evolveMaxUsd, setEvolveMaxUsd, bonModels, setBonModels, diffA, setDiffA, diffB, setDiffB, diffOut, setDiffOut,
    booted, aboutOpen, setAboutOpen, aboutInfo, setAboutInfo, pendingDelete, setPendingDelete,
    files, setFiles, skills, packs, methodologyActive: packs.some((p) => p.enabled && p.methodology), logs, setLogs, journal, doctor, vault, pendingQuit, setPendingQuit, setThreads,
    activeId, draftKey, threadRunning, anyRun, needsSetup,
    fail, refresh, onSend, onRetryLast, onContinueLast, onSlash, onStop, onResolve, refreshDiff, applySelected, onNew, onNewIn, ensureThread, ensureLearnScope, openThread, openVideoProject, ingestVideo, startRemake, startFilm, openCut, leaveCut, openCreateHandoff, loadVideoHistory, patchConfig, requestQuit,
    refreshTrace, loadSpill, refreshCtx, reloadSkills,
    /** 没有当前会话就返回。取消后按列表重写条数。id 传给 api.queueCancel。 */
    onQueueCancel: async (id: string) => {
      if (!activeId) return;
      await api.queueCancel(activeId, id);
      const q = await api.queueList(activeId).catch(() => []);
      setQueueItems(q);
      setQueued(q.length);
    },
    /** 没有当前会话就返回。调整后按列表重写条数。id 传给 api.queueReorder。delta 传给 api.queueReorder。 */
    onQueueReorder: async (id: string, delta: number) => {
      if (!activeId) return;
      await api.queueReorder(activeId, id, delta);
      const q = await api.queueList(activeId).catch(() => []);
      setQueueItems(q);
      setQueued(q.length);
    },
    /** 没有当前会话就返回。成功后把返回的会话并进列表。 */
    onApplyWorktree: async () => {
      if (!activeId) return;
      const t = await api.applySessionWorktree(activeId);
      setActive(t);
      setThreads((list) => list.map((x) => (x.id === t.id ? { ...x, ...t } : x)));
      toast.success(copy.app.worktreeApplied);
    },
    /** 先刷新会话列表。找到这条才打开。id 与 x.id 比较是否相等。 */
    onOpenChild: async (id: string) => {
      const fresh = await api.listSessions().catch(() => threads);
      setThreads(fresh);
      const t = fresh.find((x) => x.id === id) || threads.find((x) => x.id === id);
      if (t) openThread(t);
    },
    /** 没有当前会话就返回。成功后提示返回的说明，没有说明就用已恢复文案，并刷新差异。from 传给 api.restoreSessionFiles。 */
    onRestoreFiles: async (from: string) => {
      if (!activeId) return;
      try {
        const note = await api.restoreSessionFiles(activeId, from);
        toast.success(note || copy.app.restored);
        void refreshDiff();
      } catch (e) {
        fail(e);
      }
    },
    /** 没有当前会话就返回。新会话插到最前并打开。from 传给 api.forkSession。 */
    onForkFrom: async (from: string) => {
      if (!activeId) return;
      try {
        const t = await api.forkSession(activeId, from);
        setThreads((prev) => [t, ...prev.filter((x) => x.id !== t.id)]);
        openThread(t);
        toast.success(copy.app.forked);
      } catch (e) {
        fail(e);
      }
    },
    runHarbor, runHarborAt, runEvolve, runOverlay, compareHarness, checkoutHarness, rollbackHarness, revealHarness,
  };
}
