/**
 * 拥有：会话左侧的一天。今天、任务、记忆。晨间简报在这里审计资产，点选留在简报上。点开任务留在这一栏，对话不换页。连接在设置里。
 * 不拥有：工单怎么执行，也不拥有检查器里的差异。执行在 Go 的 superagent，审查页由 App 传入。
 */

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Calendar, ChevronLeft, ChevronRight, Clock, Folder, ListChecks, Mail, MessageCircle, Monitor, Plus, Search, SlidersHorizontal, Sun, type LucideIcon } from "lucide-react";
import { orchestratorAct, orchestratorDecide, orchestratorDeliver, orchestratorView, memoryList, memoryWrite, poiRefresh } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import type { DeskView, Rule, Ticket, Trigger } from "./desk-model";
import { BRIEF_EVENT, latestInitiative, noteFeedback, runDeskFeed } from "./desk-feed";
import { TaskRead } from "./TaskPage";
import { briefDay, briefPrompt, briefStorageKey, BRIEF_PREFS_KEY, defaultBriefPrefs, dueItems, filterTickets, memoryFolderRows, memoryPortrait, nowTickets, parseBriefPrefs, parseStoredBrief, serializeStoredBrief, slotDue, taskGroups, unseenSuggestions, visibleIncoming, type BriefCard, type BriefPhrases, type BriefSlot, type BriefPrefs, type MemoryFolder, type MemoryRow, type MorningBrief } from "./work-model";

type Tab = "now" | "tasks" | "memory";

