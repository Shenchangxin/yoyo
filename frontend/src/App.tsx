import { useCallback, useEffect, useRef, useState } from "react";
import { Group, Panel } from "react-resizable-panels";
import { toast } from "sonner";
import * as api from "./lib/client";
import { mergeItem, subscribeItems, subscribeSession, subscribeSessions } from "./lib/stream";
import { readLayout, writeLayout, type ShellLayout } from "./lib/layout";
import { useMedia } from "./lib/media";
import { workspaceReady } from "./lib/workspace";
import { applyLocale, useCopy } from "./lib/i18n";
import { mergeKeymap, matchKey } from "./lib/keymap";
import { useUI } from "./lib/store";
import type { AppConfig, Approval, Attachment, ContextUsage, FileHit, Health, Hunk, Item, SkillInfo, Thread } from "./lib/protocol";
import { WindowChrome } from "./features/WindowChrome";
import { CommandPalette } from "./features/CommandPalette";
import { ResizeHandle } from "./features/ResizeHandle";
import { Titlebar } from "./features/Titlebar";
import { ThreadRail } from "./features/ThreadRail";
import { Transcript } from "./features/Transcript";
import { Composer } from "./features/Composer";
import { Inspector } from "./features/Inspector";
import { ChatDock } from "./features/ChatDock";
import { FirstRun } from "./features/FirstRun";
import { About } from "./features/About";
import { ConfirmDialog } from "./features/ConfirmDialog";
import { BootSkeleton } from "./features/BootSkeleton";
import { HarborLab } from "./features/labs/HarborLab";
import { EvolveLab } from "./features/labs/EvolveLab";
import { HarnessLab } from "./features/labs/HarnessLab";
import { ControlLab } from "./features/labs/ControlLab";

const emptyHealth: Health = { ok: false, harness: "", model: "", version: "", isolated: false, budgetUsd: 0, usageUsd: 0, workspaceReady: false };
const emptyCfg: AppConfig = { provider: "openai", model: "", baseUrl: "", workspace: "", autoAllow: false, maxBudgetUsd: 0, usdPerMtok: 0, models: [], closeToTray: false, updateUrl: "", keymap: {}, locale: "" };
const emptyCtx: ContextUsage = { tokens: 0, budget: 0, note: "", layers: [], elided: 0 };

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

