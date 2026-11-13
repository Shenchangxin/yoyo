/**
 * 拥有：设置页的版式零件：分区（标题在上，行收在一张卡里，行间细线）、行、空态、代码面板、分段、键帽和保存条。
 * 不拥有：具体字段的值。只排版，不调用 api。卡片里行的上下留白和细线在 styles.css 的 .setting-card。
 */
import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { ChevronRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import { displayShortcut } from "../../lib/keymap";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import { cn } from "../../lib/utils";

/** CONTROL_SM 是组内控件的统一宽度，同一分区的字段左边和尺度对齐。 */
export const CONTROL_SM = "h-8 w-full text-ui-body";
/** CONTROL_MD 沿用同一套控件宽度。 */
export const CONTROL_MD = "h-8 w-full text-ui-body";
/** CONTROL_LG 沿用同一套控件宽度。 */
export const CONTROL_LG = "h-8 w-full text-ui-body";

/** SettingsGroupLabel 画几个分区上面的小标题。children 作为子节点渲染。 */
export function SettingsGroupLabel({ children }: { children: ReactNode }) {
  return (
    <div className="mb-3 px-1 text-ui-caption font-medium text-muted">
      {children}
    </div>
  );
}

/**
 * SettingSection 是一个分区：标题和说明在卡片上面，行收在一张卡里，脚注在卡片下面。
 * id 写到 h2 的 id；没有标题时写到卡片上，深链靠它滚动。
 * title 为假时只留 headerActions 那一行。
 * description 为真时画在标题下面。
 * footnote 为真时画在卡片下面。
 * actions 为真时在卡片末尾另起一行，靠右。
 * headerActions 为真时和标题同一行，靠右。
 * children 是卡片里的行；每个直接子节点算一行，行间有细线。
 * bare 为真时不要卡片，子节点自己有表面，例如主题网格。
 * compact 为真时去掉分区下面的间距，放在折叠里用。
 * tight 为真时行更矮，生成目录用这个。
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
  /** 行用更紧的节奏，生成目录用这个。 */
  tight?: boolean;
}) {
  const heading = title || headerActions ? (
    <div className="flex min-h-7 items-center gap-3">
      {title ? (
        <h2 id={id} className="min-w-0 flex-1 text-ui-body-l font-semibold text-foreground">
          {title}
        </h2>
      ) : (
        <div className="min-w-0 flex-1" />
      )}
      {headerActions ? <div className="flex shrink-0 items-center gap-2">{headerActions}</div> : null}
    </div>
  ) : null;
  const desc = description ? (
    <p className="text-ui-body leading-[1.55] text-muted">{description}</p>
  ) : null;
  const header = heading || desc ? (
    <div className="mb-2.5 flex flex-col gap-0.5 px-1">
      {heading}
      {desc}
    </div>
  ) : null;
  return (
    <section className={cn(!compact && (tight ? "mb-6" : "mb-8"), "w-full min-w-0 scroll-mt-6")} data-setting-section-highlight-target={id}>
      {header}
      {bare ? (
        <>
          <div id={title ? undefined : id} data-setting-section-id={id} className={cn("flex w-full min-w-0 flex-col", tight ? "gap-2" : "gap-[var(--space-group)]")}>
            {children}
          </div>
          {actions ? <div className="mt-[var(--space-item)] flex items-center justify-end gap-2">{actions}</div> : null}
        </>
      ) : (
        <div
          id={title ? undefined : id}
          data-setting-section-id={id}
          className={cn("setting-card w-full min-w-0 rounded-lg bg-card px-4", tight && "setting-card-tight")}
        >
          {children}
          {actions ? <div className="flex items-center justify-end gap-2">{actions}</div> : null}
        </div>
      )}
      {footnote ? <p className="mt-2 px-1 text-ui-footnote leading-[1.5] text-muted">{footnote}</p> : null}
    </section>
  );
}

/**
 * SettingRow 是表单行，默认左边标题和说明、右边控件；stack 时控件另起一行占满。
 * title 为真时画在左边。
 * description 为真时画在标题下面。
 * children 是控件。控件里有占满宽度的输入框时，右边固定成一列，同一分区的输入框左边对齐；开关、分段和按钮只占自己的宽度，说明能写得更长。
 * _border 函数体内没有读取，行间细线由卡片统一画。
 * stack 为真时标题在上、控件在下。
 * _align 函数体内没有读取。
 * list 为真且没有标题时，控件占满整行。
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
      <div className="flex min-h-8 w-full min-w-0 items-center gap-6">
        {title ? (
          <div className="min-w-0 flex-1">
            <div className="text-ui-body-l leading-[1.45] text-foreground">{title}</div>
            {description ? <div className="mt-0.5 text-ui-body leading-[1.5] text-muted">{description}</div> : null}
          </div>
        ) : null}
        {children ? (
          <div
            className={cn(
              "flex min-w-0 flex-wrap items-center justify-end gap-2",
              title ? "max-w-[62%] shrink-0 has-[>.w-full]:w-[min(21rem,50%)]" : "flex-1",
            )}
          >
            {children}
          </div>
        ) : null}
      </div>
    );
  }
  return (
    <div className="flex w-full min-w-0 flex-col items-stretch gap-2">
      {title ? (
        <div className="min-w-0">
          <div className="text-ui-body-l leading-[1.45] text-foreground">{title}</div>
          {description ? <div className="mt-0.5 text-ui-body leading-[1.5] text-muted">{description}</div> : null}
        </div>
      ) : null}
      {children ? <div className="flex w-full min-w-0 flex-wrap items-center justify-start gap-2">{children}</div> : null}
    </div>
  );
}

/** SettingValueRow 是只读状态行，左标签右值，没有控件。title 写到 SettingRow 的 title。value 作为子节点渲染。mono 为真时用等宽字。_border 函数体内没有读取。 */
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
      <span className={cn("min-w-0 break-all text-right text-muted", mono ? "font-mono text-ui-footnote tabular-nums" : "text-ui-body")}>{value}</span>
    </SettingRow>
  );
}

