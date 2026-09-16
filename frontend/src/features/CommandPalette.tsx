import * as Dialog from "@radix-ui/react-dialog";
import { Command } from "cmdk";
import type { Lab, Thread } from "../lib/protocol";
import { copy } from "../lib/copy";
import { chrome } from "../lib/chrome";

export function CommandPalette(props: {
  open: boolean;
  threads: Thread[];
  onClose: () => void;
  onNew: () => void;
  onLab: (lab: Lab) => void;
  onSelectThread: (t: Thread) => void;
  onDiff: () => void;
}) {
  return (
    <Dialog.Root open={props.open} onOpenChange={(v) => { if (!v) props.onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-background/70" />
        <Dialog.Content className="fixed left-1/2 top-[16vh] z-50 w-[min(520px,calc(100%-2rem))] -translate-x-1/2 overflow-hidden rounded-2xl border border-border bg-panel focus:outline-none">
          <Dialog.Title className="sr-only">{copy.palette.title}</Dialog.Title>
          <Dialog.Description className="sr-only">{copy.palette.desc}</Dialog.Description>
          <Command label={copy.palette.title}>
            <Command.Input placeholder={copy.palette.placeholder} />
            <Command.List>
              <Command.Empty>{copy.palette.empty}</Command.Empty>
              <Command.Group heading="Actions">
                <Command.Item value="new chat" onSelect={() => { props.onNew(); props.onClose(); }}>New chat</Command.Item>
                <Command.Item value="agent" onSelect={() => { props.onLab("agent"); props.onClose(); }}>Agent</Command.Item>
                <Command.Item value="harbor eval" onSelect={() => { props.onLab("harbor"); props.onClose(); }}>Harbor</Command.Item>
                <Command.Item value="evolve ace" onSelect={() => { props.onLab("evolve"); props.onClose(); }}>Evolve</Command.Item>
                <Command.Item value="harness refs" onSelect={() => { props.onLab("harness"); props.onClose(); }}>Harness</Command.Item>
                <Command.Item value="control settings theme" onSelect={() => { props.onLab("control"); props.onClose(); }}>Control</Command.Item>
                <Command.Item value="refresh diff" onSelect={() => { props.onDiff(); props.onClose(); }}>Refresh diff</Command.Item>
                <Command.Item value="quit" onSelect={() => { chrome.close(); props.onClose(); }}>Quit</Command.Item>
              </Command.Group>
              {props.threads.length ? (
                <Command.Group heading="Chats">
                  {props.threads.slice(0, 20).map((t) => (
                    <Command.Item
                      key={t.id}
                      value={t.title + " " + t.id}
                      onSelect={() => {
                        props.onSelectThread(t);
                        props.onLab("agent");
                        props.onClose();
                      }}
                    >
                      {t.title || t.id.slice(0, 8)}
                    </Command.Item>
                  ))}
                </Command.Group>
              ) : null}
            </Command.List>
          </Command>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
