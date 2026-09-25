import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import * as api from "../../lib/client";
import { bannerError, classifyItem, shortError } from "../../lib/error";
import { dropTrailingErrors, foldLiveIntoSeed, mergeItem, subscribeItems, subscribeSession, subscribeSessions } from "../../lib/stream";
import { asArray, num, str } from "../../lib/normalize";
import { pathReady, workspaceReady } from "../../lib/workspace";
import { applyLocale, useCopy } from "../../lib/i18n";
import { mergeKeymap, matchKey } from "../../lib/keymap";
import { useUI } from "../../lib/store";
import { HARNESS_TABS } from "../../lib/surface";
import { applyUiScale } from "../../lib/scale";
import { parseDarkPalette, parseLightPalette, parseThemePref, useTheme } from "../../lib/theme";
import { readPopoutId } from "../../lib/popout";
import type { AppConfig, Approval, Attachment, ContextUsage, FileHit, HarborKind, Health, Hunk, Item, RunStatus, SessionTrace, SkillInfo, SpillBlob, Thread, ThreadChannel, VideoProject } from "../../lib/protocol";
import { channelForSurface, threadChannel } from "../../lib/protocol";
import { bindVideoThread } from "../video/bind";
import { ensureCanvasProject, useCanvasHost } from "../video/canvas-host/session";
import { useDramaSelection } from "../video/workshop-store";

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
  updateUrl: "",
  keymap: {},
  locale: "",
  alwaysOnTop: false,
  startAtLogin: false,
  notificationsEnabled: true,
  notifyWhenUnfocusedOnly: false,
  uiScale: 1,
  updateChannel: "nightly",
  theme: "system",
  paletteDark: "ink",
  paletteLight: "neutral",
  gateMode: "manual",
  crashResume: true,
  searchUrl: "",
  searchKey: "",
};
const emptyCtx: ContextUsage = { tokens: 0, budget: 0, window: 0, prefixTokens: 0, dynamicTokens: 0, schemaTokens: 0, providerPrompt: 0, note: "", layers: [], elided: 0 };

function pickChannelThread(list: Thread[], channel: ThreadChannel, preferred: string, cur: Thread | null): Thread | null {
  if (cur && threadChannel(cur) === channel && list.some((t) => t.id === cur.id)) {
    return list.find((t) => t.id === cur.id) || cur;
  }
  if (preferred) {
    const hit = list.find((t) => t.id === preferred && threadChannel(t) === channel);
    if (hit) return hit;
  }
  return list.find((t) => threadChannel(t) === channel && !t.archived)
    || list.find((t) => threadChannel(t) === channel)
    || null;
}

