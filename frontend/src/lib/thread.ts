/**
 * One conversation column shared by transcript and composer so left edges
 * align. It tracks the stage (`100%`) and only ceilings on very wide panes
 * (`--thread-measure`). The scroll container stays pane-wide so the
 * scrollbar hugs the pane edge, not the text.
 *
 * Gutters follow the stage container, not the viewport — review open/closed
 * must not jump padding as if the window resized.
 */
export const THREAD_MAX = "max-w-[min(100%,var(--thread-measure))]";
export const THREAD_COL = `mx-auto w-full min-w-0 ${THREAD_MAX}`;
export const THREAD_GUTTER = "px-4 @[36rem]:px-6 @[52rem]:px-8";
export const THREAD_GUTTER_COMPACT = "px-3";

export const LONG_THREAD_TURNS = 50;
