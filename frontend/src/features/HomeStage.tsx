// 本文件拥有空会话的问候、标题和起步句。不拥有输入框，输入框是旁边的兄弟节点，从首页进到会话时那块 DOM 还是同一个。App 在没有消息时使用它。
// 创作桌只有机器人、一句话、一行提示和三句例子；做什么由导演自己判断，不靠按钮。

import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import type { Copy } from "../lib/copy";
import { PresenceAnchor } from "./presence";
import { CreateHome } from "./video/CreateHome";
import { FeaturedCases } from "./video/FeaturedCases";
import { useCreateMode } from "./video/create-mode";
import { isFilmMode, type FilmMode } from "./video/film-cast";
import { useCutDesk } from "./video/cut-desk";
import { useFilmDesk } from "./video/film-desk";
import { useRemakeDesk } from "./video/remake-desk";
import { cn } from "../lib/utils";
import { useMedia } from "../lib/media";

/** hello 12 点前用早上，18 点前用下午，否则用晚上。copy 读取 transcript。 */
function hello(copy: Copy): string {
  const h = new Date().getHours();
  if (h < 12) return copy.transcript.helloMorning;
  if (h < 18) return copy.transcript.helloAfternoon;
  return copy.transcript.helloEvening;
}

/** startersOf 创作桌的例子在 HomeStarters 里另取。不在视频面用普通起步，画布和创作各用自己的，其余视频面用视频起步。copy 读取 transcript。videoing 为假时走另一支。videoMode 与 canvas、creative 比较是否相等。createDesk 为真时返回空。 */
function startersOf(copy: Copy, videoing: boolean, videoMode: string, createDesk: boolean) {
  if (createDesk) return [];
  if (!videoing) return copy.transcript.starters;
  if (videoMode === "canvas") return copy.transcript.canvasStarters;
  if (videoMode === "creative") return copy.transcript.creativeStarters;
  return copy.transcript.videoStarters;
}

/** readyOf 和起步句同一套分支，换成对应的就绪标题。创作桌用 conductorReady。copy 读取 transcript。videoing 为假时走另一支。videoMode 与 canvas、creative 比较是否相等。createDesk 为真时才继续。 */
function readyOf(copy: Copy, videoing: boolean, videoMode: string, createDesk: boolean) {
  if (createDesk) return copy.transcript.conductorReady;
  if (!videoing) return copy.transcript.ready;
  if (videoMode === "canvas") return copy.transcript.canvasReady;
  if (videoMode === "creative") return copy.transcript.creativeReady;
  return copy.transcript.videoReady;
}

// 从界面状态取出是否在视频面、视频模式，以及是不是创作桌。
function useHomeCopy() {
  const copy = useCopy();
  const surface = useUI((s) => s.surface);
  const videoMode = useUI((s) => s.videoMode);
  const videoPane = useUI((s) => s.videoPane);
  const videoing = surface === "video";
  const createDesk = videoing && videoPane === "create";
  return { copy, videoing, videoMode, createDesk };
}

/** HomeHero 是空会话的字标块。创作桌画机器人、一句标题和一行提示，不画问候。锚点尺寸在创作桌是 72，窄于 799px 是 96，否则 148。_props 函数体内没有读取。 */
export function HomeHero(_props: {
  workspace?: string;
  onBrowseWorkspace?: () => void;
}) {
  const { copy, videoing, videoMode, createDesk } = useHomeCopy();
  const narrow = useMedia("(max-width: 799px)");
  const c = copy.video.conductor;
  const film = copy.video.film;
  const mode = useCreateMode((s) => s.mode);
  const filmOpen = useFilmDesk((s) => createDesk && !!s.project?.id);
  const filmHints: Record<FilmMode, string> = {
    talking: film.talkingHint,
    drama: film.dramaHint,
    podcast: film.podcastHint,
    interview: film.interviewHint,
    ranking: film.rankingHint,
    explainer: film.explainerHint,
    narration: film.narrationHint,
  };
  const filmLead = !createDesk || !isFilmMode(mode) ? "" : filmHints[mode];
  const bound = useCutDesk((s) => createDesk && !!s.project?.source_hash);
  const boundTitle = useCutDesk((s) => (createDesk ? s.project?.title || "" : ""));
  if (createDesk && !bound && filmLead && !filmOpen) return null;
  return (
    <div
      className={cn(
        "mx-auto flex w-full max-w-[min(100%,var(--thread-measure))] flex-col px-5 sm:px-8",
        createDesk ? "shrink-0 pb-1 pt-4" : "pt-2",
      )}
      data-testid="empty-turn"
    >
      <PresenceAnchor id="home" size={createDesk ? 72 : narrow ? 96 : 148} className="mx-auto cursor-pointer" />
      {createDesk ? null : (
        <p className="mt-5 text-center text-ui-body tracking-[-0.01em] text-muted">{hello(copy)}</p>
      )}
      {createDesk && filmOpen ? null : (
        <h1 className={cn(
          "text-center font-semibold leading-[1.2] tracking-[-0.036em] text-pretty text-foreground text-ui-display",
          createDesk ? "mt-3" : "mt-1.5",
        )}>
          {bound ? c.copilotTitle : readyOf(copy, videoing, videoMode, createDesk)}
        </h1>
      )}
      {createDesk && bound ? (
        <>
          {boundTitle ? <p className="mt-1 text-center text-ui-body text-foreground">{boundTitle}</p> : null}
          <p className="mx-auto mt-1.5 max-w-[40ch] text-center text-ui-footnote leading-[1.5] text-muted">{c.finishHint}</p>
        </>
      ) : createDesk && !filmOpen ? (
        <p className="mx-auto mt-1.5 max-w-[40ch] text-center text-ui-body leading-[1.5] text-muted">{c.lead}</p>
      ) : null}
    </div>
  );
}

