/**
 * 拥有：产品站里那扇假窗口，用来演示提问、差异、短片和材料指针。
 * 不拥有：真正的会话和文件写入。点选只改这一页的状态。
 */

import { useEffect, useState } from "react";
import type { Copy, GateKind, Scene } from "./copy";
import { ShotLight } from "./stage-canvas";

export type VideoTab = "create" | "video" | "canvas";

type Props = {
  t: Copy;
  scene: Scene;
  onScene: (scene: Scene) => void;
  embed?: boolean;
  videoTab?: VideoTab;
  onVideoTab?: (tab: VideoTab) => void;
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

/** Workstation 按 scene 换中间那一栏，标题栏上的模块名跟着文案走。 */
export function Workstation(props: Props) {
  const { t, scene, onScene } = props;
  const phase = 3;
  const chrome =
    scene === "diff" ? t.demo.review : scene === "drama" ? t.demo.video : scene === "trust" ? t.demo.harbor : t.demo.agent;

  return (
    <div className={props.embed ? "window is-embed" : "window"}>
      <div className="w-title">
        <span className="w-name">yoyo</span>
        <span className="w-module">{chrome}</span>
        <span className="w-model">{t.model.modelName}</span>
      </div>
      <div className="w-body">
        <aside className="rail">
          <p className="rail-label">{t.demo.module}</p>
          <button type="button" className={scene === "ask" ? "is-on" : undefined} onClick={() => onScene("ask")}>
            {t.demo.agent}
          </button>
          <button type="button" className={scene === "diff" ? "is-on" : undefined} onClick={() => onScene("diff")}>
            {t.demo.review}
          </button>
          <button type="button" className={scene === "drama" ? "is-on" : undefined} onClick={() => onScene("drama")}>
            {t.demo.video}
          </button>
          <button type="button" className={scene === "trust" ? "is-on" : undefined} onClick={() => onScene("trust")}>
            {t.demo.harbor}
          </button>
          <p className="rail-label">{t.demo.chats}</p>
          <button type="button" className={scene === "ask" ? "thread is-live is-on" : "thread is-live"} onClick={() => onScene("ask")}>
            {t.demo.threadLive}
            <small>{t.demo.threadLiveSub}</small>
          </button>
          <button type="button" className={scene === "trust" ? "thread is-on" : "thread"} onClick={() => onScene("trust")}>
            {t.demo.threadIdle}
          </button>
          <p className="thread is-quiet">{t.demo.newChat}</p>
        </aside>
        <div className="mid">
          {scene === "ask" ? <Ask t={t} phase={phase} gate={props.gate} onGate={props.onGate} /> : null}
          {scene === "diff" ? <Diff t={t} hunks={props.hunks} onHunk={props.onHunk} /> : null}
          {scene === "drama" ? (
            <Drama t={t} episode={props.episode} onEpisode={props.onEpisode} tab={props.videoTab} onTab={props.onVideoTab} />
          ) : null}
          {scene === "trust" ? (
            <TrustPane t={t} promoted={props.promoted} onCheckout={props.onCheckout} refKey={props.refKey} onRef={props.onRef} />
          ) : null}
        </div>
        {scene === "ask" || scene === "diff" ? (
          <aside className="queue">
            <p className="rail-label">{t.demo.queue}</p>
            <button type="button" onClick={() => onScene("diff")}>
              <b>loop.go</b>
              <span>+24 −6</span>
            </button>
          </aside>
        ) : null}
      </div>
      <div className="w-status">
        <span>{t.demo.statusFolder}</span>
        <span>{t.model.modelName}</span>
        <span>{t.demo.local}</span>
      </div>
    </div>
  );
}

/** Ask 演示一次提问和审批。phase 在 0 和 3 之间时发送钮画成停止。 */
function Ask({ t, phase, gate, onGate }: { t: Copy; phase: number; gate: GateKind; onGate: (k: GateKind) => void }) {
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
    <div className="pane">
      <div className="pane-scroll">
        <p className="hello">{t.demo.hello}</p>
        <p className="bubble">{t.demo.bubble}</p>
        {phase >= 1 ? (
          <p className="proc" role="status">
            <i className={running && phase === 1 ? "dot is-live" : "dot"} aria-hidden="true" />
            <span>{phase === 1 ? t.demo.reading : t.demo.read}</span>
            <code>loop.go</code>
          </p>
        ) : null}
        {phase >= 2 ? (
          <p className="proc">
            <i className={phase === 2 ? "dot is-live" : "dot is-done"} aria-hidden="true" />
            <span>{phase === 2 ? t.demo.patching : t.demo.patch}</span>
          </p>
        ) : null}
        {phase >= 3 ? (
          <div className="gate-card" role="status">
            <p className="gate-kicker">{t.demo.needsApproval}</p>
            <p className="gate-action">
              <strong>{t.demo.action}</strong>
              <code>{t.demo.path}</code>
            </p>
            <div className="chips">
              {GATES.map((kind) => (
                <button
                  key={kind}
                  type="button"
                  className={kind === gate ? "chip is-hot" : "chip"}
                  data-kind={kind}
                  aria-pressed={kind === gate}
                  onClick={() => onGate(kind)}
                >
                  {labels[kind]}
                  <kbd>{keys[kind]}</kbd>
                </button>
              ))}
            </div>
            <p className="verdict">{hint}</p>
          </div>
        ) : null}
      </div>
      <div className="composer" aria-hidden="true">
        <span className="plan">{t.demo.plan}</span>
        <span className="composer-text">{t.demo.composer}</span>
        <span className={running ? "send is-stop" : "send"} />
      </div>
    </div>
  );
}

