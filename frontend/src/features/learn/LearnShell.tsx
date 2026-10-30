/** 学习页的页头。一次只放一个主动作，标签和分段都不换行。 */

import type { ReactNode } from "react";
import { ArrowLeft } from "lucide-react";
import { cn } from "../../lib/utils";

/** LearnShell 画出返回、标题、一句说明和可选的一个动作。scroll 为假时内容区自己撑满高度。 */
export function LearnShell(props: {
  title: string;
  subtitle?: string;
  action?: ReactNode;
  back?: { label: string; onClick: () => void };
  tabs?: ReactNode;
  children: ReactNode;
  scroll?: boolean;
}) {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="shrink-0 px-8 pt-6">
        {props.back ? (
          <button
            type="button"
            className="mb-3 inline-flex items-center gap-1.5 text-[12px] text-muted hover:text-foreground"
            onClick={props.back.onClick}
          >
            <ArrowLeft className="size-3.5 shrink-0" aria-hidden />
            <span className="whitespace-nowrap">{props.back.label}</span>
          </button>
        ) : null}
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h1 className="truncate text-[21px] font-semibold tracking-[-0.03em]">{props.title}</h1>
            {props.subtitle ? <p className="mt-1 max-w-[48ch] text-[13px] leading-relaxed text-muted">{props.subtitle}</p> : null}
          </div>
          {props.action ? <div className="shrink-0">{props.action}</div> : null}
        </div>
        {props.tabs ? <div className="mt-4">{props.tabs}</div> : null}
      </header>
      <div className={cn("min-h-0 flex-1", props.scroll === false ? "flex flex-col" : "overflow-auto")}>{props.children}</div>
    </div>
  );
}

/** LearnTabs 用下划线切换同一页里的几块。选项不换行，窄时横向滚动。 */
export function LearnTabs(props: {
  label: string;
  items: { id: string; label: string }[];
  value: string;
  onChange: (id: string) => void;
}) {
  return (
    <div role="tablist" aria-label={props.label} className="flex gap-1 overflow-x-auto border-b border-border/60">
      {props.items.map((item) => {
        const on = props.value === item.id;
        return (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={on}
            className={cn(
              "-mb-px shrink-0 whitespace-nowrap border-b-2 px-3 pb-2 text-[13px]",
              on ? "border-foreground text-foreground" : "border-transparent text-muted hover:text-foreground",
            )}
            onClick={() => props.onChange(item.id)}
          >
            {item.label}
          </button>
        );
      })}
    </div>
  );
}

/** LearnSegment 是一组互斥的短标签。每个标签保持一行。 */
export function LearnSegment(props: {
  label: string;
  value: string;
  options: { id: string; label: string }[];
  onChange: (id: string) => void;
}) {
  return (
    <div role="group" aria-label={props.label} className="inline-flex h-8 max-w-full items-center overflow-x-auto rounded-lg bg-lift p-0.5">
      {props.options.map((opt) => {
        const on = props.value === opt.id;
        return (
          <button
            key={opt.id}
            type="button"
            aria-pressed={on}
            className={cn(
              "h-7 shrink-0 whitespace-nowrap rounded-md px-2.5 text-[12px]",
              on ? "bg-background text-foreground" : "text-muted hover:text-foreground",
            )}
            onClick={() => props.onChange(opt.id)}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
