// 本文件拥有一次提问的翻页、选项和空闲倒计时。不拥有问题怎么解析，那一步在 parseAskCommand。调用方传入 approval 和 onResolve。

import { useEffect, useRef, useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { Approval } from "../lib/protocol";
import {
  ASK_IDLE_MS,
  askCanIdle,
  chosenAnswer,
  idleAnswer,
  parseAskCommand,
  recommendedPicks,
} from "../lib/ask-form";
import { useCopy } from "../lib/i18n";
import { cn } from "../lib/utils";

/** AskDeck 没有当前问题时不渲染。指针或键盘一碰上就停住倒计时。秒数至少显示 1。approval 读取 command、id。onResolve 被调用，并传入 approval.id、"once"。 */
export function AskDeck({
  approval,
  onResolve,
}: {
  approval: Approval;
  onResolve: (id: string, decision: string, answer?: string) => void;
}) {
  const copy = useCopy();
  const form = parseAskCommand(approval.command);
  const canIdle = askCanIdle(form);
  const [index, setIndex] = useState(0);
  const [held, setHeld] = useState(false);
  const [remain, setRemain] = useState(ASK_IDLE_MS);
  const [picks, setPicks] = useState(() => recommendedPicks(form));
  const [free, setFree] = useState<Record<string, string>>({});
  const done = useRef(false);
  const formRef = useRef(form);
  formRef.current = form;

  const page = Math.min(index, Math.max(0, form.questions.length - 1));
  const question = form.questions[page];
  const seconds = Math.max(1, Math.ceil(remain / 1000));

  /** finish 已经交过就不再交。决策固定为 once。answer 传给 onResolve。 */
  function finish(answer: string) {
    if (done.current) return;
    done.current = true;
    onResolve(approval.id, "once", answer);
  }

  // 没按住且允许空闲时，每 200 毫秒减一次。减到 0 就用 idleAnswer 交卷。
  useEffect(() => {
    if (held || !canIdle) return;
    const start = Date.now();
    const tick = window.setInterval(() => {
      const left = ASK_IDLE_MS - (Date.now() - start);
      if (left <= 0) {
        window.clearInterval(tick);
        finish(idleAnswer(formRef.current));
        return;
      }
      setRemain(left);
    }, 200);
    return () => window.clearInterval(tick);
  }, [held, canIdle, approval.id]);

  if (!question) return null;
  const many = form.questions.length > 1;
  const progress = copy.transcript.askProgress
    .replace("{n}", String(page + 1))
    .replace("{total}", String(form.questions.length));

  return (
    <section
      className="rounded-2xl border border-border bg-card px-3.5 py-3 shadow-[var(--shadow-card)]"
      data-testid="ask-deck"
      role="region"
      aria-label={copy.transcript.askDeck}
      onPointerDownCapture={() => setHeld(true)}
      onKeyDownCapture={() => setHeld(true)}
    >
      <div className="flex items-center gap-2 text-[11px] font-medium text-muted">
        <span>{form.title || copy.transcript.askDeck}</span>
        {many ? <span className="ml-auto tabular-nums">{progress}</span> : null}
      </div>
      <p className="mt-1 text-[14px] font-medium tracking-tight text-foreground">{question.prompt}</p>
      {question.options.length ? (
        <div className="mt-2.5 grid gap-1.5" role="radiogroup" aria-label={question.prompt}>
          {question.options.map((opt) => {
            const on = (picks[question.id] || question.recommended) === opt.id;
            return (
              <button
                key={opt.id}
                type="button"
                role="radio"
                aria-checked={on}
                className={cn(
                  "flex w-full items-start gap-2 rounded-xl border px-3 py-2 text-left text-[13px] leading-snug transition-colors",
                  on ? "border-foreground/25 bg-lift text-foreground" : "border-border/70 text-foreground/90 hover:bg-lift",
                )}
                onClick={() => setPicks((prev) => ({ ...prev, [question.id]: opt.id }))}
              >
                <span className="min-w-0 flex-1">{opt.label}</span>
                {opt.id === question.recommended ? (
                  <span className="shrink-0 text-[11px] text-muted">{copy.transcript.askSuggested}</span>
                ) : null}
              </button>
            );
          })}
        </div>
      ) : (
        <textarea
          className="mt-2 min-h-[4.5rem] w-full resize-y rounded-lg border border-border/60 bg-sidebar/70 px-3 py-2 text-[13px] leading-[1.5] text-foreground outline-none focus:border-foreground/30"
          value={free[question.id] || ""}
          placeholder={copy.transcript.answerPlaceholder}
          autoComplete="off"
          spellCheck={false}
          onChange={(e) => setFree((prev) => ({ ...prev, [question.id]: e.target.value }))}
        />
      )}
      <div className="mt-3 flex flex-wrap items-center gap-1.5">
        {many ? (
          <>
            <button
              type="button"
              className="inline-flex items-center gap-1 rounded-full border border-border px-2.5 py-1.5 text-[12px] text-foreground disabled:opacity-40"
              disabled={page === 0}
              onClick={() => setIndex((n) => Math.max(0, n - 1))}
            >
              <ChevronLeft className="size-3.5" aria-hidden />
              {copy.transcript.askPrev}
            </button>
            <button
              type="button"
              className="inline-flex items-center gap-1 rounded-full border border-border px-2.5 py-1.5 text-[12px] text-foreground disabled:opacity-40"
              disabled={page >= form.questions.length - 1}
              onClick={() => setIndex((n) => Math.min(form.questions.length - 1, n + 1))}
            >
              {copy.transcript.askNext}
              <ChevronRight className="size-3.5" aria-hidden />
            </button>
          </>
        ) : null}
        <button
          type="button"
          className="inline-flex items-center rounded-full bg-foreground px-3 py-1.5 text-[12px] font-medium text-background hover:opacity-[0.92]"
          onClick={() => finish(chosenAnswer(form, picks, free))}
        >
          {copy.transcript.askContinue}
        </button>
        <span className="ml-auto text-[11px] text-muted">
          {held ? copy.transcript.askHold : canIdle ? copy.transcript.askIdle.replace("{n}", String(seconds)) : ""}
        </span>
      </div>
    </section>
  );
}
