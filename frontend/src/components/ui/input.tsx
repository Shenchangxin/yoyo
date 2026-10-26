// 本文件拥有单行和多行输入的边框与焦点环。不拥有输入值。调用方传入 value 和 onChange。

import { forwardRef, type InputHTMLAttributes, type TextareaHTMLAttributes } from "react";
import { cn } from "../../lib/utils";

/**
 * Input 默认关掉自动完成和拼写检查，并带上 no-drag，避免窗口拖拽吃掉输入。
 * className 传给 cn。
 * autoComplete 写到 input 的 autoComplete。
 * spellCheck 写到 input 的 spellCheck。
 * props 展开到 input。
 * ref 写到 input 的 ref。
 */
export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(function Input(
  { className, autoComplete = "off", spellCheck = false, ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      autoComplete={autoComplete}
      spellCheck={spellCheck}
      className={cn(
        "no-drag h-8 w-full rounded-lg border border-border bg-background px-2.5 text-[13px] text-foreground placeholder:text-muted/70",
        "outline-none transition-[border-color,box-shadow,background-color] duration-200 ease-[var(--ease-out)]",
        "focus-visible:border-accent/45 focus-visible:ring-2 focus-visible:ring-accent/20",
        "disabled:cursor-default disabled:opacity-40",
        className,
      )}
      {...props}
    />
  );
});

/** Textarea 去掉缩放和焦点轮廓，高度由调用方的行数或 class 决定。className 传给 cn。props 展开到 textarea。ref 写到 textarea 的 ref。 */
export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(function Textarea(
  { className, ...props },
  ref,
) {
  return (
    <textarea
      ref={ref}
      className={cn(
        "no-drag w-full resize-none bg-transparent text-[15px] leading-6 text-foreground placeholder:text-muted",
        "outline-none focus:outline-none focus-visible:outline-none",
        className,
      )}
      {...props}
    />
  );
});
