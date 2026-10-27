// 本文件拥有上下文用量、耗时和最近调用。不拥有这些数字怎么算，那一步在 context-usage。调用方传入 ctx 和 trace。

import { type ReactNode } from "react";
import { RefreshCw } from "lucide-react";
import { RollTime } from "./session-chrome/Counter";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { formatTokens } from "../lib/models-dev";
import {
  cacheHitPct,
  callStats,
  contextBreakdown,
  formatHitPct,
  lastTurnMs,
  modelCallsOf,
  type CtxSliceId,
} from "../lib/context-usage";
import type { ContextUsage, SessionTrace } from "../lib/protocol";
import type { Copy } from "../lib/copy";

// 每种切片对应一条颜色。free 在条上可以缩到 0 宽，图例里仍可能单独出现。
const SLICE_TONE: Record<CtxSliceId, string> = {
  system: "bg-[var(--ctx-system)]",
  rules: "bg-[var(--ctx-rules)]",
  catalog: "bg-[var(--ctx-catalog)]",
  skills: "bg-[var(--ctx-skills)]",
  tools: "bg-[var(--ctx-tools)]",
  memory: "bg-[var(--ctx-memory)]",
  today: "bg-[var(--ctx-today)]",
  notes: "bg-[var(--ctx-notes)]",
  chat: "bg-[var(--ctx-chat)]",
  free: "bg-[var(--ctx-free)]",
};

