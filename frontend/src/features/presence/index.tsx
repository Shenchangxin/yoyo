/** 拥有把工作台状态投影成在场层的 PresenceLayer，并转出陪伴窗、精灵和导演。不拥有情绪球实现，也不另开模型循环。不改服务端。 */
import { useMemo, type ReactNode } from "react";
import { useBusyMs } from "./busy";
import { PresenceRuntime } from "./PresenceRuntime";
import { presenceOf, type PresenceInput } from "./director";
import { useMotionReduced } from "../../lib/motion";
import { useUI } from "../../lib/store";

/** PresenceAnchor、PresenceSprite、PresenceStamp、PresencePerch、PresenceModuleLoading、COMPOSER_PRESENCE_SIZE 从运行时再导出，定义不在这里。 */
export { PresenceAnchor, PresenceSprite, PresenceStamp, PresencePerch, PresenceModuleLoading, COMPOSER_PRESENCE_SIZE } from "./PresenceRuntime";
/** CompanionApp 从陪伴窗再导出。 */
export { CompanionApp } from "./CompanionApp";
/** presenceOf 从导演再导出。 */
export { presenceOf } from "./director";
/** PresenceInput 从导演再导出。 */
export type { PresenceInput } from "./director";

/** PresenceLayer 用当前语音和加载状态算出一个表情，再交给运行时。 */
export function PresenceLayer(props: { input: Omit<PresenceInput, "voicePhase">; children: ReactNode }) {
  const reduced = useMotionReduced();
  const voicePhase = useUI((s) => s.voicePhase);
  const moduleLoading = useUI((s) => s.moduleLoading);
  const loading = !!(moduleLoading || props.input.moduleLoading);
  const busyMs = useBusyMs(!!(props.input.running || loading), `${props.input.running}:${loading}`);
  const out = useMemo(
    () => presenceOf({
      ...props.input,
      voicePhase,
      moduleLoading: loading,
      busyMs: props.input.busyMs ?? busyMs,
    }),
    [props.input, voicePhase, loading, busyMs],
  );
  return (
    <PresenceRuntime emotionId={out.emotionId} reduced={reduced}>
      {props.children}
    </PresenceRuntime>
  );
}
