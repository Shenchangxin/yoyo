/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized from ConversationHeader view tabs (Chat / Trajectory).
 */
import { ChevronRight } from "lucide-react";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";

export type SessionView = "chat" | "trajectory";

export function SessionHeader(props: {
  view: SessionView;
  onView: (view: SessionView) => void;
  crumb?: string;
  trail?: string;
}) {
  const copy = useCopy();
  return (
    <div className="flex h-9 shrink-0 items-center gap-3 border-b border-border/40 px-4" data-testid="session-header">
      {props.crumb || props.trail ? (
        <span className="flex min-w-0 items-center gap-1 text-[12px] text-muted">
          {props.crumb ? (
            <span className="min-w-0 truncate" title={props.crumb}>{props.crumb}</span>
          ) : null}
          {props.crumb && props.trail ? <ChevronRight className="size-3 shrink-0 opacity-50" aria-hidden /> : null}
          {props.trail ? (
            <span className="min-w-0 truncate text-foreground" title={props.trail} data-testid="session-subagent-crumb">{props.trail}</span>
          ) : null}
        </span>
      ) : null}
      <div className="ml-auto flex items-center gap-0.5 rounded-lg bg-lift/70 p-0.5">
        <button
          type="button"
          className={cn("h-6 rounded-md px-2 text-[11px] font-medium", props.view === "chat" ? "bg-background text-foreground shadow-sm" : "text-muted hover:text-foreground")}
          aria-pressed={props.view === "chat"}
          data-testid="session-view-chat"
          onClick={() => props.onView("chat")}
        >
          {copy.composer.viewChat}
        </button>
        <button
          type="button"
          className={cn("h-6 rounded-md px-2 text-[11px] font-medium", props.view === "trajectory" ? "bg-background text-foreground shadow-sm" : "text-muted hover:text-foreground")}
          aria-pressed={props.view === "trajectory"}
          data-testid="session-view-trajectory"
          onClick={() => props.onView("trajectory")}
        >
          {copy.composer.viewTrajectory}
        </button>
      </div>
    </div>
  );
}
