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
    <div className={cn("flex flex-col items-center justify-center px-6 py-14 text-center", props.className)}>
      {props.icon ? <div className="mb-4 text-muted/80">{props.icon}</div> : null}
      <p className="text-[16px] font-semibold tracking-[-0.028em] text-pretty text-foreground">{props.title}</p>
      {props.body ? <p className="mt-2 max-w-[36ch] text-[13px] leading-[1.6] text-muted">{props.body}</p> : null}
      {props.action ? <div className="mt-4">{props.action}</div> : null}
    </div>
  );
}
