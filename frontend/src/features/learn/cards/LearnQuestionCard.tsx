/**
 * 拥有：一道掌握路径的题在界面上的样子：作答、放弃、换一题，以及判完之后的结论和解析。
 * 不拥有：判分。作答交给引擎，引擎判完卡片才显示答案。也不决定下一步学什么。
 */

import { useCallback, useEffect, useState } from "react";
import { useCopy } from "../../../lib/i18n";
import * as learn from "../../../lib/learn-api";
import type { GradeResult, PublicQuestion, QuestionState, QuizOption } from "../../../lib/learn-api";
import { subscribeLearn } from "../../../lib/learn-events";
import { errMessage } from "../../../lib/normalize";
import { Markdown } from "../../../lib/markdown";
import { Button } from "../../../components/ui/button";
import { cn } from "../../../lib/utils";

/** pickedLabel 从学习者的作答里认出选的是哪个选项。作答是字母或选项正文。认不出时返回空串。 */
function pickedLabel(options: QuizOption[], answer: string | undefined): string {
  const text = (answer || "").trim();
  if (!text) return "";
  const byLabel = options.find((o) => o.label.toLowerCase() === text.toLowerCase());
  if (byLabel) return byLabel.label;
  return options.find((o) => o.body === text)?.label || "";
}

/** optionText 把字母还原成「字母. 正文」，不是选项字母时原样返回。 */
function optionText(options: QuizOption[], label: string | undefined): string {
  const hit = options.find((o) => o.label === label);
  return hit ? `${hit.label}. ${hit.body}` : label || "";
}

/**
 * LearnQuestionCard 画出一道题。题目还在等时可以作答；判完后显示结论、答案和解析。
 * pathId 和 questionId 定位这道题。initial 是出题时就有的题面，状态读回来之前先用它。
 * sessionId 决定判分用哪段对话的模型。onGraded 在引擎给出最终判定后调用，带上学习者说的话。onSkipped 在换一题之后调用。
 */
