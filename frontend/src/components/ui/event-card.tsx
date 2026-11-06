// 本文件拥有对话里的事件卡和单行回执。简报推送、交给任务、任务完成、已建定时用事件卡；写进记忆用回执。不拥有点下去做什么。

import type { ReactNode } from "react";
import { Check } from "lucide-react";
import { cn } from "../../lib/utils";

/** EventCard 是一行高的卡：28px 图标块、标题、一句说明、右边动作。整卡可点时传 onClick。 */
export function EventCard(props: {
  icon: ReactNode;
  title: ReactNode;
  detail?: ReactNode;
  actions?: ReactNode;
  onClick?: () => void;
  testId?: string;
  className?: string;
}) {
  const clickable = !!props.onClick;
  return (
    <div
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      data-testid={props.testId}
      className={cn(
        "flex w-full min-w-0 items-center gap-2.5 rounded-lg border border-border bg-card px-2.5 py-2 text-left shadow-[var(--shadow-card)]",
        clickable && "cursor-pointer transition-colors duration-[var(--duration-fast)] hover:bg-lift/40",
        props.className,
      )}
      onClick={props.onClick}
      onKeyDown={clickable ? (event) => {
        if (event.target !== event.currentTarget) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          props.onClick?.();
        }
      } : undefined}
    >
      <span className="grid size-7 shrink-0 place-items-center rounded-md bg-lift text-foreground/85 [&_svg]:size-3.5">{props.icon}</span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-ui-body font-medium text-foreground">{props.title}</span>
        {props.detail ? <span className="block truncate text-ui-footnote text-muted">{props.detail}</span> : null}
      </span>
      {props.actions ? (
        <span className="flex shrink-0 items-center gap-1" onClick={(event) => event.stopPropagation()}>
          {props.actions}
        </span>
      ) : null}
    </div>
  );
}

/** Receipt 是一行回执，比如「已记住：…」。传了 action 时末尾有一个点状下划线的动作。 */
export function Receipt(props: { children: ReactNode; action?: { label: string; onClick: () => void }; className?: string; testId?: string }) {
  return (
    <p className={cn("flex min-w-0 items-center gap-1.5 text-ui-footnote text-muted", props.className)} data-testid={props.testId}>
      <Check className="size-3.5 shrink-0" aria-hidden />
      <span className="min-w-0 truncate">{props.children}</span>
      {props.action ? (
        <button
          type="button"
          className="shrink-0 text-foreground/85 underline decoration-dotted underline-offset-[3px] hover:text-foreground"
          onClick={props.action.onClick}
        >
          {props.action.label}
        </button>
      ) : null}
    </p>
  );
}
