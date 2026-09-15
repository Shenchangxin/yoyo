import { useEffect, useMemo, useRef, useState } from "react";
import * as api from "./lib/client";

const views = [
  ["agent", "Agent"],
  ["trajectory", "Trajectory"],
  ["harness", "Harness"],
  ["plugins", "Plugins"],
  ["eval", "Eval Lab"],
  ["evolve", "Evolution Lab"],
  ["settings", "Settings"],
] as const;

type ChatItem = { kind: "user" | "assistant" | "tool"; text: string; name?: string };

function eventsToChat(events: any[]): ChatItem[] {
  const out: ChatItem[] = [];
  for (const ev of events) {
    const typ = ev.type || ev.Type;
    const payload = ev.payload || ev.Payload || {};
    if (typ === "user" && payload.text) out.push({ kind: "user", text: payload.text });
    if (typ === "assistant" && payload.text) {
      if (payload.delta) {
        const last = out[out.length - 1];
        if (last?.kind === "assistant") last.text += payload.text;
        else out.push({ kind: "assistant", text: payload.text });
      } else {
        const last = out[out.length - 1];
        if (last?.kind === "assistant") last.text = payload.text;
        else out.push({ kind: "assistant", text: payload.text });
      }
    }
    if (typ === "error" && (payload.error || payload.text)) {
      out.push({ kind: "assistant", name: "error", text: String(payload.error || payload.text) });
    }
    if (typ === "tool_call") {
      out.push({ kind: "tool", name: payload.name, text: payload.arguments || "" });
    }
    if (typ === "tool_result") {
      out.push({ kind: "tool", name: (payload.name || "tool") + " result", text: String(payload.content || "").slice(0, 2000) });
    }
    if (typ === "context_injection" && payload.text) {
      out.push({ kind: "tool", name: "mention", text: String(payload.text).slice(0, 2000) });
    }
  }
  return out;
}

function passClass(ok: boolean) {
  return ok ? "ok-chip" : "bad-chip";
}

