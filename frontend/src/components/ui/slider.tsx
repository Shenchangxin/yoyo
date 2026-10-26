// 本文件拥有滑块的轨道、已选区间和拇指。不拥有数值的业务含义。调用方传入 value 和 onValueChange。

import type { ComponentProps } from "react";
import * as SliderPrimitive from "@radix-ui/react-slider";
import { cn } from "../../lib/utils";

/** Slider 拼好轨道和拇指，取值与步进仍交给 Radix。className 传给 cn。props 展开到 SliderPrimitive.Root。 */
export function Slider({ className, ...props }: ComponentProps<typeof SliderPrimitive.Root>) {
  return (
    <SliderPrimitive.Root className={cn("relative flex w-40 touch-none items-center select-none", className)} {...props}>
      <SliderPrimitive.Track className="relative h-1.5 w-full grow overflow-hidden rounded-full bg-lift">
        <SliderPrimitive.Range className="absolute h-full bg-accent" />
      </SliderPrimitive.Track>
      <SliderPrimitive.Thumb className="block size-4 rounded-full border border-border bg-foreground shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40" />
    </SliderPrimitive.Root>
  );
}
