import { useEffect, useRef, useState } from "react";

/** Elapsed ms of the current live/loading bout. Resets when `key` changes. */
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
