import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { DURATION_FAST, motionTransition, useMotionReduced } from "../../lib/motion";
import { cn } from "../../lib/utils";

/** Cross-fade + scale between two glyphs. Motion is the state change, then it stops. */
export function IconSwap({
  on,
  off,
  live,
  className,
}: {
  on: boolean;
  off: ReactNode;
  live: ReactNode;
  className?: string;
}) {
  const reduced = useMotionReduced();
  const swap = motionTransition(reduced, DURATION_FAST);
  return (
    <span className={cn("relative inline-grid size-[1em] place-items-center", className)} aria-hidden>
      <AnimatePresence initial={false}>
        <motion.span
          key={on ? "on" : "off"}
          className="absolute inset-0 grid place-items-center"
          initial={reduced ? false : { opacity: 0, scale: 0.72 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={reduced ? undefined : { opacity: 0, scale: 0.72 }}
          transition={swap}
          aria-hidden
        >
          {on ? live : off}
        </motion.span>
      </AnimatePresence>
    </span>
  );
}
