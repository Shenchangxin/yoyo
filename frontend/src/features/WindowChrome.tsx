import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Minus, Monitor, Moon, Search, Square, Sun, X } from "lucide-react";
import { chrome, isMac, showCaptionButtons } from "../lib/chrome";
import { useTheme } from "../lib/theme";
import { cn } from "../lib/utils";

export function WindowChrome(props: {
  connected: boolean;
  isolated?: boolean;
  onPalette: () => void;
}) {
  const mac = isMac();
  const caps = showCaptionButtons() && !mac;
  const { cycle, pref, resolved } = useTheme();
  const ThemeIcon = pref === "system" ? Monitor : resolved === "dark" ? Sun : Moon;
  return (
    <header
      className={cn("chrome drag flex h-10 shrink-0 items-center border-b border-border bg-sidebar", mac && "pl-[76px]")}
      onDoubleClick={() => chrome.toggleMaximise()}
    >
      <div className="flex min-w-0 flex-1 items-center gap-2 pl-3">
        <span className="text-[13px] font-semibold tracking-tight">Yoyo</span>
        <span className={cn("size-1.5 rounded-full", props.connected ? "bg-accent" : "bg-muted")} title={props.connected ? "Connected" : "Disconnected"} />
        {props.isolated ? <span className="rounded-full bg-lift px-1.5 py-0.5 text-[10px] text-muted">isolated</span> : null}
      </div>
      <div className="no-drag flex h-full items-center gap-1 pr-1">
        <button
          type="button"
          className="mr-1 hidden h-7 items-center gap-2 rounded-lg bg-lift px-2.5 text-[11px] text-muted hover:text-foreground sm:flex"
          onClick={props.onPalette}
          aria-label="Open command palette"
        >
          <Search className="size-3.5" aria-hidden />
          Jump
          <kbd className="rounded border border-border px-1 text-[10px]">{mac ? "⌘K" : "Ctrl K"}</kbd>
        </button>
        <button
          type="button"
          className="grid size-8 place-items-center rounded-lg text-muted hover:bg-lift hover:text-foreground"
          onClick={cycle}
          aria-label={`Theme ${pref}`}
          title={`Theme: ${pref}`}
        >
          <ThemeIcon className="size-3.5" />
        </button>
        {caps ? (
          <div className="ml-1 flex h-full">
            <Cap aria-label="Minimise" onClick={() => chrome.minimise()}>
              <Minus className="size-3.5" />
            </Cap>
            <Cap aria-label="Maximise" onClick={() => chrome.toggleMaximise()}>
              <Square className="size-3" />
            </Cap>
            <Cap aria-label="Close" onClick={() => chrome.close()} danger>
              <X className="size-3.5" />
            </Cap>
          </div>
        ) : null}
      </div>
    </header>
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
        "grid h-full w-11 place-items-center text-muted hover:bg-lift hover:text-foreground",
        danger && "hover:bg-danger hover:text-white",
      )}
      {...props}
    >
      {children}
    </button>
  );
}
