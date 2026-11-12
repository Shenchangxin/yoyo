/**
 * 拥有：练习的统计这一页：记住率、每天的复习量、未来七天的到期预报和题目的来源。
 * 不拥有：数字怎么算，那在引擎里。这里只读不写。
 */

import { useCallback, useEffect, useState } from "react";
import { getLocale, useCopy } from "../../../lib/i18n";
import * as practice from "../../../lib/learn-practice";
import type { Insights } from "../../../lib/learn-practice";
import { subscribeLearn } from "../../../lib/learn-events";
import { fill, percent } from "../../../lib/learn-format";
import { barHeights } from "../../../lib/learn-practice-util";
import { errMessage } from "../../../lib/normalize";
import { EmptyState } from "../../../components/ui/empty-state";
import { cn } from "../../../lib/utils";
import { LearnSegment } from "../LearnShell";
import { sourceLabel } from "./parts";

/** dayOf 把 2026-03-02 这样的日期读成本地那一天。 */
function dayOf(date: string): Date {
  const [y, m, d] = date.split("-").map(Number);
  return new Date(y || 1970, (m || 1) - 1, d || 1);
}

/** Stat 是一个数字和它下面的一行说明。 */
function Stat(props: { value: string; label: string; hint?: string }) {
  return (
    <div className="min-w-0">
      <p className="text-ui-title font-semibold tabular-nums tracking-[-0.03em]">{props.value}</p>
      <p className="mt-0.5 truncate text-ui-footnote text-muted">{props.label}</p>
      {props.hint ? <p className="text-ui-caption leading-snug text-muted/70">{props.hint}</p> : null}
    </div>
  );
}

/** PracticeInsights 画出最近一段时间的练习统计。题库有变化时自己重读。 */
export function PracticeInsights() {
  const L = useCopy().learn.drill;
  const [days, setDays] = useState("30");
  const [data, setData] = useState<Insights | null>(null);
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await practice.insights(Number(days)));
      setErr("");
    } catch (e) {
      setErr(errMessage(e));
    }
  }, [days]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => subscribeLearn(/** 评了分或题库变了就重读。 */ (change) => {
    if (change.scope === "practice") void load();
  }), [load]);

  const locale = getLocale();
  const short = new Intl.DateTimeFormat(locale, { month: "short", day: "numeric" });
  const weekday = new Intl.DateTimeFormat(locale, { weekday: "short" });
  const daily = data?.daily || [];
  const reviews = daily.reduce((n, d) => n + d.reviews, 0);
  const fresh = daily.reduce((n, d) => n + d.new_cards, 0);
  const mistakes = daily.reduce((n, d) => n + d.mistakes, 0);
  const heights = barHeights(daily.map((d) => d.reviews));
  const forecast = data?.forecast || [];
  const forecastHeights = barHeights(forecast.map((f) => f.due));

  return (
    <div className="mx-auto flex max-w-[720px] flex-col gap-7 px-8 py-6" data-testid="learn-insights">
      {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
      <div>
        <LearnSegment
          label={L.windowLabel}
          value={days}
          onChange={setDays}
          options={[{ id: "7", label: L.days7 }, { id: "30", label: L.days30 }, { id: "90", label: L.days90 }]}
        />
      </div>
      {data ? (
        <>
          <div className="grid grid-cols-4 gap-4">
            <Stat
              value={data.retention.reviews > 0 ? `${percent(data.retention.rate)}%` : "—"}
              label={L.statRetention}
              hint={data.retention.reviews > 0 ? fill(L.statRetentionHint, { n: data.retention.reviews }) : undefined}
            />
            <Stat value={String(reviews)} label={L.statReviews} />
            <Stat value={String(fresh)} label={L.statNew} />
            <Stat value={String(mistakes)} label={L.statMistakes} />
          </div>

          <section className="flex flex-col gap-2">
            <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.daily}</p>
            {reviews === 0 ? <EmptyState title={L.insightsEmpty} body="" /> : (
              <>
                <div className="flex h-28 items-end gap-px border-b border-border/60" role="img" aria-label={L.daily}>
                  {daily.map((d, i) => {
                    const againShare = d.reviews > 0 ? (d.again / d.reviews) * 100 : 0;
                    return (
                      <div
                        key={d.date}
                        className="flex min-w-0 flex-1 flex-col justify-end"
                        style={{ height: `${heights[i]}%` }}
                        title={fill(L.dailyBar, { date: short.format(dayOf(d.date)), reviews: d.reviews, again: d.again })}
                      >
                        {againShare > 0 ? <div className="w-full bg-danger/60" style={{ height: `${againShare}%` }} /> : null}
                        <div className="w-full flex-1 bg-foreground/25" />
                      </div>
                    );
                  })}
                </div>
                <div className="flex justify-between text-ui-caption tabular-nums text-muted/70">
                  <span>{daily[0] ? short.format(dayOf(daily[0].date)) : ""}</span>
                  <span>{L.today}</span>
                </div>
              </>
            )}
          </section>

          <section className="flex flex-col gap-2">
            <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.forecast}</p>
            <div className="grid grid-cols-7 gap-2">
              {forecast.map((f, i) => (
                <div key={f.date} className="flex flex-col items-center gap-1" title={fill(L.forecastBar, { date: short.format(dayOf(f.date)), n: f.due })}>
                  <span className={cn("text-ui-footnote tabular-nums", f.due > 0 ? "text-foreground" : "text-muted/60")}>{f.due}</span>
                  <div className="flex h-14 w-full items-end border-b border-border/60">
                    <div className={cn("w-full", i === 0 ? "bg-foreground/45" : "bg-foreground/20")} style={{ height: `${forecastHeights[i]}%` }} />
                  </div>
                  <span className="text-ui-caption text-muted/70">{i === 0 ? L.today : weekday.format(dayOf(f.date))}</span>
                </div>
              ))}
            </div>
          </section>

          {data.sources.length > 0 ? (
            <section className="flex flex-col gap-1">
              <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.sources}</p>
              <ul>
                {data.sources.map((s) => (
                  <li key={s.source} className="flex items-center justify-between gap-3 border-b border-border/50 py-2 text-ui-body">
                    <span className="min-w-0 truncate">{sourceLabel(L, s.source)}</span>
                    <span className="shrink-0 tabular-nums text-muted">{s.count}</span>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
        </>
      ) : null}
    </div>
  );
}
