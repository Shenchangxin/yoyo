/**
 * 拥有：与设置导航胶囊相同的缓动，以及减弱动效时把过渡时长收成 0。
 * 不拥有：具体组件怎么编排动画。
 */

import { useReducedMotion, type Transition } from "motion/react";

/** EASE 与 `--ease-out` 以及设置导航胶囊相同的曲线。 */
export const EASE = [0.32, 0.72, 0, 1] as const;
/** DURATION 默认过渡秒数。 */
export const DURATION = 0.22;
/** DURATION_FAST 更短的过渡秒数。 */
export const DURATION_FAST = 0.14;
/** DURATION_SHELL 壳层过渡秒数。 */
export const DURATION_SHELL = 0.24;

/** useMotionReduced 系统是否要求减弱动效。 */
export function useMotionReduced(): boolean {
  return useReducedMotion() === true;
}

/** motionTransition 减弱动效时时长为 0，否则用给定秒数和 EASE。 */
export function motionTransition(reduced: boolean, duration = DURATION): Transition {
  if (reduced) return { duration: 0 };
  return { duration, ease: EASE };
}
