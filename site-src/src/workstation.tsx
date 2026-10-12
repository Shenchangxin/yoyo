import { useEffect, useState } from "react";
import type { Copy, GateKind, Scene } from "./copy";

type Props = {
  t: Copy;
  scene: Scene;
  gate: GateKind;
  onGate: (kind: GateKind) => void;
  promoted: boolean;
  onCheckout: () => void;
  refKey: "staging" | "canary" | "active";
  onRef: (key: "staging" | "canary" | "active") => void;
  hunks: [boolean, boolean];
  onHunk: (index: 0 | 1) => void;
  episode: string;
  onEpisode: (id: string) => void;
};

const GATES: GateKind[] = ["once", "session", "always", "deny"];

export function Workstation(props: Props) {
  const { t } = props;
  const chrome =
    props.scene === "diff"
      ? t.demo.review
      : props.scene === "drama"
        ? t.demo.video
        : props.scene === "trust"
          ? t.demo.harbor
          : props.scene === "model"
            ? t.nav.model
            : t.demo.agent;

  return (
    <div className="frame-shell">
      <div className="mini">
        <div className="chrome">
          <span className="mark">yoyo</span>
          <span className="chrome-mod">{chrome}</span>
        </div>
        {props.scene === "ask" ? <Ask t={t} gate={props.gate} onGate={props.onGate} /> : null}
        {props.scene === "model" ? <ModelPane t={t} /> : null}
        {props.scene === "diff" ? <Diff t={t} hunks={props.hunks} onHunk={props.onHunk} /> : null}
        {props.scene === "drama" ? <Drama t={t} episode={props.episode} onEpisode={props.onEpisode} /> : null}
        {props.scene === "trust" ? (
          <TrustPane t={t} promoted={props.promoted} onCheckout={props.onCheckout} refKey={props.refKey} onRef={props.onRef} />
        ) : null}
      </div>
    </div>
  );
}

function useTurnPhase(active: boolean) {
  const [phase, setPhase] = useState(3);
  useEffect(() => {
    if (!active) return;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduce) {
      setPhase(3);
      return;
    }
    setPhase(0);
    const timers = [520, 1280, 2140].map((ms, index) => window.setTimeout(() => setPhase(index + 1), ms));
    return () => timers.forEach((id) => window.clearTimeout(id));
  }, [active]);
  return phase;
}

