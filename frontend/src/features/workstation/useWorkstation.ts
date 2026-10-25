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
import { channelForSurface, isSubagentThread, parentSessionId, threadChannel } from "../../lib/protocol";
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

function sliceNum(p: Record<string, any>, snake: string, camel: string, prev?: number): number | undefined {
  if (p && (snake in p || camel in p)) return num(p[snake] ?? p[camel], 0);
  return prev;
}

const emptyHealth: Health = { ok: false, harness: "", model: "", version: "", isolated: false, isolationKind: "", budgetUsd: 0, usageUsd: 0, workspaceReady: false };
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
const emptyCtx: ContextUsage = { tokens: 0, budget: 0, window: 0, prefixTokens: 0, dynamicTokens: 0, schemaTokens: 0, providerPrompt: 0, note: "", layers: [], elided: 0 };

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

function resolveOperatorThread(list: Thread[], t: Thread | null, channel: ThreadChannel): Thread | null {
  if (!t) return null;
  if (!isSubagentThread(t) && threadChannel(t) === channel) return t;
  const pid = t.parentId || parentSessionId(t.id);
  const parent = list.find((x) => x.id === pid && !isSubagentThread(x) && threadChannel(x) === channel);
  if (parent) return parent;
  return list.find((x) => !isSubagentThread(x) && threadChannel(x) === channel && !x.archived)
    || list.find((x) => !isSubagentThread(x) && threadChannel(x) === channel)
    || null;
}

function pickChannelThread(list: Thread[], channel: ThreadChannel, preferred: string, cur: Thread | null): Thread | null {
  const chats = list.filter((t) => !isSubagentThread(t) && threadChannel(t) === channel);
  if (cur && list.some((t) => t.id === cur.id)) {
    return resolveOperatorThread(list, list.find((t) => t.id === cur.id) || cur, channel);
  }
  if (preferred) {
    const hit = list.find((t) => t.id === preferred);
    if (hit) return resolveOperatorThread(list, hit, channel);
  }
  return chats.find((t) => !t.archived) || chats[0] || null;
}

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

function runningMap(ids: string[], prev: Record<string, boolean> = {}): Record<string, boolean> {
  const next: Record<string, boolean> = {};
  for (const id of Object.keys(prev)) next[id] = false;
  for (const id of ids) next[id] = true;
  return next;
}

/**
 * After the stream says a turn ended, a poll answer that is still in flight
 * (or a backend still doing post-turn bookkeeping) may answer `running: true`.
 * Treat the event as authoritative for a short grace window so the working
 * line and tool spinners cannot flicker back on a finished turn.
 */
const SETTLE_GRACE_MS = 6000;

function settled(endedAt: Record<string, number>, id: string): boolean {
  const t = endedAt[id];
  return !!t && Date.now() - t < SETTLE_GRACE_MS;
}

function withTimeout<T>(p: Promise<T>, ms: number, label: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const t = window.setTimeout(() => reject(new Error(`${label} timed out`)), ms);
    p.then(
      (v) => { window.clearTimeout(t); resolve(v); },
      (e) => { window.clearTimeout(t); reject(e); },
    );
  });
}

