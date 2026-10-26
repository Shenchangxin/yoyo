// 本文件拥有空会话的问候、标题和起步按钮。不拥有输入框，输入框是旁边的兄弟节点，从首页进到会话时那块 DOM 还是同一个。App 在没有消息时使用它。

import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import type { Copy } from "../lib/copy";
import { PresenceAnchor } from "./presence";
import { CreateHome } from "./video/CreateHome";
import { FeaturedCases } from "./video/FeaturedCases";
import { useCutDesk, useCutReadiness } from "./video/cut-desk";
import { cn } from "../lib/utils";
import { useMedia } from "../lib/media";

// 12 点前用早上，18 点前用下午，否则用晚上。
function hello(copy: Copy): string {
  const h = new Date().getHours();
  if (h < 12) return copy.transcript.helloMorning;
  if (h < 18) return copy.transcript.helloAfternoon;
  return copy.transcript.helloEvening;
}

// 已加载且没有 ffmpeg 时剪辑按钮禁用；已加载且没有语音时配音禁用。提示里的 {id} 和 {hash} 换成当前工程。
function CreateBoundActions(props: {
  projectId: string;
  sourceHash: string;
  onPrompt?: (text: string) => void;
}) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const ready = useCutReadiness();
  const cutReason = ready.loaded && !ready.ffmpeg ? c.needFfmpeg : "";
  const dubReason = ready.loaded && !ready.speech ? c.needSpeech : "";
  return (
    <>
      <div className="mt-3 flex flex-wrap justify-center gap-2">
        <button
          type="button"
          className="starter-tile is-stack"
          disabled={!!cutReason}
          title={cutReason || c.clipThis}
          onClick={() => props.onPrompt?.(c.sendPrompt.replace("{id}", props.projectId).replace("{hash}", props.sourceHash))}
        >
          <span className="starter-tile-label">{c.clipThis}</span>
        </button>
        <button
          type="button"
          className="starter-tile is-stack"
          disabled={!!dubReason}
          title={dubReason || c.dub}
          onClick={() => props.onPrompt?.(c.dubPrompt.replace("{id}", props.projectId).replace("{hash}", props.sourceHash))}
        >
          <span className="starter-tile-label">{c.dub}</span>
          {dubReason ? <span className="starter-tile-hint">{dubReason}</span> : null}
        </button>
      </div>
    </>
  );
}

// 创作桌没有起步句。不在视频面用普通起步，画布和创作各用自己的，其余视频面用视频起步。
function startersOf(copy: Copy, videoing: boolean, videoMode: string, createDesk: boolean) {
  if (createDesk) return [];
  if (!videoing) return copy.transcript.starters;
  if (videoMode === "canvas") return copy.transcript.canvasStarters;
  if (videoMode === "creative") return copy.transcript.creativeStarters;
  return copy.transcript.videoStarters;
}

// 和起步句同一套分支，换成对应的就绪标题。创作桌用 conductorReady。
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

// HomeHero 是空会话的字标块。workspace 和 onBrowseWorkspace 当前不使用。创作桌还没绑定时不画锚点和问候。锚点尺寸在创作桌是 72，窄于 799px 是 96，否则 148。
export function HomeHero(_props: {
  workspace?: string;
  onBrowseWorkspace?: () => void;
}) {
  const { copy, videoing, videoMode, createDesk } = useHomeCopy();
  const narrow = useMedia("(max-width: 799px)");
  const c = copy.video.conductor;
  const bound = useCutDesk((s) => createDesk && !!s.project?.source_hash);
  const boundTitle = useCutDesk((s) => (createDesk ? s.project?.title || "" : ""));
  return (
    <div
      className={cn(
        "mx-auto flex w-full max-w-[min(100%,var(--thread-measure))] flex-col px-5 sm:px-8",
        createDesk ? "shrink-0 pb-1 pt-4" : "pt-2",
      )}
      data-testid="empty-turn"
    >
      {createDesk && !bound ? null : (
        <>
          <PresenceAnchor id="home" size={createDesk ? 72 : narrow ? 96 : 148} className="mx-auto cursor-pointer" />
          <p className={cn("text-center text-[13px] tracking-[-0.01em] text-muted", createDesk ? "mt-3" : "mt-5")}>
            {hello(copy)}
          </p>
        </>
      )}
      <h1 className={cn(
        "text-center font-semibold leading-[1.2] tracking-[-0.036em] text-pretty text-foreground",
        createDesk ? "mt-1 text-[19px]" : "mt-1.5 text-[21px]",
      )}>
        {bound ? c.copilotTitle : readyOf(copy, videoing, videoMode, createDesk)}
      </h1>
      {createDesk && bound ? (
        <>
          {boundTitle ? <p className="mt-1 text-center text-[13px] text-foreground">{boundTitle}</p> : null}
          <p className="mx-auto mt-1.5 max-w-[40ch] text-center text-[12px] leading-[1.5] text-muted">{c.finishHint}</p>
        </>
      ) : createDesk && !bound ? (
        <p className="mx-auto mt-1.5 max-w-[36ch] text-center text-[13px] leading-[1.5] text-muted">
          {c.lead}
        </p>
      ) : null}
    </div>
  );
}

// HomeStarters 在创作桌、未绑定且导入、翻拍、成片、交接、打开剪辑都传入时才挂 CreateHome。案例只在创作桌未绑定且有 onPrompt 时出现。按钮延迟从 80 毫秒起，每个再加 50。
export function HomeStarters(props: {
  onPrompt?: (text: string) => void;
  onContinueLast?: () => void;
  onImportClip?: () => void;
  onRemake?: () => void;
  onFilm?: (structure: "ranking" | "podcast" | "interview", description?: string) => void;
  onHandoff?: (dest: "studio" | "drama" | "canvas") => void;
  onOpenCut?: (id: string) => void;
  draftKey?: string;
}) {
  const { copy, videoing, videoMode, createDesk } = useHomeCopy();
  const project = useCutDesk((s) => (createDesk ? s.project : null));
  const starters = startersOf(copy, videoing, videoMode, createDesk);
  const bound = !!project?.source_hash && !!project.id;
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
        {bound && project?.id && project.source_hash ? (
          <CreateBoundActions projectId={project.id} sourceHash={project.source_hash} onPrompt={props.onPrompt} />
        ) : null}
        {starters.length ? (
        <div className={cn("starter-row flex flex-wrap justify-center", createDesk ? "mt-3 gap-2" : "mt-4 gap-1.5")}>
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
        ) : null}
      </div>
      {createDesk && !bound && props.onImportClip && props.onRemake && props.onFilm && props.onHandoff && props.onOpenCut ? (
        <CreateHome
          onImport={props.onImportClip}
          onRemake={props.onRemake}
          onFilm={props.onFilm}
          onHandoff={props.onHandoff}
          onOpenCut={props.onOpenCut}
          onPrompt={props.onPrompt}
          draftKey={props.draftKey}
        />
      ) : null}
      {createDesk && !bound && props.onPrompt ? <FeaturedCases onPrompt={props.onPrompt} /> : null}
    </>
  );
}
