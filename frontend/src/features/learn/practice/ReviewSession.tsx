/**
 * 拥有：一轮复习：一次一道题，作答、核对、看答案、评分，直到这一轮的题走完。
 * 不拥有：对错和日程，核对和评分都交给引擎；评分按钮上的间隔也是引擎给的。问导师经 onSay。
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { Star } from "lucide-react";
import { getLocale, useCopy } from "../../../lib/i18n";
import * as practice from "../../../lib/learn-practice";
import type { CheckResult, Question, Rating } from "../../../lib/learn-practice";
import { RATINGS } from "../../../lib/learn-practice";
import { fill } from "../../../lib/learn-format";
import { newRequestId, objective, toggleLabel } from "../../../lib/learn-practice-util";
import { errMessage } from "../../../lib/normalize";
import { Markdown } from "../../../lib/markdown";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { EmptyState } from "../../../components/ui/empty-state";
import { ProgressHairline } from "../../../components/ui/progress-hairline";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../../components/ui/dropdown-menu";
import { cn } from "../../../lib/utils";
import { LearnShell } from "../LearnShell";
import { dueLabel, intervalLabel, OptionKey, stateLabel, textareaClass, typeLabel } from "./parts";

/** SessionPlan 说明这一轮练什么。ids 非空时只练点名的题；lead 是要排在最前面的那道；其余是队列的筛选。 */
export type SessionPlan = { ids?: string[]; lead?: string; mistakesOnly?: boolean; tag?: string };

/** typing 在按键落在输入框里时为真，这时数字键和回车不当快捷键。 */
function typing(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el || !el.tagName) return false;
  return el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable;
}

/**
 * ReviewSession 走完一轮复习。plan 决定练哪些题。sessionId 是练习的导师对话，请模型判分时用它的模型。
 * onExit 回到练习首页。onSay 把一句话发给导师。onCard 在换题时报出当前题干，导师栏的标题跟着它。
 */
