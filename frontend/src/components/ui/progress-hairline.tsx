// 本文件拥有一条 2px 进度细线。不拥有进度从哪计算。调用方在工作仍在进行时传 indeterminate，否则传 value。

import { cn } from "../../lib/utils";

// ProgressHairline 用宽度表示确定进度，并把它夹在 0 到 100。不确定态不报 aria-valuenow，也不设宽度。
export function ProgressHairline({
  value,
  indeterminate,
  className,
}: {
  value?: number;
  indeterminate?: boolean;
  className?: string;
}) {
  const pct = Math.min(100, Math.max(0, value ?? 0));
  return (
    <span
      className={cn("progress-hairline", className)}
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={indeterminate ? undefined : Math.round(pct)}
    >
      <span
        className={indeterminate ? "is-indeterminate" : undefined}
        style={indeterminate ? undefined : { width: `${pct}%` }}
      />
    </span>
  );
}
