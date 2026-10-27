// @ts-nocheck
import { AIMessageMarkdown } from "@yingce/components/ai/ai-message-markdown";
import { ThinkingStream } from "@/features/session-chrome/ThoughtLine";
import { useCopy } from "@/lib/i18n";

export function MessageReasoning({ reasoning, isStreaming }: { reasoning: string; isStreaming: boolean }) {
    const copy = useCopy();
    const done = copy.transcript.thoughtFor.split("{n}")[0]?.trim() || copy.transcript.thinking;
    return (
        <div className="mb-2">
            <ThinkingStream
                working={isStreaming}
                label={copy.transcript.live.think}
                doneLabel={done}
                idleLabel={copy.transcript.thinking}
                fontSize={12}
            >
                {reasoning ? (
                    <div className="rounded-md bg-surface-active p-3 text-foreground/70">
                        <AIMessageMarkdown isStreaming={isStreaming}>{reasoning}</AIMessageMarkdown>
                    </div>
                ) : null}
            </ThinkingStream>
        </div>
    );
}
