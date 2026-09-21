import { cn } from "../../lib/utils";

export function SandboxedFrame({ html, title, fill }: { html: string; title: string; fill?: boolean }) {
  return (
    <iframe
      title={title}
      sandbox="allow-scripts allow-forms"
      className={cn(
        "w-full min-w-0 max-w-full bg-background",
        fill ? "h-full min-h-0 rounded-none border-0" : "mt-2 h-[28rem] rounded-lg border border-border/70",
      )}
      srcDoc={html}
      data-testid="html-preview"
    />
  );
}
