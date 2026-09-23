import { cn } from "../../lib/utils";
import wordmark from "../../assets/yoyo-wordmark.png";

type MarkProps = {
  className?: string;
  /** Tight crop for chrome. Page marks stay cropped, just a step larger. */
  compact?: boolean;
};

/** Custom YOYO wordmark — overlapping terracotta discs between two Ys. */
export function YoyoMark({ className, compact }: MarkProps) {
  return (
    <span
      className={cn(
        "relative inline-flex shrink-0 overflow-hidden",
        compact ? "h-5 w-[2.35rem] rounded-[4px]" : "h-8 w-[3.7rem] rounded-md",
        className,
      )}
    >
      <img
        src={wordmark}
        alt="Yoyo"
        draggable={false}
        className="h-full w-full origin-center scale-[1.42] object-cover"
      />
    </span>
  );
}

/** Empty still-life: a quiet stamp, not a poster. */
export function MarkWell({ className, markClassName }: { className?: string; markClassName?: string }) {
  return (
    <span className={cn("inline-flex", className)} aria-hidden>
      <YoyoMark className={cn("h-8 w-[3.7rem] rounded-md", markClassName)} />
    </span>
  );
}
