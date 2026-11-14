/**
 * 拥有：今天栏里反复出现的那几样小东西：软卡片、分组标题、圆钮、胶囊按钮、文字按钮、圆开关、返回键、页标题，以及栏里的输入框。
 * 样子照着今天、任务、记忆现有的写法，字号走令牌，大圆角走 day-pane.css。Buddy 页用这一套，和旁边三页是一个样子。
 * 不拥有：任何数据和动作。WorkPane.tsx 里原有的组件没有改成用这里的。
 */

import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, TextareaHTMLAttributes } from "react";
import { ChevronLeft } from "lucide-react";
import { cn } from "../../lib/utils";
import "./day-pane.css";

/** DayCard 是大卡片，放一段完整的内容。 */
export function DayCard(props: { children: ReactNode; className?: string; testId?: string; label?: string }) {
  return (
    <section className={cn("day-card px-5 py-5", props.className)} data-testid={props.testId} aria-label={props.label}>
      {props.children}
    </section>
  );
}

/** DayTile 是小卡片，放列表里的一行或一组行。 */
export function DayTile(props: { children: ReactNode; className?: string; testId?: string }) {
  return (
    <div className={cn("day-tile px-4", props.className)} data-testid={props.testId}>
      {props.children}
    </div>
  );
}

/** DaySection 是一组内容：标题、可选的个数和右侧的小动作，下面是内容。person 为真时个数用提醒色。 */
export function DaySection(props: { title: string; count?: number; person?: boolean; hint?: string; action?: ReactNode; children: ReactNode; testId?: string }) {
  return (
    <section className="mt-6" data-testid={props.testId} aria-label={props.title}>
      <div className="flex items-center justify-between gap-2 px-1">
        <h2 className="flex min-w-0 items-center gap-1.5 text-ui-title-s font-medium">
          <span className="truncate">{props.title}</span>
          {props.count ? <span className={cn("shrink-0 tabular-nums text-ui-caption font-normal", props.person ? "text-warning" : "text-muted")}>{props.count}</span> : null}
        </h2>
        {props.action}
      </div>
      {props.hint ? <p className="mt-1 px-1 text-ui-footnote leading-5 text-muted">{props.hint}</p> : null}
      <div className="mt-2.5">{props.children}</div>
    </section>
  );
}

/** DayTitle 是一页的标题。 */
export function DayTitle(props: { children: ReactNode; className?: string }) {
  return <h1 className={cn("text-ui-title font-semibold tracking-[-0.03em]", props.className)}>{props.children}</h1>;
}

/** DayBack 是页面左上角的返回圆钮。 */
export function DayBack(props: { label: string; onClick: () => void; testId?: string }) {
  return (
    <button
      type="button"
      className="grid size-8 shrink-0 place-items-center rounded-full text-muted hover:bg-lift hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
      aria-label={props.label}
      data-testid={props.testId}
      onClick={props.onClick}
    >
      <ChevronLeft className="size-4" aria-hidden />
    </button>
  );
}

