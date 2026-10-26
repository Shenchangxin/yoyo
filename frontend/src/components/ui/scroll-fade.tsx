// 本文件拥有一条可滚动视口和纵向滚动条。不拥有滚动内容。调用方把要滚的子节点放进来。

import type { ComponentProps } from "react";
import * as ScrollArea from "@radix-ui/react-scroll-area";
import { cn } from "../../lib/utils";

// ScrollFade 把子节点放进带 scroll-fade 的视口，并画出纵向拇指。
export function ScrollFade({ className, children, ...props }: ComponentProps<typeof ScrollArea.Root>) {
  return (
    <ScrollArea.Root className={cn("min-h-0 overflow-hidden", className)} {...props}>
      <ScrollArea.Viewport className="scroll-fade h-full w-full">{children}</ScrollArea.Viewport>
      <ScrollArea.Scrollbar
        orientation="vertical"
        className="flex w-2 touch-none p-0.5 select-none"
      >
        <ScrollArea.Thumb className="relative flex-1 rounded-full bg-lift" />
      </ScrollArea.Scrollbar>
    </ScrollArea.Root>
  );
}
