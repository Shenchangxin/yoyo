/**
 * 拥有：订阅一条 matchMedia 查询，并在匹配变化时更新。
 * 不拥有：具体断点取什么值。
 */

import { useEffect, useState } from "react";

/** useMedia 当前视口是否匹配该查询。没有 window 或 matchMedia 时为假。query 传给 window.matchMedia。 */
export function useMedia(query: string): boolean {
  const [match, setMatch] = useState(() => {
    if (typeof window === "undefined" || !window.matchMedia) return false;
    return window.matchMedia(query).matches;
  });
  useEffect(() => {
    const mq = window.matchMedia(query);
    const on = () => setMatch(mq.matches);
    on();
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, [query]);
  return match;
}
