// 本文件拥有标签条的外观。不拥有每个标签页里的内容。调用方用 value 选定当前页。

import type { ComponentProps } from "react";
import * as TabsPrimitive from "@radix-ui/react-tabs";
import { cn } from "../../lib/utils";

// Tabs 转发标签根。
export const Tabs = TabsPrimitive.Root;

// TabsList 把触发器排成一条带内边距的横条。
export function TabsList({ className, ...props }: ComponentProps<typeof TabsPrimitive.List>) {
  return (
    <TabsPrimitive.List
      className={cn("flex gap-1 rounded-lg bg-lift p-0.5", className)}
      {...props}
    />
  );
}

// TabsTrigger 在 data-state=active 时换上面板底和前景色。
export function TabsTrigger({ className, ...props }: ComponentProps<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      className={cn(
        "rounded-md px-2 py-1 text-[11px] text-muted transition-colors data-[state=active]:bg-panel data-[state=active]:text-foreground",
        className,
      )}
      {...props}
    />
  );
}

// TabsContent 占满剩余高度，并去掉焦点轮廓。
export function TabsContent({ className, ...props }: ComponentProps<typeof TabsPrimitive.Content>) {
  return <TabsPrimitive.Content className={cn("min-h-0 flex-1 outline-none", className)} {...props} />;
}
