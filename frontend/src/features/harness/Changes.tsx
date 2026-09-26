import { useCopy } from "../../lib/i18n";
import type { MaterialChange } from "../../lib/harness-refs";
import { cn } from "../../lib/utils";

export function MaterialChangeList({ changes, empty }: { changes: MaterialChange[]; empty?: string }) {
  const copy = useCopy();
  if (!changes.length) {
    return empty ? <p className="text-[13px] text-muted">{empty}</p> : null;
  }
  return (
    <ul className="space-y-1" data-testid="harness-changes">
      {changes.map((c, i) => (
        <li key={`${c.surface}-${c.op}-${c.id}-${i}`} className="flex flex-wrap items-baseline gap-x-2 text-[13px] leading-5">
          <span className="font-medium text-foreground/85">{c.surface}</span>
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

function opLabel(copy: ReturnType<typeof useCopy>, op: string): string {
  if (op === "added") return copy.rsi.added;
  if (op === "removed") return copy.rsi.removed;
  if (op === "changed") return copy.rsi.changed;
  return op;
}
