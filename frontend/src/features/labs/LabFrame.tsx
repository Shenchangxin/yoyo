import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export function LabFrame({ children }: { title?: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="min-h-0 flex-1 overflow-auto">
        <div className="mx-auto w-full max-w-6xl px-6 py-6">{children}</div>
      </div>
    </div>
  );
}

/** One instrument strip: primary verb + overflow, not a form dump. */
export function LabStrip({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("mb-6 flex flex-wrap items-center gap-2 rounded-[12px] border border-border/80 bg-card/80 px-3 py-2.5 surface-inset", className)}>
      {children}
    </div>
  );
}

export function LabEyebrow({ children }: { children: ReactNode }) {
  return <div className="text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80">{children}</div>;
}

export function LabStat({ label, value, bad }: { label: string; value: string; bad?: boolean }) {
  return (
    <div className="surface-inset rounded-xl border border-border/80 bg-card px-3.5 py-3">
      <div className="text-[11px] text-muted">{label}</div>
      <div className={cn("mt-1 truncate font-medium tabular-nums", value.length > 10 ? "font-mono text-[13px]" : "text-[15px]", bad && "text-danger")}>{value}</div>
    </div>
  );
}

export function LabChip({ ok, children }: { ok: boolean; children: string }) {
  return (
    <span className={cn("rounded-full px-2 py-0.5 text-[11px] font-medium", ok ? "bg-lift text-foreground" : "bg-danger/15 text-danger")}>
      {children}
    </span>
  );
}

export function LabTable({ children }: { children: ReactNode }) {
  return (
    <div className="surface-inset overflow-hidden rounded-xl border border-border/80 bg-card">
      <table className="w-full text-left text-[13px]">{children}</table>
    </div>
  );
}

export function LabCard({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("surface-inset rounded-xl border border-border/80 bg-card p-4", className)}>{children}</div>;
}
