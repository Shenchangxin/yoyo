/**
 * 拥有：学习计划这一页：现在轮到哪几件事、四个入口，以及使命和「记下的」。
 * 不拥有：轮到什么由引擎算。点开一件事经回调交给调用方，这里不切泳道。
 */

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { BookOpen, Library, ListChecks, PenLine } from "lucide-react";
import { useCopy } from "../../../lib/i18n";
import * as api from "../../../lib/client";
import type { PathSummary } from "../../../lib/learn-api";
import type { BookSummary } from "../../../lib/learn-books";
import type { Material } from "../../../lib/learn-library";
import type { PracticeSummary } from "../../../lib/learn-practice";
import { subscribeLearn } from "../../../lib/learn-events";
import { fill, percent } from "../../../lib/learn-format";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { EmptyState } from "../../../components/ui/empty-state";
import { LearnShell } from "../LearnShell";

type Journal = { topic?: string; why?: string; level?: string; summary?: string; next_focus?: string; records?: { id: string; insight: string }[] };
type Today = { journal: Journal; practice?: PracticeSummary; paths: PathSummary[]; books: BookSummary[]; materials: Material[] };

/** Row 是「现在做这个」里的一行：左边是哪一类，中间是名字，右边是一句状态。 */
function Row(props: { kind: string; title: string; meta: string; warn?: boolean; testId?: string; onClick: () => void }) {
  return (
    <li>
      <button type="button" data-testid={props.testId} className="flex w-full items-center gap-3 border-b border-border/50 py-3 text-left hover:bg-lift/40" onClick={props.onClick}>
        <span className="w-12 shrink-0 whitespace-nowrap text-ui-footnote text-muted">{props.kind}</span>
        <span className="min-w-0 flex-1 truncate text-ui-body-l">{props.title}</span>
        <span className={props.warn ? "shrink-0 whitespace-nowrap text-ui-footnote tabular-nums text-warning" : "shrink-0 whitespace-nowrap text-ui-footnote tabular-nums text-muted"}>{props.meta}</span>
      </button>
    </li>
  );
}

/** Labeled 是表单里的一行：上面是标签，下面是控件。 */
function Labeled(props: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-ui-footnote text-muted">{props.label}</span>
      {props.children}
    </label>
  );
}

/**
 * HomePane 画出学习计划。各个回调把学习者带到对应的地方：onPractice 开始练习，onPath 打开一条路径，
 * onMaterial 接着读一份材料，onBook 打开一本课本；onLane 走四个入口之一。
 */
