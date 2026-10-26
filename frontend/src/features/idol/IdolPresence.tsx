/**
 * 拥有：通话立绘槽。按阶段在 Spine、照片、字标和宠物之间切换，并在阶段变化时点一次头。
 * 不拥有：通话请求、Spine 绘制循环和宠物表情球循环。它不调用对话或通话接口。
 * 这是投影，不改服务端。方向由传入的阶段和情绪决定。
 */

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useMotionReduced } from "../../lib/motion";
import { PresenceSprite } from "../presence/PresenceRuntime";
import { imageURL } from "./model";
import { nodFor, petFace, standMood } from "./stand";
import type { Emotion, StandDirection } from "./stand/animations";
import { StandView, standURL } from "./stand/StandView";
import type { Framing } from "./stand/engine";
import type { CallPhase } from "./useIdolCall";

/** IdolPresence 渲染当前演员。听或拨号时朝前看，说话且非仅文字时张嘴。从听到想、或开口时按情绪点头或摇头。减少动效时不点头。props 读取 emotion、stand、kind、portrait、name、phase 等。 */
export function IdolPresence(props: {
  name: string;
  color: string;
  portrait: string;
  kind: string;
  stand: string;
  phase: CallPhase;
  emotion: string;
  textOnly?: boolean;
  framing: Framing;
}) {
  const reduced = useMotionReduced();
  const slotRef = useRef<HTMLDivElement>(null);
  const prevPhase = useRef("");
  const nodID = useRef(0);
  const [petSize, setPetSize] = useState(220);
  const [nod, setNod] = useState<StandDirection["nod"]>(null);
  const [spinePhase, setSpinePhase] = useState<"loading" | "shown" | "failed">("loading");
  const mood = standMood(props.emotion) as Emotion;
  const spine = standURL(props.stand);
  const actor = spine ? "spine" : props.kind === "pet" || (!props.portrait && props.kind !== "catalog") ? "pet" : props.portrait ? "photo" : "mark";
  const letter = Array.from(props.name || "偶")[0];
  const speaking = props.phase === "speaking" && !props.textOnly;
  const looking = props.phase === "listening" || props.phase === "dialing";

  useLayoutEffect(() => {
    const el = slotRef.current;
    if (!el || actor !== "pet") return;
    /** 按槽位量宠物尺寸，夹在 160 和 360 之间。 */
    const measure = () => {
      const next = Math.round(Math.min(el.clientWidth * 0.72, el.clientHeight * 0.62));
      setPetSize(Math.max(160, Math.min(360, next || 220)));
    };
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [actor]);

  useEffect(() => {
    const prev = prevPhase.current;
    prevPhase.current = props.phase;
    if (reduced || !prev || prev === props.phase) return;
    let kind: "yes" | "no" | null = null;
    if (prev === "listening" && props.phase === "thinking") kind = "yes";
    else if (props.phase === "speaking") kind = nodFor(mood);
    if (!kind) return;
    nodID.current += 1;
    setNod({ kind, id: nodID.current });
  }, [props.phase, mood, reduced]);

  const direction: StandDirection = {
    mood,
    speaking,
    looking,
    nod,
  };

  return (
    <div
      ref={slotRef}
      className="idol-figure-slot"
      data-testid="idol-figure"
      data-actor={actor}
      data-mood={props.phase === "thinking" ? "thinking" : mood}
      data-phase={props.phase}
    >
      {actor === "spine" ? (
        <>
          {spinePhase === "failed" && props.portrait ? <img className="idol-photo-stand" alt="" src={imageURL(props.portrait)} /> : null}
          <StandView stand={props.stand} framing={props.framing} direction={direction} reduced={reduced} onPhase={setSpinePhase} />
        </>
      ) : actor === "photo" ? (
        <img className="idol-photo-stand" alt="" src={imageURL(props.portrait)} />
      ) : actor === "mark" ? (
        <span className="idol-mark" style={{ background: props.color || "#c47b8a" }}>{letter}</span>
      ) : (
        <div className="idol-figure-pet">
          <PresenceSprite emotionId={petFace(props.phase, mood)} size={petSize} reduced={reduced} followPointer />
        </div>
      )}
    </div>
  );
}
