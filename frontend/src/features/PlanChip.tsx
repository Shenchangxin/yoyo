import { useId, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Check, ChevronRight } from "lucide-react";
import { ProgressHairline } from "../components/ui/progress-hairline";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { DURATION, motionTransition, useMotionReduced } from "../lib/motion";
import { planDoneCount, planFocus, type PlanStatus, type PlanStep, type TaskPlan } from "../lib/plan";

export function PlanChip({
  plan,
  running,
  compact,
}: {
  plan: TaskPlan;
  running: boolean;
  compact?: boolean;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const stepsId = useId();
  const [open, setOpen] = useState(false);
  const done = planDoneCount(plan);
  const total = plan.steps.length;
  const focus = planFocus(plan);
  const progress = copy.composer.taskPlanProgress.replace("{done}", String(done)).replace("{total}", String(total));
  const toggleLabel = open ? copy.composer.hidePlan : copy.composer.showPlan;

  return (
    <div
      className="min-w-0 border-b border-border/50"
      data-testid="task-plan"
      role="region"
      aria-label={copy.transcript.plan}
    >
      <div className="relative">
        <button
          type="button"
          className={cn(
            "group/plan flex w-full min-w-0 items-center gap-2 text-left select-none transition-[background-color,color] duration-150",
            "rounded-t-[calc(var(--radius-composer)-1px)] hover:bg-lift/70 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring",
            compact ? "h-7 px-3" : "h-8 px-3.5",
          )}
          aria-expanded={open}
          aria-controls={stepsId}
          aria-label={toggleLabel}
          title={toggleLabel}
          onClick={() => setOpen((v) => !v)}
        >
          <ChevronRight
            className={cn(
              "size-2.5 shrink-0 text-muted/55 transition-[transform,color] duration-150 group-hover/plan:text-muted",
              open && "rotate-90",
            )}
            aria-hidden
          />
          <span className="shrink-0 text-[13px] font-medium tracking-[-0.01em] text-foreground/90">{copy.transcript.plan}</span>
          {!open && focus ? (
            <span className="min-w-0 flex-1 truncate text-[12.5px] text-muted" title={focus.step}>
              {focus.step}
            </span>
          ) : (
            <span className="min-w-0 flex-1" />
          )}
          <span className="shrink-0 tabular-nums text-[11px] text-muted/80">{progress}</span>
        </button>
        {running || (done > 0 && done < total) ? (
          <ProgressHairline
            className="absolute inset-x-0 bottom-0 rounded-none"
            value={total ? (100 * done) / total : 0}
            indeterminate={running && done === 0}
          />
        ) : null}
      </div>
      <AnimatePresence initial={false}>
        {open ? (
          <motion.div
            id={stepsId}
            key="steps"
            initial={reduced ? false : { height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={reduced ? undefined : { height: 0, opacity: 0 }}
            transition={motionTransition(reduced, DURATION)}
            className="overflow-hidden"
          >
            {plan.explanation && !compact ? (
              <p className="px-3.5 pb-1 pl-8 text-[11.5px] leading-[1.4] text-muted">{plan.explanation}</p>
            ) : null}
            <ol className={cn("min-w-0", compact ? "max-h-28 overflow-y-auto pb-1.5" : "max-h-40 overflow-y-auto pb-2")}>
              {plan.steps.map((step, i) => (
                <PlanStepRow key={`${i}:${step.step}`} step={step} running={running} compact={compact} />
              ))}
            </ol>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

function PlanStepRow({
  step,
  running,
  compact,
}: {
  step: PlanStep;
  running: boolean;
  compact?: boolean;
}) {
  const copy = useCopy();
  const label = statusLabel(step.status, copy.composer);
  const live = step.status === "in_progress" && running;
  return (
    <li
      className={cn(
        "flex min-w-0 items-start gap-2.5 pr-3.5",
        compact ? "py-1 pl-7" : "py-[5px] pl-8",
      )}
      aria-label={`${label}: ${step.step}`}
      data-status={step.status}
    >
      <PlanGlyph status={step.status} live={live} />
      <span
        className={cn(
          "min-w-0 flex-1 text-[13px] leading-[1.35] tracking-[-0.011em]",
          compact && "line-clamp-1",
          !compact && "line-clamp-2",
          step.status === "in_progress" && "font-medium text-foreground",
          step.status === "complete" && "text-muted",
          step.status === "pending" && "text-foreground/85",
        )}
      >
        {step.step}
      </span>
    </li>
  );
}

function PlanGlyph({ status, live }: { status: PlanStatus; live: boolean }) {
  return (
    <span className="mt-[2px] grid size-3.5 shrink-0 place-items-center" aria-hidden>
      {status === "complete" ? (
        <span className="grid size-3.5 place-items-center rounded-full bg-success text-background">
          <Check className="size-2.5" strokeWidth={3} />
        </span>
      ) : status === "in_progress" ? (
        <span className="grid size-3.5 place-items-center rounded-full border-[1.5px] border-accent">
          {live ? <span className="pulse-dot" /> : <span className="size-1.5 rounded-full bg-accent" />}
        </span>
      ) : (
        <span className="size-3.5 rounded-full border-[1.5px] border-muted/45" />
      )}
    </span>
  );
}

function statusLabel(status: PlanStatus, c: { planPending: string; planActive: string; planDone: string }): string {
  if (status === "complete") return c.planDone;
  if (status === "in_progress") return c.planActive;
  return c.planPending;
}
