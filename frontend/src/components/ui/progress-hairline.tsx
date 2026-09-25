import { cn } from "../../lib/utils";

/** 2px track. Determinate width is evidence; indeterminate only while work is live. */
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
