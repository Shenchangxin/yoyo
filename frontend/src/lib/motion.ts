import { useReducedMotion, type Transition } from "motion/react";

/** Same curve as `--ease-out` / the settings nav pill. */
export const EASE = [0.32, 0.72, 0, 1] as const;
export const DURATION = 0.22;
export const DURATION_FAST = 0.14;
export const DURATION_SHELL = 0.24;

export function useMotionReduced(): boolean {
  return useReducedMotion() === true;
}

export function motionTransition(reduced: boolean, duration = DURATION): Transition {
  if (reduced) return { duration: 0 };
  return { duration, ease: EASE };
}
