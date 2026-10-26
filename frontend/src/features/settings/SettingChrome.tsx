/**
 * 拥有：设置页的分区、行、空态和保存条。
 * 不拥有：具体字段的值。只排版，不调用 api。
 */
import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { ChevronRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import { cn } from "../../lib/utils";

/** CONTROL_SM 是组内控件的统一宽度，同一分区的字段左边和尺度对齐。 */
export const CONTROL_SM = "h-8 w-full text-[13px]";
/** CONTROL_MD 沿用同一套控件宽度。 */
export const CONTROL_MD = "h-8 w-full text-[13px]";
/** CONTROL_LG 沿用同一套控件宽度。 */
export const CONTROL_LG = "h-8 w-full text-[13px]";

/** SettingsGroupLabel 画一组上面的小标题。children 作为子节点渲染。 */
export function SettingsGroupLabel({ children }: { children: ReactNode }) {
  return (
    <div className="mb-2 text-[11px] font-medium text-muted">
      {children}
    </div>
  );
}

/**
 * SettingSection 把标题、说明和行收成一块。
 * id 作为分支结果，写到 h2 的 id。
 * title 为假时用 headerActions，作为子节点渲染。
 * description 为真时才继续，作为子节点渲染。
 * footnote 为真时才继续，作为子节点渲染。
 * actions 为真时才继续，作为子节点渲染。
 * headerActions 为真时才继续，作为后备值，作为子节点渲染。
 * children 作为子节点渲染。
 * bare 为真时才继续。
 * compact 为假时走另一支。
 * tight 为真时才继续。
 */
export function SettingSection({
  id,
  title,
  description,
  footnote,
  actions,
  headerActions,
  children,
  bare,
  compact,
  tight,
}: {
  id: string;
  title?: string;
  description?: string;
  footnote?: string;
  actions?: ReactNode;
  headerActions?: ReactNode;
  children: ReactNode;
  /** 这一段自己有表面时去掉卡片壳，例如代码面板和网格。 */
  bare?: boolean;
  /** 放在折叠里时不再加一段间距。 */
  compact?: boolean;
  /** 标题、说明和行用更紧的节奏，生成目录用这个。 */
  tight?: boolean;
}) {
  const heading = title || headerActions ? (
    <div className="flex items-center gap-3">
      {title ? (
        <h2 id={id} className="min-w-0 flex-1 text-[12px] font-medium text-muted">
          {title}
        </h2>
      ) : (
        <div className="min-w-0 flex-1" />
      )}
      {headerActions ? <div className="flex shrink-0 items-center gap-2">{headerActions}</div> : null}
    </div>
  ) : null;
  const desc = description ? (
    <p className={cn("text-[12px] text-muted", tight ? "leading-[1.4]" : "leading-[1.55]")}>{description}</p>
  ) : null;
  const header = heading || desc ? (
    <div className={cn("flex flex-col", tight ? "gap-1" : "gap-[var(--space-item)]")}>
      {heading}
      {desc}
    </div>
  ) : null;
  const stackGap = tight ? "gap-2" : "gap-[var(--space-group)]";
  return (
    <section className={cn(!compact && (tight ? "mb-4" : "mb-6"), "w-full min-w-0 scroll-mt-6")} data-setting-section-highlight-target={id}>
      {bare ? (
        <>
          {header}
          <div id={title ? undefined : id} data-setting-section-id={id} className={cn("flex w-full min-w-0 flex-col", stackGap, tight && header && "mt-2")}>
            {children}
          </div>
          {actions ? <div className="mt-[var(--space-item)] flex items-center justify-end gap-2">{actions}</div> : null}
          {footnote ? <p className="mt-[var(--space-item)] text-[12px] leading-[1.45] text-muted">{footnote}</p> : null}
        </>
      ) : (
        <>
          <div
            id={title ? undefined : id}
            data-setting-section-id={id}
            className={cn("flex w-full min-w-0 flex-col rounded-[12px] bg-card shadow-[var(--shadow-card)]", stackGap, tight ? "px-5 py-2.5" : "px-5 py-3")}
          >
            {header}
            {children}
            {actions ? <div className="flex items-center justify-end gap-2">{actions}</div> : null}
          </div>
          {footnote ? <p className="mt-1.5 text-[12px] leading-[1.45] text-muted">{footnote}</p> : null}
        </>
      )}
    </section>
  );
}

/**
 * SettingRow 是表单行，默认左标签右控件；stack 时控件占满整行。
 * title 为真时才继续，作为后备值，作为子节点渲染。
 * description 为真时才继续，作为子节点渲染。
 * children 为真时才继续，作为子节点渲染。
 * _border 函数体内没有读取。
 * _align 函数体内没有读取。
 * list 为假时用 title。
 */
export function SettingRow({
  title,
  description,
  children,
  border: _border = true,
  stack = false,
  align: _align = "center",
  list = false,
}: {
  title?: string;
  description?: string;
  children?: ReactNode;
  border?: boolean;
  stack?: boolean;
  align?: "center" | "start";
  list?: boolean;
}) {
  if (!stack && (list || title)) {
    return (
      <div className="flex min-h-10 w-full min-w-0 items-center gap-6">
        {title ? (
          <div className="min-w-0 flex-[1_1_16rem]">
            <div className="text-[13px] font-medium leading-[1.4] text-foreground">{title}</div>
            {description ? <div className="mt-0.5 text-[12px] leading-[1.45] text-muted">{description}</div> : null}
          </div>
        ) : null}
        {children ? (
          <div className="flex min-w-0 flex-[1_1_18rem] items-center justify-end gap-2">{children}</div>
        ) : null}
      </div>
    );
  }
  return (
    <div className="flex w-full min-w-0 flex-col items-stretch gap-[var(--space-item)]">
      {title ? (
        <div className="min-w-0">
          <div className="text-[13px] font-medium leading-[1.4] text-foreground">{title}</div>
          {description ? <div className="mt-1 text-[13px] leading-[1.5] text-muted">{description}</div> : null}
        </div>
      ) : null}
      {children ? <div className="flex w-full min-w-0 flex-wrap items-center justify-start gap-2">{children}</div> : null}
    </div>
  );
}

/** SettingValueRow 是只读状态行，左标签右值，没有控件。title 写到 SettingRow 的 title。value 作为子节点渲染。mono 为真时才继续。_border 函数体内没有读取。 */
export function SettingValueRow({
  title,
  value,
  mono,
  border: _border = true,
}: {
  title: string;
  value: ReactNode;
  mono?: boolean;
  border?: boolean;
}) {
  return (
    <SettingRow title={title} list>
      <span className={cn("text-[13px] text-muted", mono && "font-mono tabular-nums")}>{value}</span>
    </SettingRow>
  );
}

/**
 * SettingActionRow 是会导航或触发动作的一行，导航时带箭头。
 * title 作为子节点渲染。
 * description 为真时才继续，作为子节点渲染。
 * onClick 写到 button 的 onClick。
 * trailing 为真时才继续，作为子节点渲染。
 * _border 函数体内没有读取。
 * danger 为真时才继续。
 * chevron 为真时才继续。
 */
export function SettingActionRow({
  title,
  description,
  onClick,
  trailing,
  border: _border = true,
  danger,
  chevron = true,
}: {
  title: string;
  description?: string;
  onClick: () => void;
  trailing?: ReactNode;
  border?: boolean;
  danger?: boolean;
  /** 就地提交时关掉，例如添加和运行；导航才带箭头。 */
  chevron?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
          className="flex min-h-11 w-full items-center gap-4 px-1 text-left transition-colors hover:bg-lift/70"
    >
      <span className="min-w-0 flex-1">
        <span className={cn("block text-[13px] font-medium leading-[1.4]", danger ? "text-danger" : "text-foreground")}>
          {title}
        </span>
        {description ? <span className="mt-0.5 block text-[12px] leading-[1.5] text-muted">{description}</span> : null}
      </span>
      {trailing ? <span className="flex shrink-0 items-center text-[12px] text-muted">{trailing}</span> : null}
      {chevron ? <ChevronRight className="size-4 shrink-0 text-muted/70" aria-hidden /> : null}
    </button>
  );
}

/** SettingEmpty 在卡片里放一行空态，节奏和普通行一样。children 作为子节点渲染。compact 为真时才继续。_border 函数体内没有读取。 */
export function SettingEmpty({ children, compact, border: _border }: { children: ReactNode; compact?: boolean; border?: boolean }) {
  return <p className={cn(compact ? "py-0.5" : "py-6", "text-left text-[13px] text-muted")}>{children}</p>;
}

/** SettingCodePanel 是等宽输出，可以带工具条，用于日记、诊断和 JSON。children 作为子节点渲染。actions 为真时才继续，作为子节点渲染。className 传给 cn。 */
export function SettingCodePanel({
  children,
  actions,
  className,
}: {
  children: ReactNode;
  actions?: ReactNode;
  className?: string;
}) {
  return (
    <div>
      <pre
        className={cn(
          "overflow-auto whitespace-pre-wrap break-words py-1 font-mono text-[12px] leading-[1.65] text-muted",
          className,
        )}
      >
        {children}
      </pre>
      {actions ? <div className="mt-[var(--space-group)] flex items-center justify-end gap-2">{actions}</div> : null}
    </div>
  );
}

/** SettingSegmented 是一组互斥选项。id 拼进字符串。value 与 o.value 比较是否相等。options 做映射。onChange 被调用，并传入 o.value。ariaLabel 写到 div 的 aria-label。 */
export function SettingSegmented<T extends string>({
  id,
  value,
  options,
  onChange,
  ariaLabel,
}: {
  id: string;
  value: T;
  options: readonly { value: T; label: string }[];
  onChange: (next: T) => void;
  ariaLabel: string;
}) {
  const reduced = useMotionReduced();
  const pill = motionTransition(reduced, DURATION);
  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      className="inline-flex items-center rounded-[9px] border border-border bg-background p-[2px]"
    >
      {options.map((o) => {
        const active = o.value === value;
        return (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => onChange(o.value)}
            className={cn(
              "relative rounded-[7px] px-2.5 py-[5px] text-[12px] font-medium transition-colors",
              active ? "text-foreground" : "text-muted hover:text-foreground",
            )}
          >
            {active ? (
              <motion.span
                layoutId={`segmented-${id}`}
                className="absolute inset-0 rounded-[7px] bg-lift"
                transition={pill}
              />
            ) : null}
            <span className="relative z-10">{o.label}</span>
          </button>
        );
      })}
    </div>
  );
}

