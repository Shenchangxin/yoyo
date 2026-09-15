import type { HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

export function Badge({ className, ...props }: HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        "inline-flex max-w-[220px] items-center truncate rounded-full border border-border bg-lift px-2.5 py-0.5 text-[11px] tabular-nums text-muted",
        className,
      )}
      {...props}
    />
  );
}
