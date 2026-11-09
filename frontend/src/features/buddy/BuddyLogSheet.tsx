/**
 * 拥有：一个 agent 的日志侧板：读安装和探测日志的末尾，可刷新；正在安装时每两秒自动刷新。
 * 不拥有：日志内容和打码。后端写日志时已经把密钥打码，这里原样显示。
 */

import { RefreshCw, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Button } from "../../components/ui/button";
import { Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle } from "../../components/ui/sheet";
import { buddy, type BuddyView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { fill } from "./buddy-model";
import { errText } from "./buddy-store";

/** LogBody 读日志末尾并显示。installing 为真时每两秒重读一次。 */
function LogBody({ id, installing }: { id: string; installing: boolean }) {
  const c = useCopy().buddy;
  const [text, setText] = useState<string | null>(null);
  const load = useCallback(() => {
    void buddy
      .logTail(id, 400)
      .then((res) => setText(res?.text ?? ""))
      .catch((e) => setText(errText(e)));
  }, [id]);
  useEffect(() => {
    load();
    if (!installing) return;
    const timer = window.setInterval(load, 2000);
    return () => window.clearInterval(timer);
  }, [load, installing]);
  return (
    <>
      <div className="flex shrink-0 items-center justify-end px-4 pt-3">
        <Button size="xs" variant="ghost" onClick={load}>
          <RefreshCw aria-hidden />
          {c.log.refresh}
        </Button>
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-4 pb-4 pt-2">
        {text === null ? (
          <div className="h-24 animate-pulse rounded-md bg-lift/70" />
        ) : text.trim() ? (
          <pre className="whitespace-pre-wrap break-words font-mono text-ui-caption leading-5 text-foreground/85" data-testid="buddy-log-text">
            {text}
          </pre>
        ) : (
          <p className="text-ui-footnote text-muted">{c.log.empty}</p>
        )}
      </div>
    </>
  );
}

/** BuddyLogSheet 是从右边滑出的日志侧板。v 为空时关着。 */
export function BuddyLogSheet({ v, onClose }: { v: BuddyView | undefined; onClose: () => void }) {
  const c = useCopy().buddy;
  return (
    <Sheet open={!!v} onOpenChange={(next) => !next && onClose()}>
      <SheetContent side="right" className="flex w-[min(560px,94%)] flex-col" data-testid="buddy-log">
        <div className="flex h-12 shrink-0 items-center justify-between border-b border-border px-4">
          <SheetTitle className="truncate text-ui-body font-medium">{v ? fill(c.log.title, { name: v.name }) : ""}</SheetTitle>
          <SheetClose className="grid size-8 place-items-center rounded-lg text-muted hover:bg-lift hover:text-foreground" aria-label={c.action.cancel}>
            <X className="size-4" />
          </SheetClose>
        </div>
        <SheetDescription className="px-4 pt-3 text-ui-footnote text-muted">{c.log.body}</SheetDescription>
        {v ? <LogBody key={v.id} id={v.id} installing={v.state === "installing"} /> : null}
      </SheetContent>
    </Sheet>
  );
}
