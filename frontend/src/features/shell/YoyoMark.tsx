import { cn } from "../../lib/utils";

/** Four-cell CAS mark — content-addressed, not a sparkle. */
export function YoyoMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={cn("size-3.5 text-foreground", className)} aria-hidden>
      <rect x="1.4" y="1.4" width="5.6" height="5.6" rx="1.1" fill="currentColor" />
      <rect x="9" y="1.4" width="5.6" height="5.6" rx="1.1" fill="currentColor" opacity="0.32" />
      <rect x="1.4" y="9" width="5.6" height="5.6" rx="1.1" fill="currentColor" opacity="0.32" />
      <rect x="9" y="9" width="5.6" height="5.6" rx="1.1" fill="currentColor" opacity="0.72" />
    </svg>
  );
}
