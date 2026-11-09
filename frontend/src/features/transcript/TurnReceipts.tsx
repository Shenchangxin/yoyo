/**
 * 拥有：对话里和主动式相关的两块小东西：一轮末尾的「已记住」回执，以及工人回报上面那一行状态和做事的 agent。
 * 不拥有：去看之后翻到哪里，那由今天栏接住 openDesk。
 */

import { Receipt } from "../../components/ui/event-card";
import { StatusDot, type StatusTone } from "../../components/ui/status-dot";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import type { AgentPart } from "../../lib/transcript-layout";
import { BuddyMark } from "../buddy/BuddyMark";
import { useBuddy } from "../buddy/buddy-store";
import { openDesk, useNeeds } from "../superagent/needs";
import { memoryReceipts } from "./receipts";

/** TurnReceipts 在一轮末尾列出这一轮写进记忆的句子。今天栏在场时可以点过去看。 */
export function TurnReceipts(props: { parts: AgentPart[] }) {
  const copy = useCopy();
  const mounted = useNeeds((s) => s.paneMounted);
  const notes = memoryReceipts(props.parts);
  if (!notes.length) return null;
  return (
    <div className="mt-2 space-y-0.5" data-testid="memory-receipts">
      {notes.map((text) => (
        <Receipt key={text} action={mounted ? { label: copy.desk.rememberedOpen, onClick: () => { openDesk({ tab: "memory" }); } } : undefined}>
          {copy.desk.remembered.replace("{text}", text)}
        </Receipt>
      ))}
    </div>
  );
}

/**
 * TaskEventHead 是工人回报上面的一行：状态点、这张工单现在怎样，以及去今天栏看它。
 * 交给外部 agent 的工单还写出是谁：名字先用 Buddy 列表里的，列表还没取到时用后端写进消息的 agentName。要登录时给一个「去登录」。
 */
export function TaskEventHead(props: { ticket: string; state: string; handoff: boolean; agent?: string; agentName?: string; need?: string }) {
  const copy = useCopy();
  const mounted = useNeeds((s) => s.paneMounted);
  const buddy = useBuddy((s) => (props.agent ? s.list.find((v) => v.id === props.agent) : undefined));
  const agentName = buddy?.name || props.agentName || props.agent || "";
  const failed = props.state === "failed" || props.state === "blocked" || props.state === "delivery";
  const tone: StatusTone = props.handoff
    ? (failed ? "failed" : "needs")
    : props.state === "stopped" ? "idle" : props.state === "waiting" ? "live" : "done";
  const label = props.handoff
    ? copy.transcript.needsYou
    : props.state === "stopped" ? copy.work.taskStopped : props.state === "waiting" ? copy.work.waiting : copy.work.taskDone;
  return (
    <div className="flex min-w-0 items-center gap-1.5 text-ui-caption text-muted" data-testid="task-event-head">
      <StatusDot tone={tone} still />
      <span className={props.handoff ? "font-medium text-foreground/85" : "font-medium"}>{label}</span>
      {props.agent ? (
        <span className="flex min-w-0 items-center gap-1" data-testid="task-event-agent">
          <span aria-hidden>·</span>
          <BuddyMark v={buddy ?? { name: agentName }} className="size-3 text-ui-badge" />
          <span className="truncate">{agentName}</span>
        </span>
      ) : null}
      <span className="ml-auto flex shrink-0 items-center gap-3">
        {props.need === "login" && props.agent ? (
          <button
            type="button"
            className="text-foreground/85 underline decoration-dotted underline-offset-[3px] hover:text-foreground"
            data-testid="task-event-login"
            onClick={() => useUI.getState().openBuddy(props.agent)}
          >
            {copy.tasks.goLogin}
          </button>
        ) : null}
        {props.ticket && mounted ? (
          <button
            type="button"
            className="text-foreground/85 underline decoration-dotted underline-offset-[3px] hover:text-foreground"
            onClick={() => { openDesk({ tab: "tasks", ticket: props.ticket }); }}
          >
            {copy.desk.eventOpen}
          </button>
        ) : null}
      </span>
    </div>
  );
}
