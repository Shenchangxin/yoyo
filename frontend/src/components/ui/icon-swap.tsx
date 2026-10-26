// 本文件拥有两个字形切换时的淡入和缩放。不拥有字形代表的状态。调用方用 on 选定当前字形。

import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { DURATION_FAST, motionTransition, useMotionReduced } from "../../lib/motion";
import { cn } from "../../lib/utils";

/** IconSwap 按 on 在 off 和 live 之间交叉淡入并缩放到 0.72。动的是这次切换，停下来就不再动。减少动效时不做入场和退场。className 传给 cn。 */
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