/** Diff 演示勾选两段差异。没有选中或已经点过应用时，按钮不可用。hunks 变化会把已应用状态清掉。 */
function Diff({ t, hunks, onHunk }: { t: Copy; hunks: [boolean, boolean]; onHunk: (index: 0 | 1) => void }) {
  const [kept, setKept] = useState(false);
  const n = hunks.filter(Boolean).length;
  useEffect(() => {
    setKept(false);
  }, [hunks]);

  return (
    <div className="pane diff-pane">
      <div className="diffhead">
        <b>loop.go</b>
        <span>
          {n} {t.demo.hunks}
        </span>
        <button type="button" className="apply" disabled={n === 0 || kept} onClick={() => setKept(true)}>
          {kept ? t.demo.kept : `${t.demo.apply}${n ? ` ${n}` : ""}`}
        </button>
      </div>
      <div className="diff-file">
        <p className="ctx">func parse(src string) error {"{"}</p>
        <Hunk on={hunks[0]} onToggle={() => onHunk(0)} del="− if err != nil { return }" add="+ if err != nil { return err }" />
        <Hunk on={hunks[1]} onToggle={() => onHunk(1)} del="− log.Print(got)" add='+ return fmt.Errorf("parse: %w", err)' />
        <p className="ctx">{"  return nil"}</p>
        <p className="ctx">{"}"}</p>
        {kept ? <p className="kept-note">{t.demo.keptNote}</p> : null}
      </div>
    </div>
  );
}

/** Hunk 是一段可勾选的删除行和新增行。 */
function Hunk({ on, onToggle, del, add }: { on: boolean; onToggle: () => void; del: string; add: string }) {
  return (
    <button type="button" className={on ? "hunk is-on" : "hunk"} aria-pressed={on} onClick={onToggle}>
      <i aria-hidden="true" />
      <span className="del">{del}</span>
      <span className="add">{add}</span>
    </button>
  );
}

/** Drama 在创作、成片和画布三个页签之间切换，并列出示例分集。 */
function Drama({
  t,
  episode,
  onEpisode,
  tab: tabProp,
  onTab,
}: {
  t: Copy;
  episode: string;
  onEpisode: (id: string) => void;
  tab?: VideoTab;
  onTab?: (tab: VideoTab) => void;
}) {
  const [localTab, setLocalTab] = useState<VideoTab>("video");
  const tab = tabProp ?? localTab;
  const setTab = (next: VideoTab) => {
    onTab?.(next);
    if (tabProp === undefined) setLocalTab(next);
  };
  const eps = [
    ["01", t.demo.ep1],
    ["02", t.demo.ep2],
    ["03", t.demo.ep3],
  ] as const;

  return (
    <div className="pane drama-pane">
      <div className="vseg" role="tablist" aria-label={t.demo.video}>
        {(
          [
            ["create", t.demo.createTab],
            ["video", t.demo.video],
            ["canvas", t.demo.canvasTab],
          ] as const
        ).map(([id, label]) => (
          <button key={id} type="button" role="tab" aria-selected={tab === id} onClick={() => setTab(id)}>
            {label}
          </button>
        ))}
      </div>
      {tab === "create" ? (
        <div className="vbody">
          <p className="bubble">{t.demo.createHint}</p>
          <figure className="shot">
            <ShotLight lit />
            <figcaption>{t.demo.createHint}</figcaption>
          </figure>
          <p className="vcopy">{t.demo.createBody}</p>
        </div>
      ) : null}
      {tab === "video" ? (
        <div className="vbody">
          <p className="bubble">{t.demo.dramaPrompt}</p>
          <div className="eps">
            {eps.map(([id, label]) => (
              <button key={id} type="button" className={episode === id ? "is-on" : undefined} aria-pressed={episode === id} onClick={() => onEpisode(id)}>
                <b>{id}</b>
                <span>{label}</span>
              </button>
            ))}
          </div>
          <figure className="shot">
            <ShotLight lit={episode === "02"} />
            <figcaption>{episode === "02" ? t.demo.shotCap : t.demo.createHint}</figcaption>
          </figure>
        </div>
      ) : null}
      {tab === "canvas" ? (
        <div className="vbody">
          <div className="board">
            {eps.map(([id, label]) => (
              <button key={id} type="button" className={episode === id ? "is-on" : undefined} aria-pressed={episode === id} onClick={() => onEpisode(id)}>
                <i />
                <span>
                  {id} {label}
                </span>
              </button>
            ))}
            <div className="board-empty" aria-hidden="true" />
          </div>
          <p className="vcopy">{t.demo.canvasHint}</p>
        </div>
      ) : null}
    </div>
  );
}

/** TrustPane 演示 staging、canary 和 active 三个名字。只有 promoted 且当前是 active 时显示完成说明。点 checkout 由外层把指针改成 active。 */
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
  const refs = [
    ["staging", t.trust.staging],
    ["canary", t.trust.canary],
    ["active", t.trust.active],
  ] as const;

  return (
    <div className="pane trust-pane">
      <div className="w-refs" role="radiogroup" aria-label={t.nav.trust}>
        {refs.map(([id, label]) => (
          <button key={id} type="button" role="radio" data-ref={id} aria-checked={refKey === id} onClick={() => onRef(id)}>
            <b>{label}</b>
            <i>{id}</i>
          </button>
        ))}
      </div>
      <p className="w-ref-note" aria-live="polite">
        {note}
      </p>
      <button type="button" className="apply" disabled={promoted || refKey !== "canary"} onClick={onCheckout}>
        {t.trust.checkout}
      </button>
    </div>
  );
}
