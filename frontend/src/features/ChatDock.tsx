import { Transcript } from "./Transcript";
import { Composer } from "./Composer";
import type { Approval, Item } from "../lib/protocol";
import { useCopy } from "../lib/i18n";

export function ChatDock(props: {
  items: Item[];
  approvals: Approval[];
  running: boolean;
  draftKey: string;
  disabled: boolean;
  disabledReason?: string;
  model?: string;
  models?: string[];
  onSend: (opts?: { steer?: boolean; attachments?: import("../lib/protocol").Attachment[] }) => void;
  onStop: () => void;
  onResolve: (id: string, decision: string) => void;
  onOpenAgent: () => void;
  onSlash?: (cmd: string, rest: string) => void;
  onModel?: (model: string) => void;
}) {
  const copy = useCopy();
  return (
    <aside className="flex h-full min-h-0 flex-col border-l border-border bg-background">
      <div className="flex h-11 shrink-0 items-center border-b border-border px-3">
        <span className="text-[13px] font-medium">Chat</span>
        <button type="button" className="ml-auto text-[11px] text-muted hover:text-foreground" onClick={props.onOpenAgent}>
          Expand
        </button>
      </div>
      <Transcript items={props.items.slice(-12)} approvals={props.approvals} running={props.running} compact onResolve={props.onResolve} />
      <Composer
        draftKey={props.draftKey}
        running={props.running}
        disabled={props.disabled}
        disabledReason={props.disabledReason || copy.composer.disabled}
        model={props.model}
        models={props.models}
        onSend={props.onSend}
        onStop={props.onStop}
        onSlash={props.onSlash}
        onModel={props.onModel}
      />
    </aside>
  );
}
