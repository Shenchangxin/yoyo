/**
 * 拥有：界面缩放在三个 Windows 档位上的吸附，以及写到文档和本地存储。
 * 不拥有：设置页上的缩放控件。
 */

/** UI_SCALE_STEPS 三个档位，使 16px 界面在 100%、125%、150% 下落到整设备像素。 */
export const UI_SCALE_STEPS = [1, 1.25, 1.5] as const;
/** UiScale 允许的缩放值，只能是这三个档位之一。 */
export type UiScale = (typeof UI_SCALE_STEPS)[number];

/** snapUiScale 把任意正数吸到最近档位。非有限或非正数视为 1。 */
export function snapUiScale(scale?: number): UiScale {
  const n = Number(scale);
  if (!Number.isFinite(n) || n <= 0) return 1;
  let best: UiScale = 1;
  let dist = Infinity;
  for (const step of UI_SCALE_STEPS) {
    const d = Math.abs(step - n);
    if (d < dist) {
      dist = d;
      best = step;
    }
  }
  return best;
}

/** applyUiScale 把吸附后的值写到 --ui-scale，并尝试记入本地存储。 */
export function applyUiScale(scale?: number) {
  const s = snapUiScale(scale);
  document.documentElement.style.setProperty("--ui-scale", String(s));
  try {
    localStorage.setItem("yoyo-ui-scale", String(s));
  } catch {
    /* 写入失败时忽略 */
  }
}

/** readUiScale 读出上次保存的缩放并吸附。读失败时为 1。 */
export function readUiScale(): UiScale {
  try {
    return snapUiScale(Number(localStorage.getItem("yoyo-ui-scale")));
  } catch {
    return 1;
  }
}
