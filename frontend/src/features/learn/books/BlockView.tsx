/**
 * 拥有：课本里一块内容的样子：正文、分节讲解、提示框、代码、时间线、记忆卡片、随堂题和读者笔记，以及每块上的几个动作。
 * 不拥有：内容本身和随堂题的对错。作答交给引擎，答案在作答之后才显示。改动之后由调用方重读这一章。
 */

import { useState } from "react";
import { useCopy } from "../../../lib/i18n";
import * as books from "../../../lib/learn-books";
import type { BookBlock, QuizQuestion, QuizResult } from "../../../lib/learn-books";
import { errMessage } from "../../../lib/normalize";
import { Markdown } from "../../../lib/markdown";
import { Button } from "../../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../../components/ui/dropdown-menu";
import { cn } from "../../../lib/utils";

type TomeCopy = ReturnType<typeof useCopy>["learn"]["tome"];

const noteClass = "min-h-16 w-full resize-y rounded-lg border border-border/60 bg-transparent px-3 py-2 text-ui-body outline-none focus-visible:border-accent/60";

/** variantLabel 是提示框种类的名字。 */
function variantLabel(L: TomeCopy, variant: string): string {
  if (variant === "pitfall") return L.variantPitfall;
  if (variant === "tip") return L.variantTip;
  if (variant === "summary") return L.variantSummary;
  return L.variantKeyIdea;
}

/** Cards 画出一组记忆卡片。点一张翻面，再点翻回来。 */
function Cards(props: { cards: { front: string; back: string; hint?: string }[] }) {
  const L = useCopy().learn.tome;
  const [flipped, setFlipped] = useState<Record<number, boolean>>({});
  return (
    <div className="grid grid-cols-2 gap-2">
      {props.cards.map((card, i) => {
        const on = !!flipped[i];
        return (
          <button
            key={i}
            type="button"
            className="flex min-h-24 flex-col rounded-lg border border-border/60 p-3 text-left hover:bg-lift/40"
            aria-pressed={on}
            aria-label={on ? L.cardBack : L.cardFlip}
            onClick={() => setFlipped((cur) => ({ ...cur, [i]: !cur[i] }))}
          >
            <span className={cn("text-ui-body", on && "text-muted")}>{on ? card.back : card.front}</span>
            {!on && card.hint ? <span className="mt-auto pt-2 text-ui-caption text-muted/70">{L.cardHint} · {card.hint}</span> : null}
          </button>
        );
      })}
    </div>
  );
}

/**
 * Question 画出一道随堂题。bookId、blockId 和 question 定位这道题，sessionId 决定简答题用哪段对话的模型判。
 * 作答后显示对错、答案和解析；没答对可以再答一次。
 */
