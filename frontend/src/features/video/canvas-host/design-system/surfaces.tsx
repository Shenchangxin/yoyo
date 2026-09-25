import type { CSSProperties, HTMLAttributes, ReactNode } from "react";

import { AppModal } from "@yingce/components/ui/product/app-modal/app-modal";
import { cn } from "@yingce/lib/utils";

type SurfaceProps = HTMLAttributes<HTMLDivElement> & { children: ReactNode };

export function Island({ className, ...props }: SurfaceProps) {
  return <div className={cn("sg-island", className)} {...props} />;
}

export function MenuSurface({ className, ...props }: SurfaceProps) {
  return <div role="menu" className={cn("sg-menu p-1", className)} {...props} />;
}

export function PopoverSurface({ className, ...props }: SurfaceProps) {
  return <div className={cn("sg-popover p-3", className)} {...props} />;
}

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

export function AttachedPanel({ className, style, children }: { className?: string; style?: CSSProperties; children: ReactNode }) {
  return (
    <div className={cn("sg-popover overflow-hidden", className)} style={style}>
      {children}
    </div>
  );
}

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
