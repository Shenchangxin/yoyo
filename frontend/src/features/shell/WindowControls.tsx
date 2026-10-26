// 本文件拥有最小化、最大化和关闭三个标题按钮。不拥有 macOS 的系统红绿灯，那些由系统画。PageHeader 把它放在右侧。

import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Minus, Square, X } from "lucide-react";
import { chrome, isMac, showCaptionButtons } from "../../lib/chrome";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";

// WindowControls 在 macOS 或不该显示标题按钮时不渲染。三个按钮分别调用 chrome.minimise、toggleMaximise 和 close。
export function WindowControls() {
  const copy = useCopy();
  if (isMac() || !showCaptionButtons()) return null;
  return (
    <div className="ml-1 flex items-center gap-0.5 pr-0.5">
      <Cap aria-label={copy.window.minimise} title={copy.window.minimise} onClick={() => chrome.minimise()}>
        <Minus className="size-3.5" />
      </Cap>
      <Cap aria-label={copy.window.maximise} title={copy.window.maximise} onClick={() => chrome.toggleMaximise()}>
        <Square className="size-3" />
      </Cap>
      <Cap aria-label={copy.window.close} title={copy.window.close} onClick={() => chrome.close()} danger>
        <X className="size-3.5" />
      </Cap>
    </div>
  );
}

/** Cap danger 为真时悬停用危险色。type 固定为 button。children 作为子节点渲染。props 展开到 button。 */
function Cap({
  children,
  danger,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { danger?: boolean; children?: ReactNode }) {
  return (
    <button
      type="button"
      className={cn(
        "grid size-8 place-items-center rounded-lg text-muted transition-[background-color,color,transform] duration-200 ease-[var(--ease-out)] hover:bg-lift hover:text-foreground active:scale-[0.96]",
        danger && "hover:bg-danger hover:text-white",
      )}
      {...props}
    >
      {children}
    </button>
  );
}
