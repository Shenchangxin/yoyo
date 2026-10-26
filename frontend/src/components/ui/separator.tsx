// 本文件拥有一条分隔线。不拥有它分开的两块内容。调用方决定方向。

import type { ComponentProps } from "react";
import * as SeparatorPrimitive from "@radix-ui/react-separator";
import { cn } from "../../lib/utils";

/**
 * Separator 默认横向、decorative 为真。纵向时改成全高的 1px 宽。
 * className 传给 cn。
 * orientation 与 horizontal 比较是否相等，写到 SeparatorPrimitive.Root 的 orientation。
 * props 展开到 SeparatorPrimitive.Root。
 */
export function Separator({ className, orientation = "horizontal", decorative = true, ...props }: ComponentProps<typeof SeparatorPrimitive.Root>) {
  return (
    <SeparatorPrimitive.Root
      decorative={decorative}
      orientation={orientation}
      className={cn(
        "shrink-0 bg-border",
        orientation === "horizontal" ? "h-px w-full" : "h-full w-px",
        className,
      )}
      {...props}
    />
  );
}
