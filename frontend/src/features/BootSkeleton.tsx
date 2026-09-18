export function BootSkeleton() {
  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-background p-2" aria-busy="true" aria-label="Loading">
      <div className="flex min-h-0 flex-1 gap-2">
        <div className="w-56 shrink-0 rounded-[10px] border border-border bg-sidebar p-3">
          <div className="h-9 animate-pulse rounded-xl bg-lift" />
          <div className="mt-2 h-8 animate-pulse rounded-xl bg-lift/70" />
          <div className="mt-4 space-y-2">
            <div className="h-10 animate-pulse rounded-xl bg-lift/50" />
            <div className="h-10 animate-pulse rounded-xl bg-lift/40" />
            <div className="h-10 animate-pulse rounded-xl bg-lift/30" />
          </div>
        </div>
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <div className="h-11 border-b border-border/50" />
          <div className="flex-1" />
          <div className="px-4 pb-4">
            <div className="mx-auto h-[72px] max-w-2xl animate-pulse rounded-[var(--radius-composer)] border border-border bg-input-bar" />
          </div>
        </div>
      </div>
    </div>
  );
}
