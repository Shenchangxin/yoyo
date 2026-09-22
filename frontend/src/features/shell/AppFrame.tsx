import type { ComponentProps, ReactNode } from "react";
import { cn } from "../../lib/utils";

export function AppFrame({
  children,
  overlay,
  className,
  ...props
}: ComponentProps<"div"> & { overlay?: ReactNode }) {
  return (
    <div
      className={cn("relative isolate flex h-full w-full flex-col overflow-hidden bg-background text-foreground", className)}
      {...props}
    >
      {overlay}
      <div className="chrome drag relative z-10 flex h-full min-h-0 flex-1 items-stretch overflow-hidden" data-app-frame="content">
        {children}
      </div>
    </div>
  );
}

export function MainColumn({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        "no-drag flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-transparent",
        className,
      )}
      id="main-stage"
      tabIndex={-1}
    >
      {children}
    </div>
  );
}

export function SidebarCard({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        "no-drag flex h-full min-h-0 flex-col overflow-hidden bg-sidebar",
        className,
      )}
    >
      {children}
    </div>
  );
}
