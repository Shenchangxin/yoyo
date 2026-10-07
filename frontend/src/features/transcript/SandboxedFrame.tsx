import { asPreviewDocument } from "../../lib/html-preview";
import { cn } from "../../lib/utils";

export function SandboxedFrame({ html, title, fill }: { html: string; title: string; fill?: boolean }) {
  return (
    <iframe
      title={title}
      sandbox="allow-scripts allow-forms"
      className={cn(
        "block w-full min-w-0 max-w-full bg-background",
        fill ? "h-full min-h-0 flex-1 rounded-none border-0" : "mt-2 h-[28rem] rounded-lg border border-border/70",
      )}
      style={{ colorScheme: "light" }}
      srcDoc={asPreviewDocument(html)}
      data-testid="html-preview"
    />
  );
}