/** WorkPane 在会话桌面上常驻。reviewPulse 增加时切到审查。review 是检查器。 */
export function WorkPane(props: {
  review: ReactNode;
  reviewOpen: boolean;
  reviewPulse: number;
  onReview: (open: boolean) => void;
  onNewTask: (text?: string) => void;
  onOpenSettings: () => void;
  onDraft: (text: string) => void;
  openTaskId: string;
  onOpenTask: (id: string) => void;
}) {
  const copy = useCopy();
  const [tab, setTab] = useState<Tab>("now");
  const [view, setView] = useState<DeskView | null>(null);
  const [err, setErr] = useState("");
  const [memory, setMemory] = useState<MemoryRow[]>([]);
  const [desk, setDesk] = useState(latestInitiative);
  useEffect(() => {
    const pull = () => setDesk(latestInitiative());
    window.addEventListener(BRIEF_EVENT, pull);
    return () => window.removeEventListener(BRIEF_EVENT, pull);
  }, []);
  const load = useCallback(async () => {
    try {
      await orchestratorDeliver().catch(() => undefined);
      setView(await orchestratorView());
      setErr("");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => { void load(); }, 2000);
    return () => window.clearInterval(timer);
  }, [load]);

  useEffect(() => {
    if (tab !== "memory") return;
    let stop = false;
    const run = () => {
      memoryList().then((rows) => { if (!stop) setMemory(rows as MemoryRow[]); }).catch(() => {});
    };
    poiRefresh().catch(() => null).finally(run);
    const timer = window.setInterval(run, 4000);
    return () => { stop = true; window.clearInterval(timer); };
  }, [tab]);

  useEffect(() => {
    if (props.reviewPulse > 0) props.onReview(true);
  }, [props.reviewPulse]);

  function pick(next: Tab) {
    setTab(next);
    props.onReview(false);
  }

  const tickets = view?.tickets ?? [];
  const now = nowTickets(tickets);
  const incoming = visibleIncoming(view?.queued ?? []);
  const due = dueItems(view?.queued ?? []);
  const tasks = taskGroups(tickets);
  const tabs: { id: Tab; label: string; count?: number }[] = [
    { id: "now", label: copy.work.now, count: now.length + incoming.length + due.length },
    { id: "tasks", label: copy.work.tasks, count: tasks.open.length },
    { id: "memory", label: copy.work.memory },
  ];

  if (props.reviewOpen) {
    return (
      <div data-testid="work-pane" className="flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-sidebar px-3 pb-3 pt-3 text-foreground">
        <button type="button" className="mb-2 self-start rounded-full bg-card px-3 py-1.5 text-[12px] text-muted shadow-[var(--shadow-card)] hover:text-foreground" onClick={() => props.onReview(false)}>
          {copy.work.back}
        </button>
        <div className="min-h-0 min-w-0 flex-1 overflow-hidden rounded-[28px] bg-background">{props.review}</div>
      </div>
    );
  }

  return (
    <div data-testid="work-pane" className="flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-sidebar text-foreground">
      <div className="flex shrink-0 justify-center px-3 pb-1 pt-3">
        <div role="tablist" aria-label={copy.work.title} className="flex max-w-full flex-wrap items-center justify-center gap-0.5 rounded-full bg-lift/80 p-1">
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
                  "flex h-8 items-center gap-1.5 rounded-full px-3.5 text-[13px] font-medium transition-[background-color,color,box-shadow] duration-200 ease-[var(--ease-out)]",
                  on ? "bg-card text-foreground shadow-[var(--shadow-card)]" : "text-muted hover:text-foreground",
                )}
                onClick={() => pick(item.id)}
              >
                <span>{item.label}</span>
                {item.count ? <span className="tabular-nums text-[11px] text-muted">{item.count}</span> : null}
              </button>
            );
          })}
        </div>
      </div>
      <div className="min-h-0 min-w-0 flex-1 overflow-hidden px-2 pb-2">
        <div className="h-full min-w-0 overflow-y-auto overflow-x-hidden overscroll-contain">
          {err ? <p className="px-3 py-2 text-[12px] text-danger">{err}</p> : null}
          {!view && !err ? <p className="px-3 py-3 text-[13px] text-muted">{copy.work.loading}</p> : null}
          {view && tab === "now" ? (
            <Now
              incoming={incoming}
              onDecide={async (id, kind) => { await orchestratorDecide(id, kind); await load(); }}
              onDraft={props.onDraft}
            />
          ) : null}
          {view && tab === "tasks" && props.openTaskId ? (
            <TaskRead
              ticket={(view.tickets ?? []).find((row) => row.id === props.openTaskId) ?? null}
              onBack={() => props.onOpenTask("")}
              onLook={() => props.onReview(true)}
              onContinue={(text) => props.onNewTask(text)}
            />
          ) : null}
          {view && tab === "tasks" && !props.openTaskId ? (
            <Tasks
              open={tasks.open}
              ended={tasks.ended}
              selected={props.openTaskId}
              onSelect={(id) => props.onOpenTask(id)}
              onNewTask={props.onNewTask}
              held={(desk?.signals || []).filter((signal) => signal.held)}
              notes={(desk?.open || []).filter((item) => item.class === "schedule")}
              rules={view.rules ?? []}
              onRules={async (rules) => { await orchestratorAct("rules", "", "", JSON.stringify(rules)); await load(); }}
            />
          ) : null}
          {tab === "memory" ? (
            <>
              <Memory rows={memory} onAdd={async (text) => {
                await memoryWrite("profile", text);
                setMemory(await memoryList() as MemoryRow[]);
              }} />
              <p className="px-1 pb-4 text-[12px] leading-5 text-muted">
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
  incoming: Trigger[];
  onDecide: (id: string, kind: string) => Promise<void>;
  onDraft: (text: string) => void;
}) {
  const copy = useCopy();
  const [shown, setShown] = useState<{ slot: BriefSlot; brief: MorningBrief | null }>(() => readShown(readPrefs()));
  const [prefs, setPrefs] = useState<BriefPrefs>(() => readPrefs());
  const [waiting, setWaiting] = useState(false);
  const [prefsOpen, setPrefsOpen] = useState(false);
  const busy = useRef(false);
  const card = shown.brief;
  const nextSlot: BriefSlot = slotDue(prefs, "evening") ? "evening" : "morning";
  const heading = card?.welcome ? copy.work.briefWelcome : (card ? shown.slot : nextSlot) === "evening" ? copy.work.briefEvening : copy.work.briefMorning;
  const clock = new Date().toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  const today = new Date().toLocaleDateString(undefined, { month: "long", day: "numeric", weekday: "long" });
  useEffect(() => {
    const reload = () => setShown(readShown(prefs));
    window.addEventListener(BRIEF_EVENT, reload);
    return () => window.removeEventListener(BRIEF_EVENT, reload);
  }, [prefs]);
  async function generate() {
    if (card || busy.current) return;
    busy.current = true;
    setWaiting(true);
    try {
      const slot: BriefSlot = slotDue(prefs, "evening") ? "evening" : "morning";
      await runDeskFeed(briefPhrases(copy.work), slot);
      setShown(readShown(prefs));
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
  }
  function choose(id: string, value: string) {
    setShown((prev) => {
      if (!prev.brief) return prev;
      const next = { ...prev.brief, picked: { ...prev.brief.picked, [id]: value } };
      try {
        localStorage.setItem(briefStorageKey(prev.slot), serializeStoredBrief(briefDay(), next));
      } catch {
        /* 点选写不进去时，这一屏上仍然留着。 */
      }
      return { slot: prev.slot, brief: next };
    });
  }
  if (prefsOpen) {
    return (
      <BriefPrefsView
        prefs={prefs}
        onChange={savePrefs}
        onBack={() => setPrefsOpen(false)}
      />
    );
  }
  function useCard(card: BriefCard, which: "primary" | "secondary") {
    const text = briefPrompt(card, which);
    choose(card.id, which);
    if (text.trim()) {
      props.onDraft(text.trim());
      void noteFeedback("used");
    }
  }
  return (
    <div className="relative min-w-0 px-3 pb-8 pt-3">
      <button
        type="button"
        className="absolute right-3 top-3 grid size-8 place-items-center rounded-full bg-card text-muted shadow-[var(--shadow-card)] hover:text-foreground"
        aria-label={copy.work.briefSetPrefs}
        onClick={() => setPrefsOpen(true)}
      >
        <SlidersHorizontal className="size-4" />
      </button>
      <p className="text-center text-[12px] tabular-nums text-muted">{clock}</p>
      <p className="mt-0.5 text-center text-[12px] text-muted">{today}</p>
      <p className="mt-0.5 text-center text-[15px] font-medium">{heading}</p>
      {!card && !waiting ? (
        <div className="mt-8 flex flex-col items-center gap-3">
          <button type="button" className="h-10 rounded-full bg-foreground px-5 text-[13px] font-medium text-background" onClick={() => { void generate(); }}>
            {heading}
          </button>
          <p className="max-w-64 text-center text-[12px] leading-5 text-muted">{copy.work.briefReady}</p>
        </div>
      ) : null}
      {waiting ? <p className="py-10 text-center text-[14px] text-accent">{copy.work.briefRunning}</p> : null}
      {card ? <BriefStack brief={card} onUse={useCard} filledLabel={copy.work.briefFilled} /> : null}
      {card ? <BriefFeedback /> : null}
      {props.incoming.length > 0 ? (
        <ul className="mt-4 rounded-[24px] bg-card px-4 py-2 shadow-[var(--shadow-card)]">
          {props.incoming.slice(0, 3).map((row) => (
            <li key={row.id} className="py-2">
              <p className="line-clamp-2 text-[13px] leading-5">{row.text || row.id}</p>
              <div className="mt-2 flex gap-2">
                <TextButton onClick={() => { void props.onDecide(row.id, "accept"); }}>{copy.work.accept}</TextButton>
                <TextButton quiet onClick={() => { void props.onDecide(row.id, "ignore"); }}>{copy.work.ignore}</TextButton>
              </div>
            </li>
          ))}
        </ul>
      ) : null}
      <p className="mt-3 text-center text-[12px] leading-5 text-muted">{card?.welcome ? copy.work.briefWelcomeFoot : copy.work.briefFoot}</p>
    </div>
  );
}

function readShown(prefs: BriefPrefs): { slot: BriefSlot; brief: MorningBrief | null } {
  const day = briefDay();
  if (slotDue(prefs, "evening")) {
    const evening = parseStoredBrief(localStorage.getItem(briefStorageKey("evening")), day);
    if (evening) return { slot: "evening", brief: evening };
  }
  return { slot: "morning", brief: parseStoredBrief(localStorage.getItem(briefStorageKey("morning")), day) };
}

function readPrefs(): BriefPrefs {
  try {
    return parseBriefPrefs(localStorage.getItem(BRIEF_PREFS_KEY));
  } catch {
    return defaultBriefPrefs();
  }
}

export function BriefStack(props: {
  brief: MorningBrief;
  onUse: (card: BriefCard, which: "primary" | "secondary") => void;
  filledLabel: string;
}) {
  return (
    <div className="mt-5 space-y-3">
      {props.brief.cards.map((card) => {
        const picked = props.brief.picked[card.id] || "";
        const layout = card.layout || (card.choices.length >= 2 ? "compare" : card.items.length > 0 ? "checks" : "plain");
        return (
          <article key={card.id} className="min-w-0 rounded-[28px] bg-card px-5 py-5 shadow-[var(--shadow-card)]">
            <p className="text-[12px] text-muted">{card.kicker}</p>
            <h2 className="mt-2 text-[20px] font-semibold leading-7 tracking-[-0.03em]">{card.title}</h2>
            {card.lede ? <p className="mt-2 text-[13px] leading-6 text-muted">{card.lede}</p> : null}
            {layout === "checks" && card.items.length > 0 ? (
              <>
                {card.mark ? <p className="mt-4 text-[12px] text-muted">{card.mark}</p> : null}
                <div className="mt-2 grid grid-cols-2 gap-2">
                  {card.items.map((item, index) => (
                    <div key={`${item.title}-${index}`} className={cn("min-w-0 rounded-2xl bg-lift/50 px-3 py-3", card.items.length % 2 === 1 && index === card.items.length - 1 && "col-span-2")}>
                      <p className="text-[11px] font-medium tabular-nums text-muted">{String(index + 1).padStart(2, "0")}</p>
                      <p className="mt-1 text-[13px] font-medium">{item.title}</p>
                      {item.detail ? <p className="mt-1 text-[12px] leading-5 text-muted">{item.detail}</p> : null}
                    </div>
                  ))}
                </div>
              </>
            ) : null}
            {layout === "letter" ? (
              <div className={cn("mt-4 grid gap-2", card.items.length > 0 && card.choices.length > 0 ? "grid-cols-2" : "grid-cols-1")}>
                {card.items.length > 0 || card.choices.length === 0 ? (
                  <div className="min-w-0 rounded-2xl bg-lift/50 px-3 py-3">
                    <p className="text-[12px] font-medium text-muted">{card.leftLabel}</p>
                    <div className="mt-3 space-y-3">
                      {card.items.length === 0 ? <p className="text-[13px] text-muted">—</p> : card.items.map((item, index) => (
                        <div key={`${item.title}-${index}`}>
                          <p className="text-[12px] text-muted">{item.title}</p>
                          {item.detail ? <p className="mt-0.5 text-[13px] font-medium leading-5">{item.detail}</p> : null}
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
                {card.choices.length > 0 ? (
                  <div className="min-w-0 rounded-2xl bg-lift/50 px-3 py-3">
                    <p className="text-[12px] font-medium text-muted">{card.rightLabel}</p>
                    <div className="mt-3 space-y-3">
                      {card.choices.map((choice) => (
                        <div key={choice.id}>
                          <p className="text-[12px] text-muted">{choice.title}</p>
                          {choice.detail ? <p className="mt-0.5 text-[13px] font-medium leading-5">{choice.detail}</p> : null}
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            ) : null}
            {layout === "rows" && card.items.length > 0 ? (
              <>
                {card.mark ? <p className="mt-4 text-[12px] text-muted">{card.mark}</p> : null}
                <div className="mt-3 space-y-2">
                  {card.items.map((item, index) => (
                    <div key={`${item.title}-${index}`} className={cn("border-l-2 py-1 pl-3", item.tone === "stalled" ? "border-warning" : item.tone === "moving" ? "border-accent" : "border-border")}>
                      <p className="text-[13px] font-medium">{item.title}</p>
                      {item.detail ? <p className="mt-0.5 text-[12px] leading-5 text-muted">{item.detail}</p> : null}
                    </div>
                  ))}
                </div>
              </>
            ) : null}
            {layout === "stats" && card.stats && card.stats.length > 0 ? (
              <div className="mt-4 grid grid-cols-2 gap-2">
                {card.stats.map((stat) => (
                  <div key={stat.label} className="min-w-0 rounded-2xl bg-lift/50 px-3 py-3">
                    <p className="text-[22px] font-semibold tabular-nums tracking-[-0.03em]">{stat.value}</p>
                    <p className="mt-1 text-[12px] text-muted">{stat.label}</p>
                  </div>
                ))}
              </div>
            ) : null}
            {layout === "compare" && card.choices.length > 0 ? (
              <div className="mt-4 grid grid-cols-2 gap-2">
                {card.choices.slice(0, 2).map((choice) => (
                  <div key={choice.id} className="min-w-0 rounded-2xl bg-lift/50 px-3 py-3">
                    <p className="text-[13px] font-semibold leading-5">{choice.title}</p>
                    {choice.detail ? <p className="mt-2 whitespace-pre-line text-[12px] leading-5 text-muted">{choice.detail}</p> : null}
                  </div>
                ))}
              </div>
            ) : null}
            {layout === "evidence" && card.items.length > 0 ? (
              <div className="mt-4 space-y-2">
                {card.items.map((item, index) => (
                  <div key={`${item.title}-${index}`} className="min-w-0 rounded-2xl bg-lift/50 px-3 py-3">
                    <p className="text-[12px] font-medium">{item.title}</p>
                    {item.detail ? <p className="mt-1 text-[13px] leading-6">{item.detail}</p> : null}
                  </div>
                ))}
              </div>
            ) : null}
            {layout === "chips" && card.items.length > 0 ? (
              <div className="mt-4 flex flex-wrap gap-2">
                {card.items.map((item) => (
                  <span key={item.title} className="rounded-full bg-lift/60 px-3 py-1.5 text-[13px]">{item.title}</span>
                ))}
              </div>
            ) : null}
            {card.note ? <p className="mt-4 rounded-2xl bg-lift/50 px-3 py-3 text-[12px] leading-5 text-muted">{card.note}</p> : null}
            {card.primary || card.secondary ? (
              <div className="mt-4 flex flex-wrap gap-2">
                {card.primary ? (
                  <button type="button" className={cn("h-10 rounded-full px-4 text-[13px] font-medium", picked === "secondary" ? "bg-lift text-foreground" : "bg-accent text-white")} onClick={() => props.onUse(card, "primary")}>
                    {card.primary}
                  </button>
                ) : null}
                {card.secondary ? (
                  <button type="button" className={cn("h-10 rounded-full px-4 text-[13px] font-medium", picked === "secondary" ? "bg-foreground text-background" : "bg-lift text-foreground")} onClick={() => props.onUse(card, "secondary")}>
                    {card.secondary}
                  </button>
                ) : null}
              </div>
            ) : null}
            {picked && (card.primary || card.secondary) ? <p className="mt-2 text-[12px] text-muted">{props.filledLabel}</p> : null}
          </article>
        );
      })}
    </div>
  );
}

function BriefPrefsView(props: {
  prefs: BriefPrefs;
  onChange: (next: BriefPrefs) => void;
  onBack: () => void;
}) {
  const copy = useCopy();
  return (
    <div className="min-w-0 px-3 pb-8 pt-3">
      <button type="button" className="grid size-8 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground" onClick={props.onBack} aria-label={copy.work.taskBack}>
        <ChevronLeft className="size-4" />
      </button>
      <h1 className="mt-3 text-[20px] font-semibold tracking-[-0.03em]">{copy.work.briefPrefs}</h1>
      <p className="mt-2 text-[13px] leading-6 text-muted">{copy.work.briefPrefsHint}</p>
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
    </div>
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
    <section className="mt-4 rounded-[28px] bg-card px-5 py-5 shadow-[var(--shadow-card)]">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-[16px] font-semibold">{props.title}</h2>
        <button type="button" role="switch" aria-checked={props.on} aria-label={copy.work.briefStatus} className={cn("relative h-7 w-12 shrink-0 rounded-full transition-colors", props.on ? "bg-accent" : "bg-lift")} onClick={props.onToggle}>
          <span className={cn("absolute top-0.5 size-6 rounded-full bg-white shadow-sm transition-[left]", props.on ? "left-5" : "left-0.5")} />
        </button>
      </div>
      <label className={cn("mt-4 flex items-center justify-between gap-3", !props.on && "opacity-50")}>
        <span className="text-[13px] text-muted">{copy.work.briefSchedule}</span>
        <input type="time" value={props.at} disabled={!props.on} className="h-10 rounded-full bg-lift px-4 text-[14px] outline-none" onChange={(event) => props.onTime(event.target.value)} />
      </label>
    </section>
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

function BriefFeedback() {
  const copy = useCopy();
  const [noted, setNoted] = useState("");
  async function send(kind: "used" | "ignore" | "later" | "wrong") {
    await noteFeedback(kind);
    setNoted(copy.work.feedbackNoted);
  }
  const items = [
    ["used", copy.work.feedbackUsed],
    ["ignore", copy.work.feedbackIgnore],
    ["wrong", copy.work.feedbackWrong],
    ["later", copy.work.feedbackLater],
  ] as const;
  return (
    <div className="mt-3 px-1">
      <div className="flex flex-wrap gap-2">
        {items.map(([kind, label]) => (
          <button key={kind} type="button" className="h-8 rounded-full bg-card px-3 text-[12px] text-muted shadow-[var(--shadow-card)] hover:text-foreground" onClick={() => { void send(kind); }}>
            {label}
          </button>
        ))}
      </div>
      {noted ? <p className="mt-2 text-[12px] text-muted">{noted}</p> : null}
    </div>
  );
}

function Tasks(props: {
  open: Ticket[];
  ended: Ticket[];
  selected: string;
  onSelect: (id: string) => void;
  onNewTask: (text?: string) => void;
  held: { id: string; name: string; detail?: string }[];
  notes: { id: string; title: string }[];
  rules: Rule[];
  onRules: (rules: Rule[]) => Promise<void>;
}) {
  const copy = useCopy();
  const [compose, setCompose] = useState<null | { kind: "task" | "schedule"; text: string; ruleId?: string }>(null);
  const [query, setQuery] = useState("");
  const [scheduled, setScheduled] = useState(false);
  const suggestions = suggestionCopy(copy);
  const needle = query.trim().toLowerCase();
  const missing = unseenSuggestions(props.rules.map((rule) => rule.id)).filter((id) => {
    const item = suggestions[id];
    if (!item || !needle) return true;
    return `${item.title} ${item.text}`.toLowerCase().includes(needle);
  });
  const shown = filterTickets([...props.open, ...props.ended], "all", query);
  const openShown = shown.filter((ticket) => ticket.stage !== "ended");
  const endedShown = shown.filter((ticket) => ticket.stage === "ended");
  const rules = props.rules.filter((rule) => !needle || rule.text.toLowerCase().includes(needle));
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
        title={scheduled ? copy.work.createScheduled : copy.work.newTask}
        placeholder={scheduled ? copy.work.scheduleDraftPh : copy.work.taskDraftPh}
        hint={scheduled ? copy.work.scheduleDraftHint : copy.work.taskDraftHint}
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
        <section className="mb-4 rounded-[28px] bg-card px-4 py-4 shadow-[var(--shadow-card)]">
          <h2 className="text-[13px] font-medium">{copy.work.heldTitle}</h2>
          <div className="mt-3 space-y-2">
            {props.held.map((signal) => (
              <p key={signal.id} className="text-[13px] leading-5">{signal.name}{signal.detail ? <span className="text-muted"> · {signal.detail}</span> : null}</p>
            ))}
            {props.notes.map((note) => (
              <p key={note.id} className="text-[13px] leading-5">{note.title}</p>
            ))}
          </div>
        </section>
      ) : null}
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-[17px] font-semibold tracking-[-0.03em]">{copy.work.tasks}</h1>
        <button
          type="button"
          className="inline-flex h-8 shrink-0 items-center gap-1 rounded-full bg-card px-3 text-[13px] shadow-[var(--shadow-card)]"
          onClick={() => setCompose(scheduled ? { kind: "schedule", text: "" } : { kind: "task", text: "" })}
        >
          <Plus className="size-3.5" />
          {scheduled ? copy.work.createScheduled : copy.work.createTask}
        </button>
      </div>
      <div className="mt-4 grid h-10 grid-cols-2 gap-2">
        <label className={cn(
          "flex min-w-0 items-center gap-2 rounded-full px-3.5",
          scheduled ? "bg-lift/80 text-muted" : "bg-card text-foreground shadow-[var(--shadow-card)]",
        )}>
          <Search className="size-3.5 shrink-0 text-muted" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onFocus={() => setScheduled(false)}
            placeholder={copy.work.taskSearch}
            aria-label={copy.work.taskSearch}
            className="min-w-0 flex-1 bg-transparent text-[13px] outline-none placeholder:text-muted"
          />
        </label>
        <button
          type="button"
          aria-pressed={scheduled}
          className={cn(
            "flex items-center justify-center gap-1.5 rounded-full text-[13px] transition-colors duration-200",
            scheduled ? "bg-card text-foreground shadow-[var(--shadow-card)]" : "bg-lift/80 text-muted",
          )}
          onClick={() => setScheduled(true)}
        >
          <Clock className="size-3.5" />
          {copy.work.scheduled}
        </button>
      </div>
      {scheduled ? (
        <div className="mt-6">
          {rules.length > 0 ? (
            <ul className="space-y-2">
              {rules.map((rule) => (
                <li key={rule.id}>
                  <TaskPill
                    label={suggestions[rule.id]?.title || rule.text}
                    muted={!rule.enabled}
                    onClick={() => setCompose({ kind: "schedule", text: rule.text, ruleId: rule.id })}
                  />
                </li>
              ))}
            </ul>
          ) : null}
          {missing.length > 0 ? (
            <section className={rules.length > 0 ? "mt-6" : ""}>
              <h2 className="px-1 text-[15px] font-medium">{copy.work.recommended}</h2>
              <ul className="mt-2.5 space-y-2">
                {missing.map((id) => {
                  const item = suggestions[id];
                  if (!item) return null;
                  const Icon = item.icon;
                  return (
                    <li key={id}>
                      <button
                        type="button"
                        className="flex w-full items-center gap-3 rounded-[20px] bg-card px-4 py-3.5 text-left shadow-[var(--shadow-card)]"
                        onClick={() => setCompose({ kind: "schedule", text: item.text, ruleId: id })}
                      >
                        <Icon className="size-4 shrink-0 text-muted" />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-[14px] font-medium">{item.title}</span>
                          <span className="mt-0.5 block truncate text-[12px] leading-5 text-muted">{item.text}</span>
                        </span>
                        <Plus className="size-4 shrink-0 text-muted" />
                      </button>
                    </li>
                  );
                })}
              </ul>
            </section>
          ) : null}
          {rules.length === 0 && missing.length === 0 ? <Empty title={copy.work.tasksEmpty} hint={copy.work.tasksHint} /> : null}
        </div>
      ) : (
        <div>
          {openShown.length > 0 ? <TicketList tickets={openShown} onSelect={props.onSelect} /> : null}
          {endedShown.length > 0 ? <TicketList title={copy.work.earlier} tickets={endedShown} onSelect={props.onSelect} /> : null}
          {openShown.length === 0 && endedShown.length === 0 ? <Empty title={copy.work.tasksEmpty} hint={copy.work.tasksHint} /> : null}
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
        <h1 className="text-[16px] font-semibold">{props.title}</h1>
      </div>
      <div className="grid flex-1 place-items-center px-6 py-16">
        <p className="text-[14px] text-muted">{props.empty}</p>
      </div>
      <div className="space-y-1">
        {props.suggestions.map((item) => (
          <button key={item.id} type="button" className="flex w-full items-center gap-2 rounded-xl px-2 py-2 text-left text-[13px] text-muted hover:bg-lift hover:text-foreground" onClick={() => setText(item.text)}>
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
          className="w-full resize-none rounded-2xl bg-card px-3 py-3 text-[13px] leading-6 shadow-[var(--shadow-card)] outline-none placeholder:text-muted"
        />
      </label>
      <div className="mt-2 flex items-center justify-between gap-3">
        <p className="min-w-0 text-[12px] leading-5 text-muted">{props.hint}</p>
        <button
          type="button"
          className="h-8 shrink-0 rounded-full bg-foreground px-3 text-[12px] font-medium text-background disabled:opacity-40"
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

function TicketList(props: {
  title?: string;
  tickets: Ticket[];
  onSelect: (id: string) => void;
}) {
  return (
    <section className="mt-6">
      {props.title ? <h2 className="px-1 text-[15px] font-medium">{props.title}</h2> : null}
      <ul className={cn("space-y-2", props.title && "mt-2.5")}>
        {props.tickets.map((ticket) => (
          <li key={ticket.id}>
            <TaskPill label={ticket.title || ticket.id} onClick={() => props.onSelect(ticket.id)} />
          </li>
        ))}
      </ul>
    </section>
  );
}

function TaskPill(props: { label: string; muted?: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      className={cn(
        "flex h-11 w-full items-center rounded-full bg-card px-4 text-left text-[14px] shadow-[var(--shadow-card)]",
        props.muted && "text-muted",
      )}
      onClick={props.onClick}
    >
      <span className="truncate">{props.label}</span>
    </button>
  );
}

function Memory(props: { rows: MemoryRow[]; onAdd: (text: string) => Promise<void> }) {
  const copy = useCopy();
  const [folder, setFolder] = useState<MemoryFolder | "">("");
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const [saving, setSaving] = useState(false);
  const portrait = memoryPortrait(props.rows);
  const sep = /[\u4e00-\u9fff]/.test(copy.work.memoryProjectsLine) ? "、" : ", ";
  const said = portrait.said.filter((text) => text !== portrait.name);
  const lead = portrait.device || portrait.host || portrait.os || portrait.zone || portrait.git || portrait.hours || portrait.lines.length
    ? copy.work.memoryLeadMachine
    : said[0] || portrait.noted[0] || copy.work.memoryLeadEmpty;
  const title = portrait.name ? copy.work.memoryAbout.replace("{name}", portrait.name) : copy.work.memoryAboutYou;
  const when = portrait.updated ? memoryAgo(portrait.updated, copy.work) : copy.work.memoryNotYet;
  const folders: { id: MemoryFolder; label: string; icon: LucideIcon }[] = [
    { id: "device", label: copy.work.folderDevices, icon: Monitor },
    { id: "work", label: copy.work.folderWork, icon: Clock },
    { id: "project", label: copy.work.folderProject, icon: Folder },
    { id: "preference", label: copy.work.folderSemantic, icon: SlidersHorizontal },
    { id: "about", label: portrait.name ? copy.work.folderAbout.replace("{name}", portrait.name) : copy.work.folderAboutYou, icon: MessageCircle },
  ];
  const notes = memoryFolderRows(props.rows, "notes");
  if (notes.length) folders.push({ id: "notes", label: copy.work.folderNotes, icon: Folder });
  const shown = folder ? memoryFolderRows(props.rows, folder) : [];
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
    <div className="px-1 pb-2 pt-2">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-[22px] font-semibold tracking-[-0.03em]">{copy.work.memory}</h1>
        <button type="button" className="grid size-8 place-items-center rounded-full bg-card text-foreground shadow-[var(--shadow-card)]" aria-label={copy.work.memoryAdd} onClick={() => setAdding((on) => !on)}>
          <Plus className="size-4" />
        </button>
      </div>
      {adding ? (
        <form className="mt-4 rounded-[28px] bg-card px-5 py-4 shadow-[var(--shadow-card)]" onSubmit={(event) => { void add(event); }}>
          <input value={draft} onChange={(event) => setDraft(event.target.value)} placeholder={copy.work.memoryAddHint} className="h-10 w-full bg-transparent text-[14px] outline-none" />
          <div className="mt-2 flex justify-end">
            <button type="submit" disabled={!draft.trim() || saving} className="h-8 rounded-full bg-foreground px-3 text-[12px] font-medium text-background disabled:opacity-40">{copy.work.save}</button>
          </div>
        </form>
      ) : null}
      <article className="mt-4 rounded-[28px] bg-card px-5 py-5 shadow-[var(--shadow-card)]">
        <h2 className="text-[18px] font-semibold leading-7 tracking-[-0.03em]">{title}</h2>
        <p className="mt-1 text-[12px] text-muted">{portrait.updated ? copy.work.memoryUpdated.replace("{when}", when) : copy.work.memoryNotYet}</p>
        <p className="mt-4 text-[12px] text-muted">{copy.work.memorySummary}</p>
        <p className="mt-1 text-[14px] leading-7">{lead}</p>
        {portrait.device ? <p className="mt-3 text-[14px] leading-7">{copy.work.memoryDeviceLine.replace("{name}", portrait.device)}</p> : null}
        {portrait.host ? <p className="mt-2 text-[14px] leading-7">{copy.work.memoryHostLine.replace("{name}", portrait.host)}</p> : null}
        {portrait.os ? <p className="mt-2 text-[14px] leading-7">{copy.work.memoryOsLine.replace("{name}", portrait.os)}</p> : null}
        {portrait.zone ? <p className="mt-2 text-[14px] leading-7">{copy.work.memoryZoneLine.replace("{name}", portrait.zone)}</p> : null}
        {portrait.git ? <p className="mt-2 text-[14px] leading-7">{copy.work.memoryGitLine.replace("{name}", portrait.git)}</p> : null}
        {portrait.hours ? <p className="mt-2 text-[14px] leading-7">{copy.work.memoryHoursLine.replace("{hours}", portrait.hours)}</p> : null}
        {portrait.lines.length ? <p className="mt-2 text-[14px] leading-7">{copy.work.memoryProjectsLine.replace("{names}", portrait.lines.map((line) => line.name).join(sep))}</p> : null}
        {portrait.device || portrait.host || portrait.os || portrait.zone || portrait.git || portrait.hours || portrait.lines.length ? <p className="mt-3 text-[13px] leading-6 text-muted">{copy.work.memoryHardFact}</p> : null}
        {said.length ? <MemoryLines label={copy.work.memorySaid} lines={said} /> : null}
        {portrait.noted.length ? <MemoryLines label={copy.work.memoryNoted} lines={portrait.noted} /> : null}
        {portrait.preferences.length ? <MemoryLines label={copy.work.memoryPrefs} lines={portrait.preferences} /> : null}
      </article>
      <h2 className="mt-6 px-1 text-[15px] font-medium">{copy.work.folderAll}</h2>
      <div className="mt-2 grid grid-cols-2 gap-2">
        {folders.map((item) => {
          const count = memoryFolderRows(props.rows, item.id).length;
          const on = folder === item.id;
          const Icon = item.icon;
          return (
            <button
              key={item.id}
              type="button"
              className={cn("rounded-2xl bg-card px-4 py-4 text-left shadow-[var(--shadow-card)]", on && "ring-1 ring-accent")}
              onClick={() => setFolder(on ? "" : item.id)}
            >
              <Icon className="size-4 text-muted" />
              <p className="mt-3 text-[14px] font-medium">{item.label}</p>
              <p className="mt-1 text-[12px] text-muted">{copy.work.itemCount.replace("{n}", String(count))}</p>
            </button>
          );
        })}
      </div>
      {folder ? (
        shown.length === 0 ? <p className="mt-4 px-1 text-[13px] text-muted">{copy.work.memoryFolderEmpty}</p> : (
          <ul className="mt-4 space-y-2">
            {shown.map((row) => (
              <li key={row.id} className="rounded-2xl bg-card px-4 py-3.5 shadow-[var(--shadow-card)]">
                <p className="text-[14px] leading-6">{memoryRowTitle(row, copy.work)}</p>
                {memoryRowDetail(row, copy.work) ? <p className="mt-1 text-[12px] leading-5 text-muted">{memoryRowDetail(row, copy.work)}</p> : null}
              </li>
            ))}
          </ul>
        )
      ) : null}
    </div>
  );
}

function MemoryLines(props: { label: string; lines: string[] }) {
  return (
    <div className="mt-4">
      <p className="text-[12px] text-muted">{props.label}</p>
      {props.lines.map((line) => <p key={line} className="mt-1 text-[14px] leading-7">{line}</p>)}
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

function memoryAgo(iso: string, work: ReturnType<typeof useCopy>["work"]): string {
  const at = Date.parse(iso);
  if (Number.isNaN(at)) return work.memoryNotYet;
  const mins = Math.max(0, Math.round((Date.now() - at) / 60000));
  if (mins < 1) return work.memoryJustNow;
  if (mins < 60) return work.memoryMinutes.replace("{n}", String(mins));
  const hours = Math.round(mins / 60);
  if (hours < 24) return work.memoryHoursAgo.replace("{n}", String(hours));
  const days = Math.round(hours / 24);
  if (days < 14) return work.memoryDaysAgo.replace("{n}", String(days));
  return work.memoryWeeksAgo.replace("{n}", String(Math.round(days / 7)));
}

function Empty(props: { title: string; hint: string }) {
  return (
    <div className="px-3 py-6">
      <p className="text-[13px]">{props.title}</p>
      <p className="mt-1 text-[12px] leading-5 text-muted">{props.hint}</p>
    </div>
  );
}

function TextButton(props: { children: string; onClick: () => void; quiet?: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        "h-7 rounded-lg px-2.5 text-[12px] font-medium",
        props.quiet ? "text-muted hover:bg-lift hover:text-foreground" : "bg-foreground text-background",
      )}
      onClick={props.onClick}
    >
      {props.children}
    </button>
  );
}
