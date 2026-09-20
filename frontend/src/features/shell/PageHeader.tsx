import type { ReactNode } from "react";
import { chrome } from "../../lib/chrome";
import { cn } from "../../lib/utils";
import { WindowControls } from "./WindowControls";

export function PageHeader(props: {
  left?: ReactNode;
  title?: ReactNode;
  right?: ReactNode;
  macPad?: boolean;
  className?: string;
}) {
  return (
    <header
      className={cn(
        "chrome drag flex h-12 min-w-0 shrink-0 items-center gap-2 border-b border-border/30 px-3",
        props.macPad && "pl-[76px]",
        props.className,
      )}
      onDoubleClick={() => chrome.toggleMaximise()}
    >
      {props.left ? <div className="no-drag flex shrink-0 items-center gap-2">{props.left}</div> : null}
      <div className="flex min-w-0 flex-1 items-center">{props.title}</div>
      <div className="no-drag flex h-full shrink-0 items-center gap-1">
        {props.right}
        <WindowControls />
      </div>
    </header>
  );
}