function Question(props: { bookId: string; blockId: string; question: QuizQuestion; sessionId?: string }) {
  const L = useCopy().learn.tome;
  const { question: q } = props;
  const [answer, setAnswer] = useState("");
  const [result, setResult] = useState<QuizResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const choice = q.qtype === "choice" && q.options.length > 0;

  /** 把作答交给引擎批改。 */
  const submit = async () => {
    if (!answer.trim() || busy) return;
    setBusy(true);
    setErr("");
    try {
      setResult(await books.answerQuiz(props.bookId, props.blockId, q.id, answer.trim(), props.sessionId));
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const tone = result?.correct === true ? "text-success" : result?.correct === false ? "text-danger" : "text-muted";
  return (
    <div className="flex flex-col gap-2" data-testid="learn-book-question">
      <div className="text-ui-body-l leading-relaxed"><Markdown text={q.question} rich={false} /></div>
      {choice ? (
        <div className="flex flex-col gap-1" role="radiogroup">
          {q.options.map((o) => {
            const on = answer === o.label;
            const right = !!result && result.answer === o.label;
            const wrong = !!result && on && !right;
            return (
              <button
                key={o.label}
                type="button"
                role="radio"
                aria-checked={on}
                disabled={!!result || busy}
                onClick={() => setAnswer(o.label)}
                className={cn(
                  "flex items-start gap-2.5 rounded-lg border border-border/60 px-3 py-2 text-left text-ui-body",
                  !result && on && "border-accent/60 bg-accent/10",
                  !result && !on && "hover:bg-lift",
                  right && "border-success/50 bg-success/[0.11]",
                  wrong && "border-danger/50 bg-danger/[0.11]",
                  result && !right && !wrong && "opacity-70",
                )}
              >
                <span className="mt-px w-4 shrink-0 font-medium tabular-nums text-muted">{o.label}</span>
                <span className="min-w-0 flex-1 break-words">{o.body}</span>
              </button>
            );
          })}
        </div>
      ) : (
        <textarea className={noteClass} value={answer} placeholder={L.quizAnswerPh} aria-label={L.quizAnswerPh} disabled={!!result || busy} onChange={(e) => setAnswer(e.target.value)} />
      )}
      {err ? <p className="text-ui-footnote text-danger">{err}</p> : null}
      {result ? (
        <div className="flex flex-col gap-1 text-ui-body" aria-live="polite">
          <p className={cn("font-medium", tone)}>{result.correct === true ? L.quizRight : result.correct === false ? L.quizWrong : L.quizOpen}</p>
          {result.feedback ? <p className="text-muted">{result.feedback}</p> : null}
          {!choice && result.answer ? <p><span className="text-muted">{L.quizAnswer}</span><span className="text-muted/50"> · </span>{result.answer}</p> : null}
          {result.explanation ? <div className="text-muted"><span className="text-ui-footnote text-muted/80">{L.quizWhy}</span><Markdown text={result.explanation} quiet rich={false} /></div> : null}
          {result.correct !== true ? (
            <div><Button size="xs" variant="ghost" onClick={() => { setResult(null); setAnswer(""); }}>{L.quizRetry}</Button></div>
          ) : null}
        </div>
      ) : (
        <div><Button size="sm" variant="outline" className="whitespace-nowrap" disabled={!answer.trim() || busy} onClick={() => { void submit(); }}>{busy ? L.quizChecking : L.quizSubmit}</Button></div>
      )}
    </div>
  );
}

/** Body 按种类画出一块写好的内容。 */
function Body(props: { bookId: string; block: BookBlock; sessionId?: string }) {
  const L = useCopy().learn.tome;
  const { block } = props;
  const p = block.payload || {};
  switch (block.kind) {
    case "section":
      return (
        <div className="flex flex-col gap-3">
          {block.title ? <h3 className="text-ui-title-s font-semibold tracking-[-0.02em]">{block.title}</h3> : null}
          {p.intro ? <Markdown text={String(p.intro)} /> : null}
          {(Array.isArray(p.subsections) ? p.subsections : []).map((sub: { heading: string; body: string }, i: number) => (
            <div key={i} className="flex flex-col gap-1">
              {sub.heading ? <h4 className="text-ui-body-l font-medium">{sub.heading}</h4> : null}
              <Markdown text={String(sub.body || "")} />
            </div>
          ))}
          {p.key_takeaway ? (
            <p className="border-l-2 border-foreground/40 pl-3 text-ui-body-l"><span className="text-ui-footnote text-muted">{L.takeaway}</span><br />{String(p.key_takeaway)}</p>
          ) : null}
        </div>
      );
    case "callout":
      return (
        <div className={cn("border-l-2 pl-3", p.variant === "pitfall" ? "border-warning/70" : "border-border")}>
          <p className={cn("text-ui-footnote", p.variant === "pitfall" ? "text-warning" : "text-muted")}>{variantLabel(L, String(p.variant || ""))}</p>
          <Markdown text={String(p.body || "")} />
        </div>
      );
    case "code":
      return (
        <div className="flex flex-col gap-2">
          {block.title ? <p className="text-ui-footnote text-muted">{block.title}</p> : null}
          <Markdown text={`\`\`\`${String(p.language || "")}\n${String(p.code || "")}\n\`\`\``} />
          {p.explanation ? <Markdown text={String(p.explanation)} quiet /> : null}
        </div>
      );
    case "timeline":
      return (
        <ol className="flex flex-col">
          {(Array.isArray(p.events) ? p.events : []).map((e: { date: string; title: string; description: string }, i: number) => (
            <li key={i} className="flex gap-3 border-b border-border/40 py-2 last:border-b-0">
              <span className="w-24 shrink-0 text-ui-footnote tabular-nums text-muted">{e.date}</span>
              <span className="min-w-0 flex-1">
                <span className="block text-ui-body-l font-medium">{e.title}</span>
                <span className="block text-ui-body text-muted">{e.description}</span>
              </span>
            </li>
          ))}
        </ol>
      );
    case "flash_cards":
      return (
        <div className="flex flex-col gap-2">
          <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.cards}</p>
          <Cards cards={Array.isArray(p.cards) ? p.cards : []} />
        </div>
      );
    case "quiz":
      return (
        <div className="flex flex-col gap-5 rounded-xl border border-border/60 p-4">
          <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.quiz}</p>
          {(Array.isArray(p.questions) ? p.questions : []).map((q: QuizQuestion) => (
            <Question key={q.id} bookId={props.bookId} blockId={block.id} question={{ ...q, options: Array.isArray(q.options) ? q.options : [] }} sessionId={props.sessionId} />
          ))}
        </div>
      );
    case "user_note":
      return (
        <div className="border-l-2 border-accent/50 pl-3">
          <p className="text-ui-footnote text-muted">{L.yourNote}</p>
          <p className="whitespace-pre-wrap text-ui-body-l">{String(p.body || "")}</p>
        </div>
      );
    default:
      return <Markdown text={String(p.body || "")} />;
  }
}