/**
 * SettingsSaveBar 是需要明确保存时浮出的条，用于密钥和预算这类表面。
 * open 为真时才继续。
 * onSave 写到 Button 的 onClick。
 * onDiscard 为真时才继续，写到 Button 的 onClick。
 * label 为假时改用后备值，作为子节点渲染。
 * saveLabel 为假时改用后备值，作为子节点渲染。
 */
export function SettingsSaveBar({
  open,
  onSave,
  onDiscard,
  label,
  saveLabel,
}: {
  open: boolean;
  onSave: () => void;
  onDiscard?: () => void;
  label?: string;
  saveLabel?: string;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  return (
    <AnimatePresence>
      {open ? (
        <motion.div
          initial={reduced ? false : { opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          exit={reduced ? undefined : { opacity: 0, y: 12 }}
          transition={motionTransition(reduced, DURATION)}
          className="sticky bottom-3 z-20 mt-8"
        >
          <div className="flex items-center gap-2 rounded-2xl border border-border bg-popover py-2 pl-4 pr-2 shadow-[var(--shadow-popover)]">
            <span className="flex-1 truncate text-[13px] text-muted">{label || copy.settings.unsaved}</span>
            {onDiscard ? (
              <Button size="sm" variant="ghost" onClick={onDiscard}>
                {copy.settings.discard}
              </Button>
            ) : null}
            <Button size="sm" onClick={onSave}>
              {saveLabel || copy.settings.saveChanges}
            </Button>
          </div>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}
