/**
 * 拥有：练习这条泳道：现在可练多少、开始一轮复习、日程设置，以及题库和统计两个标签。
 * 不拥有：日程和判分，那在引擎里。也不拥有导师对话，问导师经 onSay。
 */

import { useCallback, useEffect, useState } from "react";
import { getLocale, useCopy } from "../../../lib/i18n";
import * as practice from "../../../lib/learn-practice";
import type { PracticeSummary } from "../../../lib/learn-practice";
import { subscribeLearn } from "../../../lib/learn-events";
import { fill, whenLabel } from "../../../lib/learn-format";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { EmptyState } from "../../../components/ui/empty-state";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../../components/ui/dropdown-menu";
import { LearnSegment, LearnShell, LearnTabs } from "../LearnShell";
import { PracticeInsights } from "./PracticeInsights";
import { QuestionBank, type BankPanel } from "./QuestionBank";
import { ReviewSession, type SessionPlan } from "./ReviewSession";

type Tab = "review" | "bank" | "insights";

const RETENTIONS = [0.8, 0.85, 0.9, 0.95];
const NEW_PER_DAY = [0, 5, 10, 20, 50];

/** nearest 返回 options 里离 value 最近的那个，设置的分段用它定高亮。 */
function nearest(options: number[], value: number): number {
  return options.reduce((best, o) => (Math.abs(o - value) < Math.abs(best - value) ? o : best), options[0]);
}

/**
 * PracticePane 画出练习首页，或者正在进行的一轮复习。sessionId 是练习的导师对话，判分和出题用它的模型。
 * focusId 是从学习计划点进来的那道题，有值时直接开一轮并把它排在最前面；onFocusDone 在这一轮结束后调用。
 * autoStart 为真时进来就直接开一轮，开了之后调 onAutoStarted。
 * onSay 把一句话发给导师。onCard 报出正在练的题干，导师栏的标题跟着它。
 */
