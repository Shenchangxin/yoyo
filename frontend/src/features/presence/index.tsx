import { useMemo, type ReactNode } from "react";
import { useBusyMs } from "./busy";
import { PresenceRuntime } from "./PresenceRuntime";
import { presenceOf, type PresenceInput } from "./director";
import { useMotionReduced } from "../../lib/motion";
import { useUI } from "../../lib/store";

export { PresenceAnchor, PresenceSprite, PresenceStamp, PresenceModuleLoading } from "./PresenceRuntime";
export { CompanionApp } from "./CompanionApp";
export { presenceOf } from "./director";
export type { PresenceInput } from "./director";

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
