import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { statusBarVisible, type TurnStatus } from "../../lib/turn-status";
import { WorkingLine } from "./ProcessGroup";

function statusLabel(status: TurnStatus, other: string, live: Record<string, string>): string {
  if (status.verb === "other") return other.replace("{name}", status.toolName || "tool");
  return live[status.verb] || live.think;
}

/**
 * Pinned above the composer. Independent of the virtual transcript so a
 * frozen letter cannot hide the fact that the turn is still running.
 */
export function TurnStatusBar({
  status,
  compact,
}: {
  status: TurnStatus | null;
  compact?: boolean;
}) {
  const copy = useCopy();
  if (!statusBarVisible(status)) return null;
  const st = status;
  const bytes = st.bytes > 2048 ? `${Math.round(st.bytes / 1024)}KB` : "";
  const detail = [st.detail, bytes].filter(Boolean).join(" · ");
  return (
    <div className={cn("u-chrome min-w-0", compact ? "px-1 pb-1" : "pb-1.5")} data-testid="turn-status">
      <WorkingLine
        since={st.since}
        label={statusLabel(st, copy.transcript.live.other, copy.transcript.live)}
        detail={detail}
        hint={compact ? undefined : copy.transcript.live.stopHint}
      />
    </div>
  );
}
