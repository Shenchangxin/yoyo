// 本文件拥有居中浮起的对话框面板和遮罩。不拥有面板里的业务。调用方用 open 控制开关，用 className 定尺寸。

import type { ComponentProps, ReactNode } from "react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { cn } from "../../lib/utils";

// Dialog 转发对话框根，开关状态留在 Radix。
export const Dialog = DialogPrimitive.Root;
// DialogClose 转发关闭按钮。
export const DialogClose = DialogPrimitive.Close;
// DialogTitle 转发标题。
export const DialogTitle = DialogPrimitive.Title;
// DialogDescription 转发说明。
export const DialogDescription = DialogPrimitive.Description;

/**
 * DialogContent 用 inset-0 加 m-auto 居中，不占用 transform，开合动画交给 .dialog-pop。
 * 默认宽至多 960px、高至多 680px，四周至少留 1.5rem。className 传给 cn，可以覆盖尺寸。
 * children 作为子节点渲染。props 展开到 DialogPrimitive.Content。
 */
export function DialogContent({ className, children, ...props }: ComponentProps<typeof DialogPrimitive.Content> & { children?: ReactNode }) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="overlay-scrim dialog-scrim fixed inset-0 z-50" />
      <DialogPrimitive.Content
        className={cn(
          "dialog-pop fixed inset-0 z-50 m-auto flex h-[min(680px,calc(100dvh-3rem))] w-[min(960px,calc(100%-3rem))] flex-col overflow-hidden overscroll-contain rounded-xl border border-border bg-popover shadow-[var(--shadow-popover)] outline-none",
          className,
        )}
        {...props}
      >
        {children}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}