/**
 * HomeStarters 画起步句。创作桌画三句例子：没绑定片子时是想法，绑定后是怎么剪；点一句只放进输入框，不发送。没绑定时下面再画案例目录。成片或翻拍稿打开时不画例子和案例，交给 CreateHome。按钮延迟从 80 毫秒起，每个再加 50。
 * props 读取 onContinueLast、onPrompt、onOpenCut。创作桌不画「继续上次」。
 */
export function HomeStarters(props: {
  onPrompt?: (text: string) => void;
  onContinueLast?: () => void;
  onOpenCut?: (id: string) => void;
  draftKey?: string;
}) {
  const { copy, videoing, videoMode, createDesk } = useHomeCopy();
  const c = copy.video.conductor;
  const project = useCutDesk((s) => (createDesk ? s.project : null));
  const resumeId = useCutDesk((s) => (createDesk && s.parked ? s.resumeId : ""));
  const resumeTitle = useCutDesk((s) => (createDesk && s.parked ? s.resumeTitle : ""));
  const filmOpen = useFilmDesk((s) => !!s.project?.id);
  const remakeOpen = useRemakeDesk((s) => !!s.draft?.id);
  const bound = !!project?.source_hash && !!project.id;
  const rows: ReadonlyArray<{ label: string; text: string; hint?: string }> = createDesk
    ? (filmOpen || remakeOpen ? [] : bound ? c.cutExamples : c.examples)
    : startersOf(copy, videoing, videoMode, createDesk);
  return (
    <>
      <div className={cn(
        "mx-auto flex w-full max-w-[min(100%,var(--thread-measure))] flex-col px-5 sm:px-8",
        createDesk ? "pb-2" : "pb-8",
      )}>
        {/* 创作桌不画「继续上次」：历史在左栏的作品列表里。 */}
        {props.onContinueLast && !createDesk ? (
          <div className="mt-3 flex justify-center">
            <button
              type="button"
              className="text-ui-footnote text-muted hover:text-foreground"
              onClick={() => props.onContinueLast?.()}
            >
              {copy.rail.continueLast}
            </button>
          </div>
        ) : null}
        {rows.length ? (
          <div className={cn("starter-row flex flex-wrap justify-center", createDesk ? "mt-3 gap-2" : "mt-4 gap-1.5")}>
            {rows.map((s, i) => (
              <button
                type="button"
                key={s.label}
                className="starter-tile"
                data-testid={createDesk ? `create-example-${i}` : undefined}
                style={{ animationDelay: `${80 + i * 50}ms` }}
                title={s.hint || s.text}
                aria-label={s.hint ? `${s.label}，${s.hint}` : s.label}
                onClick={() => props.onPrompt?.(s.text)}
              >
                <span className="starter-tile-label">{s.label}</span>
              </button>
            ))}
          </div>
        ) : null}
        {createDesk && !bound && !filmOpen && !remakeOpen && resumeId && props.onOpenCut ? (
          <div className="mt-3 flex justify-center">
            <button
              type="button"
              data-testid="continue-cut"
              className="text-ui-footnote text-muted hover:text-foreground"
              title={c.handoffContinueHint}
              onClick={() => props.onOpenCut?.(resumeId)}
            >
              {resumeTitle ? `${c.handoffContinue} · ${resumeTitle}` : c.handoffContinue}
            </button>
          </div>
        ) : null}
      </div>
      {createDesk && !bound && props.onOpenCut ? <CreateHome onOpenCut={props.onOpenCut} /> : null}
      {/* 案例目录留在空的创作桌上，点一张只把做法填进输入框，不生成。 */}
      {createDesk && !bound && !filmOpen && !remakeOpen && props.onPrompt ? <FeaturedCases onPrompt={props.onPrompt} /> : null}
    </>
  );
}