/**
 * 拥有：编排台的页。总览、来源、触发、评估、执行、设置，以及工单和线程的内页。
 * 不拥有：工单怎么执行。那一段在 Go 的 superagent。这一页自己占满主栏，不叠在消息上。
 */

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { getLocale, useCopy } from "../../lib/i18n";
import { orchestratorAct, orchestratorAdmission, orchestratorDecide, orchestratorDeliver, orchestratorMode, orchestratorPaused, orchestratorView } from "../../lib/client";
import {
  decisionsFor, parseRoute, popRoute, pushRoute, searchDesk, tabRoute, threadsFor, ticketsFor, triggerText,
  decisionFilters, settingGroups, tabs, threadFilters, ticketFilters,
  type DecisionFilter, type DeskView, type Hit, type Location, type Problem, type Route, type SettingsGroup, type Tab, type Ticket, type TicketFilter, type ThreadFilter,
} from "./desk-model";
import "./desk.css";

const routeKey = "yoyo-orchestrator-route";
type Say = (en: string, cn: string) => string;

/** readLocation 读上次停留的页。读不到时从总览开始。 */
function readLocation(): Location {
  try {
    const raw = JSON.parse(sessionStorage.getItem(routeKey) ?? "null") as { route?: unknown; stack?: unknown } | null;
    const stack = Array.isArray(raw?.stack) ? raw.stack.map(parseRoute) : [];
    return { route: parseRoute(raw?.route), stack };
  } catch {
    return { route: { page: "overview" }, stack: [] };
  }
}

