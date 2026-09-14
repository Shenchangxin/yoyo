import { useEffect, useMemo, useState } from "react";
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

export default function App() {
  const [view, setView] = useState<(typeof views)[number][0]>("agent");
  const [health, setHealth] = useState<any>({});
  const [session, setSession] = useState<any>(null);
  const [message, setMessage] = useState("Write hello.txt containing hello");
  const [output, setOutput] = useState("");
  const [events, setEvents] = useState<any[]>([]);
  const [harness, setHarness] = useState<any>({});
  const [plugins, setPlugins] = useState<any>({});
  const [evalReport, setEvalReport] = useState<any>(null);
  const [evolve, setEvolve] = useState<any>(null);
  const [tree, setTree] = useState<any[]>([]);
  const [cfg, setCfg] = useState<any>({});
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function refresh() {
    try {
      const h = await api.health();
      setHealth(h);
      setHarness(await api.harness());
      setPlugins(await api.plugins());
      setTree(await api.archive());
      setCfg(await api.getConfig());
      const sessions = await api.listSessions();
      if (sessions?.length && !session) setSession(sessions[0]);
    } catch (e: any) {
      setErr(String(e.message || e));
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  const title = useMemo(() => views.find((v) => v[0] === view)?.[1], [view]);

  async function ensureSession() {
    if (session?.id || session?.ID) return session;
    const s = await api.createSession(cfg.workspace || "");
    setSession(s);
    return s;
  }

  async function onSend() {
    setBusy(true);
    setErr("");
    try {
      const s = await ensureSession();
      const id = s.id || s.ID;
      const text = await api.send(id, message);
      setOutput(text);
      setEvents(await api.trajectory(id));
      await refresh();
    } catch (e: any) {
      setErr(String(e.message || e));
    } finally {
      setBusy(false);
    }
  }

  async function loadTrajectory() {
    const id = session?.id || session?.ID;
    if (!id) return;
    setEvents(await api.trajectory(id));
  }

  return (
    <div className="app">
      <aside className="side">
        <div className="brand">Yoyo</div>
        {views.map(([id, label]) => (
          <button key={id} className={view === id ? "nav active" : "nav"} onClick={() => setView(id)}>
            {label}
          </button>
        ))}
      </aside>
      <section className="main">
        <header className="top">
          <span>{title}</span>
          <span>
            {health.model} · <span className="hash">{(health.harness || "").slice(0, 12)}</span>
          </span>
        </header>
        <div className="panel">
          {err ? <div className="bad">{err}</div> : null}

          {view === "agent" && (
            <>
              <textarea value={message} onChange={(e) => setMessage(e.target.value)} />
              <div className="row">
                <button className="btn" disabled={busy} onClick={onSend}>
                  {busy ? "Running…" : "Send"}
                </button>
                <button className="btn ghost" onClick={() => ensureSession()}>
                  New session
                </button>
              </div>
              <div className="log">{output || "No assistant output yet."}</div>
            </>
          )}

          {view === "trajectory" && (
            <>
              <div className="row">
                <button className="btn" onClick={loadTrajectory}>
                  Reload
                </button>
              </div>
              <div className="log">
                {events.map((ev, i) => (
                  <div key={i}>
                    [{ev.type || ev.Type}] {ev.source || ev.Source} {JSON.stringify(ev.payload || ev.Payload)}
                  </div>
                ))}
                {events.length === 0 ? "No events. Run the agent first." : null}
              </div>
            </>
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
                    <button className="btn ghost" onClick={() => api.checkout(String(v)).then(refresh)}>
                      Checkout
                    </button>
                  </div>
                ))}
              </div>
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
              <button className="btn" disabled={busy} onClick={async () => {
                setBusy(true);
                try {
                  setEvalReport(await api.runEval());
                } catch (e: any) {
                  setErr(String(e.message || e));
                } finally {
                  setBusy(false);
                }
              }}>
                Run suite
              </button>
              <pre>{evalReport ? JSON.stringify(evalReport, null, 2) : "Run the sealed local suite (held-in / held-out)."}</pre>
            </>
          )}

          {view === "evolve" && (
            <>
              <button className="btn" disabled={busy} onClick={async () => {
                setBusy(true);
                try {
                  setEvolve(await api.evolve());
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
              <pre>{evolve ? JSON.stringify(evolve, null, 2) : "No cycle yet."}</pre>
              <h3>Archive</h3>
              <pre>{JSON.stringify(tree, null, 2)}</pre>
            </>
          )}

          {view === "settings" && (
            <>
              <input placeholder="model" value={cfg.model || ""} onChange={(e) => setCfg({ ...cfg, model: e.target.value })} />
              <input placeholder="base_url" value={cfg.base_url || cfg.BaseURL || ""} onChange={(e) => setCfg({ ...cfg, base_url: e.target.value, BaseURL: e.target.value })} />
              <input placeholder="workspace" value={cfg.workspace || cfg.Workspace || ""} onChange={(e) => setCfg({ ...cfg, workspace: e.target.value, Workspace: e.target.value })} />
              <input placeholder="API key" type="password" onBlur={(e) => e.target.value && api.setAPIKey(e.target.value)} />
              <button className="btn" onClick={() => api.setConfig(cfg).then(refresh)}>
                Save
              </button>
            </>
          )}
        </div>
      </section>
    </div>
  );
}