export function HomePane(props: {
  onPractice: () => void;
  onPath: (id: string) => void;
  onMaterial: (libraryId: string, documentId: string) => void;
  onBook: (id: string) => void;
  onLane: (target: "study" | "read" | "book" | "practice") => void;
}) {
  const copy = useCopy().learn;
  const L = copy.home;
  const [data, setData] = useState<Today | null>(null);
  const [editing, setEditing] = useState(false);
  const [topic, setTopic] = useState("");
  const [why, setWhy] = useState("");
  const [next, setNext] = useState("");
  const [insight, setInsight] = useState("");
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api.learnCall("learn.today");
      setData({
        journal: res?.journal || {},
        practice: res?.practice,
        paths: Array.isArray(res?.paths) ? res.paths : [],
        books: Array.isArray(res?.books) ? res.books : [],
        materials: Array.isArray(res?.materials) ? res.materials : [],
      });
      setErr("");
    } catch (e) {
      setErr(errMessage(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => subscribeLearn(/** 学习数据有变化就重读今天的安排。阅读位置的变化太密，不跟。 */ (change) => {
    if (change.scope !== "material") void load();
  }), [load]);

  const j = data?.journal || {};
  useEffect(() => {
    setTopic(j.topic || "");
    setWhy(j.why || "");
    setNext(j.next_focus || "");
  }, [j.topic, j.why, j.next_focus]);

  /** 保存使命。上一课的小结和水平不在这里改，原样带回去。 */
  const saveMission = async () => {
    try {
      await api.learnCall("learn.journal.set", { topic, why, next_focus: next, summary: j.summary || "", level: j.level || "" });
      setEditing(false);
      await load();
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  /** 记下一条今天弄懂的事。 */
  const keep = async () => {
    const text = insight.trim();
    if (!text) return;
    try {
      await api.learnCall("learn.journal.add", { insight: text });
      setInsight("");
      await load();
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  const practice = data?.practice;
  const ready = practice ? practice.due + practice.new : 0;
  const paths = (data?.paths || []).slice(0, 4);
  const materials = (data?.materials || []).filter((m) => m.percent < 1).slice(0, 2);
  const books = (data?.books || []).filter((b) => b.status !== "ready" || b.percent < 1).slice(0, 2);
  const rows = (ready > 0 ? 1 : 0) + paths.length + materials.length + books.length;
  const records = Array.isArray(j.records) ? j.records.slice(0, 3) : [];
  const ways: { id: "study" | "read" | "book" | "practice"; title: string; hint: string; icon: typeof ListChecks }[] = [
    { id: "study", title: copy.mastery.title, hint: copy.mastery.subtitle, icon: ListChecks },
    { id: "read", title: copy.reader, hint: copy.wayReadHint, icon: Library },
    { id: "book", title: copy.books, hint: copy.tome.subtitle, icon: BookOpen },
    { id: "practice", title: copy.drill.title, hint: copy.drill.subtitle, icon: PenLine },
  ];

  /** bookMeta 说出一本课本现在等着什么。 */
  const bookMeta = (b: BookSummary): string => {
    if (b.status === "draft") return b.pause_reason === "building_spine" ? copy.tome.statusBuilding : copy.tome.statusDraft;
    if (b.status === "spine_ready") return copy.tome.statusSpine;
    if (b.status === "paused") return copy.tome.statusPaused;
    if (b.status === "error") return copy.tome.statusError;
    return fill(copy.tome.chaptersDone, { done: b.ready_chapters, total: b.chapters });
  };

  return (
    <LearnShell title={L.title} subtitle={L.subtitle}>
      <div className="mx-auto flex max-w-[720px] flex-col gap-8 px-8 py-6" data-testid="learn-home">
        {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}

        <section className="flex flex-col gap-1">
          <div className="flex items-baseline justify-between gap-3">
            <h2 className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.now}</h2>
            {practice && practice.reviewed_today > 0 ? (
              <span className="text-ui-footnote tabular-nums text-muted">
                {fill(L.reviewedToday, { n: practice.reviewed_today })}{practice.streak_days > 1 ? ` · ${fill(L.streak, { n: practice.streak_days })}` : ""}
              </span>
            ) : null}
          </div>
          {data && rows === 0 ? <EmptyState title={L.nothing} body={L.nothingHint} /> : null}
          <ul className="flex flex-col">
            {practice && ready > 0 ? (
              <Row kind={L.practice} title={fill(copy.drill.dueSplit, { due: practice.due, fresh: practice.new })} meta={L.begin} warn={practice.due > 0} testId="learn-home-practice" onClick={props.onPractice} />
            ) : null}
            {paths.map((p) => (
              <Row
                key={p.id}
                kind={L.mastery}
                title={p.name}
                meta={p.pending_question ? copy.mastery.pendingBadge : p.due_reviews > 0 ? fill(copy.mastery.dueBadge, { n: p.due_reviews }) : fill(L.mastered, { done: p.counts.mastered, total: p.counts.total })}
                warn={p.pending_question || p.due_reviews > 0}
                onClick={() => props.onPath(p.id)}
              />
            ))}
            {materials.map((m) => (
              <Row
                key={m.id}
                kind={L.reading}
                title={m.title}
                meta={`${fill(m.unit === "page" ? copy.shelf.wherePage : copy.shelf.whereSection, { n: m.locator })} · ${fill(copy.shelf.progress, { n: percent(m.percent) })}`}
                onClick={() => props.onMaterial(m.library_id, m.document_id)}
              />
            ))}
            {books.map((b) => <Row key={b.id} kind={L.book} title={b.title} meta={bookMeta(b)} warn={b.status === "draft" || b.status === "spine_ready"} onClick={() => props.onBook(b.id)} />)}
          </ul>
        </section>

        <section className="flex flex-col gap-2">
          <h2 className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.startNew}</h2>
          <div className="grid grid-cols-2 gap-2">
            {ways.map((way) => {
              const Icon = way.icon;
              return (
                <button key={way.id} type="button" className="flex items-start gap-3 rounded-xl border border-border/60 p-3 text-left hover:bg-lift/50" onClick={() => props.onLane(way.id)}>
                  <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-lift text-muted"><Icon className="size-4" aria-hidden /></span>
                  <span className="min-w-0">
                    <span className="block truncate text-ui-body-l font-medium">{way.title}</span>
                    <span className="mt-0.5 line-clamp-2 block text-ui-footnote leading-relaxed text-muted">{way.hint}</span>
                  </span>
                </button>
              );
            })}
          </div>
        </section>

        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.mission}</h2>
            <button type="button" className="shrink-0 whitespace-nowrap text-ui-footnote text-muted hover:text-foreground" aria-expanded={editing} onClick={() => setEditing((v) => !v)}>{editing ? L.cancel : L.edit}</button>
          </div>
          {editing ? (
            <div className="flex flex-col gap-3">
              <Labeled label={L.mission}><Input value={topic} placeholder={L.missionPh} onChange={(e) => setTopic(e.target.value)} /></Labeled>
              <Labeled label={L.why}><Input value={why} placeholder={L.whyPh} onChange={(e) => setWhy(e.target.value)} /></Labeled>
              <Labeled label={L.next}><Input value={next} placeholder={L.nextPh} onChange={(e) => setNext(e.target.value)} /></Labeled>
              <div><Button size="sm" className="whitespace-nowrap" onClick={() => { void saveMission(); }}>{L.save}</Button></div>
            </div>
          ) : j.topic ? (
            <div>
              <p className="text-ui-title-s leading-snug">{j.topic}</p>
              {j.why ? <p className="mt-1 text-ui-body text-muted">{j.why}</p> : null}
              {j.next_focus ? <p className="mt-1 text-ui-body text-muted"><span className="text-muted/70">{L.next}</span><span className="text-muted/50"> · </span>{j.next_focus}</p> : null}
            </div>
          ) : (
            <p className="text-ui-body text-muted">{L.missionEmpty}</p>
          )}
        </section>

        <section className="flex flex-col gap-2">
          <h2 className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.kept}</h2>
          {records.map((row) => <p key={row.id} className="text-ui-body leading-relaxed text-muted">{row.insight}</p>)}
          <form
            className="flex items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              void keep();
            }}
          >
            <Input className="min-w-0 flex-1" value={insight} placeholder={L.keptPh} aria-label={L.keptPh} onChange={(e) => setInsight(e.target.value)} />
            <Button size="sm" type="submit" variant="ghost" className="shrink-0 whitespace-nowrap" disabled={!insight.trim()}>{L.keep}</Button>
          </form>
        </section>
      </div>
    </LearnShell>
  );
}
