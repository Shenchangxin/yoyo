import { useState } from "react";
import { Group, Panel } from "react-resizable-panels";
import { toast } from "sonner";
import { cn } from "./lib/utils";
import { THREAD_COL, THREAD_GUTTER } from "./lib/thread";
import * as api from "./lib/client";
import { writeClipboard } from "./lib/clipboard";
import { readLayout, writeLayout } from "./lib/layout";
import { useMedia } from "./lib/media";
import { useUI } from "./lib/store";
import { AppFrame, MainColumn } from "./features/shell/AppFrame";
import { PageHeader } from "./features/shell/PageHeader";
import { CommandPalette } from "./features/CommandPalette";
import { ResizeHandle } from "./features/ResizeHandle";
import { Titlebar } from "./features/Titlebar";
import { ThreadRail } from "./features/ThreadRail";
import { Transcript } from "./features/Transcript";
import { HomeStage } from "./features/HomeStage";
import { Composer } from "./features/Composer";
import { Inspector } from "./features/Inspector";
import { ChatDock } from "./features/ChatDock";
import { About } from "./features/About";
import { ConfirmDialog } from "./features/ConfirmDialog";
import { BootSkeleton } from "./features/BootSkeleton";
import { HarborLab } from "./features/labs/HarborLab";
import { EvolveLab } from "./features/labs/EvolveLab";
import { HarnessLab } from "./features/labs/HarnessLab";
import { HarnessWorkspace } from "./features/harness/HarnessWorkspace";
import { canaryDirty, parseHarnessRefs, shortHash, stagingDirty } from "./lib/harness-refs";
import { SettingsPage } from "./features/settings/SettingsPage";
import { SkillsWorkspace } from "./features/skills/SkillsWorkspace";
import { VideoWorkshop } from "./features/video/VideoWorkshop";
import { CanvasStudio } from "./features/video/CanvasStudio";
import { useCanvasHost } from "./features/video/canvas-host/session";
import { useSettingsHash } from "./features/settings/useSettingsHash";
import { useWorkstation } from "./features/workstation/useWorkstation";
import { readPopoutId } from "./lib/popout";
import { Sheet, SheetContent, SheetDescription, SheetTitle } from "./components/ui/sheet";
import { Button } from "./components/ui/button";
import { Kbd } from "./components/ui/kbd";
import { PanelLeft } from "lucide-react";
import { DEFAULT_KEYMAP, displayShortcut } from "./lib/keymap";
import { isMac } from "./lib/chrome";
import { displayTitle } from "./lib/display-title";
import { recentWorkspaces } from "./lib/workspace";
import type { AuthMode, Thread } from "./lib/protocol";
import { latestTaskPlan } from "./lib/plan";

function patchThread(list: Thread[], id: string, patch: Partial<Thread>): Thread[] {
  const next = list.map((x) => (x.id === id ? { ...x, ...patch } : x));
  next.sort((a, b) => Number(!!b.pinned) - Number(!!a.pinned));
  return next;
}

