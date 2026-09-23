import { YoyoMark } from "./shell/YoyoMark";

export function BootSkeleton() {
  return (
    <div className="relative flex h-full min-h-0 overflow-hidden bg-background" aria-busy="true" aria-label="Loading">
      <div className="app-grain" aria-hidden />
      <div className="flex w-60 shrink-0 flex-col bg-sidebar p-3">
        <div className="mb-4 flex h-8 items-center px-0.5">
          <YoyoMark compact />
        </div>
        <div className="h-9 animate-pulse rounded-xl bg-accent/25" />
        <div className="mt-2.5 h-8 animate-pulse rounded-xl bg-lift/70" />
        <div className="mt-5 space-y-2">
          <div className="h-10 animate-pulse rounded-xl bg-lift/50" />
          <div className="h-10 animate-pulse rounded-xl bg-lift/40" />
          <div className="h-10 animate-pulse rounded-xl bg-lift/30" />
        </div>
      </div>
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="h-12 border-b border-border/40" />
        <div className="flex-1" />
        <div className="px-4 pb-5">
          <div className="composer-bezel mx-auto w-full max-w-[min(100%,var(--thread-measure))]">
            <div className="h-[84px] animate-pulse rounded-[var(--radius-composer)] bg-input-bar" />
          </div>
        </div>
      </div>
    </div>
  );
}
