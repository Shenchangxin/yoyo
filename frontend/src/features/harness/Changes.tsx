// 本文件拥有一份材料变更的列表外观。不拥有变更如何算出来。HarnessWorkspace 和 HarnessLab 传入 changes。

import { useCopy } from "../../lib/i18n";
import type { MaterialChange } from "../../lib/harness-refs";
import { surfaceLabel } from "../../lib/harness-copy";
import { cn } from "../../lib/utils";

/** MaterialChangeList 没有条目时：传了 empty 就显示它，否则什么都不画。added 用成功色，removed 用危险色，其余用弱色。l3 为真时多标一枚文案。changes 读取 length，做映射。 */
export function MaterialChangeList({ changes, empty }: { changes: MaterialChange[]; empty?: string }) {
  const copy = useCopy();
  if (!changes.length) {
    return empty ? <p className="text-[13px] text-muted">{empty}</p> : null;
  }
  return (
    <ul className="space-y-1" data-testid="harness-changes">
      {changes.map((c, i) => (
        <li key={`${c.surface}-${c.op}-${c.id}-${i}`} className="flex flex-wrap items-baseline gap-x-2 text-[13px] leading-5">
          <span className="font-medium text-foreground/85">{surfaceLabel(copy, c.surface)}</span>
          <span className={cn(
            "text-[11px] font-medium",
            c.op === "added" ? "text-success" : c.op === "removed" ? "text-danger" : "text-muted",
          )}>
            {opLabel(copy, c.op)}
          </span>
          {c.l3 ? <span className="text-[11px] text-warning">{copy.rsi.l3}</span> : null}
          <span className="min-w-0 break-words text-foreground/90">{c.detail || c.id || `${c.from} → ${c.to}`}</span>
        </li>
      ))}
    </ul>
  );
}

/** opLabel added、removed、changed 换成文案，其余操作名原样显示。copy 读取 rsi。op 与 added、removed、changed 比较是否相等，有的分支原样返回。 */
function opLabel(copy: ReturnType<typeof useCopy>, op: string): string {
  if (op === "added") return copy.rsi.added;
  if (op === "removed") return copy.rsi.removed;
  if (op === "changed") return copy.rsi.changed;
  return op;
}
