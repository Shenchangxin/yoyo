export type ShellLayout = { rail: number; main: number; inspect: number };

const KEY = "yoyo-layout-v1";
/** Review needs room for a diff; 30% of a 1440 window is ~430px of hunk. */
const FALLBACK: ShellLayout = { rail: 19, main: 51, inspect: 30 };

export function readLayout(): ShellLayout {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) || "");
    if (v && typeof v.rail === "number" && typeof v.main === "number") {
      return {
        rail: clamp(v.rail, 14, 34),
        main: clamp(v.main, 30, 80),
        inspect: clamp(v.inspect ?? FALLBACK.inspect, 18, 42),
      };
    }
  } catch {
    /* ignore */
  }
  return FALLBACK;
}

export function writeLayout(next: Partial<ShellLayout>) {
  const cur = readLayout();
  const merged: ShellLayout = {
    rail: clamp(next.rail ?? cur.rail, 14, 34),
    main: clamp(next.main ?? cur.main, 30, 80),
    inspect: clamp(next.inspect ?? cur.inspect, 18, 42),
  };
  try {
    localStorage.setItem(KEY, JSON.stringify(merged));
  } catch {
    /* ignore */
  }
  return merged;
}

function clamp(n: number, min: number, max: number) {
  if (!Number.isFinite(n)) return min;
  return Math.min(max, Math.max(min, n));
}
