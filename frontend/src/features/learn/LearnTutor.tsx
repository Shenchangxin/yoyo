/**
 * 拥有：学习表面右侧的导师栏。
 * 不拥有：对话正文和发送。调用方把紧凑转录和输入放进 children。
 */

import type { ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";

export type TutorPrompt = { id: string; label: string; text: string };

/** LearnTutor 标出导师正在看的内容。收起时只留一条边。quiet 时才给出建议句。 */
export function LearnTutor(props: {
  open: boolean;
  subject: string;
  about: string;
  prompts?: TutorPrompt[];
  quiet?: boolean;
  onOpen: (open: boolean) => void;
  onPrompt?: (text: string) => void;
  children?: ReactNode;
}) {
  const L = useCopy().learn;
  if (!props.open) {
    return (
      <div className="flex h-full w-11 shrink-0 flex-col border-l border-border/60" data-testid="learn-tutor">
        <button
          type="button"
          className="flex h-full flex-col items-center gap-2 px-1 pt-3 text-[12px] text-muted hover:text-foreground"
          aria-expanded={false}
          aria-label={L.tutorShow}
          onClick={() => props.onOpen(true)}
        >
          <span className="[writing-mode:vertical-rl]">{L.tutor}</span>
        </button>
      </div>
    );
  }
  const prompts = props.quiet ? props.prompts || [] : [];
  return (
    <aside
      className="flex h-full min-h-0 w-[min(340px,36%)] shrink-0 flex-col border-l border-border/60"
      data-testid="learn-tutor"
      aria-label={L.tutor}
    >
      <div className="flex h-12 shrink-0 items-center gap-2 px-3">
        <div className="min-w-0 flex-1">
          <p className="text-[11px] text-muted">{L.tutor}</p>
          <p className="truncate text-[13px] font-medium tracking-[-0.02em]">{props.subject}</p>
        </div>
        <button
          type="button"
          className="grid size-7 shrink-0 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground"
          aria-expanded
          aria-label={L.tutorHide}
          onClick={() => props.onOpen(false)}
        >
          <ChevronRight className="size-4" aria-hidden />
        </button>
      </div>
      <p className="px-3 pb-2 text-[12px] leading-relaxed text-muted">{props.about}</p>
      {prompts.length > 0 ? (
        <div className="flex flex-col px-2 pb-2">
          {prompts.map((prompt) => (
            <button
              key={prompt.id}
              type="button"
              className={cn("truncate whitespace-nowrap rounded-md px-2 py-1.5 text-left text-[12px] text-muted hover:bg-lift hover:text-foreground")}
              onClick={() => props.onPrompt?.(prompt.text)}
            >
              {prompt.label}
            </button>
          ))}
        </div>
      ) : null}
      <div className="flex min-h-0 flex-1 flex-col">{props.children}</div>
    </aside>
  );
}