/**
 * BlockView 画出章节里的一块和它的动作。bookId 和 block 定位这一块，chapterId 是它所在的章。
 * sessionId 是这本书的导师对话，重写和判随堂题用它的模型。onChanged 在改动之后调用，调用方重读这一章。
 */
export function BlockView(props: { bookId: string; chapterId: string; block: BookBlock; sessionId?: string; onChanged: () => void }) {
  const L = useCopy().learn.tome;
  const { block, bookId } = props;
  const [mode, setMode] = useState<"" | "edit" | "note">("");
  const [draft, setDraft] = useState("");
  const [err, setErr] = useState("");
  const editable = block.kind === "text" || block.kind === "callout" || block.kind === "user_note";

  /** 跑一个改动，出错时把原因留在这一块上，成功后让调用方重读。 */
  const run = async (fn: () => Promise<unknown>) => {
    setErr("");
    try {
      await fn();
      setMode("");
      setDraft("");
      props.onChanged();
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  return (
    <div className="group/block flex flex-col gap-2" data-testid="learn-book-block" data-kind={block.kind} data-status={block.status}>
      <div className="flex items-start gap-2">
        <div className="prose-select min-w-0 flex-1 text-ui-body-l leading-[1.75]">
          {block.status === "ready" ? (
            mode === "edit" ? null : <Body bookId={bookId} block={block} sessionId={props.sessionId} />
          ) : block.status === "error" ? (
            <p className="text-ui-body text-danger">{L.blockFailed}{block.error ? <span className="text-muted"> {block.error}</span> : null}</p>
          ) : (
            <p className="text-ui-body text-muted">{block.title ? `${block.title} · ` : ""}{block.status === "generating" ? L.blockWriting : L.blockWaiting}</p>
          )}
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="xs" variant="ghost" className="shrink-0 opacity-0 focus-visible:opacity-100 group-hover/block:opacity-100 data-[state=open]:opacity-100">{L.more}</Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { setDraft(""); setMode("note"); }}>{L.addNote}</DropdownMenuItem>
            {editable && block.status === "ready" ? (
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { setDraft(String(block.payload?.body || "")); setMode("edit"); }}>{L.editBlock}</DropdownMenuItem>
            ) : null}
            {block.kind !== "user_note" ? (
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void run(() => books.regenerateBlock(bookId, block.id, props.sessionId)); }}>{L.rewriteBlock}</DropdownMenuItem>
            ) : null}
            <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void run(() => books.deleteBlock(bookId, block.id)); }}>{L.deleteBlock}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      {block.edited && block.kind !== "user_note" ? <p className="text-ui-caption text-muted/70">{L.edited}</p> : null}
      {err ? <p className="text-ui-footnote text-danger">{err}</p> : null}
      {mode ? (
        <form
          className="flex flex-col gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            const text = draft.trim();
            if (!text) return;
            void run(() => (mode === "edit" ? books.updateBlock(bookId, block.id, text) : books.insertNote(bookId, props.chapterId, block.id, text)));
          }}
        >
          <textarea className={noteClass} value={draft} placeholder={L.notePh} aria-label={L.notePh} autoFocus onChange={(e) => setDraft(e.target.value)} />
          <div className="flex items-center gap-2">
            <Button size="sm" type="submit" className="whitespace-nowrap" disabled={!draft.trim()}>{L.save}</Button>
            <Button size="sm" type="button" variant="ghost" className="whitespace-nowrap" onClick={() => setMode("")}>{L.cancel}</Button>
          </div>
        </form>
      ) : null}
    </div>
  );
}
