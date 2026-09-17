import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import * as api from "../../lib/client";
import { bannerError, shortError } from "../../lib/error";
import { dropTrailingErrors, mergeItem, mergePendingUsers, subscribeItems, subscribeSession, subscribeSessions } from "../../lib/stream";
import { num, str } from "../../lib/normalize";
import { workspaceReady } from "../../lib/workspace";
import { applyLocale, useCopy } from "../../lib/i18n";
import { mergeKeymap, matchKey } from "../../lib/keymap";
import { useUI } from "../../lib/store";
import { applyUiScale } from "../../lib/scale";
import { useTheme } from "../../lib/theme";
import type { AppConfig, Approval, Attachment, ContextUsage, FileHit, Health, Hunk, Item, SessionTrace, SkillInfo, SpillBlob, Thread } from "../../lib/protocol";

const emptyHealth: Health = { ok: false, harness: "", model: "", version: "", isolated: false, budgetUsd: 0, usageUsd: 0, workspaceReady: false };
export const emptyCfg: AppConfig = {
  provider: "openai",
  model: "",
  baseUrl: "",
  workspace: "",
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
};
const emptyCtx: ContextUsage = { tokens: 0, budget: 0, window: 0, prefixTokens: 0, dynamicTokens: 0, schemaTokens: 0, providerPrompt: 0, note: "", layers: [], elided: 0 };

function runningMap(ids: string[], prev: Record<string, boolean> = {}): Record<string, boolean> {
  const next: Record<string, boolean> = {};
  for (const id of Object.keys(prev)) next[id] = false;
  for (const id of ids) next[id] = true;
  return next;
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
    if (v === "diff" || v === "files" || v === "trace") return v;
  } catch {
    /* ignore */
  }
  return null;
}

