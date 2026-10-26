// 本文件拥有表单标签的字号和颜色。不拥有控件本身。调用方用 htmlFor 把它指到输入框。

import type { ComponentProps } from "react";
import * as LabelPrimitive from "@radix-ui/react-label";
import { cn } from "../../lib/utils";

/** Label 转发 Radix 标签，并加上小号弱色。className 传给 cn。props 展开到 LabelPrimitive.Root。 */
export function Label({ className, ...props }: ComponentProps<typeof LabelPrimitive.Root>) {
  return (
    <LabelPrimitive.Root
      className={cn("text-xs text-muted", className)}
      {...props}
    />
  );
}
