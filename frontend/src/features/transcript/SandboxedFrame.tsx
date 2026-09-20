export function SandboxedFrame({ html, title }: { html: string; title: string }) {
  return (
    <iframe
      title={title}
      sandbox="allow-scripts allow-forms"
      className="mt-2 h-[28rem] w-full min-w-0 max-w-full rounded-lg border border-border/70 bg-background"
      srcDoc={html}
      data-testid="html-preview"
    />
  );
}
