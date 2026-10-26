/**
 * 本文件拥有新建短片的标题框、短片 id 的复制，以及选题分数表。
 * 不拥有选题和工作台。复制只写剪贴板，分数交给回调，本文件不改服务端。
 */
import { useState, type FormEvent } from "react";
import { ChevronDown, Copy } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { writeClipboard } from "../../lib/clipboard";
import { cn } from "../../lib/utils";
import { SCORE_KEYS, setScore, topicScoreAvg, type ScoreKey } from "./studio-topic";

/** CreateShortField 提交非空标题，成功后清空输入。props 的 disabled 拦住提交，hint、placeholder、autoFocus、testId、titleTestId 和 submitLabel 画表单，onSubmit 收到去掉空白的标题。 */
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

  /** 去掉空白后提交标题。e 有传入时阻止表单默认提交。 */
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
          onChange={/** e 取出 target.value 写进标题。 */ (e) => setTitle(e.target.value)}
        />
        <Button size="sm" variant="default" className="h-8 shrink-0" disabled={!ready} data-testid={props.testId} type="submit">
          {props.submitLabel}
        </Button>
      </div>
    </form>
  );
}

/** StudioIdChip 显示短片 id，有会话时再显示一颗可复制的会话 id。props 没有 projectId 就不显示；projectId 和 sessionId 可复制，projectLabel、sessionLabel、copyLabel、copiedLabel 和 inspectHint 用于文案和提示。 */
export function StudioIdChip(props: {
  projectId: string;
  sessionId?: string;
  projectLabel: string;
  sessionLabel: string;
  copyLabel: string;
  copiedLabel: string;
  inspectHint: string;
}) {
  /** 写入剪贴板，成功就提示已复制。value 交给 writeClipboard。 */
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

/** TopicScoreSheet 展开后用 1 到 5 改七项分数。props 的 scores 用来算平均和改分，labels、avgLabel、emptyLabel 和 hint 显示文案，disabled 传给分数按钮，onChange 收到改完的分数。 */
export function TopicScoreSheet(props: {
  scores?: Record<string, number>;
  labels: Record<ScoreKey, string>;
  avgLabel: string;
  emptyLabel: string;
  hint: string;
  disabled?: boolean;
  onChange: (scores: Record<string, number>) => void;
}) {
  const [open, setOpen] = useState(false);
  const avg = topicScoreAvg(props.scores);
  return (
    <div className="mt-1.5">
      <button
        type="button"
        className="flex w-full items-center gap-2 rounded-md py-0.5 text-left text-[12px] text-muted hover:text-foreground"
        aria-expanded={open}
        data-testid="studio-topic-scores"
        onClick={() => setOpen(/** v 取反。 */ (v) => !v)}
      >
        <span>{props.avgLabel}</span>
        {avg ? (
          <span className="tabular-nums text-foreground">{avg}</span>
        ) : (
          <span>{props.emptyLabel}</span>
        )}
        <ChevronDown className={cn("ml-auto size-3.5 shrink-0 transition-transform", open && "rotate-180")} aria-hidden />
      </button>
      {open ? (
        <div className="mt-1.5 space-y-1">
          <p className="text-[11px] leading-[1.45] text-muted">{props.hint}</p>
          {SCORE_KEYS.map(/** key 取当前分和标签。 */ (key) => {
            const n = props.scores?.[key] || 0;
            return (
              <div key={key} className="flex items-center gap-2" data-testid={`studio-topic-score-${key}`}>
                <span className="w-[4.5rem] shrink-0 text-[12px] text-muted">{props.labels[key]}</span>
                <div className="flex gap-0.5" role="radiogroup" aria-label={props.labels[key]}>
                  {[1, 2, 3, 4, 5].map(/** tick 和当前分比较，点了就连同 key 交给 onChange。 */ (tick) => {
                    const on = n === tick;
                    return (
                      <button
                        key={tick}
                        type="button"
                        role="radio"
                        aria-checked={on}
                        disabled={props.disabled}
                        data-testid={`studio-topic-score-${key}-${tick}`}
                        className={cn(
                          "grid size-6 place-items-center rounded-md text-[11px] tabular-nums",
                          on ? "bg-foreground text-background" : "text-muted hover:bg-lift/70 hover:text-foreground",
                        )}
                        onClick={() => props.onChange(setScore(props.scores, key, tick))}
                      >
                        {tick}
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
