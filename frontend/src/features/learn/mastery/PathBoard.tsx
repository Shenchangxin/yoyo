/**
 * 拥有：一条掌握路径的看板：下一步、大纲和每个知识点的状态、复习计划，以及改名、重置和删除。
 * 不拥有：分数、日程和下一步怎么算，那些由引擎给出。和导师说话经 onSay 交给调用方。
 */

import { useCallback, useEffect, useState } from "react";
import { Check, ChevronDown, ChevronRight } from "lucide-react";
import { getLocale, useCopy } from "../../../lib/i18n";
import * as learn from "../../../lib/learn-api";
import type { NextStep, OutlineModule, PathMode, PathView, PointDetail, PointView, PublicQuestion } from "../../../lib/learn-api";
import { fill, percent, whenLabel } from "../../../lib/learn-format";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { EmptyState } from "../../../components/ui/empty-state";
import { ProgressHairline } from "../../../components/ui/progress-hairline";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../../components/ui/dropdown-menu";
import { cn } from "../../../lib/utils";
import { ConfirmDialog } from "../../ConfirmDialog";
import { LearnSegment, LearnShell } from "../LearnShell";
import { LearnQuestionCard } from "../cards/LearnQuestionCard";
import { kindLabel, OutlineEditor } from "./OutlineEditor";

type Copy = ReturnType<typeof useCopy>["learn"]["mastery"];

const RETENTIONS = [0.8, 0.85, 0.9, 0.95];

/** stepLine 是下一步那一行的说明。 */
function stepLine(L: Copy, next: NextStep | undefined): string {
  switch (next?.action) {
    case "outline": return L.stepOutline;
    case "answer_pending": return L.stepAnswer;
    case "review": return L.stepReview;
    case "teach": return L.stepTeach;
    case "practice": return L.stepPractice;
    case "assess": return L.stepAssess;
    case "complete": return L.stepComplete;
    default: return L.stepProbe;
  }
}

/** stepSay 是点「开始」时替学习者对导师说的话。 */
function stepSay(L: Copy, next: NextStep | undefined): string {
  const name = next?.name || "";
  if (!name) return L.sayNext;
  switch (next?.action) {
    case "review": return fill(L.sayReview, { name });
    case "practice": return fill(L.sayPractice, { name });
    case "assess": return fill(L.sayAssess, { name });
    default: return fill(L.sayStart, { name });
  }
}

/** statusLabel 是知识点状态的名字。 */
function statusLabel(L: Copy, status: string): string {
  if (status === "mastered") return L.statusMastered;
  if (status === "learning") return L.statusLearning;
  return L.statusNew;
}

/** PointMark 用一个圆点表示知识点状态：实心带勾是已掌握，空心粗边是学习中，细边是未开始。 */
function PointMark(props: { status: string }) {
  const mastered = props.status === "mastered";
  return (
    <span
      className={cn(
        "grid size-5 shrink-0 place-items-center rounded-full border",
        mastered ? "border-foreground bg-foreground text-background" : props.status === "learning" ? "border-foreground" : "border-border",
      )}
      aria-hidden
    >
      {mastered ? <Check className="size-3" /> : null}
    </span>
  );
}

/**
 * PathBoard 画出一条路径。path 是引擎给的地图。sessionId 是这条路径的导师对话，判分和起草用它的模型。
 * onReload 在界面改了路径之后让调用方重读。onSay 把一句话发给导师。onClose 回到列表。onDeleted 在删除之后调用。
 */
