import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export function LabFrame({ children }: { title?: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="min-h-0 flex-1 overflow-auto">
        <div className="mx-auto max-w-4xl px-8 py-6">{children}</div>
      </div>
    </div>
  );
}

export function LabStat({ label, value, bad }: { label: string; value: string; bad?: boolean }) {
  return (
    <div className="rounded-xl border border-border/80 bg-card p-4">
      <div className="text-[11px] text-muted">{label}</div>
      <div className={cn("mt-1 truncate font-semibold", value.length > 10 ? "font-mono text-sm" : "text-xl tabular-nums", bad && "text-danger")}>{value}</div>
    </div>
  );
}

export function LabChip({ ok, children }: { ok: boolean; children: string }) {
  return (
    <span className={cn("rounded-full px-2 py-0.5 text-[11px] font-medium", ok ? "bg-accent/15 text-accent" : "bg-danger/15 text-danger")}>
      {children}
    </span>
  );
}

export function LabTable({ children }: { children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-xl border border-border/80 bg-card">
      <table className="w-full text-left text-[13px]">{children}</table>
    </div>
  );
}

export function LabCard({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("rounded-xl border border-border/80 bg-card p-4", className)}>{children}</div>;
}
