/** Windows-style steps so 16px chrome lands on whole device pixels at 100/125/150%. */
export const UI_SCALE_STEPS = [1, 1.25, 1.5] as const;
export type UiScale = (typeof UI_SCALE_STEPS)[number];

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

export function applyUiScale(scale?: number) {
  const s = snapUiScale(scale);
  document.documentElement.style.setProperty("--ui-scale", String(s));
  try {
    localStorage.setItem("yoyo-ui-scale", String(s));
  } catch {
    /* ignore */
  }
}

export function readUiScale(): UiScale {
  try {
    return snapUiScale(Number(localStorage.getItem("yoyo-ui-scale")));
  } catch {
    return 1;
  }
}