export function ReviewSession(props: {
  plan: SessionPlan;
  sessionId?: string;
  onExit: () => void;
  onSay: (text: string) => void;
  onCard?: (stem: string) => void;
}) {
  const copy = useCopy().learn;
  const L = copy.drill;
  const { plan } = props;
  const planKey = JSON.stringify(plan);
  const [cards, setCards] = useState<Question[]>([]);
  const [index, setIndex] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const [done, setDone] = useState(false);
  const [more, setMore] = useState(0);
  const [stats, setStats] = useState({ reviewed: 0, again: 0 });
  const [answer, setAnswer] = useState("");
  const [result, setResult] = useState<CheckResult | null>(null);
  const [busy, setBusy] = useState<"" | "check" | "judge" | "rate">("");
  const [err, setErr] = useState("");
  const requestRef = useRef(newRequestId());
  const first = useRef(true);

  /** 读出这一轮的题。第一次读时把 lead 排到最前面。读不出来时把原因留在页面上。 */
  const load = useCallback(async () => {
    setErr("");
    try {
      let rows: Question[];
      if (plan.ids && plan.ids.length > 0) {
        rows = await practice.queue({ ids: plan.ids });
      } else {
        rows = await practice.queue({ mistakesOnly: plan.mistakesOnly, tag: plan.tag });
        if (first.current && plan.lead) {
          const lead = rows.find((q) => q.id === plan.lead) || (await practice.queue({ ids: [plan.lead] }))[0];
          if (lead) rows = [lead, ...rows.filter((q) => q.id !== lead.id)];
        }
      }
      first.current = false;
      setCards(rows);
      setIndex(0);
      setDone(rows.length === 0);
    } catch (e) {
      setErr(`${L.loadFailed} ${errMessage(e)}`);
    } finally {
      setLoaded(true);
    }
  }, [planKey]);

  useEffect(() => {
    first.current = true;
    setStats({ reviewed: 0, again: 0 });
    void load();
  }, [load]);

  const card = done ? undefined : cards[index];
  const cardId = card?.id || "";
  const stem = card?.stem || "";

  useEffect(() => {
    setAnswer("");
    setResult(null);
    setErr("");
    requestRef.current = newRequestId();
    props.onCard?.(stem);
  }, [cardId]);

  useEffect(() => () => props.onCard?.(""), []);

  const isObjective = !!card && objective(card.qtype);
  const options = card?.options || [];
  const ready = !!answer.trim();
  const onlyAgain = isObjective && result?.correct === false;

  /** 核对作答并亮出答案。judge 为真时请模型判简答和开放题。 */
  const runCheck = async (judge: boolean) => {
    if (!card || busy) return;
    setBusy(judge ? "judge" : "check");
    setErr("");
    try {
      setResult(await practice.check(card.id, answer.trim(), { judge, session: props.sessionId }));
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy("");
    }
  };

  /** 这一轮的题走完后，看看还有没有可练的。点名练习不再往下找。 */
  const finish = async () => {
    setDone(true);
    if (plan.ids && plan.ids.length > 0) {
      setMore(0);
      return;
    }
    try {
      const sum = await practice.summary();
      setMore(sum.due + sum.new);
    } catch {
      setMore(0);
    }
  };

  /** 往下走一题。 */
  const advance = () => {
    if (index + 1 < cards.length) setIndex(index + 1);
    else void finish();
  };

  /** 按评分排下一次复习，然后换下一题。自评的题里，模型的判断只在学习者同样评为重来时才算作答错。 */
  const rate = async (rating: Rating) => {
    if (!card || !result || busy) return;
    if (onlyAgain && rating !== "again") return;
    let correct: boolean | null | undefined = result.correct;
    if (!isObjective) {
      if (rating === "again") correct = result.correct === false ? false : undefined;
      else correct = result.correct === true ? true : undefined;
    }
    setBusy("rate");
    setErr("");
    try {
      await practice.review(card.id, rating, { answer: answer.trim(), correct, requestId: requestRef.current });
      setStats((s) => ({ reviewed: s.reviewed + 1, again: s.again + (rating === "again" ? 1 : 0) }));
      advance();
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy("");
    }
  };

  /** 改当前这道题的收藏或暂停。暂停之后直接换下一题。 */
  const flag = async (flags: { bookmarked?: boolean; suspended?: boolean }) => {
    if (!card) return;
    try {
      await practice.flagQuestions([card.id], flags);
      if (flags.suspended) {
        advance();
        return;
      }
      setCards((list) => list.map((q) => (q.id === card.id ? { ...q, ...flags } : q)));
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  const keys = useRef({ rate, runCheck, result, busy, ready, isObjective, onlyAgain });
  keys.current = { rate, runCheck, result, busy, ready, isObjective, onlyAgain };
  useEffect(() => {
    /** 亮了答案之后数字键 1 到 4 评分；之前回车核对或看答案。在输入框里打字时不抢键。 */
    const onKey = (e: KeyboardEvent) => {
      const k = keys.current;
      if (e.metaKey || e.ctrlKey || e.altKey || k.busy) return;
      if (k.result) {
        if (typing(e.target)) return;
        const at = ["1", "2", "3", "4"].indexOf(e.key);
        if (at < 0) return;
        e.preventDefault();
        void k.rate(RATINGS[at]);
        return;
      }
      if (e.key !== "Enter" || (e.target as HTMLElement | null)?.tagName === "TEXTAREA") return;
      if (k.isObjective && !k.ready) return;
      e.preventDefault();
      void k.runCheck(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  if (!loaded) {
    return <LearnShell title={L.title} back={{ label: L.backToDesk, onClick: props.onExit }}><div /></LearnShell>;
  }

  if (!card) {
    return (
      <LearnShell title={L.title} back={{ label: L.backToDesk, onClick: props.onExit }}>
        <div className="mx-auto flex max-w-[640px] flex-col gap-4 px-8 py-6" data-testid="learn-practice-done">
          {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
          <EmptyState
            title={stats.reviewed > 0 ? L.roundDone : L.clear}
            body={stats.reviewed > 0 ? fill(L.roundStats, { n: stats.reviewed, again: stats.again }) : L.clearHint}
          />
          {stats.again > 0 ? <p className="text-center text-ui-footnote text-muted">{L.roundLater}</p> : null}
          <div className="flex items-center justify-center gap-2">
            {more > 0 ? (
              <Button size="sm" className="whitespace-nowrap" onClick={() => { void load(); }}>{L.keepGoing}<span className="text-background/70"> · {fill(L.roundMore, { n: more })}</span></Button>
            ) : null}
            <Button size="sm" variant={more > 0 ? "ghost" : "default"} className="whitespace-nowrap" onClick={props.onExit}>{L.backToDesk}</Button>
          </div>
        </div>
      </LearnShell>
    );
  }

  const meta = [typeLabel(L, card.qtype), card.mistake ? L.mistake : stateLabel(L, card.state)];
  if (card.state !== "new") meta.push(dueLabel(L, card, getLocale()));
  const shown = result?.reveal.answer || "";
  const tone = result?.correct === true ? "text-success" : result?.correct === false ? "text-danger" : "text-muted";
  const verdict = result?.correct === true ? L.verdictRight : result?.correct === false ? L.verdictWrong : L.verdictOpen;
  const pickable = card.qtype === "choice" || card.qtype === "multi" || card.qtype === "true_false";

  return (
    <LearnShell
      title={L.title}
      subtitle={fill(L.progress, { i: index + 1, n: cards.length })}
      back={{ label: L.endSession, onClick: props.onExit }}
      action={(
        <div className="flex items-center gap-1">
          <Button
            size="icon"
            variant="ghost"
            aria-label={card.bookmarked ? L.unbookmark : L.bookmark}
            aria-pressed={card.bookmarked}
            onClick={() => { void flag({ bookmarked: !card.bookmarked }); }}
          >
            <Star className={cn("size-4", card.bookmarked && "fill-current text-foreground")} aria-hidden />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="whitespace-nowrap">{copy.more}</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void flag({ suspended: true }); }}>{L.suspend}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}
    >
      <div className="mx-auto flex max-w-[640px] flex-col gap-5 px-8 py-6">
        <ProgressHairline value={cards.length > 0 ? (index / cards.length) * 100 : 0} />
        <article data-testid="learn-practice-card" data-qtype={card.qtype} data-phase={result ? "revealed" : "answer"} className="flex flex-col gap-4">
          <div>
            <p className="text-ui-footnote text-muted">{meta.join(" · ")}{card.tags.length > 0 ? <span className="text-muted/70"> · {card.tags.map((t) => `#${t}`).join(" ")}</span> : null}</p>
            <div className="mt-2 text-ui-title-s font-medium leading-[1.55] tracking-[-0.01em]">
              <Markdown text={card.stem} rich={false} />
            </div>
          </div>

          {result ? (
            <div className="flex flex-col gap-3" aria-live="polite">
              {pickable && options.length > 0 ? (
                <OptionKey options={options} answer={shown} picked={answer} />
              ) : (
                <div className="flex flex-col gap-1 text-ui-body">
                  {answer.trim() ? <p><span className="text-muted">{L.yourAnswer}</span><span className="text-muted/50"> · </span>{answer.trim()}</p> : null}
                  {shown ? <p><span className="text-muted">{L.answer}</span><span className="text-muted/50"> · </span>{card.qtype === "fill" ? shown.split("|").join(" / ") : shown}</p> : null}
                </div>
              )}
              <p className={cn("text-ui-body font-medium", tone)} data-testid="learn-practice-verdict">{verdict}</p>
              {result.feedback ? <p className="text-ui-body text-muted">{result.feedback}</p> : null}
              {result.reveal.explanation ? (
                <div className="text-ui-body text-muted">
                  <p className="text-ui-footnote text-muted/80">{L.explanation}</p>
                  <Markdown text={result.reveal.explanation} quiet rich={false} />
                </div>
              ) : null}
            </div>
          ) : pickable && options.length > 0 ? (
            <div className="flex flex-col gap-1" role={card.qtype === "multi" ? "group" : "radiogroup"} aria-label={typeLabel(L, card.qtype)}>
              {options.map((o) => {
                const multi = card.qtype === "multi";
                const on = multi ? answer.split(",").includes(o.label) : answer === o.label;
                return (
                  <button
                    key={o.label}
                    type="button"
                    role={multi ? "checkbox" : "radio"}
                    aria-checked={on}
                    disabled={!!busy}
                    onClick={() => setAnswer(multi ? toggleLabel(options, answer, o.label) : o.label)}
                    className={cn(
                      "flex items-start gap-2.5 rounded-lg border border-border/60 px-3 py-2 text-left text-ui-body transition-colors duration-[var(--duration-fast)]",
                      on ? "border-accent/60 bg-accent/10" : "hover:bg-lift",
                    )}
                  >
                    <span className="mt-px w-4 shrink-0 font-medium tabular-nums text-muted">{o.label}</span>
                    <span className="min-w-0 flex-1 break-words">{o.body}</span>
                  </button>
                );
              })}
            </div>
          ) : card.qtype === "fill" || isObjective ? (
            <Input value={answer} placeholder={L.fillPh} aria-label={L.fillPh} autoFocus disabled={!!busy} onChange={(e) => setAnswer(e.target.value)} />
          ) : (
            <textarea
              className={cn(textareaClass, "min-h-24")}
              value={answer}
              placeholder={L.answerPh}
              aria-label={L.answerPh}
              disabled={!!busy}
              onChange={(e) => setAnswer(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault();
                  void runCheck(false);
                }
              }}
            />
          )}

          {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}

          {result ? (
            <div className="flex flex-col gap-2 border-t border-border/50 pt-4">
              {onlyAgain ? null : <p className="text-ui-footnote text-muted">{L.rateLabel}</p>}
              <div className={cn("grid gap-2", onlyAgain ? "grid-cols-1" : "grid-cols-4")}>
                {(onlyAgain ? (["again"] as Rating[]) : RATINGS).map((rating) => (
                  <Button
                    key={rating}
                    variant={result.suggested === rating || (onlyAgain && rating === "again") ? "default" : "outline"}
                    className="h-auto w-full flex-col gap-0 whitespace-nowrap px-1 py-1.5"
                    disabled={!!busy}
                    data-rating={rating}
                    onClick={() => { void rate(rating); }}
                  >
                    <span className="text-ui-body">{L[rating]}</span>
                    <span className="text-ui-caption tabular-nums opacity-70">{intervalLabel(L, result.intervals?.[rating] ?? 0)}</span>
                  </Button>
                ))}
              </div>
              <div>
                <Button
                  size="sm"
                  variant="ghost"
                  className="whitespace-nowrap"
                  onClick={() => props.onSay(fill(L.sayExplain, {
                    stem: card.stem,
                    answer: pickable ? options.filter((o) => shown.split(",").includes(o.label)).map((o) => `${o.label}. ${o.body}`).join("; ") || shown : shown || L.noAnswerGiven,
                    mine: (pickable ? options.filter((o) => answer.split(",").includes(o.label)).map((o) => `${o.label}. ${o.body}`).join("; ") : answer.trim()) || L.noAnswerGiven,
                  }))}
                >
                  {L.askTutor}
                </Button>
              </div>
            </div>
          ) : isObjective ? (
            <div>
              <Button size="sm" className="whitespace-nowrap" disabled={!ready || !!busy} onClick={() => { void runCheck(false); }}>{busy === "check" ? L.checking : L.check}</Button>
            </div>
          ) : (
            <div className="flex items-center gap-1.5">
              <Button size="sm" className="whitespace-nowrap" disabled={!!busy} onClick={() => { void runCheck(false); }}>{L.reveal}</Button>
              <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={!ready || !!busy} onClick={() => { void runCheck(true); }}>{busy === "judge" ? L.judging : L.judge}</Button>
            </div>
          )}
        </article>
      </div>
    </LearnShell>
  );
}
