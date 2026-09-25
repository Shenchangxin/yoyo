import { canvasThemes, type CanvasColorTheme, type CanvasTheme } from "@yingce/lib/canvas-theme";

/** Hosted canvas HUD reads Yoyo CSS variables via canvas-theme token fallbacks. */
export function hostedCanvasTheme(mode: CanvasColorTheme): CanvasTheme {
  return canvasThemes[mode];
}

export { canvasThemes };
