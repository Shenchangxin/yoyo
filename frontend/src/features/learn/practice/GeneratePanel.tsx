/**
 * 拥有：让模型出一组题的面板：写主题、挑题型和数量，看草稿，勾出要留下的再存进题库。
 * 不拥有：出题和校验，那在引擎里。草稿不落库，留下的才写。
 */

import { useEffect, useState } from "react";
import { useCopy } from "../../../lib/i18n";
import * as api from "../../../lib/client";
import * as practice from "../../../lib/learn-practice";
import type { DraftQuestion, QuestionType } from "../../../lib/learn-practice";
import { fill } from "../../../lib/learn-format";
import { splitTags } from "../../../lib/learn-practice-util";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Checkbox } from "../../../components/ui/checkbox";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../components/ui/select";
import { cn } from "../../../lib/utils";
import { LearnSegment } from "../LearnShell";
import { Field, OptionKey, textareaClass, typeLabel } from "./parts";

const TYPES: QuestionType[] = ["choice", "multi", "true_false", "fill", "short"];
const NO_LIBRARY = "none";

/**
 * GeneratePanel 画出出题的面板。sessionId 是练习的导师对话，出题用它的模型。
 * onDone 在题目写入之后调用，带上一句结果。onCancel 收起面板。
 */
export function GeneratePanel(props: { sessionId?: string; onDone: (message: string) => void; onCancel: () => void }) {
  const L = useCopy().learn.drill;
  const [topic, setTopic] = useState("");
  const [libraryId, setLibraryId] = useState(NO_LIBRARY);
  const [libs, setLibs] = useState<{ id: string; name: string }[]>([]);
  const [count, setCount] = useState("5");
  const [difficulty, setDifficulty] = useState("mixed");
  const [types, setTypes] = useState<QuestionType[]>(["choice", "true_false", "fill", "short"]);
  const [drafts, setDrafts] = useState<DraftQuestion[] | null>(null);
  const [kept, setKept] = useState<boolean[]>([]);
  const [tags, setTags] = useState("");
  const [busy, setBusy] = useState<"" | "write" | "keep">("");
  const [err, setErr] = useState("");

  useEffect(() => {
    void api.learnCall("learn.libraries").then(/** 笔记库不当出题的依据。 */ (rows) => {
      const all = Array.isArray(rows) ? rows : [];
      setLibs(all.filter((lib) => lib?.id && lib.id !== "notes").map((lib) => ({ id: String(lib.id), name: String(lib.name || lib.id) })));
    }).catch(() => setLibs([]));
  }, []);

  /** 请模型出一组草稿。 */
  const write = async () => {
    if (!topic.trim() || busy) return;
    setBusy("write");
    setErr("");
    try {
      const items = await practice.generate({
        topic: topic.trim(), libraryId: libraryId === NO_LIBRARY ? "" : libraryId, count: Number(count),
        qtypes: types, difficulty: difficulty === "mixed" ? "" : difficulty, session: props.sessionId,
      });
      setDrafts(items);
      setKept(items.map(() => true));
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy("");
    }
  };

  /** 把勾中的草稿存进题库。 */
  const keep = async () => {
    if (!drafts || busy) return;
    const chosen = drafts.filter((_, i) => kept[i]);
    if (chosen.length === 0) return;
    setBusy("keep");
    setErr("");
    try {
      const res = await practice.addQuestions(chosen, splitTags(tags), props.sessionId || "");
      props.onDone(res.skipped > 0 ? fill(L.keptSkipped, { n: res.added, skipped: res.skipped }) : fill(L.kept, { n: res.added }));
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy("");
    }
  };

  const chosen = kept.filter(Boolean).length;

  return (
    <section className="flex flex-col gap-4" data-testid="learn-generate">
      <div>
        <p className="text-ui-body font-medium">{L.genTitle}</p>
        <p className="mt-1 text-ui-footnote leading-relaxed text-muted">{L.genHint}</p>
      </div>
      {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
      {!drafts ? (
        <>
          <Field label={L.topicLabel}>
            <textarea className={cn(textareaClass, "min-h-16")} value={topic} placeholder={L.topicPh} autoFocus onChange={(e) => setTopic(e.target.value)} />
          </Field>
          {libs.length > 0 ? (
            <div className="flex flex-col gap-1">
              <span className="text-ui-footnote text-muted">{L.libraryLabel}</span>
              <Select value={libraryId} onValueChange={setLibraryId}>
                <SelectTrigger className="w-full max-w-xs" aria-label={L.libraryLabel}><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value={NO_LIBRARY}>{L.libraryNone}</SelectItem>
                  {libs.map((lib) => <SelectItem key={lib.id} value={lib.id}>{lib.name}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          ) : null}
          <div className="flex flex-wrap items-end gap-4">
            <div className="flex flex-col gap-1">
              <span className="text-ui-footnote text-muted">{L.countLabel}</span>
              <LearnSegment label={L.countLabel} value={count} onChange={setCount} options={["5", "10", "20"].map((n) => ({ id: n, label: n }))} />
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-ui-footnote text-muted">{L.difficultyLabel}</span>
              <LearnSegment
                label={L.difficultyLabel}
                value={difficulty}
                onChange={setDifficulty}
                options={[{ id: "mixed", label: L.diffMixed }, { id: "easy", label: L.diffEasy }, { id: "medium", label: L.diffMedium }, { id: "hard", label: L.diffHard }]}
              />
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <span className="text-ui-footnote text-muted">{L.typesLabel}</span>
            <div className="flex flex-wrap gap-1.5" role="group" aria-label={L.typesLabel}>
              {TYPES.map((t) => {
                const on = types.includes(t);
                return (
                  <button
                    key={t}
                    type="button"
                    aria-pressed={on}
                    className={cn("h-7 whitespace-nowrap rounded-md border px-2.5 text-ui-footnote", on ? "border-foreground/40 bg-lift text-foreground" : "border-border/60 text-muted hover:text-foreground")}
                    onClick={() => setTypes((cur) => (on ? (cur.length > 1 ? cur.filter((x) => x !== t) : cur) : [...cur, t]))}
                  >
                    {typeLabel(L, t)}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" className="whitespace-nowrap" disabled={!topic.trim() || !!busy} onClick={() => { void write(); }}>{busy === "write" ? L.writing : L.write}</Button>
            <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={props.onCancel}>{L.cancel}</Button>
          </div>
        </>
      ) : (
        <>
          <ul className="flex flex-col">
            {drafts.map((d, i) => (
              <li key={i} className="flex gap-3 border-b border-border/50 py-3">
                <Checkbox className="mt-1" checked={!!kept[i]} aria-label={L.pickLabel} onChange={(e) => setKept((cur) => cur.map((v, at) => (at === i ? e.target.checked : v)))} />
                <div className={cn("flex min-w-0 flex-1 flex-col gap-2", !kept[i] && "opacity-50")}>
                  <p className="text-ui-footnote text-muted">{typeLabel(L, d.qtype)}{d.difficulty ? ` · ${d.difficulty === "easy" ? L.diffEasy : d.difficulty === "hard" ? L.diffHard : L.diffMedium}` : ""}</p>
                  <p className="text-ui-body-l leading-relaxed">{d.stem}</p>
                  {d.options.length > 0 ? <OptionKey options={d.options} answer={d.answer} dense /> : (
                    <p className="text-ui-body"><span className="text-muted">{L.answer}</span><span className="text-muted/50"> · </span>{d.qtype === "fill" ? d.answer.split("|").join(" / ") : d.answer}</p>
                  )}
                  {d.explanation ? <p className="text-ui-footnote leading-relaxed text-muted">{d.explanation}</p> : null}
                </div>
              </li>
            ))}
          </ul>
          <Field label={L.tagsLabel}><Input value={tags} placeholder={L.tagsPh} onChange={(e) => setTags(e.target.value)} /></Field>
          <div className="flex items-center gap-2">
            <Button size="sm" className="whitespace-nowrap" disabled={chosen === 0 || !!busy} onClick={() => { void keep(); }}>{fill(L.keepN, { n: chosen })}</Button>
            <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={!!busy} onClick={() => setDrafts(null)}>{L.rewrite}</Button>
            <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={props.onCancel}>{L.cancel}</Button>
          </div>
        </>
      )}
    </section>
  );
}
