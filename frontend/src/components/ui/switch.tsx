// 本文件拥有开关的尺寸和开合颜色。不拥有开关代表的设置。调用方传入 checked 和 onCheckedChange。

import type { ComponentProps } from "react";
import * as SwitchPrimitive from "@radix-ui/react-switch";
import { cn } from "../../lib/utils";

/** Switch 未选中时是一条浅灰槽，选中时槽变成墨色；拇指在开合两侧都和槽反色，跟随 data-state 平移。className 传给 cn。props 展开到 SwitchPrimitive.Root。 */
export function Switch({ className, ...props }: ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      className={cn(
        "peer yoyo-switch inline-flex h-5 w-[34px] shrink-0 cursor-pointer items-center rounded-full border border-transparent bg-foreground/[0.16] transition-colors duration-200 ease-[var(--ease-out)] hover:bg-foreground/[0.22] disabled:cursor-not-allowed disabled:opacity-40 data-[state=checked]:bg-foreground data-[state=checked]:hover:bg-foreground",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb className="yoyo-switch-thumb pointer-events-none block size-4 translate-x-px rounded-full bg-card shadow-sm transition-transform duration-200 ease-[var(--ease-out)] data-[state=checked]:translate-x-[15px] data-[state=checked]:bg-background" />
    </SwitchPrimitive.Root>
  );
}
