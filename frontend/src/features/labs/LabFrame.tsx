import type { ReactNode } from "react";

export function LabFrame({ title, hint, children }: { title: string; hint: string; children: ReactNode }) {
  return (
    <div className="flex h-full min-h-0 flex-col bg-background">
      <div className="shrink-0 border-b border-border px-8 py-5">
        <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
        <p className="mt-1 max-w-2xl text-sm text-muted">{hint}</p>
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        <div className="mx-auto max-w-4xl px-8 py-6">{children}</div>
      </div>
    </div>
  );
}
