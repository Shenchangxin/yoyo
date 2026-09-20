import type { ReactNode } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
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
import type { HarnessTab } from "../../lib/protocol";
import { HARNESS_TABS } from "../../lib/surface";
import { cn } from "../../lib/utils";
import { LabEyebrow, LabFrame, LabStat } from "../labs/LabFrame";
import { MaterialChangeList } from "./Changes";

const STEP_INDEX: Partial<Record<HarnessTab, string>> = {
  propose: "01",
  prove: "02",
  promote: "03",
};

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
        className="process-tabs flex h-10 shrink-0 items-stretch gap-0.5 px-2"
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
                "relative flex h-full items-center gap-1.5 rounded-md px-2.5 text-[12.5px] font-medium transition-colors",
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
              <span
                className={cn(
                  "absolute inset-x-2 -bottom-px h-[1.5px] rounded-full bg-foreground transition-opacity duration-150",
                  on ? "opacity-100" : "opacity-0",
                )}
                aria-hidden
              />
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

function HarnessOverview(props: {
  refs: HarnessRefs;
  harness: unknown;
  report: unknown;
  next: HarnessNext;
  onTab: (tab: HarnessTab) => void;
  onReveal?: (hash?: string) => Promise<void> | void;
}) {
  const copy = useCopy();
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
    { name: copy.rsi.active, hash: props.refs.active, kind: "active" as const },
    { name: copy.rsi.staging, hash: props.refs.staging, kind: "staging" as const },
    { name: copy.rsi.canary, hash: props.refs.canary, kind: "canary" as const },
    { name: copy.rsi.head, hash: props.refs.head, kind: "head" as const },
  ].filter((c) => c.hash);

  return (
    <LabFrame>
      <div data-testid="harness-overview">
        <div
          className={cn(
            "mb-6 flex flex-wrap items-center justify-between gap-3 rounded-[12px] border bg-card px-5 py-4 surface-inset",
            props.next.action === "blocked" ? "border-danger/30" : "border-border/80",
          )}
        >
          <div className="min-w-0">
            <LabEyebrow>{copy.rsi.next}</LabEyebrow>
            <p className="mt-1 max-w-[46ch] text-[15px] font-medium leading-snug tracking-[-0.02em] text-pretty text-foreground">{reason}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {props.onReveal ? (
              <Button variant="lift" onClick={() => void reveal(props.onReveal, props.refs.active)}>
                {copy.rsi.openArtifacts}
              </Button>
            ) : null}
            <Button onClick={() => props.onTab(props.next.tab)}>{copy.rsi.go} {goLabel}</Button>
          </div>
        </div>
        {pointers.length === 0 ? (
          <p className="mb-6 text-[13px] text-muted">{copy.rsi.noRefs}</p>
        ) : (
          <div className="mb-6 overflow-hidden rounded-[12px] border border-border/80 bg-card surface-inset">
            {pointers.map((c, i) => (
              <div key={c.name + c.hash} className={cn("grid grid-cols-[7rem_minmax(0,1fr)_auto] items-center gap-3 px-4 py-2.5", i > 0 && "border-t border-border/70")}>
                <div className="text-[13px] font-medium">{c.name}</div>
                <div className="min-w-0">
                  <HashLine hash={c.hash} compact />
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
            <Button size="sm" variant="ghost" onClick={() => void reveal(props.onReveal, "")}>{copy.rsi.openStore}</Button>
          ) : null}
        </div>
        {lineage.length === 0 ? (
          <p className="mb-6 text-[13px] text-muted">{copy.rsi.noLineage}</p>
        ) : (
          <div className="lineage-rail mb-6 overflow-hidden rounded-[12px] border border-border/80 bg-card pl-1" data-testid="harness-lineage">
            {lineage.map((n, i) => (
              <LineageRow key={n.hash} node={n} first={i === 0} onReveal={props.onReveal} />
            ))}
          </div>
        )}
        {metrics ? (
          <>
            <h3 className="mb-3 text-[13px] font-medium">{copy.rsi.lastEval}</h3>
            <div className="grid grid-cols-3 gap-3">
              <LabStat label={copy.harbor.heldIn} value={`${metrics.heldInPass}/${metrics.heldInTotal}`} />
              <LabStat label={copy.harbor.heldOut} value={`${metrics.heldOutPass}/${metrics.heldOutTotal}`} />
              <LabStat label={copy.harbor.safety} value={String(metrics.safetyFail)} bad={metrics.safetyFail > 0} />
            </div>
          </>
        ) : null}
      </div>
    </LabFrame>
  );
}

function LineageRow({ node, first, onReveal }: { node: LineageNode; first: boolean; onReveal?: (hash?: string) => Promise<void> | void }) {
  const copy = useCopy();
  const stamp = formatStamp(node.createdAt);
  return (
    <div className={cn("relative px-4 py-3 pl-8", !first && "border-t border-border/70")} data-testid="harness-lineage-node">
      <span className="absolute left-[10px] top-[18px] size-1.5 rounded-full bg-foreground/80" aria-hidden />
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <HashLine hash={node.hash} compact />
            {node.seed ? <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">{copy.rsi.seed}</span> : null}
            {node.refs.map((r) => (
              <span key={r} className="rounded-full bg-lift px-2 py-0.5 font-mono text-[11px] text-muted">{r}</span>
            ))}
            {node.fromArchive ? (
              <span className="text-[11px] text-muted">{node.accepted ? copy.labs.accepted : copy.labs.rejected}</span>
            ) : null}
          </div>
          <div className="mt-0.5 text-[12px] text-muted">
            {node.note || node.proposalId || (node.parent ? `${copy.rsi.vsParent} ${shortHash(node.parent, 12)}` : copy.rsi.seed)}
            {stamp ? ` · ${stamp}` : ""}
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

function HashLine({ hash, compact }: { hash: string; compact?: boolean }) {
  const copy = useCopy();
  return (
    <button
      type="button"
      className={cn("block max-w-full cursor-pointer truncate font-mono text-[11px] text-muted hover:text-foreground", !compact && "mt-1")}
      title={copy.rsi.copyHash}
      onClick={async () => {
        const ok = await writeClipboard(hash);
        toast.success(ok ? copy.rsi.copied : copy.transcript.copyFailed);
      }}
    >
      {shortHash(hash, 12)}
    </button>
  );
}

async function reveal(fn: ((hash?: string) => Promise<void> | void) | undefined, hash: string) {
  if (!fn) return;
  try {
    await fn(hash);
  } catch {
    /* caller toasts */
  }
}
