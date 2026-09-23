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
