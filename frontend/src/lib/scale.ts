export function applyUiScale(scale?: number) {
  const s = Number.isFinite(scale) && (scale || 0) > 0 ? Number(scale) : 1;
  document.documentElement.style.setProperty("--ui-scale", String(s));
  try {
    localStorage.setItem("yoyo-ui-scale", String(s));
  } catch {
    /* ignore */
  }
}

export function readUiScale(): number {
  try {
    const n = Number(localStorage.getItem("yoyo-ui-scale"));
    if (Number.isFinite(n) && n > 0) return n;
  } catch {
    /* ignore */
  }
  return 1;
}
