import { YoyoMark } from "./shell/YoyoMark";
import { PresenceAnchor } from "./presence";

export function BootSkeleton() {
  return (
    <div className="relative flex h-full min-h-0 overflow-hidden bg-background" aria-busy="true" aria-label="Loading">
      <div className="app-grain" aria-hidden />
      <div className="flex w-60 shrink-0 flex-col bg-sidebar p-3">
        <div className="mb-4 flex h-8 items-center px-0.5">
          <YoyoMark compact />
        </div>
        <div className="skeleton-sweep is-accent h-9 rounded-xl" />
        <div className="skeleton-sweep is-soft mt-2.5 h-8 rounded-xl" />
        <div className="mt-5 space-y-2">
          <div className="skeleton-sweep h-10 rounded-xl opacity-90" />
          <div className="skeleton-sweep h-10 rounded-xl opacity-70" />
          <div className="skeleton-sweep h-10 rounded-xl opacity-50" />
        </div>
      </div>
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="h-12 border-b border-border/40" />
        <div className="flex flex-1 items-center justify-center">
          <PresenceAnchor id="boot" size={132} />
        </div>
        <div className="px-4 pb-5">
          <div className="composer-bezel mx-auto w-full max-w-[min(100%,var(--thread-measure))]">
            <div className="skeleton-sweep h-[84px] rounded-[var(--radius-composer)]" />
          </div>
        </div>
      </div>
    </div>
  );
}
