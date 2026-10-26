// 本文件拥有一枚截断的胶囊标签。不拥有标签文案从哪来。调用方传入子节点。

import type { HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

/** Badge 把文字收进最大 220px 的圆角胶囊，超出截断。className 传给 cn。props 展开到 span。 */
export function Badge({ className, ...props }: HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        "inline-flex max-w-[220px] items-center truncate rounded-full bg-lift px-2 py-0.5 text-[11px] tabular-nums text-muted",
        className,
      )}
      {...props}
    />
  );
}
