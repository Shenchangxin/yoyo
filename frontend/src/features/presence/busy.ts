/** 拥有当前进行或加载这一轮已经过去的时间。不拥有表情选择。本地计时，不改服务端。 */
import { useEffect, useRef, useState } from "react";

/** useBusyMs 记下这一轮已经过去的时间，key 一变就重新算。 */
export function useBusyMs(live: boolean, key = ""): number {
  const since = useRef<number | null>(null);
  const [ms, setMs] = useState(0);
  useEffect(() => {
    if (!live) {
      since.current = null;
      setMs(0);
      return;
    }
    since.current = Date.now();
    setMs(0);
    const t = window.setInterval(() => {
      if (since.current != null) setMs(Date.now() - since.current);
    }, 700);
    return () => window.clearInterval(t);
  }, [live, key]);
  return ms;
}
