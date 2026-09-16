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
    <div className={cn("flex flex-col items-center justify-center px-4 py-10 text-center", props.className)}>
      {props.icon ? <div className="mb-2 text-muted">{props.icon}</div> : null}
      <p className="text-[13px] font-medium text-foreground">{props.title}</p>
      {props.body ? <p className="mt-1 max-w-xs text-[11px] text-muted">{props.body}</p> : null}
      {props.action ? <div className="mt-3">{props.action}</div> : null}
    </div>
  );
}