export function PathBoard(props: {
  path: PathView;
  sessionId?: string;
  onReload: () => Promise<void> | void;
  onSay: (text: string) => void;
  onClose: () => void;
  onDeleted: () => void;
}) {
  const L = useCopy().learn.mastery;
  const { path } = props;
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<{ modules: OutlineModule[]; hint: string; description: string } | null>(null);
  const [drafting, setDrafting] = useState(false);
  const [openPoint, setOpenPoint] = useState("");
  const [confirm, setConfirm] = useState<"" | "reset" | "delete">("");
  const [renaming, setRenaming] = useState(false);
  const [name, setName] = useState(path.name);
  const [shown, setShown] = useState<PublicQuestion | null>(null);

  useEffect(() => {
    setEditing(null);
    setShown(null);
    setOpenPoint("");
    setRenaming(false);
    setName(path.name);
    setErr("");
  }, [path.id]);

  /** 跑一次改动，出错时把原因留在页面上，成功后重读路径。 */
  const run = async (fn: () => Promise<unknown>) => {
    setErr("");
    setBusy(true);
    try {
      await fn();
      await props.onReload();
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  /** 请模型起草大纲，放进编辑器，不写入。 */
  const draft = async () => {
    setErr("");
    setDrafting(true);
    try {
      const d = await learn.draftOutline(path.id, props.sessionId);
      setEditing({ modules: d.modules, hint: L.draftHint, description: d.description });
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setDrafting(false);
    }
  };

  /** 打开编辑器，内容是当前大纲。 */
  const edit = async () => {
    setErr("");
    try {
      setEditing({ modules: await learn.outline(path.id), hint: "", description: "" });
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  /** 大纲定下来之前导师不能出题，所以开始学之前先把模式切到学习。往下走时收起刚判完的那道题。 */
  const start = async () => {
    const say = stepSay(L, path.next);
    setShown(null);
    if (path.mode === "outline" && path.counts.total > 0) {
      await run(() => learn.setMode(path.id, "study"));
    }
    props.onSay(say);
  };

  const next = path.next;
  const pending = next?.action === "answer_pending" ? next.question : undefined;
  const pendingId = pending?.id || "";
  // 判完之后题目不再是「待答」，但学习者还要看结论和解析，所以卡片留在原处，直到往下走或来了新题。
  useEffect(() => {
    if (pending) setShown(pending);
  }, [pendingId]);
  const card = pending || shown;
  const total = path.counts.total;
  const subtitle = path.goal && path.goal !== path.name ? path.goal : path.description || "";

  return (
    <LearnShell
      title={path.name}
      subtitle={subtitle}
      back={{ label: L.allPaths, onClick: props.onClose }}
      action={(
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" variant="ghost" className="whitespace-nowrap">{L.more}</Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setRenaming(true)}>{L.rename}</DropdownMenuItem>
            {total > 0 ? <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void edit(); }}>{L.editOutline}</DropdownMenuItem> : null}
            <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setConfirm("reset")}>{L.reset}</DropdownMenuItem>
            <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setConfirm("delete")}>{L.remove}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    >
      <div className="mx-auto flex max-w-[680px] flex-col gap-6 px-8 py-6" data-testid="learn-path-board">
        {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
        {renaming ? (
          <form
            className="flex items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              if (!name.trim()) return;
              setRenaming(false);
              void run(() => learn.renamePath(path.id, name.trim()));
            }}
          >
            <Input className="min-w-0 flex-1" value={name} placeholder={L.renamePh} aria-label={L.renamePh} autoFocus onChange={(e) => setName(e.target.value)} />
            <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap">{L.rename}</Button>
            <Button size="sm" variant="ghost" className="shrink-0 whitespace-nowrap" onClick={() => { setRenaming(false); setName(path.name); }}>{L.cancel}</Button>
          </form>
        ) : null}
        {total > 0 ? (
          <div className="flex flex-col gap-2">
            <div className="flex items-baseline justify-between gap-3 text-ui-footnote text-muted">
              <span className="tabular-nums">{path.counts.mastered}/{total}</span>
              {path.due_reviews > 0 ? <span className="tabular-nums text-warning">{fill(L.dueBadge, { n: path.due_reviews })}</span> : null}
            </div>
            <ProgressHairline value={(path.counts.mastered / total) * 100} />
            <div className="mt-1 flex flex-wrap items-center gap-2">
              <LearnSegment
                label={L.modeLabel}
                value={path.mode}
                onChange={(mode) => { void run(() => learn.setMode(path.id, mode as PathMode)); }}
                options={[
                  { id: "outline", label: L.modeOutline },
                  { id: "study", label: L.modeStudy },
                  { id: "review", label: L.modeReview },
                ]}
              />
              <LearnSegment
                label={L.teachLabel}
                value={path.profile.teaching}
                onChange={(teaching) => { void run(() => learn.setProfile(path.id, { teaching: teaching as PathView["profile"]["teaching"] })); }}
                options={[
                  { id: "probe_first", label: L.probeFirst },
                  { id: "teach_first", label: L.teachFirst },
                ]}
              />
            </div>
          </div>
        ) : null}

        {editing ? (
          <OutlineEditor
            initial={editing.modules}
            hint={editing.hint}
            busy={busy}
            onCancel={() => setEditing(null)}
            onSave={(modules) => {
              const description = editing.description;
              void run(async () => {
                await learn.saveOutline(path.id, modules);
                if (description && !path.description) await learn.describePath(path.id, description);
                setEditing(null);
              });
            }}
          />
        ) : (
          <>
            <section data-testid="learn-next" className="flex flex-col gap-3">
              {card ? (
                <LearnQuestionCard
                  key={card.id}
                  pathId={path.id}
                  questionId={card.id}
                  initial={card}
                  sessionId={props.sessionId}
                  onGraded={(said) => { props.onSay(said); void props.onReload(); }}
                  onSkipped={() => { setShown(null); void props.onReload(); }}
                />
              ) : null}
              {pending ? null : (
                <div>
                  <p className="mb-3 text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.nextLabel}</p>
                  {next?.name ? <p className="text-ui-title font-semibold leading-snug tracking-[-0.03em]">{next.name}</p> : null}
                  <p className={cn("text-ui-body text-muted", next?.name && "mt-1")}>{stepLine(L, next)}</p>
                  {next?.action === "outline" ? (
                    <div className="mt-3 flex flex-wrap items-center gap-2">
                      <Button size="sm" className="whitespace-nowrap" onClick={() => props.onSay(L.sayOutline)}>{L.startOutline}</Button>
                      <Button size="sm" variant="outline" className="whitespace-nowrap" disabled={drafting} onClick={() => { void draft(); }}>{drafting ? L.drafting : L.draftOutline}</Button>
                      <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => setEditing({ modules: [], hint: "", description: "" })}>{L.writeOutline}</Button>
                    </div>
                  ) : next?.action !== "complete" ? (
                    <div className="mt-3">
                      <Button size="sm" className="whitespace-nowrap" disabled={busy} onClick={() => { void start(); }}>
                        {next?.action === "review" ? L.startReview : next?.action === "probe" || next?.action === "teach" ? L.start : L.continueStudy}
                      </Button>
                    </div>
                  ) : null}
                </div>
              )}
            </section>

            {total > 0 ? (
              <section className="flex flex-col gap-4">
                <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.outline}</p>
                {path.modules.map((mod) => (
                  <div key={mod.id}>
                    <div className="flex items-baseline justify-between gap-3">
                      <h3 className="min-w-0 truncate text-ui-body font-medium">{mod.name}</h3>
                      <span className="shrink-0 tabular-nums text-ui-footnote text-muted">{mod.mastered}/{mod.total}</span>
                    </div>
                    {mod.objective ? <p className="mt-0.5 text-ui-footnote leading-relaxed text-muted">{mod.objective}</p> : null}
                    <ul className="mt-1">
                      {mod.points.map((pt) => (
                        <PointRow
                          key={pt.id}
                          pathId={path.id}
                          point={pt}
                          open={openPoint === pt.id}
                          revision={path.updated_at}
                          onToggle={() => setOpenPoint((cur) => (cur === pt.id ? "" : pt.id))}
                          onChanged={() => { void props.onReload(); }}
                        />
                      ))}
                    </ul>
                  </div>
                ))}
              </section>
            ) : (
              <EmptyState title={L.outlineEmpty} body={L.outlineEmptyHint} />
            )}

            {total > 0 ? (
              <section className="flex flex-col gap-2">
                <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.reviews}</p>
                {path.reviews.length === 0 ? (
                  <p className="text-ui-body text-muted">{L.reviewsEmpty}<span className="text-muted/50"> · </span>{L.reviewsEmptyHint}</p>
                ) : (
                  <ul>
                    {path.reviews.map((r) => (
                      <li key={r.point_id} className="flex items-center gap-3 border-b border-border/50 py-2 text-ui-body">
                        <span className="min-w-0 flex-1 truncate">{r.name}</span>
                        <span className="shrink-0 tabular-nums text-ui-footnote text-muted">{fill(L.recall, { n: percent(r.retrievability) })}</span>
                        <span className={cn("w-20 shrink-0 text-right text-ui-footnote", r.due ? "text-warning" : "text-muted")}>
                          {r.due ? L.dueNow : whenLabel(r.due_at, getLocale(), L.today)}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
                <div className="mt-2 flex flex-wrap items-center gap-3">
                  <span className="text-ui-footnote text-muted">{L.retention}</span>
                  <LearnSegment
                    label={L.retention}
                    value={String(RETENTIONS.reduce((best, r) => (Math.abs(r - path.desired_retention) < Math.abs(best - path.desired_retention) ? r : best), RETENTIONS[0]))}
                    onChange={(v) => { void run(() => learn.setRetention(path.id, Number(v))); }}
                    options={RETENTIONS.map((r) => ({ id: String(r), label: `${Math.round(r * 100)}%` }))}
                  />
                  <span className="text-ui-footnote text-muted/70">{L.retentionHint}</span>
                </div>
              </section>
            ) : null}
          </>
        )}
      </div>
      <ConfirmDialog
        open={confirm === "reset"}
        title={L.reset}
        body={L.resetBody}
        confirmLabel={L.reset}
        danger
        onCancel={() => setConfirm("")}
        onConfirm={async () => {
          setConfirm("");
          await run(() => learn.resetPath(path.id));
        }}
      />
      <ConfirmDialog
        open={confirm === "delete"}
        title={L.remove}
        body={L.removeBody}
        confirmLabel={L.remove}
        danger
        onCancel={() => setConfirm("")}
        onConfirm={async () => {
          setConfirm("");
          try {
            await learn.deletePath(path.id);
            props.onDeleted();
          } catch (e) {
            setErr(errMessage(e));
          }
        }}
      />
    </LearnShell>
  );
}

/**
 * PointRow 是大纲里的一个知识点。点开后读出它的记录，并给出声明会了和先放一放两个动作。
 * pathId 和 point 定位知识点。open 为真时展开。revision 变了说明路径有更新，展开着的详情要重读。
 * onToggle 展开或收起。onChanged 在声明或暂缓之后调用。
 */
function PointRow(props: {
  pathId: string;
  point: PointView;
  open: boolean;
  revision: string;
  onToggle: () => void;
  onChanged: () => void;
}) {
  const L = useCopy().learn.mastery;
  const { point: pt, pathId, open } = props;
  const [detail, setDetail] = useState<PointDetail | null>(null);
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    try {
      setDetail(await learn.pointDetail(pathId, pt.id));
      setErr("");
    } catch (e) {
      setErr(errMessage(e));
    }
  }, [pathId, pt.id]);

  useEffect(() => {
    if (open) void load();
  }, [open, load, props.revision, pt.status, pt.mastery]);

  /** 改完这个知识点后重读详情，并让看板重读路径。 */
  const act = async (fn: () => Promise<void>) => {
    try {
      await fn();
      props.onChanged();
      await load();
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  const meta = [kindLabel(L, pt.kind), statusLabel(L, pt.status)];
  if (pt.gate === "quantitative" && pt.attempts > 0 && pt.status !== "mastered") meta.push(`${percent(pt.mastery)}%`);
  if (pt.mastery_source === "learner") meta.push(L.claimed);
  if (pt.deferred) meta.push(L.deferred);
  else if (pt.blocked) meta.push(L.blocked);
  const claimed = pt.mastery_source === "learner";

  return (
    <li className="border-b border-border/50">
      <button type="button" className="flex w-full items-center gap-3 py-2.5 text-left" aria-expanded={open} onClick={props.onToggle}>
        <PointMark status={pt.status} />
        <span className="min-w-0 flex-1">
          <span className={cn("block truncate text-ui-body-l", pt.blocked && pt.status === "new" && "text-muted")}>{pt.name}</span>
          <span className="mt-0.5 block truncate text-ui-footnote text-muted">{meta.join(" · ")}</span>
        </span>
        {open ? <ChevronDown className="size-3.5 shrink-0 text-muted" aria-hidden /> : <ChevronRight className="size-3.5 shrink-0 text-muted" aria-hidden />}
      </button>
      {open ? (
        <div className="flex flex-col gap-2 pb-3 pl-8 text-ui-footnote" data-testid="learn-point-detail">
          <p className="text-muted">{pt.gate === "qualitative" ? L.gateExplain : L.gateScore}</p>
          {detail ? (
            <p className="tabular-nums text-muted">
              {[
                fill(L.attempts, { n: pt.attempts }),
                detail.reps > 0 ? fill(L.reps, { n: detail.reps }) : "",
                detail.lapses > 0 ? fill(L.lapses, { n: detail.lapses }) : "",
                pt.due_at ? fill(L.nextReview, { when: whenLabel(pt.due_at, getLocale(), L.today) }) : "",
              ].filter(Boolean).join(" · ")}
            </p>
          ) : null}
          <div className="flex flex-wrap items-center gap-1.5">
            {pt.status !== "mastered" ? (
              <Button size="xs" variant="outline" onClick={() => { void act(() => learn.claim(pathId, pt.id, true)); }}>{L.claim}</Button>
            ) : claimed ? (
              <Button size="xs" variant="outline" onClick={() => { void act(() => learn.claim(pathId, pt.id, false)); }}>{L.unclaim}</Button>
            ) : null}
            {pt.status !== "mastered" ? (
              <Button size="xs" variant="ghost" onClick={() => { void act(() => learn.defer(pathId, pt.id, !pt.deferred)); }}>{pt.deferred ? L.undefer : L.deferAct}</Button>
            ) : null}
          </div>
          {err ? <p className="text-danger">{err}</p> : null}
          {detail && detail.evidence.length > 0 ? (
            <ul className="mt-1 flex flex-col gap-1.5">
              {detail.evidence.slice(0, 6).map((ev, i) => (
                <li key={`${ev.at}-${i}`} className="flex gap-2">
                  <span className="w-16 shrink-0 text-muted/80">{ev.kind === "qualitative" ? L.evQualitative : ev.kind === "claim" ? L.evClaim : L.evQuiz}</span>
                  <span
                    className={cn(
                      "w-14 shrink-0",
                      ev.result === "correct" ? "text-success" : ev.result === "partial" ? "text-warning" : ev.result === "incorrect" ? "text-danger" : "text-muted",
                    )}
                  >
                    {ev.result === "correct" ? L.resCorrect : ev.result === "partial" ? L.resPartial : ev.result === "incorrect" ? L.resIncorrect : ev.result === "on" ? L.resOn : L.resOff}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-muted">
                    {ev.prompt || ev.answer || ""}{ev.kind === "quiz" && !ev.independent ? ` · ${L.hinted}` : ""}
                  </span>
                </li>
              ))}
            </ul>
          ) : detail ? (
            <p className="text-muted/80">{L.historyEmpty}</p>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}
