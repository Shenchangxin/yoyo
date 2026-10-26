// 本文件拥有「这段聊天不行」的入口按钮。不拥有改进流程。App 和 ChatDock 传入 onImprove。

import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";

/** ChatNotWorking 点击时调用 onImprove。compact 为真时左右留出 12px。props 读取 compact、onImprove。 */
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
