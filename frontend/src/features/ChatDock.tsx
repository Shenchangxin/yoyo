import { Transcript } from "./Transcript";
import { Composer } from "./Composer";
import type { Approval, ContextUsage, Item } from "../lib/protocol";
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
  provider?: string;
  ctx?: ContextUsage;
  onSend: (opts?: { steer?: boolean; attachments?: import("../lib/protocol").Attachment[] }) => void;
  onStop: () => void;
  onResolve: (id: string, decision: string) => void;
  onOpenAgent: () => void;
  onSlash?: (cmd: string, rest: string) => void;
  onModel?: (model: string) => void;
}) {
  const copy = useCopy();
  return (
    <aside className="flex h-full min-h-0 flex-col bg-transparent">
      <div className="flex h-9 shrink-0 items-center border-b border-border/80 px-3">
        <span className="text-[13px] font-medium">{copy.dock.chat}</span>
        <button type="button" className="ml-auto rounded-md px-2 py-1 text-[11px] text-muted hover:bg-lift hover:text-foreground" onClick={props.onOpenAgent}>
          {copy.dock.expand}
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
        provider={props.provider}
        ctx={props.ctx}
        onSend={props.onSend}
        onStop={props.onStop}
        onSlash={props.onSlash}
        onModel={props.onModel}
        compact
      />
    </aside>
  );
}
