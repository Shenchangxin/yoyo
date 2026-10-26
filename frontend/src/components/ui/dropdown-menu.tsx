// 本文件拥有下拉菜单的面板、条目和子菜单。不拥有菜单项被点之后做什么。调用方传入条目和 onSelect。

import type { ComponentProps } from "react";
import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import { ChevronRight } from "lucide-react";
import { cn } from "../../lib/utils";

// DropdownMenu 转发菜单根。
export const DropdownMenu = DropdownMenuPrimitive.Root;
// DropdownMenuTrigger 转发打开按钮。
export const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;
// DropdownMenuGroup 转发分组。
export const DropdownMenuGroup = DropdownMenuPrimitive.Group;
// DropdownMenuPortal 转发传送门。
export const DropdownMenuPortal = DropdownMenuPrimitive.Portal;

/**
 * DropdownMenuContent 把面板送进传送门，与触发器默认隔开 6。
 * className 传给 cn。
 * sideOffset 写到 DropdownMenuPrimitive.Content 的 sideOffset。
 * props 展开到 DropdownMenuPrimitive.Content。
 */
export function DropdownMenuContent({ className, sideOffset = 6, ...props }: ComponentProps<typeof DropdownMenuPrimitive.Content>) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        sideOffset={sideOffset}
        className={cn(
          "z-50 min-w-[160px] overflow-hidden rounded-xl border border-border bg-popover p-1.5 text-[13px] text-foreground shadow-[var(--shadow-popover)]",
          className,
        )}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  );
}

/** DropdownMenuItem 在高亮时铺 lift，禁用时不接收指针。className 传给 cn。props 展开到 DropdownMenuPrimitive.Item。 */
export function DropdownMenuItem({ className, ...props }: ComponentProps<typeof DropdownMenuPrimitive.Item>) {
  return (
    <DropdownMenuPrimitive.Item
      className={cn(
        "flex cursor-pointer items-center gap-2 rounded-lg px-2.5 py-1.5 outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-40 data-[highlighted]:bg-lift",
        className,
      )}
      {...props}
    />
  );
}

/** DropdownMenuSeparator 在条目之间画一条横线。className 传给 cn。props 展开到 DropdownMenuPrimitive.Separator。 */
export function DropdownMenuSeparator({ className, ...props }: ComponentProps<typeof DropdownMenuPrimitive.Separator>) {
  return <DropdownMenuPrimitive.Separator className={cn("-mx-1 my-1 h-px bg-border", className)} {...props} />;
}

/** DropdownMenuLabel 用小号弱色标出一组条目，本身不可选。className 传给 cn。props 展开到 DropdownMenuPrimitive.Label。 */
export function DropdownMenuLabel({ className, ...props }: ComponentProps<typeof DropdownMenuPrimitive.Label>) {
  return <DropdownMenuPrimitive.Label className={cn("px-2.5 py-1.5 text-[11px] text-muted", className)} {...props} />;
}

// DropdownMenuSub 转发子菜单根。
export const DropdownMenuSub = DropdownMenuPrimitive.Sub;

/** DropdownMenuSubTrigger 在右侧放一个向右的箭头，打开时铺 lift。className 传给 cn。children 作为子节点渲染。props 展开到 DropdownMenuPrimitive.SubTrigger。 */
export function DropdownMenuSubTrigger({ className, children, ...props }: ComponentProps<typeof DropdownMenuPrimitive.SubTrigger>) {
  return (
    <DropdownMenuPrimitive.SubTrigger
      className={cn(
        "flex cursor-pointer items-center gap-2 rounded-lg px-2.5 py-1.5 outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-40 data-[highlighted]:bg-lift data-[state=open]:bg-lift",
        className,
      )}
      {...props}
    >
      {children}
      <ChevronRight className="ml-auto size-3.5 opacity-60" aria-hidden />
    </DropdownMenuPrimitive.SubTrigger>
  );
}

/** DropdownMenuSubContent 把子面板送进传送门，最小宽度 160px。className 传给 cn。props 展开到 DropdownMenuPrimitive.SubContent。 */
export function DropdownMenuSubContent({ className, ...props }: ComponentProps<typeof DropdownMenuPrimitive.SubContent>) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.SubContent
        className={cn(
          "z-50 min-w-[160px] overflow-hidden rounded-xl border border-border bg-popover p-1.5 text-[13px] text-foreground shadow-[var(--shadow-popover)]",
          className,
        )}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  );
}
