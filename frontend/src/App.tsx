import { useState } from "react";
import { Group, Panel } from "react-resizable-panels";
import { toast } from "sonner";
import * as api from "./lib/client";
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
import { Composer } from "./features/Composer";
import { Inspector } from "./features/Inspector";
import { ChatDock } from "./features/ChatDock";
import { About } from "./features/About";
import { ConfirmDialog } from "./features/ConfirmDialog";
import { BootSkeleton } from "./features/BootSkeleton";
import { HarborLab } from "./features/labs/HarborLab";
import { EvolveLab } from "./features/labs/EvolveLab";
import { HarnessLab } from "./features/labs/HarnessLab";
import { SettingsPage } from "./features/settings/SettingsPage";
import { useSettingsHash } from "./features/settings/useSettingsHash";
import { useWorkstation } from "./features/workstation/useWorkstation";
import { Sheet, SheetContent, SheetDescription, SheetTitle } from "./components/ui/sheet";
import { Button } from "./components/ui/button";
import { Kbd } from "./components/ui/kbd";
import { PanelLeft } from "lucide-react";
import { DEFAULT_KEYMAP, displayShortcut } from "./lib/keymap";
import { isMac } from "./lib/chrome";
import { displayTitle } from "./lib/display-title";
import type { Thread } from "./lib/protocol";

function patchThread(list: Thread[], id: string, patch: Partial<Thread>): Thread[] {
  const next = list.map((x) => (x.id === id ? { ...x, ...patch } : x));
  next.sort((a, b) => Number(!!b.pinned) - Number(!!a.pinned));
  return next;
}

