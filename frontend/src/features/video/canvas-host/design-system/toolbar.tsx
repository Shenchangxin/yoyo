/**
 * 本文件拥有画布工具条上的幽灵按钮、成组玻璃、填充按钮和危险按钮。
 * 不拥有业务动作。它们只是界面投影，不改服务端。
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";

import { cn } from "@yingce/lib/utils";

type GhostProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  active?: boolean;
  danger?: boolean;
};

/** GhostButton 是可标成选中或危险的幽灵按钮。 */
export function GhostButton({ className, active, danger, type = "button", ...props }: GhostProps) {
  return (
    <button
      type={type}
      className={cn("sg-ghost", active && "is-active", danger && "sg-danger", className)}
      {...props}
    />
  );
}

/** GlassGroup 把一组控件包进玻璃容器。 */
export function GlassGroup({ className, children }: { className?: string; children: ReactNode }) {
  return <div className={cn("sg-glass-group", className)}>{children}</div>;
}

/** FillButton 是填充按钮。 */
export function FillButton({ className, type = "button", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button type={type} className={cn("sg-fill", className)} {...props} />;
}

/** DangerButton 是危险按钮。 */
export function DangerButton({ className, type = "button", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button type={type} className={cn("sg-danger", className)} {...props} />;
}