export default function App() {
  const ws = useWorkstation();
  useSettingsHash();
  const copy = ws.copy;
  const popoutId = readPopoutId();
  const popout = !!popoutId;
  const [layout, setLayout] = useState(readLayout);
  const [reviewFile, setReviewFile] = useState("");
  const sheetInspect = useMedia("(max-width: 1099px)");
  const railNarrow = useMedia("(max-width: 799px)");
  const settings = !popout && ws.surface === "settings";
  const skills = !popout && ws.surface === "skills";
  const videoing = !popout && ws.surface === "video";
  const harnessing = !popout && ws.surface === "harness";
  const videoMode = useUI((s) => s.videoMode);
  const canvasing = videoing && videoMode === "canvas";
  const canvasTitle = useCanvasHost((s) => s.title);
  const agent = popout || ws.surface === "agent";
  const home = !canvasing && (agent || videoing) && ws.items.length === 0 && !ws.threadRunning && ws.approvals.length === 0;
  const three = agent && !popout && ws.inspector && !sheetInspect;
  const dock = (harnessing || canvasing) && ws.chatDock && !sheetInspect;
  const inspectOpen = three || dock;
  const showRail = !popout && !ws.sidebarCollapsed && !railNarrow;
  const overlayRail = !popout && !showRail && ws.sidebarHover;
  const mac = isMac();
  const stagePct = showRail ? Math.max(66, 100 - layout.rail) : 100;
  const innerInspect = Math.min(46, Math.max(22, (layout.inspect / stagePct) * 100));
  const refs = parseHarnessRefs(ws.harness, ws.health.harness);
  const sheetRight = sheetInspect && ((agent && ws.inspector) || ((harnessing || canvasing) && ws.chatDock));

  const sessionWs = ws.active?.workspace || ws.savedCfg.workspace;
  const toolRoot = ws.active?.toolRoot || sessionWs;
  const applySessionWorkspace = async (path: string) => {
    if (!path) return;
    if (!ws.activeId) {
      const t = await api.createSession(path, videoing ? "video" : "agent");
      ws.setActive(t);
      ws.setThreads((list) => [t, ...list.filter((x) => x.id !== t.id)]);
      return;
    }
    const t = await api.setSessionWorkspace(ws.activeId, path);
    ws.setActive(t);
    ws.setThreads((list) => patchThread(list, t.id, t));
  };
  const bindComposer = {
    files: ws.files,
    skills: ws.skills,
    authMode: ws.active?.authMode || "full",
    workspace: sessionWs,
    workspaces: recentWorkspaces([sessionWs, ws.savedCfg.workspace, ...ws.threads.map((t) => t.workspace)]),
    isolate: !!ws.active?.isolate,
    onWorkspace: (path: string) => { void applySessionWorkspace(path); },
    onBrowseWorkspace: async () => {
      const p = await api.pickFolder();
      if (p) await applySessionWorkspace(p);
    },
    onIsolate: async (isolate: boolean) => {
      if (!ws.activeId) return;
      const t = await api.setSessionIsolate(ws.activeId, isolate);
      ws.setActive(t);
      ws.setThreads((list) => patchThread(list, t.id, t));
    },
    onSearchFiles: (q: string) => { void api.searchFiles(toolRoot, q).then(ws.setFiles).catch(() => {}); },
    onPickFiles: async () => {
      const paths = await api.pickFiles();
      return paths.map((path) => ({ path, name: path.replace(/^.*[\\/]/, "") }));
    },
    onAuthMode: async (mode: AuthMode) => {
      if (!ws.activeId) return;
      const t = await api.setSessionAuthMode(ws.activeId, mode);
      ws.setActive(t);
      ws.setThreads((list) => patchThread(list, t.id, t));
    },
    onClipboard: async () => api.clipboardRead(),
    onScreenshot: async () => api.captureScreenshot(),
    taskPlan: latestTaskPlan(ws.items),
  };

  const composer = (
    <Composer
      draftKey={ws.draftKey}
      running={ws.threadRunning}
      disabled={false}
      disabledReason=""
      model={ws.active?.model || ws.savedCfg.model || ws.health.model}
      models={ws.savedCfg.models}
      provider={ws.savedCfg.provider}
      ctx={ws.ctx}
      queued={ws.queued}
      onSend={ws.onSend}
      onStop={ws.onStop}
      onSlash={(cmd, rest) => { void ws.onSlash(cmd, rest); }}
      onModel={async (model) => {
        if (!ws.activeId) return;
        const t = await api.setSessionModel(ws.activeId, model);
        ws.setActive(t);
        ws.refreshCtx();
      }}
      flush
      hero={home}
      {...bindComposer}
    />
  );

  const inspect = (
    <Inspector
      tab={ws.inspTab}
      onTab={ws.setInspTab}
      diff={ws.diff}
      hunks={ws.hunks}
      selected={ws.hunkSel}
      mode={ws.diffMode}
      onMode={ws.setDiffMode}
      onToggle={(id) => ws.setHunkSel((s) => ({ ...s, [id]: !s[id] }))}
      onSelect={(ids, on) => ws.setHunkSel((s) => {
        const next = { ...s };
        for (const id of ids) next[id] = on;
        return next;
      })}
      onApply={ws.applySelected}
      onRefreshDiff={ws.refreshDiff}
      onQuote={(text) => {
        const cur = useUI.getState().drafts[ws.draftKey] || "";
        useUI.getState().setDraft(ws.draftKey, cur ? `${cur}\n${text}` : text);
      }}
      sessionId={ws.activeId}
      running={ws.threadRunning}
      trace={ws.trace}
      thread={ws.active}
      workspace={sessionWs}
      focusFile={reviewFile}
      onRefreshTrace={() => { void ws.refreshTrace(); }}
      onLoadSpill={ws.loadSpill}
      onResolve={ws.onResolve}
          onOpenThread={(id) => {
            const t = ws.threads.find((x) => x.id === id);
            if (t) ws.openThread(t);
          }}
      onOpenPath={(rel) => {
        const root = ws.active?.workspace || ws.savedCfg.workspace;
        const path = joinWorkspace(root, rel);
        if (path) void api.openInEditor(path);
      }}
    />
  );

  const agentPane = home ? (
    <HomeStage
      onPrompt={(text) => useUI.getState().setDraft(ws.draftKey, text)}
    >
      {composer}
    </HomeStage>
  ) : (
    <section className="@container relative flex h-full min-h-0 min-w-0 flex-col">
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <Transcript
          items={ws.items}
          approvals={ws.approvals}
          running={ws.threadRunning}
          workspace={sessionWs}
          onResolve={ws.onResolve}
          onPrompt={(text) => useUI.getState().setDraft(ws.draftKey, text)}
          onRetry={() => { void ws.onRetryLast(); }}
          onOpenReview={(path) => {
            ws.setInspector(true);
            if (path) {
              setReviewFile(path);
              ws.setInspTab("files");
            } else {
              ws.setInspTab("diff");
            }
            void ws.refreshDiff();
          }}
        />
      </div>
      <div className={cn(THREAD_COL, THREAD_GUTTER)}>
        {composer}
      </div>
    </section>
  );

  const labPane = (
    <HarnessWorkspace
      tab={ws.harnessTab}
      onTab={ws.setHarnessTab}
      harness={ws.harness}
      fallbackActive={ws.health.harness}
      report={ws.evalReport}
      onReveal={ws.revealHarness}
    >
      {{
        prove: (
          <HarborLab
            busy={ws.labBusy}
            error={ws.harborErr}
            lastKind={ws.harborKind}
            report={ws.evalReport}
            best={ws.bestReport}
            models={ws.bonModels}
            onModels={ws.setBonModels}
            onRun={(kind) => { void ws.runHarbor(kind); }}
          />
        ),
        propose: (
          <EvolveLab
            busy={!!ws.labBusy}
            k={ws.evolveK}
            onK={ws.setEvolveK}
            rounds={ws.evolveRounds}
            onRounds={ws.setEvolveRounds}
            sealed={ws.evolveSealed}
            onSealed={ws.setEvolveSealed}
            behavior={ws.evolveBehavior}
            onBehavior={ws.setEvolveBehavior}
            index={ws.evolveIndex}
            onIndex={ws.setEvolveIndex}
            baselines={ws.evolveBaselines}
            onBaselines={ws.setEvolveBaselines}
            maxUsd={ws.evolveMaxUsd}
            onMaxUsd={ws.setEvolveMaxUsd}
            evolve={ws.evolve}
            playbook={ws.playbook}
            archive={ws.tree}
            onRun={() => { void ws.runEvolve(); }}
            onRate={async (id, helpful) => {
              ws.setPlaybook(await api.ratePlaybook(id, helpful));
              await ws.refresh();
            }}
          />
        ),
        promote: (
          <HarnessLab
            harness={ws.harness}
            diffA={ws.diffA}
            diffB={ws.diffB}
            diffOut={ws.diffOut}
            onA={ws.setDiffA}
            onB={ws.setDiffB}
            onCompare={() => { void ws.compareHarness(); }}
            onCheckout={ws.checkoutHarness}
            onRollback={() => { void ws.rollbackHarness(); }}
          />
        ),
      }}
    </HarnessWorkspace>
  );

  const rail = (
    <ThreadRail
      threads={ws.threads}
      activeId={ws.activeId}
      query={ws.query}
      running={ws.running}
      lab={ws.lab}
      surface={ws.surface}
      harness={ws.harness}
      fallbackActive={ws.health.harness}
      showArchived={ws.showArchived}
      notices={ws.notices}
      noticesOpen={ws.noticesOpen}
      connected={ws.health.ok}
      isolated={ws.health.isolated}
      isolationKind={ws.health.isolationKind}
      onQuery={ws.setQuery}
      onSelect={ws.openThread}
      onNew={ws.onNew}
      onNewIn={(path) => { void ws.onNewIn(path); }}
      workspace={ws.savedCfg.workspace}
      onLab={ws.setLab}
      onHarness={() => {
        if (ws.surface === "harness") ws.showConversation();
        else ws.openHarness("overview");
      }}
      onSkills={() => {
        if (ws.surface === "skills") ws.closeSkills();
        else ws.openSkills();
      }}
      onVideo={() => {
        if (ws.surface === "video") ws.closeVideo();
        else ws.openVideo();
      }}
      onSettings={() => {
        if (ws.surface === "settings") ws.closeSettings();
        else ws.openSettings();
      }}
      onCollapse={() => ws.setSidebarCollapsed(true)}
      onToggleArchived={() => ws.setShowArchived((v) => !v)}
      onToggleNotices={() => ws.setNoticesOpen((v) => !v)}
      onClearNotices={ws.clearNotices}
      onNotice={(n) => {
        if (n.sessionId) {
          const t = ws.threads.find((x) => x.id === n.sessionId);
          if (t) ws.openThread(t);
        }
        ws.setNoticesOpen(false);
      }}
      onPin={async (t, pinned) => {
        ws.setThreads((prev) => patchThread(prev, t.id, { pinned }));
        try {
          const next = await api.pinSession(t.id, pinned);
          ws.setThreads((prev) => patchThread(prev, t.id, next));
          toast.success(pinned ? copy.app.pinned : copy.app.unpinned);
        } catch (e) {
          await ws.refresh();
          ws.fail(e);
        }
      }}
      onArchive={async (t, archived) => {
        ws.setThreads((prev) => prev.map((x) => (x.id === t.id ? { ...x, archived } : x)));
        try {
          const next = await api.archiveSession(t.id, archived);
          ws.setThreads((prev) => prev.map((x) => (x.id === t.id ? next : x)));
          toast.success(archived ? copy.app.archived : copy.app.unarchived);
        } catch (e) {
          await ws.refresh();
          ws.fail(e);
        }
      }}
      onDelete={(t) => ws.setPendingDelete(t)}
      onFork={async (t) => {
        try {
          const next = await api.forkSession(t.id);
          ws.setThreads((prev) => [next, ...prev]);
          ws.openThread(next);
          toast.success(copy.app.forked);
        } catch (e) {
          ws.fail(e);
        }
      }}
      onPopOut={(t) => { void api.popOutThread(t.id); }}
      onOpenEditor={(t) => {
        const path = t.workspace || ws.savedCfg.workspace;
        if (path) void api.openInEditor(path);
      }}
      onOpenTerminal={(t) => {
        const path = t.workspace || ws.savedCfg.workspace;
        if (path) void api.openWorkspaceTerminal(path);
      }}
      onRename={async (t, title) => {
        try {
          await api.renameSession(t.id, title);
          ws.setThreads((prev) => prev.map((x) => (x.id === t.id ? { ...x, title } : x)));
          if (ws.active?.id === t.id) ws.setActive({ ...ws.active, title });
        } catch (e) {
          await ws.refresh();
          ws.fail(e);
        }
      }}
    />
  );

  const headerLeft = ws.sidebarCollapsed || railNarrow ? (
    <Button size="icon" variant="ghost" aria-label={copy.rail.expand} onClick={() => { ws.setSidebarCollapsed(false); ws.setSidebarHover(true); }}>
      <PanelLeft />
    </Button>
  ) : null;

  const headerRight = (
    <>
      {harnessing || canvasing ? (
        <Button
          size="sm"
          variant="ghost"
          aria-pressed={ws.chatDock}
          aria-label={copy.dock.toggle}
          className={ws.chatDock ? "bg-lift text-foreground" : undefined}
          onClick={() => ws.setChatDock((v) => !v)}
        >
          {copy.dock.chat}
        </Button>
      ) : null}
      <button
        type="button"
        className="mr-0.5 hidden items-center rounded-lg px-1.5 py-1 text-muted transition-[background-color,color] duration-200 ease-[var(--ease-out)] hover:bg-lift hover:text-foreground sm:inline-flex"
        onClick={() => ws.setPalette(true)}
        aria-label={copy.rail.jump}
        title={copy.rail.jump}
      >
        <Kbd>{displayShortcut(DEFAULT_KEYMAP.palette)}</Kbd>
      </button>
    </>
  );

  const headerTitle = settings
    ? <span className="text-[13px] font-medium">{copy.settings.title}</span>
    : skills
      ? <span className="text-[13px] font-medium">{copy.skills.title}</span>
    : canvasing
      ? <span className="truncate text-[13px] font-medium">{canvasTitle || copy.video.canvas}</span>
    : harnessing
      ? (
        <div className="flex min-w-0 items-center gap-2">
          <span className="text-[13px] font-medium">{copy.rsi.title}</span>
          {refs.active ? (
            <span className="truncate font-mono text-[11px] text-muted" title={refs.active}>{shortHash(refs.active)}</span>
          ) : null}
          {stagingDirty(refs) ? (
            <span className="rounded-md bg-lift px-1.5 py-0.5 text-[10px] text-muted">{copy.rail.stagingDirty}</span>
          ) : canaryDirty(refs) ? (
            <span className="rounded-md bg-lift px-1.5 py-0.5 text-[10px] text-muted">{copy.rail.canaryDirty}</span>
          ) : null}
        </div>
      )
      : (
            <Titlebar
              inspector={ws.inspector}
              hideInspector={videoing}
              hideInbox={videoing}
              title={home ? "" : (ws.active ? displayTitle(ws.active.title, copy.rail.untitled) : copy.rail.newChat)}
              runningCount={Object.values(ws.running).filter(Boolean).length}
              runningThreads={ws.threads.filter((t) => ws.running[t.id])}
              runningStatus={ws.runStatus}
              onSelectRunning={(t) => ws.openThread(t)}
              renameTick={ws.renameTick}
              onToggleInspector={() => ws.setInspector((v) => !v)}
              onRename={async (title) => {
                if (!ws.activeId) return;
                await api.renameSession(ws.activeId, title);
                await ws.refresh();
              }}
            />
          );

  const skillsPane = (
    <SkillsWorkspace
      installed={ws.skills}
      pinned={ws.active?.pinnedSkills || []}
      loaded={ws.active?.loadedSkills || []}
      threadId={ws.activeId}
      onPinSkills={async (names) => {
        if (!ws.activeId) return;
        const t = await api.setSessionPinnedSkills(ws.activeId, names);
        ws.setActive(t);
        ws.setThreads((list) => patchThread(list, t.id, t));
      }}
      onRefreshInstalled={() => { void ws.reloadSkills(); }}
    />
  );

  const workshop = (
    <VideoWorkshop
      sessionId={ws.activeId}
      onNeedSession={() => {
        if (!ws.activeId) void ws.onNew();
      }}
      onClose={() => ws.setVideoBoard(false)}
    />
  );

  const workspace = settings ? (
    <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden bg-background">
      <SettingsSurface ws={ws} />
    </div>
  ) : skills ? (
    <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden bg-background">{skillsPane}</div>
  ) : canvasing ? (
    <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden bg-sidebar" data-testid="video-board">
      <CanvasStudio
        sessionId={ws.activeId}
        onNeedSession={() => {
          if (!ws.activeId) void ws.ensureThread();
        }}
      />
    </div>
  ) : harnessing ? (
    <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden bg-background">{labPane}</div>
  ) : agentPane;

  return (
    <AppFrame
      overlay={(
        <>
          <a href="#main-stage" className="skip-to-content no-drag">{copy.app.skipToContent}</a>
          {!ws.booted ? <div className="absolute inset-0 z-30"><BootSkeleton /></div> : null}
          <CommandPalette
            open={ws.palette}
            threads={ws.threads}
            onClose={() => ws.setPalette(false)}
            onNew={ws.onNew}
            onLab={ws.setLab}
            onOpenHarness={() => ws.openHarness("overview")}
            onSelectThread={ws.openThread}
            onDiff={ws.refreshDiff}
            onAbout={async () => { ws.setAboutInfo(await api.about().catch(() => ({}))); ws.setAboutOpen(true); }}
            onQuit={ws.requestQuit}
            onRunSuite={() => { void ws.runHarbor("suite"); }}
            onRunCycle={() => { void ws.runEvolve(); }}
          />
          <About open={ws.aboutOpen} info={ws.aboutInfo} onClose={() => ws.setAboutOpen(false)} />
          <ConfirmDialog
            open={!!ws.pendingDelete}
            title={copy.rail.delete}
            body={ws.pendingDelete?.title || ws.pendingDelete?.id || ""}
            danger
            confirmLabel={copy.rail.delete}
            onCancel={() => ws.setPendingDelete(null)}
            onConfirm={async () => {
              const t = ws.pendingDelete;
              if (!t) return;
              const rest = ws.threads.filter((x) => x.id !== t.id);
              ws.setPendingDelete(null);
              ws.setThreads(rest);
              if (ws.active?.id === t.id) ws.setActive(rest[0] || null);
              try {
                await api.deleteSession(t.id);
                toast.success(copy.app.deleted);
                await ws.refresh();
              } catch (e) {
                await ws.refresh();
                ws.fail(e);
              }
            }}
          />
          <ConfirmDialog
            open={ws.pendingQuit}
            title={copy.app.quit}
            body={copy.control.quitRunning}
            danger
            confirmLabel={copy.app.quit}
            onCancel={() => ws.setPendingQuit(false)}
            onConfirm={() => {
              ws.setPendingQuit(false);
              void api.quit();
            }}
          />
          <Sheet modal={false} open={sheetRight} onOpenChange={(open) => {
            if (agent) ws.setInspector(open);
            else ws.setChatDock(open);
          }}>
            <SheetContent side="right" overlay={false} className="flex flex-col p-0">
              <SheetTitle className="sr-only">{agent ? copy.review.toggle : copy.dock.toggle}</SheetTitle>
              <SheetDescription className="sr-only">{agent ? copy.review.toggle : copy.dock.toggle}</SheetDescription>
              {agent ? inspect : (
                <ChatDock
                  items={ws.items}
                  approvals={ws.approvals}
                  running={ws.threadRunning}
                  draftKey={ws.draftKey}
                  disabled={false}
                  disabledReason=""
                  model={ws.active?.model || ws.savedCfg.model || ws.health.model}
                  models={ws.savedCfg.models}
                  provider={ws.savedCfg.provider}
                  ctx={ws.ctx}
                  queued={ws.queued}
                  onSend={ws.onSend}
                  onStop={ws.onStop}
                  onResolve={ws.onResolve}
                  onRetry={() => { void ws.onRetryLast(); }}
                  onOpenAgent={() => ws.showConversation()}
                  onSlash={(cmd, rest) => { void ws.onSlash(cmd, rest); }}
                  onModel={async (model) => {
                    if (!ws.activeId) return;
                    const t = await api.setSessionModel(ws.activeId, model);
                    ws.setActive(t);
                    ws.refreshCtx();
                  }}
                  {...bindComposer}
                />
              )}
            </SheetContent>
          </Sheet>
        </>
      )}
    >
      <Group
        key={showRail ? "rail" : "norail"}
        className="h-full min-h-0 min-w-0 flex-1"
        style={{ height: "100%" }}
        orientation="horizontal"
        defaultLayout={showRail
          ? { rail: layout.rail, stage: Math.max(66, 100 - layout.rail) }
          : { stage: 100 }}
        onLayoutChanged={(next, meta) => {
          if (!meta.isUserInteraction || typeof next.rail !== "number") return;
          setLayout((prev) => writeLayout({
            rail: next.rail,
            main: Math.max(30, 100 - next.rail - prev.inspect),
          }));
        }}
      >
        {showRail ? (
          <Panel id="rail" minSize="14" maxSize="34" className="h-full min-h-0">
            {rail}
          </Panel>
        ) : null}
        {showRail ? <ResizeHandle /> : null}
        <Panel id="stage" minSize="42" className="h-full min-h-0 min-w-0">
          <MainColumn>
            <PageHeader left={headerLeft} title={headerTitle} right={headerRight} macPad={mac && !showRail} />
            {ws.err ? (
              <div data-testid="app-error" className="mx-4 mb-1 flex items-start gap-3 rounded-xl bg-danger/10 px-3.5 py-2.5 text-[13px] text-danger">
                <span className="min-w-0 flex-1 whitespace-pre-wrap break-words">{ws.err}</span>
                <button type="button" className="ml-auto shrink-0 text-[11px] underline" onClick={() => ws.setErr("")}>{copy.app.dismiss}</button>
                <button type="button" className="shrink-0 text-[11px] underline" onClick={() => { ws.setErr(""); void ws.refresh(); }}>{copy.app.retry}</button>
              </div>
            ) : null}
            {videoing && ws.videoBoard && !canvasing ? (
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="video-board">
                <div className="h-full min-h-0 overflow-hidden bg-sidebar">
                  {workshop}
                </div>
              </div>
            ) : inspectOpen ? (
              <Group
                key={three ? "agent-inspect" : "lab-dock"}
                className="min-h-0 min-w-0 flex-1"
                orientation="horizontal"
                defaultLayout={{ main: Math.max(54, 100 - innerInspect), inspect: innerInspect }}
                onLayoutChanged={(next, meta) => {
                  if (!meta.isUserInteraction || typeof next.inspect !== "number") return;
                  const inspectWindow = (next.inspect / 100) * stagePct;
                  setLayout((prev) => writeLayout({
                    inspect: inspectWindow,
                    main: Math.max(30, 100 - prev.rail - inspectWindow),
                  }));
                }}
              >
                <Panel id="main" minSize="32" className="h-full min-h-0 min-w-0">
                  {workspace}
                </Panel>
                <ResizeHandle />
                <Panel id="inspect" minSize="16" maxSize="48" className="h-full min-h-0 min-w-0">
                  <div className="h-full min-h-0 overflow-hidden bg-sidebar">
                    {three ? inspect : (
                      <ChatDock
                        items={ws.items}
                        approvals={ws.approvals}
                        running={ws.threadRunning}
                        draftKey={ws.draftKey}
                        disabled={false}
                        disabledReason=""
                        model={ws.active?.model || ws.savedCfg.model || ws.health.model}
                        models={ws.savedCfg.models}
                        provider={ws.savedCfg.provider}
                        ctx={ws.ctx}
                        queued={ws.queued}
                        onSend={ws.onSend}
                        onStop={ws.onStop}
                        onResolve={ws.onResolve}
                        onRetry={() => { void ws.onRetryLast(); }}
                        onOpenAgent={() => ws.showConversation()}
                        onSlash={(cmd, rest) => { void ws.onSlash(cmd, rest); }}
                        onModel={async (model) => {
                          if (!ws.activeId) return;
                          const t = await api.setSessionModel(ws.activeId, model);
                          ws.setActive(t);
                          ws.refreshCtx();
                        }}
                        {...bindComposer}
                      />
                    )}
                  </div>
                </Panel>
              </Group>
            ) : (
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{workspace}</div>
            )}
          </MainColumn>
        </Panel>
      </Group>
      {!showRail && !popout ? (
        <div
          className={overlayRail
            ? "absolute inset-y-0 left-0 z-20 w-[min(288px,90%)]"
            : "absolute inset-y-0 left-0 z-20 w-2"}
          onMouseEnter={() => ws.setSidebarHover(true)}
          onMouseLeave={() => ws.setSidebarHover(false)}
        >
          {overlayRail ? <div className="h-full">{rail}</div> : null}
        </div>
      ) : null}
    </AppFrame>
  );
}

