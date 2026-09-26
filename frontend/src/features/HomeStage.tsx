import type { ReactNode } from "react";
import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import type { Copy } from "../lib/copy";
import { PresenceAnchor } from "./presence";

function hello(copy: Copy): string {
  const h = new Date().getHours();
  if (h < 12) return copy.transcript.helloMorning;
  if (h < 18) return copy.transcript.helloAfternoon;
  return copy.transcript.helloEvening;
}

function startersOf(copy: Copy, videoing: boolean, videoMode: string) {
  if (!videoing) return copy.transcript.starters;
  if (videoMode === "canvas") return copy.transcript.canvasStarters;
  if (videoMode === "creative") return copy.transcript.creativeStarters;
  return copy.transcript.videoStarters;
}

function readyOf(copy: Copy, videoing: boolean, videoMode: string) {
  if (!videoing) return copy.transcript.ready;
  if (videoMode === "canvas") return copy.transcript.canvasReady;
  if (videoMode === "creative") return copy.transcript.creativeReady;
  return copy.transcript.videoReady;
}

export function HomeStage(props: {
  onPrompt?: (text: string) => void;
  children: ReactNode;
}) {
  const copy = useCopy();
  const surface = useUI((s) => s.surface);
  const videoMode = useUI((s) => s.videoMode);
  const videoing = surface === "video";
  const starters = startersOf(copy, videoing, videoMode);

  return (
    <section className="@container relative flex h-full min-h-0 min-w-0 flex-col" data-testid="conversation-column">
      <div className="home-stage empty-rise flex min-h-0 flex-1 flex-col overflow-auto" data-testid="empty-turn">
        <div className="mx-auto flex w-full max-w-[min(100%,var(--thread-measure))] flex-1 flex-col justify-center px-5 py-10 sm:px-8">
          <PresenceAnchor id="home" size={140} className="mx-auto cursor-pointer" />
          <p className="mt-5 text-center text-[13px] tracking-[-0.01em] text-muted">
            {hello(copy)}
          </p>
          <h1 className="mt-1.5 text-center text-[21px] font-semibold leading-[1.2] tracking-[-0.036em] text-pretty text-foreground">
            {readyOf(copy, videoing, videoMode)}
          </h1>
          <div className="mt-8 w-full">
            {props.children}
          </div>
          <div className="mt-4 flex flex-wrap justify-center gap-1.5">
            {starters.map((s, i) => (
              <button
                type="button"
                key={s.label}
                className="starter-tile"
                style={{ animationDelay: `${80 + i * 50}ms` }}
                onClick={() => props.onPrompt?.(s.text)}
              >
                <span className="text-[13px] font-medium tracking-[-0.02em] text-foreground">{s.label}</span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