export function PracticePane(props: {
  sessionId?: string;
  focusId?: string;
  autoStart?: boolean;
  onAutoStarted?: () => void;
  onFocusDone?: () => void;
  onSay: (text: string) => void;
  onCard?: (stem: string) => void;
}) {
  const L = useCopy().learn.drill;
  const [tab, setTab] = useState<Tab>("review");
  const [sum, setSum] = useState<PracticeSummary | null>(null);
  const [plan, setPlan] = useState<SessionPlan | null>(props.focusId ? { lead: props.focusId } : null);
  const [panel, setPanel] = useState<BankPanel>("");
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    try {
      setSum(await practice.summary());
      setErr("");
    } catch (e) {
      setErr(errMessage(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => subscribeLearn(/** 题库或日程变了就重读汇总。掌握路径上答错的题也会进题库。 */ (change) => {
    if (change.scope === "practice" || change.scope === "path") void load();
  }), [load]);

  useEffect(() => {
    if (props.focusId) setPlan({ lead: props.focusId });
  }, [props.focusId]);

  useEffect(() => {
    // 学习计划上的「开始」带着 autoStart 进来，直接开一轮，省掉在练习首页再点一次。开了就告诉调用方，下次进来不再自动开。
    if (!props.autoStart) return;
    setPlan({});
    props.onAutoStarted?.();
  }, [props.autoStart]);

  /** 结束一轮，回到首页并重读汇总。 */
  const exit = () => {
    setPlan(null);
    props.onFocusDone?.();
    void load();
  };

  /** 改一项日程设置。 */
  const saveSettings = async (patch: { retention?: number; new_per_day?: number }) => {
    try {
      await practice.saveSettings(patch);
      await load();
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  if (plan) {
    return <ReviewSession plan={plan} sessionId={props.sessionId} onExit={exit} onSay={props.onSay} onCard={props.onCard} />;
  }

  const ready = sum ? sum.due + sum.new : 0;
  const tabs = (
    <LearnTabs
      label={L.tabsLabel}
      value={tab}
      onChange={(id) => setTab(id as Tab)}
      items={[
        { id: "review", label: L.tabReview },
        { id: "bank", label: L.tabBank },
        { id: "insights", label: L.tabInsights },
      ]}
    />
  );
  const action = tab === "bank" ? (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button size="sm" className="whitespace-nowrap">{L.add}</Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setPanel("new")}>{L.addOne}</DropdownMenuItem>
        <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setPanel("import")}>{L.importFile}</DropdownMenuItem>
        <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setPanel("generate")}>{L.generate}</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ) : undefined;

  return (
    <LearnShell title={L.title} subtitle={L.subtitle} tabs={tabs} action={action}>
      {tab === "bank" ? (
        <QuestionBank panel={panel} onPanel={setPanel} sessionId={props.sessionId} onPractice={(ids) => { if (ids.length > 0) setPlan({ ids }); }} />
      ) : tab === "insights" ? (
        <PracticeInsights />
      ) : (
        <div className="mx-auto flex max-w-[640px] flex-col gap-6 px-8 py-6" data-testid="learn-practice-desk">
          {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
          {sum ? (
            <>
              <section className="flex items-end justify-between gap-4 border-b border-border/60 pb-5">
                <div className="min-w-0">
                  <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.dueLabel}</p>
                  <p className="mt-1 text-ui-display font-semibold tabular-nums tracking-[-0.03em]" data-testid="learn-practice-ready">{ready}</p>
                  <p className="mt-2 text-ui-body tabular-nums text-muted">{fill(L.dueSplit, { due: sum.due, fresh: sum.new })}</p>
                </div>
                <div className="flex shrink-0 flex-col items-end gap-2">
                  <Button className="whitespace-nowrap" disabled={ready === 0} onClick={() => setPlan({})}>{L.start}</Button>
                  {sum.mistakes > 0 ? (
                    <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => setPlan({ mistakesOnly: true })}>{fill(L.startMistakes, { n: sum.mistakes })}</Button>
                  ) : null}
                </div>
              </section>
              <p className="text-ui-body tabular-nums text-muted">
                {[
                  fill(L.reviewedToday, { n: sum.reviewed_today }),
                  sum.streak_days > 1 ? fill(L.streak, { n: sum.streak_days }) : "",
                  ready === 0 && sum.next_due_at ? fill(L.nextDue, { when: whenLabel(sum.next_due_at, getLocale(), L.today) }) : "",
                ].filter(Boolean).join(" · ")}
              </p>
              {ready === 0 ? (
                sum.total === 0 ? (
                  <EmptyState
                    title={L.emptyBank}
                    body={L.emptyBankHint}
                    action={<Button size="sm" variant="outline" className="whitespace-nowrap" onClick={() => setTab("bank")}>{L.openBank}</Button>}
                  />
                ) : (
                  <EmptyState title={L.clear} body={L.clearHint} />
                )
              ) : null}
              {sum.total > 0 ? (
                <section className="flex flex-col gap-3">
                  <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.settings}</p>
                  <div className="flex flex-wrap items-center gap-3">
                    <span className="w-24 shrink-0 text-ui-footnote text-muted">{L.retention}</span>
                    <LearnSegment
                      label={L.retention}
                      value={String(nearest(RETENTIONS, sum.retention))}
                      onChange={(v) => { void saveSettings({ retention: Number(v) }); }}
                      options={RETENTIONS.map((r) => ({ id: String(r), label: `${Math.round(r * 100)}%` }))}
                    />
                  </div>
                  <div className="flex flex-wrap items-center gap-3">
                    <span className="w-24 shrink-0 text-ui-footnote text-muted">{L.newPerDay}</span>
                    <LearnSegment
                      label={L.newPerDay}
                      value={String(nearest(NEW_PER_DAY, sum.new_per_day))}
                      onChange={(v) => { void saveSettings({ new_per_day: Number(v) }); }}
                      options={NEW_PER_DAY.map((n) => ({ id: String(n), label: String(n) }))}
                    />
                  </div>
                  <p className="text-ui-footnote text-muted/70">{L.retentionHint}</p>
                </section>
              ) : null}
            </>
          ) : null}
        </div>
      )}
    </LearnShell>
  );
}
