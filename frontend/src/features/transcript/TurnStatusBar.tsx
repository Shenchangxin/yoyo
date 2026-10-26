// 本文件拥有钉在输入框上方的回合状态条。不拥有转录列表，所以一封冻住的信藏不住仍在跑的回合。Composer 传入 status。

import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { statusBarVisible, type TurnStatus } from "../../lib/turn-status";
import { WorkingLine } from "./ProcessGroup";

/** statusLabel verb 为 other 时用 other 文案并换上工具名，缺名时用 tool。其余动词取 live 里的句子，没有就用 live.think。status 读取 verb、toolName。 */
function statusLabel(status: TurnStatus, other: string, live: Record<string, string>): string {
  if (status.verb === "other") return other.replace("{name}", status.toolName || "tool");
  return live[status.verb] || live.think;
}

/** TurnStatusBar 在 statusBarVisible 为假时不渲染。字节超过 2048 时改显示成 KB。compact 为真时不传停止提示。status 传给 statusBarVisible。 */
export function TurnStatusBar({
  status,
  compact,
}: {
  status?: TurnStatus | null;
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
