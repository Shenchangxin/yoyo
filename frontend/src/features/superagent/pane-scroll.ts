/**
 * 拥有：把一个从顶部量起的滚动容器接到内存位置上。
 * 不拥有：容器里放什么。没有记录时停在顶部。切走时不把被夹短的 scrollTop 写回去。
 */

import { useEffect, useLayoutEffect, useRef, type RefObject } from "react";
import { readPaneTop, writePaneTop } from "../../lib/pane-place";

/** usePaneScroll 按 key 记住 scrollTop。key 为空表示这一屏不滚动，不读也不写。noteGesture 告诉它这次滚动是人拖的。 */
export function usePaneScroll(key: string): { ref: RefObject<HTMLDivElement | null>; noteGesture: () => void } {
  const ref = useRef<HTMLDivElement>(null);
  const keyRef = useRef<string | null>(null);
  const pending = useRef<number | null>(null);
  const writing = useRef(false);
  const gestured = useRef(false);
  if (keyRef.current !== key) {
    keyRef.current = key;
    pending.current = key ? readPaneTop(key) : null;
    gestured.current = false;
  }

  useLayoutEffect(() => {
    const el = ref.current;
    const top = pending.current;
    if (!el || top == null || !keyRef.current) return;
    const max = Math.max(0, el.scrollHeight - el.clientHeight);
    const next = Math.min(top, max);
    if (Math.abs(el.scrollTop - next) > 1) {
      writing.current = true;
      el.scrollTop = next;
      writing.current = false;
    }
    if (max + 0.5 >= top) pending.current = null;
  });

  useEffect(() => {
    const el = ref.current;
    if (!el || !key) return;
    const apply = () => {
      const top = pending.current;
      const node = ref.current;
      if (top == null || !node) return;
      const max = Math.max(0, node.scrollHeight - node.clientHeight);
      const next = Math.min(top, max);
      if (Math.abs(node.scrollTop - next) > 1) {
        writing.current = true;
        node.scrollTop = next;
        writing.current = false;
      }
      if (max + 0.5 >= top) pending.current = null;
    };
    const onScroll = () => {
      if (writing.current) return;
      // 页签刚换、内容还没长到上次的位置时，滚动会被夹短。这时不能把夹短的数写进去。
      if (pending.current != null) {
        const max = Math.max(0, el.scrollHeight - el.clientHeight);
        if (!gestured.current || max + 0.5 < pending.current) return;
      }
      pending.current = null;
      gestured.current = false;
      writePaneTop(keyRef.current || "", el.scrollTop);
    };
    const mark = () => { gestured.current = true; };
    apply();
    el.addEventListener("scroll", onScroll, { passive: true });
    el.addEventListener("wheel", mark, { passive: true });
    el.addEventListener("pointerdown", mark);
    el.addEventListener("touchmove", mark, { passive: true });
    const ro = new ResizeObserver(() => {
      for (const child of el.children) ro.observe(child);
      apply();
    });
    ro.observe(el);
    for (const child of el.children) ro.observe(child);
    const mo = new MutationObserver(() => {
      for (const child of el.children) ro.observe(child);
      apply();
    });
    mo.observe(el, { childList: true });
    return () => {
      // key 换了时这里仍是上一屏的 effect。位置已经在滚动时记下，不在内容换掉之后再写。卸载时这一屏还在，把最后的位置留下。
      const node = ref.current;
      if (node && keyRef.current === key && pending.current == null) writePaneTop(key, node.scrollTop);
      el.removeEventListener("scroll", onScroll);
      el.removeEventListener("wheel", mark);
      el.removeEventListener("pointerdown", mark);
      el.removeEventListener("touchmove", mark);
      ro.disconnect();
      mo.disconnect();
    };
  }, [key]);

  return { ref, noteGesture: () => { gestured.current = true; } };
}
