export function SandboxedFrame({ html, title }: { html: string; title: string }) {
  return (
    <iframe
      title={title}
      sandbox="allow-scripts allow-forms"
      className="mt-2 h-64 w-full rounded-md border border-border/80 bg-background"
      srcDoc={html}
    />
  );
}
