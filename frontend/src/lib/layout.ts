/**
 * 拥有：壳层三栏宽度的读取、夹紧和写回。
 * 不拥有：拖拽分割条的手势。
 */

/** ShellLayout 轨道、主栏和检查器的百分比。 */
export type ShellLayout = { rail: number; main: number; inspect: number };

const KEY = "yoyo-layout-v1";
/** 审阅要放下一块差异。1440 宽的窗口上 30% 大约是 430 像素的块。 */
const FALLBACK: ShellLayout = { rail: 19, main: 51, inspect: 30 };

/** readLayout 从本地读出三栏。缺字段或损坏时用缺省。轨道夹在 14 到 34，主栏 30 到 80，检查器 18 到 42。 */
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
    /* 读失败时忽略 */
  }
  return FALLBACK;
}

/** writeLayout 与当前值合并、夹紧后写回，并返回合并结果。写入失败时仍返回合并值。 */
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
    /* 写入失败时忽略 */
  }
  return merged;
}

/** 把有限数字夹进闭区间。非有限时取下界。 */
function clamp(n: number, min: number, max: number) {
  if (!Number.isFinite(n)) return min;
  return Math.min(max, Math.max(min, n));
}
