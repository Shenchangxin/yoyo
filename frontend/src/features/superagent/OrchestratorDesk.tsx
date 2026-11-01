/**
 * 拥有：单独一页的编排台。总览、工单、线程、评估、执行、来源和设置。
 * 不拥有：工单怎么执行，也不占对话栏。那一段在 Go 的 superagent。
 */

import { useCallback, useEffect, useState } from "react";
import { getLocale, useCopy } from "../../lib/i18n";
import { orchestratorAct, orchestratorAdmission, orchestratorDecide, orchestratorDeliver, orchestratorMode, orchestratorPaused, orchestratorView } from "../../lib/client";
import "./desk.css";

type Source = { name: string; state: string; detail: string; repo?: string; chat?: string; app?: string; api?: string; hasSecret?: boolean };
type Input = { id: string; text: string; human: boolean; delivered: boolean };
type Follow = { thread: string; own: boolean; closed?: string };
type Assignment = { id: string; state: string; report?: string; summary?: string };
type Ticket = {
  id: string; stage: string; action: string; person: boolean; origin: string; outcome?: string;
  strength?: string; sender?: string; attention?: string; deliveryError?: string; originUrl?: string;
  inputs?: Input[]; follows?: Follow[]; assignments?: Assignment[];
};
type Thread = { key: string; queued: number; subs: number };
type Trigger = { id: string; source: string; sender: string; text: string; addressed: boolean };
type Decision = { id: string; triggerId: string; kind: string; reason: string; ticketId?: string; dismissed?: boolean; at?: string };
type Eval = { id: string; thread: string; strength: string; deadline?: string; delivered?: boolean; triggers: number };
type Execution = { ticketId: string; assignmentId: string; state: string; report?: string };
type Problem = { kind: string; id: string; title: string; detail?: string; triggerId?: string; decisionId?: string };
type Admission = { listedUnaddressed: string; unlistedAddressed: string; unlistedUnaddressed: string };
type View = {
  mode: string;
  paused: boolean;
  admission: Admission;
  sources: Source[];
  tickets: Ticket[];
  threads: Thread[];
  queued: Trigger[];
  evals: Eval[];
  decisions: Decision[];
  executions: Execution[];
  problems?: Problem[];
  allowed?: string[];
  workers?: number;
  health: string;
  at: string;
  evalTimeout?: number;
};

const tabs = ["overview", "tickets", "threads", "evals", "execution", "sources", "settings"] as const;
type Tab = (typeof tabs)[number];
const cells = ["explicit", "inferred", "ignore"] as const;
const decisionKinds = ["all", "accept", "ignore", "unavailable", "stop"] as const;

