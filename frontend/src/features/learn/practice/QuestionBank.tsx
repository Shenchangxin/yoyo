/**
 * 拥有：题库这一页：搜索和筛选、题目列表、点开后的答案和记录，以及写题、导入、出题三块面板的切换。
 * 不拥有：一轮复习，练哪些题经 onPractice 交给调用方。也不拥有日程和校验。
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight, Star } from "lucide-react";
import { getLocale, useCopy } from "../../../lib/i18n";
import * as practice from "../../../lib/learn-practice";
import type { BankFilter, Question, QuestionDetail } from "../../../lib/learn-practice";
import { subscribeLearn } from "../../../lib/learn-events";
import { fill, percent } from "../../../lib/learn-format";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { EmptyState } from "../../../components/ui/empty-state";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../components/ui/select";
import { cn } from "../../../lib/utils";
import { ConfirmDialog } from "../../ConfirmDialog";
import { LearnSegment } from "../LearnShell";
import { GeneratePanel } from "./GeneratePanel";
import { ImportPanel } from "./ImportPanel";
import { QuestionForm } from "./QuestionForm";
import { dueLabel, OptionKey, sourceLabel, typeLabel } from "./parts";

/** BankPanel 是题库上方打开着的那块面板。空串表示都收着。 */
export type BankPanel = "" | "new" | "import" | "generate";

type View = "all" | "mistakes" | "bookmarked" | "suspended";

const PAGE = 40;
const ALL_TAGS = "__all__";

/**
 * QuestionBank 画出题库。panel 是打开着的面板，onPanel 换掉它。sessionId 是练习的导师对话，出题用它的模型。
 * onPractice 用点名的题开一轮复习。
 */
export function QuestionBank(props: {
  panel: BankPanel;
  onPanel: (panel: BankPanel) => void;
  sessionId?: string;
  onPractice: (ids: string[]) => void;
}) {
  const L = useCopy().learn.drill;
  const [query, setQuery] = useState("");
  const [typed, setTyped] = useState("");
  const [view, setView] = useState<View>("all");
  const [tag, setTag] = useState(ALL_TAGS);
  const [sort, setSort] = useState<NonNullable<BankFilter["sort"]>>("recent");
  const [items, setItems] = useState<Question[]>([]);
  const [total, setTotal] = useState(0);
  const [tags, setTags] = useState<{ tag: string; count: number }[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [openId, setOpenId] = useState("");
  const [editing, setEditing] = useState<QuestionDetail | null>(null);
  const [note, setNote] = useState("");
  const [err, setErr] = useState("");
  const shown = useRef(PAGE);

  /** 按当前筛选读出题库。keep 为真时保持已经展开的条数，否则回到第一页。 */
  const load = useCallback(async (keep: boolean) => {
    const limit = keep ? Math.max(PAGE, shown.current) : PAGE;
    try {
      const [page, tagList] = await Promise.all([
        practice.questions({
          query, sort, limit, tag: tag === ALL_TAGS ? "" : tag,
          mistakesOnly: view === "mistakes",
          bookmarked: view === "bookmarked" ? true : undefined,
          suspended: view === "suspended" ? true : undefined,
        }),
        practice.tags(),
      ]);
      shown.current = limit;
      setItems(page.items);
      setTotal(page.total);
      setTags(tagList);
      setErr("");
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setLoaded(true);
    }
  }, [query, sort, tag, view]);

  useEffect(() => {
    void load(false);
  }, [load]);

  useEffect(() => subscribeLearn(/** 题库有变化就重读，保持已经展开的条数。掌握路径上答错的题也会进来。 */ (change) => {
    if (change.scope === "practice" || change.scope === "path") void load(true);
  }), [load]);

  useEffect(() => {
    /** 停笔四分之一秒后才搜。 */
    const timer = window.setTimeout(() => setQuery(typed.trim()), 250);
    return () => window.clearTimeout(timer);
  }, [typed]);

  /** 再读一页。 */
  const more = async () => {
    shown.current += PAGE;
    await load(true);
  };

  /** 面板办完事后收起，把结果说一句，并重读列表。 */
  const finished = (message: string) => {
    props.onPanel("");
    setEditing(null);
    setNote(message);
    void load(true);
  };

  return (
    <div className="mx-auto flex max-w-[720px] flex-col gap-4 px-8 py-6" data-testid="learn-bank">
      {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
      {note ? <p className="text-ui-footnote text-muted" aria-live="polite">{note}</p> : null}
      {props.panel === "new" || editing ? (
        <div className="border-b border-border/50 pb-5">
          <QuestionForm
            key={editing?.question.id || "new"}
            initial={editing || undefined}
            onSaved={() => finished("")}
            onCancel={() => { props.onPanel(""); setEditing(null); }}
          />
        </div>
      ) : null}
      {props.panel === "import" ? <div className="border-b border-border/50 pb-5"><ImportPanel onDone={finished} onCancel={() => props.onPanel("")} /></div> : null}
      {props.panel === "generate" ? <div className="border-b border-border/50 pb-5"><GeneratePanel sessionId={props.sessionId} onDone={finished} onCancel={() => props.onPanel("")} /></div> : null}

      <div className="flex flex-wrap items-center gap-2">
        <Input className="min-w-[140px] flex-1" value={typed} placeholder={L.searchPh} aria-label={L.searchPh} onChange={(e) => setTyped(e.target.value)} />
        {tags.length > 0 ? (
          <Select value={tag} onValueChange={setTag}>
            <SelectTrigger className="w-[128px]" aria-label={L.tagLabel}><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_TAGS}>{L.tagAll}</SelectItem>
              {tags.map((t) => <SelectItem key={t.tag} value={t.tag}>{t.tag} · {t.count}</SelectItem>)}
            </SelectContent>
          </Select>
        ) : null}
        <Select value={sort} onValueChange={(v) => setSort(v as NonNullable<BankFilter["sort"]>)}>
          <SelectTrigger className="w-[104px]" aria-label={L.sortLabel}><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="recent">{L.sortRecent}</SelectItem>
            <SelectItem value="oldest">{L.sortOldest}</SelectItem>
            <SelectItem value="due">{L.sortDue}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <LearnSegment
          label={L.filterLabel}
          value={view}
          onChange={(v) => setView(v as View)}
          options={[
            { id: "all", label: L.filterAll },
            { id: "mistakes", label: L.filterMistakes },
            { id: "bookmarked", label: L.filterBookmarked },
            { id: "suspended", label: L.filterSuspended },
          ]}
        />
        <div className="flex items-center gap-2">
          <span className="tabular-nums text-ui-footnote text-muted">{fill(L.total, { n: total })}</span>
          {items.length > 0 && view !== "suspended" ? (
            <Button size="sm" variant="outline" className="whitespace-nowrap" onClick={() => props.onPractice(items.filter((q) => !q.suspended).slice(0, 100).map((q) => q.id))}>{L.practiceThese}</Button>
          ) : null}
        </div>
      </div>

      {loaded && items.length === 0 ? (
        <EmptyState title={total === 0 && !query && view === "all" && tag === ALL_TAGS ? L.emptyBank : L.bankEmpty} body={total === 0 && !query && view === "all" && tag === ALL_TAGS ? L.emptyBankHint : L.bankEmptyHint} />
      ) : null}
      <ul className="flex flex-col">
        {items.map((q) => (
          <BankRow
            key={q.id}
            question={q}
            open={openId === q.id}
            onToggle={() => setOpenId((cur) => (cur === q.id ? "" : q.id))}
            onChanged={() => { void load(true); }}
            onEdit={(detail) => { props.onPanel(""); setEditing(detail); }}
            onPractice={() => props.onPractice([q.id])}
          />
        ))}
      </ul>
      {items.length < total ? (
        <div><Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => { void more(); }}>{L.loadMore}</Button></div>
      ) : null}
    </div>
  );
}

