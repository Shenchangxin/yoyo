import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { ChevronRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import { cn } from "../../lib/utils";

/** One control width inside a group, so fields in the same section share a left edge and a measure. */
export const CONTROL_SM = "h-8 w-full text-[13px]";
export const CONTROL_MD = "h-8 w-full text-[13px]";
export const CONTROL_LG = "h-8 w-full text-[13px]";

export function SettingsGroupLabel({ children }: { children: ReactNode }) {
  return (
    <div className="mb-2 text-[11px] font-medium text-muted">
      {children}
    </div>
  );
}

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
  /** Drop the card shell when the section owns its own surface (code panels, grids). */
  bare?: boolean;
  /** Inside a disclosure: no extra module gap. */
  compact?: boolean;
  /** Title, intro, and rows sit on a tighter rhythm (Generation catalog). */
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
 * Form field. Default is a full-width split: label left, control right.
 * Pass `stack` when the control should take the whole row (paths, text areas).
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

/** Read-only status line — label left, value right, no control. */
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

/** Row that navigates or fires an action — carries the macOS disclosure chevron. */
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
  /** Off for actions that commit in place (add, run) rather than navigate. */
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

/** Empty copy inside a card, kept on the same rhythm as a row. */
export function SettingEmpty({ children, compact, border: _border }: { children: ReactNode; compact?: boolean; border?: boolean }) {
  return <p className={cn(compact ? "py-0.5" : "py-6", "text-left text-[13px] text-muted")}>{children}</p>;
}

/** Monospace output block with an optional toolbar — journal, doctor, JSON. */
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

/** Floating bar for the surfaces that need an explicit Save (secrets, budget). */
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
