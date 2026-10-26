// 本文件拥有确认对话框的遮罩、面板和底部按钮。不拥有确认之后做什么。调用方传入标题、说明和动作。

import type { ComponentProps, ReactNode } from "react";
import * as AlertDialogPrimitive from "@radix-ui/react-alert-dialog";
import { cn } from "../../lib/utils";
import { Button } from "./button";

// AlertDialog 转发对话框根。
export const AlertDialog = AlertDialogPrimitive.Root;
// AlertDialogTrigger 转发打开按钮。
export const AlertDialogTrigger = AlertDialogPrimitive.Trigger;

// AlertDialogContent 挂上遮罩，面板宽度不超过 420px。
export function AlertDialogContent({ className, ...props }: ComponentProps<typeof AlertDialogPrimitive.Content>) {
  return (
    <AlertDialogPrimitive.Portal>
      <AlertDialogPrimitive.Overlay className="overlay-scrim fixed inset-0 z-50" />
      <AlertDialogPrimitive.Content
        className={cn(
          "dialog-sheet fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-popover p-5 shadow-[var(--shadow-popover)] focus:outline-none",
          className,
        )}
        {...props}
      />
    </AlertDialogPrimitive.Portal>
  );
}

// AlertDialogTitle 转发标题。
export const AlertDialogTitle = AlertDialogPrimitive.Title;
// AlertDialogDescription 转发说明。
export const AlertDialogDescription = AlertDialogPrimitive.Description;

// AlertDialogFooter 把动作排进 dialog-actions。
export function AlertDialogFooter({ children }: { children: ReactNode }) {
  return <div className="dialog-actions">{children}</div>;
}

// AlertDialogCancel 用 lift 按钮关闭，不执行确认。
export function AlertDialogCancel({ children, ...props }: ComponentProps<typeof AlertDialogPrimitive.Cancel>) {
  return (
    <AlertDialogPrimitive.Cancel asChild {...props}>
      <Button variant="lift">{children}</Button>
    </AlertDialogPrimitive.Cancel>
  );
}

// AlertDialogAction 在 danger 为真时用危险按钮，否则用 default。
export function AlertDialogAction({ children, danger, ...props }: ComponentProps<typeof AlertDialogPrimitive.Action> & { danger?: boolean }) {
  return (
    <AlertDialogPrimitive.Action asChild {...props}>
      <Button variant={danger ? "danger" : "default"}>{children}</Button>
    </AlertDialogPrimitive.Action>
  );
}
