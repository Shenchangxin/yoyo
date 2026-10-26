/**
 * 本文件拥有画布上的岛、菜单、气泡、对话框和工作台表面。
 * 不拥有里面的业务。它们只是界面投影，不改服务端。
 */
import type { CSSProperties, HTMLAttributes, ReactNode } from "react";

import { AppModal } from "@yingce/components/ui/product/app-modal/app-modal";
import { cn } from "@yingce/lib/utils";

type SurfaceProps = HTMLAttributes<HTMLDivElement> & { children: ReactNode };

/** Island 是一块岛表面。className 拼进岛的类名。props 摊到外层 div。 */
export function Island({ className, ...props }: SurfaceProps) {
  return <div className={cn("sg-island", className)} {...props} />;
}

/** MenuSurface 是带菜单角色的表面。className 拼进菜单类名。props 摊到外层 div。 */
export function MenuSurface({ className, ...props }: SurfaceProps) {
  return <div role="menu" className={cn("sg-menu p-1", className)} {...props} />;
}

/** PopoverSurface 是气泡表面。className 拼进气泡类名。props 摊到外层 div。 */
export function PopoverSurface({ className, ...props }: SurfaceProps) {
  return <div className={cn("sg-popover p-3", className)} {...props} />;
}

/** DialogSurface 包一层对话框，默认宽度 480。open 传给 AppModal 的 open。title 传给 AppModal 的 title。footer 传给 AppModal 的 footer。onClose 传给 AppModal 的 onCancel。width 传给 AppModal 的 width。children 放进对话框。 */
export function DialogSurface({
  open,
  title,
  footer,
  onClose,
  width = 480,
  children,
}: {
  open: boolean;
  title?: ReactNode;
  footer?: ReactNode;
  onClose: () => void;
  width?: number | string;
  children: ReactNode;
}) {
  return (
    <AppModal
      open={open}
      title={title}
      footer={footer}
      onCancel={onClose}
      width={width}
      centered
      rootClassName="sg-dialog-root"
      className="sg-dialog"
    >
      {children}
    </AppModal>
  );
}

/** WorkbenchSurface 包一层没有页脚的工作台对话框。open 传给 AppModal 的 open。title 传给 AppModal 的 title。onClose 传给 AppModal 的 onCancel。children 放进对话框。width 传给 AppModal 的 width。 */
export function WorkbenchSurface({
  open,
  title,
  onClose,
  children,
  width = "min(1120px, calc(100vw - 48px))",
}: {
  open: boolean;
  title?: ReactNode;
  onClose: () => void;
  children: ReactNode;
  width?: number | string;
}) {
  return (
    <AppModal
      open={open}
      title={title}
      footer={null}
      onCancel={onClose}
      width={width}
      centered
      flush
      rootClassName="sg-workbench-root"
      className="sg-workbench"
    >
      {children}
    </AppModal>
  );
}

/** AttachedPanel 是贴在触发器旁边的面板。className 拼进面板类名。style 传给外层 div。children 放进面板。 */
export function AttachedPanel({ className, style, children }: { className?: string; style?: CSSProperties; children: ReactNode }) {
  return (
    <div className={cn("sg-popover overflow-hidden", className)} style={style}>
      {children}
    </div>
  );
}

/** MenuItem 是一行菜单项，可以带图标、快捷键和危险态。icon 有内容时放进 16 像素的格子。label 显示成一行，过长截断。shortcut 有内容时显示成快捷键。danger 为真时加上 sg-danger。disabled 传给按钮的 disabled。active 为真时加上 is-active。onClick 传给按钮的 onClick。 */
export function MenuItem({
  icon,
  label,
  shortcut,
  danger,
  disabled,
  active,
  onClick,
}: {
  icon?: ReactNode;
  label: string;
  shortcut?: string;
  danger?: boolean;
  disabled?: boolean;
  active?: boolean;
  onClick?: () => void;
}) {
  return (
    <button type="button" role="menuitem" className={cn("sg-menu-item", danger && "sg-danger", active && "is-active")} disabled={disabled} onClick={onClick}>
      {icon ? <span className="grid size-4 shrink-0 place-items-center [&_svg]:size-4">{icon}</span> : null}
      <span className="min-w-0 truncate">{label}</span>
      {shortcut ? <span className="sg-kbd" aria-hidden>{shortcut}</span> : null}
    </button>
  );
}
