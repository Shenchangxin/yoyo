/** 拥有这一轮的活动和用量药丸。不拥有计费。只投影统计，不改服务端。 */
import { memo, useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Database, Gauge } from "lucide-react";
import type { Copy } from "../../lib/copy";
import type { ContextUsage } from "../../lib/protocol";
import type { TurnStatus } from "../../lib/turn-status";
import {
  billedInputTokens,
  formatCacheHitPercent,
  formatCompactDuration,
  formatCompactTokens,
  formatExactTokens,
  formatTokensPerSecond,
  type SessionStats,
} from "../../lib/session-stats";

/** 管一颗药丸的弹出说明开合。 */
function useStatDialog() {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLSpanElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);

  useEffect(() => {
    if (!open || !rootRef.current) return;
    const r = rootRef.current.getBoundingClientRect();
    const left = Math.min(Math.max(12, r.left), window.innerWidth - 312);
    setPos({ top: r.top - 8, left });
  }, [open]);

  useEffect(() => {
    if (!open) return;
    /** 点到药丸和面板外面就收起。 */
    const onDoc = (e: PointerEvent) => {
      const t = e.target as Node;
      if (rootRef.current?.contains(t) || panelRef.current?.contains(t)) return;
      setOpen(false);
    };
    /** Escape 收起面板。 */
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return { open, setOpen, rootRef, panelRef, pos };
}

/** 把说明画到药丸旁边。 */
function StatDialog(props: {
  title: string;
  icon: ReactNode;
  titleValue?: string;
  children: ReactNode;
  panelRef: React.RefObject<HTMLDivElement | null>;
  pos: { top: number; left: number };
}) {
  return createPortal(
    <div
      ref={props.panelRef}
      className="sc-stat-panel"
      role="dialog"
      aria-label={props.title}
      style={{ left: props.pos.left, top: props.pos.top, transform: "translateY(-100%)" }}
    >
      <div className="sc-stat-title">
        <span className="sc-stat-title-label">
          {props.icon}
          {props.title}
        </span>
        {props.titleValue ? <span className="sc-stat-title-value">{props.titleValue}</span> : null}
      </div>
      <div className="sc-stat-rule" aria-hidden />
      <dl className="sc-stat-details">{props.children}</dl>
    </div>,
    document.body,
  );
}

/** 画说明里的一行标签和值。 */
function Row({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </>
  );
}

/** 把两段用间隔点开，第二段空就只留第一段。 */
function joined(first: string, second: string | null): ReactNode {
  if (!second) return first;
  return (
    <>
      {first}
      <span className="sc-pill-sep" aria-hidden>·</span>
      {second}
    </>
  );
}

/** 有解码统计就用它，否则用当场的速度。 */
function decodeSpeed(stats: SessionStats, liveTps: number, copy: Copy): string | null {
  if (stats.decodeMs > 0 && stats.decodeTokens > 0) {
    const tps = stats.decodeTokens / (stats.decodeMs / 1_000);
    return copy.composer.tokensPerSec.replace("{n}", formatTokensPerSecond(tps));
  }
  if (liveTps > 0) return copy.composer.tokensPerSec.replace("{n}", formatTokensPerSecond(liveTps));
  return null;
}

/** 用这一轮已用记号和已过时间算出速度。 */
function liveTokensPerSec(ctx: ContextUsage | undefined, turnStatus: TurnStatus | null | undefined): number {
  const tok = ctx?.tokens || 0;
  const elapsed = turnStatus?.since ? Math.max(0, Date.now() - turnStatus.since) : 0;
  const outTokens = Math.max(0, tok - (ctx?.prefixTokens || 0) - (ctx?.schemaTokens || 0));
  if (elapsed <= 400 || outTokens <= 0) return 0;
  return (outTokens / elapsed) * 1000;
}

/** 把记号数收成简短文案。 */
function tokenCount(n: number, copy: Copy): string {
  return copy.composer.tokenCount.replace("{count}", formatCompactTokens(n));
}

/** 把记号数收成完整文案。 */
function exactCount(n: number, copy: Copy): string {
  return copy.composer.tokenCount.replace("{count}", formatExactTokens(n));
}

/** 画活动药丸的图标。 */
function GaugeIcon() {
  return <Gauge size={14} strokeWidth={1} className="shrink-0 opacity-80" />;
}

/** 画用量药丸的图标。 */
function DatabaseIcon() {
  return <Database size={14} strokeWidth={1} className="shrink-0 opacity-80" />;
}

/** 画一颗不打开说明的药丸。 */
function PlainPill({ stat, icon, label }: { stat: string; icon: ReactNode; label: ReactNode }) {
  return (
    <span className="sc-pill-anchor" data-composer-stat={stat}>
      <span className="sc-pill sc-pill-static">
        {icon}
        <span>{label}</span>
      </span>
    </span>
  );
}

/** StatsPills 投影活动和用量两颗药丸。 */
export const StatsPills = memo(function StatsPills(props: {
  stats?: SessionStats | null;
  ctx?: ContextUsage;
  turnStatus?: TurnStatus | null;
  copy: Copy;
  compact?: boolean;
}) {
  const stats = props.stats;
  const copy = props.copy;
  const liveTps = liveTokensPerSec(props.ctx, props.turnStatus);
  const speed = stats ? decodeSpeed(stats, liveTps, copy) : (liveTps > 0 ? copy.composer.tokensPerSec.replace("{n}", formatTokensPerSecond(liveTps)) : null);
  const activity = useStatDialog();
  const usage = useStatDialog();

  const billed = stats ? billedInputTokens(stats) : 0;
  const output = stats?.outputTokens || 0;
  const hasUsage = billed > 0 || output > 0;
  const cacheHit = stats && billed > 0 ? formatCacheHitPercent(stats.cacheReadTokens, billed) : null;
  const cacheHitText = cacheHit != null ? copy.composer.cacheHit.replace("{percent}", cacheHit) : null;
  const steps = stats?.steps || 0;
  const showActivity = props.compact ? speed != null : steps > 0;
  if (!showActivity && !hasUsage) return null;

  const counts = stats
    ? copy.composer.statsCounts.replace("{turns}", String(stats.turns)).replace("{steps}", String(steps))
    : "";
  const total = billed + output;
  const totalText = hasUsage ? tokenCount(total, copy) : null;
  const compact = !!props.compact;

  return (
    <div className="sc-pills" data-testid="composer-stats">
      {showActivity ? (
        compact ? (
          speed ? <PlainPill stat="activity" icon={<GaugeIcon />} label={speed} /> : null
        ) : (
          <ActivityButton
            counts={counts}
            speed={speed}
            stats={stats}
            copy={copy}
            dialog={activity}
          />
        )
      ) : null}
      {hasUsage ? (
        compact ? (
          cacheHitText ? <PlainPill stat="usage" icon={<DatabaseIcon />} label={cacheHitText} /> : null
        ) : (
          <UsageButton
            totalText={totalText || ""}
            cacheHitText={cacheHitText}
            cacheHit={cacheHit}
            stats={stats}
            total={total}
            copy={copy}
            dialog={usage}
          />
        )
      ) : null}
    </div>
  );
});

/** 画活动药丸，打开后列出轮次和步。 */
function ActivityButton(props: {
  counts: string;
  speed: string | null;
  stats?: SessionStats | null;
  copy: Copy;
  dialog: ReturnType<typeof useStatDialog>;
}) {
  const stats = props.stats;
  const label = joined(props.counts, props.speed);
  const aria = props.speed ? `${props.counts} · ${props.speed}` : props.counts;
  const emptyDialog = !stats || (stats.llmMs <= 0 && stats.toolMs <= 0 && stats.ttftSteps <= 0 && !props.speed);
  if (emptyDialog) return <PlainPill stat="activity" icon={<GaugeIcon />} label={label} />;
  return (
    <span ref={props.dialog.rootRef} className="sc-pill-anchor" data-composer-stat="activity">
      <button
        type="button"
        className="sc-pill"
        aria-haspopup="dialog"
        aria-expanded={props.dialog.open}
        aria-label={aria}
        onClick={() => props.dialog.setOpen(!props.dialog.open)}
      >
        <GaugeIcon />
        <span>{label}</span>
      </button>
      {props.dialog.open && props.dialog.pos ? (
        <StatDialog
          title={props.copy.composer.statsDialogTitle}
          icon={<GaugeIcon />}
          panelRef={props.dialog.panelRef}
          pos={props.dialog.pos}
        >
          {stats.llmMs > 0 ? <Row label={props.copy.composer.statsLlmTime} value={formatCompactDuration(stats.llmMs)} /> : null}
          {stats.toolMs > 0 ? <Row label={props.copy.composer.statsToolTime} value={formatCompactDuration(stats.toolMs)} /> : null}
          {stats.ttftSteps > 0 ? <Row label={props.copy.composer.statsTtft} value={formatCompactDuration(stats.ttftMs / stats.ttftSteps)} /> : null}
          {props.speed ? <Row label={props.copy.composer.statsSpeed} value={props.speed} /> : null}
        </StatDialog>
      ) : null}
    </span>
  );
}

/** 画用量药丸，打开后列出记号和缓存。 */
function UsageButton(props: {
  totalText: string;
  cacheHitText: string | null;
  cacheHit: string | null;
  stats?: SessionStats | null;
  total: number;
  copy: Copy;
  dialog: ReturnType<typeof useStatDialog>;
}) {
  const stats = props.stats;
  if (!stats) return null;
  const label = joined(props.totalText, props.cacheHitText);
  const aria = props.cacheHitText ? `${props.totalText} · ${props.cacheHitText}` : props.totalText;
  return (
    <span ref={props.dialog.rootRef} className="sc-pill-anchor" data-composer-stat="usage">
      <button
        type="button"
        className="sc-pill"
        aria-haspopup="dialog"
        aria-expanded={props.dialog.open}
        aria-label={aria}
        onClick={() => props.dialog.setOpen(!props.dialog.open)}
      >
        <DatabaseIcon />
        <span>{label}</span>
      </button>
      {props.dialog.open && props.dialog.pos ? (
        <StatDialog
          title={props.copy.composer.statsUsageTitle}
          icon={<DatabaseIcon />}
          titleValue={exactCount(props.total, props.copy)}
          panelRef={props.dialog.panelRef}
          pos={props.dialog.pos}
        >
          {props.cacheHit != null ? <Row label={props.copy.composer.ctxCache} value={`${props.cacheHit}%`} /> : null}
          <Row label={props.copy.composer.statsInput} value={exactCount(stats.uncachedInputTokens, props.copy)} />
          <Row label={props.copy.composer.statsCacheRead} value={exactCount(stats.cacheReadTokens, props.copy)} />
          {stats.cacheWriteTokens > 0 ? <Row label={props.copy.composer.statsCacheWrite} value={exactCount(stats.cacheWriteTokens, props.copy)} /> : null}
          <Row label={props.copy.composer.statsOutput} value={exactCount(stats.outputTokens, props.copy)} />
        </StatDialog>
      ) : null}
    </span>
  );
}
