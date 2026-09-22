import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export function EmptyState(props: {
  icon?: ReactNode;
  title: string;
  body?: string;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col items-center justify-center px-6 py-12 text-center", props.className)}>
      {props.icon ? <div className="mb-3 text-muted/80">{props.icon}</div> : null}
      <p className="text-[15px] font-medium tracking-[-0.02em] text-pretty text-foreground">{props.title}</p>
      {props.body ? <p className="mt-1.5 max-w-sm text-[13px] leading-[1.55] text-muted">{props.body}</p> : null}
      {props.action ? <div className="mt-3">{props.action}</div> : null}
    </div>
  );
}
