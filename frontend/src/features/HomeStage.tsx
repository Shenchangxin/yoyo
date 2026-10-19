import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import type { Copy } from "../lib/copy";
import { PresenceAnchor } from "./presence";
import { CreationGallery } from "./video/CreationGallery";
import { cn } from "../lib/utils";

function hello(copy: Copy): string {
  const h = new Date().getHours();
  if (h < 12) return copy.transcript.helloMorning;
  if (h < 18) return copy.transcript.helloAfternoon;
  return copy.transcript.helloEvening;
}

function startersOf(copy: Copy, videoing: boolean, videoMode: string, createDesk: boolean) {
  if (createDesk) return copy.transcript.conductorStarters;
  if (!videoing) return copy.transcript.starters;
  if (videoMode === "canvas") return copy.transcript.canvasStarters;
  if (videoMode === "creative") return copy.transcript.creativeStarters;
  return copy.transcript.videoStarters;
}

function readyOf(copy: Copy, videoing: boolean, videoMode: string, createDesk: boolean) {
  if (createDesk) return copy.transcript.conductorReady;
  if (!videoing) return copy.transcript.ready;
  if (videoMode === "canvas") return copy.transcript.canvasReady;
  if (videoMode === "creative") return copy.transcript.creativeReady;
  return copy.transcript.videoReady;
}

function useHomeCopy() {
  const copy = useCopy();
  const surface = useUI((s) => s.surface);
  const videoMode = useUI((s) => s.videoMode);
  const videoPane = useUI((s) => s.videoPane);
  const videoing = surface === "video";
  const createDesk = videoing && videoPane === "create";
  return { copy, videoing, videoMode, createDesk };
}

/** Empty-session brand block. Composer stays a sibling so Hero→active keeps the same input DOM. */
export function HomeHero(_props: {
  workspace?: string;
  onBrowseWorkspace?: () => void;
}) {
  const { copy, videoing, videoMode, createDesk } = useHomeCopy();
  return (
    <div
      className={cn(
        "mx-auto flex w-full max-w-[min(100%,var(--thread-measure))] flex-col px-5 sm:px-8",
        createDesk ? "shrink-0 pb-1 pt-4" : "pt-2",
      )}
      data-testid="empty-turn"
    >
      <PresenceAnchor id="home" size={createDesk ? 88 : 148} className="mx-auto cursor-pointer" />
      <p className={cn("text-center text-[13px] tracking-[-0.01em] text-muted", createDesk ? "mt-3" : "mt-5")}>
        {hello(copy)}
      </p>
      <h1 className={cn(
        "text-center font-semibold leading-[1.2] tracking-[-0.036em] text-pretty text-foreground",
        createDesk ? "mt-1 text-[19px]" : "mt-1.5 text-[21px]",
      )}>
        {readyOf(copy, videoing, videoMode, createDesk)}
      </h1>
      {createDesk ? (
        <p className="mx-auto mt-1.5 max-w-[36ch] text-center text-[13px] leading-[1.5] text-muted">
          {copy.video.conductor.lead}
        </p>
      ) : null}
    </div>
  );
}

export function HomeStarters(props: {
  onPrompt?: (text: string) => void;
  onContinueLast?: () => void;
}) {
  const { copy, videoing, videoMode, createDesk } = useHomeCopy();
  const starters = startersOf(copy, videoing, videoMode, createDesk);
  return (
    <>
      <div className={cn(
        "mx-auto flex w-full max-w-[min(100%,var(--thread-measure))] flex-col px-5 sm:px-8",
        createDesk ? "pb-2" : "pb-8",
      )}>
        {props.onContinueLast ? (
          <div className="mt-3 flex justify-center">
            <button
              type="button"
              className="text-[12px] text-muted hover:text-foreground"
              onClick={() => props.onContinueLast?.()}
            >
              {copy.rail.continueLast}
            </button>
          </div>
        ) : null}
        <div className={cn("flex flex-wrap justify-center", createDesk ? "mt-3 gap-2" : "mt-4 gap-1.5")}>
          {starters.map((s, i) => (
            <button
              type="button"
              key={s.label}
              className={cn("starter-tile", createDesk && "is-stack")}
              style={{ animationDelay: `${80 + i * 50}ms` }}
              title={s.hint}
              aria-label={s.hint ? `${s.label}，${s.hint}` : s.label}
              onClick={() => props.onPrompt?.(s.text)}
            >
              <span className="starter-tile-label">{s.label}</span>
              {createDesk && s.hint ? <span className="starter-tile-hint">{s.hint}</span> : null}
            </button>
          ))}
        </div>
      </div>
      {createDesk ? <CreationGallery onStartPrompt={(prompt) => props.onPrompt?.(prompt)} /> : null}
    </>
  );
}
