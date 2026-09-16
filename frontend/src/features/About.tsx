import { useCopy } from "../lib/i18n";
import { Button } from "../components/ui/button";
import * as Dialog from "@radix-ui/react-dialog";

export function About(props: {
  open: boolean;
  info: Record<string, any>;
  onClose: () => void;
}) {
  const copy = useCopy();
  const rows: [string, string][] = [
    ["version", String(props.info.version || "")],
    ["harness", String(props.info.harness || "")],
    ["model", String(props.info.model || "")],
    ["isolated", String(!!props.info.isolated)],
    ["workspace", String(props.info.workspace_ready ?? props.info.workspaceReady ?? "")],
  ];
  return (
    <Dialog.Root open={props.open} onOpenChange={(v) => { if (!v) props.onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-background/70" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-panel p-5 focus:outline-none">
          <Dialog.Title className="text-base font-semibold">{copy.about.title}</Dialog.Title>
          <Dialog.Description className="mt-2 text-sm text-muted">{copy.about.body}</Dialog.Description>
          <dl className="mt-4 space-y-2 font-mono text-xs">
            {rows.map(([k, v]) => (
              <div key={k} className="flex justify-between gap-4">
                <dt className="text-muted">{k}</dt>
                <dd className="truncate text-foreground">{v || "—"}</dd>
              </div>
            ))}
          </dl>
          <div className="mt-5 flex justify-end">
            <Button onClick={props.onClose}>{copy.about.close}</Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