/** ContextPane 没有 sessionId 时只显示空状态。没有用量、耗时和调用时也显示空状态。命中率至少 70 用好颜色，低于 40 用警告色，没报出来不着色。最近调用只取末尾 8 条并倒序。props 读取 ctx、trace、sessionId、running，调用 onRefresh。 */
export function ContextPane(props: {
  sessionId?: string;
  running?: boolean;
  ctx?: ContextUsage;
  trace?: SessionTrace | null;
  onRefresh?: () => void;
}) {
  const copy = useCopy();
  const br = contextBreakdown(props.ctx);
  const calls = modelCallsOf(props.trace);
  const stats = callStats(calls);
  const hit = cacheHitPct(br.cachedTokens, br.providerPrompt, br.cacheReported);
  const turnMs = lastTurnMs(props.trace?.events);
  const sessionMs = props.trace?.stats.durationMs || 0;
  const hasTokens = br.used > 0 || br.capacity > 0 || br.providerPrompt > 0;
  const hasTiming = !!(stats.last?.elapsedMs || stats.last?.ttftMs || turnMs || sessionMs);
  const empty = !hasTokens && !hasTiming && !calls.length;

  if (!props.sessionId) {
    return <Empty title={copy.review.contextEmpty} hint={copy.review.contextEmptyHint} />;
  }

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="context-pane">
      <div className="flex h-9 shrink-0 items-center justify-between border-b border-border/50 px-2.5">
        <span className="text-[11px] font-medium uppercase tracking-[0.06em] text-muted/70">{copy.review.contextTab}</span>
        <button
          type="button"
          className="inline-flex h-6 items-center gap-1 rounded-md px-1.5 text-[11px] text-muted hover:bg-lift hover:text-foreground"
          onClick={() => props.onRefresh?.()}
          aria-label={copy.review.refreshContext}
        >
          <RefreshCw className={cn("size-3", props.running && "animate-spin")} aria-hidden />
          {copy.review.refresh}
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-3 py-3">
        {empty ? (
          <Empty title={copy.review.contextEmpty} hint={copy.review.contextEmptyHint} />
        ) : (
          <div className="flex flex-col gap-4">
            {hasTokens ? (
              <section className="flex flex-col gap-2.5" data-testid="usage-tokens">
                <WindowMeter br={br} copy={copy} />
                <div className="grid grid-cols-2 gap-1.5">
                  <Fact label={copy.review.usageInput} value={formatTokens(br.providerPrompt || br.used)} />
                  <Fact
                    label={copy.review.usageOutput}
                    value={stats.last?.completionTokens || stats.totalCompletion ? formatTokens(stats.last?.completionTokens || stats.totalCompletion) : "—"}
                  />
                  <Fact
                    label={copy.composer.ctxCache}
                    value={br.cacheReported ? formatTokens(br.cachedTokens) : copy.review.usageUnknown}
                  />
                  <Fact
                    label={copy.review.usageHitRate}
                    value={formatHitPct(hit)}
                    tone={hit == null ? undefined : hit >= 70 ? "good" : hit >= 40 ? undefined : "warn"}
                  />
                </div>
                <p className="text-[11px] leading-[1.5] text-muted">
                  {br.cacheReported
                    ? (br.cacheStable ? copy.review.usageStable : copy.review.usageUnstable)
                    : copy.composer.ctxCacheUnknown}
                  {br.elided > 0 ? ` · ${copy.review.elided} ${br.elided}` : ""}
                  {br.layers.length ? ` · ${br.layers.join(" + ")}` : ""}
                </p>
                {br.note ? <p className="text-[11px] text-muted">{br.note}</p> : null}
              </section>
            ) : null}

            <section className="flex flex-col gap-2.5" data-testid="usage-timing">
              <h3 className="text-[11px] font-medium uppercase tracking-[0.06em] text-muted/70">{copy.trace.duration}</h3>
              <div className="grid grid-cols-2 gap-1.5">
                <Fact label={copy.review.usageLastCall} value={<RollTime ms={stats.last?.elapsedMs || 0} kind="meter" />} />
                <Fact label={copy.review.usageFirstToken} value={<RollTime ms={stats.last?.ttftMs || 0} kind="meter" />} />
                <Fact label={copy.review.usageAvgCall} value={<RollTime ms={stats.avgElapsed} kind="meter" />} />
                <Fact label={copy.review.usageTurn} value={<RollTime ms={turnMs} kind="meter" />} />
              </div>
              {sessionMs > 0 ? (
                <p className="text-[11px] tabular-nums text-muted">
                  {copy.review.usageSession} <RollTime ms={sessionMs} kind="meter" />
                  {stats.avgTtft ? <> · {copy.review.usageFirstToken} <RollTime ms={stats.avgTtft} kind="meter" /></> : null}
                </p>
              ) : null}
            </section>

            {calls.length ? (
              <section className="flex flex-col gap-1.5" data-testid="usage-calls">
                <h3 className="text-[11px] font-medium uppercase tracking-[0.06em] text-muted/70">
                  {copy.review.usageRecent}
                  <span className="ml-1.5 font-normal normal-case tracking-normal text-muted/55">{calls.length}</span>
                </h3>
                <ol className="divide-y divide-border/40 overflow-hidden rounded-lg border border-border/60">
                  {calls.slice(-8).reverse().map((c, i) => {
                    const pct = cacheHitPct(c.cachedTokens, c.promptTokens, c.cacheReported);
                    const label = c.round.replace(/^.*:r/, "r") || `#${calls.length - i}`;
                    return (
                      <li key={`${c.round}-${c.ts}-${i}`} className="flex items-baseline gap-2 px-2.5 py-1.5 text-[12px]">
                        <span className="w-8 shrink-0 font-mono text-[11px] text-muted">{label}</span>
                        <span className="min-w-0 flex-1 truncate tabular-nums text-foreground/85">
                          <RollTime ms={c.elapsedMs} kind="meter" />
                          {c.ttftMs ? <> · {copy.review.usageFirstToken} <RollTime ms={c.ttftMs} kind="meter" /></> : null}
                        </span>
                        <span className="shrink-0 tabular-nums text-[11px] text-muted">{formatHitPct(pct)}</span>
                      </li>
                    );
                  })}
                </ol>
              </section>
            ) : null}
          </div>
        )}
      </div>
    </div>
  );
}