export default function App() {
  const copy = useCopy();
  const lab = useUI((s) => s.lab);
  const setLab = useUI((s) => s.setLab);
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
  const setupDismissed = useUI((s) => s.setupDismissed);
  const setSetupDismissed = useUI((s) => s.setSetupDismissed);

  const [health, setHealth] = useState<Health>(emptyHealth);
  const [savedCfg, setSavedCfg] = useState<AppConfig>(emptyCfg);
  const [threads, setThreads] = useState<Thread[]>([]);
  const [active, setActive] = useState<Thread | null>(null);
  const [items, setItems] = useState<Item[]>([]);
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [running, setRunning] = useState<Record<string, boolean>>({});
  const [ctx, setCtx] = useState<ContextUsage>(emptyCtx);
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
  const [layout, setLayout] = useState<ShellLayout>(readLayout);
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
  const narrow = useMedia("(max-width: 1099px)");
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
  const three = lab === "agent" && inspector && !narrow;
  const dock = lab !== "agent";

  const fail = (e: unknown) => {
    const m = api.errMessage(e);
    setErr(m);
    toast.error(m);
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

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => {
    if (savedCfg.locale) applyLocale(savedCfg.locale);
  }, [savedCfg.locale]);
  useEffect(() => subscribeSessions(refresh), [refresh]);
  useEffect(() => {
    if (lab !== "control") return;
    api.logs().then(setLogs).catch(() => {});
    api.doctor().then(setDoctor).catch(() => {});
  }, [lab]);
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
        offs.push(Events.On("yoyo:doctor", (info: any) => { setDoctor(info || {}); setLab("control"); }));
      } catch {
        /* browser */
      }
    })();
    return () => {
      alive = false;
      offs.forEach((o) => o());
    };
  }, []);

  useEffect(() => subscribeItems((item) => {
    if (item.type === "turn_end" || item.type === "error") {
      const id = item.sessionId;
      if (id) setRunning((m) => ({ ...m, [id]: false }));
    }
    if (item.type === "approval") {
      api.approvals().then(setApprovals).catch(() => {});
    }
  }), []);

  useEffect(() => {
    if (!activeId) {
      setItems([]);
      return;
    }
    setItems([]);
    let acc: Item[] = [];
    const unsub = subscribeSession(activeId, (item) => {
      acc = mergeItem(acc, item);
      setItems(acc.slice());
      if (item.type === "approval") {
        api.approvals().then(setApprovals).catch(() => {});
        setInspTab("approvals");
        setInspector(true);
      }
      if (item.type === "turn_end" || item.type === "error") {
        setRunning((m) => ({ ...m, [activeId]: false }));
        api.contextUsage(activeId).then(setCtx).catch(() => {});
      }
    });
    api.approvals().then(setApprovals).catch(() => {});
    api.contextUsage(activeId).then(setCtx).catch(() => {});
    api.running(activeId).then((live) => setRunning((m) => ({ ...m, [activeId]: live }))).catch(() => {});
    return unsub;
  }, [activeId, setInspTab, setInspector]);

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
        if (!live) api.contextUsage(activeId).then(setCtx).catch(() => {});
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
      setLab("control");
      return;
    }
    setErr("");
    try {
      const t = await ensureThread();
      if (opts?.steer && running[t.id]) {
        await api.steer(t.id, text);
        useUI.getState().patchDrafts({ [t.id]: "", _new: "" });
        toast.success(copy.app.steered);
        return;
      }
      const wasRunning = !!running[t.id];
      useUI.getState().patchDrafts({ [t.id]: "", _new: "" });
      setRunning((m) => ({ ...m, [t.id]: true }));
      setItems((prev) => (t.id === activeId ? [...prev, localUser(t.id, text)] : prev));
      const res = await api.send(t.id, text, { plan: useUI.getState().plan, attachments: opts?.attachments });
      if (wasRunning || res.queued) toast.message(copy.app.queued);
    } catch (e) {
      const m = api.errMessage(e);
      if (!m.includes("already running") && !m.includes("queued")) {
        if (activeId) setRunning((s) => ({ ...s, [activeId]: false }));
      }
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
      if (!title || !activeId) return;
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
      return;
    }
    if (raw === "model") {
      if (!rest.trim()) return;
      const t = await api.setSessionModel(activeId, rest.trim());
      setActive(t);
      await refresh();
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
    setItems([]);
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
        setLab("control");
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
  }, [savedCfg.keymap, approvals, setLab, setInspector, setPalette]);

  const composer = (
    <Composer
      draftKey={draftKey}
      running={threadRunning}
      disabled={needsSetup}
      disabledReason={copy.composer.disabled}
      model={active?.model || savedCfg.model || health.model}
      models={savedCfg.models.length ? savedCfg.models : [savedCfg.model || health.model].filter(Boolean)}
      files={files}
      skills={skills}
      onSearchFiles={(q) => { void api.searchFiles(savedCfg.workspace, q).then(setFiles).catch(() => {}); }}
      onSend={onSend}
      onStop={onStop}
      onSlash={(cmd, rest) => { void onSlash(cmd, rest); }}
      onModel={async (model) => {
        if (!activeId) return;
        const t = await api.setSessionModel(activeId, model);
        setActive(t);
      }}
      onPickFiles={async () => {
        const paths = await api.pickFiles();
        return paths.map((path) => ({ path, name: path.replace(/^.*[\\/]/, "") }));
      }}
    />
  );

  const inspect = (
    <Inspector
      tab={inspTab}
      onTab={setInspTab}
      diff={diff}
      hunks={hunks}
      selected={hunkSel}
      mode={diffMode}
      onMode={setDiffMode}
      onToggle={(id) => setHunkSel((s) => ({ ...s, [id]: !s[id] }))}
      onApply={applySelected}
      onRefreshDiff={refreshDiff}
      ctx={ctx}
      approvals={approvals}
      onResolve={onResolve}
      onQuote={(text) => {
        const cur = useUI.getState().drafts[draftKey] || "";
        useUI.getState().setDraft(draftKey, cur ? `${cur}\n${text}` : text);
      }}
    />
  );

  const agentPane = (
    <section className="relative flex h-full min-h-0 flex-col bg-background">
      <Titlebar
        health={health}
        ctx={ctx}
        workspace={savedCfg.workspace}
        inspector={inspector}
        title={active?.title || "New chat"}
        runningCount={Object.values(running).filter(Boolean).length}
        runningThreads={threads.filter((t) => running[t.id])}
        onSelectRunning={(t) => { setActive(t); setLab("agent"); }}
        onToggleInspector={() => setInspector((v) => !v)}
        onControl={() => setLab("control")}
        onRename={async (title) => {
          if (!activeId) return;
          await api.renameSession(activeId, title);
          await refresh();
        }}
        onFork={async () => {
          if (!activeId) return;
          const t = await api.forkSession(activeId);
          setThreads((prev) => [t, ...prev]);
          setActive(t);
        }}
        onDiff={refreshDiff}
      />
      <Transcript
        items={items}
        approvals={approvals}
        running={threadRunning}
        needsSetup={needsSetup}
        onResolve={onResolve}
        onPrompt={(text) => useUI.getState().setDraft(draftKey, text)}
        onSetup={() => setLab("control")}
        onOpenReview={() => {
          setInspector(true);
          setInspTab("diff");
          void refreshDiff();
        }}
      />
      {composer}
      {narrow && inspector && lab === "agent" ? (
        <div className="absolute bottom-0 right-0 top-11 z-20 w-[min(360px,92%)] border-l border-border bg-sidebar shadow-xl">
          {inspect}
        </div>
      ) : null}
    </section>
  );

  const labPane =
    lab === "harbor" ? (
      <HarborLab
        busy={labBusy}
        error={harborErr}
        lastKind={harborKind}
        report={evalReport}
        best={bestReport}
        models={bonModels}
        onModels={setBonModels}
        onRun={async (kind) => {
          setHarborKind(kind);
          setHarborErr("");
          setLabBusy("Harbor running…");
          try {
            if (kind === "suite") { setEvalReport(await api.runEval()); setBestReport(null); }
            if (kind === "safety") { setEvalReport(await api.runEvalSafety()); setBestReport(null); }
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
          }
          finally { setLabBusy(null); }
        }}
      />
    ) : lab === "evolve" ? (
      <EvolveLab
        busy={!!labBusy}
        k={evolveK}
        onK={setEvolveK}
        evolve={evolve}
        playbook={playbook}
        archive={tree}
        onRun={async () => {
          setLabBusy("Evolve running…");
          try {
            setEvolve(await api.evolve(evolveK));
            setTree(await api.archive());
            await refresh();
          } catch (e) { fail(e); }
          finally { setLabBusy(null); }
        }}
        onRate={async (id, helpful) => {
          setPlaybook(await api.ratePlaybook(id, helpful));
          await refresh();
        }}
      />
    ) : lab === "harness" ? (
      <HarnessLab
        harness={harness}
        diffA={diffA}
        diffB={diffB}
        diffOut={diffOut}
        onA={setDiffA}
        onB={setDiffB}
        onCompare={async () => {
          try { setDiffOut(await api.diff(diffA || health.harness, diffB)); }
          catch (e) { fail(e); }
        }}
        onCheckout={async (hash, l3) => {
          await api.checkout(hash, !!l3);
          await refresh();
          toast.success(copy.app.checkedOut);
        }}
        onRollback={async () => { await api.rollback(); await refresh(); toast.success(copy.app.rolledBack); }}
      />
    ) : lab === "control" ? (
      <ControlLab
        saved={savedCfg}
        plugins={plugins}
        logs={logs}
        doctor={doctor}
        vault={vault}
        onSave={async (next, key) => {
          if (key.trim()) await api.setAPIKey(key.trim());
          await api.setConfig(next);
          await refresh();
          setSavedCfg(next);
          toast.success(copy.app.controlSaved);
        }}
        onUpdate={async () => {
          try { await api.applyUpdate(); await refresh(); toast.success(copy.app.stagedUpdate); }
          catch (e) { fail(e); }
        }}
        onCheckUpdate={async () => {
          try {
            const u = await api.checkUpdate();
            if (u?.error) toast.error(String(u.error));
            else if (u?.downloaded) toast.success(`downloaded ${u.size || 0} bytes`);
            else if (u?.newer) toast.message("newer build verified — apply staged");
            else toast.message(u?.staged ? `staged ${u.size || 0} bytes` : "no staged update");
          } catch (e) { fail(e); }
        }}
        onUnload={async (name) => { await api.unloadFiber(name); await refresh(); }}
        onBrowse={async () => api.pickFolder()}
        onStartMcp={async (name, command, args) => { await api.startMCP(name, command, args); await refresh(); }}
        onStopMcp={async (name) => { await api.stopMCP(name); await refresh(); }}
      />
    ) : null;

  const showWizard = booted && needsSetup && !setupDismissed;
  const showSetup = booted && needsSetup && setupDismissed && lab !== "control";

  handlers.current.slash = onSlash;
  handlers.current.command = (c) => {
    if (c === "review") setInspector((v) => !v);
    else if (c === "palette") setPalette(true);
    else if (c === "control") setLab("control");
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

  return (
    <div className="relative flex h-full min-h-0 flex-col overflow-hidden bg-background text-foreground">
      {!booted ? (
        <div className="pointer-events-none absolute inset-0 z-30">
          <BootSkeleton />
        </div>
      ) : null}
      <WindowChrome connected={health.ok} isolated={health.isolated} onPalette={() => setPalette(true)} />
      {err ? (
        <div className="flex items-center gap-3 border-b border-danger/30 bg-danger/10 px-4 py-2 text-sm text-danger">
          {err}
          <button type="button" className="ml-auto text-xs underline" onClick={() => setErr("")}>{copy.app.dismiss}</button>
        </div>
      ) : null}
      {showSetup ? (
        <div className="flex items-center gap-3 border-b border-border bg-panel px-4 py-2 text-sm text-muted">
          {copy.app.setupBanner}
          <button type="button" className="rounded-full bg-lift px-3 py-1 text-xs text-foreground" onClick={() => setLab("control")}>
            {copy.app.openControl}
          </button>
        </div>
      ) : null}
      <Group
        key={three ? "shell-3" : dock ? "shell-dock" : "shell-2"}
        className="min-h-0 flex-1"
        style={{ height: "auto", minHeight: 0 }}
        orientation="horizontal"
        defaultLayout={three
          ? { rail: layout.rail, main: Math.max(30, 100 - layout.rail - layout.inspect), inspect: layout.inspect }
          : dock
            ? { rail: layout.rail, main: Math.max(36, 100 - layout.rail - 24), inspect: 24 }
            : { rail: layout.rail, main: Math.max(40, 100 - layout.rail) }}
        onLayoutChanged={(next, meta) => {
          if (!meta.isUserInteraction) return;
          setLayout((prev) => {
            if (typeof next.inspect === "number") {
              return writeLayout({ rail: next.rail, main: next.main, inspect: next.inspect });
            }
            const rail = next.rail ?? prev.rail;
            return writeLayout({ rail, main: Math.max(30, 100 - rail - prev.inspect) });
          });
        }}
      >
        <Panel id="rail" minSize="14" maxSize="34" className="h-full min-h-0">
          <ThreadRail
            threads={threads}
            activeId={activeId}
            query={query}
            running={running}
            lab={lab}
            showArchived={showArchived}
            onQuery={setQuery}
            onSelect={setActive}
            onNew={onNew}
            onLab={setLab}
            onControl={() => setLab("control")}
            onToggleArchived={() => setShowArchived((v) => !v)}
            onPin={async (t, pinned) => { await api.pinSession(t.id, pinned); await refresh(); toast.success(copy.app.pinned); }}
            onArchive={async (t, archived) => { await api.archiveSession(t.id, archived); await refresh(); toast.success(copy.app.archived); }}
            onDelete={(t) => setPendingDelete(t)}
            onRename={async (t, title) => {
              await api.renameSession(t.id, title);
              await refresh();
            }}
          />
        </Panel>
        <ResizeHandle />
        <Panel id="main" minSize="30" className="h-full min-h-0">
          {lab === "agent" ? agentPane : <div className="h-full min-h-0 bg-background">{labPane}</div>}
        </Panel>
        {three || dock ? <ResizeHandle /> : null}
        {three ? (
          <Panel id="inspect" minSize="18" maxSize="42" className="h-full min-h-0">
            {inspect}
          </Panel>
        ) : null}
        {dock ? (
          <Panel id="inspect" minSize="18" maxSize="40" className="h-full min-h-0">
            <ChatDock
              items={items}
              approvals={approvals}
              running={threadRunning}
              draftKey={draftKey}
              disabled={needsSetup}
              disabledReason={copy.composer.disabled}
              model={active?.model || savedCfg.model || health.model}
              models={savedCfg.models}
              onSend={onSend}
              onStop={onStop}
              onResolve={onResolve}
              onOpenAgent={() => setLab("agent")}
              onSlash={(cmd, rest) => { void onSlash(cmd, rest); }}
              onModel={async (model) => {
                if (!activeId) return;
                const t = await api.setSessionModel(activeId, model);
                setActive(t);
              }}
            />
          </Panel>
        ) : null}
      </Group>
      <CommandPalette
        open={palette}
        threads={threads}
        onClose={() => setPalette(false)}
        onNew={onNew}
        onLab={setLab}
        onSelectThread={setActive}
        onDiff={refreshDiff}
        onAbout={async () => { setAboutInfo(await api.about().catch(() => ({}))); setAboutOpen(true); }}
        onQuit={requestQuit}
      />
      <FirstRun
        open={showWizard}
        cfg={savedCfg}
        onBrowse={async () => api.pickFolder()}
        onSkip={() => setSetupDismissed(true)}
        onFinish={async (values) => {
          if (values.apiKey.trim()) await api.setAPIKey(values.apiKey.trim());
          const next = { ...savedCfg, workspace: values.workspace, model: values.model };
          await api.setConfig(next);
          await refresh();
          setSavedCfg(next);
          setSetupDismissed(true);
          toast.success(copy.app.controlSaved);
        }}
      />
      <About open={aboutOpen} info={aboutInfo} onClose={() => setAboutOpen(false)} />
      <ConfirmDialog
        open={!!pendingDelete}
        title={copy.rail.delete}
        body={pendingDelete?.title || pendingDelete?.id || ""}
        danger
        confirmLabel={copy.rail.delete}
        onCancel={() => setPendingDelete(null)}
        onConfirm={async () => {
          if (!pendingDelete) return;
          await api.deleteSession(pendingDelete.id);
          setPendingDelete(null);
          await refresh();
          toast.success(copy.app.deleted);
        }}
      />
      <ConfirmDialog
        open={pendingQuit}
        title={copy.control.quitRunning}
        body={copy.control.quitRunning}
        danger
        confirmLabel={copy.app.quit}
        onCancel={() => setPendingQuit(false)}
        onConfirm={() => {
          setPendingQuit(false);
          void api.quit();
        }}
      />
    </div>
  );
}
