import { useEffect, useState } from "react";
import { Clapperboard, LayoutGrid, PenLine } from "lucide-react";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { EmptyState } from "../../components/ui/empty-state";
import { DramaStudio } from "./DramaStudio";

type Mode = "drama" | "canvas" | "creative";

function readMode(): Mode {
  try {
    const v = localStorage.getItem("yoyo-video-mode");
    if (v === "drama" || v === "canvas" || v === "creative") return v;
  } catch {
    /* ignore */
  }
  return "drama";
}

export function VideoHub(props: { sessionId?: string; onNeedSession: () => void }) {
  const copy = useCopy();
  const [mode, setMode] = useState<Mode>(readMode);
  useEffect(() => {
    try {
      localStorage.setItem("yoyo-video-mode", mode);
    } catch {
      /* ignore */
    }
  }, [mode]);

  const modes: { id: Mode; label: string; hint: string; icon: typeof Clapperboard; ready: boolean }[] = [
    { id: "drama", label: copy.video.drama, hint: copy.video.dramaHint, icon: Clapperboard, ready: true },
    { id: "canvas", label: copy.video.canvas, hint: copy.video.canvasHint, icon: LayoutGrid, ready: false },
    { id: "creative", label: copy.video.creative, hint: copy.video.creativeHint, icon: PenLine, ready: false },
  ];

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="video-hub">
      <div className="flex h-10 shrink-0 items-center gap-3 border-b border-border/70 px-3">
        <div className="process-tabs flex h-10 items-stretch gap-0.5" role="tablist" aria-label={copy.video.modes}>
          {modes.map((m) => {
            const on = mode === m.id;
            return (
              <button
                key={m.id}
                type="button"
                role="tab"
                title={m.hint}
                aria-selected={on}
                className={cn(
                  "relative flex h-full cursor-pointer items-center gap-1.5 px-2.5 text-[12.5px] font-medium transition-colors",
                  on ? "text-foreground" : "text-muted hover:text-foreground",
                )}
                onClick={() => setMode(m.id)}
              >
                <m.icon className="size-3.5" aria-hidden />
                {m.label}
                {!m.ready ? <span className="text-[10.5px] font-normal text-muted/70">{copy.video.soon}</span> : null}
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
      </div>
      <div className="min-h-0 flex-1">
        {mode === "drama" ? (
          <DramaStudio sessionId={props.sessionId} onNeedSession={props.onNeedSession} />
        ) : (
          <EmptyState
            icon={mode === "canvas" ? <LayoutGrid className="size-5" /> : <PenLine className="size-5" />}
            title={mode === "canvas" ? copy.video.canvas : copy.video.creative}
            body={copy.video.later}
          />
        )}
      </div>
    </div>
  );
}
