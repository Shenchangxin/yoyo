/**
 * 拥有：会话左侧的一天。今天、任务、记忆、Buddy。等人的事排在最前，计数等于它指向的那一列。晨间简报在这里审计资产，点选留在简报上。
 * 点开任务留在这一栏，对话不换页，也可以放到中间去聊。记忆写错的在夹里改掉或忘掉。连接在设置里。
 * Buddy 是这台电脑上能接任务的 agent 应用，页面本身在 features/buddy/BuddyPane.tsx，这里只管页签和它的数字。
 * 不拥有：工单怎么执行，也不拥有检查器里的差异。执行在 Go 的 superagent，审查页由 App 传入。开口门怎么算在 Go 的 initiative。
 */

import { useCallback, useEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { motion } from "motion/react";
import { Activity, ArrowRight, Brain, Calendar, Check, CheckCheck, ChevronDown, ChevronLeft, ChevronRight, Clock, Coffee, Compass, Folder, FolderOpen, GitBranch, Inbox, Info, ListChecks, Mail, MapPin, Megaphone, MessageCircle, Monitor, Moon, Plus, Search, Send, ShieldCheck, SlidersHorizontal, Sparkles, Sun, type LucideIcon } from "lucide-react";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import { toast } from "sonner";
import { orchestratorAct, orchestratorDecide, orchestratorDeliver, orchestratorView, inboxDismiss, inboxRead, memoryEdit, memoryForget, memoryList, memoryWrite, personalAnswer, personalDecide, type DeskMute } from "../../lib/client";
import type { Attachment, FileHit, Notice, SkillInfo } from "../../lib/protocol";
import { dateLocale, useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { triggerText, type Rule, type Ticket } from "./desk-model";
import { BRIEF_EVENT, DESK_EVENT, briefEditions, deskMutes, latestInitiative, refreshDesk, runDeskFeed, sendFeedback, type BriefEdition } from "./desk-feed";
import { TaskRead } from "./TaskPage";
import { agoLabel, TaskRow } from "./TaskRow";
import { GroupRow } from "./GroupRow";
import { GroupPage } from "./GroupPage";
import { useBuddy } from "../buddy/buddy-store";
import { attentionCount } from "../buddy/buddy-model";
import { BuddyPane } from "../buddy/BuddyPane";
import { useUI } from "../../lib/store";
import { onFocusBrief, onOpenDesk, refreshQueue, useNeeds, type BriefTarget, type DeskTarget } from "./needs";
import { needItems, type NeedItem } from "./needs-model";
import { actionLabel, briefLayout, briefProgress, cardHandled, checkedCount, itemKey, shortKicker } from "./brief-board";
import { briefDay, briefPrompt, briefStorageKey, BRIEF_PREFS_KEY, defaultBriefPrefs, dueDelta, filterItems, groupRows, itemSections, memoryFolderRows, memoryForgettable, memoryOrigin, memoryPast, memoryPortrait, memoryTouched, offerSlot, parseBriefPrefs, refStatus, serializeStoredBrief, tasksBadge, unseenSuggestions, wouldAccept, type BriefCard, type BriefPhrases, type BriefSlot, type BriefPrefs, type MemoryFolder, type MemoryRow, type MorningBrief, type TaskItem, type TaskSection } from "./work-model";
import { BRIEF_PREFS_EVENT, type ProactiveLevel } from "../../lib/proactive";

type Tab = "now" | "tasks" | "memory" | "buddy";

/** 「需要你」先列这么多条，其余点开再看。 */
const NEEDS_SHOWN = 5;

/** seenBuddyPulse 是今天栏已经应过的那一次「打开 Buddy」。栏不在场时点的，等它挂上再翻过去。 */
let seenBuddyPulse = 0;

/** WorkPane 在会话桌面上常驻。reviewPulse 增加时切到审查。review 是检查器。 */
export function WorkPane(props: {
  review: ReactNode;
  reviewOpen: boolean;
  reviewPulse: number;
  onReview: (open: boolean) => void;
  onNewTask: (text?: string) => void;
  onOpenSettings: () => void;
  onOpenProactive?: (section?: string) => void;
  onDraft: (text: string) => void;
  openTaskId: string;
  onOpenTask: (id: string) => void;
  onFocusTask?: (id: string) => void;
  onFocusGroup?: (id: string) => void;
  onResolve?: (id: string, decision: string) => void;
  onOpenThread?: (id: string) => void;
  notices?: Notice[];
  onNotice?: (n: Notice) => void;
  onClearNotices?: () => void;
  paused?: boolean;
  onPause?: () => void;
  level?: ProactiveLevel;
  onLevel?: (level: ProactiveLevel) => void;
  onBriefTimes?: (times: BriefPrefs) => void;
  skills?: SkillInfo[];
  files?: FileHit[];
  workspace?: string;
  onSearchFiles?: (q: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
  onClipboard?: () => Promise<string>;
  onScreenshot?: () => Promise<string>;
}) {
  const copy = useCopy();
  const [tab, setTab] = useState<Tab>("now");
  // groupId 是任务栏里打开的任务组。从任务组点进成员时留着，读完工单返回时回到任务组。
  const [groupId, setGroupId] = useState("");
  const view = useNeeds((s) => s.view);
  const queue = useNeeds((s) => s.queue);
  const [err, setErr] = useState("");
  const [memory, setMemory] = useState<MemoryRow[]>([]);
  const [desk, setDesk] = useState(latestInitiative);
  const [focusEdition, setFocusEdition] = useState<{ target: BriefTarget | null; nonce: number }>({ target: null, nonce: 0 });
  useEffect(() => {
    const pull = () => setDesk(latestInitiative());
    window.addEventListener(BRIEF_EVENT, pull);
    window.addEventListener(DESK_EVENT, pull);
    return () => {
      window.removeEventListener(BRIEF_EVENT, pull);
      window.removeEventListener(DESK_EVENT, pull);
    };
  }, []);
  const load = useCallback(async () => {
    try {
      await orchestratorDeliver().catch(() => undefined);
      useNeeds.getState().setView(await orchestratorView());
      setErr("");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void load();
    void refreshQueue();
    const timer = window.setInterval(() => { void load(); }, 2000);
    const queueTimer = window.setInterval(() => { void refreshQueue(); }, 10_000);
    return () => {
      window.clearInterval(timer);
      window.clearInterval(queueTimer);
    };
  }, [load]);

  useEffect(() => onFocusBrief((target) => {
    setTab("now");
    setFocusEdition((prev) => ({ target, nonce: prev.nonce + 1 }));
    props.onReview(false);
  }), []);

  useEffect(() => {
    const go = (target: DeskTarget) => {
      props.onReview(false);
      setTab(target.tab);
      if (target.tab !== "tasks") return;
      if ("group" in target) {
        setGroupId(target.group);
        props.onOpenTask("");
      } else {
        setGroupId("");
        props.onOpenTask(target.ticket);
      }
    };
    const state = useNeeds.getState();
    state.setPaneMounted(true);
    if (state.pendingDesk) {
      go(state.pendingDesk);
      state.setPendingDesk(null);
    }
    const off = onOpenDesk(go);
    return () => {
      off();
      useNeeds.getState().setPaneMounted(false);
    };
  }, []);

  const pullMemory = useCallback(async () => {
    setMemory(await memoryList().catch(() => []) as MemoryRow[]);
  }, []);

  useEffect(() => {
    if (tab !== "memory") return;
    let stop = false;
    const run = () => {
      memoryList().then((rows) => { if (!stop) setMemory(rows as MemoryRow[]); }).catch(() => {});
    };
    run();
    const timer = window.setInterval(run, 10_000);
    return () => { stop = true; window.clearInterval(timer); };
  }, [tab]);

  useEffect(() => {
    if (props.reviewPulse > 0) props.onReview(true);
  }, [props.reviewPulse]);

  // 别处点了「去 Buddy」（@ 了一个还不能派的 agent、任务上的「去登录」、命令面板）：翻到 Buddy 页，审查先收起来。
  const buddyPulse = useUI((s) => s.buddyPulse);
  const buddyNeeds = useBuddy((s) => attentionCount(s.list));
  useEffect(() => {
    if (buddyPulse === seenBuddyPulse) return;
    seenBuddyPulse = buddyPulse;
    props.onReview(false);
    setTab("buddy");
  }, [buddyPulse]);

  function pick(next: Tab) {
    setTab(next);
    props.onReview(false);
  }

  const openProactive = (section?: string) => (props.onOpenProactive ? props.onOpenProactive(section) : props.onOpenSettings());
  const tickets = view?.tickets ?? [];
  const needs = needItems(view, queue);
  const sections = itemSections(groupRows(tickets, view?.groups ?? []));
  const taskCount = tasksBadge(tickets);
  const observed = view
    ? wouldAccept(view.decisions ?? []).map((row) => ({ id: row.id, text: triggerText(view, row.triggerId) || row.thread || row.triggerId, at: row.at || "" }))
    : [];
  const reading = tab === "tasks" && (!!props.openTaskId || !!groupId);
  const openGroup = groupId ? (view?.groups ?? []).find((row) => row.id === groupId) ?? null : null;
  const tabs: { id: Tab; label: string; count?: number; person?: boolean }[] = [
    { id: "now", label: copy.work.now, count: needs.length, person: needs.length > 0 },
    { id: "tasks", label: copy.work.tasks, count: taskCount.count, person: taskCount.person },
    { id: "memory", label: copy.work.memory },
    // Buddy 上的数字只数要人处理的：装好了却要登录、要配置、出错或离线的。
    { id: "buddy", label: copy.work.buddy, count: buddyNeeds, person: buddyNeeds > 0 },
  ];

  const reduced = useMotionReduced();
  if (props.reviewOpen) {
    return (
      <div data-testid="work-pane" className="rail-scroll-fade flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-transparent px-3 pb-16 pt-2 text-foreground">
        <button type="button" className="mb-2 self-start rounded-full border border-border bg-card px-3 py-1 text-ui-footnote text-muted shadow-[var(--shadow-card)] hover:bg-lift hover:text-foreground" onClick={() => props.onReview(false)}>
          {copy.work.back}
        </button>
        <div className="min-h-0 min-w-0 flex-1 overflow-hidden rounded-xl border border-border bg-background">{props.review}</div>
      </div>
    );
  }

  return (
    <div data-testid="work-pane" className="flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-transparent text-foreground">
      <div className="flex shrink-0 justify-center px-3 pb-1.5 pt-2">
        <div role="tablist" aria-label={copy.work.title} className="flex max-w-full flex-wrap items-center justify-center gap-0.5 rounded-full border border-border/60 bg-lift/75 p-1">
          {tabs.map((item) => {
            const on = tab === item.id;
            return (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={on}
                data-testid={`work-tab-${item.id}`}
                className={cn(
                  "relative flex h-7 items-center gap-1.5 rounded-full px-3 text-ui-body font-medium transition-colors duration-200 ease-[var(--ease-out)]",
                  on ? "text-foreground" : "text-muted hover:text-foreground",
                )}
                onClick={() => pick(item.id)}
              >
                {on ? (
                  <motion.span
                    layoutId="work-pane-tab-pill"
                    className="absolute inset-0 rounded-full border border-border/70 bg-card shadow-[var(--shadow-card)]"
                    transition={motionTransition(reduced, DURATION)}
                  />
                ) : null}
                <span className="relative z-10">{item.label}</span>
                {item.count ? <span className={cn("relative z-10 tabular-nums text-ui-caption", item.person ? "text-warning" : "text-muted")}>{item.count}</span> : null}
              </button>
            );
          })}
        </div>
      </div>
      <div className="rail-scroll-fade min-h-0 min-w-0 flex-1 overflow-hidden px-2">
        <div className={cn("h-full min-w-0", reading ? "overflow-hidden pb-14" : "overflow-y-auto overflow-x-hidden overscroll-contain pb-16")}>
          {err && tab !== "buddy" ? <p className="px-3 py-2 text-ui-footnote text-danger">{err}</p> : null}
          {!view && !err && tab !== "buddy" ? <p className="px-3 py-3 text-ui-body text-muted">{copy.work.loading}</p> : null}
          {tab === "buddy" ? <BuddyPane /> : null}
          {view && tab === "now" ? (
            <Now
              needs={needs}
              tickets={tickets}
              focusEdition={focusEdition}
              paused={!!props.paused}
              onPause={props.onPause}
              level={props.level || "normal"}
              onLevel={props.onLevel}
              notices={props.notices || []}
              onNotice={props.onNotice}
              onClearNotices={props.onClearNotices}
              onDecide={async (id, kind) => { await orchestratorDecide(id, kind); await load(); }}
              onDraft={props.onDraft}
              onResolve={(id, decision) => {
                props.onResolve?.(id, decision);
                window.setTimeout(() => { void refreshQueue(); }, 400);
              }}
              onOpenTask={(id) => { pick("tasks"); setGroupId(""); props.onOpenTask(id); }}
              onOpenThread={props.onOpenThread}
              onOpenSettings={() => openProactive()}
              onBriefTimes={props.onBriefTimes}
            />
          ) : null}
          {view && tab === "tasks" && props.openTaskId ? (
            <TaskRead
              ticket={(view.tickets ?? []).find((row) => row.id === props.openTaskId) ?? null}
              onBack={() => props.onOpenTask("")}
              onFocus={props.onFocusTask}
              onOpenGroup={(id) => { setGroupId(id); props.onOpenTask(""); }}
              onSent={() => { void load(); }}
              skills={props.skills}
              files={props.files}
              workspace={props.workspace}
              onSearchFiles={props.onSearchFiles}
              onPickFiles={props.onPickFiles}
              onClipboard={props.onClipboard}
              onScreenshot={props.onScreenshot}
            />
          ) : null}
          {view && tab === "tasks" && !props.openTaskId && groupId ? (
            <GroupPage
              group={openGroup}
              tickets={tickets}
              onBack={() => setGroupId("")}
              onOpenTask={(id) => props.onOpenTask(id)}
              onOpenMain={props.onOpenThread}
              onFocus={props.onFocusGroup ? (id) => { setGroupId(""); props.onFocusGroup?.(id); } : undefined}
              onSent={() => { void load(); }}
            />
          ) : null}
          {view && tab === "tasks" && !props.openTaskId && !groupId ? (
            <Tasks
              sections={sections}
              observed={observed}
              selected={props.openTaskId}
              onSelect={(id) => props.onOpenTask(id)}
              onOpenGroup={setGroupId}
              onNewTask={props.onNewTask}
              held={(desk?.signals || []).filter((signal) => signal.held)}
              notes={(desk?.open || []).filter((item) => item.class === "schedule")}
              rules={view.rules ?? []}
              mode={view.mode}
              onRules={async (rules) => { await orchestratorAct("rules", "", "", JSON.stringify(rules)); await load(); }}
              onMoreSchedules={props.onOpenProactive ? () => openProactive("personal-jobs") : undefined}
            />
          ) : null}
          {tab === "memory" ? (
            <>
              <Memory
                rows={memory}
                onAdd={async (text) => {
                  await memoryWrite("profile", text);
                  await pullMemory();
                }}
                onForget={async (id) => {
                  await memoryForget(id).catch(() => undefined);
                  await pullMemory();
                }}
                onEdit={async (id, text) => {
                  const saved = await memoryEdit(id, text).catch(() => ({ id: "" }));
                  await pullMemory();
                  return !!saved.id;
                }}
              />
              <p className="px-1 pb-4 text-ui-footnote leading-5 text-muted">
                {copy.work.connectionsInSettings}{" "}
                <button type="button" className="underline-offset-2 hover:underline" onClick={props.onOpenSettings}>{copy.work.openConnections}</button>
              </p>
            </>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function Now(props: {
  needs: NeedItem[];
  tickets: Ticket[];
  focusEdition: { target: BriefTarget | null; nonce: number };
  paused: boolean;
  onPause?: () => void;
  level: ProactiveLevel;
  onLevel?: (level: ProactiveLevel) => void;
  notices: Notice[];
  onNotice?: (n: Notice) => void;
  onClearNotices?: () => void;
  onDecide: (id: string, kind: string) => Promise<void>;
  onDraft: (text: string) => void;
  onResolve: (id: string, decision: string) => void;
  onOpenTask: (id: string) => void;
  onOpenThread?: (id: string) => void;
  onOpenSettings: () => void;
  onBriefTimes?: (times: BriefPrefs) => void;
}) {
  const copy = useCopy();
  const d = copy.desk;
  const [editions, setEditions] = useState<BriefEdition[]>([]);
  const [prefs, setPrefs] = useState<BriefPrefs>(() => readPrefs());
  const [waiting, setWaiting] = useState(false);
  const [prefsOpen, setPrefsOpen] = useState(false);
  const [picked, setPicked] = useState("");
  const [toggled, setToggled] = useState<Record<string, boolean>>({});
  const busy = useRef(false);
  const today = new Date().toLocaleDateString(dateLocale(), { month: "long", day: "numeric", weekday: "long" });
  const dueSlot: BriefSlot | null = offerSlot(prefs);
  const dueMissing = !!dueSlot && !editions.some((row) => row.slot === dueSlot && row.day === briefDay());
  const heading = dueSlot === "evening" ? copy.work.briefEvening : copy.work.briefMorning;
  useEffect(() => {
    let alive = true;
    const reload = () => {
      void briefEditions().then((rows) => {
        if (alive) setEditions(rows);
      });
    };
    const rereadPrefs = () => setPrefs(readPrefs());
    reload();
    window.addEventListener(BRIEF_EVENT, reload);
    window.addEventListener(BRIEF_PREFS_EVENT, rereadPrefs);
    return () => {
      alive = false;
      window.removeEventListener(BRIEF_EVENT, reload);
      window.removeEventListener(BRIEF_PREFS_EVENT, rereadPrefs);
    };
  }, []);
  useEffect(() => {
    const target = props.focusEdition.target;
    if (!target) return;
    let alive = true;
    void briefEditions().then((rows) => {
      if (!alive) return;
      const hit = rows.find((row) => (target.id && row.id === target.id) || (row.slot === target.slot && row.day === target.day));
      const brief = hit ? null : payloadBrief(target.brief);
      const extra: BriefEdition | null = brief
        ? { id: target.id || `brief:${target.slot}:${target.day}`, slot: target.slot === "evening" ? "evening" : "morning", day: target.day, chip: target.chip, at: "", brief }
        : null;
      setEditions(extra ? [extra, ...rows] : rows);
      setPicked(hit?.id || extra?.id || "");
      setToggled({});
    });
    return () => { alive = false; };
  }, [props.focusEdition.nonce]);
  async function generate() {
    if (!dueMissing || busy.current) return;
    busy.current = true;
    setWaiting(true);
    try {
      if (!dueSlot) return;
      await runDeskFeed(briefPhrases(copy.work), dueSlot);
      const rows = await briefEditions();
      setEditions(rows);
      setPicked(rows[0]?.id || "");
    } finally {
      busy.current = false;
      setWaiting(false);
    }
  }
  function savePrefs(next: BriefPrefs) {
    setPrefs(next);
    try {
      localStorage.setItem(BRIEF_PREFS_KEY, JSON.stringify(next));
    } catch {
      /* 偏好写不进去时，这次打开里仍然用。 */
    }
    props.onBriefTimes?.(next);
  }
  function choose(edition: BriefEdition, id: string, value: string) {
    setEditions((prev) => prev.map((row) => {
      if (row.id !== edition.id) return row;
      const next = { ...row.brief, picked: { ...row.brief.picked, [id]: value } };
      if (row.day === briefDay()) {
        try {
          localStorage.setItem(briefStorageKey(row.slot), serializeStoredBrief(row.day, next));
        } catch {
          /* 点选写不进去时，这一屏上仍然留着。 */
        }
      }
      return { ...row, brief: next };
    }));
  }
  if (prefsOpen) {
    return (
      <BriefPrefsView
        prefs={prefs}
        onChange={savePrefs}
        onBack={() => setPrefsOpen(false)}
        onOpenSettings={props.onOpenSettings}
      />
    );
  }
  function useCard(edition: BriefEdition, card: BriefCard, which: "primary" | "secondary") {
    const text = briefPrompt(card, which);
    choose(edition, card.id, which);
    if (text.trim()) props.onDraft(text.trim());
  }
  const shown = editions.find((row) => row.id === picked) || editions[0];
  const caption = shown ? editionCaption(shown, copy.work.briefMorning, copy.work.briefEvening, copy.work.briefYesterday) : null;
  function jump(id: string) {
    setToggled((prev) => ({ ...prev, [id]: false }));
    requestAnimationFrame(() => {
      const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      document.getElementById(briefCardAnchor(id))?.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
    });
  }
  return (
    <div className="relative min-w-0 px-3 pb-8 pt-3">
      <DeskMenu
        paused={props.paused}
        onPause={props.onPause}
        level={props.level}
        onLevel={props.onLevel}
        notices={props.notices}
        onNotice={props.onNotice}
        onClearNotices={props.onClearNotices}
        onPrefs={() => setPrefsOpen(true)}
      />
      <p className="flex h-8 items-center justify-center text-ui-footnote text-muted">{today}</p>
      {props.needs.length > 0 ? (
        <NeedsYou
          items={props.needs}
          onDecide={props.onDecide}
          onDraft={props.onDraft}
          onResolve={props.onResolve}
          onOpenTask={props.onOpenTask}
          onOpenThread={props.onOpenThread}
          onOpenSettings={props.onOpenSettings}
        />
      ) : null}
      {editions.length === 0 && !waiting ? (
        <div className="mt-8 flex flex-col items-center gap-3">
          {dueMissing ? (
            <button type="button" className="h-9 rounded-full bg-foreground px-5 text-ui-body font-medium text-background" onClick={() => { void generate(); }}>
              {heading}
            </button>
          ) : null}
          <p className="max-w-64 text-center text-ui-footnote leading-5 text-muted">{dueMissing ? copy.work.briefReady : copy.work.briefStays}</p>
        </div>
      ) : null}
      {waiting ? <p className="py-10 text-center text-ui-body-l text-accent">{copy.work.briefRunning}</p> : null}
      {shown && caption ? (
        <div className="mt-4 space-y-3" data-testid="brief-edition">
          <BriefMasthead
            edition={shown}
            editions={editions}
            caption={caption}
            onPick={(id) => { setPicked(id); setToggled({}); }}
            onJump={jump}
          />
          <BriefStack
            brief={shown.brief}
            onUse={(card, which) => useCard(shown, card, which)}
            onMark={(key, value) => choose(shown, key, value)}
            onDraft={props.onDraft}
            filledLabel={copy.work.briefFilled}
            source={d.briefWritten.replace("{slot}", caption.title).replace("{time}", caption.clock)}
            tickets={props.tickets}
            onOpenTicket={props.onOpenTask}
            toggled={toggled}
            onToggle={(id, collapsed) => setToggled((prev) => ({ ...prev, [id]: collapsed }))}
          />
        </div>
      ) : null}
      {dueMissing && editions.length > 0 && !waiting ? (
        <div className="mt-6 flex justify-center">
          <button type="button" className="h-9 rounded-full border border-border bg-card px-4 text-ui-body font-medium shadow-[var(--shadow-card)] hover:bg-lift" onClick={() => { void generate(); }}>
            {heading}
          </button>
        </div>
      ) : null}
    </div>
  );
}

/** DeskMenu 是今天右上角那颗圆钮：主动程度、暂停、简报时间和安静时段，以及最近动态。 */
function DeskMenu(props: {
  paused: boolean;
  onPause?: () => void;
  level: ProactiveLevel;
  onLevel?: (level: ProactiveLevel) => void;
  notices: Notice[];
  onNotice?: (n: Notice) => void;
  onClearNotices?: () => void;
  onPrefs: () => void;
}) {
  const copy = useCopy();
  const d = copy.desk;
  const levels: { id: ProactiveLevel; label: string }[] = [
    { id: "less", label: d.levelLess },
    { id: "normal", label: d.levelNormal },
    { id: "more", label: d.levelMore },
  ];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="absolute right-3 top-3 grid size-8 place-items-center rounded-full border border-border bg-card text-muted shadow-[var(--shadow-card)] transition-colors hover:bg-lift hover:text-foreground"
          aria-label={d.menu}
          data-testid="desk-menu"
        >
          <SlidersHorizontal className="size-4" />
          {props.notices.length ? <span className="absolute right-1 top-1 size-1.5 rounded-full bg-accent" aria-hidden /> : null}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel>{d.level}</DropdownMenuLabel>
        {levels.map((item) => (
          <DropdownMenuItem key={item.id} onSelect={() => props.onLevel?.(item.id)}>
            <span className="flex-1">{item.label}</span>
            {props.level === item.id ? <Check className="size-3.5 text-muted" /> : null}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        {props.onPause ? (
          <DropdownMenuItem onSelect={() => props.onPause?.()}>{props.paused ? d.resume : d.pause}</DropdownMenuItem>
        ) : null}
        <DropdownMenuItem onSelect={props.onPrefs}>{d.prefs}</DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>{d.activity}</DropdownMenuLabel>
        {props.notices.length ? (
          <>
            {props.notices.slice(0, 6).map((n) => (
              <DropdownMenuItem key={n.id} onSelect={() => props.onNotice?.(n)} className="flex-col items-start gap-0">
                <span className="w-full truncate">{n.title}</span>
                {n.body ? <span className="w-full truncate text-ui-caption text-muted">{n.body}</span> : null}
              </DropdownMenuItem>
            ))}
            {props.onClearNotices ? <DropdownMenuItem onSelect={() => props.onClearNotices?.()} className="text-muted">{d.activityClear}</DropdownMenuItem> : null}
          </>
        ) : (
          <p className="px-2 py-1.5 text-ui-footnote text-muted">{d.activityEmpty}</p>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** NeedsYou 把所有要人决定的事排在今天最上面：日程、审批、任务提问、来信、待发送、个人事务、收件。数字等于这里的条数。 */
function NeedsYou(props: {
  items: NeedItem[];
  onDecide: (id: string, kind: string) => Promise<void>;
  onDraft: (text: string) => void;
  onResolve: (id: string, decision: string) => void;
  onOpenTask: (id: string) => void;
  onOpenThread?: (id: string) => void;
  onOpenSettings: () => void;
}) {
  const copy = useCopy();
  const [all, setAll] = useState(false);
  const shown = all ? props.items : props.items.slice(0, NEEDS_SHOWN);
  const rest = props.items.length - shown.length;
  return (
    <section className="day-card mt-4 px-4 py-3" data-testid="needs-you">
      <h2 className="flex items-center gap-1.5 text-ui-body font-medium">
        {copy.work.sectionPerson}
        <span className="tabular-nums text-ui-caption text-warning">{props.items.length}</span>
      </h2>
      <ul className="mt-1 divide-y divide-border/60">
        {shown.map((item) => <NeedRow key={item.id} {...props} item={item} />)}
      </ul>
      {rest > 0 ? (
        <button type="button" className="pt-1 text-ui-footnote text-muted hover:text-foreground" onClick={() => setAll(true)}>
          {copy.desk.needsMore.replace("{n}", String(rest))}
        </button>
      ) : null}
    </section>
  );
}

/** NeedRow 按种类写一句话，动作就在这一行。改动外部的动作只出现在审批和发送上。 */
function NeedRow(props: {
  item: NeedItem;
  onDecide: (id: string, kind: string) => Promise<void>;
  onDraft: (text: string) => void;
  onResolve: (id: string, decision: string) => void;
  onOpenTask: (id: string) => void;
  onOpenThread?: (id: string) => void;
  onOpenSettings: () => void;
}) {
  const copy = useCopy();
  const d = copy.desk;
  const item = props.item;
  const when = item.at ? agoLabel(item.at, copy.work) : "";
  switch (item.kind) {
    case "due": {
      const delta = dueDelta(item.due || "", new Date());
      const label = delta === null ? "" : delta < 0 ? copy.work.auditOverdue : delta === 0 ? copy.work.auditDueToday : copy.work.dueIn.replace("{n}", String(delta));
      return (
        <li className="flex items-start gap-3 py-2.5" data-testid="need-due">
          <Calendar className="mt-0.5 size-3.5 shrink-0 text-muted" aria-hidden />
          <div className="min-w-0 flex-1">
            <p className="line-clamp-2 text-ui-body leading-5">{item.title}</p>
            {label ? <p className={cn("mt-0.5 text-ui-footnote", delta !== null && delta < 0 ? "text-warning" : "text-muted")}>{label}</p> : null}
          </div>
          <TextButton quiet onClick={() => props.onDraft(copy.work.dueTask.replace("{title}", item.title).replace("{when}", label || item.due || ""))}>{copy.work.duePrepare}</TextButton>
        </li>
      );
    }
    case "approval":
      return (
        <li className="py-2.5" data-testid="need-approval">
          <p className="flex items-start gap-2 text-ui-body leading-5">
            <ShieldCheck className="mt-0.5 size-3.5 shrink-0 text-warning" aria-hidden />
            <span className="min-w-0">{d.needApproval.replace("{action}", item.title)}</span>
          </p>
          {item.detail ? <p className="mt-1.5 truncate rounded-lg border border-border/60 bg-lift/55 px-2.5 py-1 font-mono text-ui-footnote text-foreground/85 [font-variant-ligatures:none]">{item.detail}</p> : null}
          {when ? <p className="mt-1 text-ui-footnote text-muted">{when}</p> : null}
          <div className="mt-2 flex gap-2">
            <TextButton onClick={() => props.onResolve(item.offerId || "", "once")}>{d.allowOnce}</TextButton>
            <TextButton quiet onClick={() => props.onResolve(item.offerId || "", "deny")}>{d.deny}</TextButton>
            {item.sessionId && props.onOpenThread ? <TextButton quiet onClick={() => props.onOpenThread?.(item.sessionId || "")}>{d.open}</TextButton> : null}
          </div>
        </li>
      );
    case "question":
    case "review":
    case "failed":
      return (
        <li className="flex items-start gap-3 py-2.5" data-testid="need-task">
          <span className={cn("mt-[7px] size-1.5 shrink-0 rounded-full", item.kind === "failed" ? "bg-danger" : "bg-warning")} aria-hidden />
          <button type="button" className="min-w-0 flex-1 text-left" onClick={() => props.onOpenTask(item.ticketId || "")}>
            <span className="block truncate text-ui-body font-medium leading-5">{item.title}</span>
            <span className="mt-0.5 line-clamp-2 block text-ui-footnote leading-5 text-muted">
              {item.detail || (item.kind === "review" ? d.needReview : item.kind === "failed" ? d.needFailed : d.needQuestion)}
            </span>
          </button>
          <TextButton quiet={item.kind !== "question"} onClick={() => props.onOpenTask(item.ticketId || "")}>
            {item.kind === "question" ? d.answer : d.look}
          </TextButton>
        </li>
      );
    case "incoming":
      return (
        <li className="py-2.5" data-testid="need-incoming">
          <p className="line-clamp-2 text-ui-body leading-5">{item.title}</p>
          {item.detail || when ? <p className="mt-0.5 text-ui-footnote text-muted">{[item.detail, when].filter(Boolean).join(" · ")}</p> : null}
          <div className="mt-2 flex gap-2">
            <TextButton onClick={() => { void props.onDecide(item.triggerId || "", "accept"); }}>{copy.work.accept}</TextButton>
            <TextButton quiet onClick={() => { void props.onDecide(item.triggerId || "", "ignore"); }}>{copy.work.ignore}</TextButton>
          </div>
        </li>
      );
    case "draft":
      return (
        <li className="flex items-start gap-3 py-2.5" data-testid="need-draft">
          <Mail className="mt-0.5 size-3.5 shrink-0 text-muted" aria-hidden />
          <div className="min-w-0 flex-1">
            <p className="truncate text-ui-body leading-5">{item.title}</p>
            <p className="mt-0.5 truncate text-ui-footnote text-muted">{[item.detail, d.needDraft].filter(Boolean).join(" · ")}</p>
          </div>
          <TextButton
            quiet
            onClick={() => props.onDraft(d.draftPrompt.replace("{id}", item.draftId || "").replace("{to}", item.title).replace("{subject}", item.detail || item.title))}
          >
            {d.confirm}
          </TextButton>
        </li>
      );
    case "proposal":
      return (
        <li className="py-2.5" data-testid="need-proposal">
          <p className="flex items-start gap-2 text-ui-body leading-5">
            <Send className="mt-0.5 size-3.5 shrink-0 text-warning" aria-hidden />
            <span className="min-w-0">{d.needProposal.replace("{title}", item.title)}</span>
          </p>
          {item.detail || when ? <p className="mt-0.5 text-ui-footnote text-muted">{[item.detail, when].filter(Boolean).join(" · ")}</p> : null}
          <div className="mt-2 flex gap-2">
            <TextButton onClick={() => { void personalDecide(item.proposalId || "", item.hash || "", true).then(() => refreshQueue()).catch((e) => toast.error(String(e))); }}>{d.send}</TextButton>
            <TextButton quiet onClick={() => { void personalDecide(item.proposalId || "", item.hash || "", false).then(() => refreshQueue()).catch((e) => toast.error(String(e))); }}>{d.dontSend}</TextButton>
          </div>
        </li>
      );
    case "personal":
      return <PersonalNeedRow item={item} when={when} onOpenSettings={props.onOpenSettings} />;
    case "inbox":
      return (
        <li className="py-2.5" data-testid="need-inbox">
          <p className="flex items-start gap-2 text-ui-body leading-5">
            <Inbox className="mt-0.5 size-3.5 shrink-0 text-muted" aria-hidden />
            <span className="min-w-0 line-clamp-2">{item.title}</span>
          </p>
          {item.detail ? <p className="mt-0.5 line-clamp-2 text-ui-footnote leading-5 text-muted">{item.detail}</p> : null}
          <div className="mt-2 flex gap-2">
            {item.sessionId && props.onOpenThread ? (
              <TextButton onClick={() => {
                void inboxRead(item.inboxId || "");
                props.onOpenThread?.(item.sessionId || "");
              }}>{d.open}</TextButton>
            ) : null}
            <TextButton quiet onClick={() => { void inboxDismiss(item.inboxId || "").then(() => refreshQueue()); }}>{d.gotIt}</TextButton>
          </div>
        </li>
      );
    default:
      return null;
  }
}

/** PersonalNeedRow 是在等你的个人事务。等回答的就地展开一行输入；等批准的去设置里看全貌。 */
function PersonalNeedRow(props: { item: NeedItem; when: string; onOpenSettings: () => void }) {
  const copy = useCopy();
  const d = copy.desk;
  const item = props.item;
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const answerable = item.status === "waiting_input";
  async function submit() {
    const answer = text.trim();
    if (!answer || busy) return;
    setBusy(true);
    try {
      await personalAnswer(item.personalId || "", answer);
      setText("");
      setOpen(false);
      await refreshQueue();
    } catch (e) {
      toast.error(String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <li className="py-2.5" data-testid="need-personal">
      <div className="flex items-start gap-3">
        <span className="mt-[7px] size-1.5 shrink-0 rounded-full bg-warning" aria-hidden />
        <div className="min-w-0 flex-1">
          <p className="truncate text-ui-body font-medium leading-5">{item.title}</p>
          <p className="mt-0.5 line-clamp-2 text-ui-footnote leading-5 text-muted">{item.detail || d.needPersonal}</p>
          <p className="mt-0.5 text-ui-caption text-muted/80">{[d.personalTag, props.when].filter(Boolean).join(" · ")}</p>
        </div>
        {answerable ? (
          <TextButton quiet={open} onClick={() => setOpen((on) => !on)}>{open ? copy.work.memoryEditCancel : d.answer}</TextButton>
        ) : (
          <TextButton quiet onClick={props.onOpenSettings}>{d.look}</TextButton>
        )}
      </div>
      {open ? (
        <form className="mt-2 flex items-center gap-2 pl-[18px]" onSubmit={(event) => { event.preventDefault(); void submit(); }}>
          <input
            value={text}
            autoFocus
            onChange={(event) => setText(event.target.value)}
            placeholder={d.answerPh}
            aria-label={d.answerPh}
            className="h-8 min-w-0 flex-1 rounded-full border border-border bg-lift/75 px-3 text-ui-body outline-none placeholder:text-muted focus-visible:border-accent/50"
          />
          <button type="submit" disabled={!text.trim() || busy} className="h-7 shrink-0 rounded-full bg-foreground px-3 text-ui-footnote font-medium text-background disabled:opacity-40">{d.answer}</button>
        </form>
      ) : null}
    </li>
  );
}

/** payloadBrief 读对话里简报消息带的那一份。最近的档案里找不到这一期时用它。 */
function payloadBrief(raw: string): MorningBrief | null {
  try {
    const body = JSON.parse(raw) as MorningBrief;
    if (!body || !Array.isArray(body.cards) || body.cards.length < 1) return null;
    return { cards: body.cards, picked: body.picked || {}, welcome: body.welcome === true, woven: body.woven === true };
  } catch {
    return null;
  }
}

function editionCaption(edition: BriefEdition, morning: string, evening: string, yesterday: string): { clock: string; title: string; when: string } {
  const title = edition.slot === "evening" ? evening : morning;
  const parsed = Date.parse(edition.at);
  const clock = Number.isFinite(parsed) ? new Date(parsed).toLocaleTimeString(dateLocale(), { hour: "2-digit", minute: "2-digit" }) : "";
  const today = briefDay();
  if (edition.day === today) return { clock, title, when: "" };
  const prior = new Date();
  prior.setDate(prior.getDate() - 1);
  if (edition.day === briefDay(prior)) return { clock, title, when: yesterday };
  const day = new Date(`${edition.day}T00:00:00`);
  const when = Number.isNaN(day.getTime()) ? edition.day : day.toLocaleDateString(dateLocale(), { month: "short", day: "numeric" });
  return { clock, title, when };
}

function readPrefs(): BriefPrefs {
  try {
    return parseBriefPrefs(localStorage.getItem(BRIEF_PREFS_KEY));
  } catch {
    return defaultBriefPrefs();
  }
}

/** cardClass 把一张卡落到开口门的类别上，反馈写到这一类。 */
function cardClass(card: BriefCard): string {
  if (card.id === "lines") return "line";
  if (card.id === "speak") return "reach";
  if (card.id === "around" || card.id === "place" || card.id === "tools" || card.id === "sense") return "poi";
  return "brief";
}

/** briefCardAnchor 是一张卡在页面上的 id，目录点过来时滚到这里。 */
function briefCardAnchor(id: string): string {
  return `brief-card-${id}`;
}

/** tint 是一块带颜色的底：字用这个颜色，底是它的淡版。 */
function tint(color: string, strength: number): CSSProperties {
  return { background: `color-mix(in srgb, ${color} ${strength}%, transparent)`, color };
}

/** cardLook 给每类卡一个图标和颜色，看板上一眼认出是哪一类。颜色只分类别，不分好坏。 */
function cardLook(card: BriefCard): { icon: LucideIcon; color: string } {
  switch (card.id) {
    case "situation": return { icon: ListChecks, color: "var(--warning)" };
    case "letter": return { icon: Mail, color: "var(--accent)" };
    case "lines": return { icon: GitBranch, color: "var(--accent)" };
    case "done": return { icon: CheckCheck, color: "var(--success)" };
    case "place": return { icon: FolderOpen, color: "var(--mark)" };
    case "tools": return { icon: Monitor, color: "var(--accent)" };
    case "direction": return { icon: Compass, color: "var(--accent)" };
    case "next": return { icon: ArrowRight, color: "var(--accent)" };
    case "kept": return { icon: Brain, color: "var(--mark)" };
    case "around": return { icon: MapPin, color: "var(--success)" };
    case "pause": return { icon: Coffee, color: "var(--warning)" };
    case "quiet": return { icon: Moon, color: "var(--muted)" };
    case "speak": return { icon: Megaphone, color: "var(--mark)" };
    case "sense": return { icon: Activity, color: "var(--accent)" };
    default: return { icon: Sparkles, color: "var(--accent)" };
  }
}

/** BriefMasthead 是看板的报头：哪一期、几点写好、写完不再改，处理到第几张，以及一排目录。点目录跳到那张卡。 */
function BriefMasthead(props: {
  edition: BriefEdition;
  editions: BriefEdition[];
  caption: { clock: string; title: string; when: string };
  onPick: (id: string) => void;
  onJump: (id: string) => void;
}) {
  const copy = useCopy();
  const d = copy.desk;
  const brief = props.edition.brief;
  const progress = briefProgress(brief);
  const pct = progress.total ? Math.round((progress.done / progress.total) * 100) : 0;
  const evening = props.edition.slot === "evening";
  const meta = [props.caption.when, props.caption.clock ? d.briefWrittenAt.replace("{time}", props.caption.clock) : "", d.briefFixed].filter(Boolean).join(" · ");
  return (
    <section className="day-card px-5 py-4" data-testid="brief-masthead">
      <div className="flex items-center gap-3">
        <span className="grid size-10 shrink-0 place-items-center rounded-full" style={tint(evening ? "var(--accent)" : "var(--warning)", 16)} aria-hidden>
          {evening ? <Moon className="size-[18px]" /> : <Sun className="size-[18px]" />}
        </span>
        <div className="min-w-0 flex-1">
          {props.editions.length > 1 ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button type="button" className="-ml-1.5 inline-flex max-w-full items-center gap-1 rounded-full px-1.5 text-ui-title font-semibold tracking-[-0.02em] hover:bg-lift/70" aria-label={d.briefPick} data-testid="brief-pick">
                  <span className="truncate">{props.caption.title}</span>
                  <ChevronDown className="size-4 shrink-0 text-muted" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-56">
                {props.editions.map((edition) => {
                  const c = editionCaption(edition, copy.work.briefMorning, copy.work.briefEvening, copy.work.briefYesterday);
                  return (
                    <DropdownMenuItem key={edition.id} onSelect={() => props.onPick(edition.id)}>
                      <span className="flex-1 truncate">{[c.when, c.title].filter(Boolean).join(" ")}</span>
                      <span className="tabular-nums text-ui-caption text-muted">{c.clock}</span>
                      {edition.id === props.edition.id ? <Check className="size-3.5 text-muted" /> : null}
                    </DropdownMenuItem>
                  );
                })}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <h2 className="truncate text-ui-title font-semibold tracking-[-0.02em]">{props.caption.title}</h2>
          )}
          <p className="mt-0.5 truncate text-ui-footnote text-muted">{meta}</p>
        </div>
      </div>
      {progress.total > 0 ? (
        <div className="mt-4" data-testid="brief-progress">
          <div className="flex items-baseline justify-between text-ui-footnote">
            <span className="text-muted">{d.briefHandled}</span>
            <span className="tabular-nums text-foreground/80">{progress.done}<span className="text-muted"> / {progress.total}</span></span>
          </div>
          <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-lift">
            <div className="h-full rounded-full bg-accent transition-[width] duration-500 ease-[var(--ease-out)]" style={{ width: `${pct}%` }} />
          </div>
        </div>
      ) : null}
      {brief.cards.length > 1 ? (
        <div className="mt-3.5 flex flex-wrap gap-1.5" data-testid="brief-index">
          {brief.cards.map((card) => {
            const done = cardHandled(card, brief.picked);
            return (
              <button
                key={card.id}
                type="button"
                className={cn("inline-flex h-7 max-w-full items-center gap-1.5 rounded-full border border-border/60 bg-lift/65 pl-2 pr-2.5 text-ui-footnote transition-colors hover:bg-lift", done ? "text-muted" : "text-foreground/85")}
                onClick={() => props.onJump(card.id)}
              >
                {done ? <Check className="size-3 shrink-0 text-success" /> : <span className="size-1.5 shrink-0 rounded-full" style={{ background: cardLook(card).color }} aria-hidden />}
                <span className="truncate">{shortKicker(card)}</span>
              </button>
            );
          })}
        </div>
      ) : null}
    </section>
  );
}

/** BriefStack 把一期简报摆成一列卡。卡上的内容写完不变；勾掉的条目、点过的按钮、收起的卡是你的标记。
 *  处理完的卡默认收成一行，点标题展开。toggled 和 onToggle 不传时自己记住收起的状态。 */
export function BriefStack(props: {
  brief: MorningBrief;
  onUse: (card: BriefCard, which: "primary" | "secondary") => void;
  filledLabel: string;
  source?: string;
  tickets?: Pick<Ticket, "id" | "stage">[];
  onOpenTicket?: (id: string) => void;
  onMark?: (key: string, value: string) => void;
  onDraft?: (text: string) => void;
  toggled?: Record<string, boolean>;
  onToggle?: (id: string, collapsed: boolean) => void;
}) {
  const [own, setOwn] = useState<Record<string, boolean>>({});
  const toggled = props.toggled ?? own;
  const toggle = props.onToggle ?? ((id: string, collapsed: boolean) => setOwn((prev) => ({ ...prev, [id]: collapsed })));
  // 刚处理的卡留在展开的样子，让人看见自己点了什么；只有打开前就处理完的卡才收成一行。
  const hold = (id: string) => {
    if (toggled[id] === undefined) toggle(id, false);
  };
  const mark = props.onMark;
  return (
    <div className="space-y-3">
      {props.brief.cards.map((card) => {
        const handled = cardHandled(card, props.brief.picked);
        return (
          <BriefCardView
            key={card.id}
            card={card}
            picked={props.brief.picked}
            handled={handled}
            collapsed={toggled[card.id] ?? handled}
            onToggle={(collapsed) => toggle(card.id, collapsed)}
            onUse={(target, which) => { hold(target.id); props.onUse(target, which); }}
            onMark={mark ? (key, value) => { hold(card.id); mark(key, value); } : undefined}
            onDraft={props.onDraft}
            filledLabel={props.filledLabel}
            source={props.source || ""}
            tickets={props.tickets}
            onOpenTicket={props.onOpenTicket}
          />
        );
      })}
    </div>
  );
}

function BriefCardView(props: {
  card: BriefCard;
  picked: Record<string, string>;
  handled: boolean;
  collapsed: boolean;
  onToggle: (collapsed: boolean) => void;
  onUse: (card: BriefCard, which: "primary" | "secondary") => void;
  onMark?: (key: string, value: string) => void;
  onDraft?: (text: string) => void;
  filledLabel: string;
  source: string;
  tickets?: Pick<Ticket, "id" | "stage">[];
  onOpenTicket?: (id: string) => void;
}) {
  const copy = useCopy();
  const d = copy.desk;
  const card = props.card;
  const chosen = props.picked[card.id] || "";
  const layout = briefLayout(card);
  const look = cardLook(card);
  const Icon = look.icon;
  const kinds: Record<string, string> = { ticket: copy.work.refTicket, date: copy.work.refDate, line: copy.work.refLine, memory: copy.work.refMemory };
  const primaryLabel = actionLabel(card.primary, card.task);
  const secondaryLabel = actionLabel(card.secondary, card.alt);
  const checks = layout === "checks" && card.items.length > 0;
  const ticked = checkedCount(card, props.picked);
  const refs = card.refs ?? [];
  const basis = card.lede || card.note || card.items[0]?.detail || "";
  const draft = props.onDraft;
  const ask = draft
    ? (title: string, detail: string) => draft(detail ? d.askItem.replace("{title}", title).replace("{detail}", detail) : d.askItemBare.replace("{title}", title))
    : null;
  const tones = card.id === "done"
    ? { moving: d.toneFinished, stalled: d.toneEmpty, quiet: d.toneQuiet }
    : { moving: d.toneMoving, stalled: d.toneStalled, quiet: d.toneQuiet };
  return (
    <article
      id={briefCardAnchor(card.id)}
      className="day-card min-w-0 scroll-mt-3 px-5 py-4"
      data-testid="brief-card"
      data-handled={props.handled ? "true" : undefined}
    >
      <div className="flex items-start gap-3">
        <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-full" style={tint(look.color, 15)} aria-hidden>
          <Icon className="size-4" />
        </span>
        <button type="button" className="min-w-0 flex-1 text-left" aria-expanded={!props.collapsed} onClick={() => props.onToggle(!props.collapsed)}>
          {card.kicker ? <span className="block text-ui-caption font-medium tracking-[0.04em] text-muted">{card.kicker}</span> : null}
          <span className={cn("mt-0.5 block text-ui-title font-semibold leading-6 tracking-[-0.02em] [text-wrap:pretty]", props.collapsed && props.handled && "text-foreground/65")}>{card.title}</span>
        </button>
        <div className="-mr-1.5 flex shrink-0 items-center gap-0.5">
          {checks ? <span className="px-1 text-ui-caption tabular-nums text-muted">{ticked}/{card.items.length}</span> : null}
          {props.handled ? (
            <span className="grid size-6 place-items-center rounded-full" style={tint("var(--success)", 16)} title={d.briefHandled}>
              <Check className="size-3.5" />
            </span>
          ) : null}
          <CardWhy card={card} basis={basis} source={props.source} />
        </div>
      </div>
      {props.collapsed ? (
        <button type="button" className="mt-2 flex w-full items-center gap-1 text-left text-ui-footnote text-muted hover:text-foreground" onClick={() => props.onToggle(false)}>
          <span className="min-w-0 truncate">{props.handled ? (chosen ? props.filledLabel : d.briefAllTicked) : card.lede || d.briefExpand}</span>
          <ChevronDown className="size-3.5 shrink-0" />
        </button>
      ) : (
        <>
          {card.lede ? <p className="mt-2 text-ui-body leading-6 text-muted">{card.lede}</p> : null}
          {card.mark && (layout === "checks" || layout === "evidence" || layout === "rows") ? <p className="mt-3 text-ui-footnote tabular-nums text-muted">{card.mark}</p> : null}
          {checks ? (
            <ul className="mt-2 space-y-0.5">
              {card.items.map((item, index) => {
                const key = itemKey(card.id, index);
                const done = !!props.picked[key];
                return (
                  <li key={`${item.title}-${index}`} className="group/item -mx-2 flex items-start gap-3 rounded-xl px-2 py-1.5 transition-colors hover:bg-lift/50">
                    <button
                      type="button"
                      role="checkbox"
                      aria-checked={done}
                      aria-label={item.title}
                      disabled={!props.onMark}
                      className={cn(
                        "mt-0.5 grid size-[18px] shrink-0 place-items-center rounded-full border transition-colors disabled:cursor-default",
                        done ? "border-transparent bg-success text-background" : "border-foreground/25 hover:border-foreground/50",
                      )}
                      onClick={() => props.onMark?.(key, done ? "" : "done")}
                    >
                      {done ? <Check className="size-3" strokeWidth={3} /> : null}
                    </button>
                    <div className={cn("min-w-0 flex-1 transition-opacity", done && "opacity-55")}>
                      <p className={cn("text-ui-body font-medium leading-5", done && "line-through decoration-foreground/30")}>{item.title}</p>
                      {item.detail ? <p className="mt-0.5 text-ui-footnote leading-5 text-muted">{item.detail}</p> : null}
                    </div>
                    {ask ? <AskButton onClick={() => ask(item.title, item.detail)} /> : null}
                  </li>
                );
              })}
            </ul>
          ) : null}
          {layout === "evidence" && card.items.length > 0 ? (
            <ol className="relative mt-3 space-y-3 pl-5">
              <span className="absolute bottom-2 left-[5px] top-2 w-px bg-border" aria-hidden />
              {card.items.map((item, index) => (
                <li key={`${item.title}-${index}`} className="group/item relative flex items-start gap-2">
                  <span className="absolute -left-5 top-[5px] size-[11px] rounded-full border-2 border-card" style={{ background: look.color }} aria-hidden />
                  <div className="min-w-0 flex-1">
                    <p className="text-ui-body font-medium leading-5">{item.title}</p>
                    {item.detail ? <p className="mt-0.5 text-ui-footnote leading-5 text-muted">{item.detail}</p> : null}
                  </div>
                  {ask ? <AskButton onClick={() => ask(item.title, item.detail)} /> : null}
                </li>
              ))}
            </ol>
          ) : null}
          {layout === "rows" && card.items.length > 0 ? (
            <ul className="mt-2 divide-y divide-border/50">
              {card.items.map((item, index) => {
                const label = item.tone === "moving" ? tones.moving : item.tone === "stalled" ? tones.stalled : item.tone === "quiet" ? tones.quiet : "";
                const color = item.tone === "moving" ? (card.id === "done" ? "var(--success)" : "var(--accent)") : item.tone === "stalled" ? "var(--warning)" : "var(--muted)";
                return (
                  <li key={`${item.title}-${index}`} className="group/item flex items-center gap-3 py-2.5">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-ui-body font-medium leading-5">{item.title}</p>
                      {item.detail ? <p className="mt-0.5 line-clamp-2 text-ui-footnote leading-5 text-muted">{item.detail}</p> : null}
                    </div>
                    {ask ? <AskButton onClick={() => ask(item.title, item.detail)} /> : null}
                    {label ? <span className="shrink-0 rounded-full px-2 py-0.5 text-ui-caption font-medium" style={tint(color, 14)}>{label}</span> : null}
                  </li>
                );
              })}
            </ul>
          ) : null}
          {layout === "stats" && card.stats && card.stats.length > 0 ? (
            <dl className="mt-3 grid grid-cols-2 gap-2">
              {card.stats.map((stat, index) => (
                <div key={stat.label} className="min-w-0 rounded-xl border border-border/60 bg-lift/45 px-3.5 py-3">
                  <dd className="text-ui-display font-semibold leading-7 tabular-nums tracking-[-0.03em]" style={index === 0 ? { color: look.color } : undefined}>{stat.value}</dd>
                  <dt className="mt-0.5 truncate text-ui-footnote text-muted">{stat.label}</dt>
                </div>
              ))}
            </dl>
          ) : null}
          {layout === "compare" && card.choices.length > 0 ? (
            <div className="mt-3 grid grid-cols-2 gap-2">
              {card.choices.slice(0, 2).map((choice, index) => {
                const which = index === 0 ? "primary" : "secondary";
                const selectable = !!(primaryLabel && secondaryLabel);
                const on = selectable && chosen === which;
                const body = (
                  <>
                    <p className="text-ui-body font-semibold leading-5">{choice.title}</p>
                    {choice.detail ? <p className="mt-1 whitespace-pre-line text-ui-footnote leading-5 text-muted">{choice.detail}</p> : null}
                    {on ? <p className="mt-2 inline-flex items-center gap-1 text-ui-caption font-medium text-accent"><Check className="size-3" />{d.briefChosen}</p> : null}
                  </>
                );
                return selectable ? (
                  <button
                    key={choice.id}
                    type="button"
                    aria-pressed={on}
                    className={cn("min-w-0 rounded-xl border px-3.5 py-3 text-left transition-colors", on ? "border-accent/45 bg-accent/10 text-foreground" : "border-border/60 bg-lift/40 hover:bg-lift")}
                    onClick={() => props.onUse(card, which)}
                  >
                    {body}
                  </button>
                ) : (
                  <div key={choice.id} className="min-w-0 rounded-xl border border-border/60 bg-lift/40 px-3.5 py-3">{body}</div>
                );
              })}
            </div>
          ) : null}
          {layout === "letter" ? (
            <div className={cn("mt-3 grid gap-2", card.items.length > 0 && card.choices.length > 0 ? "grid-cols-2" : "grid-cols-1")}>
              {card.items.length > 0 || card.choices.length === 0 ? (
                <div className="min-w-0 rounded-xl border border-border/60 bg-lift/45 px-3.5 py-3">
                  <p className="text-ui-caption font-medium text-muted">{card.leftLabel}</p>
                  <div className="mt-2 space-y-2">
                    {card.items.length === 0 ? <p className="text-ui-body text-muted">—</p> : card.items.map((item, index) => (
                      <div key={`${item.title}-${index}`}>
                        <p className="text-ui-footnote text-muted">{item.title}</p>
                        {item.detail ? <p className="text-ui-body font-medium leading-5">{item.detail}</p> : null}
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {card.choices.length > 0 ? (
                <div className="min-w-0 rounded-xl border border-dashed px-3.5 py-3" style={{ borderColor: "color-mix(in srgb, var(--warning) 45%, transparent)" }}>
                  <p className="text-ui-caption font-medium text-warning">{card.rightLabel}</p>
                  <div className="mt-2 space-y-2">
                    {card.choices.map((choice) => (
                      <div key={choice.id}>
                        <p className="text-ui-footnote text-muted">{choice.title}</p>
                        {choice.detail ? <p className="whitespace-pre-line text-ui-body font-medium leading-5">{choice.detail}</p> : null}
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}
          {layout === "chips" && card.items.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-1.5">
              {card.items.map((item) => ask ? (
                <button key={item.title} type="button" className="rounded-full border border-border/60 bg-lift/65 px-3 py-1 text-ui-footnote transition-colors hover:bg-lift" title={d.briefAsk} onClick={() => ask(item.title, item.detail)}>
                  {item.title}
                </button>
              ) : (
                <span key={item.title} className="rounded-full border border-border/60 bg-lift/65 px-3 py-1 text-ui-footnote">{item.title}</span>
              ))}
            </div>
          ) : null}
          {card.note ? <p className="mt-3 rounded-xl border border-border/50 bg-lift/45 px-3 py-2 text-ui-footnote leading-5 text-muted">{card.note}</p> : null}
          {refs.length > 0 ? (
            <div className="mt-3 flex flex-wrap items-center gap-1.5" data-testid="brief-refs">
              <span className="text-ui-caption text-muted/80">{copy.work.briefRefs}</span>
              {refs.map((ref) => {
                const ended = refStatus(ref, props.tickets ?? []) === "ended";
                const label = `${kinds[ref.kind] || ref.kind} · ${ref.label}${ended ? ` · ${copy.work.refEnded}` : ""}`;
                const open = ref.kind === "ticket" && ref.target && props.onOpenTicket ? () => props.onOpenTicket?.(ref.target || "") : null;
                return open ? (
                  <button key={ref.id} type="button" className={cn("h-6 max-w-full truncate rounded-full border border-border/60 bg-lift/65 px-2.5 text-ui-caption hover:bg-lift hover:text-foreground", ended ? "text-muted" : "text-foreground/85")} onClick={open}>
                    {label}
                  </button>
                ) : (
                  <span key={ref.id} className="inline-flex h-6 max-w-full items-center truncate rounded-full border border-border/50 bg-lift/45 px-2.5 text-ui-caption text-muted">{label}</span>
                );
              })}
            </div>
          ) : null}
          {primaryLabel || secondaryLabel ? (
            <div className="mt-4 flex flex-wrap items-center gap-2">
              {primaryLabel ? (
                <button type="button" className={cn("h-8 rounded-full px-3.5 text-ui-body font-medium transition-colors", chosen === "secondary" ? "border border-border bg-lift/75 text-foreground hover:bg-lift" : "bg-foreground text-background hover:opacity-[0.92]")} onClick={() => props.onUse(card, "primary")}>
                  {primaryLabel}
                </button>
              ) : null}
              {secondaryLabel ? (
                <button type="button" className={cn("h-8 rounded-full px-3.5 text-ui-body font-medium transition-colors", chosen === "secondary" ? "bg-foreground text-background hover:opacity-[0.92]" : "border border-border bg-lift/75 text-foreground hover:bg-lift")} onClick={() => props.onUse(card, "secondary")}>
                  {secondaryLabel}
                </button>
              ) : null}
              {chosen ? (
                <span className="ml-1 inline-flex items-center gap-1.5 text-ui-footnote text-muted">
                  <Check className="size-3.5 text-success" />
                  {props.filledLabel}
                  {props.onMark ? <button type="button" className="text-foreground/80 underline-offset-2 hover:underline" onClick={() => props.onMark?.(card.id, "")}>{d.briefUndo}</button> : null}
                </span>
              ) : null}
            </div>
          ) : null}
        </>
      )}
    </article>
  );
}

/** AskButton 是条目右边的「追问」：把这一条放进输入框，不替你发。鼠标停在这一行时才出现。 */
function AskButton(props: { onClick: () => void }) {
  const copy = useCopy();
  return (
    <button
      type="button"
      className="h-6 shrink-0 rounded-full px-2 text-ui-caption text-muted opacity-0 transition-opacity duration-[var(--duration-fast)] hover:bg-lift hover:text-foreground focus-visible:opacity-100 group-hover/item:opacity-100 [@media(hover:none)]:opacity-100"
      onClick={props.onClick}
    >
      {copy.desk.briefAsk}
    </button>
  );
}

/** CardWhy 是卡片右上角的 ⓘ：先说为什么给你看，再给三种反馈。反馈写到开口门对应的类别上。 */
function CardWhy(props: { card: BriefCard; basis: string; source: string }) {
  const copy = useCopy();
  const d = copy.desk;
  const className = cardClass(props.card);
  async function feedback(kind: "ignore" | "later" | "wrong") {
    const ok = await sendFeedback(className, kind, "", `card:${props.card.id}`);
    if (ok) toast.success(d.fbDone);
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="grid size-7 place-items-center rounded-full text-muted/70 hover:bg-lift hover:text-foreground" aria-label={d.why} data-testid="brief-why">
          <Info className="size-3.5" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-72">
        <DropdownMenuLabel>{d.why}</DropdownMenuLabel>
        <div className="space-y-1 px-2 pb-2 text-ui-footnote leading-5">
          {props.basis ? (
            <p className="flex gap-2"><span className="w-10 shrink-0 text-muted">{d.whyBasis}</span><span className="min-w-0 text-foreground/85">{props.basis}</span></p>
          ) : null}
          {props.source ? (
            <p className="flex gap-2"><span className="w-10 shrink-0 text-muted">{d.whySource}</span><span className="min-w-0 text-foreground/85">{props.source}</span></p>
          ) : null}
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => { void feedback("ignore"); }}>{d.fbLess}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => { void feedback("later"); }}>{d.fbLater}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => { void feedback("wrong"); }}>{d.fbWrong}</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function BriefPrefsView(props: {
  prefs: BriefPrefs;
  onChange: (next: BriefPrefs) => void;
  onBack: () => void;
  onOpenSettings: () => void;
}) {
  const copy = useCopy();
  return (
    <div className="min-w-0 px-3 pb-8 pt-3">
      <button type="button" className="grid size-8 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground" onClick={props.onBack} aria-label={copy.work.taskBack}>
        <ChevronLeft className="size-4" />
      </button>
      <h1 className="mt-3 text-ui-display font-semibold tracking-[-0.03em]">{copy.work.briefPrefs}</h1>
      <p className="mt-2 text-ui-body leading-6 text-muted">{copy.work.briefPrefsHint}</p>
      <BriefPrefCard
        title={copy.work.briefMorning}
        on={props.prefs.morningOn}
        at={props.prefs.morningAt}
        onToggle={() => props.onChange({ ...props.prefs, morningOn: !props.prefs.morningOn })}
        onTime={(morningAt) => props.onChange({ ...props.prefs, morningAt })}
      />
      <BriefPrefCard
        title={copy.work.briefEvening}
        on={props.prefs.eveningOn}
        at={props.prefs.eveningAt}
        onToggle={() => props.onChange({ ...props.prefs, eveningOn: !props.prefs.eveningOn })}
        onTime={(eveningAt) => props.onChange({ ...props.prefs, eveningAt })}
      />
      <QuietPrefs />
      <p className="mt-4 px-1 text-ui-footnote leading-5 text-muted">
        <button type="button" className="underline-offset-2 hover:underline" onClick={props.onOpenSettings}>{copy.desk.prefsMore}</button>
      </p>
    </div>
  );
}

/** QuietPrefs 列出今天仍被按住的开口类别，一类一行，可以打开。公开检索便条另有一个开关。 */
function QuietPrefs() {
  const copy = useCopy();
  const [mutes, setMutes] = useState<DeskMute[]>(() => deskMutes());
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    const pull = () => setMutes(deskMutes());
    window.addEventListener(DESK_EVENT, pull);
    void refreshDesk();
    return () => window.removeEventListener(DESK_EVENT, pull);
  }, []);
  const labels: Record<string, string> = {
    calendar: copy.work.classCalendar,
    source: copy.work.classSource,
    line: copy.work.classLine,
    git: copy.work.classGit,
    power: copy.work.classPower,
    net: copy.work.classNet,
    reach: copy.work.classReach,
    poi: copy.work.classPoi,
  };
  const reachOff = mutes.some((row) => row.class === "reach" && row.until === "9999-12-31");
  const others = mutes.filter((row) => row.class !== "reach" || !reachOff);
  async function send(className: string, kind: "mute" | "unmute") {
    if (busy) return;
    setBusy(true);
    await sendFeedback(className, kind);
    setBusy(false);
  }
  return (
    <section className="day-card mt-4 px-5 py-5" data-testid="quiet-prefs">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-ui-title-s font-semibold">{copy.work.reachSwitch}</h2>
          <p className="mt-1 text-ui-footnote leading-5 text-muted">{copy.work.reachSwitchHint}</p>
        </div>
        <PillSwitch on={!reachOff} label={copy.work.reachSwitch} disabled={busy} onChange={() => { void send("reach", reachOff ? "unmute" : "mute"); }} />
      </div>
      <h2 className="mt-5 text-ui-body font-medium">{copy.work.quietTitle}</h2>
      <p className="mt-1 text-ui-footnote leading-5 text-muted">{others.length ? copy.work.quietLede : copy.work.quietEmpty}</p>
      {others.length ? (
        <ul className="mt-2 divide-y divide-border/50">
          {others.map((row) => (
            <li key={row.class} className="flex items-center justify-between gap-3 py-2">
              <div className="min-w-0">
                <p className="truncate text-ui-body">{labels[row.class] || row.class}</p>
                <p className="text-ui-footnote text-muted">{row.until === "9999-12-31" ? copy.work.quietOff : copy.work.quietToday}</p>
              </div>
              <TextButton quiet onClick={() => { void send(row.class, "unmute"); }}>{copy.work.quietOn}</TextButton>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function BriefPrefCard(props: {
  title: string;
  on: boolean;
  at: string;
  onToggle: () => void;
  onTime: (value: string) => void;
}) {
  const copy = useCopy();
  return (
    <section className="day-card mt-4 px-5 py-5">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-ui-title-s font-semibold">{props.title}</h2>
        <PillSwitch on={props.on} label={copy.work.briefStatus} onChange={props.onToggle} />
      </div>
      <label className={cn("mt-4 flex items-center justify-between gap-3", !props.on && "opacity-50")}>
        <span className="text-ui-body text-muted">{copy.work.briefSchedule}</span>
        <input type="time" value={props.at} disabled={!props.on} className="h-9 rounded-full border border-border bg-lift/75 px-4 text-ui-body outline-none" onChange={(event) => props.onTime(event.target.value)} />
      </label>
    </section>
  );
}

/** PillSwitch 是这一栏自己的圆开关。small 用在一行高的药丸里。 */
function PillSwitch(props: { on: boolean; label: string; disabled?: boolean; small?: boolean; onChange: (on: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={props.on}
      aria-label={props.label}
      disabled={props.disabled}
      className={cn("relative shrink-0 rounded-full transition-colors disabled:opacity-50", props.small ? "h-6 w-10" : "h-7 w-12", props.on ? "bg-accent" : "bg-lift")}
      onClick={() => props.onChange(!props.on)}
    >
      <span className={cn(
        "absolute top-0.5 rounded-full bg-white shadow-sm transition-[left]",
        props.small ? "size-5" : "size-6",
        props.on ? (props.small ? "left-[18px]" : "left-5") : "left-0.5",
      )} />
    </button>
  );
}

export function briefPhrases(work: ReturnType<typeof useCopy>["work"]): BriefPhrases {
  return {
    dueTitle: work.auditDueTitle,
    personTitle: work.auditPersonTitle,
    stuckTitle: work.auditStuckTitle,
    quietTitle: work.auditQuietTitle,
    lede: work.auditLede,
    quietLede: work.auditQuietLede,
    insightTitle: work.auditInsightTitle,
    insight: work.auditInsight,
    rules: work.auditRules,
    overdue: work.auditOverdue,
    dueToday: work.auditDueToday,
    dueSoon: work.auditDueSoon,
    waiting: work.auditWaiting,
    blocked: work.auditBlocked,
    failed: work.auditFailed,
    observed: work.auditObserved,
    source: work.auditSource,
    profile: work.auditProfile,
    project: work.auditProject,
    episodic: work.briefEpisodic,
    semantic: work.briefSemantic,
    poiTitle: work.briefPoiTitle,
    poiLede: work.briefPoiLede,
    aroundTitle: work.aroundTitle,
    aroundLede: work.aroundLede,
    aroundEmpty: work.aroundEmpty,
    aroundEmptyDetail: work.aroundEmptyDetail,
    appSeen: work.appSeen,
    traceSeen: work.traceSeen,
    statApps: work.statApps,
    statTraces: work.statTraces,
    statOpen: work.statOpen,
    statDated: work.statDated,
    open: work.auditOpen,
    nothing: work.auditNothing,
    nothingDetail: work.auditNothingDetail,
    keep: work.briefKeep,
    hold: work.briefHold,
    directionTitle: work.briefDirectionTitle,
    directionLede: work.briefDirectionLede,
    useChoice: work.briefUseChoice,
    keepAll: work.briefKeepAll,
    keptTitle: work.briefKeptTitle,
    keptLede: work.briefKeptLede,
    useMemory: work.briefUseMemory,
    skipMemory: work.briefSkipMemory,
    afterTitle: work.briefAfterTitle,
    afterLede: work.briefAfterLede,
    lookFirst: work.briefLookFirst,
    leaveAll: work.briefLeaveAll,
    watchTitle: work.briefWatchTitle,
    watchLede: work.briefWatchLede,
    watchDue: work.briefWatchDue,
    watchWait: work.briefWatchWait,
    watchOpen: work.briefWatchOpen,
    watchMemory: work.briefWatchMemory,
    saveWatch: work.briefSaveWatch,
    leaveWatch: work.briefLeaveWatch,
    situationKicker: work.briefSituation,
    emptyKicker: work.briefEmptyKicker,
    gateKicker: work.gateKicker,
    gateTitle: work.gateTitle,
    gateLede: work.gateLede,
    gatePrimary: work.gatePrimary,
    gateSecondary: work.gateSecondary,
    gateTask: work.gateTask,
    gateOne: work.gateOne,
    splitKicker: work.splitKicker,
    splitTitle: work.splitTitle,
    splitLede: work.splitLede,
    splitPrimary: work.splitPrimary,
    splitTask: work.splitTask,
    memoryKicker: work.memoryKicker,
    memoryTitle: work.memoryTitle,
    memoryLede: work.memoryLede,
    memoryPrimary: work.memoryPrimary,
    memorySecondary: work.memorySecondary,
    memoryTask: work.memoryTask,
    memoryOne: work.memoryOne,
    aroundKicker: work.aroundKicker,
    aroundPrimary: work.aroundPrimary,
    aroundSecondary: work.aroundSecondary,
    aroundTask: work.aroundTask,
    aroundOne: work.aroundOne,
    nextTitle: work.nextTitle,
    nextPrimary: work.nextPrimary,
    nextTask: work.nextTask,
    quietPrimary: work.quietPrimary,
    quietTask: work.quietTask,
    countLabel: work.countLabel,
    letterKicker: work.letterKicker,
    letterKickerToday: work.letterKickerToday,
    letterHello: work.letterHello,
    letterHelloBare: work.letterHelloBare,
    letterLede: work.letterLede,
    letterLedeToday: work.letterLedeToday,
    letterKnown: work.letterKnown,
    letterMissing: work.letterMissing,
    letterLines: work.letterLines,
    letterIdentity: work.letterIdentity,
    letterHost: work.letterHost,
    letterOs: work.letterOs,
    letterGit: work.letterGit,
    letterApps: work.letterApps,
    letterHours: work.letterHours,
    letterNoMail: work.letterNoMail,
    letterNoMailDetail: work.letterNoMailDetail,
    letterNoPlan: work.letterNoPlan,
    letterNoPlanDetail: work.letterNoPlanDetail,
    linesKicker: work.linesKicker,
    linesTitle: work.linesTitle,
    linesLede: work.linesLede,
    lineMoving: work.lineMoving,
    lineStalled: work.lineStalled,
    lineQuiet: work.lineQuiet,
    linesPrimary: work.linesPrimary,
    linesTask: work.linesTask,
    doneKicker: work.doneKicker,
    doneTitle: work.doneTitle,
    doneLede: work.doneLede,
    doneMark: work.doneMark,
    doneFinished: work.doneFinished,
    doneEmpty: work.doneEmpty,
    donePrimary: work.donePrimary,
    doneAlt: work.doneAlt,
    doneTask: work.doneTask,
    placeKicker: work.placeKicker,
    placeTitle: work.placeTitle,
    placeLede: work.placeLede,
    placeLeft: work.placeLeft,
    placeRight: work.placeRight,
    placeNone: work.placeNone,
    placePrimary: work.placePrimary,
    placeTask: work.placeTask,
    toolKicker: work.toolKicker,
    toolTitle: work.toolTitle,
    toolLede: work.toolLede,
    toolProjects: work.toolProjects,
    toolApps: work.toolApps,
    toolDownloads: work.toolDownloads,
    toolTraces: work.toolTraces,
    toolPrimary: work.toolPrimary,
    toolTask: work.toolTask,
    pauseKicker: work.pauseKicker,
    pauseTitle: work.pauseTitle,
    pauseLede: work.pauseLede,
    pauseGo: work.pauseGo,
    pauseGoDetail: work.pauseGoDetail,
    pauseStop: work.pauseStop,
    pauseStopDetail: work.pauseStopDetail,
    pauseNote: work.pauseNote,
  };
}

function Tasks(props: {
  sections: Record<TaskSection, TaskItem[]>;
  observed: { id: string; text: string; at: string }[];
  selected: string;
  onSelect: (id: string) => void;
  onOpenGroup: (id: string) => void;
  onNewTask: (text?: string) => void;
  held: { id: string; name: string; detail?: string }[];
  notes: { id: string; title: string }[];
  rules: Rule[];
  mode: string;
  onRules: (rules: Rule[]) => Promise<void>;
  onMoreSchedules?: () => void;
}) {
  const copy = useCopy();
  const d = copy.desk;
  const buddies = useBuddy((s) => s.list);
  const [compose, setCompose] = useState<null | { kind: "task" | "schedule"; text: string; ruleId?: string }>(null);
  const [query, setQuery] = useState("");
  const [scheduled, setScheduled] = useState(false);
  const suggestions = suggestionCopy(copy);
  const agentName = (id: string) => buddies.find((row) => row.id === id)?.name || id;
  const needle = query.trim().toLowerCase();
  const missing = unseenSuggestions(props.rules.map((rule) => rule.id)).filter((id) => {
    const item = suggestions[id];
    if (!item || !needle) return true;
    return `${item.title} ${item.text}`.toLowerCase().includes(needle);
  });
  const order: { id: TaskSection; title: string }[] = [
    { id: "person", title: copy.work.sectionPerson },
    { id: "working", title: copy.work.sectionWorking },
    { id: "idle", title: copy.work.sectionIdle },
  ];
  const groups = order
    .map((group) => ({ ...group, items: filterItems(props.sections[group.id], query, agentName) }))
    .filter((group) => group.items.length > 0);
  const ended = filterItems(props.sections.ended, query, agentName);
  const rules = props.rules.filter((rule) => !needle || rule.text.toLowerCase().includes(needle));
  const ruleMeta = (rule: Rule) => !rule.enabled ? d.ruleOff : props.mode === "auto" ? d.ruleAuto : props.mode === "off" ? d.ruleOff : d.ruleObserve;
  if (compose) {
    const scheduled = compose.kind === "schedule";
    const starters = scheduled
      ? missing.slice(0, 3).map((id) => ({ id, title: suggestions[id]?.title || id, text: suggestions[id]?.text || "" })).filter((item) => item.text)
      : [
          { id: "next", title: copy.work.draftNext, text: copy.work.draftNext },
          { id: "desk", title: copy.work.draftDesk, text: copy.work.draftDesk },
          { id: "wait", title: copy.work.draftWait, text: copy.work.draftWait },
        ];
    return (
      <TaskCompose
        title={scheduled ? copy.work.ruleAdd : copy.work.newTask}
        placeholder={scheduled ? copy.work.ruleText : copy.work.taskDraftPh}
        hint={scheduled ? copy.work.ruleDraftHint : copy.work.taskDraftHint}
        empty={copy.work.noSubtasks}
        submitLabel={scheduled ? copy.work.saveSchedule : copy.work.startTask}
        suggestions={starters}
        initial={compose.text}
        onBack={() => setCompose(null)}
        onSubmit={(text) => {
          if (scheduled) {
            const id = compose.ruleId || `custom-${Date.now()}`;
            const next = props.rules.some((rule) => rule.id === id)
              ? props.rules.map((rule) => rule.id === id ? { ...rule, text, enabled: true } : rule)
              : [...props.rules, { id, text, enabled: true }];
            void props.onRules(next);
          } else {
            props.onNewTask(text);
          }
          setCompose(null);
        }}
      />
    );
  }
  return (
    <div className="px-3 pb-8 pt-2">
      {props.held.length || props.notes.length ? (
        <section className="day-card mb-4 px-4 py-4">
          <h2 className="text-ui-body font-medium">{copy.work.heldTitle}</h2>
          <div className="mt-3 space-y-2">
            {props.held.map((signal) => (
              <p key={signal.id} className="text-ui-body leading-5">{signal.name}{signal.detail ? <span className="text-muted"> · {signal.detail}</span> : null}</p>
            ))}
            {props.notes.map((note) => (
              <p key={note.id} className="text-ui-body leading-5">{note.title}</p>
            ))}
          </div>
        </section>
      ) : null}
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-ui-title font-semibold tracking-[-0.025em]">{copy.work.tasks}</h1>
        <button
          type="button"
          className="inline-flex h-8 shrink-0 items-center gap-1 rounded-full border border-border bg-card px-3 text-ui-body font-medium shadow-[var(--shadow-card)] transition-colors hover:bg-lift"
          onClick={() => setCompose(scheduled ? { kind: "schedule", text: "" } : { kind: "task", text: "" })}
        >
          <Plus className="size-3.5" />
          {scheduled ? copy.work.ruleAdd : copy.work.createTask}
        </button>
      </div>
      <div className="mt-4 grid h-9 grid-cols-2 gap-2">
        <label className={cn(
          "flex min-w-0 items-center gap-2 rounded-full border px-3.5 transition-colors",
          scheduled ? "border-transparent bg-lift/75 text-muted" : "border-border bg-card text-foreground shadow-[var(--shadow-card)]",
        )}>
          <Search className="size-3.5 shrink-0 text-muted" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onFocus={() => setScheduled(false)}
            placeholder={copy.work.taskSearch}
            aria-label={copy.work.taskSearch}
            className="min-w-0 flex-1 bg-transparent text-ui-body outline-none placeholder:text-muted"
          />
        </label>
        <button
          type="button"
          aria-pressed={scheduled}
          className={cn(
            "flex items-center justify-center gap-1.5 rounded-full border text-ui-body font-medium transition-colors duration-200",
            scheduled ? "border-border bg-card text-foreground shadow-[var(--shadow-card)]" : "border-transparent bg-lift/75 text-muted hover:text-foreground",
          )}
          onClick={() => setScheduled(true)}
        >
          <ListChecks className="size-3.5" />
          {copy.work.rules}
          {props.rules.length ? <span className="tabular-nums text-ui-caption text-muted">{props.rules.filter((rule) => rule.enabled).length}</span> : null}
        </button>
      </div>
      {scheduled ? (
        <div className="mt-6" data-testid="task-rules">
          {rules.length > 0 ? (
            <ul className="space-y-2">
              {rules.map((rule) => (
                <li key={rule.id}>
                  <RulePill
                    label={suggestions[rule.id]?.title || rule.text}
                    meta={ruleMeta(rule)}
                    on={rule.enabled}
                    onOpen={() => setCompose({ kind: "schedule", text: rule.text, ruleId: rule.id })}
                    onToggle={(on) => { void props.onRules(props.rules.map((row) => row.id === rule.id ? { ...row, enabled: on } : row)); }}
                  />
                </li>
              ))}
            </ul>
          ) : null}
          {props.observed.length > 0 ? (
            <section className={rules.length > 0 ? "mt-6" : ""} data-testid="would-accept">
              <h2 className="flex items-center gap-1.5 px-1 text-ui-title-s font-medium">
                {copy.work.wouldAccept}
                <span className="tabular-nums text-ui-caption font-normal text-muted">{props.observed.length}</span>
              </h2>
              <p className="mt-1 px-1 text-ui-footnote leading-5 text-muted">{copy.work.observedHint}</p>
              <ul className="day-card mt-2.5 divide-y divide-border/60 px-4">
                {props.observed.map((row) => (
                  <li key={row.id} className="py-2.5">
                    <p className="line-clamp-2 text-ui-body leading-5">{row.text}</p>
                    {row.at ? <p className="mt-0.5 text-ui-footnote text-muted">{agoLabel(row.at, copy.work)}</p> : null}
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
          {missing.length > 0 ? (
            <section className={rules.length > 0 || props.observed.length > 0 ? "mt-6" : ""}>
              <h2 className="px-1 text-ui-title-s font-medium">{copy.work.recommended}</h2>
              <ul className="mt-2.5 space-y-2">
                {missing.map((id) => {
                  const item = suggestions[id];
                  if (!item) return null;
                  const Icon = item.icon;
                  return (
                    <li key={id}>
                      <button
                        type="button"
                        className="day-tile flex w-full items-center gap-3 px-4 py-3.5 text-left"
                        onClick={() => setCompose({ kind: "schedule", text: item.text, ruleId: id })}
                      >
                        <Icon className="size-4 shrink-0 text-muted" />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-ui-body-l font-medium">{item.title}</span>
                          <span className="mt-0.5 block truncate text-ui-footnote leading-5 text-muted">{item.text}</span>
                        </span>
                        <Plus className="size-4 shrink-0 text-muted" />
                      </button>
                    </li>
                  );
                })}
              </ul>
            </section>
          ) : null}
          {rules.length === 0 && missing.length === 0 && props.observed.length === 0 ? <Empty title={copy.work.tasksEmpty} hint={copy.work.tasksHint} /> : null}
          {props.onMoreSchedules ? (
            <button type="button" className="mt-5 w-full rounded-full py-2 text-ui-footnote text-muted hover:bg-lift hover:text-foreground" onClick={props.onMoreSchedules}>
              {d.moreSchedules}
            </button>
          ) : null}
        </div>
      ) : (
        <div>
          {groups.map((group) => (
            <TicketList key={group.id} title={group.title} count={group.items.length} person={group.id === "person"} items={group.items} onSelect={props.onSelect} onOpenGroup={props.onOpenGroup} />
          ))}
          {ended.length > 0 ? (
            <TicketList title={copy.work.sectionEnded} count={ended.length} items={ended} onSelect={props.onSelect} onOpenGroup={props.onOpenGroup} folded />
          ) : null}
          {groups.length === 0 && ended.length === 0 ? <Empty title={copy.work.tasksEmpty} hint={copy.work.tasksHint} /> : null}
        </div>
      )}
    </div>
  );
}

function TaskCompose(props: {
  title: string;
  placeholder: string;
  hint: string;
  empty: string;
  submitLabel: string;
  suggestions: { id: string; title: string; text: string }[];
  initial: string;
  onBack: () => void;
  onSubmit: (text: string) => void;
}) {
  const [text, setText] = useState(props.initial);
  return (
    <div className="flex min-h-full flex-col px-1 pb-4 pt-2">
      <div className="flex items-center gap-1">
        <button type="button" className="grid size-8 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground" onClick={props.onBack} aria-label={props.title}>
          <ChevronLeft className="size-4" />
        </button>
        <h1 className="text-ui-title-s font-semibold">{props.title}</h1>
      </div>
      <div className="grid flex-1 place-items-center px-6 py-16">
        <p className="text-ui-body-l text-muted">{props.empty}</p>
      </div>
      <div className="space-y-1">
        {props.suggestions.map((item) => (
          <button key={item.id} type="button" className="flex w-full items-center gap-2 rounded-xl px-2 py-2 text-left text-ui-body text-muted hover:bg-lift hover:text-foreground" onClick={() => setText(item.text)}>
            <ChevronRight className="size-3.5 shrink-0" />
            <span className="truncate">{item.title}</span>
          </button>
        ))}
      </div>
      <label className="mt-3 block">
        <textarea
          value={text}
          onChange={(event) => setText(event.target.value)}
          placeholder={props.placeholder}
          rows={3}
          className="day-card w-full resize-none px-3.5 py-3 text-ui-body leading-6 outline-none placeholder:text-muted"
        />
      </label>
      <div className="mt-2 flex items-center justify-between gap-3">
        <p className="min-w-0 text-ui-footnote leading-5 text-muted">{props.hint}</p>
        <button
          type="button"
          className="h-8 shrink-0 rounded-full bg-foreground px-3.5 text-ui-footnote font-medium text-background disabled:opacity-40"
          disabled={!text.trim()}
          onClick={() => props.onSubmit(text.trim())}
        >
          {props.submitLabel}
        </button>
      </div>
    </div>
  );
}

function suggestionCopy(copy: ReturnType<typeof useCopy>): Record<string, { title: string; text: string; icon: LucideIcon }> {
  const work = copy.work;
  return {
    "daily-brief": { title: work.suggestDaily, text: work.suggestDailyText, icon: Sun },
    "waiting-reply": { title: work.suggestReply, text: work.suggestReplyText, icon: MessageCircle },
    "meeting-prep": { title: work.suggestMeet, text: work.suggestMeetText, icon: Calendar },
    "weekly-status": { title: work.suggestWeek, text: work.suggestWeekText, icon: ListChecks },
    "deadline-watch": { title: work.suggestDeadline, text: work.suggestDeadlineText, icon: Clock },
    "mentioned-mail": { title: work.suggestMail, text: work.suggestMailText, icon: Mail },
  };
}

/** TicketList 是一组任务，任务组占一项。folded 的一组默认收起，点标题展开。 */
function TicketList(props: {
  title: string;
  count: number;
  person?: boolean;
  folded?: boolean;
  items: TaskItem[];
  onSelect: (id: string) => void;
  onOpenGroup: (id: string) => void;
}) {
  const [open, setOpen] = useState(!props.folded);
  const head = (
    <>
      {props.title}
      <span className={cn("tabular-nums text-ui-caption font-normal", props.person ? "text-warning" : "text-muted")}>{props.count}</span>
    </>
  );
  return (
    <section className="mt-6" data-testid="task-section">
      {props.folded ? (
        <button type="button" aria-expanded={open} className="flex items-center gap-1.5 px-1 text-ui-title-s font-medium hover:text-foreground/80" onClick={() => setOpen((on) => !on)}>
          {head}
          <ChevronDown className={cn("size-3.5 text-muted transition-transform duration-200", !open && "-rotate-90")} />
        </button>
      ) : (
        <h2 className="flex items-center gap-1.5 px-1 text-ui-title-s font-medium">{head}</h2>
      )}
      {open ? (
        <ul className="mt-2.5 space-y-2">
          {props.items.map((item) =>
            item.kind === "group" ? (
              <li key={`group:${item.group.id}`}>
                <GroupRow group={item.group} members={item.members} onOpen={props.onOpenGroup} onSelect={props.onSelect} />
              </li>
            ) : (
              <li key={item.ticket.id}>
                <TaskRow ticket={item.ticket} onClick={() => props.onSelect(item.ticket.id)} />
              </li>
            ),
          )}
        </ul>
      ) : null}
    </section>
  );
}

/** RulePill 是一条定时：点名字改写，右边的开关就地开关。 */
function RulePill(props: { label: string; meta: string; on: boolean; onOpen: () => void; onToggle: (on: boolean) => void }) {
  return (
    <div className="day-tile flex h-11 w-full items-center gap-2 rounded-full pl-4 pr-2.5">
      <button type="button" className={cn("flex min-w-0 flex-1 items-baseline gap-2 text-left", !props.on && "text-muted")} onClick={props.onOpen}>
        <span className="truncate text-ui-body-l">{props.label}</span>
        <span className="shrink-0 text-ui-caption text-muted">{props.meta}</span>
      </button>
      <PillSwitch small on={props.on} label={props.label} onChange={props.onToggle} />
    </div>
  );
}

const FOLDER_PAGE = 30;

function Memory(props: { rows: MemoryRow[]; onAdd: (text: string) => Promise<void>; onForget: (id: string) => Promise<void>; onEdit: (id: string, text: string) => Promise<boolean> }) {
  const copy = useCopy();
  const [folder, setFolder] = useState<MemoryFolder | "">("");
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const [saving, setSaving] = useState(false);
  const [all, setAll] = useState(false);
  const [editing, setEditing] = useState<{ id: string; text: string } | null>(null);
  const portrait = memoryPortrait(props.rows);
  const sep = /[\u4e00-\u9fff]/.test(copy.work.memoryProjectsLine) ? "、" : ", ";
  const said = portrait.said.filter((text) => text !== portrait.name);
  const lead = portrait.device || portrait.host || portrait.os || portrait.zone || portrait.git || portrait.hours || portrait.lines.length
    ? copy.work.memoryLeadMachine
    : said[0] || portrait.noted[0] || copy.work.memoryLeadEmpty;
  const title = portrait.name ? copy.work.memoryAbout.replace("{name}", portrait.name) : copy.work.memoryAboutYou;
  const when = portrait.updated ? agoLabel(portrait.updated, copy.work) : copy.work.memoryNotYet;
  const folders: { id: MemoryFolder; label: string; icon: LucideIcon }[] = [
    { id: "device", label: copy.work.folderDevices, icon: Monitor },
    { id: "work", label: copy.work.folderWork, icon: Clock },
    { id: "project", label: copy.work.folderProject, icon: Folder },
    { id: "preference", label: copy.work.folderSemantic, icon: SlidersHorizontal },
    { id: "about", label: portrait.name ? copy.work.folderAbout.replace("{name}", portrait.name) : copy.work.folderAboutYou, icon: MessageCircle },
  ];
  const notes = memoryFolderRows(props.rows, "notes");
  if (notes.length) folders.push({ id: "notes", label: copy.work.folderNotes, icon: Folder });
  const inFolder = folder ? memoryFolderRows(props.rows, folder) : [];
  const shown = all ? inFolder : inFolder.slice(0, FOLDER_PAGE);
  const origins: Record<string, string> = {
    said: copy.work.originSaid,
    machine: copy.work.originMachine,
    task: copy.work.originTask,
    noted: copy.work.originNoted,
    chat: copy.work.originChat,
  };
  async function add(event: { preventDefault: () => void }) {
    event.preventDefault();
    const text = draft.trim();
    if (!text || saving) return;
    setSaving(true);
    try {
      await props.onAdd(text);
      setDraft("");
      setAdding(false);
    } finally {
      setSaving(false);
    }
  }
  return (
    <div className="px-2 pb-2 pt-2">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-ui-display font-semibold tracking-[-0.03em]">{copy.work.memory}</h1>
        <button type="button" className="grid size-8 place-items-center rounded-full border border-border bg-card text-foreground shadow-[var(--shadow-card)] transition-colors hover:bg-lift" aria-label={copy.work.memoryAdd} onClick={() => setAdding((on) => !on)}>
          <Plus className="size-4" />
        </button>
      </div>
      {adding ? (
        <form className="day-card mt-4 px-5 py-4" onSubmit={(event) => { void add(event); }}>
          <input value={draft} onChange={(event) => setDraft(event.target.value)} placeholder={copy.work.memoryAddHint} className="h-10 w-full bg-transparent text-ui-body-l outline-none" />
          <div className="mt-2 flex justify-end">
            <button type="submit" disabled={!draft.trim() || saving} className="h-8 rounded-full bg-foreground px-3.5 text-ui-footnote font-medium text-background disabled:opacity-40">{copy.work.save}</button>
          </div>
        </form>
      ) : null}
      <article className="day-card mt-4 px-5 py-5" data-testid="memory-about">
        <h2 className="text-ui-title font-semibold leading-6 tracking-[-0.025em]">{title}</h2>
        <p className="mt-1 text-ui-footnote text-muted">{portrait.updated ? copy.work.memoryUpdated.replace("{when}", when) : copy.work.memoryNotYet}</p>
        <p className="mt-4 text-ui-footnote text-muted">{copy.work.memorySummary}</p>
        <p className="mt-1 text-ui-body-l leading-6">{lead}</p>
        {portrait.device ? <p className="mt-3 text-ui-body-l leading-6">{copy.work.memoryDeviceLine.replace("{name}", portrait.device)}</p> : null}
        {portrait.host ? <p className="mt-2 text-ui-body-l leading-6">{copy.work.memoryHostLine.replace("{name}", portrait.host)}</p> : null}
        {portrait.os ? <p className="mt-2 text-ui-body-l leading-6">{copy.work.memoryOsLine.replace("{name}", portrait.os)}</p> : null}
        {portrait.zone ? <p className="mt-2 text-ui-body-l leading-6">{copy.work.memoryZoneLine.replace("{name}", portrait.zone)}</p> : null}
        {portrait.git ? <p className="mt-2 text-ui-body-l leading-6">{copy.work.memoryGitLine.replace("{name}", portrait.git)}</p> : null}
        {portrait.hours ? <p className="mt-2 text-ui-body-l leading-6">{copy.work.memoryHoursLine.replace("{hours}", portrait.hours)}</p> : null}
        {portrait.lines.length ? <p className="mt-2 text-ui-body-l leading-6">{copy.work.memoryProjectsLine.replace("{names}", portrait.lines.map((line) => line.name).join(sep))}</p> : null}
        {portrait.device || portrait.host || portrait.os || portrait.zone || portrait.git || portrait.hours || portrait.lines.length ? <p className="mt-3 text-ui-body leading-6 text-muted">{copy.work.memoryHardFact}</p> : null}
        {said.length ? <MemoryLines label={copy.work.memorySaid} lines={said} /> : null}
        {portrait.noted.length ? <MemoryLines label={copy.work.memoryNoted} lines={portrait.noted} /> : null}
        {portrait.preferences.length ? <MemoryLines label={copy.work.memoryPrefs} lines={portrait.preferences} /> : null}
      </article>
      <h2 className="mt-6 px-1 text-ui-title-s font-medium">{copy.work.folderAll}</h2>
      <div className="mt-2.5 grid grid-cols-2 gap-2">
        {folders.map((item) => {
          const count = memoryFolderRows(props.rows, item.id).length;
          const on = folder === item.id;
          const Icon = item.icon;
          return (
            <button
              key={item.id}
              type="button"
              className={cn("day-tile px-4 py-3.5 text-left", on && "border-accent/50 bg-accent/6 ring-1 ring-accent/40")}
              onClick={() => { setFolder(on ? "" : item.id); setAll(false); }}
            >
              <Icon className={cn("size-4", on ? "text-accent" : "text-muted")} />
              <p className="mt-2.5 text-ui-body-l font-medium">{item.label}</p>
              <p className="mt-0.5 tabular-nums text-ui-footnote text-muted">{copy.work.itemCount.replace("{n}", String(count))}</p>
            </button>
          );
        })}
      </div>
      {folder ? (
        shown.length === 0 ? <p className="mt-4 px-1 text-ui-body text-muted">{copy.work.memoryFolderEmpty}</p> : (
          <>
            <ul className="mt-4 space-y-2">
              {shown.map((row) => {
                const origin = origins[memoryOrigin(row)] || "";
                const touched = memoryTouched(row);
                const at = touched ? agoLabel(touched, copy.work) : "";
                const past = memoryPast(row);
                const before = [...(row.history ?? [])].reverse().map((rev) => copy.work.memoryBefore.replace("{text}", rev.text));
                const meta = [
                  origin,
                  past ? copy.work.memoryPast : row.valid_until ? copy.work.memoryUntil.replace("{date}", row.valid_until) : "",
                  at,
                  before.length ? copy.work.memoryChanged.replace("{n}", String(before.length)) : "",
                  (row.seen ?? 0) > 1 ? copy.work.memorySeen.replace("{n}", String(row.seen)) : "",
                ];
                const open = editing?.id === row.id;
                return (
                  <li key={row.id} className="day-tile group/memory px-4 py-3.5" data-testid="memory-row">
                    {open ? (
                      <form
                        onSubmit={async (event) => {
                          event.preventDefault();
                          const text = editing.text.trim();
                          if (!text || text === row.text) {
                            setEditing(null);
                            return;
                          }
                          if (await props.onEdit(row.id, text)) setEditing(null);
                        }}
                      >
                        <textarea
                          autoFocus
                          rows={2}
                          value={editing.text}
                          onChange={(event) => setEditing({ id: row.id, text: event.target.value })}
                          onKeyDown={(event) => { if (event.key === "Escape") setEditing(null); }}
                          className="w-full resize-none rounded-lg border border-border bg-lift/60 px-2.5 py-2 text-ui-body-l leading-6 outline-none focus-visible:border-accent/50"
                        />
                        <div className="mt-1.5 flex justify-end gap-1.5">
                          <TextButton quiet onClick={() => setEditing(null)}>{copy.work.memoryEditCancel}</TextButton>
                          <button type="submit" disabled={!editing.text.trim()} className="h-7 rounded-lg bg-foreground px-2.5 text-ui-footnote font-medium text-background disabled:opacity-40">{copy.work.save}</button>
                        </div>
                      </form>
                    ) : (
                      <>
                        <p className={cn("text-ui-body-l leading-6", past && "text-muted")}>{memoryRowTitle(row, copy.work)}</p>
                        {memoryRowDetail(row, copy.work) ? <p className="mt-1 text-ui-footnote leading-5 text-muted">{memoryRowDetail(row, copy.work)}</p> : null}
                        <div className="mt-1.5 flex min-h-6 items-center justify-between gap-2">
                          <p className="min-w-0 truncate text-ui-caption text-muted/80" data-testid="memory-meta" title={before.length ? before.join("\n") : undefined}>{meta.filter(Boolean).join(" · ")}</p>
                          {memoryForgettable(row) ? (
                            <span className="flex shrink-0 items-center gap-0.5">
                              <button
                                type="button"
                                className="h-6 rounded-md px-2 text-ui-caption font-medium text-muted opacity-0 transition-colors duration-[var(--duration-fast)] hover:bg-lift hover:text-foreground focus-visible:opacity-100 group-hover/memory:opacity-100 [@media(hover:none)]:opacity-100"
                                onClick={() => setEditing({ id: row.id, text: row.text })}
                              >
                                {copy.work.memoryEdit}
                              </button>
                              <ForgetButton onForget={() => props.onForget(row.id)} />
                            </span>
                          ) : null}
                        </div>
                      </>
                    )}
                  </li>
                );
              })}
            </ul>
            {!all && inFolder.length > shown.length ? (
              <button type="button" className="mt-2 w-full rounded-full py-2 text-ui-footnote text-muted hover:bg-lift hover:text-foreground" onClick={() => setAll(true)}>
                {copy.work.memoryMore.replace("{n}", String(inFolder.length - shown.length))}
              </button>
            ) : null}
          </>
        )
      ) : null}
    </div>
  );
}

/** ForgetButton 要点两次。第一次变成确认，三秒内不再点就收回。 */
function ForgetButton(props: { onForget: () => Promise<void> }) {
  const copy = useCopy();
  const [armed, setArmed] = useState(false);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!armed) return;
    const timer = window.setTimeout(() => setArmed(false), 3000);
    return () => window.clearTimeout(timer);
  }, [armed]);
  return (
    <button
      type="button"
      disabled={busy}
      className={cn(
        "h-6 shrink-0 rounded-md px-2 text-ui-caption font-medium transition-colors duration-[var(--duration-fast)] disabled:opacity-50",
        armed ? "bg-danger/10 text-danger" : "text-muted opacity-0 hover:bg-lift hover:text-danger focus-visible:opacity-100 group-hover/memory:opacity-100 [@media(hover:none)]:opacity-100",
      )}
      onClick={async () => {
        if (!armed) {
          setArmed(true);
          return;
        }
        setBusy(true);
        try {
          await props.onForget();
        } finally {
          setBusy(false);
          setArmed(false);
        }
      }}
    >
      {armed ? copy.work.memoryForgetConfirm : copy.work.memoryForget}
    </button>
  );
}

function MemoryLines(props: { label: string; lines: string[] }) {
  return (
    <div className="mt-4">
      <p className="text-ui-footnote text-muted">{props.label}</p>
      {props.lines.map((line) => <p key={line} className="mt-1 text-ui-body-l leading-6">{line}</p>)}
    </div>
  );
}

function memoryRowTitle(row: MemoryRow, work: ReturnType<typeof useCopy>["work"]): string {
  if (row.id === "machine:device") return work.memoryDeviceLine.replace("{name}", row.text);
  if (row.id === "machine:host") return work.memoryHostLine.replace("{name}", row.text);
  if (row.id === "machine:os") return work.memoryOsLine.replace("{name}", row.text);
  if (row.id === "machine:zone") return work.memoryZoneLine.replace("{name}", row.text);
  if (row.id === "machine:git") return work.memoryGitLine.replace("{name}", row.text);
  if (row.id === "machine:hours" || row.project === "work") return work.memoryHoursLine.replace("{hours}", row.text);
  return row.text;
}

function memoryRowDetail(row: MemoryRow, work: ReturnType<typeof useCopy>["work"]): string {
  if (row.kind !== "project") return "";
  const state = row.state === "stalled" ? work.lineStalled : row.state === "quiet" ? work.lineQuiet : row.state === "moving" ? work.lineMoving : "";
  return [state, row.evidence, row.when].filter(Boolean).join(" · ");
}

function Empty(props: { title: string; hint: string }) {
  return (
    <div className="px-3 py-6">
      <p className="text-ui-body">{props.title}</p>
      <p className="mt-1 text-ui-footnote leading-5 text-muted">{props.hint}</p>
    </div>
  );
}

function TextButton(props: { children: string; onClick: () => void; quiet?: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        "h-7 shrink-0 rounded-full px-3 text-ui-footnote font-medium transition-colors duration-[var(--duration-fast)]",
        props.quiet
          ? "text-muted hover:bg-lift hover:text-foreground"
          : "border border-accent/25 bg-accent/14 text-accent hover:bg-accent/22",
      )}
      onClick={props.onClick}
    >
      {props.children}
    </button>
  );
}
