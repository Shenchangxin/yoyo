// 本文件拥有「这段聊天不行」的入口按钮。不拥有改进流程。App 和 ChatDock 传入 onImprove。

import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { cn } from "../../lib/utils";

/** ChatNotWorking 点击时调用 onImprove。compact 为真时左右留出 12px。props 读取 compact、onImprove。 */
export function ChatNotWorking(props: { onImprove: () => void; compact?: boolean }) {
  const copy = useCopy();
  return (
    <div className={cn("pb-1", props.compact ? "px-3" : "")}>
      <button
        type="button"
        data-testid="chat-not-working"
        className="text-ui-footnote text-muted transition-colors hover:text-foreground"
        onClick={props.onImprove}
      >
        {copy.rsi.chatNotWorking}
      </button>
    </div>
  );
}

/** lastTurnFailed 为真时最后一轮以出错收尾。主对话只在这时给出入口，平时不占位置。 */
export function lastTurnFailed(items: Item[]): boolean {
  for (let i = items.length - 1; i >= 0; i--) {
    const type = items[i].type;
    if (type === "turn_end" || type === "system" || type === "compaction") continue;
    return type === "error";
  }
  return false;
}
