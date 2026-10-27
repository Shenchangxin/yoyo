// 本文件拥有正在跑的会话入口和耗时。不拥有会话是否在跑。调用方传入 threads、status 和 onSelect。窄屏不显示。

import { useEffect, useMemo, useState } from "react";
import { useCopy } from "../lib/i18n";
import { RollTime } from "./session-chrome/Counter";
import type { RunStatus, Thread } from "../lib/protocol";

/** RunningHub 没有线程时不渲染。标题空了就显示 id 的前 8 位。有线程时每 1000 毫秒刷新一次现在。props 读取 threads、status，调用 onSelect。 */
export function RunningHub(props: {
  threads: Thread[];
  status?: RunStatus[];
  onSelect: (t: Thread) => void;
}) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  // 下一拍再监听按下和 Escape，避免打开它的那一下立刻关掉。卸下或合上时摘掉监听。
  useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    /** onKey 调用 close。e 读取 key。 */
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") close(); };
    const tmr = window.setTimeout(() => {
      window.addEventListener("mousedown", close);
      window.addEventListener("keydown", onKey);
    }, 0);
    return () => {
      window.clearTimeout(tmr);
      window.removeEventListener("mousedown", close);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);
  useEffect(() => {
    if (!props.threads.length) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [props.threads.length]);
  const byId = useMemo(() => {
    const m = new Map<string, RunStatus>();
    for (const s of props.status || []) m.set(s.id, s);
    return m;
  }, [props.status]);
  if (!props.threads.length) return null;
  return (
    <div className="relative hidden sm:block">
      <button
        type="button"
        className="inline-flex items-center gap-1.5 rounded-full bg-lift px-2 py-0.5 text-[11px] tabular-nums text-foreground transition-[background-color,transform] duration-[var(--duration-fast)] ease-[var(--ease-out)] hover:bg-lift/80 active:scale-[0.98]"
        aria-expanded={open}
        aria-haspopup="listbox"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="pulse-dot" aria-hidden />
        {props.threads.length} {copy.titlebar.live}
      </button>
      {open ? (
        <div
          role="listbox"
          className="absolute right-0 z-30 mt-1 min-w-[240px] overflow-hidden rounded-xl border border-border bg-popover py-1 shadow-[var(--shadow-popover)]"
          onMouseDown={(e) => e.stopPropagation()}
        >
          {props.threads.map((t) => {
            const st = byId.get(t.id);
            const started = st?.startedAt ? Date.parse(st.startedAt) : NaN;
            return (
              <button
                type="button"
                key={t.id}
                role="option"
                className="u-row-hover block w-full px-3 py-1.5 text-left"
                onClick={() => {
                  props.onSelect(t);
                  setOpen(false);
                }}
              >
                <span className="block truncate text-[12px]">{t.title || t.id.slice(0, 8)}</span>
                <span className="mt-0.5 flex items-center gap-2 text-[11px] tabular-nums text-muted">
                  {Number.isFinite(started) ? <RollTime ms={Math.max(0, now - started)} kind="clock" live /> : null}
                  {st?.lastTool ? <span className="truncate">{st.lastTool}</span> : <span>{copy.titlebar.waiting}</span>}
                </span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
