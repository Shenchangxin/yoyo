/**
 * 拥有：主会话里任务组唤醒路由时的那条回执：居中一条细线，中间写任务组完成了或需要你决定，点它打开任务组。
 * 不拥有：什么时候唤醒、写给路由的回报正文（superagent 的 watchGroups 和 GroupDigest）。回报是写给路由看的，这里不显示正文。
 */

import { StatusDot } from "../../components/ui/status-dot";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { fill } from "../buddy/buddy-model";
import { showGroup, useNeeds } from "../superagent/needs";

/** WakeReceipt 是一条唤醒回执。只有做完的任务组说完成了，有人在问、有没成的都说需要你决定。 */
export function WakeReceipt({ item }: { item: Item }) {
  const copy = useCopy();
  const gid = String(item.payload?.group || "");
  const state = String(item.payload?.state || "");
  const known = useNeeds((s) => s.view?.groups?.find((row) => row.id === gid)?.title || "");
  const title = String(item.payload?.title || "") || known || gid;
  const done = state === "done";
  const text = fill(done ? copy.transcript.wakeDone : copy.transcript.wakeAttention, { title });
  return (
    <div className="flex items-center gap-3 py-1 text-ui-caption text-muted" data-testid="wake-receipt" data-state={state}>
      <span className="h-px flex-1 bg-border/70" aria-hidden />
      <button
        type="button"
        className="flex min-w-0 items-center gap-1.5 hover:text-foreground disabled:hover:text-muted"
        disabled={!gid}
        onClick={() => showGroup(gid)}
      >
        <StatusDot tone={done ? "done" : "needs"} still />
        <span className="truncate">{text}</span>
      </button>
      <span className="h-px flex-1 bg-border/70" aria-hidden />
    </div>
  );
}