/** WindowMeter 百分比达到 90 用危险色。图例去掉 free，但 chat 即使 token 为 0 也留着。每段至少按 1 份伸展，free 不设最小宽度。br 读取 slices、pct、used、capacity。copy 读取 composer，传给 sliceLabel。 */
function WindowMeter({ br, copy }: { br: ReturnType<typeof contextBreakdown>; copy: Copy }) {
  const visible = br.slices.filter((s) => s.tokens > 0);
  const legend = br.slices.filter((s) => s.id !== "free" && (s.tokens > 0 || s.id === "chat"));
  return (
    <div>
      <div className="mb-1.5 flex items-baseline justify-between gap-2 text-[12px] tabular-nums">
        <span className="text-[11px] font-medium uppercase tracking-[0.06em] text-muted/70">{copy.composer.ctxWindow}</span>
        <span className={cn(br.pct >= 90 ? "text-danger" : "text-foreground/85")}>
          {br.pct}% · {formatTokens(br.used)} / {formatTokens(br.capacity)}
        </span>
      </div>
      <div
        className="flex h-1.5 w-full overflow-hidden rounded-full bg-[var(--ctx-free)]"
        role="meter"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={br.pct}
        aria-label={`${copy.composer.context} ${br.pct}%`}
      >
        {visible.map((s) => (
          <span
            key={s.id}
            className={cn("h-full min-w-[2px]", SLICE_TONE[s.id], s.id === "free" && "min-w-0")}
            style={{ flexGrow: Math.max(s.tokens, 1), flexBasis: 0 }}
          />
        ))}
      </div>
      <div className="mt-2 flex flex-wrap gap-x-2.5 gap-y-1 text-[11px] tabular-nums text-muted">
        {legend.map((s) => (
          <span key={s.id} className="inline-flex items-center gap-1.5">
            <i className={cn("size-1.5 rounded-full", SLICE_TONE[s.id])} aria-hidden />
            <span>{sliceLabel(s.id, copy)}</span>
            <span className="text-foreground/75">{formatTokens(s.tokens)}</span>
          </span>
        ))}
      </div>
    </div>
  );
}

/** sliceLabel 已知切片换成文案，其余当空闲。id 用来做分支。copy 读取 composer。 */
function sliceLabel(id: CtxSliceId, copy: Copy): string {
  switch (id) {
    case "system": return copy.composer.ctxSystem;
    case "rules": return copy.composer.ctxRules;
    case "catalog": return copy.composer.ctxCatalog;
    case "skills": return copy.composer.ctxSkillBody;
    case "tools": return copy.composer.ctxTools;
    case "memory": return copy.composer.ctxMemory;
    case "today": return copy.composer.ctxToday;
    case "notes": return copy.composer.ctxNotes;
    case "chat": return copy.composer.ctxChat;
    default: return copy.composer.ctxFree;
  }
}

/** Fact good 用绿色，warn 用琥珀色，没传 tone 用前景色。label 作为子节点渲染。value 作为子节点渲染。 */
function Fact({ label, value, tone }: { label: string; value: ReactNode; tone?: "good" | "warn" }) {
  return (
    <div className="rounded-lg border border-border/60 bg-lift/30 px-2.5 py-2">
      <div className="text-[10px] font-medium uppercase tracking-[0.05em] text-muted/70">{label}</div>
      <div
        className={cn(
          "mt-0.5 text-[13px] font-medium tabular-nums",
          tone === "good" && "text-emerald-500",
          tone === "warn" && "text-amber-500",
          !tone && "text-foreground/90",
        )}
      >
        {value}
      </div>
    </div>
  );
}

/** Empty 居中的空状态，最小高度 10rem。title 作为子节点渲染。hint 作为子节点渲染。 */
function Empty({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="flex h-full min-h-[10rem] flex-col items-center justify-center px-6 text-center">
      <p className="text-[13px] font-medium text-foreground/85">{title}</p>
      <p className="mt-1 max-w-[22rem] text-[12px] leading-[1.55] text-muted">{hint}</p>
    </div>
  );
}
