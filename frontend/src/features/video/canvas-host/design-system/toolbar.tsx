import type { ButtonHTMLAttributes, ReactNode } from "react";

import { cn } from "@yingce/lib/utils";

type GhostProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  active?: boolean;
  danger?: boolean;
};

export function GhostButton({ className, active, danger, type = "button", ...props }: GhostProps) {
  return (
    <button
      type={type}
      className={cn("sg-ghost", active && "is-active", danger && "sg-danger", className)}
      {...props}
    />
  );
}

export function GlassGroup({ className, children }: { className?: string; children: ReactNode }) {
  return <div className={cn("sg-glass-group", className)}>{children}</div>;
}

export function FillButton({ className, type = "button", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button type={type} className={cn("sg-fill", className)} {...props} />;
}

export function DangerButton({ className, type = "button", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button type={type} className={cn("sg-danger", className)} {...props} />;
}
