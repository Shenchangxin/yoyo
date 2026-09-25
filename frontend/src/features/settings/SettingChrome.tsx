import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { ChevronRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import { DURATION, EASE } from "../../lib/motion";
import { cn } from "../../lib/utils";

/** One control width inside a group, so fields in the same section share a left edge and a measure. */
export const CONTROL_SM = "h-8 w-full text-[12.5px]";
export const CONTROL_MD = "h-8 w-full text-[12.5px]";
export const CONTROL_LG = "h-8 w-full text-[12.5px]";

export function SettingsPageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <header className="mb-[var(--space-section)]">
      <h1 className="text-[21px] font-semibold tracking-[-0.03em] text-pretty text-foreground">{title}</h1>
      {description ? (
        <p className="mt-[var(--space-item)] max-w-[46ch] text-[13.5px] leading-[1.6] text-muted">{description}</p>
      ) : null}
      {actions ? <div className="mt-[var(--space-group)] flex items-center justify-end gap-2">{actions}</div> : null}
    </header>
  );
}

export function SettingSection({
  id,
  title,
  description,
  footnote,
  actions,
  children,
  bare,
  compact,
}: {
  id: string;
  title?: string;
  description?: string;
  footnote?: string;
  actions?: ReactNode;
  children: ReactNode;
  /** Drop the card shell when the section owns its own surface (code panels, grids). */
  bare?: boolean;
  /** Inside a disclosure: no extra module gap. */
  compact?: boolean;
}) {
  return (
    <section className={cn(!compact && "mb-[var(--space-section)]", "scroll-mt-6")} data-setting-section-highlight-target={id}>
      {title ? (
        <h2 id={id} className="mb-[var(--space-item)] text-[13px] font-semibold tracking-[-0.01em] text-foreground">
          {title}
        </h2>
      ) : null}
      {description ? <p className="mb-[var(--space-group)] text-[12.5px] leading-[1.55] text-muted">{description}</p> : null}
      <div
        id={title ? undefined : id}
        data-setting-section-id={id}
        className={cn(
          "flex flex-col gap-[var(--space-group)]",
          !bare && "rounded-[12px] bg-card px-4 py-1 shadow-[var(--shadow-card)]",
        )}
      >
        {children}
      </div>
      {actions ? <div className="mt-[var(--space-group)] flex items-center justify-end gap-2">{actions}</div> : null}
      {footnote ? <p className="mt-[var(--space-item)] text-[12px] leading-[1.55] text-muted">{footnote}</p> : null}
    </section>
  );
}

/**
 * Form field: label above the control, left aligned, full measure of the group.
 * Pass `list` for a list row (title leading, trailing control, one row height).
 */
export function SettingRow({
  title,
  description,
  children,
  border: _border = true,
  stack: _stack,
  align: _align = "center",
  list = false,
}: {
  title?: string;
  description?: string;
  children?: ReactNode;
  border?: boolean;
  /** Kept so existing call sites stay valid. Form rows always stack. */
  stack?: boolean;
  align?: "center" | "start";
  /** List row: primary text first, auxiliary control on the right, vertically centered. */
  list?: boolean;
}) {
  if (list) {
    return (
      <div className="flex min-h-10 items-center gap-4">
        {title ? (
          <div className="min-w-0 flex-1">
            <div className="truncate text-[13px] font-medium leading-[1.4] text-foreground">{title}</div>
            {description ? <div className="mt-0.5 truncate text-[12px] leading-[1.45] text-muted">{description}</div> : null}
          </div>
        ) : null}
        {children ? <div className="flex shrink-0 items-center gap-2">{children}</div> : null}
      </div>
    );
  }
  return (
    <div className="flex flex-col items-stretch gap-[var(--space-item)]">
      {title ? (
        <div className="min-w-0">
          <div className="text-[13px] font-medium leading-[1.4] text-foreground">{title}</div>
          {description ? <div className="mt-1 text-[12.5px] leading-[1.5] text-muted">{description}</div> : null}
        </div>
      ) : null}
      {children ? <div className="flex w-full flex-wrap items-center justify-start gap-2">{children}</div> : null}
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
      <span className={cn("text-[12.5px] text-muted", mono && "font-mono tabular-nums")}>{value}</span>
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
        {description ? <span className="mt-0.5 block text-[11.5px] leading-[1.5] text-muted">{description}</span> : null}
      </span>
      {trailing ? <span className="flex shrink-0 items-center text-[12px] text-muted">{trailing}</span> : null}
      {chevron ? <ChevronRight className="size-4 shrink-0 text-muted/70" aria-hidden /> : null}
    </button>
  );
}

/** Empty copy inside a card, kept on the same rhythm as a row. */
export function SettingEmpty({ children, border: _border }: { children: ReactNode; border?: boolean }) {
  return <p className="py-6 text-left text-[13px] text-muted">{children}</p>;
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
          "overflow-auto whitespace-pre-wrap break-words py-1 font-mono text-[11.5px] leading-[1.65] text-muted",
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
                transition={{ duration: DURATION, ease: EASE }}
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
  return (
    <AnimatePresence>
      {open ? (
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 12 }}
          transition={{ duration: DURATION, ease: EASE }}
          className="sticky bottom-3 z-20 mt-8"
        >
          <div className="flex items-center gap-2 rounded-2xl border border-border bg-popover py-2 pl-4 pr-2 shadow-[var(--shadow-popover)]">
            <span className="flex-1 truncate text-[12.5px] text-muted">{label || copy.settings.unsaved}</span>
            {onDiscard ? (
              <Button size="sm" variant="ghost" onClick={onDiscard}>
                {copy.settings.discard}
              </Button>
            ) : null}
            <Button size="sm" variant="accent" onClick={onSave}>
              {saveLabel || copy.settings.saveChanges}
            </Button>
          </div>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}
