// 本文件拥有改进工作区的四个页签、总览、谱系和哈希复制。不拥有下一步怎么判定，那一步在 harnessNext。调用方传入 tab、harness、report 和各页 children。

import { useState, type ReactNode } from "react";
import { Check, Copy } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { IconSwap } from "../../components/ui/icon-swap";
import { DURATION_SHELL, motionTransition, useMotionReduced } from "../../lib/motion";
import { useCopy } from "../../lib/i18n";
import { writeClipboard } from "../../lib/clipboard";
import {
  canaryDirty,
  formatStamp,
  harnessNext,
  parseHarnessRefs,
  parseLineage,
  readHarborMetrics,
  shortHash,
  stagingDirty,
  type HarnessNext,
  type HarnessRefs,
  type LineageNode,
} from "../../lib/harness-refs";
import { checkHeadline, pointerLabel } from "../../lib/harness-copy";
import type { HarnessTab } from "../../lib/protocol";
import { HARNESS_TABS } from "../../lib/surface";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { LabEyebrow, LabFrame, StoryRow } from "../labs/LabFrame";
import { MaterialChangeList } from "./Changes";

// 提出、证明、采用带 01、02、03。总览没有序号。
const STEP_INDEX: Partial<Record<HarnessTab, string>> = {
  propose: "01",
  prove: "02",
  promote: "03",
};

/** HarnessWorkspace 画页签。总览用 HarnessOverview，其余三页直接放 children 里对应的一块。props 读取 harness、fallbackActive、report、tab、onTab、onReveal 等，调用 onTab。 */
export function HarnessWorkspace(props: {
  tab: HarnessTab;
  onTab: (tab: HarnessTab) => void;
  harness: unknown;
  fallbackActive?: string;
  report: unknown;
  onReveal?: (hash?: string) => Promise<void> | void;
  children: Record<Exclude<HarnessTab, "overview">, ReactNode>;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const underline = motionTransition(reduced, DURATION_SHELL);
  const refs = parseHarnessRefs(props.harness, props.fallbackActive);
  const next = harnessNext(refs, props.report);
  const labels: Record<HarnessTab, { label: string; hint: string }> = {
    overview: { label: copy.rsi.overview, hint: copy.rsi.overviewHint },
    propose: { label: copy.rsi.propose, hint: copy.rsi.proposeHint },
    prove: { label: copy.rsi.prove, hint: copy.rsi.proveHint },
    promote: { label: copy.rsi.promote, hint: copy.rsi.promoteHint },
  };

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="harness-workspace">
      <div
        className="process-tabs flex h-11 shrink-0 items-stretch gap-0.5 px-2"
        role="tablist"
        aria-label={copy.rsi.title}
      >
        {HARNESS_TABS.map((id) => {
          const on = props.tab === id;
          const n = STEP_INDEX[id];
          return (
            <button
              type="button"
              key={id}
              role="tab"
              title={labels[id].hint}
              aria-label={labels[id].label}
              aria-selected={on}
              className={cn(
                "relative flex h-full items-center gap-1.5 rounded-md px-2.5 text-[13px] font-medium transition-colors",
                on ? "text-foreground" : "text-muted hover:text-foreground",
              )}
              onClick={() => props.onTab(id)}
            >
              {n ? (
                <span className="font-mono text-[10px] tabular-nums text-muted/70" aria-hidden>
                  {n}
                </span>
              ) : null}
              {labels[id].label}
              {on ? (
                <motion.span
                  layoutId="harness-tab-underline"
                  className="absolute inset-x-2 -bottom-px h-[2px] rounded-full bg-accent"
                  transition={underline}
                  aria-hidden
                />
              ) : null}
            </button>
          );
        })}
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        {props.tab === "overview" ? (
          <HarnessOverview refs={refs} harness={props.harness} report={props.report} next={next} onTab={props.onTab} onReveal={props.onReveal} />
        ) : props.tab === "propose" ? (
          props.children.propose
        ) : props.tab === "prove" ? (
          props.children.prove
        ) : (
          props.children.promote
        )}
      </div>
    </div>
  );
}

