/**
 * 拥有：写一道题或改一道题的表单：题型、题干、选项和正确项、答案、解析、难度和标签。
 * 不拥有：校验。题型和答案对不对得上由引擎说了算，这里只把引擎的话显示出来。
 */

import { useState } from "react";
import { Plus, X } from "lucide-react";
import { useCopy } from "../../../lib/i18n";
import * as practice from "../../../lib/learn-practice";
import type { DraftQuestion, Question, QuestionDetail, QuestionType } from "../../../lib/learn-practice";
import { fill } from "../../../lib/learn-format";
import { answerLabels, splitTags } from "../../../lib/learn-practice-util";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Checkbox } from "../../../components/ui/checkbox";
import { LearnSegment } from "../LearnShell";
import { Field, textareaClass, typeLabel } from "./parts";

const TYPES: QuestionType[] = ["choice", "multi", "true_false", "fill", "short", "open"];

type Row = { body: string; correct: boolean };

/** blankRows 是新选择题起手的四个空选项。 */
function blankRows(): Row[] {
  return [0, 1, 2, 3].map(() => ({ body: "", correct: false }));
}

/** rowsOf 把一道已有的题的选项和答案变成表单里的行。不是选择题时给四个空行。 */
function rowsOf(detail?: QuestionDetail): Row[] {
  const q = detail?.question;
  if (!q || (q.qtype !== "choice" && q.qtype !== "multi") || q.options.length === 0) return blankRows();
  const right = answerLabels(detail.reveal.answer);
  return q.options.map((o) => ({ body: o.body, correct: right.has(o.label) }));
}

/**
 * QuestionForm 画出写题的表单。initial 有值时是修改那道题，否则是新建。
 * onSaved 在引擎收下之后调用，带上保存后的题。onCancel 收起表单。
 */