/**
 * SettingActionRow 是会导航或触发动作的一行，导航时带箭头。
 * title 作为子节点渲染。
 * description 为真时画在标题下面。
 * onClick 写到 button 的 onClick。
 * trailing 为真时画在右边。
 * _border 函数体内没有读取。
 * danger 为真时标题用危险色。
 * chevron 为真时带箭头。
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
      className="group flex min-h-8 w-full items-center gap-4 text-left"
    >
      <span className="min-w-0 flex-1">
        <span className={cn("block text-ui-body-l leading-[1.45]", danger ? "text-danger" : "text-foreground")}>
          {title}
        </span>
        {description ? <span className="mt-0.5 block text-ui-body leading-[1.5] text-muted">{description}</span> : null}
      </span>
      {trailing ? <span className="flex shrink-0 items-center text-ui-footnote text-muted transition-colors group-hover:text-foreground">{trailing}</span> : null}
      {chevron ? <ChevronRight className="size-4 shrink-0 text-muted/70 transition-[color,transform] duration-[var(--duration-fast)] group-hover:translate-x-0.5 group-hover:text-foreground" aria-hidden /> : null}
    </button>
  );
}

/** SettingEmpty 在卡片里放一行空态。children 作为子节点渲染。compact 为真时和普通行一样高，否则多留一点空。_border 函数体内没有读取。 */
export function SettingEmpty({ children, compact, border: _border }: { children: ReactNode; compact?: boolean; border?: boolean }) {
  return <p className={cn(!compact && "py-5", "text-left text-ui-body text-muted")}>{children}</p>;
}

/** SettingCodePanel 是等宽输出，可以带工具条，用于日记、诊断和 JSON。children 作为子节点渲染。actions 为真时画在下面，靠右。className 传给 cn。 */
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
          "prose-select overflow-auto whitespace-pre-wrap break-words rounded-md border border-border/70 bg-sidebar/60 px-3 py-2.5 font-mono text-ui-footnote leading-[1.65] text-muted",
          className,
        )}
      >
        {children}
      </pre>
      {actions ? <div className="mt-[var(--space-group)] flex items-center justify-end gap-2">{actions}</div> : null}
    </div>
  );
}

/** SettingSegmented 是一组互斥选项：一条浅槽，选中的那格是一块滑过去的底。id 拼进滑块的 layoutId。value 与 o.value 比较是否相等。options 做映射。onChange 被调用，并传入 o.value。ariaLabel 写到 div 的 aria-label。 */
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
      className="inline-flex h-8 shrink-0 items-center rounded-lg bg-lift p-0.5"
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
              "relative h-7 whitespace-nowrap rounded-md px-3 text-ui-footnote font-medium transition-colors",
              active ? "text-foreground" : "text-muted hover:text-foreground",
            )}
          >
            {active ? (
              <motion.span
                layoutId={`segmented-${id}`}
                className="absolute inset-0 rounded-md bg-background shadow-[var(--shadow-card)]"
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

/** shortcutKeys 把一条快捷键拆成一个个键帽上的字。Mac 上修饰键是符号，其他平台是 Ctrl、Shift、Alt。spec 交给 displayShortcut 的同一套写法逐段换算。 */
export function shortcutKeys(spec: string): string[] {
  return spec
    .split("+")
    .map((part) => part.trim())
    .filter(Boolean)
    .map((part) => displayShortcut(part));
}

/** Keycaps 把一条快捷键画成一排键帽。spec 传给 shortcutKeys。className 传给 cn。 */
export function Keycaps({ spec, className }: { spec: string; className?: string }) {
  return (
    <span className={cn("inline-flex items-center gap-1", className)}>
      {shortcutKeys(spec).map((key, i) => (
        <span
          key={`${key}-${i}`}
          className="inline-flex h-6 min-w-6 items-center justify-center rounded-sm border border-border bg-background px-1.5 text-ui-caption font-medium tabular-nums text-foreground/85"
        >
          {key}
        </span>
      ))}
    </span>
  );
}

/**
 * SettingsSaveBar 是需要明确保存时浮出的条，用于密钥和预算这类表面。
 * open 为真时才画。
 * onSave 写到 Button 的 onClick。
 * onDiscard 为真时多一个放弃按钮。
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
          className="sticky bottom-4 z-20 mt-8"
        >
          <div className="flex items-center gap-2 rounded-xl border border-border bg-popover py-2 pl-4 pr-2 shadow-[var(--shadow-popover)]">
            <span className="flex-1 truncate text-ui-body text-muted">{label || copy.settings.unsaved}</span>
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
