/**
 * 拥有：主会话里一条开口短讯下面的「为什么是这条」、依据，以及反馈。用上了、没用、此刻别提醒，记忆来的短讯还有事实错了。
 * 不拥有：开口门怎么算。反馈只记到这条短讯的类别上，按住和打开由 Go 的开口记录决定；依据和理由由 Go 写进载荷。
 */

import { useState } from "react";
import { senseRefs } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { givenFeedback, sendFeedback, type FeedbackKind } from "./desk-feed";
import { speakClass } from "./work-model";

/** SpeakFeedback 在答过之后收成一句回执。认不出类别的旧短讯不显示。理由和依据答过之后也留着。 */
export function SpeakFeedback(props: { payload: Record<string, unknown> }) {
  const copy = useCopy();
  const noteId = String(props.payload.id || "");
  const className = speakClass(props.payload);
  const memoryId = String(props.payload.memoryId || "");
  const reason = String(props.payload.reason || "").trim();
  const refs = senseRefs(props.payload.refs);
  const [given, setGiven] = useState<FeedbackKind | "">(() => givenFeedback(noteId));
  const [busy, setBusy] = useState(false);
  if (!className) return null;
  const why = reason || refs.length ? (
    <p className="mt-2 text-ui-footnote leading-5 text-muted" data-testid="speak-why">
      {reason}
      {refs.length ? (
        <span data-testid="speak-refs">
          {reason ? " · " : ""}
          {refs.map((ref, i) => (
            <span key={`${ref.kind}:${ref.label}:${i}`} className="tabular-nums">
              {i ? "、" : ""}
              {ref.label}
            </span>
          ))}
        </span>
      ) : null}
    </p>
  ) : null;
  if (given) {
    const done: Partial<Record<FeedbackKind, string>> = {
      used: copy.work.noteUsedDone,
      ignore: copy.work.noteIgnoreDone,
      later: copy.work.noteLaterDone,
      wrong: copy.work.noteWrongDone,
    };
    return (
      <>
        {why}
        <p className="mt-2 text-ui-footnote text-muted" data-testid="speak-feedback-done">{done[given] || copy.work.noteUsedDone}</p>
      </>
    );
  }
  const actions: { kind: FeedbackKind; label: string }[] = [
    { kind: "used", label: copy.work.noteUsed },
    { kind: "ignore", label: copy.work.noteIgnore },
    { kind: "later", label: copy.work.noteLater },
  ];
  if (memoryId) actions.push({ kind: "wrong", label: copy.work.noteWrong });
  async function act(kind: FeedbackKind) {
    if (busy) return;
    setBusy(true);
    const ok = await sendFeedback(className, kind, memoryId, noteId);
    setBusy(false);
    if (ok) setGiven(kind);
  }
  return (
    <>
      {why}
      <div className="mt-2 flex flex-wrap items-center gap-1" data-testid="speak-feedback">
        {actions.map((action) => (
          <button
            key={action.kind}
            type="button"
            disabled={busy}
            className="h-6 rounded-md px-2 text-ui-caption font-medium text-muted transition-colors duration-[var(--duration-fast)] hover:bg-lift hover:text-foreground disabled:opacity-50"
            onClick={() => { void act(action.kind); }}
          >
            {action.label}
          </button>
        ))}
      </div>
    </>
  );
}
