/**
 * 拥有：今天栏里那条右侧滚动条。原生滚动条藏起来，换成一条短粗的胶囊。
 * 胶囊画在列表渐隐遮罩外面。悬停时整段显现，可以按住拖动；离开时整段一起变淡。
 * 不拥有：栏里排什么。滚动的是调用方放进来的内容。
 */

import { useEffect, useRef, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";
import { cn } from "../../lib/utils";

/** 胶囊的固定长度。比一截短点更长一些，仍不跟内容比例拉满整条轨道。 */
const OPEN_PX = 72;

/** DayScroll 是今天、任务、记忆、Buddy 共用的滚动层。reading 时不滚动，胶囊也不出现。 */
export function DayScroll({ reading, children }: { reading: boolean; children: ReactNode }) {
  const host = useRef<HTMLDivElement>(null);
  const scroller = useRef<HTMLDivElement>(null);
  const thumb = useRef<HTMLDivElement>(null);
  const open = useRef(false);
  const dragging = useRef(false);
  const paint = useRef<() => void>(() => {});

  useEffect(() => {
    const el = scroller.current;
    const bar = thumb.current;
    if (!el || !bar) return;
    let frame = 0;
    const draw = () => {
      frame = 0;
      const { scrollTop, scrollHeight, clientHeight } = el;
      const overflow = !reading && scrollHeight > clientHeight + 1;
      bar.style.transform = "none";
      if (!overflow) {
        bar.dataset.open = "false";
        return;
      }
      const length = Math.min(OPEN_PX, clientHeight);
      const range = scrollHeight - clientHeight;
      const travel = Math.max(0, clientHeight - length);
      bar.style.height = `${length}px`;
      bar.style.top = `${range <= 0 ? 0 : (scrollTop / range) * travel}px`;
      bar.dataset.open = open.current || dragging.current ? "true" : "false";
    };
    const schedule = () => {
      if (frame) return;
      frame = requestAnimationFrame(draw);
    };
    paint.current = schedule;
    schedule();
    el.addEventListener("scroll", schedule, { passive: true });
    const ro = new ResizeObserver(schedule);
    ro.observe(el);
    for (const child of el.children) ro.observe(child);
    const mo = new MutationObserver(() => {
      for (const child of el.children) ro.observe(child);
      schedule();
    });
    mo.observe(el, { childList: true });
    return () => {
      if (frame) cancelAnimationFrame(frame);
      el.removeEventListener("scroll", schedule);
      ro.disconnect();
      mo.disconnect();
    };
  }, [reading]);

  const onThumbDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    const el = scroller.current;
    const bar = thumb.current;
    if (!el || !bar) return;
    event.preventDefault();
    event.stopPropagation();
    const length = bar.offsetHeight || OPEN_PX;
    const travel = Math.max(1, el.clientHeight - length);
    const range = Math.max(0, el.scrollHeight - el.clientHeight);
    const startY = event.clientY;
    const startTop = el.scrollTop;
    dragging.current = true;
    open.current = true;
    bar.dataset.open = "true";
    bar.dataset.drag = "true";
    bar.setPointerCapture(event.pointerId);
    const onMove = (ev: PointerEvent) => {
      el.scrollTop = startTop + ((ev.clientY - startY) * range) / travel;
    };
    const onUp = (ev: PointerEvent) => {
      dragging.current = false;
      bar.dataset.drag = "false";
      bar.removeEventListener("pointermove", onMove);
      bar.removeEventListener("pointerup", onUp);
      bar.removeEventListener("pointercancel", onUp);
      if (bar.hasPointerCapture(ev.pointerId)) bar.releasePointerCapture(ev.pointerId);
      const under = document.elementFromPoint(ev.clientX, ev.clientY);
      if (!host.current?.contains(under)) {
        open.current = false;
        bar.dataset.open = "false";
      }
    };
    bar.addEventListener("pointermove", onMove);
    bar.addEventListener("pointerup", onUp);
    bar.addEventListener("pointercancel", onUp);
  };

  return (
    <div
      ref={host}
      className="relative min-h-0 min-w-0 flex-1 overflow-hidden"
      onMouseEnter={() => {
        const el = scroller.current;
        const bar = thumb.current;
        const overflow = !!el && el.scrollHeight > el.clientHeight + 1;
        open.current = overflow;
        if (bar && overflow) {
          bar.dataset.open = "true";
          bar.style.transform = "none";
        }
        paint.current();
      }}
      onMouseLeave={() => {
        if (dragging.current) return;
        open.current = false;
        const bar = thumb.current;
        if (!bar) return;
        bar.dataset.open = "false";
        bar.style.transform = "none";
      }}
    >
      <div className="rail-scroll-fade h-full min-h-0 overflow-hidden px-2">
        <div
          ref={scroller}
          className={cn(
            "day-scroll h-full min-w-0",
            reading ? "overflow-hidden pb-14" : "overflow-y-auto overflow-x-hidden overscroll-contain pb-16",
          )}
        >
          {children}
        </div>
      </div>
      <div
        ref={thumb}
        className="day-scroll-thumb"
        role="scrollbar"
        aria-orientation="vertical"
        onPointerDown={onThumbDown}
      />
    </div>
  );
}
