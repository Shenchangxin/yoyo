/**
 * 拥有：读头像的全局状态（Go 的 app/presence.go 算出来，跟着开口门状态一起回来）。导航轨头像、伴侣窗读的是同一份。
 * 不拥有：状态怎么算。这里只拉取，开口门有变化时立刻再拉一次。
 */

import { useEffect, useState } from "react";
import { deskState, type DeskPresence } from "../../lib/client";
import { DESK_EVENT } from "../superagent/desk-feed";

/** useDeskPresence 每 every 毫秒读一次头像状态。读不到时留着上一次的。 */
export function useDeskPresence(every = 5000): DeskPresence | null {
  const [presence, setPresence] = useState<DeskPresence | null>(null);
  useEffect(() => {
    let alive = true;
    const pull = () => {
      void deskState()
        .then((state) => { if (alive && state.presence.state) setPresence(state.presence); })
        .catch(() => {});
    };
    pull();
    const timer = window.setInterval(pull, every);
    window.addEventListener(DESK_EVENT, pull);
    return () => {
      alive = false;
      window.clearInterval(timer);
      window.removeEventListener(DESK_EVENT, pull);
    };
  }, [every]);
  return presence;
}
