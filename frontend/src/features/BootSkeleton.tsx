export function BootSkeleton() {
  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-background" aria-busy="true" aria-label="Loading">
      <div className="h-10 shrink-0 border-b border-border bg-sidebar" />
      <div className="flex min-h-0 flex-1">
        <div className="w-56 shrink-0 border-r border-border bg-sidebar p-3">
          <div className="h-9 animate-pulse rounded-xl bg-lift" />
          <div className="mt-2 h-8 animate-pulse rounded-xl bg-lift/70" />
          <div className="mt-4 space-y-2">
            <div className="h-10 animate-pulse rounded-xl bg-lift/50" />
            <div className="h-10 animate-pulse rounded-xl bg-lift/40" />
            <div className="h-10 animate-pulse rounded-xl bg-lift/30" />
          </div>
        </div>
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="h-11 border-b border-border" />
          <div className="flex-1 p-6">
            <div className="h-7 w-48 animate-pulse rounded-lg bg-lift" />
            <div className="mt-3 h-4 w-96 max-w-full animate-pulse rounded bg-lift/60" />
          </div>
          <div className="h-24 border-t border-border p-4">
            <div className="h-full animate-pulse rounded-[var(--radius-composer)] bg-panel" />
          </div>
        </div>
      </div>
    </div>
  );
}
