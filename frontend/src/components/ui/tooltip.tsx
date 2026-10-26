// 本文件拥有提示气泡的延迟和定位。不拥有提示文案的含义。main 挂上 TooltipProvider，各控件用 Tooltip。

import type { ComponentProps, ReactNode } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { cn } from "../../lib/utils";

// TooltipProvider 把悬停延迟默认设为 400 毫秒。
export function TooltipProvider({ children, delayDuration = 400 }: { children: ReactNode; delayDuration?: number }) {
  return <TooltipPrimitive.Provider delayDuration={delayDuration}>{children}</TooltipPrimitive.Provider>;
}

// Tooltip 在 content 为空时只渲染 children，不挂气泡。默认出现在下方，间距 6。
export function Tooltip({
  content,
  children,
  side = "bottom",
  className,
}: {
  content: ReactNode;
  children: ReactNode;
  side?: ComponentProps<typeof TooltipPrimitive.Content>["side"];
  className?: string;
}) {
  if (!content) return <>{children}</>;
  return (
    <TooltipPrimitive.Root>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side={side}
          sideOffset={6}
          className={cn(
            "z-50 max-w-xs overflow-visible rounded-lg border border-border bg-popover text-[12px] text-foreground shadow-[var(--shadow-popover)]",
            className,
          )}
        >
          <span className="yoyo-tooltip-in px-2.5 py-1.5">{content}</span>
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );
}
