import { forwardRef, type InputHTMLAttributes, type TextareaHTMLAttributes } from "react";
import { cn } from "../../lib/utils";

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(function Input(
  { className, ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      className={cn(
        "no-drag h-8 w-full rounded-md border border-border bg-background px-2.5 text-[13px] text-foreground placeholder:text-muted/70",
        "outline-none transition-colors focus-visible:border-accent/40 focus-visible:ring-1 focus-visible:ring-accent/20",
        "disabled:cursor-default disabled:opacity-40",
        className,
      )}
      {...props}
    />
  );
});

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