export default function App() {
  const ws = useWorkstation();
  useSettingsHash();
  const copy = ws.copy;
  const [layout, setLayout] = useState(readLayout);
  const sheetInspect = useMedia("(max-width: 799px)");
  const railNarrow = useMedia("(max-width: 799px)");
  const settings = ws.surface === "settings";
  const three = ws.surface === "agent" && ws.inspector && !sheetInspect;
  const dock = ws.surface !== "agent" && !settings;
  const inspectOpen = three || dock;
  const showRail = !ws.sidebarCollapsed && !railNarrow;
  const overlayRail = !showRail && ws.sidebarHover;
  const mac = isMac();
  const stagePct = showRail ? Math.max(66, 100 - layout.rail) : 100;
  const innerInspect = Math.min(46, Math.max(22, (layout.inspect / stagePct) * 100));

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
      files={ws.files}
      skills={ws.skills}
      onSearchFiles={(q) => { void api.searchFiles(ws.savedCfg.workspace, q).then(ws.setFiles).catch(() => {}); }}
      onSend={ws.onSend}
      onStop={ws.onStop}
      onSlash={(cmd, rest) => { void ws.onSlash(cmd, rest); }}
      onModel={async (model) => {
        if (!ws.activeId) return;
        const t = await api.setSessionModel(ws.activeId, model);
        ws.setActive(t);
        ws.refreshCtx();
      }}
      onPickFiles={async () => {
        const paths = await api.pickFiles();
        return paths.map((path) => ({ path, name: path.replace(/^.*[\\/]/, "") }));
      }}
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
      onApply={ws.applySelected}
      onRefreshDiff={ws.refreshDiff}
      onQuote={(text) => {
        const cur = useUI.getState().drafts[ws.draftKey] || "";
        useUI.getState().setDraft(ws.draftKey, cur ? `${cur}\n${text}` : text);
      }}
      sessionId={ws.activeId}
      running={ws.threadRunning}
      trace={ws.trace}
      onRefreshTrace={() => { void ws.refreshTrace(); }}
      onLoadSpill={ws.loadSpill}
    />
  );

  const agentPane = (
    <section className="relative flex h-full min-h-0 min-w-0 flex-col">
      <Transcript
        items={ws.items}
        approvals={ws.approvals}
        running={ws.threadRunning}
        onResolve={ws.onResolve}
        onPrompt={(text) => useUI.getState().setDraft(ws.draftKey, text)}
        onRetry={() => { void ws.onRetryLast(); }}
        onOpenReview={() => {
          ws.setInspector(true);
          ws.setInspTab("diff");
          void ws.refreshDiff();
        }}
      />
      {composer}
    </section>
  );

  const labPane =
    ws.lab === "harbor" ? (
      <HarborLab
        busy={ws.labBusy}
        error={ws.harborErr}
        lastKind={ws.harborKind}
        report={ws.evalReport}
        best={ws.bestReport}
        models={ws.bonModels}
        onModels={ws.setBonModels}
        onRun={async (kind) => {
          ws.setHarborKind(kind);
          ws.setHarborErr("");
          ws.setLabBusy(copy.labs.busyHarbor);
          try {
            if (kind === "suite") { ws.setEvalReport(await api.runEval()); ws.setBestReport(null); }
            if (kind === "safety") { ws.setEvalReport(await api.runEvalSafety()); ws.setBestReport(null); }
            if (kind === "tb") { ws.setEvalReport(await api.runEvalTB()); ws.setBestReport(null); }
            if (kind === "bon") {
              const r = await api.bestOfN(3);
              ws.setBestReport(r);
              ws.setEvalReport(r.best || r.Best);
            }
            if (kind === "models") {
              const r = await api.bestOfModels(ws.bonModels.split(",").map((s) => s.trim()).filter(Boolean));
              ws.setBestReport(r);
              ws.setEvalReport(r.best || r.Best);
            }
          } catch (e) {
            ws.setHarborErr(api.errMessage(e));
            ws.fail(e);
          } finally {
            ws.setLabBusy(null);
          }
        }}
      />
    ) : ws.lab === "evolve" ? (
      <EvolveLab
        busy={!!ws.labBusy}
        k={ws.evolveK}
        onK={ws.setEvolveK}
        evolve={ws.evolve}
        playbook={ws.playbook}
        archive={ws.tree}
        onRun={async () => {
          ws.setLabBusy(copy.labs.busyEvolve);
          try {
            ws.setEvolve(await api.evolve(ws.evolveK));
            ws.setTree(await api.archive());
            await ws.refresh();
          } catch (e) { ws.fail(e); }
          finally { ws.setLabBusy(null); }
        }}
        onRate={async (id, helpful) => {
          ws.setPlaybook(await api.ratePlaybook(id, helpful));
          await ws.refresh();
        }}
      />
    ) : ws.lab === "harness" ? (
      <HarnessLab
        harness={ws.harness}
        diffA={ws.diffA}
        diffB={ws.diffB}
        diffOut={ws.diffOut}
        onA={ws.setDiffA}
        onB={ws.setDiffB}
        onCompare={async () => {
          try { ws.setDiffOut(await api.diff(ws.diffA || ws.health.harness, ws.diffB)); }
          catch (e) { ws.fail(e); }
        }}
        onCheckout={async (hash, l3) => {
          await api.checkout(hash, !!l3);
          await ws.refresh();
          toast.success(copy.app.checkedOut);
        }}
        onRollback={async () => { await api.rollback(); await ws.refresh(); toast.success(copy.app.rolledBack); }}
      />
    ) : null;

  const rail = (
    <ThreadRail
      threads={ws.threads}
      activeId={ws.activeId}
      query={ws.query}
      running={ws.running}
      lab={ws.lab}
      surface={ws.surface}
      showArchived={ws.showArchived}
      notices={ws.notices}
      noticesOpen={ws.noticesOpen}
      connected={ws.health.ok}
      isolated={ws.health.isolated}
      onQuery={ws.setQuery}
      onSelect={ws.setActive}
      onNew={ws.onNew}
      onLab={ws.setLab}
      onSettings={() => ws.openSettings()}
      onCollapse={() => ws.setSidebarCollapsed(true)}
      onToggleArchived={() => ws.setShowArchived((v) => !v)}
      onToggleNotices={() => ws.setNoticesOpen((v) => !v)}
      onClearNotices={ws.clearNotices}
      onNotice={(n) => {
        if (n.sessionId) {
          const t = ws.threads.find((x) => x.id === n.sessionId);
          if (t) {
            ws.setActive(t);
            ws.setLab("agent");
          }
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
          ws.setActive(next);
          ws.setLab("agent");
          toast.success(copy.app.forked);
        } catch (e) {
          ws.fail(e);
        }
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

  const headerLeft = settings ? (
    <Button size="sm" variant="ghost" onClick={ws.closeSettings}>{copy.settings.back}</Button>
  ) : ws.sidebarCollapsed || railNarrow ? (
    <Button size="icon" variant="ghost" aria-label={copy.rail.expand} onClick={() => { ws.setSidebarCollapsed(false); ws.setSidebarHover(true); }}>
      <PanelLeft />
    </Button>
  ) : null;

  const headerRight = (
    <button
      type="button"
      className="mr-0.5 hidden items-center rounded-lg px-1.5 py-1 text-muted transition-colors hover:bg-lift hover:text-foreground sm:inline-flex"
      onClick={() => ws.setPalette(true)}
      aria-label={copy.rail.jump}
      title={copy.rail.jump}
    >
      <Kbd>{displayShortcut(DEFAULT_KEYMAP.palette)}</Kbd>
    </button>
  );

  const headerTitle = settings
    ? <span className="text-[13px] font-medium">{copy.settings.title}</span>
    : ws.surface === "harbor"
      ? <HeaderLabel title={copy.labs.harbor} hint={copy.labs.harborHint} />
      : ws.surface === "evolve"
        ? <HeaderLabel title={copy.labs.evolve} hint={copy.labs.evolveHint} />
        : ws.surface === "harness"
          ? <HeaderLabel title={copy.labs.harness} hint={copy.labs.harnessHint} />
          : (
            <Titlebar
              workspace={ws.savedCfg.workspace}
              inspector={ws.inspector}
              title={ws.active ? displayTitle(ws.active.title, copy.rail.untitled) : copy.rail.newChat}
              runningCount={Object.values(ws.running).filter(Boolean).length}
              runningThreads={ws.threads.filter((t) => ws.running[t.id])}
              onSelectRunning={(t) => { ws.setActive(t); ws.setLab("agent"); }}
              renameTick={ws.renameTick}
              onToggleInspector={() => ws.setInspector((v) => !v)}
              onRename={async (title) => {
                if (!ws.activeId) return;
                await api.renameSession(ws.activeId, title);
                await ws.refresh();
              }}
              onFork={async () => {
                if (!ws.activeId) return;
                const t = await api.forkSession(ws.activeId);
                ws.setThreads((prev) => [t, ...prev]);
                ws.setActive(t);
              }}
              onDiff={ws.refreshDiff}
            />
          );

  const workspace = settings ? (
    <div className="h-full min-h-0 overflow-hidden rounded-[10px] border border-border bg-sidebar">
      <SettingsSurface ws={ws} />
    </div>
  ) : ws.surface === "agent" ? agentPane : labPane;

  return (
    <AppFrame
      overlay={(
        <>
          {!ws.booted ? <div className="absolute inset-0 z-30"><BootSkeleton /></div> : null}
          <CommandPalette
            open={ws.palette}
            threads={ws.threads}
            onClose={() => ws.setPalette(false)}
            onNew={ws.onNew}
            onLab={ws.setLab}
            onSelectThread={ws.setActive}
            onDiff={ws.refreshDiff}
            onAbout={async () => { ws.setAboutInfo(await api.about().catch(() => ({}))); ws.setAboutOpen(true); }}
            onQuit={ws.requestQuit}
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
          <Sheet modal={false} open={sheetInspect && ws.inspector && ws.surface === "agent"} onOpenChange={ws.setInspector}>
            <SheetContent side="right" overlay={false} className="flex flex-col p-0">
              <SheetTitle className="sr-only">{copy.review.toggle}</SheetTitle>
              <SheetDescription className="sr-only">{copy.review.toggle}</SheetDescription>
              {inspect}
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
              <div className="flex items-center gap-3 border-b border-danger/30 bg-danger/10 px-4 py-1.5 text-[13px] text-danger">
                {ws.err}
                <button type="button" className="ml-auto text-[11px] underline" onClick={() => ws.setErr("")}>{copy.app.dismiss}</button>
                <button type="button" className="text-[11px] underline" onClick={() => { ws.setErr(""); void ws.refresh(); }}>{copy.app.retry}</button>
              </div>
            ) : null}
            {inspectOpen ? (
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
                <Panel id="inspect" minSize="16" maxSize="48" className="h-full min-h-0 min-w-0 pl-2">
                  <div className="h-full min-h-0 overflow-hidden rounded-xl border border-border bg-sidebar">
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
                        onOpenAgent={() => ws.setLab("agent")}
                        onSlash={(cmd, rest) => { void ws.onSlash(cmd, rest); }}
                        onModel={async (model) => {
                          if (!ws.activeId) return;
                          const t = await api.setSessionModel(ws.activeId, model);
                          ws.setActive(t);
                          ws.refreshCtx();
                        }}
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
      {!showRail ? (
        <div
          className={overlayRail
            ? "absolute top-2 bottom-2 left-0 z-20 w-[min(288px,90%)]"
            : "absolute top-2 bottom-2 left-0 z-20 w-2"}
          onMouseEnter={() => ws.setSidebarHover(true)}
          onMouseLeave={() => ws.setSidebarHover(false)}
        >
          {overlayRail ? <div className="h-full pl-2">{rail}</div> : null}
        </div>
      ) : null}
    </AppFrame>
  );
}

function HeaderLabel({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="min-w-0 truncate text-[13px] font-medium" title={hint}>
      {title}
    </div>
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
        doctor: ws.doctor,
        vault: ws.vault,
        patch: ws.patchConfig,
        saveProvider: async (next, key) => {
          if (key.trim()) await api.setAPIKey(key.trim());
          await ws.patchConfig(next);
          await ws.refresh();
        },
        onBrowse: () => api.pickFolder(),
        onStartMcp: async (name, command, args) => { await api.startMCP(name, command, args); await ws.refresh(); },
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
      }}
    />
  );
}
