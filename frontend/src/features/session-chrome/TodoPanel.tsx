/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized from deepseek-harness TodoPanel. Projects Yoyo update_plan.
 */
import { useState } from "react";
import { ChevronDown, ChevronUp, ListChecks } from "lucide-react";
import type { Copy } from "../../lib/copy";
import type { PlanStatus, TaskPlan } from "../../lib/plan";
import { planDoneCount } from "../../lib/plan";

function progressLabel(plan: TaskPlan, copy: Copy): string {
  const done = planDoneCount(plan);
  const active = plan.steps.filter((s) => s.status === "in_progress").length;
  const pending = plan.steps.length - done - active;
  return [
    done > 0 ? copy.composer.todoProgressDone.replace("{done}", String(done)) : "",
    active > 0 ? copy.composer.todoProgressActive.replace("{active}", String(active)) : "",
    pending > 0 ? copy.composer.todoProgressPending.replace("{pending}", String(pending)) : "",
  ].filter(Boolean).join("\u2002·\u2002");
}

function statusLabel(status: PlanStatus, copy: Copy): string {
  if (status === "complete") return copy.composer.todoStatusCompleted;
  if (status === "in_progress") return copy.composer.todoStatusInProgress;
  return copy.composer.todoStatusPending;
}

function StatusDot({ status }: { status: PlanStatus }) {
  if (status === "in_progress") {
    return (
      <svg className="sc-todo-spin" width="14" height="14" viewBox="0 0 24 24" aria-hidden>
        <g className="sc-todo-spin-motion">
          <circle className="sc-todo-spin-track" cx="12" cy="12" r="9.5" />
          <circle className="sc-todo-spin-arc" cx="12" cy="12" r="9.5" />
        </g>
      </svg>
    );
  }
  return <span className="sc-todo-dot" data-state={status === "complete" ? "done" : "idle"} aria-hidden />;
}

export function TodoPanel(props: { plan: TaskPlan | null | undefined; copy: Copy }) {
  const [collapsed, setCollapsed] = useState(true);
  const plan = props.plan;
  if (!plan?.steps.length) return null;
  return (
    <section className="sc-todo" data-testid="todo-panel" aria-label={props.copy.composer.todoTitle}>
      <div className="sc-todo-body">
        <button
          type="button"
          className="sc-todo-head"
          aria-expanded={!collapsed}
          onClick={() => setCollapsed((v) => !v)}
        >
          <span className="sc-todo-lead" aria-hidden>
            <ListChecks size={16} strokeWidth={1} />
          </span>
          <span className="sc-todo-title">{props.copy.composer.todoTitle}</span>
          <span className="sc-todo-progress">{progressLabel(plan, props.copy)}</span>
          <span className="sc-todo-chevron" aria-hidden>
            {collapsed ? <ChevronUp size={16} strokeWidth={1} /> : <ChevronDown size={16} strokeWidth={1} />}
          </span>
        </button>
        {collapsed ? null : (
          <ul className="sc-todo-list">
            {plan.steps.map((step, i) => (
              <li key={`${step.step}-${i}`} className="sc-todo-row" data-status={step.status}>
                <span className="sc-todo-glyph" role="img" aria-label={statusLabel(step.status, props.copy)}>
                  <StatusDot status={step.status} />
                </span>
                <span className="sc-todo-content">{step.step}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
