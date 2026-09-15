import * as Dialog from "@radix-ui/react-dialog";
import type { ReactNode } from "react";
import { Button } from "../components/ui/button";

export function ConfirmDialog(props: {
  open: boolean;
  title: string;
  body: ReactNode;
  confirmLabel?: string;
  danger?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog.Root open={props.open} onOpenChange={(v) => { if (!v) props.onCancel(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-background/70" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-panel p-5 focus:outline-none">
          <Dialog.Title className="text-base font-semibold">{props.title}</Dialog.Title>
          <Dialog.Description className="mt-2 text-sm text-muted">{props.body}</Dialog.Description>
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="lift" onClick={props.onCancel}>Cancel</Button>
            <Button variant={props.danger ? "danger" : "default"} onClick={props.onConfirm}>
              {props.confirmLabel || "Confirm"}
            </Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
