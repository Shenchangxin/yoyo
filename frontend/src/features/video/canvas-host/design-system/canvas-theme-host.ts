/**
 * 本文件拥有托管画布要读的主题入口。不拥有令牌表。
 * 它只做转发，不改服务端。
 */
import { canvasThemes, type CanvasColorTheme, type CanvasTheme } from "@yingce/lib/canvas-theme";

/** hostedCanvasTheme 按明暗模式返回对应的画布主题。mode 用作 canvasThemes 的键。 */
export function hostedCanvasTheme(mode: CanvasColorTheme): CanvasTheme {
  return canvasThemes[mode];
}

/** canvasThemes 原样转出。 */
export { canvasThemes };
