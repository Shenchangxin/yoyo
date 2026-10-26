// 本文件拥有下拉选择的触发器、弹出层和选项勾。不拥有选项列表从哪来。调用方传入 items 和 onValueChange。

import type { ComponentProps } from "react";
import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "../../lib/utils";

// Select 转发选择根。
export const Select = SelectPrimitive.Root;
// SelectValue 转发当前值的显示。
export const SelectValue = SelectPrimitive.Value;
// SelectGroup 转发分组。
export const SelectGroup = SelectPrimitive.Group;

// SelectTrigger 在占位时用弱色，并在右侧放一个向下的箭头。
export function SelectTrigger({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Trigger>) {
  return (
    <SelectPrimitive.Trigger
      className={cn(
        "no-drag inline-flex h-8 min-w-[8rem] items-center justify-between gap-2 rounded-lg border border-border bg-background px-2.5 text-[12px] text-foreground outline-none transition-[background-color,border-color] duration-200 ease-[var(--ease-out)] hover:bg-lift/70 disabled:opacity-40 data-[placeholder]:text-muted",
        className,
      )}
      {...props}
    >
      {children}
      <SelectPrimitive.Icon>
        <ChevronDown className="size-3.5 text-muted" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
  );
}

// SelectContent 用弹出层对齐触发器宽度，选项放在 Viewport 里。
export function SelectContent({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Content>) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Content
        position="popper"
        className={cn(
          "z-50 min-w-[var(--radix-select-trigger-width)] overflow-hidden rounded-xl border border-border bg-popover shadow-[var(--shadow-popover)]",
          className,
        )}
        {...props}
      >
        <SelectPrimitive.Viewport className="p-1">{children}</SelectPrimitive.Viewport>
      </SelectPrimitive.Content>
    </SelectPrimitive.Portal>
  );
}

// SelectLabel 用小号大写字标出一组选项。
export function SelectLabel({ className, ...props }: ComponentProps<typeof SelectPrimitive.Label>) {
  return (
    <SelectPrimitive.Label
      className={cn("px-2 py-1.5 text-[10px] font-medium uppercase tracking-[0.04em] text-muted", className)}
      {...props}
    />
  );
}

// SelectSeparator 在选项之间画一条横线。
export function SelectSeparator({ className, ...props }: ComponentProps<typeof SelectPrimitive.Separator>) {
  return <SelectPrimitive.Separator className={cn("-mx-1 my-1 h-px bg-border", className)} {...props} />;
}

// SelectItem 把文字放进 ItemText，选中时在右侧画勾。禁用时不接收指针。
export function SelectItem({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Item>) {
  return (
    <SelectPrimitive.Item
      className={cn(
        "relative flex cursor-pointer items-center rounded-lg py-1.5 pr-8 pl-2 text-[12px] outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-40 data-[highlighted]:bg-lift",
        className,
      )}
      {...props}
    >
      <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
      <SelectPrimitive.ItemIndicator className="absolute right-2">
        <Check className="size-3.5" />
      </SelectPrimitive.ItemIndicator>
    </SelectPrimitive.Item>
  );
}
