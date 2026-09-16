import type { ComponentProps } from "react";
import * as ScrollArea from "@radix-ui/react-scroll-area";
import { cn } from "../../lib/utils";

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
