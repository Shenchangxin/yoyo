import { cn } from "../../lib/utils";

type MarkProps = {
  className?: string;
  /** Tight crop for chrome. Page marks stay cropped, just a step larger. */
  compact?: boolean;
};

/** Lowercase yoyo wordmark in the UI face. */
export function YoyoMark({ className, compact }: MarkProps) {
  return (
    <span
      className={cn(
        "yoyo-mark inline-flex shrink-0 items-center font-semibold tracking-[-0.04em] text-foreground",
        compact ? "text-[15px] leading-none" : "text-[17px] leading-none",
        className,
      )}
    >
      yoyo
    </span>
  );
}

/** Empty still-life: a quiet stamp, not a poster. */
export function MarkWell({ className, markClassName }: { className?: string; markClassName?: string }) {
  return (
    <span className={cn("mark-well inline-flex", className)} aria-hidden>
      <span className="mark-well-core">
        <YoyoMark className={markClassName} />
      </span>
    </span>
  );
}
