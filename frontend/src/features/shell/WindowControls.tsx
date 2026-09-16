import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Minus, Square, X } from "lucide-react";
import { chrome, isMac, showCaptionButtons } from "../../lib/chrome";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";

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

function Cap({
  children,
  danger,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { danger?: boolean; children?: ReactNode }) {
  return (
    <button
      type="button"
      className={cn(
        "grid size-8 place-items-center rounded-md text-muted transition-colors duration-150 hover:bg-lift hover:text-foreground",
        danger && "hover:bg-danger hover:text-white",
      )}
      {...props}
    >
      {children}
    </button>
  );
}