export function useWorkstation() {
  const copy = useCopy();
  const { setPref } = useTheme();
  const lab = useUI((s) => s.lab);
  const setLab = useUI((s) => s.setLab);
  const surface = useUI((s) => s.surface);
  const openSettings = useUI((s) => s.openSettings);
  const closeSettings = useUI((s) => s.closeSettings);
  const inspector = useUI((s) => s.inspector);
  const setInspector = useUI((s) => s.setInspector);
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
  const [active, setActive] = useState<Thread | null>(null);
  const [items, setItems] = useState<Item[]>([]);
  const itemsAcc = useRef<Item[]>([]);
  const [queued, setQueued] = useState(0);
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [running, setRunning] = useState<Record<string, boolean>>({});
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
  const [harborKind, setHarborKind] = useState<"suite" | "safety" | "tb" | "bon" | "models">("suite");
  const [evolve, setEvolve] = useState<any>(null);
  const [playbook, setPlaybook] = useState<any>(null);
  const [tree, setTree] = useState<any[]>([]);
  const [labBusy, setLabBusy] = useState<string | null>(null);
  const [evolveK, setEvolveK] = useState(3);
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
  const [logs, setLogs] = useState<any>(null);
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

  const syncRunning = useCallback(async () => {
    try {
      const ids = await api.runningIDs();
      setRunning((prev) => runningMap(ids, prev));
    } catch {
      /* ignore */
    }
  }, []);

  const refresh = useCallback(async () => {
    try {
      const [h, c, list, hs, pl] = await Promise.all([
        api.health(),
        api.getConfig(),
        api.listSessions(),
        api.harness().catch(() => ({})),
        api.plugins().catch(() => ({})),
      ]);
      setHealth(h);
      setSavedCfg(c);
      setThreads(list);
      setHarness(hs);
      setPlugins(pl);
      setActive((cur) => {
        if (cur && list.some((t) => t.id === cur.id)) return list.find((t) => t.id === cur.id) || cur;
        return list[0] || null;
      });
      setBooted(true);
      try { setPlaybook(await api.playbook()); } catch { /* optional */ }
      try { setTree(await api.archive()); } catch { /* optional */ }
      try { setSkills(await api.listSkills()); } catch { /* optional */ }
      try { setVault(await api.keyStatus()); } catch { /* optional */ }
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
    if (savedCfg.theme === "dark" || savedCfg.theme === "light" || savedCfg.theme === "system") {
      setPref(savedCfg.theme);
    }
  }, [savedCfg.locale, savedCfg.uiScale, savedCfg.theme, setPref]);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => subscribeSessions(refresh), [refresh]);
  useEffect(() => {
    if (surface !== "settings") return;
    api.logs().then(setLogs).catch(() => {});
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
        }));
        offs.push(Events.On("yoyo:command", (cmd: any) => handlers.current.command(String(cmd || ""))));
        offs.push(Events.On("yoyo:quit", () => handlers.current.quit()));
        offs.push(Events.On("yoyo:focus", (session: any) => handlers.current.focus(typeof session === "string" ? session : "")));
        offs.push(Events.On("yoyo:doctor", (info: any) => { setDoctor(info || {}); openSettings("logs", "logs-doctor"); }));
        offs.push(Events.On("yoyo:logs", (info: any) => { setLogs(info || {}); openSettings("logs", "logs-journal"); }));
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
    if (item.type === "turn_end" || item.type === "error") {
      const id = item.sessionId;
      if (id) setRunning((m) => ({ ...m, [id]: false }));
    }
    if (item.type === "approval") {
      api.approvals().then(setApprovals).catch(() => {});
    }
    if (item.type === "turn_end" || item.type === "approval" || item.type === "error") {
      pushNotice({
        id: item.key || `${item.type}-${Date.now()}`,
        title: item.type === "approval" ? copy.app.approval : item.type === "error" ? copy.app.errorNotice : copy.app.turnFinished,
        body: item.type === "error" ? shortError(item, copy.transcript) : (item.text || item.type),
        ts: item.ts,
        sessionId: item.sessionId,
      });
    }
  }), [pushNotice]);

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
            note: str(p.note, prev.note),
            layers: Array.isArray(p.layers) ? p.layers.map(String) : prev.layers,
            elided: num(p.elided, prev.elided),
          }));
        }
        if (item.type === "approval") {
          api.approvals().then(setApprovals).catch(() => {});
        }
        if (item.type === "turn_end" || item.type === "error") {
          setRunning((m) => ({ ...m, [activeId]: false }));
          api.contextUsage(activeId).then(setCtx).catch(() => {});
          api.queueList(activeId).then((q) => setQueued(q.length)).catch(() => setQueued(0));
        }
      },
      (seed) => {
        itemsAcc.current = mergePendingUsers(seed, itemsAcc.current);
        setItems(itemsAcc.current.slice());
      },
    );
    api.approvals().then(setApprovals).catch(() => {});
    refreshCtx();
    api.running(activeId).then((live) => setRunning((m) => ({ ...m, [activeId]: live }))).catch(() => {});
    return unsub;
  }, [activeId, refreshCtx]);

  useEffect(() => {
    if (!anyRun) return;
    const t = window.setInterval(async () => {
      await syncRunning();
      if (activeId) {
        const [live, offers] = await Promise.all([
          api.running(activeId).catch(() => false),
          api.approvals().catch(() => [] as Approval[]),
        ]);
        setRunning((m) => ({ ...m, [activeId]: live }));
        setApprovals(offers);
        api.contextUsage(activeId).then(setCtx).catch(() => {});
      }
    }, 1500);
    return () => window.clearInterval(t);
  }, [anyRun, activeId, syncRunning]);

  async function ensureThread(): Promise<Thread> {
    if (active) return active;
    const t = await api.createSession(savedCfg.workspace);
    setThreads((prev) => [t, ...prev]);
    setActive(t);
    return t;
  }

  async function onSend(opts?: { steer?: boolean; attachments?: Attachment[] }) {
    const text = (useUI.getState().drafts[draftKey] || "").trim();
    if (!text) return;
    if (!workspaceReady(health.workspaceReady, savedCfg.workspace)) {
      toast.message(copy.app.setupFirst);
      openSettings("general", "general-basics");
      return;
    }
    setErr("");
    try {
      const t = await ensureThread();
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
      setRunning((m) => ({ ...m, [t.id]: true }));
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
      if (activeId) setRunning((s) => ({ ...s, [activeId]: false }));
      fail(e);
    }
  }

  async function onRetryLast() {
    if (!activeId || running[activeId]) return;
    const hasTurn = itemsAcc.current.some((it) => (it.type === "user" && it.source !== "steer") || it.type === "assistant");
    if (!hasTurn) return;
    setErr("");
    setRunning((m) => ({ ...m, [activeId]: true }));
    try {
      await api.retry(activeId);
      itemsAcc.current = dropTrailingErrors(itemsAcc.current);
      setItems(itemsAcc.current.slice());
    } catch (e) {
      setRunning((s) => ({ ...s, [activeId]: false }));
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
      const note = await api.compactSession(activeId);
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
    setRunning((m) => ({ ...m, [activeId]: false }));
  }

  async function onResolve(id: string, decision: string) {
    await api.resolveApproval(id, decision);
    setApprovals((prev) => prev.filter((a) => a.id !== id));
  }

  async function refreshDiff() {
    try {
      const h = await api.workspaceHunks(savedCfg.workspace || active?.workspace || "");
      setDiff(h.diff);
      setHunks(h.hunks);
      setHunkSel({});
      if (h.hunks.length) {
        setInspector(true);
        setInspTab("diff");
      }
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
    const workspace = savedCfg.workspace || active?.workspace || "";
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

  async function onNew() {
    const t = await api.createSession(savedCfg.workspace);
    setThreads((prev) => [t, ...prev]);
    setActive(t);
    setLab("agent");
    itemsAcc.current = [];
    setItems([]);
    setQueued(0);
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
        setInspector((v) => !v);
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
        if (useUI.getState().surface === "settings") {
          closeSettings();
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
  }, [savedCfg.keymap, approvals, openSettings, closeSettings, setInspector, setPalette, setLab]);

  handlers.current.slash = onSlash;
  handlers.current.command = (c) => {
    if (c === "review") setInspector((v) => !v);
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
    if (t) {
      setActive(t);
      setLab("agent");
    }
  };

  return {
    copy, lab, setLab, surface, openSettings, closeSettings, inspector, setInspector,
    palette, setPalette, query, setQuery, inspTab, setInspTab, diffMode, setDiffMode,
    sidebarCollapsed, setSidebarCollapsed, sidebarHover, setSidebarHover,
    notices, noticesOpen, setNoticesOpen, clearNotices, renameTick,
    health, savedCfg, setSavedCfg, threads, active, setActive, items, approvals, running, queued, ctx, trace, err, setErr,
    diff, hunks, hunkSel, setHunkSel, harness, plugins, evalReport, setEvalReport, bestReport, setBestReport, harborErr, setHarborErr, harborKind, setHarborKind,
    evolve, setEvolve, playbook, setPlaybook, tree, setTree, labBusy, setLabBusy, evolveK, setEvolveK, bonModels, setBonModels, diffA, setDiffA, diffB, setDiffB, diffOut, setDiffOut,
    booted, showArchived, setShowArchived, aboutOpen, setAboutOpen, aboutInfo, setAboutInfo, pendingDelete, setPendingDelete,
    files, setFiles, skills, logs, setLogs, doctor, vault, pendingQuit, setPendingQuit, setThreads,
    activeId, draftKey, threadRunning, anyRun, needsSetup,
    fail, refresh, onSend, onRetryLast, onSlash, onStop, onResolve, refreshDiff, applySelected, onNew, patchConfig, requestQuit,
    refreshTrace, loadSpill, refreshCtx,
  };
}
