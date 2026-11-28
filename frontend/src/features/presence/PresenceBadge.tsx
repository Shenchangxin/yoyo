/**
 * 拥有：导航轨上的头像和一句状态。三级访问的前两级在这里：悬停看一句话（状态），点开在右侧面板预览那张工单（预览）；
 * 预览里的「放到中间聊」是第三级（接管），由任务页负责。
 * 不拥有：状态怎么算（Go 的 app/presence.go），也不拥有工单页。
 */

import { StatusDot } from "../../components/ui/status-dot";
import { Tooltip } from "../../components/ui/tooltip";
import { showTask } from "../superagent/needs";
import { useDeskPresence } from "./desk-presence";
import { PresenceStamp } from "./PresenceRuntime";

type Tone = Parameters<typeof StatusDot>[0]["tone"];

const TONES: Record<string, Tone> = {
  working: "live",
  needs: "needs",
  blocked: "failed",
  done: "done",
  paused: "idle",
  idle: "idle",
};

/** PresenceBadge 在状态还没读到时不画。有工单时点开预览它。 */
export function PresenceBadge() {
  const p = useDeskPresence();
  if (!p || !p.state) return null;
  const tone = TONES[p.state] ?? "idle";
  return (
    <Tooltip content={p.line}>
      <button
        type="button"
        className="flex w-full min-w-0 items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors duration-[var(--duration-fast)] hover:bg-lift disabled:cursor-default disabled:hover:bg-transparent"
        data-testid="presence-badge"
        data-state={p.state}
        aria-label={p.line}
        disabled={!p.ticket}
        onClick={() => { if (p.ticket) showTask(p.ticket); }}
      >
        <PresenceStamp size={22} live={p.state === "working"} />
        <span className="min-w-0 flex-1 truncate text-ui-footnote text-muted">{p.line}</span>
        <StatusDot tone={tone} still={p.state !== "working"} />
      </button>
    </Tooltip>
  );
}