function SettingsSurface({ ws }: { ws: ReturnType<typeof useWorkstation> }) {
  const copy = ws.copy;
  return (
    <SettingsPage
      host={{
        cfg: ws.savedCfg,
        health: ws.health,
        plugins: ws.plugins,
        logs: ws.logs,
        journal: ws.journal,
        doctor: ws.doctor,
        vault: ws.vault,
        sessionId: ws.activeId,
        patch: ws.patchConfig,
        saveProvider: async (next, key) => {
          if (key.trim()) await api.setAPIKey(key.trim());
          await ws.patchConfig(next);
          await ws.refresh();
        },
        onBrowse: () => api.pickFolder(),
        onStartMcp: async (name, command, args) => { await api.startMCP(name, command, args); await ws.refresh(); },
        onStartMcpHttp: async (name, endpoint) => { await api.startMCPHTTP(name, endpoint); await ws.refresh(); },
        onStopMcp: async (name) => { await api.stopMCP(name); await ws.refresh(); },
        onReplaceMcp: async (servers) => { await api.replaceMCP(servers); await ws.refresh(); },
        onUnload: async (name) => { await api.unloadFiber(name); await ws.refresh(); },
        onCheckUpdate: () => api.checkUpdate(),
        onApplyUpdate: async () => {
          try { await api.applyUpdate(); await ws.refresh(); toast.success(copy.app.stagedUpdate); }
          catch (e) { ws.fail(e); }
        },
        onTestProvider: () => api.testProvider(),
        onRevealLogs: () => api.revealLogs(),
        onRevealJournal: () => api.revealJournal(),
        onCopyLogPath: async () => {
          const path = String(ws.logs?.path || "");
          if (path) await writeClipboard(path);
        },
        onExportDiagnostics: async () => {
          await api.exportDiagnostics();
        },
        onDiagnose: async () => {
          if (!ws.activeId) return;
          await api.diagnoseSession(ws.activeId);
        },
      }}
    />
  );
}

function joinWorkspace(root: string, rel: string): string {
  if (!rel) return root;
  if (/^[a-zA-Z]:[\\/]/.test(rel) || rel.startsWith("/")) return rel;
  const base = (root || "").replace(/[\\/]+$/, "");
  return base ? `${base}/${rel.replace(/^[\\/]+/, "")}` : rel;
}
