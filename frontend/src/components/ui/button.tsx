import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { forwardRef, type ButtonHTMLAttributes } from "react";
import { cn } from "../../lib/utils";

const buttonVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-2 rounded-lg text-[13px] font-medium transition-[color,background-color,opacity,transform,box-shadow] duration-200 ease-[var(--ease-out)] disabled:pointer-events-none disabled:opacity-40 aria-busy:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/50 active:scale-[0.98] [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-foreground text-background hover:bg-foreground/92",
        accent: "bg-accent text-accent-fg hover:opacity-[0.92]",
        ghost: "bg-transparent text-muted hover:bg-lift hover:text-foreground",
        outline: "border border-border bg-transparent text-foreground hover:bg-lift",
        danger: "bg-danger/12 text-danger hover:bg-danger/18",
        lift: "bg-lift text-foreground hover:bg-lift/80",
      },
      size: {
        default: "h-8 px-3",
        sm: "h-7 px-2.5 text-xs",
        lg: "h-9 px-4",
        icon: "size-8 rounded-lg",
        send: "size-8 rounded-full",
      },
    },
    defaultVariants: { variant: "default", size: "default" },
  },
);

type Props = ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & { asChild?: boolean };

export const Button = forwardRef<HTMLButtonElement, Props>(function Button(
  { className, variant, size, asChild, type = "button", ...props },
  ref,
) {
  const Comp = asChild ? Slot : "button";
  return <Comp ref={ref} type={type} className={cn(buttonVariants({ variant, size }), className)} {...props} />;
});
