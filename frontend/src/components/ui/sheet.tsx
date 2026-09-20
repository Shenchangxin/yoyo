import type { ComponentProps, ReactNode } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { cn } from "../../lib/utils";

export function Sheet(props: ComponentProps<typeof Dialog.Root>) {
  return <Dialog.Root {...props} />;
}

export const SheetTrigger = Dialog.Trigger;
export const SheetClose = Dialog.Close;
export const SheetTitle = Dialog.Title;
export const SheetDescription = Dialog.Description;

export function SheetContent({
  side = "right",
  className,
  children,
  overlay = true,
  ...props
}: ComponentProps<typeof Dialog.Content> & { side?: "right" | "left" | "bottom"; overlay?: boolean; children?: ReactNode }) {
  return (
    <Dialog.Portal>
      {overlay ? <Dialog.Overlay className="fixed inset-0 z-40 bg-background/40" /> : null}
      <Dialog.Content
        className={cn(
          "fixed z-50 overscroll-contain border-border bg-sidebar shadow-[var(--shadow-popover)] outline-none",
          side === "right" && "top-2 right-2 bottom-2 w-[min(380px,92%)] rounded-[10px] border",
          side === "left" && "top-2 left-2 bottom-2 w-[min(280px,88%)] rounded-[10px] border",
          side === "bottom" && "right-2 bottom-2 left-2 h-[min(70vh,520px)] rounded-[10px] border",
          className,
        )}
        {...props}
      >
        {children}
      </Dialog.Content>
    </Dialog.Portal>
  );
}
