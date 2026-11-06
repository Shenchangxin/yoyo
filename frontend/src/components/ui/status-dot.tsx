// 本文件拥有状态点：需要你、正在做、完成、失败、安静。颜色来自 --status-* 令牌。不拥有状态怎么算。

import { cn } from "../../lib/utils";

export type StatusTone = "needs" | "live" | "done" | "failed" | "idle";

/** StatusDot 画一个 6px 的点。live 带呼吸外圈，只在真的在跑时传；still 为真时 live 也是静止的蓝点（比如简报里"在动"的工程线）。 */
export function StatusDot({ tone, still, className }: { tone: StatusTone; still?: boolean; className?: string }) {
  if (tone === "live" && !still) return <span className={cn("pulse-dot is-accent", className)} aria-hidden />;
  return (
    <span
      aria-hidden
      className={cn(
        "inline-block size-1.5 shrink-0 rounded-full",
        tone === "live" && "bg-status-live",
        tone === "needs" && "bg-status-needs",
        tone === "done" && "bg-status-done",
        tone === "failed" && "bg-status-failed",
        tone === "idle" && "border border-status-idle/70 bg-transparent",
        className,
      )}
    />
  );
}
