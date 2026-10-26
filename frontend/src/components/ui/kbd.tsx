// 本文件拥有快捷键的 kbd 外观。不拥有按键含义。调用方传入要显示的键名。

import type { HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

/** Kbd 套上 kbd class，其余属性原样交给 kbd。className 传给 cn。props 展开到 kbd。 */
export function Kbd({ className, ...props }: HTMLAttributes<HTMLElement>) {
  return <kbd className={cn("kbd", className)} {...props} />;
}