function newRequestId(): string {
  try {
    if (typeof crypto !== "undefined" && crypto.randomUUID) return crypto.randomUUID();
  } catch {
    /* ignore */
  }
  return `req-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function commitHot(fold: ConversationFold, list: Item[]): Item[] {
  const next = foldTurnErrors(list);
  const trimmed = lastUserTurns(next, HOT_TRANSCRIPT_TURNS);
  if (trimmed.length !== next.length) fold.trim(trimmed);
  return trimmed;
}

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

function readInspTab(id: string) {
  try {
    return coerceInspTab(localStorage.getItem(`yoyo-insp-${id}`));
  } catch {
    /* ignore */
  }
  return null;
}

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
  // Notices age out on their own; a minute of slack is fine for a window that lasts a day.
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
    slash: async (_cmd: string, _rest: string) => {},
    command: (_c: string) => {},
    quit: () => {},
    focus: (_id: string) => {},
    route: (_payload: Record<string, unknown>) => {},
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

  const refreshCtx = useCallback(() => {
    if (!activeId) {
      setCtx(emptyCtx);
      return;
    }
    api.contextUsage(activeId).then(setCtx).catch(() => {});
  }, [activeId]);

  const refreshTrace = useCallback(async () => {
    if (!activeId) {
      setTrace(null);
      return;
    }
    try {
      setTrace(await api.sessionTrace(activeId));
    } catch {
      /* keep last projection */
    }
  }, [activeId]);

  const loadSpill = useCallback(async (blobID: string): Promise<SpillBlob> => {
    if (!activeId) return { id: blobID, bytes: 0, text: "", truncated: false };
    return api.spillBlob(activeId, blobID);
  }, [activeId]);

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
      /* keep last directory */
    }
  }, [activeId]);

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

  const fail = (e: unknown) => {
    const msg = bannerError(api.errMessage(e), copy.transcript);
    setErr(msg);
    toast.error(msg);
  };

  const markEnded = useCallback((id: string) => {
    if (!id) return;
    endedAt.current[id] = Date.now();
    setRunning((m) => (m[id] ? { ...m, [id]: false } : m));
  }, []);

  const markStarted = useCallback((id: string) => {
    if (!id) return;
    delete endedAt.current[id];
    setRunning((m) => ({ ...m, [id]: true }));
  }, []);

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
      /* ignore */
    }
  }, []);

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
        const preferred = useUI.getState().lastThreadId(ch);
        return pickChannelThread(list, ch, preferred, cur);
      });
      setBooted(true);
      try { setPlaybook(await api.playbook()); } catch { /* optional */ }
      try { setTree(await api.archive()); } catch { /* optional */ }
      try { setVault(await api.keyStatus()); } catch { /* optional */ }
      try {
        const last = await api.lastEval();
        if (last && (last.suite || last.Suite || last.results || last.Results)) setEvalReport(last);
      } catch { /* optional */ }
      try {
        const last = await api.lastEvolve();
        if (last && (last.tried || last.Tried || last.compare || last.Compare || last.promoted || last.Promoted)) setEvolve(last);
      } catch { /* optional */ }
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
    setActive((cur) => pickChannelThread(threads, ch, preferred, cur));
  }, [surface]);
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
      /* ignore */
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
        /* browser */
      }
    })();
    return () => {
      alive = false;
      offs.forEach((o) => o());
    };
  }, [openSettings]);

  useEffect(() => subscribeItems((item) => {
    // The loop emits exactly one `system` event when a run starts, so a queued
    // turn kicked right after turn_end lifts the settle grace immediately.
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
    const flushItems = () => {
      itemsRaf.current = 0;
      if (browsingRef.current) {
        setItems(withLiveTail(itemsAcc.current, hotItemsRef.current));
      } else {
        setItems(hotItemsRef.current.slice());
      }
    };
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
          // Repair after a hole in the live tail. The seat fold already holds
          // streamed text, so keep the browse position and the review pane.
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
      // A turn in flight may have lost its tail event; pull the ring so the
      // transcript catches up within one poll instead of waiting for a notice.
      if (live) catchUpSession(activeId);
    }, 1500);
    return () => {
      alive = false;
      window.clearInterval(t);
    };
  }, [anyRun, activeId, syncRunning]);

  async function currentChannel(): Promise<ThreadChannel> {
    return channelForSurface(useUI.getState().surface);
  }

  function channelWorkspace(ch: ThreadChannel): string {
    if (ch === "video") return savedCfg.videoWorkspace || health.videoWorkspace || savedCfg.workspace;
    return savedCfg.workspace;
  }

  function activateLaneThread(t: Thread, lane: VideoLane) {
    rememberLaneThread(lane, t.id);
    if (activeRef.current?.id === t.id) return;
    itemsAcc.current = [];
    setItems([]);
    setQueued(0);
    openThread(t, { keepPane: true });
  }

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

  function findVideoThread(id: string): Thread | null {
    const sid = (id || "").trim();
    if (!sid) return null;
    return threadsRef.current.find((t) => t.id === sid) || (activeRef.current?.id === sid ? activeRef.current : null);
  }

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

  async function prepareWelcomeThread(): Promise<Thread> {
    const inflight = laneThreadJobs.current.create;
    if (inflight) return inflight;
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

  async function ensureCutThread(project: CutProject): Promise<Thread> {
    const projectId = str(project.id);
    if (!projectId) throw new Error("cut project required");
    const inflight = cutThreadJobs.current[projectId];
    if (inflight) return inflight;
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

  async function ensureLaneThread(lane: VideoLane): Promise<Thread> {
    if (lane === "create") {
      const t = await prepareWelcomeThread();
      activateLaneThread(t, lane);
      return t;
    }
    const inflight = laneThreadJobs.current[lane];
    if (inflight) return inflight;
    const job = (async () => {
      const projects = videoProjectsRef.current;
      const chats = threadsRef.current.filter((t) => !isSubagentThread(t) && threadChannel(t) === "video");
      const remembered = lastLaneThread(lane);
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
        /* the desk can stay on the thread it already has */
      }
      return;
    }
    await ensureLaneThread("create");
  }

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
      /* bind lookup is optional */
    }
  }

  function rememberStudioProject(p: VideoProject, sessionId: string) {
    const row = asStudioProject({ ...p, sessionId });
    setVideoProjects((prev) => {
      const next = [row, ...prev.filter((x) => !(x.kind === "studio" && x.id === p.id))];
      videoProjectsRef.current = next;
      return next;
    });
  }

  async function ensureStudioThread(p: VideoProject): Promise<Thread> {
    const projectId = str(p.id);
    if (!projectId) {
      throw new Error("studio project required");
    }
    const inflight = studioThreadJobs.current[projectId];
    if (inflight) return inflight;
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

  async function onContinueLast() {
    const ch = surface === "video" ? "video" : "agent";
    const last = useUI.getState().lastThreadId(ch);
    const hit = threads.find((t) => t.id === last)
      || threads.find((t) => t.interrupted || (t.queued || 0) > 0)
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

  function requestQuit() {
    if (anyRun) {
      setPendingQuit(true);
      return;
    }
    void api.quit();
  }

  async function onStop() {
    if (!activeId) return;
    // Do not mark the turn ended here. The backend unwinds asynchronously: a
    // running tool or model stream can still emit output after the click. The
    // optimistic end made that tail look like a ghost run. The turn closes on
    // its terminal event (turn_end / error / stopped) or on the runningIDs poll.
    void api.interrupt(activeId).catch(() => {});
  }

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

  async function onResolve(id: string, decision: string, answer?: string) {
    await api.resolveApproval(id, decision, answer);
    setApprovals((prev) => prev.filter((a) => a.id !== id));
  }

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

  async function onNew() {
    if (useUI.getState().surface === "video") {
      useUI.getState().openVideoPane("create");
      return;
    }
    await onNewIn();
  }

  function openThread(t: Thread, opts?: { keepPane?: boolean }) {
    const ui = useUI.getState();
    ui.rememberThread(t);
    // Looking at a conversation reads its notices, and a pending pick belonged to a fresh chat, not this one.
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
    } else {
      ui.showConversation();
    }
    setActive(t);
  }

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
      try { setOverlayLibrary(await api.overlayLibrary(activeId)); } catch { /* empty library is fine */ }
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

  async function checkoutHarness(hash: string, l3?: boolean) {
    await api.checkout(hash, !!l3);
    await refresh();
    toast.success(copy.app.checkedOut);
  }

  async function rollbackHarness() {
    await api.rollback();
    await refresh();
    toast.success(copy.app.rolledBack);
  }

  async function revealHarness(hash?: string) {
    try {
      await api.revealHarness(hash || "");
    } catch (e) {
      fail(e);
      throw e;
    }
  }

  async function patchConfig(partial: Partial<AppConfig>) {
    const next = { ...savedCfg, ...partial };
    setSavedCfg(next);
    await api.setConfig(next);
  }

  const leaveGen = useRef(0);
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
        // Menus and confirm sheets own the first Escape. Leaving the module
        // on the same keypress made "More checks" feel like a trap.
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

  async function startFilm(structure: "ranking" | "podcast" | "interview", description = "") {
    const toastId = toast.loading(copy.video.film.working);
    try {
      const project = await api.video.filmDescribe(structure, "", description);
      if (!project?.id) throw new Error(copy.video.film.needPicture);
      useFilmDesk.getState().setProject(project);
      toast.success(copy.video.film.ready, { id: toastId });
    } catch (e) {
      toast.error(api.errMessage(e), { id: toastId });
    }
  }

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

  async function openCreateHandoff(dest: "studio" | "drama" | "canvas") {
    try {
      await openCreateHandoffInner(dest);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

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
  handlers.current.focus = (id) => {
    if (!id) return;
    const t = threads.find((x) => x.id === id);
    if (t) openThread(t);
  };
  handlers.current.route = (payload) => { applyVideoRoute(payload); };
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
    fail, refresh, onSend, onRetryLast, onContinueLast, onSlash, onStop, onResolve, refreshDiff, applySelected, onNew, onNewIn, ensureThread, openThread, openVideoProject, ingestVideo, startRemake, startFilm, openCut, leaveCut, openCreateHandoff, loadVideoHistory, patchConfig, requestQuit,
    refreshTrace, loadSpill, refreshCtx, reloadSkills,
    onQueueCancel: async (id: string) => {
      if (!activeId) return;
      await api.queueCancel(activeId, id);
      const q = await api.queueList(activeId).catch(() => []);
      setQueueItems(q);
      setQueued(q.length);
    },
    onQueueReorder: async (id: string, delta: number) => {
      if (!activeId) return;
      await api.queueReorder(activeId, id, delta);
      const q = await api.queueList(activeId).catch(() => []);
      setQueueItems(q);
      setQueued(q.length);
    },
    onApplyWorktree: async () => {
      if (!activeId) return;
      const t = await api.applySessionWorktree(activeId);
      setActive(t);
      setThreads((list) => list.map((x) => (x.id === t.id ? { ...x, ...t } : x)));
      toast.success(copy.app.worktreeApplied);
    },
    onOpenChild: async (id: string) => {
      const fresh = await api.listSessions().catch(() => threads);
      setThreads(fresh);
      const t = fresh.find((x) => x.id === id) || threads.find((x) => x.id === id);
      if (t) openThread(t);
    },
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
