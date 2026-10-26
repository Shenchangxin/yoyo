/**
 * 拥有：作曲区被审批或提问占住时的面板。提问正文复用 AskDeck。
 * 不拥有：审批队列。决定交给外面的回调送出。不调用 api。
 */
import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import type { Approval } from "../../lib/protocol";
import { AskDeck } from "../AskDeck";
import { useCopy } from "../../lib/i18n";

/** ApprovalTakeover 占住作曲区，让人回答或作出决定。 */
export function ApprovalTakeover(props: {
  ask?: Approval | null;
  approval?: Approval | null;
  onResolve: (id: string, decision: string, answer?: string) => void;
}) {
  const copy = useCopy();
  const [answered, setAnswered] = useState(false);
  const composing = useRef(false);
  const compositionEnded = useRef(false);
  const item = props.ask || props.approval;
  useEffect(() => {
    setAnswered(false);
  }, [item?.id]);
  if (!item) return null;

  if (props.ask) {
    return (
      <div className="sc-takeover" data-testid="composer-takeover">
        <AskDeck key={props.ask.id} approval={props.ask} onResolve={props.onResolve} />
      </div>
    );
  }

  /** 同一条只接受一次。记下已回答再回调。 */
  const answer = (decision: "once" | "deny") => {
    if (answered) return;
    setAnswered(true);
    props.onResolve(item.id, decision);
  };

  /** 焦点在输入里时不抢键。回车允许一次，退出则拒绝。 */
  const onKey = (event: KeyboardEvent<HTMLDivElement>) => {
    const el = event.target as Element;
    if (event.defaultPrevented || !event.currentTarget.contains(document.activeElement)) return;
    if (el.closest("input, textarea, select, [contenteditable='true']")) return;
    if (event.key !== "Enter" && event.key !== "Escape") return;
    if (event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) return;
    event.preventDefault();
    event.stopPropagation();
    if (event.repeat || composing.current || compositionEnded.current || event.nativeEvent.isComposing || event.keyCode === 229) return;
    answer(event.key === "Enter" ? "once" : "deny");
  };

  const subject = item.command || item.path || item.level;
  return (
    <div
      className="sc-takeover"
      data-testid="composer-takeover"
      aria-busy={answered}
      onKeyDown={onKey}
      onKeyUpCapture={() => { compositionEnded.current = false; }}
      onCompositionStartCapture={() => { composing.current = true; }}
      onCompositionEndCapture={() => { composing.current = false; compositionEnded.current = true; }}
    >
      <div className="sc-takeover-card">
        <div className="sc-takeover-strip">
          <span className="size-2 rounded-full bg-warning" aria-hidden />
          {copy.transcript.needsApproval}
        </div>
        <div tabIndex={0} role="group" aria-label={copy.transcript.needsApproval}>
          <div className="text-[14px] font-medium tracking-tight text-foreground">{item.action || "action"}</div>
          {subject ? (
            <pre className="mt-2 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[12px] leading-[1.5] text-foreground/85">
              {subject}
            </pre>
          ) : null}
        </div>
        <div className="sc-takeover-actions">
          <button type="button" className="sc-btn sc-btn-outline" disabled={answered} onClick={() => answer("deny")}>
            {copy.transcript.reject}
          </button>
          <button type="button" className="sc-btn sc-btn-primary" disabled={answered} onClick={() => answer("once")}>
            {copy.transcript.allow}
          </button>
        </div>
      </div>
    </div>
  );
}
