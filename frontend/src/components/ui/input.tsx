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
        "no-drag h-9 w-full rounded-lg border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted",
        "outline-none focus-visible:border-foreground/25 focus-visible:ring-1 focus-visible:ring-foreground/15",
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
