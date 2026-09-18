import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { ChevronRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import { DURATION, EASE } from "../../lib/motion";
import { cn } from "../../lib/utils";

/** One width vocabulary for every inline control, so rows line up down the page. */
export const CONTROL_SM = "h-8 w-[132px] text-[12.5px]";
export const CONTROL_MD = "h-8 w-[208px] text-[12.5px]";
export const CONTROL_LG = "h-8 w-[268px] text-[12.5px]";

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
    <header className="mb-7 flex items-start justify-between gap-6">
      <div className="min-w-0">
        <h1 className="text-[21px] font-semibold tracking-[-0.02em] text-foreground">{title}</h1>
        {description ? (
          <p className="mt-1.5 max-w-[46ch] text-[12.5px] leading-[1.55] text-muted">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2 pt-1">{actions}</div> : null}
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
}: {
  id: string;
  title?: string;
  description?: string;
  footnote?: string;
  actions?: ReactNode;
  children: ReactNode;
  /** Drop the card shell when the section owns its own surface (code panels, grids). */
  bare?: boolean;
}) {
  return (
    <section className="mb-7 scroll-mt-6" data-setting-section-highlight-target={id}>
      {title ? (
        <div className="mb-2 flex items-end justify-between gap-4 px-1">
          <h2 id={id} className="text-[12.5px] font-semibold tracking-[-0.005em] text-foreground/75">
            {title}
          </h2>
          {actions ? <div className="flex items-center gap-1.5">{actions}</div> : null}
        </div>
      ) : null}
      {description ? <p className="mb-2 px-1 text-[12px] leading-[1.55] text-muted">{description}</p> : null}
      <div
        id={title ? undefined : id}
        data-setting-section-id={id}
        className={cn(
          "overflow-hidden rounded-[12px]",
          !bare && "surface-inset border border-border bg-card",
        )}
      >
        {children}
      </div>
      {footnote ? <p className="mt-2 px-1 text-[11.5px] leading-[1.55] text-muted">{footnote}</p> : null}
    </section>
  );
}

/**
 * Inset-grouped row: label column on the left, one control on the right, and a
 * hairline that starts where the label starts instead of running edge to edge.
 */
export function SettingRow({
  title,
  description,
  children,
  border = true,
  stack,
  align = "center",
}: {
  title?: string;
  description?: string;
  children?: ReactNode;
  border?: boolean;
  /** Put the control on its own line — for textareas, grids, and long inputs. */
  stack?: boolean;
  align?: "center" | "start";
}) {
  return (
    <div
      className={cn(
        "relative px-4",
        stack ? "py-3.5" : "flex min-h-[46px] gap-6 py-2.5",
        !stack && (align === "start" ? "items-start" : "items-center"),
      )}
    >
      {title ? (
        <div className={cn("min-w-0", stack ? "mb-2.5" : "flex-1 basis-0")}>
          <div className="text-[13px] font-medium leading-[1.4] text-foreground">{title}</div>
          {description ? (
            <div className="mt-0.5 text-[11.5px] leading-[1.5] text-muted">{description}</div>
          ) : null}
        </div>
      ) : null}
      {children ? (
        <div className={cn(stack ? "w-full" : "flex shrink-0 items-center justify-end gap-2")}>{children}</div>
      ) : null}
      {border ? <span aria-hidden className="pointer-events-none absolute bottom-0 left-4 right-0 h-px bg-border" /> : null}
    </div>
  );
}

/** Read-only status line — label left, value right, no control. */
export function SettingValueRow({
  title,
  value,
  mono,
  border = true,
}: {
  title: string;
  value: ReactNode;
  mono?: boolean;
  border?: boolean;
}) {
  return (
    <SettingRow title={title} border={border}>
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
  border = true,
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
      className="relative flex min-h-[46px] w-full items-center gap-4 px-4 py-2.5 text-left transition-colors hover:bg-lift/55"
    >
      <span className="min-w-0 flex-1">
        <span className={cn("block text-[13px] font-medium leading-[1.4]", danger ? "text-danger" : "text-foreground")}>
          {title}
        </span>
        {description ? <span className="mt-0.5 block text-[11.5px] leading-[1.5] text-muted">{description}</span> : null}
      </span>
      {trailing ? <span className="flex shrink-0 items-center text-[12px] text-muted">{trailing}</span> : null}
      {chevron ? <ChevronRight className="size-4 shrink-0 text-muted/70" aria-hidden /> : null}
      {border ? <span aria-hidden className="pointer-events-none absolute bottom-0 left-4 right-0 h-px bg-border" /> : null}
    </button>
  );
}

/** Empty copy inside a card, kept on the same rhythm as a row. */
export function SettingEmpty({ children, border }: { children: ReactNode; border?: boolean }) {
  return (
    <div className="relative">
      <p className="px-4 py-6 text-center text-[12.5px] text-muted">{children}</p>
      {border ? <span aria-hidden className="pointer-events-none absolute bottom-0 left-4 right-0 h-px bg-border" /> : null}
    </div>
  );
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
      {actions ? (
        <div className="relative flex items-center justify-end gap-2 px-3 py-2">
          {actions}
          <span aria-hidden className="pointer-events-none absolute bottom-0 left-4 right-0 h-px bg-border" />
        </div>
      ) : null}
      <pre
        className={cn(
          "overflow-auto whitespace-pre-wrap break-words px-4 py-3 font-mono text-[11px] leading-[1.65] text-muted",
          className,
        )}
      >
        {children}
      </pre>
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
          <div className="flex items-center gap-2 rounded-[12px] border border-border bg-popover/90 py-2 pl-4 pr-2 shadow-[var(--shadow-popover)] backdrop-blur-xl">
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