/**
 * HarnessOverview 按 next.action 选原因和要去的页：checkout 去采用，eval 和 blocked 去证明，其余去提出。实验室开关打开才多显示 head，谱系上的打开按钮也只在那时传下去。
 * props 读取 report、refs、harness、next、onTab、onReveal，调用 onTab。
 */
function HarnessOverview(props: {
  refs: HarnessRefs;
  harness: unknown;
  report: unknown;
  next: HarnessNext;
  onTab: (tab: HarnessTab) => void;
  onReveal?: (hash?: string) => Promise<void> | void;
}) {
  const copy = useCopy();
  const lab = useUI((s) => s.showHarnessLab);
  const metrics = readHarborMetrics(props.report);
  const dirty = stagingDirty(props.refs);
  const canary = canaryDirty(props.refs);
  const lineage = parseLineage(props.harness);
  const reason =
    props.next.action === "blocked" ? copy.rsi.nextBlocked
      : props.next.action === "checkout" ? (dirty ? copy.rsi.nextCheckout : copy.rsi.nextCanaryCheckout)
        : props.next.action === "eval" ? (dirty ? copy.rsi.nextEval : copy.rsi.nextCanaryEval)
          : props.next.action === "idle" ? copy.rsi.nextIdle
            : copy.rsi.nextEvolve;
  const goLabel =
    props.next.action === "checkout" ? copy.rsi.promote
      : props.next.action === "eval" || props.next.action === "blocked" ? copy.rsi.prove
        : copy.rsi.propose;
  const pointers = [
    { name: copy.rsi.active, hash: props.refs.active, kind: "active" as const, hint: copy.rsi.versionInUseHint },
    { name: copy.rsi.staging, hash: props.refs.staging, kind: "staging" as const, hint: copy.rsi.versionDraftHint },
    { name: copy.rsi.canary, hash: props.refs.canary, kind: "canary" as const, hint: copy.rsi.versionCandidateHint },
    ...(lab ? [{ name: copy.rsi.head, hash: props.refs.head, kind: "head" as const, hint: copy.rsi.head }] : []),
  ].filter((c) => c.hash);

  return (
    <LabFrame>
      <div data-testid="harness-overview">
        <div
          className={cn(
            "mb-6 flex flex-wrap items-center justify-between gap-3 rounded-md bg-sidebar/40 px-4 py-3",
            props.next.action === "blocked" ? "border-danger/30" : "border-border/80",
          )}
        >
          <div className="min-w-0">
            <LabEyebrow>{copy.rsi.next}</LabEyebrow>
            <p className="mt-1 max-w-[46ch] text-[15px] font-medium leading-snug tracking-[-0.02em] text-pretty text-foreground">{reason}</p>
          </div>
          <Button onClick={() => props.onTab(props.next.tab)}>{goLabel}</Button>
        </div>
        {pointers.length === 0 ? (
          <FirstRun onTab={props.onTab} />
        ) : (
          <div className="mb-6 overflow-hidden rounded-md bg-sidebar/40" data-testid="harness-versions">
            {pointers.map((c, i) => (
              <div key={c.name + c.hash} className={cn("grid grid-cols-1 items-start gap-1.5 px-4 py-2.5 sm:grid-cols-[7.5rem_minmax(0,1fr)_auto] sm:items-center sm:gap-3", i > 0 && "border-t border-border/70")}>
                <div className="text-[13px] font-medium">{c.name}</div>
                <div className="min-w-0 text-[12px] text-muted">
                  {lab ? <HashLine hash={c.hash} compact /> : <span className="truncate">{c.hint}</span>}
                </div>
                <div className="shrink-0">
                  {c.kind === "active" ? (
                    <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-foreground">{copy.labs.trusted}</span>
                  ) : null}
                  {c.kind === "staging" ? (
                    <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">
                      {dirty ? copy.rail.stagingDirty : copy.labs.notTrusted}
                    </span>
                  ) : null}
                  {c.kind === "canary" ? (
                    <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">
                      {canary ? copy.rsi.canaryDirty : copy.labs.notTrusted}
                    </span>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        )}
        <div className="mb-2 flex flex-wrap items-end justify-between gap-2">
          <div>
            <h3 className="text-[13px] font-medium">{copy.rsi.lineage}</h3>
            <p className="mt-0.5 max-w-[62ch] text-[12px] text-muted">{copy.rsi.lineageHint}</p>
          </div>
          {props.onReveal ? (
            <div className="flex flex-wrap gap-1">
              <Button size="sm" variant="ghost" onClick={() => void reveal(props.onReveal, props.refs.active)}>{copy.rsi.openArtifacts}</Button>
              <Button size="sm" variant="ghost" onClick={() => void reveal(props.onReveal, "")}>{copy.rsi.openStore}</Button>
            </div>
          ) : null}
        </div>
        {lineage.length === 0 ? (
          <p className="mb-6 text-[13px] text-muted">{copy.rsi.noLineage}</p>
        ) : (
          <div className="lineage-rail mb-6 overflow-hidden rounded-md bg-sidebar/40 pl-1" data-testid="harness-lineage">
            {lineage.map((n, i) => (
              <LineageRow key={n.hash} node={n} first={i === 0} onReveal={lab ? props.onReveal : undefined} showIds={lab} />
            ))}
          </div>
        )}
        {metrics ? (
          <>
            <h3 className="mb-3 text-[13px] font-medium">{copy.rsi.lastEval} · {checkHeadline(copy, metrics)}</h3>
            <div className="overflow-hidden rounded-md bg-sidebar/40">
              <StoryRow
                step="1"
                label={copy.harbor.heldIn}
                detail={copy.harbor.storyHeldIn}
                value={`${metrics.heldInPass}/${metrics.heldInTotal}`}
                ok={metrics.heldInTotal === 0 || metrics.heldInPass === metrics.heldInTotal}
              />
              <StoryRow
                step="2"
                label={copy.harbor.heldOut}
                detail={copy.harbor.storyHeldOut}
                value={`${metrics.heldOutPass}/${metrics.heldOutTotal}`}
                ok={metrics.heldOutTotal === 0 || metrics.heldOutPass === metrics.heldOutTotal}
              />
              <StoryRow
                step="3"
                label={copy.harbor.safety}
                detail={copy.harbor.storySafety}
                value={String(metrics.safetyFail)}
                ok={metrics.safetyFail === 0}
                last
              />
            </div>
          </>
        ) : null}
      </div>
    </LabFrame>
  );
}

/** FirstRun 没有指针时列出提出、证明、采用三步。第一步用 default 按钮，后两步用 lift。onTab 被调用，并传入 s.tab。 */
function FirstRun({ onTab }: { onTab: (tab: HarnessTab) => void }) {
  const copy = useCopy();
  const steps: { tab: HarnessTab; n: string; hint: string }[] = [
    { tab: "propose", n: "01", hint: copy.rsi.firstImprove },
    { tab: "prove", n: "02", hint: copy.rsi.firstCheck },
    { tab: "promote", n: "03", hint: copy.rsi.firstUse },
  ];
  const labels: Record<HarnessTab, string> = {
    overview: copy.rsi.overview,
    propose: copy.rsi.propose,
    prove: copy.rsi.prove,
    promote: copy.rsi.promote,
  };
  return (
    <div className="mb-6 overflow-hidden rounded-md bg-sidebar/40" data-testid="harness-first-run">
      <div className="px-4 py-3">
        <h3 className="text-[13px] font-medium">{copy.rsi.firstTitle}</h3>
        <p className="mt-1 max-w-[52ch] text-[12px] text-muted">{copy.rsi.firstLead}</p>
      </div>
      {steps.map((s, i) => (
        <div key={s.tab} className={cn("flex flex-wrap items-center gap-3 px-4 py-3", "border-t border-border/70")}>
          <span className="font-mono text-[10px] tabular-nums text-muted/70" aria-hidden>{s.n}</span>
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-medium">{labels[s.tab]}</div>
            <p className="mt-0.5 text-[12px] text-muted">{s.hint}</p>
          </div>
          <Button size="sm" variant={i === 0 ? "default" : "lift"} onClick={() => onTab(s.tab)}>{labels[s.tab]}</Button>
        </div>
      ))}
    </div>
  );
}

/**
 * LineageRow 种子且没有变更时不画变更列表。有父节点但没有变更时才用空文案。标题依次取 note、proposalId、对照父节点或种子。
 * node 读取 createdAt、note、proposalId、parent、hash、seed 等。
 * first 为假时走另一支。
 * onReveal 为真时才继续，传给 reveal。
 * showIds 为真时才继续。
 */
function LineageRow({ node, first, onReveal, showIds }: { node: LineageNode; first: boolean; onReveal?: (hash?: string) => Promise<void> | void; showIds: boolean }) {
  const copy = useCopy();
  const stamp = formatStamp(node.createdAt);
  const title = node.note || node.proposalId || (node.parent ? copy.rsi.vsParent : copy.rsi.seed);
  return (
    <div className={cn("relative px-4 py-3 pl-8", !first && "border-t border-border/70")} data-testid="harness-lineage-node">
      <span className="absolute left-[10px] top-[18px] size-1.5 rounded-full bg-foreground/80" aria-hidden />
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            {showIds ? <HashLine hash={node.hash} compact /> : <span className="text-[13px] font-medium">{title}</span>}
            {node.seed ? <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">{copy.rsi.seed}</span> : null}
            {node.refs.map((r) => (
              <span key={r} className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">{pointerLabel(copy, r)}</span>
            ))}
            {node.fromArchive ? (
              <span className="text-[11px] text-muted">{node.accepted ? copy.labs.accepted : copy.labs.rejected}</span>
            ) : null}
          </div>
          <div className="mt-0.5 text-[12px] text-muted">
            {showIds ? title : null}
            {stamp ? `${showIds ? " · " : ""}${stamp}` : ""}
          </div>
        </div>
        {onReveal ? (
          <Button size="sm" variant="ghost" onClick={() => void reveal(onReveal, node.hash)}>{copy.rsi.openArtifacts}</Button>
        ) : null}
      </div>
      <div className="mt-2">
        {node.seed && !node.changes.length ? null : (
          <MaterialChangeList changes={node.changes} empty={node.parent ? copy.rsi.noChanges : undefined} />
        )}
      </div>
    </div>
  );
}

/** HashLine 显示 12 位短哈希。复制成功后勾保持 1200 毫秒，失败时提示复制失败。hash 传给 writeClipboard，传给 shortHash。compact 为假时走另一支。 */
function HashLine({ hash, compact }: { hash: string; compact?: boolean }) {
  const copy = useCopy();
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      className={cn(
        "group inline-flex max-w-full min-w-0 cursor-pointer items-center gap-1 font-mono text-[11px] text-muted hover:text-foreground",
        !compact && "mt-1",
      )}
      title={copy.rsi.copyHash}
      onClick={async () => {
        const ok = await writeClipboard(hash);
        if (ok) {
          setDone(true);
          window.setTimeout(() => setDone(false), 1200);
        }
        toast.success(ok ? copy.rsi.copied : copy.transcript.copyFailed);
      }}
    >
      <span className="truncate">{shortHash(hash, 12)}</span>
      <IconSwap
        on={done}
        className={cn("size-3 shrink-0 transition-opacity duration-[var(--duration-fast)] ease-[var(--ease-out)]", done ? "opacity-100" : "opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100")}
        off={<Copy className="size-3" aria-hidden />}
        live={<Check className="size-3" aria-hidden />}
      />
    </button>
  );
}

/** reveal 没有回调就返回。出错时不在这里提示，由调用方去弹提示。fn 为假时走另一支，被调用，并传入 hash。hash 传给 fn。 */
async function reveal(fn: ((hash?: string) => Promise<void> | void) | undefined, hash: string) {
  if (!fn) return;
  try {
    await fn(hash);
  } catch {
    /* 调用方自己提示 */
  }
}
