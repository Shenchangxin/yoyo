import { useState, type FormEvent } from "react";
import { Copy } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { writeClipboard } from "../../lib/clipboard";
import { cn } from "../../lib/utils";
import { SCORE_KEYS, cycleScore, topicScoreAvg, type ScoreKey } from "./studio-topic";

export function CreateShortField(props: {
  disabled?: boolean;
  autoFocus?: boolean;
  placeholder: string;
  submitLabel: string;
  hint?: string;
  testId: string;
  titleTestId?: string;
  onSubmit: (title: string) => void | Promise<void>;
}) {
  const [title, setTitle] = useState("");
  const ready = !!title.trim() && !props.disabled;

  function go(e?: FormEvent) {
    e?.preventDefault();
    const next = title.trim();
    if (!next || props.disabled) return;
    void Promise.resolve(props.onSubmit(next)).then(() => setTitle(""));
  }

  return (
    <form className="flex flex-col gap-1.5" onSubmit={go} data-testid={`${props.testId}-form`}>
      {props.hint ? <p className="text-[12px] leading-[1.45] text-muted">{props.hint}</p> : null}
      <div className="flex gap-2">
        <Input
          className="h-8 min-w-0 flex-1"
          value={title}
          autoFocus={props.autoFocus}
          placeholder={props.placeholder}
          data-testid={props.titleTestId || `${props.testId}-title`}
          onChange={(e) => setTitle(e.target.value)}
        />
        <Button size="sm" variant="default" className="h-8 shrink-0" disabled={!ready} data-testid={props.testId} type="submit">
          {props.submitLabel}
        </Button>
      </div>
    </form>
  );
}

export function StudioIdChip(props: {
  projectId: string;
  sessionId?: string;
  projectLabel: string;
  sessionLabel: string;
  copyLabel: string;
  copiedLabel: string;
  inspectHint: string;
}) {
  async function copy(value: string) {
    const ok = await writeClipboard(value);
    if (ok) toast.success(props.copiedLabel);
  }
  if (!props.projectId) return null;
  return (
    <div className="flex min-w-0 flex-col gap-0.5" data-testid="studio-id-chip">
      <button
        type="button"
        className="flex min-w-0 max-w-full items-center gap-1.5 rounded-md px-0.5 py-0.5 text-left hover:bg-lift/70"
        title={props.inspectHint}
        aria-label={props.copyLabel}
        data-testid="studio-copy-id"
        onClick={() => void copy(props.projectId)}
      >
        <span className="shrink-0 text-[11px] text-muted">{props.projectLabel}</span>
        <code className="min-w-0 flex-1 truncate font-mono text-[11px] text-foreground">{props.projectId}</code>
        <Copy className="size-3 shrink-0 opacity-60" aria-hidden />
      </button>
      {props.sessionId ? (
        <button
          type="button"
          className="flex min-w-0 max-w-full items-center gap-1.5 rounded-md px-0.5 py-0.5 text-left hover:bg-lift/70"
          title={props.inspectHint}
          aria-label={props.sessionLabel}
          data-testid="studio-copy-session"
          onClick={() => void copy(props.sessionId || "")}
        >
          <span className="shrink-0 text-[11px] text-muted">{props.sessionLabel}</span>
          <code className="min-w-0 flex-1 truncate font-mono text-[11px] text-foreground">{props.sessionId}</code>
          <Copy className="size-3 shrink-0 opacity-60" aria-hidden />
        </button>
      ) : null}
    </div>
  );
}

export function ScorePills(props: {
  scores?: Record<string, number>;
  labels: Record<ScoreKey, string>;
  avgLabel: string;
  disabled?: boolean;
  onChange: (scores: Record<string, number>) => void;
}) {
  const avg = topicScoreAvg(props.scores);
  return (
    <details className="mt-2">
      <summary className="cursor-pointer text-[12px] text-muted" data-testid="studio-topic-scores">
        {props.avgLabel}
        {avg ? ` ${avg}` : ""}
      </summary>
      <div className="mt-2 flex flex-wrap gap-1">
        {SCORE_KEYS.map((key) => {
          const n = props.scores?.[key] || 0;
          return (
            <button
              key={key}
              type="button"
              disabled={props.disabled}
              className={cn(
                "rounded-full px-2 py-0.5 text-[11px]",
                n ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70",
              )}
              onClick={() => {
                const next = { ...(props.scores || {}), [key]: cycleScore(n) };
                if (!next[key]) delete next[key];
                props.onChange(next);
              }}
            >
              {props.labels[key]}{n ? ` ${n}` : ""}
            </button>
          );
        })}
      </div>
    </details>
  );
}
