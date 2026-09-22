import { useCopy } from "../lib/i18n";
import { Button } from "../components/ui/button";
import * as Dialog from "@radix-ui/react-dialog";
import { MarkWell } from "./shell/YoyoMark";

export function About(props: {
  open: boolean;
  info: Record<string, any>;
  onClose: () => void;
}) {
  const copy = useCopy();
  const isolated = !!props.info.isolated;
  const rows: [string, string][] = [
    [copy.settings.currentVersion, String(props.info.version || "")],
    [copy.rail.harness, String(props.info.harness || "")],
    [copy.rail.model, String(props.info.model || "")],
    [copy.rail.isolated, isolated ? copy.settings.yes : copy.settings.no],
    [copy.rail.workspace, String(props.info.workspace_ready ?? props.info.workspaceReady ?? "")],
  ];
  return (
    <Dialog.Root open={props.open} onOpenChange={(v) => { if (!v) props.onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-background/70" />
        <Dialog.Content className="dialog-sheet fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-md border border-border bg-popover p-5 shadow-[var(--shadow-popover)] focus:outline-none">
          <MarkWell className="size-9" markClassName="size-3.5" />
          <div>
            <Dialog.Title className="text-[15px] font-semibold tracking-[-0.02em]">{copy.about.title}</Dialog.Title>
            <Dialog.Description className="mt-[var(--space-item)] text-[13px] leading-[1.55] text-muted">{copy.about.body}</Dialog.Description>
          </div>
          <dl className="flex flex-col gap-[var(--space-item)]">
            {rows.map(([k, v]) => (
              <div key={k} className="flex min-h-8 items-center justify-between gap-4 text-[13px]">
                <dt className="text-foreground">{k}</dt>
                <dd className="truncate font-mono text-[12px] tabular-nums text-muted">{v || "—"}</dd>
              </div>
            ))}
          </dl>
          <div className="dialog-actions">
            <Button onClick={props.onClose}>{copy.about.close}</Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