export default function App() {
  const [view, setView] = useState<(typeof views)[number][0]>("agent");
  const [health, setHealth] = useState<any>({});
  const [session, setSession] = useState<any>(null);
  const [sessions, setSessions] = useState<any[]>([]);
  const [message, setMessage] = useState("Write hello.txt containing hello");
  const [events, setEvents] = useState<any[]>([]);
  const [pending, setPending] = useState<any[]>([]);
  const [harness, setHarness] = useState<any>({});
  const [plugins, setPlugins] = useState<any>({});
  const [evalReport, setEvalReport] = useState<any>(null);
  const [bestReport, setBestReport] = useState<any>(null);
  const [evolve, setEvolve] = useState<any>(null);
  const [tree, setTree] = useState<any[]>([]);
  const [cfg, setCfg] = useState<any>({});
  const [busy, setBusy] = useState(false);
  const [plan, setPlan] = useState(false);
  const [ctx, setCtx] = useState<any>({});
  const [err, setErr] = useState("");
  const [diffA, setDiffA] = useState("");
  const [diffB, setDiffB] = useState("");
  const [diffOut, setDiffOut] = useState<any>(null);
  const [evolveK, setEvolveK] = useState(3);
  const [sessQ, setSessQ] = useState("");
  const [playbook, setPlaybook] = useState<any>(null);
  const [gitDiff, setGitDiff] = useState("");
  const [hunks, setHunks] = useState<any[]>([]);
  const [hunkSel, setHunkSel] = useState<Record<string, boolean>>({});
  const [bonModels, setBonModels] = useState("");
  const logRef = useRef<HTMLDivElement>(null);

  async function refresh() {
    try {
      const h = await api.health();
      setHealth(h);
      setHarness(await api.harness());
      setPlugins(await api.plugins());
      setTree(await api.archive());
      setCfg(await api.getConfig());
      try { setPlaybook(await api.playbook()); } catch { /* optional */ }
      const list = await api.listSessions();
      setSessions(list || []);
      if (list?.length && !session) setSession(list[0]);
    } catch (e: any) {
      setErr(String(e.message || e));
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  const sessionId = session?.id || session?.ID;
  useEffect(() => {
    if (!sessionId) return;
    api.trajectory(sessionId).then(setEvents).catch(() => {});
  }, [sessionId]);

  useEffect(() => {
    if (!busy || !sessionId) return;
    const t = setInterval(async () => {
      try {
        setEvents(await api.trajectory(sessionId));
        setPending(await api.approvals());
      } catch {
        /* ignore poll errors */
      }
    }, 400);
    return () => clearInterval(t);
  }, [busy, sessionId]);

  useEffect(() => {
    logRef.current?.scrollTo(0, logRef.current.scrollHeight);
  }, [events]);

  const title = useMemo(() => views.find((v) => v[0] === view)?.[1], [view]);
  const chat = useMemo(() => eventsToChat(events), [events]);
  const usage = health.usage || {};

  async function ensureSession() {
    if (session?.id || session?.ID) return session;
    const s = await api.createSession(cfg.workspace || "");
    setSession(s);
    setSessions((prev) => [s, ...prev]);
    return s;
  }

  async function onSend() {
    setBusy(true);
    setErr("");
    try {
      const s = await ensureSession();
      const id = s.id || s.ID;
      await api.send(id, message, { async: true, plan });
      const started = Date.now();
      await new Promise((r) => setTimeout(r, 80));
      while (Date.now() - started < 10 * 60 * 1000) {
        const [evs, offers, live, usageCtx] = await Promise.all([
          api.trajectory(id),
          api.approvals(),
          api.running(id),
          api.contextUsage(id).catch(() => ({})),
        ]);
        setEvents(evs);
        setPending(offers);
        setCtx(usageCtx || {});
        if (!live && offers.length === 0) break;
        await new Promise((r) => setTimeout(r, 400));
      }
      setEvents(await api.trajectory(id));
      await refresh();
    } catch (e: any) {
      setErr(String(e.message || e));
    } finally {
      setBusy(false);
    }
  }

  async function onStop() {
    if (!sessionId) return;
    await api.interrupt(sessionId);
    setBusy(false);
  }

  const metrics = evalReport?.metrics || evalReport?.Metrics;
  const results = evalReport?.results || evalReport?.Results || [];
  const trials = evolve?.tried || evolve?.Tried || [];
  const proposals = evolve?.proposals || evolve?.Proposals || [];

  return (
    <div className="app">
      <aside className="side">
        <div className="brand">Yoyo</div>
        {views.map(([id, label]) => (
          <button key={id} className={view === id ? "nav active" : "nav"} onClick={() => setView(id)}>
            {label}
          </button>
        ))}
        <div className="side-sessions">
          <input placeholder="search sessions" value={sessQ} onChange={(e) => setSessQ(e.target.value)} />
          {sessions.filter((s) => {
            const q = sessQ.toLowerCase();
            if (!q) return true;
            return String(s.title || s.Title || s.id || s.ID).toLowerCase().includes(q);
          }).slice(0, 12).map((s) => {
            const id = s.id || s.ID;
            return (
              <button key={id} className={sessionId === id ? "nav active" : "nav"} onClick={() => setSession(s)}>
                {(s.title || s.Title || id).slice(0, 18)}
              </button>
            );
          })}
        </div>
      </aside>
      <section className="main">
        <header className="top">
          <span>{title}</span>
          <span>
            {health.model} · v{health.version || "?"} · <span className="hash">{(health.harness || "").slice(0, 12)}</span>
            {ctx.tokens ? (
              <span className="ctx">
                {" "}
                ctx {ctx.tokens}/{ctx.budget || "?"}
                {ctx.note ? ` · ${ctx.note}` : ""}
              </span>
            ) : null}
            {usage.usd ? <span className="ctx"> · ${Number(usage.usd).toFixed(4)}</span> : null}
          </span>
        </header>
        <div className="panel">
          {err ? <div className="bad">{err}</div> : null}

          {view === "agent" && (
            <>
              {pending.map((p) => (
                <div className="approve" key={p.id || p.ID}>
                  <div>
                    Approve {p.request?.action || p.Request?.Action}{" "}
                    <span className="mono-inline">{p.request?.command || p.Request?.Command || p.request?.path || ""}</span>
                  </div>
                  <div className="row">
                    <button className="btn" onClick={() => api.resolveApproval(p.id || p.ID, "once").then(() => setPending([]))}>
                      Once
                    </button>
                    <button className="btn ghost" onClick={() => api.resolveApproval(p.id || p.ID, "session")}>
                      Session
                    </button>
                    <button className="btn ghost" onClick={() => api.resolveApproval(p.id || p.ID, "always").then(() => setPending([]))}>
                      Always
                    </button>
                    <button className="btn ghost" onClick={() => api.resolveApproval(p.id || p.ID, "deny")}>
                      Deny
                    </button>
                  </div>
                </div>
              ))}
              <div className="log chat" ref={logRef}>
                {chat.length === 0 ? "No turns yet. Self-harness materials load from refs/active." : null}
                {chat.map((item, i) => (
                  <div key={i} className={"bubble " + item.kind}>
                    <div className="who">{item.name || item.kind}</div>
                    <div>{item.text}</div>
                  </div>
                ))}
              </div>
              <textarea placeholder="Message. Pin context with @file:path @folder:dir @harness (bounded inject, not a repo dump)." value={message} onChange={(e) => setMessage(e.target.value)} onKeyDown={(e) => {
                if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) onSend();
              }} />
              {gitDiff ? <pre className="diff">{gitDiff}</pre> : null}
              {hunks.length ? (
                <div className="hunks">
                  {hunks.map((h) => {
                    const id = h.id || h.ID;
                    return (
                      <label key={id} className="hunk">
                        <input type="checkbox" checked={!!hunkSel[id]} onChange={(e) => setHunkSel({ ...hunkSel, [id]: e.target.checked })} />
                        <span className="mono-inline">{id}</span>
                        <pre>{(h.header || h.Header || "") + "\n" + (h.body || h.Body || "")}</pre>
                      </label>
                    );
                  })}
                  <button className="btn" onClick={async () => {
                    const ids = Object.entries(hunkSel).filter(([, v]) => v).map(([k]) => k);
                    try {
                      await api.applyHunks(cfg.workspace || "", ids);
                      const h = await api.workspaceHunks(cfg.workspace || "");
                      setGitDiff(h.diff || "");
                      setHunks(h.hunks || []);
                      setHunkSel({});
                    } catch (e: any) { setErr(String(e.message || e)); }
                  }}>Apply selected hunks</button>
                </div>
              ) : null}
              <div className="row">
                <button className="btn" disabled={busy} onClick={onSend}>
                  {busy ? "Running…" : "Send"}
                </button>
                <button className="btn ghost" disabled={!busy} onClick={onStop}>
                  Stop
                </button>
                <button className="btn ghost" onClick={() => ensureSession()}>
                  New session
                </button>
                <button className="btn ghost" disabled={!sessionId} onClick={async () => {
                  const title = window.prompt("Rename session", session?.title || session?.Title || "");
                  if (!title || !sessionId) return;
                  await api.renameSession(sessionId, title);
                  await refresh();
                }}>
                  Rename
                </button>
                <button className="btn ghost" disabled={!sessionId} onClick={async () => {
                  if (!sessionId) return;
                  const s = await api.forkSession(sessionId);
                  setSession(s);
                  await refresh();
                }}>
                  Fork
                </button>
                <button className="btn ghost" onClick={async () => {
                  try {
                    const h = await api.workspaceHunks(cfg.workspace || "");
                    setGitDiff(h.diff || "");
                    setHunks(h.hunks || []);
                    setHunkSel({});
                  } catch (e: any) { setErr(String(e.message || e)); }
                }}>
                  Git diff
                </button>
                <label className="plan">
                  <input type="checkbox" checked={plan} onChange={(e) => setPlan(e.target.checked)} /> Plan
                </label>
              </div>
            </>
          )}

          {view === "trajectory" && (
            <div className="log">
              {events.map((ev, i) => (
                <div key={i}>
                  [{ev.type || ev.Type}] {ev.source || ev.Source} {JSON.stringify(ev.payload || ev.Payload)}
                </div>
              ))}
              {events.length === 0 ? "No events. Run the agent first." : null}
            </div>
          )}

          {view === "harness" && (
            <>
              <div className="row">
                <button className="btn ghost" onClick={() => api.rollback().then(refresh)}>
                  Rollback
                </button>
              </div>
              <div className="cards">
                {Object.entries(harness.refs || {}).map(([k, v]) => (
                  <div className="card" key={k}>
                    <div>{k}</div>
                    <div className="hash">{String(v).slice(0, 16)}</div>
                    <button className="btn ghost" onClick={() => api.checkoutSafe(String(v)).then(refresh).catch((e) => setErr(String(e.message || e)))}>
                      Checkout
                    </button>
                    <button className="btn ghost" onClick={() => setDiffB(String(v))}>
                      Diff as B
                    </button>
                  </div>
                ))}
              </div>
              <h3>Diff</h3>
              <div className="row">
                <input placeholder="hash A" value={diffA} onChange={(e) => setDiffA(e.target.value)} />
                <input placeholder="hash B" value={diffB} onChange={(e) => setDiffB(e.target.value)} />
                <button className="btn" onClick={async () => {
                  try {
                    setDiffOut(await api.diff(diffA || harness.active, diffB));
                  } catch (e: any) {
                    setErr(String(e.message || e));
                  }
                }}>
                  Compare
                </button>
              </div>
              {diffOut?.fields ? (
                <table className="grid">
                  <thead>
                    <tr><th>Field</th><th>From</th><th>To</th></tr>
                  </thead>
                  <tbody>
                    {diffOut.fields.map((f: any) => (
                      <tr key={f.field || f.Field} className={f.changed || f.Changed ? "changed" : ""}>
                        <td>{f.field || f.Field}</td>
                        <td className="hash">{String(f.from || f.From || "").slice(0, 24)}</td>
                        <td className="hash">{String(f.to || f.To || "").slice(0, 24)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : null}
              {diffOut?.text ? <pre>{diffOut.text}</pre> : null}
              <h3>Active snapshot</h3>
              <pre>{JSON.stringify(harness.snapshot, null, 2)}</pre>
            </>
          )}

          {view === "plugins" && (
            <>
              <pre>{JSON.stringify(plugins, null, 2)}</pre>
              {(plugins.fibers || []).map((name: string) => (
                <button key={name} className="btn ghost" onClick={() => api.unloadFiber(name).then(refresh)}>
                  Unload {name}
                </button>
              ))}
            </>
          )}

          {view === "eval" && (
            <>
              <div className="row">
                <button className="btn" disabled={busy} onClick={async () => {
                  setBusy(true);
                  try {
                    setEvalReport(await api.runEval());
                    setBestReport(null);
                  } catch (e: any) {
                    setErr(String(e.message || e));
                  } finally {
                    setBusy(false);
                  }
                }}>
                  Run suite
                </button>
                <button className="btn ghost" disabled={busy} onClick={async () => {
                  setBusy(true);
                  try {
                    const r = await api.bestOfN(3);
                    setBestReport(r);
                    setEvalReport(r.best || r.Best);
                  } catch (e: any) {
                    setErr(String(e.message || e));
                  } finally {
                    setBusy(false);
                  }
                }}>
                  Best-of-3
                </button>
                <button className="btn ghost" disabled={busy} onClick={async () => {
                  setBusy(true);
                  try {
                    setEvalReport(await api.runEvalSafety());
                    setBestReport(null);
                  } catch (e: any) {
                    setErr(String(e.message || e));
                  } finally {
                    setBusy(false);
                  }
                }}>
                  Suite + safety
                </button>
                <button className="btn ghost" disabled={busy} onClick={async () => {
                  setBusy(true);
                  try {
                    setEvalReport(await api.runEvalTB());
                    setBestReport(null);
                  } catch (e: any) {
                    setErr(String(e.message || e));
                  } finally {
                    setBusy(false);
                  }
                }}>
                  TB subset
                </button>
                <input placeholder="models a,b" value={bonModels} onChange={(e) => setBonModels(e.target.value)} style={{ width: 160 }} />
                <button className="btn ghost" disabled={busy} onClick={async () => {
                  setBusy(true);
                  try {
                    const models = bonModels.split(",").map((s) => s.trim()).filter(Boolean);
                    const r = await api.bestOfModels(models);
                    setBestReport(r);
                    setEvalReport(r.best || r.Best);
                  } catch (e: any) {
                    setErr(String(e.message || e));
                  } finally {
                    setBusy(false);
                  }
                }}>
                  Models BoN
                </button>
              </div>
              {metrics ? (
                <div className="cards">
                  <div className="card">
                    <div>Held-in</div>
                    <div className="metric">{metrics.held_in_pass ?? metrics.HeldInPass}/{metrics.held_in_total ?? metrics.HeldInTotal}</div>
                  </div>
                  <div className="card">
                    <div>Held-out</div>
                    <div className="metric">{metrics.held_out_pass ?? metrics.HeldOutPass}/{metrics.held_out_total ?? metrics.HeldOutTotal}</div>
                  </div>
                  <div className="card">
                    <div>Safety fail</div>
                    <div className="metric">{metrics.safety_fail ?? metrics.SafetyFail ?? 0}</div>
                  </div>
                </div>
              ) : null}
              {results.length ? (
                <table className="grid">
                  <thead>
                    <tr><th>Task</th><th>Pass</th><th>Error</th></tr>
                  </thead>
                  <tbody>
                    {results.map((r: any) => (
                      <tr key={r.id || r.ID}>
                        <td>{r.id || r.ID}</td>
                        <td><span className={passClass(!!(r.pass ?? r.Pass))}>{String(r.pass ?? r.Pass)}</span></td>
                        <td>{r.error || r.Error || ""}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : <p className="muted">Run the sealed local suite (held-in write-hello / held-out write-answer). TB subset and write-readme are opt-in.</p>}
              {bestReport?.reports ? (
                <div>
                  <h3>Best-of-N runs</h3>
                  {(bestReport.reports || bestReport.Reports || []).map((r: any, i: number) => {
                    const m = r.metrics || r.Metrics || {};
                    return (
                      <div key={i} className="card">
                        run {i + 1}: in {m.held_in_pass ?? m.HeldInPass}/{m.held_in_total ?? m.HeldInTotal} · out {m.held_out_pass ?? m.HeldOutPass}/{m.held_out_total ?? m.HeldOutTotal}
                      </div>
                    );
                  })}
                </div>
              ) : null}
            </>
          )}

          {view === "evolve" && (
            <>
              <div className="row">
                <label className="plan">
                  K
                  <input type="number" min={1} max={8} value={evolveK} onChange={(e) => setEvolveK(Number(e.target.value) || 3)} style={{ width: 64 }} />
                </label>
                <button className="btn" disabled={busy} onClick={async () => {
                  setBusy(true);
                  try {
                    setEvolve(await api.evolve(evolveK));
                    setTree(await api.archive());
                    await refresh();
                  } catch (e: any) {
                    setErr(String(e.message || e));
                  } finally {
                    setBusy(false);
                  }
                }}>
                  Run Self-Harness cycle
                </button>
              </div>
              {evolve ? (
                <>
                  <div className="cards">
                    <div className="card">
                      <div>Promoted</div>
                      <div className="hash">{(evolve.promoted || evolve.Promoted || "none").toString().slice(0, 16)}</div>
                    </div>
                    <div className="card">
                      <div>Parent</div>
                      <div className="hash">{(evolve.parent_selected || evolve.ParentSelected || "").toString().slice(0, 16)}</div>
                    </div>
                    <div className="card">
                      <div>Proposals</div>
                      <div className="metric">{proposals.length}</div>
                    </div>
                  </div>
                  <h3>Trials</h3>
                  <table className="grid">
                    <thead>
                      <tr><th>Proposal</th><th>Accepted</th><th>Reason</th><th>Held-in</th><th>Held-out</th></tr>
                    </thead>
                    <tbody>
                      {trials.map((t: any, i: number) => {
                        const m = t.metrics || t.Metrics || {};
                        const p = t.proposal || t.Proposal || {};
                        return (
                          <tr key={p.id || p.ID || i}>
                            <td>{p.id || p.ID}</td>
                            <td><span className={passClass(!!(t.accepted ?? t.Accepted))}>{String(t.accepted ?? t.Accepted)}</span></td>
                            <td>{t.reason || t.Reason || ""}</td>
                            <td>{m.held_in_pass ?? m.HeldInPass}/{m.held_in_total ?? m.HeldInTotal}</td>
                            <td>{m.held_out_pass ?? m.HeldOutPass}/{m.held_out_total ?? m.HeldOutTotal}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </>
              ) : <p className="muted">No cycle yet. Harbor still owns promotion; K is proposal count.</p>}
              <h3>Playbook (thumbs write refs/staging only)</h3>
              <div className="cards">
                {(playbook?.bullets || playbook?.Bullets || []).map((b: any) => (
                  <div className="card" key={b.id || b.ID}>
                    <div>{b.text || b.Text}</div>
                    <div className="muted">+{b.helpful ?? b.Helpful} / −{b.harmful ?? b.Harmful}</div>
                    <div className="row">
                      <button className="btn ghost" onClick={async () => {
                        setPlaybook(await api.ratePlaybook(b.id || b.ID, true));
                        await refresh();
                      }}>+ helpful</button>
                      <button className="btn ghost" onClick={async () => {
                        setPlaybook(await api.ratePlaybook(b.id || b.ID, false));
                        await refresh();
                      }}>+ harmful</button>
                    </div>
                  </div>
                ))}
              </div>
              <h3>Archive</h3>
              <div className="cards">
                {tree.map((n: any) => (
                  <div className="card" key={n.id || n.ID}>
                    <div className="hash">{String(n.id || n.ID).slice(0, 16)}</div>
                    <div>{n.proposal_id || n.ProposalID || n.note || n.Note}</div>
                    <div>{(n.accepted ?? n.Accepted) ? "accepted" : "rejected"}</div>
                  </div>
                ))}
              </div>
            </>
          )}

          {view === "settings" && (
            <>
              <input placeholder="model" value={cfg.model || ""} onChange={(e) => setCfg({ ...cfg, model: e.target.value })} />
              <input placeholder="base_url" value={cfg.base_url || cfg.BaseURL || ""} onChange={(e) => setCfg({ ...cfg, base_url: e.target.value, BaseURL: e.target.value })} />
              <input placeholder="workspace" value={cfg.workspace || cfg.Workspace || ""} onChange={(e) => setCfg({ ...cfg, workspace: e.target.value, Workspace: e.target.value })} />
              <input placeholder="max_budget_usd (0 = unlimited)" value={cfg.max_budget_usd ?? cfg.MaxBudgetUSD ?? ""} onChange={(e) => setCfg({ ...cfg, max_budget_usd: Number(e.target.value), MaxBudgetUSD: Number(e.target.value) })} />
              <input placeholder="usd_per_mtok" value={cfg.usd_per_mtok ?? cfg.USDPerMTok ?? ""} onChange={(e) => setCfg({ ...cfg, usd_per_mtok: Number(e.target.value), USDPerMTok: Number(e.target.value) })} />
              <input placeholder="API key" type="password" onBlur={(e) => e.target.value && api.setAPIKey(e.target.value)} />
              <label className="plan">
                <input type="checkbox" checked={!!(cfg.auto_allow ?? cfg.AutoAllow)} onChange={(e) => setCfg({ ...cfg, auto_allow: e.target.checked, AutoAllow: e.target.checked })} />
                Auto-allow shell
              </label>
              <input placeholder="extra models (comma) for BoN" value={(cfg.models || cfg.Models || []).join?.(",") || bonModels} onChange={(e) => {
                const models = e.target.value.split(",").map((s) => s.trim()).filter(Boolean);
                setBonModels(e.target.value);
                setCfg({ ...cfg, models, Models: models });
              }} />
              <button className="btn" onClick={() => api.setConfig(cfg).then(refresh)}>
                Save
              </button>
              <button className="btn ghost" onClick={() => api.applyUpdate().then(refresh).catch((e) => setErr(String(e.message || e)))}>
                Apply staged update
              </button>
            </>
          )}
        </div>
      </section>
    </div>
  );
}