/** OrchestratorDesk 每两秒先领走评估再拉快照。这一页自己占满主栏，不叠在消息上。 */
export function OrchestratorDesk() {
  useCopy();
  const zh = getLocale() === "zh-CN";
  const t = (en: string, cn: string) => (zh ? cn : en);
  const [view, setView] = useState<View | null>(null);
  const [err, setErr] = useState("");
  const [tab, setTab] = useState<Tab>("overview");
  const [query, setQuery] = useState("");
  const [ticketID, setTicketID] = useState("");

  const load = useCallback(async () => {
    try {
      await orchestratorDeliver().catch(() => undefined);
      const next = await orchestratorView();
      setView(next);
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

  const act = async (kind: string, ticket = "", trigger = "", text = "", reason = "") => {
    try {
      await orchestratorAct(kind, ticket, trigger, text, reason);
      await load();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  };

  const mode = view?.mode || "observe";
  const queued = view?.queued?.length || 0;
  const people = view?.tickets?.filter((row) => row.person).length || 0;
  const needle = query.trim().toLowerCase();
  const keep = (...parts: Array<string | undefined>) => !needle || parts.join(" ").toLowerCase().includes(needle);
  const openTicket = (id: string) => {
    setTicketID(id);
    setTab("tickets");
  };

  return (
    <section className="orch orch-page" data-testid="orchestrator-desk">
      <header className="orch-bar">
        <span className="orch-title">{t("Orchestrator", "编排")}</span>
        <span className={`orch-dot orch-${view?.health === "ready" || !view ? "ok" : "err"}`} />
        <span className="orch-meta">{mode}{view?.paused ? ` · ${t("paused", "已暂停")}` : ""}</span>
        <span className="orch-meta">{t(`${queued} queued`, `${queued} 条待评估`)}</span>
        <span className="orch-meta">{t(`${people} need a person`, `${people} 条等人`)}</span>
      </header>
      {err ? <p className="orch-err">{err}</p> : null}
      <div className="orch-body">
        <div className="orch-tabs" role="tablist">
          {tabs.map((id) => (
            <button key={id} type="button" role="tab" aria-selected={tab === id} className={tab === id ? "on" : ""} onClick={() => setTab(id)}>
              {label(id, zh)}
            </button>
          ))}
        </div>
        <input className="orch-search" value={query} placeholder={t("Search tickets, threads, triggers, decisions", "搜索工单、线程、触发和决定")} onChange={(e) => setQuery(e.target.value)} />
        {!view ? <p className="orch-hint">{t("Loading", "正在读取")}</p> : null}
        {view && needle ? <SearchHits view={view} zh={zh} needle={needle} onTicket={openTicket} /> : null}
        {view && tab === "overview" ? <Overview view={view} zh={zh} onMode={(next) => { void orchestratorMode(next).then(load); }} onPaused={(paused) => { void orchestratorPaused(paused).then(load); }} onTicket={openTicket} onAct={act} /> : null}
        {view && tab === "tickets" ? <Tickets view={view} zh={zh} needle={keep} selected={ticketID} onSelect={setTicketID} onAct={act} /> : null}
        {view && tab === "threads" ? <Threads view={view} zh={zh} needle={keep} onAct={act} /> : null}
        {view && tab === "evals" ? <Evals view={view} zh={zh} needle={keep} onDecide={(id, kind) => { void orchestratorDecide(id, kind, "").then(load); }} onAct={act} /> : null}
        {view && tab === "execution" ? <Rows empty={t("No executions", "没有执行")} rows={(view.executions ?? []).filter((row) => keep(row.ticketId, row.state, row.report)).map((row) => ({
          id: row.assignmentId || row.ticketId,
          title: row.ticketId,
          meta: [row.state, row.report].filter(Boolean).join(" · "),
        }))} /> : null}
        {view && tab === "sources" ? <Sources view={view} zh={zh} /> : null}
        {view && tab === "settings" ? <Settings view={view} zh={zh} onAct={act} onAdmission={(adm) => { void orchestratorAdmission(adm.listedUnaddressed, adm.unlistedAddressed, adm.unlistedUnaddressed).then(load); }} /> : null}
      </div>
    </section>
  );
}

function label(id: Tab, zh: boolean) {
  const text: Record<Tab, [string, string]> = {
    overview: ["Overview", "总览"],
    tickets: ["Tickets", "工单"],
    threads: ["Threads", "线程"],
    evals: ["Evals", "评估"],
    execution: ["Execution", "执行"],
    sources: ["Sources", "来源"],
    settings: ["Settings", "设置"],
  };
  return zh ? text[id][1] : text[id][0];
}

function SearchHits(props: { view: View; zh: boolean; needle: string; onTicket: (id: string) => void }) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  const hit = (...parts: Array<string | undefined>) => parts.join(" ").toLowerCase().includes(props.needle);
  const tickets = (props.view.tickets ?? []).filter((row) => hit(row.id, row.stage, row.origin, row.sender, row.attention));
  const threads = (props.view.threads ?? []).filter((row) => hit(row.key));
  const triggers = (props.view.queued ?? []).filter((row) => hit(row.id, row.sender, row.source, row.text));
  const decisions = (props.view.decisions ?? []).filter((row) => hit(row.kind, row.reason, row.triggerId, row.ticketId));
  return (
    <div className="orch-block">
      <p className="orch-hint">{t("Matches", "匹配")}</p>
      <ul className="orch-list">
        {tickets.map((row) => <li key={`t-${row.id}`}><button type="button" className="as-text" onClick={() => props.onTicket(row.id)}>{row.id}</button><span>{row.stage}</span></li>)}
        {threads.map((row) => <li key={`h-${row.key}`}><strong>{row.key}</strong><span>{t("thread", "线程")}</span></li>)}
        {triggers.map((row) => <li key={`q-${row.id}`}><strong>{row.id}</strong><span>{row.text}</span></li>)}
        {decisions.map((row) => <li key={`d-${row.id}`}><strong>{row.kind}</strong><span>{row.triggerId}</span></li>)}
      </ul>
      {!tickets.length && !threads.length && !triggers.length && !decisions.length ? <p className="orch-hint">{t("No matches", "没有匹配")}</p> : null}
    </div>
  );
}

function Overview(props: {
  view: View; zh: boolean;
  onMode: (mode: string) => void;
  onPaused: (paused: boolean) => void;
  onTicket: (id: string) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
}) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  const waiting = (props.view.tickets ?? []).filter((row) => row.person);
  const problems = props.view.problems ?? [];
  return (
    <div className="orch-overview">
      <div className="orch-modes">
        {["off", "observe", "auto"].map((mode) => (
          <button key={mode} type="button" className={props.view.mode === mode ? "on" : ""} onClick={() => props.onMode(mode)}>{mode}</button>
        ))}
        <button type="button" className={props.view.paused ? "on" : ""} onClick={() => props.onPaused(!props.view.paused)}>
          {t("Pause", "暂停")}
        </button>
      </div>
      <p className="orch-hint">{props.view.health}{props.view.at ? ` · ${props.view.at}` : ""}</p>
      {!waiting.length ? <p className="orch-hint">{t("Nobody is waiting", "没有等人处理的工单")}</p> : (
        <ul className="orch-list">
          {waiting.map((row) => (
            <li key={row.id}>
              <button type="button" className="as-text" onClick={() => props.onTicket(row.id)}>{row.id}</button>
              <span>{[row.action, row.attention, row.deliveryError].filter(Boolean).join(" · ")}</span>
            </li>
          ))}
        </ul>
      )}
      {!problems.length ? <p className="orch-hint">{t("No problems", "没有故障")}</p> : (
        <ul className="orch-list">
          {problems.map((row) => (
            <li key={`${row.kind}-${row.id}`}>
              <strong>{row.title}</strong>
              <span>{[row.kind, row.detail].filter(Boolean).join(" · ")}</span>
              {row.kind === "unavailable" ? (
                <span className="orch-actions">
                  <button type="button" onClick={() => { void props.onAct("reevaluate", "", row.triggerId || ""); }}>{t("Requeue", "重新排队")}</button>
                  <button type="button" onClick={() => { void props.onAct("dismiss", "", row.decisionId || row.id); }}>{t("Dismiss", "驳回")}</button>
                </span>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function Rows(props: { empty: string; rows: { id: string; title: string; meta: string }[] }) {
  if (!props.rows.length) return <p className="orch-hint">{props.empty}</p>;
  return (
    <ul className="orch-list">
      {props.rows.map((row) => (
        <li key={row.id}>
          <strong>{row.title}</strong>
          <span>{row.meta}</span>
        </li>
      ))}
    </ul>
  );
}

function Threads(props: { view: View; zh: boolean; needle: (...parts: Array<string | undefined>) => boolean; onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void> }) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  const rows = (props.view.threads ?? []).filter((row) => props.needle(row.key));
  if (!rows.length) return <p className="orch-hint">{t("No threads", "没有线程")}</p>;
  return (
    <ul className="orch-list">
      {rows.map((row) => (
        <li key={row.key}>
          <strong>{row.key}</strong>
          <span>{t(`${row.queued} queued · ${row.subs} tickets`, `${row.queued} 条待评估 · ${row.subs} 张工单`)}</span>
          <button type="button" onClick={() => { void props.onAct("unsubscribe", "", "", row.key); }}>{t("Unsubscribe", "退订")}</button>
        </li>
      ))}
    </ul>
  );
}

/** Tickets 列出工单。点开后能看授权、输入、跟随和派工，并回复、交还、挂上或停掉。 */
function Tickets(props: {
  view: View;
  zh: boolean;
  needle: (...parts: Array<string | undefined>) => boolean;
  selected: string;
  onSelect: (id: string) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
}) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  const [note, setNote] = useState("");
  const [reply, setReply] = useState("");
  const [purpose, setPurpose] = useState("answer");
  const [trigger, setTrigger] = useState("");
  const [thread, setThread] = useState("");
  const [confirm, setConfirm] = useState("");
  const rows = (props.view.tickets ?? []).filter((row) => props.needle(row.id, row.stage, row.origin, row.outcome, row.sender, row.attention));
  const current = (props.view.tickets ?? []).find((row) => row.id === props.selected) ?? null;
  return (
    <div>
      {!rows.length ? <p className="orch-hint">{t("No tickets", "没有工单")}</p> : (
        <ul className="orch-list">
          {rows.map((row) => (
            <li key={row.id}>
              <button type="button" className="as-text" onClick={() => props.onSelect(row.id === props.selected ? "" : row.id)}>{row.id}</button>
              <span>{[row.stage, row.action, row.outcome].filter(Boolean).join(" · ")}</span>
            </li>
          ))}
        </ul>
      )}
      {current ? (
        <div className="orch-detail">
          <p className="orch-hint">
            {[current.strength, current.sender].filter(Boolean).join(" · ") || current.id}
            {current.originUrl ? <> · <a href={current.originUrl} target="_blank" rel="noreferrer">{t("source", "来源")}</a></> : current.origin ? ` · ${current.origin}` : ""}
          </p>
          {current.attention ? <p className="orch-hint">{current.attention}</p> : null}
          {current.deliveryError ? <p className="orch-err">{current.deliveryError}</p> : null}
          {(current.inputs ?? []).map((input) => (
            <p key={input.id} className="orch-hint">{input.delivered ? "" : t("Not delivered · ", "还没交给执行器 · ")}{input.text}</p>
          ))}
          {(current.follows ?? []).map((follow) => (
            <p key={follow.thread} className="orch-hint">{follow.thread}{follow.closed ? ` · ${follow.closed}` : ""}</p>
          ))}
          {(current.assignments ?? []).map((asg) => (
            <p key={asg.id} className="orch-hint">{[asg.state, asg.report, asg.summary].filter(Boolean).join(" · ")}</p>
          ))}
          <span className="orch-actions">
            <button type="button" onClick={() => setConfirm(confirm === "stop" ? "" : "stop")}>{t("Stop", "停止")}</button>
            <button type="button" onClick={() => setConfirm(confirm === "retry" ? "" : "retry")}>{t("Retry", "重试")}</button>
            <button type="button" onClick={() => { void props.onAct("requestStatus", current.id); }}>{t("Ask status", "问状态")}</button>
          </span>
          {confirm === "stop" ? <button type="button" onClick={() => { setConfirm(""); void props.onAct("stop", current.id, "", "", "stopped"); }}>{t("Confirm stop", "确认停止")}</button> : null}
          {confirm === "retry" ? <button type="button" onClick={() => { setConfirm(""); void props.onAct("retry", current.id); }}>{t("Confirm retry", "确认重试")}</button> : null}
          <textarea rows={2} value={note} placeholder={t("Note", "备注")} onChange={(e) => setNote(e.target.value)} />
          <span className="orch-actions">
            <button type="button" onClick={() => { void props.onAct("note", current.id, "", note).then(() => setNote("")); }}>{t("Save note", "记下")}</button>
          </span>
          <span className="orch-actions">
            <select value={purpose} onChange={(e) => setPurpose(e.target.value)}>
              <option value="acknowledgement">{t("Ack", "确认收到")}</option>
              <option value="answer">{t("Answer", "回答")}</option>
              <option value="result">{t("Result", "结果")}</option>
              <option value="attention">{t("Attention", "请人看")}</option>
            </select>
            <input value={reply} placeholder={t("Reply", "回复")} onChange={(e) => setReply(e.target.value)} />
            <button type="button" onClick={() => { void props.onAct("reply", current.id, "", reply, purpose).then(() => setReply("")); }}>{t("Send reply", "发出")}</button>
          </span>
          <span className="orch-actions">
            <input value={trigger} placeholder={t("Trigger id", "触发编号")} onChange={(e) => setTrigger(e.target.value)} />
            <button type="button" onClick={() => { void props.onAct("attach", current.id, trigger, note); }}>{t("Attach", "挂上")}</button>
            <button type="button" onClick={() => { void props.onAct("handback", current.id, trigger, "", note); }}>{t("Hand back", "交还")}</button>
          </span>
          <span className="orch-actions">
            <input value={thread} placeholder={t("Thread key", "线程")} onChange={(e) => setThread(e.target.value)} />
            <button type="button" onClick={() => { void props.onAct("follow", current.id, "", thread); }}>{t("Follow", "跟随")}</button>
            <button type="button" onClick={() => { void props.onAct("markUnrelated", current.id, "", thread); }}>{t("Unrelated", "无关")}</button>
          </span>
        </div>
      ) : null}
    </div>
  );
}

/** Evals 列出进行中的评估、队列和决定。决定可以按种类和最近一天过滤。 */
function Evals(props: {
  view: View;
  zh: boolean;
  needle: (...parts: Array<string | undefined>) => boolean;
  onDecide: (id: string, kind: string) => void;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
}) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  const [kind, setKind] = useState<(typeof decisionKinds)[number]>("all");
  const [day, setDay] = useState(false);
  const open = (props.view.evals ?? []).filter((row) => props.needle(row.id, row.thread, row.strength));
  const queued = (props.view.queued ?? []).filter((row) => props.needle(row.id, row.sender, row.source, row.text));
  const cutoff = Date.now() - 24 * 60 * 60 * 1000;
  const decisions = (props.view.decisions ?? []).filter((row) => {
    if (row.dismissed || !props.needle(row.kind, row.reason, row.triggerId)) return false;
    if (kind !== "all" && row.kind !== kind) return false;
    if (!day) return true;
    const at = row.at ? Date.parse(row.at) : 0;
    return at >= cutoff;
  });
  return (
    <div>
      <div className="orch-pills">
        {decisionKinds.map((id) => (
          <button key={id} type="button" className={kind === id ? "on" : ""} onClick={() => setKind(id)}>{id}</button>
        ))}
        <button type="button" className={day ? "on" : ""} onClick={() => setDay((on) => !on)}>{t("24h", "最近一天")}</button>
      </div>
      {open.length ? (
        <ul className="orch-list">
          {open.map((row) => (
            <li key={row.id}>
              <strong>{row.thread}</strong>
              <span>{row.delivered ? `${row.strength} · ${row.triggers} · ${row.deadline}` : t("Not with the assessor yet", "未交给评估者")}</span>
            </li>
          ))}
        </ul>
      ) : null}
      {!queued.length ? <p className="orch-hint">{t("Nothing queued", "没有待评估的触发")}</p> : (
        <ul className="orch-list">
          {queued.map((row) => (
            <li key={row.id}>
              <div>
                <strong>{row.sender || row.source}</strong>
                <span>{row.text}</span>
              </div>
              <span className="orch-actions">
                <button type="button" onClick={() => props.onDecide(row.id, "accept")}>{t("Accept", "接受")}</button>
                <button type="button" onClick={() => props.onDecide(row.id, "ignore")}>{t("Ignore", "忽略")}</button>
                <button type="button" onClick={() => { void props.onAct("reevaluate", "", row.id); }}>{t("Reevaluate", "重评")}</button>
              </span>
            </li>
          ))}
        </ul>
      )}
      {decisions.length ? (
        <ul className="orch-list orch-decisions">
          {decisions.slice(-12).map((row) => (
            <li key={row.id}>
              <span>{row.kind}{row.reason ? ` · ${row.reason}` : ""} · {row.triggerId}</span>
              {row.kind === "unavailable" ? <button type="button" onClick={() => { void props.onAct("dismiss", "", row.id); }}>{t("Dismiss", "驳回")}</button> : null}
              <button type="button" onClick={() => { void props.onAct("reevaluate", "", row.triggerId); }}>{t("Reevaluate", "重评")}</button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function Sources(props: { view: View; zh: boolean }) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  return (
    <ul className="orch-list">
      {(props.view.sources ?? []).map((src) => (
        <li key={src.name}>
          <strong>{src.name}</strong>
          <span className={`orch-state orch-${src.state}`}>{src.state === "unconfigured" ? t("not configured", "未配置") : src.state}</span>
          <span>{[src.repo, src.chat, src.app, src.hasSecret ? t("credential saved", "已保存凭证") : "", src.detail].filter(Boolean).join(" · ")}</span>
        </li>
      ))}
    </ul>
  );
}

/** Settings 改时限、准入、允许的发送者、来源凭证和工人槽。提示词仍在 harness 实验室里改。 */
function Settings(props: {
  view: View;
  zh: boolean;
  onAct: (kind: string, ticket?: string, trigger?: string, text?: string, reason?: string) => Promise<void>;
  onAdmission: (adm: Admission) => void;
}) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  const github = (props.view.sources ?? []).find((src) => src.name === "github");
  const feishu = (props.view.sources ?? []).find((src) => src.name === "feishu");
  const adm = props.view.admission ?? { listedUnaddressed: "inferred", unlistedAddressed: "inferred", unlistedUnaddressed: "ignore" };
  const [seconds, setSeconds] = useState(String(props.view.evalTimeout || 120));
  const [allowed, setAllowed] = useState((props.view.allowed ?? []).join(", "));
  const [workers, setWorkers] = useState(String(props.view.workers || 3));
  const [repo, setRepo] = useState(github?.repo || "");
  const [token, setToken] = useState("");
  const [clearToken, setClearToken] = useState(false);
  const [appId, setAppId] = useState(feishu?.app || "");
  const [chat, setChat] = useState(feishu?.chat || "");
  const [secret, setSecret] = useState("");
  const [clearSecret, setClearSecret] = useState(false);
  const setCell = (key: keyof Admission, value: string) => props.onAdmission({ ...adm, [key]: value });
  return (
    <div className="orch-detail">
      <p className="orch-hint">{t("Prompts stay in the harness lab.", "提示词在 harness 实验室里改。")}</p>
      <label className="orch-cell">
        <span>{t("Eval timeout (seconds)", "评估时限（秒）")}</span>
        <input value={seconds} onChange={(e) => setSeconds(e.target.value)} />
      </label>
      <span className="orch-actions">
        <button type="button" onClick={() => { void props.onAct("timeout", "", "", String(Number(seconds) || 120)); }}>{t("Save timeout", "保存时限")}</button>
      </span>
      <div className="orch-cells">
        <Cell label={t("Listed, not addressed", "在名单里，没被点名")} value={adm.listedUnaddressed} onChange={(value) => setCell("listedUnaddressed", value)} />
        <Cell label={t("Addressed, not listed", "被点名，不在名单里")} value={adm.unlistedAddressed} onChange={(value) => setCell("unlistedAddressed", value)} />
        <Cell label={t("Neither", "两者都不是")} value={adm.unlistedUnaddressed} onChange={(value) => setCell("unlistedUnaddressed", value)} />
      </div>
      <label className="orch-cell">
        <span>{t("Allowed senders", "允许的发送者")}</span>
        <input value={allowed} onChange={(e) => setAllowed(e.target.value)} />
      </label>
      <span className="orch-actions">
        <button type="button" onClick={() => { void props.onAct("allowed", "", "", allowed); }}>{t("Save senders", "保存名单")}</button>
      </span>
      <label className="orch-cell">
        <span>{t("Worker slots", "工人槽")}</span>
        <input value={workers} onChange={(e) => setWorkers(e.target.value)} />
      </label>
      <span className="orch-actions">
        <button type="button" onClick={() => { void props.onAct("workers", "", "", String(Number(workers) || 3)); }}>{t("Save slots", "保存槽位")}</button>
      </span>
      <label className="orch-cell"><span>GitHub repo</span><input value={repo} onChange={(e) => setRepo(e.target.value)} /></label>
      <label className="orch-cell">
        <span>{github?.hasSecret ? t("Token saved, leave blank to keep", "令牌已保存，留空则保持") : "GitHub token"}</span>
        <input type="password" value={token} onChange={(e) => setToken(e.target.value)} />
      </label>
      <label className="orch-cell"><span>{t("Clear token", "清除令牌")}</span><input type="checkbox" checked={clearToken} onChange={(e) => setClearToken(e.target.checked)} /></label>
      <label className="orch-cell"><span>{t("Feishu app", "飞书应用")}</span><input value={appId} onChange={(e) => setAppId(e.target.value)} /></label>
      <label className="orch-cell"><span>{t("Feishu chat", "飞书会话")}</span><input value={chat} onChange={(e) => setChat(e.target.value)} /></label>
      <label className="orch-cell">
        <span>{feishu?.hasSecret ? t("Secret saved, leave blank to keep", "密钥已保存，留空则保持") : t("Feishu secret", "飞书密钥")}</span>
        <input type="password" value={secret} onChange={(e) => setSecret(e.target.value)} />
      </label>
      <label className="orch-cell"><span>{t("Clear secret", "清除密钥")}</span><input type="checkbox" checked={clearSecret} onChange={(e) => setClearSecret(e.target.checked)} /></label>
      <span className="orch-actions">
        <button type="button" onClick={() => {
          void props.onAct("sources", "", "", JSON.stringify({
            githubRepo: repo,
            githubToken: token,
            clearGithubToken: clearToken,
            feishuAppId: appId,
            feishuChat: chat,
            feishuSecret: secret,
            clearFeishuSecret: clearSecret,
          }));
          setToken("");
          setSecret("");
          setClearToken(false);
          setClearSecret(false);
        }}>{t("Save sources", "保存来源")}</button>
      </span>
    </div>
  );
}

function Cell(props: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="orch-cell">
      <span>{props.label}</span>
      <select value={props.value || "inferred"} onChange={(e) => props.onChange(e.target.value)}>
        {cells.map((cell) => <option key={cell} value={cell}>{cell}</option>)}
      </select>
    </label>
  );
}
