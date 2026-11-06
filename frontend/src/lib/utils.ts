import { clsx, type ClassValue } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

/**
 * 拥有：把条件类名合并成一条，后写的 Tailwind 类覆盖冲突的前者。
 * 不拥有：具体组件的样式。
 */

/** UI_TEXT 是 styles.css 里 text-ui-* 的字号刻度。不登记时 text-ui-body 和 text-muted 会被当成同一组颜色，合并时丢掉一个。 */
const UI_TEXT = ["ui-badge", "ui-caption", "ui-footnote", "ui-body", "ui-body-l", "ui-title-s", "ui-title", "ui-display"];

const twMerge = extendTailwindMerge({ extend: { theme: { text: UI_TEXT } } });

/** cn 合并类名。inputs 传给 clsx。 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
