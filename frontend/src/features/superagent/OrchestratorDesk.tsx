/**
 * 拥有：单独一页的编排台。总览、工单、线程、评估和来源。
 * 不拥有：工单怎么执行，也不占对话栏。那一段在 Go 的 superagent。
 */

import { useCallback, useEffect, useState } from "react";
import { getLocale, useCopy } from "../../lib/i18n";
import { orchestratorAdmission, orchestratorDecide, orchestratorMode, orchestratorPaused, orchestratorView } from "../../lib/client";
import "./desk.css";

type Source = { name: string; state: string; detail: string };
type Ticket = { id: string; stage: string; action: string; person: boolean; origin: string; outcome?: string };
type Thread = { key: string; queued: number; subs: number };
type Trigger = { id: string; source: string; sender: string; text: string; addressed: boolean };
type Decision = { id: string; triggerId: string; kind: string; reason: string; ticketId?: string };
type Admission = { listedUnaddressed: string; unlistedAddressed: string; unlistedUnaddressed: string };
type View = {
  mode: string;
  paused: boolean;
  admission: Admission;
  sources: Source[];
  tickets: Ticket[];
  threads: Thread[];
  queued: Trigger[];
  decisions: Decision[];
  health: string;
  at: string;
};

const tabs = ["overview", "tickets", "threads", "evals", "sources"] as const;
type Tab = (typeof tabs)[number];

const cells = ["explicit", "inferred", "ignore"] as const;

/** OrchestratorDesk 每两秒拉一次编排快照。这一页自己占满主栏，不叠在消息上。 */
export function OrchestratorDesk() {
  useCopy();
  const zh = getLocale() === "zh-CN";
  const t = (en: string, cn: string) => (zh ? cn : en);
  const [view, setView] = useState<View | null>(null);
  const [err, setErr] = useState("");
  const [tab, setTab] = useState<Tab>("overview");

  const load = useCallback(async () => {
    try {
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

  const mode = view?.mode || "observe";
  const queued = view?.queued?.length || 0;
  const people = view?.tickets?.filter((row) => row.person).length || 0;

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
      {view ? (
        <div className="orch-body">
          <div className="orch-tabs" role="tablist">
            {tabs.map((id) => (
              <button key={id} type="button" role="tab" aria-selected={tab === id} className={tab === id ? "on" : ""} onClick={() => setTab(id)}>
                {label(id, zh)}
              </button>
            ))}
          </div>
          {tab === "overview" ? <Overview view={view} zh={zh} onMode={(next) => { void orchestratorMode(next).then(load); }} onPaused={(paused) => { void orchestratorPaused(paused).then(load); }} onAdmission={(adm) => { void orchestratorAdmission(adm.listedUnaddressed, adm.unlistedAddressed, adm.unlistedUnaddressed).then(load); }} /> : null}
          {tab === "tickets" ? <Rows empty={t("No tickets", "没有工单")} rows={(view.tickets ?? []).map((row) => ({
            id: row.id,
            title: row.id,
            meta: [row.stage, row.action, row.outcome].filter(Boolean).join(" · "),
          }))} /> : null}
          {tab === "threads" ? <Rows empty={t("No threads", "没有线程")} rows={(view.threads ?? []).map((row) => ({
            id: row.key,
            title: row.key,
            meta: t(`${row.queued} queued · ${row.subs} tickets`, `${row.queued} 条待评估 · ${row.subs} 张工单`),
          }))} /> : null}
          {tab === "evals" ? <Evals view={view} zh={zh} onDecide={(id, kind) => { void orchestratorDecide(id, kind, "").then(load); }} /> : null}
          {tab === "sources" ? <Sources view={view} zh={zh} /> : null}
        </div>
      ) : null}
    </section>
  );
}

function label(id: Tab, zh: boolean) {
  const text: Record<Tab, [string, string]> = {
    overview: ["Overview", "总览"],
    tickets: ["Tickets", "工单"],
    threads: ["Threads", "线程"],
    evals: ["Evals", "评估"],
    sources: ["Sources", "来源"],
  };
  return zh ? text[id][1] : text[id][0];
}

function Overview(props: { view: View; zh: boolean; onMode: (mode: string) => void; onPaused: (paused: boolean) => void; onAdmission: (adm: Admission) => void }) {
  const zh = props.zh;
  const t = (en: string, cn: string) => (zh ? cn : en);
  const adm = props.view.admission ?? { listedUnaddressed: "inferred", unlistedAddressed: "inferred", unlistedUnaddressed: "ignore" };
  const setCell = (key: keyof Admission, value: string) => props.onAdmission({ ...adm, [key]: value });
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
      <div className="orch-cells">
        <Cell label={t("Listed, not addressed", "在名单里，没被点名")} value={adm.listedUnaddressed} onChange={(value) => setCell("listedUnaddressed", value)} />
        <Cell label={t("Addressed, not listed", "被点名，不在名单里")} value={adm.unlistedAddressed} onChange={(value) => setCell("unlistedAddressed", value)} />
        <Cell label={t("Neither", "两者都不是")} value={adm.unlistedUnaddressed} onChange={(value) => setCell("unlistedUnaddressed", value)} />
      </div>
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

function Evals(props: { view: View; zh: boolean; onDecide: (id: string, kind: string) => void }) {
  const t = (en: string, cn: string) => (props.zh ? cn : en);
  return (
    <div>
      {!props.view.queued?.length ? <p className="orch-hint">{t("Nothing queued", "没有待评估的触发")}</p> : (
        <ul className="orch-list">
          {(props.view.queued ?? []).map((row) => (
            <li key={row.id}>
              <div>
                <strong>{row.sender || row.source}</strong>
                <span>{row.text}</span>
              </div>
              <span className="orch-actions">
                <button type="button" onClick={() => props.onDecide(row.id, "accept")}>{t("Accept", "接受")}</button>
                <button type="button" onClick={() => props.onDecide(row.id, "ignore")}>{t("Ignore", "忽略")}</button>
              </span>
            </li>
          ))}
        </ul>
      )}
      {props.view.decisions?.length ? (
        <ul className="orch-list orch-decisions">
          {props.view.decisions.slice(-8).map((row) => (
            <li key={row.id}><span>{row.kind}{row.reason ? ` · ${row.reason}` : ""} · {row.triggerId}</span></li>
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
          <span>{src.detail}</span>
        </li>
      ))}
    </ul>
  );
}
