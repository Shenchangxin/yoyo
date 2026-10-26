/**
 * 拥有：紧凑通话画面、全屏通话层和收起后的通话条。
 * 不拥有：听写、播音和挂断请求，那些在 useIdolCall。立绘绘制在 IdolPresence。
 * IdolCall 挂上 useIdolCall，会随通话发请求改服务端状态。CallSurface 和 CallBar 只投影传入的状态，交互交给回调。
 */

import { useState, type ReactNode } from "react";
import { Mic, MicOff, PhoneOff, Volume2, VolumeX } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { IdolPresence } from "./IdolPresence";
import { hexAlpha, imageURL, mmss } from "./model";
import { standURL } from "./stand/StandView";
import { useIdolCall, type CallPhase, type CallWho } from "./useIdolCall";

/** IdolCall 渲染陪伴窗里的紧凑通话。挂断、静音和发送都进 useIdolCall。props 读取 paused、name、color、portrait、kind、stand 等。 */
export function IdolCall(props: {
  paused?: boolean;
  approvals?: number;
  name?: string;
  color?: string;
  portrait?: string;
  kind?: string;
  stand?: string;
  onHangup: () => void;
}) {
  const [work, setWork] = useState(true);
  const session = useIdolCall({
    active: !props.paused,
    companion: true,
    name: props.name,
    color: props.color,
    portrait: props.portrait,
    kind: props.kind,
    stand: props.stand,
    work,
    onEnded: props.onHangup,
  });
  return (
    <CallSurface
      compact
      who={session.who}
      phase={session.phase}
      emotion={session.emotion}
      caption={session.caption}
      notice={session.notice}
      elapsed={session.elapsed}
      muteMic={session.muteMic}
      muteSpk={session.muteSpk}
      textOnly={session.textOnly}
      draft={session.draft}
      approvals={props.approvals}
      work={work}
      showWork
      onWork={setWork}
      onMuteMic={() => session.setMuteMic((v) => !v)}
      onMuteSpk={() => session.setMuteSpk((v) => !v)}
      onDraft={session.setDraft}
      onSend={session.sendDraft}
      onHangup={session.hangup}
    />
  );
}