/** OrchestratorDesk 每两秒先领走评估再拉快照，再按当前路线画一页。 */
export function OrchestratorDesk() {
  useCopy();
  const zh = getLocale() === "zh-CN";
  const say: Say = (en, cn) => (zh ? cn : en);
  const [location, setLocation] = useState<Location>(readLocation);
  const [view, setView] = useState<DeskView | null>(null);
  const [err, setErr] = useState("");
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<Hit[] | null>(null);
  const [glossary, setGlossary] = useState(false);
  const searchRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    try { sessionStorage.setItem(routeKey, JSON.stringify(location)); } catch { /* 路线只是方便回到原处。 */ }
  }, [location]);

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
    if (!hits) return;
    const close = (event: PointerEvent) => {
      if (!searchRef.current?.contains(event.target as Node)) setHits(null);
    };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [hits]);

  const go = (route: Route, replace = false) => {
    setHits(null);
    setLocation((current) => pushRoute(current, route, replace));
  };
  const act = async (kind: string, ticket = "", trigger = "", text = "", reason = "") => {
    try {
      await orchestratorAct(kind, ticket, trigger, text, reason);
      await load();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  };
  const decide = async (id: string, kind: string) => {
    try {
      await orchestratorDecide(id, kind, "");
      await load();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  };

  const route = location.route;
  const page = route.page;
  const back = location.stack.length ? location.stack[location.stack.length - 1] : undefined;
  const queued = view?.queued?.length ?? 0;
  const people = view?.tickets?.filter((row) => row.person).length ?? 0;

  return (
    <section className="orch orch-page" data-testid="orchestrator-desk">
      <header className="orch-bar">
        <h1 className="orch-title">{say("Orchestrator", "编排")}</h1>
        <form ref={searchRef} className="orch-search-wrap" role="search" onSubmit={(event) => { event.preventDefault(); setHits(view ? searchDesk(view, query) : []); }}>
          <input
            className="orch-search"
            type="search"
            value={query}
            placeholder={say("Find a ticket, thread, or trigger", "找工单、线程或触发")}
            aria-label={say("Search", "搜索")}
            onChange={(event) => {
              const next = event.target.value;
              setQuery(next);
              setHits(next.trim() && view ? searchDesk(view, next) : null);
            }}
            onKeyDown={(event) => { if (event.key === "Escape") setHits(null); }}
          />
          {hits ? <SearchPop hits={hits} say={say} onOpen={(hit) => { setQuery(""); go(hit.route); }} onClose={() => setHits(null)} /> : null}
        </form>
        <button type="button" className="orch-icon" aria-label={say("Glossary", "词表")} onClick={() => setGlossary(true)}>?</button>
        <span className={`orch-dot orch-${view?.health === "ready" || !view ? "ok" : "err"}`} title={view?.health || ""} />
      </header>
      {err ? <p className="orch-err" role="alert">{err}</p> : null}
      <div className="orch-tabs" role="tablist" aria-label={say("Orchestrator pages", "编排的页")}>
        {tabs.map((id) => (
          <button key={id} id={`orch-tab-${id}`} type="button" role="tab" aria-selected={page === id} className={page === id ? "on" : ""} onClick={() => setLocation({ route: tabRoute(id), stack: [] })}>
            {tabLabel(id, say)}
            {id === "overview" && people > 0 ? <span className="orch-badge">{people}</span> : null}
            {id === "trigger" && queued > 0 ? <span className="orch-badge">{queued}</span> : null}
          </button>
        ))}
      </div>
      <div className="orch-scroll">
        <main id="orch-page" role="tabpanel" aria-labelledby={`orch-tab-${page}`}>
          {!view ? <p className="orch-hint">{say("Loading", "正在读取")}</p> : (
            <DeskPage
              view={view}
              route={route}
              say={say}
              backLabel={back ? tabLabel(back.page, say) : ""}
              onBack={() => setLocation(popRoute)}
              onGo={go}
              onAct={act}
              onDecide={decide}
              onMode={(mode) => { void orchestratorMode(mode).then(load).catch((e) => setErr(String(e))); }}
              onPaused={(paused) => { void orchestratorPaused(paused).then(load).catch((e) => setErr(String(e))); }}
              onAdmission={(adm) => { void orchestratorAdmission(adm.listedUnaddressed, adm.unlistedAddressed, adm.unlistedUnaddressed).then(load).catch((e) => setErr(String(e))); }}
            />
          )}
        </main>
      </div>
      {glossary ? <Glossary say={say} onClose={() => setGlossary(false)} /> : null}
    </section>
  );
}

function DeskPage(props: {
  view: DeskView; route: Route; say: Say; backLabel: string;
  onBack: () => void; onGo: (route: Route, replace?: boolean) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
  onDecide: (id: string, kind: string) => Promise<void>;
  onMode: (mode: string) => void;
  onPaused: (paused: boolean) => void;
  onAdmission: (adm: DeskView["admission"]) => void;
}) {
  const { view, route, say } = props;
  if (route.page === "overview") return <Overview {...props} />;
  if (route.page === "source") return <Sources view={view} say={say} onGo={props.onGo} />;
  if (route.page === "trigger" && route.id) return <ThreadPage view={view} id={route.id} trigger={route.trigger} say={say} backLabel={props.backLabel} onBack={props.onBack} onGo={props.onGo} onAct={props.onAct} onDecide={props.onDecide} />;
  if (route.page === "trigger") return <Threads view={view} filter={route.filter} say={say} onGo={props.onGo} />;
  if (route.page === "eval") return <Evals view={view} filter={route.filter} say={say} onGo={props.onGo} onAct={props.onAct} />;
  if (route.page === "execution" && route.id) return <TicketPage view={view} id={route.id} say={say} backLabel={props.backLabel} onBack={props.onBack} onGo={props.onGo} onAct={props.onAct} />;
  if (route.page === "execution") return <Execution view={view} filter={route.filter} say={say} onGo={props.onGo} onPaused={props.onPaused} />;
  return <Settings view={view} group={route.page === "settings" ? route.group : "source"} say={say} onGo={props.onGo} onAct={props.onAct} onMode={props.onMode} onAdmission={props.onAdmission} />;
}

function Overview(props: {
  view: DeskView; say: Say;
  onGo: (route: Route, replace?: boolean) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
  onDecide: (id: string, kind: string) => Promise<void>;
}) {
  const { view, say, onGo } = props;
  const waiting = (view.tickets ?? []).filter((row) => row.person);
  const problems = view.problems ?? [];
  const flow = view.flow ?? { received: 0, filtered: 0, judged: 0, accepted: 0, ignored: 0 };
  const load = view.load ?? { running: 0, queued: 0, idle: 0, maxWorkers: view.workers || 3 };
  return (
    <>
      <p className="orch-intro">{say("Who is waiting, what arrived today, and whether the executor is busy.", "谁在等你、今天到了什么，以及执行器忙不忙。")}</p>
      <p className="orch-hint">
        {modeText(view.mode, say)}
        {view.paused ? ` · ${say("paused", "已暂停")}` : ""}
        {" · "}
        <button type="button" className="orch-link" onClick={() => onGo({ page: "settings", group: "eval" })}>{say("Change", "去改")}</button>
      </p>
      <Section title={say("Waiting on you", "待处理")} count={waiting.length}>
        {waiting.length ? waiting.map((ticket) => <TicketCard key={ticket.id} ticket={ticket} say={say} onOpen={() => onGo({ page: "execution", id: ticket.id, filter: "person" })} />) : <p className="orch-hint">{say("Nothing is waiting on a person.", "没有等人处理的工单。")}</p>}
      </Section>
      {(view.queued ?? []).length ? (
        <Section title={say("Waiting to be judged", "待评估")} count={view.queued.length}>
          <ul className="orch-stack">
            {view.queued.map((row) => (
              <li key={row.id} className="orch-line">
                <div>
                  <strong>{row.sender || row.source}</strong>
                  <p>{row.text}</p>
                </div>
                <span className="orch-actions">
                  <button type="button" onClick={() => { void props.onDecide(row.id, "accept"); }}>{say("Accept", "接单")}</button>
                  <button type="button" onClick={() => { void props.onDecide(row.id, "ignore"); }}>{say("Ignore", "忽略")}</button>
                </span>
              </li>
            ))}
          </ul>
        </Section>
      ) : null}
      {problems.length ? (
        <Section title={say("Problems", "故障")} count={problems.length}>
          <ul className="orch-stack">
            {problems.map((problem) => <ProblemLine key={`${problem.kind}-${problem.id}`} problem={problem} say={say} onGo={onGo} onAct={props.onAct} />)}
          </ul>
        </Section>
      ) : null}
      <Section title={say("Last 24 hours", "最近一天")}>
        <div className="orch-stats">
          <Stat label={say("Arrived", "到达")} value={flow.received} />
          <Stat label={say("Filtered", "滤掉")} value={flow.filtered} onOpen={flow.filtered ? () => onGo({ page: "source" }) : undefined} />
          <Stat label={say("Judged", "已评")} value={flow.judged} onOpen={() => onGo({ page: "eval", filter: "all" })} />
          <Stat label={say("Accepted", "接单")} value={flow.accepted} onOpen={() => onGo({ page: "eval", filter: "work" })} />
          <Stat label={say("Ignored", "忽略")} value={flow.ignored} onOpen={() => onGo({ page: "eval", filter: "ignore" })} />
        </div>
      </Section>
      <Section title={say("Executor", "执行")} action={<button type="button" className="orch-link" onClick={() => onGo({ page: "execution", filter: "open" })}>{say("Open", "打开")}</button>}>
        <div className="orch-stats">
          <Stat label={say("Running", "正在做")} value={`${load.running} / ${load.maxWorkers}`} onOpen={() => onGo({ page: "execution", filter: "working" })} />
          <Stat label={say("Queued", "排队")} value={load.queued} onOpen={() => onGo({ page: "execution", filter: "working" })} />
          <Stat label={say("Idle", "在等触发")} value={load.idle} onOpen={() => onGo({ page: "execution", filter: "idle" })} />
        </div>
      </Section>
    </>
  );
}

function Sources(props: { view: DeskView; say: Say; onGo: (route: Route) => void }) {
  const { say } = props;
  return (
    <>
      <p className="orch-intro">{say("Whether each source is reachable. Credentials live in settings.", "每个来源现在通不通。凭证在设置里改。")}</p>
      <ul className="orch-stack">
        {(props.view.sources ?? []).map((src) => (
          <li key={src.name} className="orch-card orch-card-static">
            <span className={`orch-stage orch-${src.state === "error" ? "err" : src.state === "ok" ? "work" : "idle"}`}>{sourceText(src.state, say)}</span>
            <span>
              <span className="orch-card-title">{src.name === "github" ? "GitHub" : say("Feishu", "飞书")}</span>
              <span className="orch-card-meta">{[src.repo, src.chat, src.app, src.hasSecret ? say("credential saved", "已保存凭证") : say("no credential", "还没有凭证"), src.detail].filter(Boolean).join(" · ")}</span>
            </span>
          </li>
        ))}
      </ul>
      <p><button type="button" className="orch-link" onClick={() => props.onGo({ page: "settings", group: "source" })}>{say("Edit sources", "改来源")}</button></p>
    </>
  );
}

function Threads(props: { view: DeskView; filter: ThreadFilter; say: Say; onGo: (route: Route, replace?: boolean) => void }) {
  const { say, filter } = props;
  const rows = threadsFor(props.view.threads ?? [], filter);
  return (
    <>
      <p className="orch-intro">{say("Triggers stay on a thread. Open one to see the messages and the tickets they reach.", "触发按线程归在一起。点开一条，看消息和它到达的工单。")}</p>
      <div className="orch-filters">
        {threadFilters.map((id) => (
          <button key={id} type="button" className={filter === id ? "on" : ""} onClick={() => props.onGo({ page: "trigger", filter: id }, true)}>{threadFilterText(id, say)}</button>
        ))}
      </div>
      {rows.length ? rows.map((thread) => (
        <button key={thread.key} type="button" className="orch-card" onClick={() => props.onGo({ page: "trigger", id: thread.key, filter })}>
          <span className={`orch-stage ${thread.evaluating ? "orch-work" : "orch-idle"}`}>{thread.evaluating ? say("Evaluating", "评估中") : thread.subs > 0 ? say("Subscribed", "已订阅") : say("Thread", "线程")}</span>
          <span>
            <span className="orch-card-title">{thread.title || thread.key}</span>
            <span className="orch-card-meta">{[thread.sender, say(`${thread.queued} waiting`, `${thread.queued} 条待评估`), say(`${thread.subs} tickets`, `${thread.subs} 张工单`)].filter(Boolean).join(" · ")}</span>
          </span>
        </button>
      )) : <p className="orch-hint">{filter === "subscribed" ? say("No ticket is subscribed to a thread yet. Switch to All to see every thread.", "还没有工单订阅的线程。切到「全部」能看到每条线程。") : say("Nothing in this filter.", "这个过滤下没有线程。")}</p>}
    </>
  );
}

function ThreadPage(props: {
  view: DeskView; id: string; trigger?: string; say: Say; backLabel: string;
  onBack: () => void; onGo: (route: Route) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
  onDecide: (id: string, kind: string) => Promise<void>;
}) {
  const { say, view } = props;
  const thread = (view.threads ?? []).find((row) => row.key === props.id);
  useEffect(() => {
    if (!props.trigger) return;
    document.getElementById(`orch-trigger-${props.trigger}`)?.scrollIntoView({ block: "center" });
  }, [props.trigger, thread?.items?.length]);
  if (!thread) {
    return <><Back label={props.backLabel} say={say} onBack={props.onBack} /><p className="orch-hint">{say("This thread is gone.", "这条线程不在了。")}</p></>;
  }
  const tickets = (thread.followers ?? []).map((id) => (view.tickets ?? []).find((row) => row.id === id)).filter((row): row is Ticket => !!row);
  return (
    <>
      <Back label={props.backLabel} say={say} onBack={props.onBack} />
      <h2 className="orch-page-title">{thread.title || thread.key}</h2>
      <p className="orch-hint">{[thread.sender, say(`${thread.queued} waiting`, `${thread.queued} 条待评估`)].filter(Boolean).join(" · ")}</p>
      {thread.subs > 0 ? <p><button type="button" onClick={() => { void props.onAct("unsubscribe", "", "", thread.key); }}>{say("Unsubscribe", "退订")}</button></p> : null}
      {tickets.length ? (
        <Section title={say("Tickets on this thread", "订阅这条线程的工单")} count={tickets.length}>
          {tickets.map((ticket) => <TicketCard key={ticket.id} ticket={ticket} say={say} onOpen={() => props.onGo({ page: "execution", id: ticket.id, filter: "open" })} />)}
        </Section>
      ) : null}
      <Section title={say("Messages", "消息")} count={thread.items?.length ?? 0}>
        {(thread.items ?? []).length ? (
          <ol className="orch-timeline">
            {(thread.items ?? []).map((item) => {
              const decision = (view.decisions ?? []).find((row) => row.triggerId === item.id && !row.dismissed);
              return (
                <li key={item.id} id={`orch-trigger-${item.id}`} className={props.trigger === item.id ? "orch-hit" : ""}>
                  <div className="orch-card-meta">{[item.sender || item.source, item.queued ? say("waiting", "待评估") : "", item.filtered ? say("filtered", "已滤掉") : "", decision ? decisionText(decision.kind, say) : ""].filter(Boolean).join(" · ")}</div>
                  <p>{item.text || say("Empty", "空")}</p>
                  {item.queued ? (
                    <span className="orch-actions">
                      <button type="button" onClick={() => { void props.onDecide(item.id, "accept"); }}>{say("Accept", "接单")}</button>
                      <button type="button" onClick={() => { void props.onDecide(item.id, "ignore"); }}>{say("Ignore", "忽略")}</button>
                      <button type="button" onClick={() => { void props.onAct("reevaluate", "", item.id); }}>{say("Requeue", "重新排队")}</button>
                    </span>
                  ) : null}
                  {decision?.kind === "unavailable" ? (
                    <span className="orch-actions">
                      <button type="button" onClick={() => { void props.onAct("reevaluate", "", item.id); }}>{say("Requeue", "重新排队")}</button>
                      <button type="button" onClick={() => { void props.onAct("dismiss", "", decision.id); }}>{say("Dismiss", "驳回")}</button>
                    </span>
                  ) : null}
                </li>
              );
            })}
          </ol>
        ) : <p className="orch-hint">{say("No messages on this thread yet.", "这条线程上还没有消息。")}</p>}
      </Section>
    </>
  );
}

function Evals(props: {
  view: DeskView; filter: DecisionFilter; say: Say;
  onGo: (route: Route, replace?: boolean) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
}) {
  const { say, view, filter } = props;
  const rows = [...decisionsFor(view.decisions ?? [], filter)].reverse();
  return (
    <>
      <p className="orch-intro">{say("What eval decided, and why.", "评估对触发做了什么决定，以及理由。")}</p>
      {view.paused ? <p className="orch-banner">{say("Eval is paused.", "评估停着。")}<button type="button" className="orch-link" onClick={() => props.onGo({ page: "execution", filter: "open" })}>{say("Resume on Execution", "到执行里恢复")}</button></p> : null}
      {(view.evals ?? []).length ? <p><button type="button" className="orch-link" onClick={() => props.onGo({ page: "trigger", filter: "evaluating" })}>{say(`${view.evals.length} evals are still open`, `${view.evals.length} 轮评估还没结束`)}</button></p> : null}
      <div className="orch-filters">
        {decisionFilters.map((id) => (
          <button key={id} type="button" className={filter === id ? "on" : ""} onClick={() => props.onGo({ page: "eval", filter: id }, true)}>{decisionFilterText(id, say)}</button>
        ))}
      </div>
      {rows.length ? (
        <ul className="orch-stack">
          {rows.map((row) => {
            const text = triggerText(view, row.triggerId);
            return (
              <li key={row.id} className="orch-line">
                <div>
                  {row.thread ? <button type="button" className="orch-link" onClick={() => props.onGo({ page: "trigger", id: row.thread, filter: "subscribed", trigger: row.triggerId })}>{text || row.thread}</button> : <strong>{text || row.triggerId}</strong>}
                  <p className="orch-card-meta">{[decisionText(row.kind, say), row.reason].filter(Boolean).join(" · ")}</p>
                </div>
                {row.kind === "unavailable" && !row.dismissed ? (
                  <span className="orch-actions">
                    <button type="button" onClick={() => { void props.onAct("reevaluate", "", row.triggerId); }}>{say("Requeue", "重新排队")}</button>
                    <button type="button" onClick={() => { void props.onAct("dismiss", "", row.id); }}>{say("Dismiss", "驳回")}</button>
                  </span>
                ) : null}
              </li>
            );
          })}
        </ul>
      ) : <p className="orch-hint">{say("No decisions in this filter.", "这个过滤下没有决定。")}</p>}
    </>
  );
}

function Execution(props: {
  view: DeskView; filter: TicketFilter; say: Say;
  onGo: (route: Route, replace?: boolean) => void;
  onPaused: (paused: boolean) => void;
}) {
  const { say, view, filter } = props;
  const [confirm, setConfirm] = useState(false);
  const rows = ticketsFor(view.tickets ?? [], filter);
  return (
    <>
      <p className="orch-intro">{say("The executor, and tickets grouped by who they wait on.", "执行器的状态，以及按在等谁分组的工单。")}</p>
      <div className="orch-status">
        <span>{view.paused ? say("Paused", "已暂停") : say("Running", "在跑")}</span>
        <span className="orch-card-meta">{say(`${view.load?.running ?? 0} / ${view.load?.maxWorkers ?? view.workers ?? 3} running`, `${view.load?.running ?? 0} / ${view.load?.maxWorkers ?? view.workers ?? 3} 正在做`)}</span>
        <button type="button" onClick={() => setConfirm(true)}>{view.paused ? say("Resume", "恢复") : say("Pause", "暂停")}</button>
      </div>
      {confirm ? <p className="orch-banner">{view.paused ? say("Resume dispatch?", "恢复派工？") : say("Pause stops new work from starting.", "暂停后不再派新的工单。")}<button type="button" onClick={() => { setConfirm(false); props.onPaused(!view.paused); }}>{say("Confirm", "确认")}</button></p> : null}
      <div className="orch-filters">
        {ticketFilters.map((id) => (
          <button key={id} type="button" className={filter === id ? "on" : ""} onClick={() => props.onGo({ page: "execution", filter: id }, true)}>{ticketFilterText(id, say)}</button>
        ))}
      </div>
      {rows.length ? rows.map((ticket) => <TicketCard key={ticket.id} ticket={ticket} say={say} onOpen={() => props.onGo({ page: "execution", id: ticket.id, filter })} />) : <p className="orch-hint">{say("No tickets in this filter.", "这个过滤下没有工单。")}</p>}
    </>
  );
}

function TicketPage(props: {
  view: DeskView; id: string; say: Say; backLabel: string;
  onBack: () => void; onGo: (route: Route) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
}) {
  const { say } = props;
  const ticket = (props.view.tickets ?? []).find((row) => row.id === props.id) ?? null;
  const [reply, setReply] = useState("");
  const [purpose, setPurpose] = useState("answer");
  const [note, setNote] = useState("");
  const [more, setMore] = useState(false);
  const [confirm, setConfirm] = useState("");
  const [trigger, setTrigger] = useState("");
  const [thread, setThread] = useState("");
  if (!ticket) return <><Back label={props.backLabel} say={say} onBack={props.onBack} /><p className="orch-hint">{say("This ticket is gone.", "这张工单不在了。")}</p></>;
  const canRetry = ticket.action === "failed" || ticket.action === "blocked" || ticket.action === "delivery";
  return (
    <>
      <Back label={props.backLabel} say={say} onBack={props.onBack} />
      <div className="orch-page-head">
        <h2 className="orch-page-title">{ticket.title || ticket.id}</h2>
        <span className="orch-actions">
          {ticket.stage !== "ended" ? <button type="button" onClick={() => setConfirm(confirm === "stop" ? "" : "stop")}>{say("Stop", "停止")}</button> : null}
          {canRetry ? <button type="button" onClick={() => setConfirm(confirm === "retry" ? "" : "retry")}>{say("Retry", "重试")}</button> : null}
          <button type="button" onClick={() => { void props.onAct("requestStatus", ticket.id); }}>{say("Ask status", "问状态")}</button>
          {ticket.originUrl ? <a href={ticket.originUrl} target="_blank" rel="noreferrer">{say("Open source", "打开来源")}</a> : null}
        </span>
      </div>
      {confirm === "stop" ? <p className="orch-banner">{say("Stop this ticket?", "停掉这张工单？")}<button type="button" onClick={() => { setConfirm(""); void props.onAct("stop", ticket.id, "", "", "stopped"); }}>{say("Confirm", "确认")}</button></p> : null}
      {confirm === "retry" ? <p className="orch-banner">{say("Start another attempt?", "再派一次？")}<button type="button" onClick={() => { setConfirm(""); void props.onAct("retry", ticket.id); }}>{say("Confirm", "确认")}</button></p> : null}
      <p className="orch-hint">{[stageText(ticket, say), ticket.strength ? modeText(ticket.strength, say) : "", ticket.sender].filter(Boolean).join(" · ")}</p>
      {ticket.attention ? <p className="orch-banner">{ticket.attention}</p> : null}
      {ticket.deliveryError ? <p className="orch-err">{ticket.deliveryError}</p> : null}
      <Section title={say("Inputs", "输入")} count={ticket.inputs?.length ?? 0}>
        {(ticket.inputs ?? []).length ? (ticket.inputs ?? []).map((input) => (
          <p key={input.id} className="orch-line">{input.delivered ? "" : say("Not with the executor yet · ", "还没交给执行器 · ")}{input.text}</p>
        )) : <p className="orch-hint">{say("No input yet.", "还没有输入。")}</p>}
      </Section>
      {(ticket.follows ?? []).length ? (
        <Section title={say("Threads", "线程")} count={(ticket.follows ?? []).length}>
          {(ticket.follows ?? []).map((follow) => (
            <button key={follow.thread} type="button" className="orch-link" onClick={() => props.onGo({ page: "trigger", id: follow.thread, filter: "subscribed" })}>
              {threadTitle(props.view, follow.thread)}{follow.closed ? ` · ${follow.closed}` : ""}
            </button>
          ))}
        </Section>
      ) : null}
      {(ticket.assignments ?? []).length ? (
        <Section title={say("Attempts", "派工")} count={(ticket.assignments ?? []).length}>
          {(ticket.assignments ?? []).map((asg, index) => (
            <p key={asg.id} className="orch-line">{say(`Attempt ${index + 1}`, `第 ${index + 1} 次`)} · {[assignmentText(asg.state, say), asg.report ? reportText(asg.report, say) : "", asg.summary].filter(Boolean).join(" · ")}</p>
          ))}
        </Section>
      ) : null}
      {ticket.stage !== "ended" ? (
        <form className="orch-form" onSubmit={(event) => { event.preventDefault(); void props.onAct("reply", ticket.id, ticket.origin, reply, purpose).then(() => setReply("")); }}>
          <label>
            <span>{say("Reply", "回复")}</span>
            <select value={purpose} onChange={(event) => setPurpose(event.target.value)}>
              <option value="acknowledgement">{say("Got it", "确认收到")}</option>
              <option value="answer">{say("Answer", "回答")}</option>
              <option value="result">{say("Result", "结果")}</option>
              <option value="attention">{say("Needs a person", "请人看")}</option>
            </select>
          </label>
          <textarea rows={3} value={reply} placeholder={say("What should be sent back", "要发回去的话")} onChange={(event) => setReply(event.target.value)} />
          <button type="submit">{say("Send", "发出")}</button>
        </form>
      ) : null}
      <button type="button" className="orch-link" onClick={() => setMore((open) => !open)}>{more ? say("Hide more", "收起") : say("Attach, note, follow", "挂上、备注、跟随")}</button>
      {more ? (
        <div className="orch-form">
          <textarea rows={2} value={note} placeholder={say("Note", "备注")} onChange={(event) => setNote(event.target.value)} />
          <button type="button" onClick={() => { void props.onAct("note", ticket.id, "", note).then(() => setNote("")); }}>{say("Save note", "记下")}</button>
          <input value={trigger} placeholder={say("Trigger id", "触发编号")} onChange={(event) => setTrigger(event.target.value)} />
          <span className="orch-actions">
            <button type="button" onClick={() => { void props.onAct("attach", ticket.id, trigger, note); }}>{say("Attach", "挂上")}</button>
            <button type="button" onClick={() => { void props.onAct("handback", ticket.id, trigger, "", note); }}>{say("Hand back", "交还")}</button>
          </span>
          <input value={thread} placeholder={say("Thread", "线程")} onChange={(event) => setThread(event.target.value)} />
          <span className="orch-actions">
            <button type="button" onClick={() => { void props.onAct("follow", ticket.id, "", thread); }}>{say("Follow", "跟随")}</button>
            <button type="button" onClick={() => { void props.onAct("markUnrelated", ticket.id, "", thread); }}>{say("Unrelated", "无关")}</button>
          </span>
        </div>
      ) : null}
    </>
  );
}

function Settings(props: {
  view: DeskView; group: SettingsGroup; say: Say;
  onGo: (route: Route, replace?: boolean) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
  onMode: (mode: string) => void;
  onAdmission: (adm: DeskView["admission"]) => void;
}) {
  const { say, view, group } = props;
  return (
    <>
      <p className="orch-intro">{say("Sources, eval, and execution are separate. Prompts stay in the harness lab.", "来源、评估和执行分开设置。提示词仍在 harness 实验室。")}</p>
      <div className="orch-filters">
        {settingGroups.map((id) => (
          <button key={id} type="button" className={group === id ? "on" : ""} onClick={() => props.onGo({ page: "settings", group: id }, true)}>{settingsText(id, say)}</button>
        ))}
      </div>
      {group === "source" ? <SourceForm view={view} say={say} onAct={props.onAct} /> : null}
      {group === "eval" ? <EvalForm view={view} say={say} onAct={props.onAct} onMode={props.onMode} onAdmission={props.onAdmission} /> : null}
      {group === "execution" ? <WorkerForm view={view} say={say} onAct={props.onAct} /> : null}
    </>
  );
}

function SourceForm(props: { view: DeskView; say: Say; onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void> }) {
  const { say } = props;
  const github = (props.view.sources ?? []).find((src) => src.name === "github");
  const feishu = (props.view.sources ?? []).find((src) => src.name === "feishu");
  const [repo, setRepo] = useState(github?.repo || "");
  const [token, setToken] = useState("");
  const [clearToken, setClearToken] = useState(false);
  const [appId, setAppId] = useState(feishu?.app || "");
  const [chat, setChat] = useState(feishu?.chat || "");
  const [secret, setSecret] = useState("");
  const [clearSecret, setClearSecret] = useState(false);
  return (
    <form className="orch-form" onSubmit={(event) => {
      event.preventDefault();
      void props.onAct("sources", "", "", JSON.stringify({
        githubRepo: repo, githubToken: token, clearGithubToken: clearToken,
        feishuAppId: appId, feishuChat: chat, feishuSecret: secret, clearFeishuSecret: clearSecret,
      }));
      setToken("");
      setSecret("");
      setClearToken(false);
      setClearSecret(false);
    }}>
      <label><span>GitHub repo</span><input value={repo} onChange={(event) => setRepo(event.target.value)} /></label>
      <label><span>{github?.hasSecret ? say("Token saved. Leave blank to keep it.", "令牌已保存，留空则保持。") : "GitHub token"}</span><input type="password" value={token} onChange={(event) => setToken(event.target.value)} /></label>
      <label className="orch-check"><input type="checkbox" checked={clearToken} onChange={(event) => setClearToken(event.target.checked)} /><span>{say("Clear token", "清除令牌")}</span></label>
      <label><span>{say("Feishu app", "飞书应用")}</span><input value={appId} onChange={(event) => setAppId(event.target.value)} /></label>
      <label><span>{say("Feishu chat", "飞书会话")}</span><input value={chat} onChange={(event) => setChat(event.target.value)} /></label>
      <label><span>{feishu?.hasSecret ? say("Secret saved. Leave blank to keep it.", "密钥已保存，留空则保持。") : say("Feishu secret", "飞书密钥")}</span><input type="password" value={secret} onChange={(event) => setSecret(event.target.value)} /></label>
      <label className="orch-check"><input type="checkbox" checked={clearSecret} onChange={(event) => setClearSecret(event.target.checked)} /><span>{say("Clear secret", "清除密钥")}</span></label>
      <button type="submit">{say("Save sources", "保存来源")}</button>
    </form>
  );
}

function EvalForm(props: {
  view: DeskView; say: Say;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
  onMode: (mode: string) => void;
  onAdmission: (adm: DeskView["admission"]) => void;
}) {
  const { say, view } = props;
  const adm = view.admission ?? { listedUnaddressed: "inferred", unlistedAddressed: "inferred", unlistedUnaddressed: "ignore" };
  const [seconds, setSeconds] = useState(String(view.evalTimeout || 120));
  const [allowed, setAllowed] = useState((view.allowed ?? []).join(", "));
  const setCell = (key: keyof DeskView["admission"], value: string) => props.onAdmission({ ...adm, [key]: value });
  return (
    <div className="orch-form">
      <div className="orch-filters" aria-label={say("Mode", "模式")}>
        {(["off", "observe", "auto"] as const).map((mode) => (
          <button key={mode} type="button" className={view.mode === mode ? "on" : ""} onClick={() => props.onMode(mode)}>{modeText(mode, say)}</button>
        ))}
      </div>
      <p className="orch-hint">{say("Off ignores new work. Observe records it and waits for you to accept. Auto accepts what the policy allows.", "关闭不再接新工作。观察会记下来等你接。自动会接策略允许的。")}</p>
      <label><span>{say("Eval timeout, seconds", "评估时限，秒")}</span><input value={seconds} onChange={(event) => setSeconds(event.target.value)} /></label>
      <button type="button" onClick={() => { void props.onAct("timeout", "", "", String(Number(seconds) || 120)); }}>{say("Save timeout", "保存时限")}</button>
      <Cell label={say("Listed, not addressed", "在名单里，没被点名")} value={adm.listedUnaddressed} say={say} onChange={(value) => setCell("listedUnaddressed", value)} />
      <Cell label={say("Addressed, not listed", "被点名，不在名单里")} value={adm.unlistedAddressed} say={say} onChange={(value) => setCell("unlistedAddressed", value)} />
      <Cell label={say("Neither", "两者都不是")} value={adm.unlistedUnaddressed} say={say} onChange={(value) => setCell("unlistedUnaddressed", value)} />
      <label><span>{say("Allowed senders", "允许的发送者")}</span><input value={allowed} onChange={(event) => setAllowed(event.target.value)} /></label>
      <button type="button" onClick={() => { void props.onAct("allowed", "", "", allowed); }}>{say("Save senders", "保存名单")}</button>
    </div>
  );
}

function WorkerForm(props: { view: DeskView; say: Say; onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void> }) {
  const { say } = props;
  const [workers, setWorkers] = useState(String(props.view.workers || 3));
  return (
    <form className="orch-form" onSubmit={(event) => { event.preventDefault(); void props.onAct("workers", "", "", String(Number(workers) || 3)); }}>
      <label><span>{say("Worker slots", "同时做几张工单")}</span><input value={workers} onChange={(event) => setWorkers(event.target.value)} /></label>
      <p className="orch-hint">{say("Pause and resume stay on the Execution page.", "暂停和恢复在执行那一页。")}</p>
      <button type="submit">{say("Save slots", "保存槽位")}</button>
    </form>
  );
}

function TicketCard(props: { ticket: Ticket; say: Say; onOpen: () => void }) {
  const { ticket, say } = props;
  return (
    <button type="button" className="orch-card" onClick={props.onOpen}>
      <span className={`orch-stage orch-${ticket.person ? "person" : ticket.stage === "working" ? "work" : "idle"}`}>{stageText(ticket, say)}</span>
      <span>
        <span className="orch-card-title">{ticket.title || ticket.id}</span>
        <span className="orch-card-meta">{[ticket.sender, ticket.attention].filter(Boolean).join(" · ")}</span>
      </span>
    </button>
  );
}

function ProblemLine(props: {
  problem: Problem; say: Say;
  onGo: (route: Route) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
}) {
  const { problem, say } = props;
  const route = problemRoute(problem);
  return (
    <li className="orch-line">
      <div>
        {route ? <button type="button" className="orch-link" onClick={() => props.onGo(route)}>{problem.title}</button> : <strong>{problem.title}</strong>}
        <p className="orch-card-meta">{[problemText(problem.kind, say), problem.detail].filter(Boolean).join(" · ")}</p>
      </div>
      {problem.kind === "unavailable" ? (
        <span className="orch-actions">
          <button type="button" onClick={() => { void props.onAct("reevaluate", "", problem.triggerId || ""); }}>{say("Requeue", "重新排队")}</button>
          <button type="button" onClick={() => { void props.onAct("dismiss", "", problem.decisionId || problem.id); }}>{say("Dismiss", "驳回")}</button>
        </span>
      ) : null}
    </li>
  );
}

function problemRoute(problem: Problem): Route | null {
  if (problem.kind === "source") return { page: "source" };
  if ((problem.kind === "overdue" || problem.kind === "unavailable") && problem.thread) {
    return { page: "trigger", id: problem.thread, filter: "subscribed", ...(problem.triggerId ? { trigger: problem.triggerId } : {}) };
  }
  if (problem.kind === "outgoing" && problem.ticketId) return { page: "execution", id: problem.ticketId, filter: "open" };
  return null;
}

function Section(props: { title: string; count?: number; action?: ReactNode; children: ReactNode }) {
  return (
    <section className="orch-section">
      <div className="orch-section-head">
        <h2>{props.title}{props.count !== undefined ? <span className="orch-count">{props.count}</span> : null}</h2>
        {props.action}
      </div>
      {props.children}
    </section>
  );
}

function Stat(props: { label: string; value: number | string; onOpen?: () => void }) {
  const body = (<><b>{props.value}</b><span>{props.label}</span></>);
  if (!props.onOpen) return <span className="orch-stat">{body}</span>;
  return <button type="button" className="orch-stat" onClick={props.onOpen}>{body}</button>;
}

function Back(props: { label: string; say: Say; onBack: () => void }) {
  return <button type="button" className="orch-back" onClick={props.onBack}>{props.say("Back", "返回")}{props.label ? ` · ${props.label}` : ""}</button>;
}

function SearchPop(props: { hits: Hit[]; say: Say; onOpen: (hit: Hit) => void; onClose: () => void }) {
  const { say } = props;
  return (
    <div className="orch-search-pop" role="region" aria-label={say("Search", "搜索")}>
      <div className="orch-search-head">
        <span className="orch-hint">{props.hits.length ? say(`${props.hits.length} matches`, `${props.hits.length} 条`) : say("No matches", "没有匹配")}</span>
        <button type="button" className="orch-icon" aria-label={say("Close", "关闭")} onClick={props.onClose}>×</button>
      </div>
      {props.hits.length ? (
        <ul>
          {props.hits.map((hit) => (
            <li key={`${hit.kind}-${hit.id}`}>
              <button type="button" onClick={() => props.onOpen(hit)}>
                <span className="orch-card-meta">{hitText(hit.kind, say)}</span>
                <span className="orch-card-title">{hit.title}</span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function Glossary(props: { say: Say; onClose: () => void }) {
  const { say } = props;
  const rows: Array<[string, string, string]> = [
    ["Ticket", "工单", say("Authorized work the executor can do.", "已经授权、执行器可以做的一件工作。")],
    ["Thread", "线程", say("The conversation a trigger belongs to.", "一条触发所在的那段对话。")],
    ["Trigger", "触发", say("One incoming message.", "进来的一条消息。")],
    ["Eval", "评估", say("The decision to accept, ignore, or stop.", "决定接、忽略还是停下。")],
    ["Execution", "执行", say("A worker doing a ticket.", "工人按工单去做。")],
    ["Source", "来源", say("GitHub or Feishu.", "GitHub 或飞书。")],
  ];
  return (
    <div className="orch-modal" role="presentation" onClick={props.onClose}>
      <div className="orch-modal-card" role="dialog" aria-label={say("Glossary", "词表")} onClick={(event) => event.stopPropagation()}>
        <h2>{say("Words", "词")}</h2>
        <dl>
          {rows.map(([en, cn, meaning]) => (
            <div key={en}>
              <dt>{say(en, cn)}</dt>
              <dd>{meaning}</dd>
            </div>
          ))}
        </dl>
        <button type="button" onClick={props.onClose}>{say("Close", "关闭")}</button>
      </div>
    </div>
  );
}

function Cell(props: { label: string; value: string; say: Say; onChange: (value: string) => void }) {
  return (
    <label>
      <span>{props.label}</span>
      <select value={props.value || "inferred"} onChange={(event) => props.onChange(event.target.value)}>
        <option value="explicit">{props.say("Explicit", "明确")}</option>
        <option value="inferred">{props.say("Inferred", "推断")}</option>
        <option value="ignore">{props.say("Ignore", "忽略")}</option>
      </select>
    </label>
  );
}

function tabLabel(id: Tab, say: Say) {
  const text: Record<Tab, [string, string]> = {
    overview: ["Overview", "总览"],
    source: ["Source", "来源"],
    trigger: ["Trigger", "触发"],
    eval: ["Eval", "评估"],
    execution: ["Execution", "执行"],
    settings: ["Settings", "设置"],
  };
  return say(text[id][0], text[id][1]);
}

function threadFilterText(id: ThreadFilter, say: Say) {
  const text: Record<ThreadFilter, [string, string]> = {
    subscribed: ["Subscribed", "已订阅"],
    evaluating: ["Evaluating", "评估中"],
    recent: ["Last 24 hours", "最近一天"],
    all: ["All", "全部"],
  };
  return say(text[id][0], text[id][1]);
}

function ticketFilterText(id: TicketFilter, say: Say) {
  const text: Record<TicketFilter, [string, string]> = {
    open: ["Open", "进行中"],
    person: ["Waiting on you", "待处理"],
    working: ["Working", "执行中"],
    idle: ["Waiting for a trigger", "在等触发"],
    ended: ["Ended", "已结束"],
  };
  return say(text[id][0], text[id][1]);
}

function decisionFilterText(id: DecisionFilter, say: Say) {
  const text: Record<DecisionFilter, [string, string]> = {
    work: ["Accepted", "接单"],
    stop: ["Stopped", "停止"],
    ignore: ["Ignored", "忽略"],
    unavailable: ["Unfinished", "未完成"],
    all: ["All", "全部"],
  };
  return say(text[id][0], text[id][1]);
}

function settingsText(id: SettingsGroup, say: Say) {
  const text: Record<SettingsGroup, [string, string]> = {
    source: ["Source", "来源"],
    eval: ["Eval", "评估"],
    execution: ["Execution", "执行"],
  };
  return say(text[id][0], text[id][1]);
}

function stageText(ticket: Ticket, say: Say) {
  if (ticket.stage === "ended") return outcomeText(ticket.outcome || "ended", say);
  if (ticket.stage === "idle") return say("Waiting for a trigger", "在等新触发");
  if (ticket.action === "queued") return say("Queued", "排队");
  if (ticket.action === "delivery") return say("Delivery failed", "投递失败");
  if (ticket.action === "answer") return say("Needs an answer", "等人回答");
  if (ticket.action === "review") return say("Needs a look", "等人看结果");
  if (ticket.action === "blocked") return say("Blocked", "卡住");
  if (ticket.action === "failed") return say("Failed", "失败");
  if (ticket.stage === "working") return say("Working", "执行中");
  return ticket.stage;
}

function outcomeText(outcome: string, say: Say) {
  const text: Record<string, [string, string]> = {
    completed: ["Completed", "完成"],
    dropped: ["Dropped", "放弃"],
    stopped: ["Stopped", "已停"],
    merged: ["Merged", "已合并"],
    closed: ["Closed", "已关闭"],
    ended: ["Ended", "已结束"],
  };
  const pair = text[outcome];
  return pair ? say(pair[0], pair[1]) : outcome;
}

function decisionText(kind: string, say: Say) {
  const text: Record<string, [string, string]> = {
    accept: ["Accepted", "接单"],
    stop: ["Stopped", "停止"],
    ignore: ["Ignored", "忽略"],
    unavailable: ["Unfinished", "未完成"],
  };
  const pair = text[kind];
  return pair ? say(pair[0], pair[1]) : kind;
}

function modeText(mode: string, say: Say) {
  const text: Record<string, [string, string]> = {
    off: ["Off", "关闭"],
    observe: ["Observe", "观察"],
    auto: ["Auto", "自动"],
    explicit: ["Explicit", "明确"],
    inferred: ["Inferred", "推断"],
  };
  const pair = text[mode];
  return pair ? say(pair[0], pair[1]) : mode;
}

function sourceText(state: string, say: Say) {
  if (state === "unconfigured") return say("Not configured", "未配置");
  if (state === "ok") return say("Reachable", "通");
  if (state === "error") return say("Error", "出错");
  return state;
}

function assignmentText(state: string, say: Say) {
  if (state === "pending") return say("Queued", "排队");
  if (state === "running") return say("Running", "在做");
  if (state === "ended") return say("Ended", "结束");
  return state;
}

function reportText(state: string, say: Say) {
  const text: Record<string, [string, string]> = {
    progress: ["Progress", "有进展"],
    waiting: ["Waiting", "在等"],
    ready: ["Ready", "做完了"],
    blocked: ["Blocked", "卡住"],
    failed: ["Failed", "失败"],
    dropped: ["Dropped", "放弃"],
  };
  const pair = text[state];
  return pair ? say(pair[0], pair[1]) : state;
}

function problemText(kind: string, say: Say) {
  const text: Record<string, [string, string]> = {
    source: ["Source", "来源"],
    unavailable: ["Eval unfinished", "评估没完成"],
    outgoing: ["Reply failed", "回复没发出"],
    overdue: ["Eval overdue", "评估超时"],
  };
  const pair = text[kind];
  return pair ? say(pair[0], pair[1]) : kind;
}

function hitText(kind: Hit["kind"], say: Say) {
  if (kind === "ticket") return say("Ticket", "工单");
  if (kind === "thread") return say("Thread", "线程");
  return say("Trigger", "触发");
}

function threadTitle(view: DeskView, key: string) {
  return (view.threads ?? []).find((row) => row.key === key)?.title || key;
}
