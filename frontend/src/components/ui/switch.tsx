// 本文件拥有开关的尺寸和开合颜色。不拥有开关代表的设置。调用方传入 checked 和 onCheckedChange。

import type { ComponentProps } from "react";
import * as SwitchPrimitive from "@radix-ui/react-switch";
import { cn } from "../../lib/utils";

/** Switch 在未选中时用 lift，选中时用 accent，拇指跟随 data-state 平移。className 传给 cn。props 展开到 SwitchPrimitive.Root。 */
export function Switch({ className, ...props }: ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      className={cn(
        "peer yoyo-switch inline-flex h-[18px] w-[32px] shrink-0 cursor-pointer items-center rounded-full border border-transparent bg-lift transition-colors duration-200 ease-[var(--ease-out)] disabled:cursor-not-allowed disabled:opacity-40 data-[state=checked]:bg-accent",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb className="yoyo-switch-thumb pointer-events-none block size-3.5 translate-x-[2px] rounded-full bg-foreground shadow-sm transition-transform duration-200 ease-[var(--ease-out)] data-[state=checked]:translate-x-[16px] data-[state=checked]:bg-accent-fg" />
    </SwitchPrimitive.Root>
  );
}
