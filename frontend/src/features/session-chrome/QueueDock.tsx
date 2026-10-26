/**
 * 拥有：下一轮内存队列的停靠条。接到的是内存里的下一轮队列，不是 Cordis 收件箱。
 * 不拥有：入队。取消和排序交给外面，不调用 api。
 */
import { ChevronDown, ChevronUp, X } from "lucide-react";
import type { Copy } from "../../lib/copy";

/** QueueDock 列出排队的下一轮，取消和排序交给外面。 */
export function QueueDock(props: {
  items: { id?: string; text?: string; plan?: boolean }[];
  onCancel?: (id: string) => void;
  onReorder?: (id: string, delta: number) => void;
  copy: Copy;
}) {
  if (!props.items.length) return null;
  return (
    <div className="sc-queue" data-testid="composer-queue">
      <div className="sc-queue-panel">
        {props.items.map((it, i) => {
          const id = it.id || `q-${i}`;
          return (
            <div key={id} className="sc-queue-row">
              <span className="min-w-0 flex-1 truncate text-foreground/85">{it.text || props.copy.composer.queue}</span>
              {props.onReorder ? (
                <>
                  <button
                    type="button"
                    className="grid size-6 place-items-center rounded-md text-muted hover:bg-background hover:text-foreground"
                    aria-label={props.copy.composer.queueUp}
                    disabled={i === 0}
                    onClick={() => props.onReorder?.(id, -1)}
                  >
                    <ChevronUp size={14} strokeWidth={1} />
                  </button>
                  <button
                    type="button"
                    className="grid size-6 place-items-center rounded-md text-muted hover:bg-background hover:text-foreground"
                    aria-label={props.copy.composer.queueDown}
                    disabled={i === props.items.length - 1}
                    onClick={() => props.onReorder?.(id, 1)}
                  >
                    <ChevronDown size={14} strokeWidth={1} />
                  </button>
                </>
              ) : null}
              <button
                type="button"
                className="grid size-6 place-items-center rounded-md text-muted hover:bg-danger/10 hover:text-danger"
                aria-label={props.copy.composer.queueCancel}
                onClick={() => props.onCancel?.(id)}
              >
                <X size={14} strokeWidth={1} />
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
