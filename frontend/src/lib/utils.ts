import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * 拥有：把条件类名合并成一条，后写的 Tailwind 类覆盖冲突的前者。
 * 不拥有：具体组件的样式。
 */

/** cn 合并类名。 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