export function LearnQuestionCard(props: {
  pathId: string;
  questionId: string;
  initial?: PublicQuestion;
  sessionId?: string;
  onGraded?: (said: string, result: GradeResult) => void;
  onSkipped?: () => void;
}) {
  const L = useCopy().learn.mastery;
  const { pathId, questionId } = props;
  const [state, setState] = useState<QuestionState | null>(props.initial ? { question: props.initial, status: "pending" } : null);
  const [gone, setGone] = useState(false);
  const [picked, setPicked] = useState("");
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");
  const [err, setErr] = useState("");

  /** 读回这道题现在的状态。题目已经不在时只留题面。 */
  const load = useCallback(async () => {
    try {
      setState(await learn.questionState(pathId, questionId));
      setGone(false);
    } catch {
      setGone(true);
    }
  }, [pathId, questionId]);

  useEffect(() => {
    void load();
    return subscribeLearn(/** 这条路径有变化就重读，另一处作答后这里也跟着变。 */ (change) => {
      if (change.scope === "path" && change.id === pathId) void load();
    });
  }, [load, pathId]);

  if (!state) return null;
  const q = state.question;
  const options = q.options || [];
  const choice = q.question_type === "choice" && options.length > 0;
  const pending = state.status === "pending" && !gone;
  const ready = choice ? !!picked : !!text.trim();

  /** 提交作答。引擎没法判定时题目继续等，只把原因显示出来。giveUp 为真表示不会。 */
  const submit = async (giveUp: boolean) => {
    if (busy) return;
    setBusy(true);
    setErr("");
    setNote("");
    try {
      const sent = giveUp ? "" : choice ? picked : text.trim();
      const res = await learn.answer(pathId, questionId, sent, { giveUp, session: props.sessionId });
      if (!res.final) {
        setNote(res.feedback || L.ungraded);
        return;
      }
      await load();
      props.onGraded?.(giveUp ? L.idkSaid : choice ? optionText(options, picked) : text.trim(), res);
    } catch (e) {
      setErr(`${L.gradeFailed} ${errMessage(e)}`);
    } finally {
      setBusy(false);
    }
  };

  /** 放弃这道题，不计分。 */
  const skip = async () => {
    if (busy) return;
    setBusy(true);
    setErr("");
    try {
      await learn.skipQuestion(pathId);
      await load();
      props.onSkipped?.();
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const graded = state.status === "graded";
  const mine = graded && choice ? pickedLabel(options, state.answer) : "";
  const verdict = state.result === "correct" ? L.resCorrect : state.result === "partial" ? L.resPartial : L.resIncorrect;
  const tone = state.result === "correct" ? "text-success" : state.result === "partial" ? "text-warning" : "text-danger";

  return (
    <div className="rounded-xl border border-border bg-card p-4 shadow-[var(--shadow-card)]" data-testid="learn-question-card" data-status={gone ? "gone" : state.status}>
      <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.cardEyebrow}</p>
      <div className="mt-1.5 text-ui-body-l leading-[1.6]">
        <Markdown text={q.prompt} rich={false} />
      </div>
      {choice ? (
        <div className="mt-3 flex flex-col gap-1" role={pending ? "radiogroup" : undefined}>
          {options.map((o) => {
            const on = pending ? picked === o.label : mine === o.label;
            const right = graded && state.expected === o.label;
            const wrong = graded && on && !right;
            return (
              <button
                key={o.label}
                type="button"
                role={pending ? "radio" : undefined}
                aria-checked={pending ? on : undefined}
                disabled={!pending || busy}
                onClick={() => setPicked(o.label)}
                className={cn(
                  "flex items-start gap-2.5 rounded-lg border border-border/60 px-3 py-2 text-left text-ui-body transition-colors duration-[var(--duration-fast)]",
                  pending && !on && "hover:bg-lift",
                  pending && on && "border-accent/60 bg-accent/10",
                  right && "border-success/50 bg-success/[0.11]",
                  wrong && "border-danger/50 bg-danger/[0.11]",
                  !pending && !right && !wrong && "opacity-70",
                )}
              >
                <span className="mt-px w-4 shrink-0 font-medium tabular-nums text-muted">{o.label}</span>
                <span className="min-w-0 flex-1">{o.body}</span>
              </button>
            );
          })}
        </div>
      ) : pending ? (
        <textarea
          className="mt-3 min-h-20 w-full resize-y rounded-lg border border-border/60 bg-transparent px-3 py-2 text-ui-body outline-none focus-visible:border-accent/60"
          value={text}
          placeholder={L.answerPh}
          aria-label={L.answerPh}
          disabled={busy}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && ready) {
              e.preventDefault();
              void submit(false);
            }
          }}
        />
      ) : null}
      {pending ? (
        <div className="mt-3 flex items-center gap-1.5">
          <Button size="sm" className="whitespace-nowrap" disabled={!ready || busy} onClick={() => { void submit(false); }}>
            {busy ? L.grading : L.submit}
          </Button>
          <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={busy} onClick={() => { void submit(true); }}>{L.idk}</Button>
          <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={busy} onClick={() => { void skip(); }}>{L.skip}</Button>
        </div>
      ) : null}
      {note ? <p className="mt-2 text-ui-footnote text-warning" aria-live="polite">{note}</p> : null}
      {err ? <p className="mt-2 text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
      {graded ? (
        <div className="mt-3 border-t border-border/50 pt-3 text-ui-body" aria-live="polite">
          <p className={cn("font-medium", tone)}>{verdict}</p>
          {state.feedback ? <p className="mt-1 text-muted">{state.feedback}</p> : null}
          {!choice && state.answer ? (
            <p className="mt-2"><span className="text-muted">{L.yourAnswer}</span><span className="text-muted/50"> · </span>{state.answer}</p>
          ) : null}
          {state.expected && state.result !== "correct" ? (
            <p className="mt-1"><span className="text-muted">{L.expected}</span><span className="text-muted/50"> · </span>{choice ? optionText(options, state.expected) : state.expected}</p>
          ) : null}
          {state.explanation ? (
            <div className="mt-2 text-muted">
              <p className="text-ui-footnote text-muted/80">{L.explanation}</p>
              <Markdown text={state.explanation} quiet rich={false} />
            </div>
          ) : null}
        </div>
      ) : null}
      {gone ? <p className="mt-3 text-ui-footnote text-muted">{L.cardGone}</p> : null}
      {!gone && state.status === "skipped" ? <p className="mt-3 text-ui-footnote text-muted">{L.skipped}</p> : null}
      {!gone && state.status === "voided" ? <p className="mt-3 text-ui-footnote text-muted">{L.voided}</p> : null}
    </div>
  );
}