function Ask({ t, gate, onGate }: { t: Copy; gate: GateKind; onGate: (k: GateKind) => void }) {
  const phase = useTurnPhase(true);
  const hint = {
    once: t.gate.onceHint,
    session: t.gate.sessionHint,
    always: t.gate.alwaysHint,
    deny: t.gate.denyHint,
  }[gate];
  const labels: Record<GateKind, string> = {
    once: t.demo.allowOnce,
    session: t.demo.allowSession,
    always: t.demo.allowAlways,
    deny: t.demo.reject,
  };
  const keys: Record<GateKind, string> = { once: "1", session: "2", always: "3", deny: "Esc" };
  const running = phase > 0 && phase < 3;

  return (
    <div className="scene is-on">
      <div className="panes">
        <aside className="rail">
          <div className="rail-kicker">{t.demo.module}</div>
          <button type="button" className="is-on">
            {t.demo.agent}
          </button>
          <button type="button">{t.demo.video}</button>
          <div className="rail-kicker">{t.demo.chats}</div>
          <button type="button" className="thread is-live">
            {t.demo.threadLive}
            <small>{running ? t.demo.reading : t.demo.threadLiveSub}</small>
          </button>
          <button type="button" className="thread">
            {t.demo.threadIdle}
          </button>
          <button type="button" className="thread">
            {t.demo.newChat}
          </button>
        </aside>
        <div className="mid">
          <div className="mid-body">
            <div className="letter">
              <Stamp />
              <div className="letter-col">
                <p className="bubble rise">{t.demo.bubble}</p>
                {phase >= 1 ? (
                  <div className="proc rise" role="status">
                    <span className={running ? "orb is-live" : "orb"} aria-hidden="true" />
                    {phase === 1 ? <span className="shimmer-text">{t.demo.reading}</span> : <span>{t.demo.read}</span>}
                    <code>loop.go</code>
                    <em>{phase >= 3 ? t.demo.worked : "2s"}</em>
                  </div>
                ) : null}
                {phase >= 2 ? (
                  <div className="proc rise">
                    <span className={phase === 2 ? "orb is-live" : "orb is-done"} aria-hidden="true" />
                    {phase === 2 ? <span className="shimmer-text">{t.demo.patching}</span> : <span>{t.demo.patch}</span>}
                  </div>
                ) : null}
                {phase >= 3 ? (
                  <div className="gate-card rise" role="status">
                    <span className="shield" aria-hidden="true">
                      <svg viewBox="0 0 24 24" width="14" height="14">
                        <path d="M12 3.2 5.2 6.1v5.7c0 4 2.7 7 6.8 8.1 4.1-1.1 6.8-4.1 6.8-8.1V6.1L12 3.2Z" fill="none" stroke="currentColor" strokeWidth="1.6" />
                        <path d="M12 8.2v4.4" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
                        <circle cx="12" cy="15.6" r="0.8" fill="currentColor" />
                      </svg>
                    </span>
                    <div className="gate-main">
                      <div className="gate-kicker">{t.demo.needsApproval}</div>
                      <strong>{t.demo.action}</strong>
                      <em>{t.demo.path}</em>
                      <div className="chips">
                        {GATES.map((kind) => (
                          <button
                            key={kind}
                            type="button"
                            className={kind === gate ? "chip is-hot" : "chip"}
                            data-kind={kind}
                            onClick={() => onGate(kind)}
                          >
                            {labels[kind]}
                            <kbd>{keys[kind]}</kbd>
                          </button>
                        ))}
                      </div>
                      <p className="verdict">{hint}</p>
                    </div>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
          <div className="composer">
            <span className="plan">{t.demo.plan}</span>
            <span className="composer-text">{t.demo.composer}</span>
            <span className={running ? "send is-stop" : "send"} aria-hidden="true">
              {running ? <i /> : (
                <svg viewBox="0 0 16 16" width="12" height="12">
                  <path d="M3 8h8.2M8.2 4.4 12 8l-3.8 3.6" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
              )}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function Stamp() {
  return (
    <span className="stamp" aria-hidden="true">
      <svg viewBox="0 0 32 32" width="22" height="22">
        <circle cx="16" cy="16.2" r="14.2" fill="#2c2c29" />
        <ellipse cx="12.2" cy="12.2" rx="6.4" ry="4.6" fill="#3a3a36" />
        <ellipse cx="11" cy="15" rx="1.7" ry="2.5" fill="#f4f4f0" />
        <ellipse cx="20.2" cy="15" rx="1.7" ry="2.5" fill="#f4f4f0" />
      </svg>
    </span>
  );
}

function ModelPane({ t }: { t: Copy }) {
  const m = t.model;
  return (
    <div className="scene is-on model-pane">
      <p className="model-kicker">
        <b>{m.modelName}</b>
        <span>{m.weightsStill}</span>
      </p>
      <div className="harness">
        <div className="harness-row is-live">
          <i />
          <div>
            <b>{t.trust.active}</b>
            <span>{m.lineLive}</span>
          </div>
          <code>active</code>
        </div>
        <div className="harness-row is-next">
          <i />
          <div>
            <b>{t.trust.canary}</b>
            <span>{m.lineNext}</span>
          </div>
          <code>canary</code>
        </div>
      </div>
      <ul className="layers">
        {(
          [
            [m.l1, m.l1Body],
            [m.l2, m.l2Body],
            [m.l3, m.l3Body],
            [m.l4, m.l4Body],
          ] as const
        ).map(([name, body]) => (
          <li key={name}>
            <b>{name}</b>
            <span>{body}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function Diff({
  t,
  hunks,
  onHunk,
}: {
  t: Copy;
  hunks: [boolean, boolean];
  onHunk: (index: 0 | 1) => void;
}) {
  const n = hunks.filter(Boolean).length;
  return (
    <div className="scene is-on">
      <div className="diffhead">
        <b>loop.go</b>
        <span>
          {n} {t.demo.hunks}
        </span>
        <button type="button" className="apply" disabled={n === 0}>
          {t.demo.apply} {n || ""}
        </button>
      </div>
      <button type="button" className={hunks[0] ? "hunk is-on rise" : "hunk"} onClick={() => onHunk(0)}>
        <i />
        <span className="del">− if err != nil {"{ return }"}</span>
        <span className="add">+ if err != nil {"{ return err }"}</span>
      </button>
      <button type="button" className={hunks[1] ? "hunk is-on rise" : "hunk"} style={{ animationDelay: "80ms" }} onClick={() => onHunk(1)}>
        <i />
        <span className="del">− log.Print(got)</span>
        <span className="add">+ return fmt.Errorf("parse: %w", err)</span>
      </button>
    </div>
  );
}

function Drama({ t, episode, onEpisode }: { t: Copy; episode: string; onEpisode: (id: string) => void }) {
  const lamp = `${import.meta.env.BASE_URL}lamp-dusk.jpg`;
  return (
    <div className="scene is-on">
      <div className="vseg">
        <span>{t.demo.createTab}</span>
        <span className="is-on">{t.demo.video}</span>
        <span>{t.demo.canvasTab}</span>
      </div>
      <div className="vpanel">
        <p className="bubble">{t.demo.dramaPrompt}</p>
        <div className="eps">
          {(
            [
              ["01", t.demo.ep1],
              ["02", t.demo.ep2],
              ["03", t.demo.ep3],
            ] as const
          ).map(([id, label]) => (
            <button key={id} type="button" className={episode === id ? "live" : undefined} onClick={() => onEpisode(id)}>
              <b>{id}</b>
              <span>{label}</span>
              <i />
            </button>
          ))}
        </div>
        <figure className={episode === "02" ? "lamp has-still" : "lamp"}>
          {episode === "02" ? <img src={lamp} alt={t.demo.shotCap} /> : null}
          <figcaption>{episode === "02" ? t.demo.shotCap : t.demo.createHint}</figcaption>
        </figure>
      </div>
    </div>
  );
}

function TrustPane({
  t,
  promoted,
  onCheckout,
  refKey,
  onRef,
}: {
  t: Copy;
  promoted: boolean;
  onCheckout: () => void;
  refKey: "staging" | "canary" | "active";
  onRef: (key: "staging" | "canary" | "active") => void;
}) {
  const note =
    promoted && refKey === "active"
      ? t.trust.done
      : refKey === "staging"
        ? t.trust.stagingHint
        : refKey === "canary"
          ? t.trust.canaryHint
          : t.trust.activeHint;
  return (
    <div className="scene is-on harbor">
      <div className="refs">
        <button type="button" data-ref="staging" className={refKey === "staging" ? "is-on" : undefined} onClick={() => onRef("staging")}>
          <b>{t.trust.staging}</b>
          <i>staging</i>
        </button>
        <button type="button" data-ref="canary" className={refKey === "canary" ? "is-on" : undefined} onClick={() => onRef("canary")}>
          <b>{t.trust.canary}</b>
          <i>canary</i>
        </button>
        <button type="button" data-ref="active" className={refKey === "active" ? "is-on" : undefined} onClick={() => onRef("active")}>
          <b>{t.trust.active}</b>
          <i>active</i>
        </button>
      </div>
      <p className={promoted && refKey === "active" ? "harbor-note is-done" : "harbor-note"}>{note}</p>
      <button type="button" className="checkout" disabled={promoted || refKey !== "canary"} onClick={onCheckout}>
        {t.trust.checkout}
      </button>
    </div>
  );
}
