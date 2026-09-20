import { YoyoMark } from "./shell/YoyoMark";

export function BootSkeleton() {
  return (
    <div className="relative flex h-full min-h-0 flex-col overflow-hidden bg-background p-2" aria-busy="true" aria-label="Loading">
      <div className="app-grain" aria-hidden />
      <div className="relative z-10 flex min-h-0 flex-1 gap-2">
        <div className="w-56 shrink-0 rounded-[10px] border border-border bg-sidebar p-3 surface-inset">
          <div className="mb-3 flex h-8 items-center gap-2 px-0.5">
            <YoyoMark className="size-3.5 text-muted" />
            <span className="h-2.5 w-12 rounded-sm bg-lift" />
          </div>
          <div className="h-9 animate-pulse rounded-lg bg-lift" />
          <div className="mt-2 h-8 animate-pulse rounded-lg bg-lift/70" />
          <div className="mt-4 space-y-2">
            <div className="h-10 animate-pulse rounded-lg bg-lift/50" />
            <div className="h-10 animate-pulse rounded-lg bg-lift/40" />
            <div className="h-10 animate-pulse rounded-lg bg-lift/30" />
          </div>
        </div>
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <div className="h-12 border-b border-border/30" />
          <div className="flex-1" />
          <div className="px-4 pb-4">
            <div className="mx-auto h-[72px] w-full max-w-[min(100%,var(--thread-measure))] animate-pulse rounded-[var(--radius-composer)] border border-border bg-input-bar shadow-[var(--shadow-composer)]" />
          </div>
        </div>
      </div>
    </div>
  );
}
