// 本文件拥有页面顶栏的左右槽、标题和窗口按钮。不拥有标题的文案，也不实现最小化。App 传入 left、title、right 和 macPad。

import type { ReactNode } from "react";
import { browserSurface, chrome } from "../../lib/chrome";
import { cn } from "../../lib/utils";
import { WindowControls } from "./WindowControls";

/** PageHeader 高 12。macPad 为真时左侧留出 76px。双击时，若不是浏览器面，调用 chrome.toggleMaximise。右侧始终挂 WindowControls。props 读取 macPad、className、left、title、right。 */
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
        "glass-chrome chrome drag relative z-20 flex h-12 min-h-12 min-w-0 shrink-0 items-center gap-2 px-3",
        props.macPad && "pl-[76px]",
        props.className,
      )}
      onDoubleClick={() => {
        if (!browserSurface()) chrome.toggleMaximise();
      }}
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
