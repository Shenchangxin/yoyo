// 本文件拥有从边缘滑出的面板。不拥有面板里的业务。调用方用 open 控制，并决定 side。

import type { ComponentProps, ReactNode } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { cn } from "../../lib/utils";

/** Sheet 转发对话框根，开关状态留在 Radix。props 展开到 Dialog.Root。 */
export function Sheet(props: ComponentProps<typeof Dialog.Root>) {
  return <Dialog.Root {...props} />;
}

// SheetTrigger 转发打开按钮。
export const SheetTrigger = Dialog.Trigger;
// SheetClose 转发关闭按钮。
export const SheetClose = Dialog.Close;
// SheetTitle 转发标题。
export const SheetTitle = Dialog.Title;
// SheetDescription 转发说明。
export const SheetDescription = Dialog.Description;

/**
 * SheetContent 默认从右侧滑出并带遮罩。overlay 为假时不画遮罩。left 宽至多 280px，right 至多 380px，bottom 高度至多 520px 或 70vh。
 * side 与 right、left、bottom 比较是否相等。
 * className 传给 cn。
 * children 作为子节点渲染。
 * props 展开到 Dialog.Content。
 */
export function SheetContent({
  side = "right",
  className,
  children,
  overlay = true,
  ...props
}: ComponentProps<typeof Dialog.Content> & { side?: "right" | "left" | "bottom"; overlay?: boolean; children?: ReactNode }) {
  return (
    <Dialog.Portal>
      {overlay ? <Dialog.Overlay className="overlay-scrim fixed inset-0 z-40" /> : null}
      <Dialog.Content
        className={cn(
          "fixed z-50 overscroll-contain bg-sidebar outline-none",
          side === "right" && "inset-y-0 right-0 w-[min(380px,92%)] border-l border-border",
          side === "left" && "inset-y-0 left-0 w-[min(280px,88%)] border-r border-border",
          side === "bottom" && "inset-x-0 bottom-0 h-[min(70vh,520px)] border-t border-border",
          className,
        )}
        {...props}
      >
        {children}
      </Dialog.Content>
    </Dialog.Portal>
  );
}