/** DayRound 是卡片底色的圆钮，放一个图标。quiet 为真时没有底色，悬停才有。 */
export function DayRound(props: ButtonHTMLAttributes<HTMLButtonElement> & { label: string; quiet?: boolean }) {
  const { label, quiet, className, children, ...rest } = props;
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className={cn(
        "grid size-8 shrink-0 place-items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 disabled:opacity-40",
        quiet ? "text-muted hover:bg-lift hover:text-foreground" : "border border-border bg-card text-foreground shadow-[var(--shadow-card)] hover:bg-lift",
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

/**
 * DayPill 是胶囊按钮。tone 为 solid 是一页里最要紧的那一个动作（强调色微透或深底），soft 是次要动作（浅底描边），
 * danger 是要人再想一下的动作。small 用在一行里，避免纯白实心斑点过曝。
 */
export function DayPill(props: ButtonHTMLAttributes<HTMLButtonElement> & { tone?: "solid" | "soft" | "danger"; small?: boolean; busy?: boolean }) {
  const { tone = "solid", small, busy, className, children, disabled, ...rest } = props;
  return (
    <button
      type="button"
      disabled={disabled || busy}
      aria-busy={busy || undefined}
      className={cn(
        "inline-flex shrink-0 items-center justify-center gap-1.5 rounded-full font-medium transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 disabled:opacity-40",
        small ? "h-7 px-3 text-ui-footnote" : "h-9 px-4 text-ui-body",
        tone === "solid" && (small ? "border border-accent/25 bg-accent/14 text-accent hover:bg-accent/22" : "bg-foreground text-background hover:opacity-[0.92]"),
        tone === "soft" && "border border-border bg-lift/75 text-foreground hover:bg-lift",
        tone === "danger" && "border border-danger/20 bg-danger/10 text-danger hover:bg-danger/15",
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

/** DayText 是一行里的文字按钮。danger 为真时悬停变成提醒色。 */
export function DayText(props: ButtonHTMLAttributes<HTMLButtonElement> & { danger?: boolean }) {
  const { danger, className, children, ...rest } = props;
  return (
    <button
      type="button"
      className={cn(
        "h-7 shrink-0 rounded-lg px-2.5 text-ui-footnote font-medium text-muted hover:bg-lift focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 disabled:opacity-40",
        danger ? "hover:text-danger" : "hover:text-foreground",
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

/** DaySwitch 是这一栏的圆开关。 */
export function DaySwitch(props: { on: boolean; label: string; disabled?: boolean; onChange: (on: boolean) => void; testId?: string }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={props.on}
      aria-label={props.label}
      disabled={props.disabled}
      data-testid={props.testId}
      className={cn(
        "relative h-7 w-12 shrink-0 rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 disabled:opacity-50",
        props.on ? "bg-accent" : "bg-lift",
      )}
      onClick={() => props.onChange(!props.on)}
    >
      <span className={cn("absolute top-0.5 size-6 rounded-full bg-white shadow-sm transition-[left]", props.on ? "left-5" : "left-0.5")} />
    </button>
  );
}

/** DayRow 是卡片里的一行设置：左边名字和一句说明，右边一个控件。 */
export function DayRow(props: { title: string; hint?: string; children: ReactNode; className?: string }) {
  return (
    <div className={cn("flex items-center justify-between gap-3", props.className)}>
      <div className="min-w-0">
        <p className="text-ui-body-l">{props.title}</p>
        {props.hint ? <p className="mt-0.5 text-ui-footnote leading-5 text-muted">{props.hint}</p> : null}
      </div>
      {props.children}
    </div>
  );
}

/** DayField 是一个带名字的输入：名字在上，控件在下，底下可以有一句说明。 */
export function DayField(props: { label: string; htmlFor?: string; hint?: string; children: ReactNode; className?: string }) {
  return (
    <div className={cn("space-y-1.5", props.className)}>
      <label htmlFor={props.htmlFor} className="block px-1 text-ui-footnote text-muted">{props.label}</label>
      {props.children}
      {props.hint ? <p className="px-1 text-ui-caption leading-4 text-muted">{props.hint}</p> : null}
    </div>
  );
}

/** DAY_INPUT 是栏里单行输入的样子，下拉的触发器也用它。 */
export const DAY_INPUT = "h-9 w-full rounded-xl border border-border bg-lift/75 px-3.5 text-ui-body text-foreground shadow-none outline-none placeholder:text-muted focus-visible:border-accent/50 focus-visible:ring-2 focus-visible:ring-accent/20 disabled:opacity-50";

/** DayInput 是栏里的单行输入。mono 为真时用等宽字。 */
export function DayInput(props: InputHTMLAttributes<HTMLInputElement> & { mono?: boolean }) {
  const { mono, className, ...rest } = props;
  return <input className={cn(DAY_INPUT, mono && "font-mono text-ui-body", className)} {...rest} />;
}

/** DayArea 是栏里的多行输入，等宽字，给参数和环境变量用。 */
export function DayArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const { className, ...rest } = props;
  return (
    <textarea
      spellCheck={false}
      className={cn(
        "day-well block min-h-[4.5rem] w-full resize-y px-3.5 py-2.5 font-mono text-ui-footnote leading-5 text-foreground outline-none placeholder:text-muted focus-visible:ring-2 focus-visible:ring-accent/30",
        className,
      )}
      {...rest}
    />
  );
}

/** DayMore 是列表底下那条「再看几个」的整行按钮。 */
export function DayMore(props: ButtonHTMLAttributes<HTMLButtonElement>) {
  const { className, children, ...rest } = props;
  return (
    <button type="button" className={cn("mt-2 w-full rounded-full py-2 text-ui-footnote text-muted hover:bg-lift hover:text-foreground", className)} {...rest}>
      {children}
    </button>
  );
}
