/**
 * 拥有：窗口怎么拼起来：侧栏、对话、检查器、各个表面，以及设置页的宿主回调。
 * 不拥有：会话状态，那些在 useWorkstation。会调用 api 改工作区、模型、权限、隔离、置顶、归档、分叉、页面和密钥。
 */

import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Group, Panel } from "react-resizable-panels";
import { toast } from "sonner";
import { cn } from "./lib/utils";
import { THREAD_COL, THREAD_GUTTER } from "./lib/thread";
import * as api from "./lib/client";
import { writeClipboard } from "./lib/clipboard";
import { readLayout, writeLayout } from "./lib/layout";
import { useMedia } from "./lib/media";
import { useUI } from "./lib/store";
import { adoptPendingAuthMode } from "./lib/pending-auth";
import { AppFrame, MainColumn } from "./features/shell/AppFrame";
import { PageHeader } from "./features/shell/PageHeader";
import { CommandPalette } from "./features/CommandPalette";
import { ResizeHandle } from "./features/ResizeHandle";
import { Titlebar } from "./features/Titlebar";
import { UpdateButton } from "./features/UpdateButton";
import { ThreadRail } from "./features/ThreadRail";
import { Transcript } from "./features/Transcript";
import { OrchestratorDesk } from "./features/superagent/OrchestratorDesk";
import { TurnOutline } from "./features/TurnOutline";
import { HomeHero, HomeStarters } from "./features/HomeStage";
import { ChatNotWorking } from "./features/harness/ChatNotWorking";
import { Composer } from "./features/Composer";
import { SessionHeader } from "./features/session-chrome/SessionHeader";
import { Inspector } from "./features/Inspector";
import { FilePeek } from "./features/FilePeek";
import { ChatDock } from "./features/ChatDock";
import { About } from "./features/About";
import { ConfirmDialog } from "./features/ConfirmDialog";
import { BootSkeleton } from "./features/BootSkeleton";
import { canaryDirty, parseHarnessRefs, shortHash, stagingDirty } from "./lib/harness-refs";
import { useSettingsHash } from "./features/settings/useSettingsHash";
import { useWorkstation } from "./features/workstation/useWorkstation";
import { isCompanionSurface, readPopoutId } from "./lib/popout";
import { CompanionApp, PresenceLayer } from "./features/presence";
import { IdolStage } from "./features/idol/IdolStage";
import { Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle } from "./components/ui/sheet";
import { Button } from "./components/ui/button";
import { Kbd } from "./components/ui/kbd";
import { browserSurface } from "./lib/chrome";
import { ChevronLeft, PanelLeft, Search, X } from "lucide-react";
import { DEFAULT_KEYMAP, displayShortcut } from "./lib/keymap";
import { isMac } from "./lib/chrome";
import { displayTitle } from "./lib/display-title";
import { recentWorkspaces, joinWorkspace, sessionFsRoot } from "./lib/workspace";
import { pendingAsk } from "./lib/ask-form";
import { channelForSurface, type AuthMode, type Thread } from "./lib/protocol";
import { learnScopeSpec } from "./lib/learn-scope";
import { RailSlotContext } from "./features/shell/RailSlot";
import { latestTaskPlan, parsePlanText } from "./lib/plan";
import { liveAutoSkills, liveSubagentCrumb } from "./lib/transcript-layout";
import { StudioCopilotHeader } from "./features/video/StudioCopilotHeader";
import { VideoShellNav } from "./features/video/VideoShellNav";
import { CutStage } from "./features/video/CutStage";
import { useCutDesk } from "./features/video/cut-desk";
import { useFilmDesk } from "./features/video/film-desk";
import { useRemakeDesk } from "./features/video/remake-desk";
import { useCreateMode } from "./features/video/create-mode";
import { CreateSkills } from "./features/video/CreateSkills";
import { useStudioSelection } from "./features/video/workshop-store";
import { studioSessionId } from "./features/video/studio-session";
import { sessionLane } from "./features/video/video-lanes";

/** 这些页第一次打开才加载。 */
const HarborLab = lazy(() => import("./features/labs/HarborLab").then((m) => ({ default: m.HarborLab })));
const EvolveLab = lazy(() => import("./features/labs/EvolveLab").then((m) => ({ default: m.EvolveLab })));
const HarnessLab = lazy(() => import("./features/labs/HarnessLab").then((m) => ({ default: m.HarnessLab })));
const HarnessWorkspace = lazy(() => import("./features/harness/HarnessWorkspace").then((m) => ({ default: m.HarnessWorkspace })));
const SettingsPage = lazy(() => import("./features/settings/SettingsPage").then((m) => ({ default: m.SettingsPage })));
const SkillsWorkspace = lazy(() => import("./features/skills/SkillsWorkspace").then((m) => ({ default: m.SkillsWorkspace })));
const LearnWorkspace = lazy(() => import("./features/learn/LearnWorkspace").then((m) => ({ default: m.LearnWorkspace })));
const VideoWorkshop = lazy(() => import("./features/video/VideoWorkshop").then((m) => ({ default: m.VideoWorkshop })));

/** 按 id 合并字段。置顶的排在前面。list 做映射。patch 展开传递。 */
function patchThread(list: Thread[], id: string, patch: Partial<Thread>): Thread[] {
  const next = list.map((x) => (x.id === id ? { ...x, ...patch } : x));
  next.sort((a, b) => Number(!!b.pinned) - Number(!!a.pinned));
  return next;
}

/** App 在陪伴表面只渲染 CompanionApp，其余交给 WorkstationApp。 */
export default function App() {
  if (isCompanionSurface()) return <CompanionApp />;
  return <WorkstationApp />;
}