/**
 * BankRow 是题库里的一道题。点开后读出答案、解析和复习记录，并给出练、改、标记和删除几个动作。
 * question 是这道题。open 为真时展开。onToggle 展开或收起。onChanged 在标记或删除之后调用。
 * onEdit 带着详情打开表单。onPractice 单练这一道。
 */
function BankRow(props: {
  question: Question;
  open: boolean;
  onToggle: () => void;
  onChanged: () => void;
  onEdit: (detail: QuestionDetail) => void;
  onPractice: () => void;
}) {
  const L = useCopy().learn.drill;
  const { question: q, open } = props;
  const [detail, setDetail] = useState<QuestionDetail | null>(null);
  const [err, setErr] = useState("");
  const [confirm, setConfirm] = useState(false);

  const load = useCallback(async () => {
    try {
      setDetail(await practice.question(q.id));
      setErr("");
    } catch (e) {
      setErr(errMessage(e));
    }
  }, [q.id]);

  useEffect(() => {
    if (open) void load();
  }, [open, load, q.updated_at, q.reps, q.due_at]);

  /** 改这道题的标记，然后让列表重读。 */
  const flag = async (flags: { bookmarked?: boolean; suspended?: boolean; mistake?: boolean }) => {
    try {
      await practice.flagQuestions([q.id], flags);
      props.onChanged();
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  const meta = [typeLabel(L, q.qtype)];
  if (q.mistake) meta.push(L.mistake);
  meta.push(dueLabel(L, q, getLocale()));
  const pickable = q.options.length > 0 && (q.qtype === "choice" || q.qtype === "multi" || q.qtype === "true_false");

  return (
    <li className="border-b border-border/50" data-testid="learn-bank-row">
      <div className="flex items-start gap-2">
        <button type="button" className="flex min-w-0 flex-1 items-start gap-3 py-2.5 text-left" aria-expanded={open} onClick={props.onToggle}>
          <span className="mt-1 shrink-0 text-muted">{open ? <ChevronDown className="size-3.5" aria-hidden /> : <ChevronRight className="size-3.5" aria-hidden />}</span>
          <span className="min-w-0 flex-1">
            <span className={cn("block text-ui-body-l", open ? "whitespace-pre-wrap break-words" : "truncate", q.suspended && "text-muted")}>{q.stem}</span>
            <span className="mt-0.5 block truncate text-ui-footnote text-muted">
              <span className={q.mistake ? "text-warning" : ""}>{meta.join(" · ")}</span>
              {q.tags.length > 0 ? <span className="text-muted/70"> · {q.tags.map((t) => `#${t}`).join(" ")}</span> : null}
            </span>
          </span>
        </button>
        <Button
          size="icon"
          variant="ghost"
          className="mt-1 shrink-0"
          aria-label={q.bookmarked ? L.unbookmark : L.bookmark}
          aria-pressed={q.bookmarked}
          onClick={() => { void flag({ bookmarked: !q.bookmarked }); }}
        >
          <Star className={cn("size-3.5", q.bookmarked ? "fill-current text-foreground" : "text-muted/60")} aria-hidden />
        </Button>
      </div>
      {open ? (
        <div className="flex flex-col gap-3 pb-4 pl-[26px] text-ui-body" data-testid="learn-bank-detail">
          {err ? <p className="text-ui-footnote text-danger">{err}</p> : null}
          {detail ? (
            <>
              {pickable ? <OptionKey options={q.options} answer={detail.reveal.answer} dense /> : detail.reveal.answer ? (
                <p><span className="text-muted">{L.answer}</span><span className="text-muted/50"> · </span>{q.qtype === "fill" ? detail.reveal.answer.split("|").join(" / ") : detail.reveal.answer}</p>
              ) : null}
              {detail.reveal.explanation ? (
                <p className="whitespace-pre-wrap text-muted"><span className="text-ui-footnote text-muted/80">{L.explanation}</span><br />{detail.reveal.explanation}</p>
              ) : null}
              <p className="tabular-nums text-ui-footnote text-muted">
                {[
                  sourceLabel(L, q.source),
                  q.reps > 0 ? fill(L.reps, { n: q.reps }) : L.historyEmpty,
                  q.lapses > 0 ? fill(L.lapses, { n: q.lapses }) : "",
                  q.reps > 0 && q.retrievability > 0 ? fill(L.recall, { n: percent(q.retrievability) }) : "",
                ].filter(Boolean).join(" · ")}
              </p>
              {detail.events.length > 0 ? (
                <ul className="flex flex-col gap-1 text-ui-footnote">
                  {detail.events.slice(0, 6).map((ev, i) => (
                    <li key={`${ev.at}-${i}`} className="flex gap-3">
                      <span className="w-24 shrink-0 tabular-nums text-muted/80">{new Intl.DateTimeFormat(getLocale(), { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }).format(new Date(ev.at))}</span>
                      <span className={cn("w-12 shrink-0", ev.rating === "again" ? "text-danger" : "text-muted")}>{L[ev.rating] || ev.rating}</span>
                      <span className={cn("w-8 shrink-0", ev.correct === true ? "text-success" : ev.correct === false ? "text-danger" : "text-muted")}>{ev.correct === true ? L.evCorrect : ev.correct === false ? L.evWrong : ""}</span>
                      <span className="min-w-0 flex-1 truncate text-muted">{ev.answer}</span>
                    </li>
                  ))}
                </ul>
              ) : null}
              <div className="flex flex-wrap items-center gap-1.5">
                {!q.suspended ? <Button size="xs" variant="outline" onClick={props.onPractice}>{L.practiceOne}</Button> : null}
                <Button size="xs" variant="ghost" onClick={() => props.onEdit(detail)}>{L.edit}</Button>
                <Button size="xs" variant="ghost" onClick={() => { void flag({ suspended: !q.suspended }); }}>{q.suspended ? L.resume : L.suspend}</Button>
                <Button size="xs" variant="ghost" onClick={() => { void flag({ mistake: !q.mistake }); }}>{q.mistake ? L.unmarkMistake : L.markMistake}</Button>
                <Button size="xs" variant="ghost" onClick={() => setConfirm(true)}>{L.remove}</Button>
              </div>
            </>
          ) : null}
        </div>
      ) : null}
      <ConfirmDialog
        open={confirm}
        title={L.removeTitle}
        body={L.removeBody}
        confirmLabel={L.remove}
        danger
        onCancel={() => setConfirm(false)}
        onConfirm={async () => {
          setConfirm(false);
          try {
            await practice.deleteQuestions([q.id]);
            props.onChanged();
          } catch (e) {
            setErr(errMessage(e));
          }
        }}
      />
    </li>
  );
}