/** CallSurface 投影通话层。紧凑时只留立绘和挂断；展开时有计时、字幕、工作开关、草稿、静音和收起。交互交给回调。props 读取 phase、textOnly、who、emotion、compact、onHangup 等，调用 onWork，调用 onSend，调用 onDraft。 */
export function CallSurface(props: {
  compact?: boolean;
  who: CallWho;
  phase: CallPhase;
  emotion?: string;
  caption: string;
  notice: string;
  elapsed: number;
  muteMic: boolean;
  muteSpk: boolean;
  textOnly: boolean;
  draft: string;
  approvals?: number;
  work?: boolean;
  showWork?: boolean;
  onWork?: (v: boolean) => void;
  onMuteMic: () => void;
  onMuteSpk: () => void;
  onDraft: (v: string) => void;
  onSend: () => void;
  onHangup: () => void;
  onCollapse?: () => void;
}) {
  const c = useCopy().idol;
  const phaseLabel = props.phase === "dialing" ? c.dialing
    : props.phase === "listening" ? (props.textOnly ? c.typeStatus : c.listening)
    : props.phase === "thinking" ? c.thinking
    : props.phase === "speaking" ? c.speaking
    : props.phase === "ended" ? c.ended
    : c.callFailed;
  const wash = `radial-gradient(120% 80% at 50% 18%, ${hexAlpha(props.who.color, 0.78)}, #141418 62%)`;
  const posed = !!standURL(props.who.stand);
  const figure = (
    <IdolPresence
      name={props.who.name}
      color={props.who.color}
      portrait={props.who.portrait}
      kind={props.who.kind}
      stand={props.who.stand}
      phase={props.phase}
      emotion={props.emotion || "calm"}
      textOnly={props.textOnly}
      framing="full"
    />
  );
  if (props.compact) {
    return (
      <div className="idol-call-screen is-compact" data-testid="idol-call">
        {figure}
        <button type="button" className="idol-call-hangup" aria-label={c.hangup} onClick={props.onHangup}>
          <PhoneOff className="size-5" />
        </button>
      </div>
    );
  }
  return (
    <div className="idol-call-screen" data-testid="idol-call-layer">
      <div className="idol-call-wash" style={{ background: wash }} />
      {!posed && props.who.portrait ? <img className="idol-call-blur" alt="" src={imageURL(props.who.portrait)} /> : null}
      {figure}
      <div className="idol-call-shade" aria-hidden />
      <div className="relative z-[2] flex items-center justify-between gap-2 px-4 pt-3 text-[12px] text-white/80">
        <span className="min-w-0 truncate">{props.who.name || c.title}</span>
        <span className="shrink-0 tabular-nums">{props.phase === "dialing" ? c.dialing : mmss(props.elapsed)}</span>
        {props.onCollapse ? (
          <button type="button" className="shrink-0 rounded-md px-2 py-1 text-white/80 hover:bg-white/10" onClick={props.onCollapse}>{c.collapseCall}</button>
        ) : null}
      </div>
      <div className="relative z-[2] mt-auto flex w-full flex-col items-center gap-2 px-4 pb-4">
        <div role="status" className="text-[12px] text-white/75">{phaseLabel}</div>
        {props.approvals ? <div className="text-center text-[12px] text-amber-200">{c.approval}</div> : null}
        {props.notice ? <div role="status" className="max-w-sm text-center text-[13px] text-amber-100">{props.notice}</div> : null}
        <p className={cn("max-w-md overflow-auto text-center text-white/95", props.compact ? "max-h-16 text-[13px]" : "max-h-28 text-[16px] leading-relaxed")}>
          {props.caption || (props.phase === "dialing" ? c.dialing : c.listening)}
        </p>
        {props.showWork ? (
          <label className="flex items-center justify-center gap-2 text-[11px] text-white/70">
            <input type="checkbox" checked={!!props.work} onChange={(e) => props.onWork?.(e.target.checked)} />
            {c.work}
          </label>
        ) : null}
        <form className="flex w-full gap-2" onSubmit={(e) => { e.preventDefault(); props.onSend(); }}>
          <input
            className="h-10 min-w-0 flex-1 rounded-full bg-white/10 px-4 text-[13px] text-white outline-none placeholder:text-white/45"
            value={props.draft}
            placeholder={c.type}
            onChange={(e) => props.onDraft(e.target.value)}
          />
        </form>
        <div className="flex items-center justify-center gap-3">
        <Round label={props.muteMic ? c.unmuteMic : c.muteMic} onClick={props.onMuteMic}>
          {props.muteMic ? <MicOff className="size-4" /> : <Mic className="size-4" />}
        </Round>
        <button type="button" className="grid size-12 place-items-center rounded-full bg-red-500 text-white" aria-label={c.hangup} onClick={props.onHangup}>
          <PhoneOff className="size-5" />
        </button>
        <Round label={props.muteSpk ? c.unmuteSpk : c.muteSpk} onClick={props.onMuteSpk}>
          {props.muteSpk ? <VolumeX className="size-4" /> : <Volume2 className="size-4" />}
        </Round>
        </div>
      </div>
    </div>
  );
}

/** CallBar 渲染收起后的通话条。点主体展开，点挂断结束。props 读取 onExpand、dialing、name、elapsed、onHangup。 */
export function CallBar(props: { name: string; elapsed: number; dialing: boolean; onExpand: () => void; onHangup: () => void }) {
  const c = useCopy().idol;
  return (
    <div className="flex h-10 shrink-0 items-center gap-2 px-3 text-[13px] text-white" style={{ background: "#1c1c1f" }}>
      <button type="button" className="flex min-w-0 flex-1 items-center gap-2 text-left" onClick={props.onExpand}>
        <span className={cn("size-2 shrink-0 rounded-full bg-white", props.dialing && "animate-pulse")} />
        <span className="truncate">{props.name}</span>
        <span className="shrink-0 tabular-nums text-white/80">{props.dialing ? c.dialing : c.inCall.replace("{time}", mmss(props.elapsed))}</span>
        <span className="ml-auto shrink-0 text-white/70">{c.backToCall}</span>
      </button>
      <button type="button" className="grid size-7 place-items-center rounded-full bg-red-500" aria-label={c.hangup} onClick={props.onHangup}>
        <PhoneOff className="size-3.5" />
      </button>
    </div>
  );
}

/** 一颗圆形图标按钮。props 读取 label、onClick、children。 */
function Round(props: { label: string; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" className="grid size-10 place-items-center rounded-full bg-white/10 text-white" aria-label={props.label} onClick={props.onClick}>
      {props.children}
    </button>
  );
}