/** 按当前表面决定显示对话、设置、技能、页面、偶像、视频或 harness。启动完成后闪 900 毫秒。焦点每 2 秒刷新一次空闲时间。 */
function WorkstationApp() {
  const ws = useWorkstation();
  useSettingsHash();
  const copy = ws.copy;
  const popoutId = readPopoutId();
  const popout = !!popoutId;
  const [layout, setLayout] = useState(readLayout);
  const reviewFile = useUI((s) => s.reviewFile);
  const setReviewFile = useUI((s) => s.setReviewFile);
  const pendingAuthMode = useUI((s) => s.pendingAuthMode);
  const sheetInspect = useMedia("(max-width: 799px)");
  const railNarrow = useMedia("(max-width: 799px)");
  const settings = !popout && ws.surface === "settings";
  const skills = !popout && ws.surface === "skills";
  const paging = !popout && ws.surface === "pages";
  const idling = !popout && ws.surface === "idol";
  const orchestrating = !popout && ws.surface === "orchestrator";
  const videoing = !popout && ws.surface === "video";
  const harnessing = !popout && ws.surface === "harness";
  const videoPane = useUI((s) => s.videoPane);
  const studioDesk = useUI((s) => s.studioDesk);
  const createDesk = videoing && videoPane === "create";
  const cutBound = useCutDesk((s) => !!s.project?.source_hash);
  const createMode = useCreateMode((s) => s.mode);
  const createEngaged = useCreateMode((s) => s.engaged);
  const filmOpen = useFilmDesk((s) => !!s.project?.id);
  const remakeOpen = useRemakeDesk((s) => !!s.draft?.id);
  const createLauncher = createDesk && !cutBound && !filmOpen && !remakeOpen;
  const finishDesk = createDesk && cutBound;
  const talkDesk = createDesk;
  const studioId = useStudioSelection((s) => s.studioId);
  const studioCopilotSession = studioSessionId(studioId, ws.videoProjects);
  const studioCopilotForeign = studioDesk && (!studioId || (!!studioCopilotSession && studioCopilotSession !== ws.activeId));
  const activeLane = sessionLane(ws.activeId, ws.videoProjects || []);
  const paneMismatch = (createDesk && !!activeLane)
    || (videoPane === "drama" && !!activeLane && activeLane !== "drama");
  const moduleLoading = useUI((s) => s.moduleLoading);
  const showHarnessLab = useUI((s) => s.showHarnessLab);
  const [winFocused, setWinFocused] = useState(() => typeof document === "undefined" || document.hasFocus());
  const idleAt = useRef(Date.now());
  const fileSearchGen = useRef(0);
  const [now, setNow] = useState(() => Date.now());
  const [waking, setWaking] = useState(false);
  const sawBoot = useRef(false);
  const paneWas = useRef(videoPane);
  useEffect(() => {
    if (paneWas.current !== videoPane && videoPane === "create") useCreateMode.getState().setEngaged(false);
    paneWas.current = videoPane;
  }, [videoPane]);
  useEffect(() => {
    if (ws.booted && !sawBoot.current) {
      sawBoot.current = true;
      setWaking(true);
      const t = window.setTimeout(() => setWaking(false), 900);
      return () => window.clearTimeout(t);
    }
  }, [ws.booted]);
  useEffect(() => {
    /** 窗口得到焦点时记下，并重置空闲起点。 */
    const on = () => { setWinFocused(true); idleAt.current = Date.now(); };
    /** 窗口失去焦点。 */
    const off = () => setWinFocused(false);
    window.addEventListener("focus", on);
    window.addEventListener("blur", off);
    const t = window.setInterval(() => setNow(Date.now()), 2000);
    return () => {
      window.removeEventListener("focus", on);
      window.removeEventListener("blur", off);
      window.clearInterval(t);
    };
  }, []);
  const staged = videoing && (ws.videoBoard || ws.canvasStage || studioDesk || (videoPane !== "chat" && videoPane !== "create" && videoPane !== "game"));
  const agent = popout || ws.surface === "agent";
  /** 欢迎线程还在回放历史或报告残留运行时，创作入口留着。只有从这张桌子发出对话、并且对话里已经有内容时才让开。 */
  const createEntrance = createDesk && !cutBound && !(createEngaged && ws.items.length > 0);
  const home = !staged && (agent || videoing) && (createEntrance || ((ws.items.length === 0 || paneMismatch) && !ws.threadRunning && ws.approvals.length === 0));
  const copilotItems = paneMismatch ? [] : ws.items;
  const boundCopilotId = studioDesk ? studioCopilotSession : "";
  const foreignSession = !!(boundCopilotId && ws.activeId && boundCopilotId !== ws.activeId);
  const boundCopilotThread = foreignSession ? ws.threads.find((t) => t.id === boundCopilotId) : undefined;
  const subagentCrumb = liveSubagentCrumb(copilotItems);
  const three = agent && !popout && ws.inspector && !sheetInspect;
  const dock = harnessing && ws.chatDock && !sheetInspect;
  const inspectOpen = three || dock;
  const peekOpen = three && !!reviewFile && ws.inspTab === "files";
  const showRail = !popout && !ws.sidebarCollapsed && !railNarrow;
  const overlayRail = !popout && !showRail && ws.sidebarHover;
  const navInRail = showRail || overlayRail;
  const [idolSlot, setIdolSlotState] = useState<HTMLDivElement | null>(null);
  const onIdolSlot = useCallback((node: HTMLDivElement | null) => {
    setIdolSlotState((cur) => (cur === node ? cur : node));
  }, []);
  const mac = isMac();
  const stagePct = showRail ? Math.max(66, 100 - layout.rail) : 100;
  const innerInspect = Math.min(46, Math.max(22, (layout.inspect / stagePct) * 100));
  const refs = parseHarnessRefs(ws.harness, ws.health.harness);
  const sheetRight = sheetInspect && ((agent && ws.inspector) || (harnessing && ws.chatDock));
  const presenceInput = useMemo(() => ({
    booted: ws.booted,
    running: ws.threadRunning,
    items: ws.presenceItems || ws.items,
    approvals: ws.approvals.length,
    moduleLoading,
    windowFocused: winFocused,
    idleMs: winFocused ? 0 : now - idleAt.current,
    waking,
    now,
  }), [ws.booted, ws.threadRunning, ws.presenceItems, ws.items, ws.approvals.length, moduleLoading, winFocused, now, waking]);

  const sessionWs = ws.active?.workspace || ws.savedCfg.workspace;
  const toolRoot = sessionFsRoot(ws.active, sessionWs);
  /** 空路径直接返回。还没有会话时新建一个，并套用待定的权限。已有会话则改它的工作区。path 为假时走另一支，传给 api.createSession，传给 api.setSessionWorkspace。 */
  const applySessionWorkspace = async (path: string) => {
    if (!path) return;
    if (!ws.activeId) {
      const t = await adoptPendingAuthMode(await api.createSession(path, channelForSurface(ws.surface)));
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
    sessions: ws.threads.map((t) => ({
      id: t.id,
      title: t.title,
      workspace: t.workspace,
      parentId: t.parentId,
    })),
    skills: ws.skills,
    autoSkills: liveAutoSkills(ws.items),
    /** 还没有第一段对话时，显示下一段对话会用的权限。 */
    authMode: studioDesk && (studioCopilotForeign || !ws.activeId)
      ? (pendingAuthMode || "full")
      : (ws.active?.authMode || pendingAuthMode || "full"),
    workspace: toolRoot,
    workspaces: recentWorkspaces([sessionWs, ws.savedCfg.workspace, ...ws.threads.map((t) => t.workspace)]),
    isolate: !!ws.active?.isolate,
    /** onWorkspace 调用 applySessionWorkspace。path 传给 applySessionWorkspace。 */
    onWorkspace: (path: string) => { void applySessionWorkspace(path); },
    onBrowseWorkspace: async () => {
      const p = await api.pickFolder();
      if (p) await applySessionWorkspace(p);
    },
    /** onIsolate 调用 setSessionIsolate。isolate 传给 api.setSessionIsolate。 */
    onIsolate: async (isolate: boolean) => {
      if (!ws.activeId) return;
      const t = await api.setSessionIsolate(ws.activeId, isolate);
      ws.setActive(t);
      ws.setThreads((list) => patchThread(list, t.id, t));
    },
    /** onSearchFiles 调用 catch。q 传给 api.searchFiles。 */
    onSearchFiles: (q: string) => {
      const n = ++fileSearchGen.current;
      void api.searchFiles(toolRoot, q).then((hits) => {
        if (n === fileSearchGen.current) ws.setFiles(hits);
      }).catch(() => {});
    },
    onPickFiles: async () => {
      const paths = await api.pickFiles();
      return paths.map((path) => ({ path, name: path.replace(/^.*[\\/]/, "") }));
    },
    /** onIngestVideo 调用 ingestVideo。source 传给 ws.ingestVideo。 */
    onIngestVideo: (source: { path?: string; file?: File }) => ws.ingestVideo(source),
    /** onAuthMode 调用 setPendingAuthMode。mode 传给 setPendingAuthMode，传给 api.setSessionAuthMode。 */
    onAuthMode: async (mode: AuthMode) => {
      if (!ws.activeId) {
        /** 还没有对话：把选择留给即将创建的第一段（见 adoptPendingAuthMode）。 */
        useUI.getState().setPendingAuthMode(mode);
        return;
      }
      const t = await api.setSessionAuthMode(ws.activeId, mode);
      ws.setActive(t);
      ws.setThreads((list) => patchThread(list, t.id, t));
    },
    onClipboard: async () => api.clipboardRead(),
    onScreenshot: async () => api.captureScreenshot(toolRoot),
    queueItems: ws.queueItems,
    /** onQueueCancel 调用 onQueueCancel。id 传给 ws.onQueueCancel。 */
    onQueueCancel: (id: string) => { void ws.onQueueCancel(id); },
    /** onQueueReorder 调用 onQueueReorder。id 传给 ws.onQueueReorder。delta 传给 ws.onQueueReorder。 */
    onQueueReorder: (id: string, delta: number) => { void ws.onQueueReorder(id, delta); },
    onApplyWorktree: () => { void ws.onApplyWorktree(); },
    taskPlan: latestTaskPlan(ws.items) || parsePlanText(ws.active?.planText || ""),
    methodologyActive: ws.methodologyActive,
    connectionId: ws.active?.connectionId,
    chatProviders: ws.health.chatProviders,
    onBuildPlan: () => {
      const plan = latestTaskPlan(ws.items) || parsePlanText(ws.active?.planText || "");
      if (!plan) return;
      const body = [
        "Build this plan as written. Do not rewrite the plan unless a step is blocked.",
        plan.explanation,
        ...plan.steps.map((s, i) => `${i + 1}. [${s.status}] ${s.step}`),
      ].filter(Boolean).join("\n");
      useUI.getState().setPlan(false);
      void ws.onSend({ text: body });
    },
  };

  const composer = (
    <Composer
      draftKey={ws.draftKey}
      running={ws.threadRunning}
      disabled={!!ws.health.paused}
      disabledReason={ws.health.paused ? copy.composer.paused : ""}
      model={ws.active?.model || ws.savedCfg.model || ws.health.model}
      models={ws.savedCfg.models}
      provider={ws.savedCfg.provider}
      ctx={ws.ctx}
      queued={ws.queued}
      onSend={(opts) => {
        useCreateMode.getState().setEngaged(true);
        void ws.onSend(opts);
      }}
      placeholder={createLauncher ? copy.video.conductor.clipPlaceholder : undefined}
      onStop={ws.onStop}
      onSlash={(cmd, rest) => { void ws.onSlash(cmd, rest); }}
      onModel={async (model) => {
        if (!ws.activeId) return;
        const t = await api.setSessionModel(ws.activeId, model);
        ws.setActive(t);
        ws.refreshCtx();
      }}
      ask={pendingAsk(ws.approvals, ws.activeId)}
      approvals={ws.approvals}
      onResolveAsk={ws.onResolve}
      flush
      hero={home}
      perch={!home && copilotItems.length === 0}
      {...bindComposer}
      sessionId={ws.activeId}
      profileId={ws.active?.profileId}
      onProfile={async (id) => {
        const sid = (await ws.ensureThread()).id;
        const t = await api.setSessionProfile(sid, id);
        ws.setActive(t);
        ws.setThreads((list) => patchThread(list, t.id, t));
      }}
      onSavePage={async () => {
        const sid = (await ws.ensureThread()).id;
        const last = [...ws.items].reverse().find((it) => it.type === "assistant" && it.text.trim());
        const draft = useUI.getState().drafts[ws.draftKey] || "";
        const body = (last?.text || draft).trim();
        if (!body) return;
        const title = body.split("\n").find((l) => l.trim())?.trim().slice(0, 80) || copy.pages.untitled;
        await api.pagesPropose(sid, title, body);
      }}
      onFollow={async (in_) => {
        const sid = (await ws.ensureThread()).id;
        await api.scheduleCreate({ kind: "follow", spec: in_, prompt: "Check in on this thread.", session_id: sid });
      }}
      onCall={() => { void api.setCompanionCall(true); }}
      turnStatus={ws.turnStatus}
      sessionStats={ws.sessionStats}
    />
  );

  const learnChat = (
    <>
      <Transcript
        compact
        items={ws.items}
        showThinking={!!ws.savedCfg.showThinking}
        transcriptView={ws.savedCfg.transcriptView}
        collapseTiming={ws.savedCfg.collapseTiming}
        approvals={ws.approvals}
        running={ws.threadRunning}
        onResolve={ws.onResolve}
        onRetry={() => { void ws.onRetryLast(); }}
      />
      <div className="px-3 pb-3">
        <Composer
          plain
          compact
          flush
          placeholder={copy.learn.tutorPh}
          draftKey={ws.draftKey}
          running={ws.threadRunning}
          disabled={!!ws.health.paused}
          disabledReason={ws.health.paused ? copy.composer.paused : ""}
          onSend={(opts) => { void ws.onSend(opts); }}
          onStop={ws.onStop}
          ask={pendingAsk(ws.approvals, ws.activeId)}
          approvals={ws.approvals}
          onResolveAsk={ws.onResolve}
          sessionId={ws.activeId}
          turnStatus={ws.turnStatus}
        />
      </div>
    </>
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
      ctx={ws.ctx}
      onRefreshUsage={() => {
        ws.refreshCtx();
        void ws.refreshTrace();
      }}
      workspace={toolRoot}
      focusFile={reviewFile}
      peekInline={sheetInspect}
      onPreviewFile={setReviewFile}
      onRefreshTrace={() => { void ws.refreshTrace(); }}
      onLoadSpill={ws.loadSpill}
      onOpenPath={(rel) => {
        const path = joinWorkspace(toolRoot, rel);
        if (path) void api.openInEditor(path);
      }}
    />
  );

  const homeStarters = (
    <HomeStarters
      onPrompt={(text) => useUI.getState().setDraft(ws.draftKey, text)}
      onContinueLast={() => { void ws.onContinueLast(); }}
      onImportClip={() => { void ws.ingestVideo({}); }}
      onRemake={() => { void ws.startRemake(); }}
      onFilm={(structure, description) => { void ws.startFilm(structure, description); }}
      onHandoff={(dest) => { void ws.openCreateHandoff(dest); }}
      onOpenCut={(id) => { void ws.openCut(id); }}
      draftKey={ws.draftKey}
    />
  );
  const filmSurface = createDesk && !cutBound && !remakeOpen && (filmOpen || createMode !== "clip");
  const directing = home && filmSurface && createMode !== "clip";

  const agentPane = (
    <section className={cn("relative flex min-h-0 min-w-0 flex-row", studioDesk ? "flex-1" : "h-full")}>
      {home || studioDesk || talkDesk || paging ? null : (
      <TurnOutline
        turns={ws.outlineTurns}
        activeKey={ws.activeTurn}
        status={ws.outlineState}
        onJump={(turn) => { void ws.jumpToTurn(turn); }}
      />
      )}
      <div
        className="@container relative flex min-h-0 min-w-0 flex-1 flex-col"
        data-testid={home ? "conversation-column" : undefined}
      >
        {home || studioDesk || talkDesk || paging ? null : (
          <SessionHeader
            view={ws.inspector ? "trajectory" : "chat"}
            trail={subagentCrumb || undefined}
            onView={(view) => {
              if (view === "trajectory") {
                ws.setInspector(true);
                ws.setInspTab("trace");
                return;
              }
              ws.setInspector(false);
            }}
          />
        )}
        {ws.health.paused ? (
          <div className="border-b border-line/50 bg-lift px-4 py-1.5 text-[12px] text-muted" data-testid="pause-banner">
            {copy.pause.banner}
          </div>
        ) : null}
        {ws.active?.pageId && !paging && !studioDesk && !talkDesk ? (
          <div className="border-b border-line/50 px-4 py-1.5 text-[12px] text-muted">
            {copy.pages.banner}
          </div>
        ) : null}
        {foreignSession ? (
          <div className="flex items-center gap-3 border-b border-line/50 bg-lift px-4 py-1.5 text-[12px] text-muted" data-testid="foreign-session-banner">
            <span className="min-w-0 flex-1">{copy.composer.foreignSession}</span>
            {boundCopilotThread ? (
              <button
                type="button"
                className="shrink-0 text-[11px] font-medium text-foreground underline"
                data-testid="foreign-session-switch"
                onClick={() => ws.openThread(boundCopilotThread)}
              >
                {copy.composer.switchSession}
              </button>
            ) : null}
          </div>
        ) : null}
        <div
          className={cn(
            "flex min-h-0 min-w-0 w-full flex-1 flex-col",
            home && !railNarrow && "home-stage empty-rise",
            home && !railNarrow && (directing && filmOpen ? "overflow-hidden" : "overflow-auto"),
            home && railNarrow && "home-stage",
            home && !createDesk && !railNarrow && "justify-center py-10",
          )}
        >
        {railNarrow && home ? (
          <div className="min-h-0 flex-1 overflow-auto">
            <HomeHero workspace={toolRoot} onBrowseWorkspace={() => { void bindComposer.onBrowseWorkspace(); }} />
            {homeStarters}
          </div>
        ) : home ? (
          <HomeHero workspace={toolRoot} onBrowseWorkspace={() => { void bindComposer.onBrowseWorkspace(); }} />
        ) : (
          <Transcript
            items={copilotItems}
            showThinking={!!ws.savedCfg.showThinking}
            transcriptView={ws.savedCfg.transcriptView}
            collapseTiming={ws.savedCfg.collapseTiming}
            approvals={ws.approvals}
            running={ws.threadRunning}
            workspace={toolRoot}
            older={ws.older}
            loadingOlder={ws.loadingOlder}
            idle={ws.idle}
            onLoadOlder={ws.loadOlder}
            onResolve={ws.onResolve}
            onPrompt={(text) => {
              useUI.getState().setDraft(ws.draftKey, text);
            }}
            onRetry={() => { void ws.onRetryLast(); }}
            onOpenReview={(path) => {
              ws.setInspector(true);
              if (path) {
                setReviewFile(path);
                ws.setInspTab(/\.(html?|xhtml)$/i.test(path) ? "browser" : "files");
              } else {
                ws.setInspTab("files");
              }
              void ws.refreshDiff();
            }}
            jumpTo={ws.jumpTo}
            latestNonce={ws.latestNonce}
            onActiveTurn={ws.setActiveTurn}
            onRestoreFiles={(from) => { void ws.onRestoreFiles(from); }}
            onForkFrom={(from) => { void ws.onForkFrom(from); }}
            onJumpLatest={ws.jumpToLatest}
          />
        )}
        {home && !railNarrow && filmSurface ? homeStarters : null}
        <div className={cn("sc-seat", home ? cn("mx-auto w-full max-w-[min(100%,var(--thread-measure))] px-4 sm:px-8", railNarrow ? "shrink-0 pb-1" : createDesk ? "mt-4" : "mt-8") : cn(THREAD_COL, THREAD_GUTTER))}>
          {createLauncher ? <CreateSkills onPrompt={(text) => useUI.getState().setDraft(ws.draftKey, text)} /> : null}
          {!home && !videoing && !paging && !!ws.activeId && ws.active?.channel !== "video" && copilotItems.length > 0 && !ws.threadRunning ? (
            <ChatNotWorking onImprove={() => ws.openHarness("propose")} />
          ) : null}
          {composer}
        </div>
        {home && !railNarrow && !filmSurface ? homeStarters : null}
        </div>
      </div>
    </section>
  );

  const labPane = (
    <HarnessWorkspace
      tab={ws.harnessTab}
      onTab={ws.setHarnessTab}
      navInRail={navInRail}
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
            overlay={ws.overlay}
            overlayK={ws.overlayK}
            onOverlayK={ws.setOverlayK}
            overlayMode={ws.overlayMode}
            onOverlayMode={ws.setOverlayMode}
            overlayLibrary={ws.overlayLibrary}
            overlayReuse={ws.overlayReuseTurn}
            onOverlayReuse={ws.setOverlayReuseTurn}
            overlayDisabled={!ws.activeId || ws.active?.channel === "video"}
            overlayDisabledReason={!ws.activeId ? copy.labs.overlayNeedSession : ws.active?.channel === "video" ? copy.labs.overlayVideo : undefined}
            onOverlay={() => { void ws.runOverlay(); }}
            onOverlayProve={(hash) => { void ws.runHarborAt(hash); }}
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

  /** 窄屏时鼠标离开侧栏就收起悬停层。 */
  const leaveRail = () => {
    if (railNarrow) ws.setSidebarHover(false);
  };

  const rail = (
    <ThreadRail
      threads={ws.searchHits ?? ws.threads}
      activeId={ws.activeId}
      query={ws.query}
      running={ws.running}
      lab={ws.lab}
      surface={ws.surface}
      harness={ws.harness}
      fallbackActive={ws.health.harness}
      connected={ws.health.ok}
      isolated={ws.health.isolated}
      isolationKind={ws.health.isolationKind}
      showArchived={ws.showArchived}
      onShowArchived={ws.setShowArchived}
      onQuery={ws.setQuery}
      onSelect={(t) => { leaveRail(); ws.openThread(t); }}
      onSelectProject={(p) => { leaveRail(); void ws.openVideoProject(p); }}
      onOpenCut={(id) => { leaveRail(); void ws.openCut(id); }}
      onOpenFilm={(film) => { leaveRail(); useFilmDesk.getState().setProject(film); useUI.getState().openVideoPane("create"); }}
      onOpenRemake={(cut) => { leaveRail(); useRemakeDesk.getState().setDraft(cut); useUI.getState().openVideoPane("create"); }}
      videoProjects={ws.videoProjects}
      canvasProjectId={ws.canvasProjectId}
      dramaId={ws.dramaId}
      studioId={studioId}
      onNew={() => { leaveRail(); ws.onNew(); }}
      onNewIn={(path) => { leaveRail(); void ws.onNewIn(path); }}
      workspace={ws.surface === "video" ? (ws.savedCfg.videoWorkspace || ws.health.videoWorkspace || ws.savedCfg.workspace) : ws.savedCfg.workspace}
      onLab={ws.setLab}
      onChats={() => { leaveRail(); ws.showConversation(); }}
      onHarness={() => {
        leaveRail();
        if (ws.surface === "harness") ws.showConversation();
        else ws.openHarness("overview");
      }}
      onOrchestrator={() => {
        leaveRail();
        if (ws.surface === "orchestrator") ws.closeOrchestrator();
        else ws.openOrchestrator();
      }}
      onSkills={() => {
        leaveRail();
        if (ws.surface === "skills") ws.closeSkills();
        else ws.openSkills();
      }}
      onPages={() => {
        leaveRail();
        if (ws.surface === "pages") ws.closePages();
        else ws.openPages();
      }}
      onIdol={() => {
        leaveRail();
        if (ws.surface === "idol") ws.closeIdol();
        else ws.openIdol();
      }}
      onVideo={() => {
        leaveRail();
        if (ws.surface === "video") ws.closeVideo();
        else ws.openVideo();
      }}
      onSettings={() => {
        leaveRail();
        if (ws.surface === "settings") ws.closeSettings();
        else ws.openSettings();
      }}
      onCollapse={() => {
        ws.setSidebarHover(false);
        if (!railNarrow) ws.setSidebarCollapsed(true);
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
        if (!path) return;
        void api.openInEditor(path).then(() => {
          if (browserSurface()) toast.success(copy.rail.openedOnComputer);
        }).catch((e) => ws.fail(e));
      }}
      onOpenTerminal={(t) => {
        const path = t.workspace || ws.savedCfg.workspace;
        if (!path) return;
        void api.openWorkspaceTerminal(path).then(() => {
          if (browserSurface()) toast.success(copy.rail.openedOnComputer);
        }).catch((e) => ws.fail(e));
      }}
      onIdolSlot={onIdolSlot}
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

  const headerLeft = settings && !showRail ? (
    <Button size="icon" variant="ghost" aria-label={copy.settings.back} onClick={() => ws.closeSettings()}>
      <ChevronLeft />
    </Button>
  ) : ws.sidebarCollapsed || railNarrow ? (
    <Button size="icon" variant="ghost" aria-label={copy.rail.expand} onClick={() => { ws.setSidebarCollapsed(false); ws.setSidebarHover(true); }}>
      <PanelLeft />
    </Button>
  ) : null;

  const headerRight = (
    <>
      {harnessing ? (
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
        className="no-drag mr-0.5 inline-flex size-8 items-center justify-center rounded-lg text-muted transition-[background-color,color] duration-200 ease-[var(--ease-out)] hover:bg-lift hover:text-foreground sm:w-auto sm:px-1.5"
        onClick={() => ws.setPalette(true)}
        aria-label={copy.rail.jump}
        title={copy.rail.jump}
      >
        <Search className="size-4 sm:hidden" aria-hidden />
        <span className="hidden sm:inline-flex"><Kbd>{displayShortcut(DEFAULT_KEYMAP.palette)}</Kbd></span>
      </button>
      <UpdateButton />
    </>
  );

  const headerTitle = settings
    ? null
    : skills
      ? <span className="text-[13px] font-medium">{copy.skills.title}</span>
    : paging
      ? <span className="text-[13px] font-medium">{copy.rail.pages}</span>
    : idling
      ? <span className="text-[13px] font-medium">{copy.idol.title}</span>
    : harnessing
      ? (
        <div className="flex min-w-0 items-center gap-2">
          <span className="text-[13px] font-medium">{copy.rsi.title}</span>
          {showHarnessLab && refs.active ? (
            <span className="truncate font-mono text-[11px] text-muted" title={refs.active}>{shortHash(refs.active)}</span>
          ) : null}
          {stagingDirty(refs) ? (
            <span className="rounded-md bg-lift px-1.5 py-0.5 text-[10px] text-muted">{copy.rail.stagingDirty}</span>
          ) : canaryDirty(refs) ? (
            <span className="rounded-md bg-lift px-1.5 py-0.5 text-[10px] text-muted">{copy.rail.canaryDirty}</span>
          ) : null}
        </div>
      )
      : videoing
        ? (
            <Titlebar
              inspector={ws.inspector}
              hideInspector={true}
              hideInbox={true}
              title=""
              runningCount={Object.values(ws.running).filter(Boolean).length}
              runningThreads={ws.threads.filter((t) => ws.running[t.id])}
              runningStatus={ws.runStatus}
              onSelectRunning={(t) => ws.openThread(t)}
              renameTick={ws.renameTick}
              onToggleInspector={() => ws.setInspector((v) => !v)}
              paused={!!ws.health.paused}
              onPause={() => { void api.setPaused(!ws.health.paused).then(() => ws.refresh()); }}
              onRename={() => {}}
            />
        )
      : (
            <Titlebar
              inspector={ws.inspector}
              hideInspector={false}
              hideInbox={false}
              paused={!!ws.health.paused}
              onPause={() => { void api.setPaused(!ws.health.paused).then(() => ws.refresh()); }}
              title={home ? "" : (ws.active ? displayTitle(ws.active.title, copy.rail.untitled) : copy.rail.newChat)}
              runningCount={Object.values(ws.running).filter(Boolean).length}
              runningThreads={ws.threads.filter((t) => ws.running[t.id])}
              runningStatus={ws.runStatus}
              onSelectRunning={(t) => ws.openThread(t)}
              renameTick={ws.renameTick}
              onToggleInspector={() => ws.setInspector((v) => !v)}
              notices={ws.notices}
              onClearNotices={ws.clearNotices}
              onDismissNotice={ws.dismissNotice}
              threadTitleOf={(id) => ws.threads.find((x) => x.id === id)?.title}
              approvalCount={ws.approvals.length}
              onNotice={(n) => {
                if (n.sessionId) {
                  const t = ws.threads.find((x) => x.id === n.sessionId);
                  if (t) ws.openThread(t);
                }
              }}
              onOpenThread={(id) => {
                const t = ws.threads.find((x) => x.id === id);
                if (t) ws.openThread(t);
              }}
              onResolve={ws.onResolve}
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
      loaded={ws.active?.loadedSkills || []}
      workspace={ws.active?.workspace || ws.savedCfg.workspace}
      navInRail={navInRail}
      onRefreshInstalled={() => { void ws.reloadSkills(); }}
    />
  );

  const workshop = (
    <VideoWorkshop
      sessionId={ws.activeId}
      projects={ws.videoProjects}
      canvasProjectId={ws.canvasProjectId}
      dramaId={ws.dramaId}
      onOpenProject={(p) => { void ws.openVideoProject(p); }}
      onOpenCut={(id) => { void ws.openCut(id); }}
      onRefreshHistory={ws.loadVideoHistory}
      onNeedSession={async () => (await ws.ensureThread()).id}
      onClose={() => {
        useUI.getState().openVideoPane("create");
      }}
    />
  );

  const videoBoard = (
    <div className="@container flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="video-board">
      {railNarrow && !overlayRail ? <VideoShellNav layout="bar" /> : null}
      {studioDesk ? (
        studioId ? (
          <div className="studio-desk h-full min-h-0 overflow-hidden" data-testid="studio-desk">
            <div className="studio-desk-board" data-skin-pane>
              <Suspense fallback={null}>{workshop}</Suspense>
            </div>
            <div className="studio-desk-chat" data-skin-pane data-testid="studio-copilot">
              <StudioCopilotHeader />
              {agentPane}
            </div>
          </div>
        ) : (
          <div className="h-full min-h-0 overflow-hidden" data-testid="studio-desk" data-home="true">
            <div className="h-full min-h-0 overflow-hidden" data-skin-pane>
              <Suspense fallback={null}>{workshop}</Suspense>
            </div>
          </div>
        )
      ) : (
        <div className="h-full min-h-0 overflow-hidden" data-skin-pane>
          <Suspense fallback={null}>{workshop}</Suspense>
        </div>
      )}
    </div>
  );

  const workspace = settings ? (
    <Suspense fallback={null}>
      <div className="no-drag flex h-full min-h-0 w-full min-w-0 flex-col overflow-hidden">
        <SettingsSurface ws={ws} navInRail={navInRail} />
      </div>
    </Suspense>
  ) : skills ? (
    <Suspense fallback={null}>
      <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden" data-skin-pane>{skillsPane}</div>
    </Suspense>
  ) : paging ? (
    <Suspense fallback={null}>
      <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden" data-skin-pane data-testid="pages-surface">
        <LearnWorkspace
          sessionId={ws.activeId}
          boundPageId={ws.active?.channel === "learn" ? ws.active?.pageId : ""}
          navInRail={navInRail}
          tutorQuiet={ws.items.length === 0 && !ws.threadRunning}
          onTutorPrompt={(text) => useUI.getState().setDraft(ws.draftKey, text)}
          onEnsureScope={(spec) => ws.ensureLearnScope(spec)}
          onOpenInChat={(pageId) => {
            void ws.ensureLearnScope(learnScopeSpec({ lane: "library", shelf: "notes", title: copy.learn.notesTab })).then(async (t) => {
              const next = await api.setSessionPage(t.id, pageId);
              ws.setActive(next);
              ws.setThreads((list) => patchThread(list, next.id, next));
            });
          }}
          onUnbindChat={() => {
            const id = ws.active?.channel === "learn" ? ws.activeId : "";
            if (!id) return;
            void api.setSessionPage(id, "").then((t) => {
              ws.setActive(t);
              ws.setThreads((list) => patchThread(list, t.id, t));
            });
          }}
          chat={learnChat}
        />
      </div>
    </Suspense>
  ) : idling ? (
    <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden" data-skin-pane data-testid="idol-surface">
      <IdolStage />
    </div>
  ) : orchestrating ? (
    <div className="no-drag flex h-full min-h-0 w-full min-w-0 flex-col overflow-hidden" data-skin-pane data-testid="orchestrator-surface">
      <OrchestratorDesk />
    </div>
  ) : harnessing ? (
    <Suspense fallback={null}>
      <div className="no-drag flex h-full min-h-0 flex-col overflow-hidden" data-skin-pane>{labPane}</div>
    </Suspense>
  ) : finishDesk ? (
    <div className="@container h-full min-h-0" data-testid="finish-desk">
      <div className="finish-desk">
        <div className="finish-desk-board" data-skin-pane>
          <CutStage onLeave={() => { void ws.leaveCut(); }} />
        </div>
        <div className="finish-desk-chat">{agentPane}</div>
      </div>
    </div>
  ) : talkDesk ? (
    <div className="relative h-full min-h-0 overflow-hidden" data-testid="create-desk" data-skin-pane>
      {agentPane}
    </div>
  ) : agentPane;

  return (
    <PresenceLayer input={presenceInput}>
    <RailSlotContext.Provider value={idling ? idolSlot : null}>
    <AppFrame
      overlay={(
        <>
          <a href="#main-stage" className="skip-to-content no-drag">{copy.app.skipToContent}</a>
          {!ws.booted ? <div className="absolute inset-0 z-30"><BootSkeleton /></div> : null}
          <CommandPalette
            open={ws.palette}
            threads={ws.threads}
            videoProjects={ws.videoProjects}
            onClose={() => ws.setPalette(false)}
            onNew={ws.onNew}
            onLab={ws.setLab}
            onOpenHarness={() => ws.openHarness("overview")}
            onSelectThread={ws.openThread}
            onSelectProject={(p) => { void ws.openVideoProject(p); }}
            onDiff={ws.refreshDiff}
            onAbout={async () => { ws.setAboutInfo(await api.about().catch(() => ({}))); ws.setAboutOpen(true); }}
            onQuit={ws.requestQuit}
            onRunSuite={() => { void ws.runHarbor("suite"); }}
            onRunCycle={() => { void ws.runEvolve(); }}
            onRunOverlay={() => { void ws.runOverlay(); }}
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
          <Sheet open={sheetRight} onOpenChange={(open) => {
            if (agent) ws.setInspector(open);
            else ws.setChatDock(open);
          }}>
            <SheetContent side="bottom" className="flex h-[min(78dvh,720px)] flex-col rounded-t-2xl p-0 pb-[env(safe-area-inset-bottom)]">
              <div className="flex h-11 shrink-0 items-center justify-between border-b border-border px-3">
                <SheetTitle className="text-[13px] font-medium">{agent ? copy.review.toggle : copy.dock.chat}</SheetTitle>
                <SheetClose className="grid size-9 place-items-center rounded-lg text-muted" aria-label={copy.rail.collapse}>
                  <X className="size-4" />
                </SheetClose>
              </div>
              <SheetDescription className="sr-only">{agent ? copy.review.toggle : copy.dock.toggle}</SheetDescription>
              <div className="min-h-0 flex-1 overflow-hidden [&>*]:h-full">
              {agent ? inspect : (
                <ChatDock
                  items={ws.items}
                  showThinking={!!ws.savedCfg.showThinking}
                  transcriptView={ws.savedCfg.transcriptView}
                  collapseTiming={ws.savedCfg.collapseTiming}
                  approvals={ws.approvals}
                  sessionId={ws.activeId}
                  running={ws.threadRunning}
                  turnStatus={ws.turnStatus}
                  sessionStats={ws.sessionStats}
                  draftKey={ws.draftKey}
                  disabled={!!ws.health.paused}
                  disabledReason={ws.health.paused ? copy.composer.paused : ""}
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
                  onImproveThisChat={() => ws.openHarness("propose")}
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
        <Panel id="stage" minSize="42" className="h-full min-h-0 min-w-0" style={{ overflow: "hidden" }}>
          <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            <PageHeader left={headerLeft} title={headerTitle} right={headerRight} macPad={mac && !showRail} />
            {ws.err ? (
              <div data-testid="app-error" className="mx-4 mb-1 flex items-start gap-3 rounded-xl bg-danger/10 px-3.5 py-2.5 text-[13px] text-danger">
                <span className="min-w-0 flex-1 whitespace-pre-wrap break-words">{ws.err}</span>
                <button type="button" className="ml-auto shrink-0 text-[11px] underline" onClick={() => ws.setErr("")}>{copy.app.dismiss}</button>
                <button type="button" className="shrink-0 text-[11px] underline" onClick={() => { ws.setErr(""); void ws.refresh(); }}>{copy.app.retry}</button>
              </div>
            ) : null}
            <MainColumn>
            {inspectOpen ? (
              <Group
                key={peekOpen ? "agent-peek" : three ? "agent-inspect" : "lab-dock"}
                className="h-full min-h-0 min-w-0 flex-1"
                style={{ height: "100%" }}
                orientation="horizontal"
                defaultLayout={peekOpen
                  ? { main: 40, peek: 40, inspect: 20 }
                  : { main: Math.max(54, 100 - innerInspect), inspect: innerInspect }}
                onLayoutChanged={(next, meta) => {
                  if (!meta.isUserInteraction || typeof next.inspect !== "number") return;
                  const inspectWindow = (next.inspect / 100) * stagePct;
                  setLayout((prev) => writeLayout({
                    inspect: inspectWindow,
                    main: Math.max(30, 100 - prev.rail - inspectWindow),
                  }));
                }}
              >
                <Panel id="main" minSize={peekOpen ? "26" : "32"} className="h-full min-h-0 min-w-0">
                  {staged ? videoBoard : workspace}
                </Panel>
                {peekOpen ? (
                  <>
                    <ResizeHandle />
                    <Panel id="peek" minSize="28" maxSize="52" className="h-full min-h-0 min-w-0">
                      <FilePeek
                        workspace={toolRoot}
                        path={reviewFile}
                        hunks={ws.hunks}
                        mode={ws.diffMode}
                        onMode={ws.setDiffMode}
                        onClose={() => setReviewFile("")}
                        onQuote={(text) => {
                          const cur = useUI.getState().drafts[ws.draftKey] || "";
                          useUI.getState().setDraft(ws.draftKey, cur ? `${cur}\n${text}` : text);
                        }}
                        onOpenPath={(rel) => {
                          const root = ws.active?.workspace || ws.savedCfg.workspace;
                          const path = joinWorkspace(root, rel);
                          if (path) void api.openInEditor(path);
                        }}
                      />
                    </Panel>
                  </>
                ) : null}
                <ResizeHandle />
                <Panel id="inspect" minSize="14" maxSize={peekOpen ? "28" : "48"} className="h-full min-h-0 min-w-0">
                  <div className="glass-chrome h-full min-h-0 overflow-hidden" data-skin-pane>
                    {three ? inspect : (
                      <ChatDock
                        items={ws.items}
                        showThinking={!!ws.savedCfg.showThinking}
                        transcriptView={ws.savedCfg.transcriptView}
                        collapseTiming={ws.savedCfg.collapseTiming}
                        approvals={ws.approvals}
                        sessionId={ws.activeId}
                        running={ws.threadRunning}
                        turnStatus={ws.turnStatus}
                        sessionStats={ws.sessionStats}
                        draftKey={ws.draftKey}
                        disabled={!!ws.health.paused}
                        disabledReason={ws.health.paused ? copy.composer.paused : ""}
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
                        onImproveThisChat={() => ws.openHarness("propose")}
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
            ) : staged ? (
              videoBoard
            ) : (
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{workspace}</div>
            )}
            </MainColumn>
          </div>
        </Panel>
      </Group>
      {!showRail && !popout && !settings ? (
        railNarrow ? (
          overlayRail ? (
            <>
              <button
                type="button"
                className="absolute inset-0 z-20 bg-black/40"
                aria-label={copy.rail.collapse}
                onClick={() => ws.setSidebarHover(false)}
              />
              <div className="absolute inset-y-0 left-0 z-30 w-[min(20.5rem,92%)] pt-[env(safe-area-inset-top)]">
                <div className="h-full">{rail}</div>
              </div>
            </>
          ) : null
        ) : (
          <div
            className={overlayRail
              ? "absolute inset-y-0 left-0 z-20 w-[min(288px,90%)]"
              : "absolute inset-y-0 left-0 z-20 w-2"}
            onMouseEnter={() => ws.setSidebarHover(true)}
            onMouseLeave={() => ws.setSidebarHover(false)}
          >
            {overlayRail ? <div className="h-full">{rail}</div> : null}
          </div>
        )
      ) : null}
    </AppFrame>
    </RailSlotContext.Provider>
    </PresenceLayer>
  );
}

/**
 * 把工作站的配置、健康和插件操作交给设置页。写密钥时空字符串不调用 setAPIKey。
 * ws 读取 copy、savedCfg、health、plugins、logs、journal 等，调用 patchConfig，调用 refresh，调用 fail。
 * navInRail 写到 SettingsPage 的 navInRail。
 */
function SettingsSurface({ ws, navInRail }: { ws: ReturnType<typeof useWorkstation>; navInRail?: boolean }) {
  const copy = ws.copy;
  return (
    <SettingsPage
      navInRail={navInRail}
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
        /** saveProvider 调用 trim。next 传给 ws.patchConfig。key 去掉首尾空白，为真时才继续，传给 api.setAPIKey。 */
        saveProvider: async (next, key) => {
          if (key.trim()) await api.setAPIKey(key.trim());
          await ws.patchConfig(next);
          await ws.refresh();
        },
        refresh: ws.refresh,
        onBrowse: () => api.pickFolder(),
        /** onStartMcp 调用 startMCP。name 传给 api.startMCP。command 传给 api.startMCP。args 传给 api.startMCP。 */
        onStartMcp: async (name, command, args) => { await api.startMCP(name, command, args); await ws.refresh(); },
        /** onStartMcpHttp 调用 startMCPHTTP。name 传给 api.startMCPHTTP。endpoint 传给 api.startMCPHTTP。 */
        onStartMcpHttp: async (name, endpoint) => { await api.startMCPHTTP(name, endpoint); await ws.refresh(); },
        /** onStopMcp 调用 stopMCP。name 传给 api.stopMCP。 */
        onStopMcp: async (name) => { await api.stopMCP(name); await ws.refresh(); },
        /** onReplaceMcp 调用 replaceMCP。servers 传给 api.replaceMCP。 */
        onReplaceMcp: async (servers) => { await api.replaceMCP(servers); await ws.refresh(); },
        /** onUnload 调用 unloadFiber。name 传给 api.unloadFiber。 */
        onUnload: async (name) => { await api.unloadFiber(name); await ws.refresh(); },
        onCheckUpdate: () => api.checkUpdate(),
        onUpdateStatus: () => api.updateStatus(),
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