export function QuestionForm(props: {
  initial?: QuestionDetail;
  onSaved: (q: Question) => void;
  onCancel: () => void;
}) {
  const L = useCopy().learn.drill;
  const base = props.initial?.question;
  const [qtype, setQtype] = useState<QuestionType>(base?.qtype || "choice");
  const [stem, setStem] = useState(base?.stem || "");
  const [rows, setRows] = useState<Row[]>(() => rowsOf(props.initial));
  const [truth, setTruth] = useState(base?.qtype === "true_false" && props.initial?.reveal.answer === "F" ? "F" : "T");
  const [text, setText] = useState(base && base.qtype !== "choice" && base.qtype !== "multi" && base.qtype !== "true_false" ? props.initial?.reveal.answer || "" : "");
  const [explanation, setExplanation] = useState(props.initial?.reveal.explanation || "");
  const [difficulty, setDifficulty] = useState(base?.difficulty || "");
  const [tags, setTags] = useState((base?.tags || []).join(", "));
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const picking = qtype === "choice" || qtype === "multi";

  /** 改一行选项。单选时标了一个正确项，别的自动取消。 */
  const setRow = (at: number, patch: Partial<Row>) => {
    setRows((list) => list.map((row, i) => {
      if (i === at) return { ...row, ...patch };
      if (patch.correct && qtype === "choice") return { ...row, correct: false };
      return row;
    }));
  };

  /** 换题型。从多选换到单选时只留第一个正确项。 */
  const changeType = (next: string) => {
    setQtype(next as QuestionType);
    if (next === "choice") {
      let seen = false;
      setRows((list) => list.map((row) => {
        if (!row.correct) return row;
        if (seen) return { ...row, correct: false };
        seen = true;
        return row;
      }));
    }
  };

  /** 把表单收成一道题交给引擎。选择题的答案写成正确项的字母。 */
  const save = async () => {
    if (busy) return;
    const draft: DraftQuestion = { stem: stem.trim(), qtype, options: [], answer: "", explanation: explanation.trim(), difficulty, tags: splitTags(tags) };
    if (picking) {
      const kept = rows.filter((row) => row.body.trim());
      draft.options = kept.map((row, i) => ({ label: String.fromCharCode(65 + i), body: row.body.trim() }));
      draft.answer = kept.map((row, i) => (row.correct ? String.fromCharCode(65 + i) : "")).filter(Boolean).join(",");
      if (!draft.answer) {
        setErr(L.needCorrect);
        return;
      }
    } else if (qtype === "true_false") {
      draft.answer = truth;
    } else {
      draft.answer = text.trim();
    }
    setBusy(true);
    setErr("");
    try {
      props.onSaved(await practice.saveQuestion(draft, base?.id || ""));
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form
      className="flex flex-col gap-4"
      data-testid="learn-question-form"
      onSubmit={(e) => {
        e.preventDefault();
        void save();
      }}
    >
      <p className="text-ui-body font-medium">{base ? L.formEdit : L.formNew}</p>
      <div className="flex flex-col gap-1">
        <span className="text-ui-footnote text-muted">{L.typeLabel}</span>
        <div><LearnSegment label={L.typeLabel} value={qtype} onChange={changeType} options={TYPES.map((t) => ({ id: t, label: typeLabel(L, t) }))} /></div>
      </div>
      <Field label={L.stemLabel}>
        <textarea className={`${textareaClass} min-h-20`} value={stem} autoFocus={!base} onChange={(e) => setStem(e.target.value)} />
      </Field>
      {picking ? (
        <div className="flex flex-col gap-1.5">
          <span className="text-ui-footnote text-muted">{L.optionsLabel}<span className="text-muted/70"> · {L.optionsHint}</span></span>
          {rows.map((row, i) => {
            const label = String.fromCharCode(65 + i);
            return (
              <div key={i} className="flex items-center gap-2">
                <Checkbox checked={row.correct} aria-label={`${L.correctMark} ${label}`} onChange={(e) => setRow(i, { correct: e.target.checked })} />
                <span className="w-4 shrink-0 text-ui-footnote font-medium tabular-nums text-muted">{label}</span>
                <Input className="min-w-0 flex-1" value={row.body} placeholder={fill(L.optionPh, { label })} aria-label={fill(L.optionPh, { label })} onChange={(e) => setRow(i, { body: e.target.value })} />
                <Button type="button" size="icon" variant="ghost" aria-label={L.removeOption} disabled={rows.length <= 2} onClick={() => setRows((list) => list.filter((_, at) => at !== i))}>
                  <X className="size-3.5" aria-hidden />
                </Button>
              </div>
            );
          })}
          {rows.length < 8 ? (
            <div>
              <Button type="button" size="xs" variant="ghost" onClick={() => setRows((list) => [...list, { body: "", correct: false }])}>
                <Plus aria-hidden />{L.addOption}
              </Button>
            </div>
          ) : null}
        </div>
      ) : qtype === "true_false" ? (
        <div className="flex flex-col gap-1">
          <span className="text-ui-footnote text-muted">{L.answerLabel}</span>
          <div><LearnSegment label={L.answerLabel} value={truth} onChange={setTruth} options={[{ id: "T", label: L.optTrue }, { id: "F", label: L.optFalse }]} /></div>
        </div>
      ) : (
        <Field label={L.answerLabel} hint={qtype === "fill" ? L.fillHint : undefined}>
          {qtype === "fill" ? (
            <Input value={text} onChange={(e) => setText(e.target.value)} />
          ) : (
            <textarea className={`${textareaClass} min-h-16`} value={text} placeholder={L.referencePh} onChange={(e) => setText(e.target.value)} />
          )}
        </Field>
      )}
      <Field label={L.explanationLabel}>
        <textarea className={`${textareaClass} min-h-16`} value={explanation} onChange={(e) => setExplanation(e.target.value)} />
      </Field>
      <div className="flex flex-wrap items-end gap-4">
        <div className="flex flex-col gap-1">
          <span className="text-ui-footnote text-muted">{L.difficultyLabel}</span>
          <LearnSegment
            label={L.difficultyLabel}
            value={difficulty || "none"}
            onChange={(v) => setDifficulty(v === "none" ? "" : v)}
            options={[{ id: "none", label: L.diffNone }, { id: "easy", label: L.diffEasy }, { id: "medium", label: L.diffMedium }, { id: "hard", label: L.diffHard }]}
          />
        </div>
        <div className="min-w-[200px] flex-1">
          <Field label={L.tagsLabel}><Input value={tags} placeholder={L.tagsPh} onChange={(e) => setTags(e.target.value)} /></Field>
        </div>
      </div>
      {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
      <div className="flex items-center gap-2">
        <Button size="sm" type="submit" className="whitespace-nowrap" disabled={!stem.trim() || busy}>{L.save}</Button>
        <Button size="sm" type="button" variant="ghost" className="whitespace-nowrap" onClick={props.onCancel}>{L.cancel}</Button>
      </div>
    </form>
  );
}
