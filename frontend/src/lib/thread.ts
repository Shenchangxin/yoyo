/**
 * One reading column for the whole conversation. Transcript prose, user
 * bubbles, process rows and the composer all share this measure so their
 * left edges align; the scroll container itself stays pane-wide so the
 * scrollbar hugs the pane edge (macOS grammar), not the text.
 */
export const THREAD_MAX = "max-w-[min(46rem,100%)]";
export const THREAD_COL = `mx-auto w-full min-w-0 ${THREAD_MAX}`;
export const THREAD_GUTTER = "px-6 sm:px-8";
export const THREAD_GUTTER_COMPACT = "px-3";

export const LONG_THREAD_TURNS = 50;
