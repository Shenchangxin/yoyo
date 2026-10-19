import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";

export function ChatNotWorking(props: { onImprove: () => void; compact?: boolean }) {
  const copy = useCopy();
  return (
    <div className={cn("pb-1", props.compact ? "px-3" : "")}>
      <button
        type="button"
        data-testid="chat-not-working"
        className="text-[12px] text-muted transition-colors hover:text-foreground"
        onClick={props.onImprove}
      >
        {copy.rsi.chatNotWorking}
      </button>
    </div>
  );
}
