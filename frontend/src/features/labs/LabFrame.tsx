// 本文件拥有实验室页的外框、操作带、统计、表格和步骤行。不拥有实验数据。HarnessLab、HarborLab、EvolveLab 和 HarnessWorkspace 把内容放进 LabFrame。

import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

// LabFrame 只把 children 放进一条可滚动的居中栏。类型上的 title 和 hint 当前不渲染。
export function LabFrame({ children }: { title?: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="min-h-0 flex-1 overflow-auto">
        <div className="mx-auto w-full max-w-6xl px-6 py-6">{children}</div>
      </div>
    </div>
  );
}

// LabStrip 是一条操作带，放主动作和溢出项，不是一长串表单。
export function LabStrip({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("mb-6 flex flex-wrap items-center gap-2 px-0.5 py-1", className)}>
      {children}
    </div>
  );
}

// LabEyebrow 用 11px 弱色画出小节眉题。
export function LabEyebrow({ children }: { children: ReactNode }) {
  return <div className="text-[11px] font-medium text-muted">{children}</div>;
}

// LabStat 在 value 长于 10 个字符时改用 13px 等宽字。bad 为真时数值用危险色。
export function LabStat({ label, value, bad }: { label: string; value: string; bad?: boolean }) {
  return (
    <div className="px-0.5 py-2">
      <div className="text-[11px] text-muted">{label}</div>
      <div className={cn("mt-1 truncate font-medium tabular-nums", value.length > 10 ? "font-mono text-[13px]" : "text-[15px]", bad && "text-danger")}>{value}</div>
    </div>
  );
}

// LabChip 在 ok 为真时用抬起色，否则用危险色。
export function LabChip({ ok, children }: { ok: boolean; children: string }) {
  return (
    <span className={cn("rounded-full px-2 py-0.5 text-[11px] font-medium", ok ? "bg-lift text-foreground" : "bg-danger/15 text-danger")}>
      {children}
    </span>
  );
}

// LabTable 把表格收进圆角容器，表本身仍由 children 提供。
export function LabTable({ children }: { children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-2xl bg-sidebar/50">
      <table className="w-full text-left text-[13px]">{children}</table>
    </div>
  );
}

// LabCard 是一块带卡片阴影的圆角底。
export function LabCard({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("rounded-2xl bg-sidebar/50 p-4 shadow-[var(--shadow-card)]", className)}>{children}</div>;
}

// StoryRow 画出一步的标签、说明和值。last 为真时不画底边。ok 为假时值用危险色。step 缺了就不留序号。
export function StoryRow(props: { step?: string; label: string; detail: string; value: string; ok: boolean; last?: boolean }) {
  return (
    <div className={cn("flex items-start gap-3 px-4 py-3", !props.last && "border-b border-border/70")}>
      {props.step ? <span className="mt-0.5 w-4 font-mono text-[11px] text-muted">{props.step}</span> : null}
      <div className="min-w-0 flex-1">
        <div className="text-[13px] font-medium text-foreground">{props.label}</div>
        <div className="text-[12px] text-muted">{props.detail}</div>
      </div>
      <span className={cn("tabular-nums text-[13px]", !props.ok && "text-danger")}>{props.value}</span>
    </div>
  );
}