function videoProjectOf(v: any): VideoProject | null {
  const id = str(v?.id);
  if (!id) return null;
  return {
    kind: str(v?.kind) === "drama" ? "drama" : "canvas",
    id,
    title: str(v?.title),
    sessionId: str(v?.session_id || v?.sessionId),
    episodeId: str(v?.episode_id || v?.episodeId),
    updatedAt: str(v?.updated_at || v?.updatedAt),
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

function localUser(sessionId: string, text: string): Item {
  return {
    key: `ui:${sessionId}:${Date.now()}`,
    type: "user",
    sessionId,
    source: "ui",
    ts: new Date().toISOString(),
    text,
    name: "",
    delta: false,
    payload: { text },
  };
}

function readInspTab(id: string) {
  try {
    const v = localStorage.getItem(`yoyo-insp-${id}`);
    if (v === "diff" || v === "files" || v === "trace" || v === "queue" || v === "memory") return v;
  } catch {
    /* ignore */
  }
  return null;
}

export function useWorkstation() {
  const copy = useCopy();
  const { setPref, setDarkPalette, setLightPalette } = useTheme();
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
  const diffMode = useUI((s) => s.diffMode);
  const setDiffMode = useUI((s) => s.setDiffMode);
  const sidebarCollapsed = useUI((s) => s.sidebarCollapsed);
  const setSidebarCollapsed = useUI((s) => s.setSidebarCollapsed);
  const sidebarHover = useUI((s) => s.sidebarHover);
  const setSidebarHover = useUI((s) => s.setSidebarHover);
  const notices = useUI((s) => s.notices);
  const noticesOpen = useUI((s) => s.noticesOpen);
  const setNoticesOpen = useUI((s) => s.setNoticesOpen);
  const pushNotice = useUI((s) => s.pushNotice);
  const clearNotices = useUI((s) => s.clearNotices);
  const renameTick = useUI((s) => s.renameTick);

  const [health, setHealth] = useState<Health>(emptyHealth);
  const [savedCfg, setSavedCfg] = useState<AppConfig>(emptyCfg);
  const [threads, setThreads] = useState<Thread[]>([]);
  const [videoProjects, setVideoProjects] = useState<VideoProject[]>([]);
  const canvasProjectId = useCanvasHost((s) => s.projectId);
  const dramaId = useDramaSelection((s) => s.dramaId);
  const [active, setActive] = useState<Thread | null>(null);
  const [items, setItems] = useState<Item[]>([]);
  const itemsAcc = useRef<Item[]>([]);
  const [queued, setQueued] = useState(0);
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
  const [showArchived, setShowArchived] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [aboutInfo, setAboutInfo] = useState<Record<string, any>>({});
  const [pendingDelete, setPendingDelete] = useState<Thread | null>(null);
  const [files, setFiles] = useState<FileHit[]>([]);
  const [skills, setSkills] = useState<SkillInfo[]>([]);
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
  });

  const activeId = active?.id || "";
  activeIdRef.current = activeId;
  const draftKey = activeId || "_new";
  const threadRunning = !!running[activeId];
  const anyRun = Object.values(running).some(Boolean);
  const needsSetup = !workspaceReady(health.workspaceReady, savedCfg.workspace);

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
      const [ids, status] = await Promise.all([api.runningIDs(), api.runningStatus().catch(() => [] as RunStatus[])]);
      const live = ids.filter((id) => !settled(endedAt.current, id));
      setRunning((prev) => runningMap(live, prev));
      setRunStatus(status.filter((s) => !settled(endedAt.current, s.id)));
    } catch {
      /* ignore */
    }
  }, []);

  const refresh = useCallback(async () => {
    try {
      const [h, c, list, hs, pl] = await withTimeout(Promise.all([
        api.health(),
        api.getConfig(),
        api.listSessions(),
        api.harness().catch(() => ({})),
        api.plugins().catch(() => ({})),
      ]), 8000, "boot");
      setHealth(h);
      setSavedCfg(c);
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
      fail(e);
    } finally {
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
  }, [savedCfg.locale, savedCfg.uiScale, savedCfg.theme, savedCfg.paletteDark, savedCfg.paletteLight, setPref, setDarkPalette, setLightPalette]);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => subscribeSessions(refresh), [refresh]);
  useEffect(() => {
    if (active) useUI.getState().rememberThread(active);
  }, [active?.id]);
  useEffect(() => {
    const ch = channelForSurface(surface);
    const preferred = useUI.getState().lastThreadId(ch);
    setActive((cur) => pickChannelThread(threads, ch, preferred, cur));
  }, [surface]);
  const loadVideoHistory = useCallback(async () => {
    try {
      const raw = await api.video.history();
      const list = Array.isArray(raw) ? raw : raw?.items || raw?.projects || [];
      setVideoProjects(asArray(list).map(videoProjectOf).filter((p): p is VideoProject => !!p));
    } catch {
      setVideoProjects([]);
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
    if (!booted) return;
    const path = active?.workspace || savedCfg.workspace;
    api.listSkills(path).then(setSkills).catch(() => {});
  }, [booted, active?.workspace, savedCfg.workspace]);

  const reloadSkills = useCallback(() => {
    const path = active?.workspace || savedCfg.workspace;
    api.listSkills(path).then(setSkills).catch(() => {});
  }, [active?.workspace, savedCfg.workspace]);
  useEffect(() => {
    if (surface !== "settings") return;
    api.logs().then(setLogs).catch(() => {});
    api.journal().then(setJournal).catch(() => {});
    api.doctor().then(setDoctor).catch(() => {});
  }, [surface]);

  useEffect(() => {
    if (!activeId) return;
    const saved = readInspTab(activeId);
    if (saved) setInspTab(saved);
  }, [activeId, setInspTab]);

  useEffect(() => {
    if (!activeId) return;
    try {
      localStorage.setItem(`yoyo-insp-${activeId}`, inspTab);
    } catch {
      /* ignore */
    }
  }, [activeId, inspTab]);

  useEffect(() => {
    if (inspTab !== "trace") return;
    void refreshTrace();
    if (!threadRunning) return;
    const timer = window.setInterval(() => { void refreshTrace(); }, 2000);
    return () => window.clearInterval(timer);
  }, [inspTab, refreshTrace, threadRunning]);

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
        offs.push(Events.On("yoyo:about", (info: any) => { setAboutInfo(info || {}); setAboutOpen(true); }));
        offs.push(Events.On("yoyo:workspace", (path: any) => {
          const p = typeof path === "string" ? path : "";
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
        offs.push(Events.On("yoyo:command", (cmd: any) => handlers.current.command(String(cmd || ""))));
        offs.push(Events.On("yoyo:quit", () => handlers.current.quit()));
        offs.push(Events.On("yoyo:focus", (session: any) => handlers.current.focus(typeof session === "string" ? session : "")));
        offs.push(Events.On("yoyo:doctor", (info: any) => { setDoctor(info || {}); openSettings("advanced", "advanced-doctor"); }));
        offs.push(Events.On("yoyo:logs", (info: any) => { setLogs(info || {}); openSettings("advanced", "advanced-logs"); }));
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
    // A deliberate Stop lands as a "canceled" error card in the stream; it is
    // not news worth a notice.
    const stopped = item.type === "error" && classifyItem(item).kind === "canceled";
    if ((item.type === "turn_end" || item.type === "approval" || item.type === "error") && !stopped) {
      pushNotice({
        id: item.key || `${item.type}-${Date.now()}`,
        title: item.type === "approval" ? copy.app.approval : item.type === "error" ? copy.app.errorNotice : copy.app.turnFinished,
        body: item.type === "error" ? shortError(item, copy.transcript) : (item.text || item.type),
        ts: item.ts,
        sessionId: item.sessionId,
      });
    }
  }), [pushNotice, markEnded, markStarted]);

  useEffect(() => {
    if (!activeId) {
      itemsAcc.current = [];
      setItems([]);
      setCtx(emptyCtx);
      return;
    }
    itemsAcc.current = [];
    setItems([]);
    setQueued(0);
    setCtx(emptyCtx);
    const unsub = subscribeSession(
      activeId,
      (item) => {
        itemsAcc.current = mergeItem(itemsAcc.current, item);
        setItems(itemsAcc.current.slice());
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
        if (item.type === "turn_end" || item.type === "error") {
          markEnded(activeId);
          api.contextUsage(activeId).then(setCtx).catch(() => {});
          api.queueList(activeId).then((q) => setQueued(q.length)).catch(() => setQueued(0));
        }
      },
      (seed) => {
        itemsAcc.current = foldLiveIntoSeed(seed, itemsAcc.current);
        setItems(itemsAcc.current.slice());
      },
    );
    api.approvals().then(setApprovals).catch(() => {});
    api.contextUsage(activeId).then(setCtx).catch(() => {});
    api.running(activeId)
      .then((live) => setRunning((m) => ({ ...m, [activeId]: live && !settled(endedAt.current, activeId) })))
      .catch(() => {});
    return unsub;
  }, [activeId, markEnded]);

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

  async function restoreVideoProject(sessionId: string) {
    if (!sessionId) return;
    try {
      const st = await api.video.sessionProject(sessionId);
      const canvasId = str(st?.canvas_id || (st?.kind === "canvas" ? st?.id : ""));
      const nextDrama = str(st?.drama_id);
      const nextEpisode = str(st?.episode_id);
      const title = str(st?.title || st?.drama_title);
      if (canvasId) useCanvasHost.setState({ projectId: canvasId, title: title || copy.video.canvas });
      if (nextDrama) {
        useDramaSelection.getState().setDramaId(nextDrama);
        if (nextEpisode) useDramaSelection.getState().setEpisodeId(nextEpisode);
      }
      const kind = str(st?.kind);
      if (kind === "canvas") useUI.getState().setVideoMode("canvas");
      else if (kind === "drama") useUI.getState().setVideoMode("drama");
    } catch {
      /* bind lookup is optional */
    }
  }

  async function openVideoProject(p: VideoProject) {
    const ch: ThreadChannel = "video";
    const ui = useUI.getState();
    const staged = ui.videoBoard || ui.canvasStage || ui.videoPane !== "chat";
    let t = p.sessionId ? threads.find((x) => x.id === p.sessionId) || null : null;
    if (!t) {
      const created = await api.createSession(channelWorkspace(ch), ch);
      t = created.channel ? created : { ...created, channel: ch };
      setThreads((prev) => [t!, ...prev.filter((x) => x.id !== t!.id)]);
      if (p.kind === "canvas") {
        await api.video.canvasBind(t.id, p.id).catch(() => {});
        useCanvasHost.setState({ projectId: p.id, title: p.title || copy.video.canvas });
        if (staged) ui.openVideoPane("canvas", { canvasFocus: "editor" });
        else ui.setVideoMode("canvas");
      } else {
        if (p.episodeId) await api.video.bind(t.id, p.episodeId).catch(() => {});
        useDramaSelection.getState().setDramaId(p.id);
        if (p.episodeId) useDramaSelection.getState().setEpisodeId(p.episodeId);
        if (staged) ui.openVideoPane("drama");
        else ui.setVideoMode("drama");
      }
      if (p.title) await api.renameSession(t.id, p.title).catch(() => {});
      t = { ...t, title: p.title || t.title };
      setThreads((prev) => prev.map((x) => (x.id === t!.id ? t! : x)));
    } else if (p.kind === "canvas") {
      useCanvasHost.setState({ projectId: p.id, title: p.title || copy.video.canvas });
      await api.video.canvasBind(t.id, p.id).catch(() => {});
      if (staged) ui.openVideoPane("canvas", { canvasFocus: "editor" });
      else ui.setVideoMode("canvas");
    } else {
      useDramaSelection.getState().setDramaId(p.id);
      if (p.episodeId) useDramaSelection.getState().setEpisodeId(p.episodeId);
      if (p.episodeId) await api.video.bind(t.id, p.episodeId).catch(() => {});
      if (staged) ui.openVideoPane("drama");
      else ui.setVideoMode("drama");
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
    const created = await api.createSession(channelWorkspace(ch), ch);
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
      const t = await ensureThread();
      if (useUI.getState().surface === "video" && useUI.getState().videoMode === "drama") {
        await bindVideoThread(t.id).catch(() => {});
      }
      if (useUI.getState().surface === "video" && useUI.getState().videoMode === "canvas") {
        await ensureCanvasProject(t.id).catch(() => {});
        void loadVideoHistory();
      }
      if (opts?.steer && running[t.id]) {
        await api.steer(t.id, text);
        useUI.getState().patchDrafts({ [t.id]: "", _new: "" });
        if (t.id === activeId) {
          itemsAcc.current = mergeItem(itemsAcc.current, {
            ...localUser(t.id, text),
            source: "steer",
            key: `ui-steer:${t.id}:${Date.now()}`,
          });
          setItems(itemsAcc.current.slice());
        }
        toast.success(copy.app.steered);
        return;
      }
      const wasRunning = !!running[t.id];
      useUI.getState().patchDrafts({ [t.id]: "", _new: "" });
      markStarted(t.id);
      if (t.id === activeId) {
        itemsAcc.current = mergeItem(itemsAcc.current, localUser(t.id, text));
        setItems(itemsAcc.current.slice());
      }
      const res = await api.send(t.id, text, { plan: useUI.getState().plan, attachments: opts?.attachments });
      if (wasRunning || res.queued) {
        setQueued((n) => n + 1);
        toast.message(copy.app.queued);
      } else {
        setQueued(0);
      }
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

  async function onRetryLast() {
    if (!activeId || running[activeId]) return;
    const hasTurn = itemsAcc.current.some((it) => (it.type === "user" && it.source !== "steer") || it.type === "assistant");
    if (!hasTurn) return;
    setErr("");
    markStarted(activeId);
    try {
      await api.retry(activeId);
      itemsAcc.current = dropTrailingErrors(itemsAcc.current);
      setItems(itemsAcc.current.slice());
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
      const text = rest.trim();
      if (!text) return;
      await api.memoryWrite("episodic", text);
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
      const t = await api.forkSession(activeId);
      setThreads((prev) => [t, ...prev]);
      setActive(t);
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
      const focus = rest.trim() ? "from " + rest.trim() : "from ";
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
      await navigator.clipboard.writeText(md);
      toast.success(copy.app.exported);
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
    await api.interrupt(activeId);
    markEnded(activeId);
  }

  async function onResolve(id: string, decision: string) {
    await api.resolveApproval(id, decision);
    setApprovals((prev) => prev.filter((a) => a.id !== id));
  }

  async function refreshDiff() {
    try {
      const h = await api.workspaceHunks(active?.workspace || savedCfg.workspace || "");
      setDiff(h.diff);
      setHunks(h.hunks);
      setHunkSel({});
      if (h.hunks.length) setInspector(true);
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
    const created = await api.createSession(workspace || channelWorkspace(ch), ch);
    const t = created.channel ? created : { ...created, channel: ch };
    setThreads((prev) => [t, ...prev.filter((x) => x.id !== t.id)]);
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
    if (threadChannel(t) === "video") {
      if (opts?.keepPane) {
        if (ui.surface !== "video") useUI.setState({ surface: "video" });
      } else {
        useUI.setState({ surface: "video", videoBoard: false, canvasStage: false, videoPane: "chat" });
      }
    } else {
      ui.showConversation();
    }
    setActive(t);
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
        if (useUI.getState().surface === "video") return;
        if (useUI.getState().surface === "harness") setChatDock((v) => !v);
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
        if (useUI.getState().palette) {
          setPalette(false);
          return;
        }
        const ui = useUI.getState();
        if (ui.videoBoard || ui.canvasStage || ui.videoPane !== "chat") {
          ui.openVideoPane("chat");
          return;
        }
        if (useUI.getState().surface === "settings" || useUI.getState().surface === "skills" || useUI.getState().surface === "harness") {
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
  }, [savedCfg.keymap, approvals, openSettings, showConversation, setInspector, setPalette, setLab]);

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
    else if (c === "new") void onNew();
    else void onSlash("/" + c, "");
  };
  handlers.current.quit = requestQuit;
  handlers.current.focus = (id) => {
    if (!id) return;
    const t = threads.find((x) => x.id === id);
    if (t) openThread(t);
  };

  return {
    copy, lab, setLab, surface, harnessTab, setHarnessTab, openHarness, openSettings, closeSettings, openSkills, closeSkills, openVideo, closeVideo, showConversation, videoBoard, setVideoBoard, canvasStage, setCanvasStage, inspector, setInspector,
    chatDock, setChatDock,
    palette, setPalette, query, setQuery, inspTab, setInspTab, diffMode, setDiffMode,
    sidebarCollapsed, setSidebarCollapsed, sidebarHover, setSidebarHover,
    notices, noticesOpen, setNoticesOpen, clearNotices, renameTick,
    health, savedCfg, setSavedCfg, threads, videoProjects, canvasProjectId, dramaId, active, setActive, items, approvals, running, runStatus, queued, ctx, trace, err, setErr,
    diff, hunks, hunkSel, setHunkSel, harness, plugins, evalReport, setEvalReport, bestReport, setBestReport, harborErr, setHarborErr, harborKind, setHarborKind,
    evolve, setEvolve, playbook, setPlaybook, tree, setTree, labBusy, setLabBusy, evolveK, setEvolveK, evolveRounds, setEvolveRounds, evolveSealed, setEvolveSealed, evolveBehavior, setEvolveBehavior, evolveIndex, setEvolveIndex, evolveBaselines, setEvolveBaselines, evolveMaxUsd, setEvolveMaxUsd, bonModels, setBonModels, diffA, setDiffA, diffB, setDiffB, diffOut, setDiffOut,
    booted, showArchived, setShowArchived, aboutOpen, setAboutOpen, aboutInfo, setAboutInfo, pendingDelete, setPendingDelete,
    files, setFiles, skills, logs, setLogs, journal, doctor, vault, pendingQuit, setPendingQuit, setThreads,
    activeId, draftKey, threadRunning, anyRun, needsSetup,
    fail, refresh, onSend, onRetryLast, onSlash, onStop, onResolve, refreshDiff, applySelected, onNew, onNewIn, ensureThread, openThread, openVideoProject, loadVideoHistory, patchConfig, requestQuit,
    refreshTrace, loadSpill, refreshCtx, reloadSkills,
    runHarbor, runEvolve, compareHarness, checkoutHarness, rollbackHarness, revealHarness,
  };
}
