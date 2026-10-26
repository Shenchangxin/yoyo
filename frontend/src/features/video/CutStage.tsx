/**
 * 本文件拥有剪辑台的预览、保留片段、找片段、出片、字幕和配音。
 * 不拥有创作入口。保留、提议、出片、转写、略过、配音、混音和烧录都会改服务端。
 */
import { useEffect, useState } from "react";
import { ArrowLeft } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { ConfirmDialog } from "../ConfirmDialog";
import { cutClips, cutCues, useCutDesk, useCutReadiness, type CutClip, type CutProject } from "./cut-desk";
import { playbackSrc } from "./playback-src";

/** 把秒数收成一位小数的字符串。n 收成秒并保留一位小数，空的当 0。 */
function span(n?: number) {
  return Number(n || 0).toFixed(1);
}

/** 不足 60 秒显示秒，否则显示分和秒。无效时长返回空。n 转成秒，无效就返回空。 */
function formatDuration(n: number | undefined) {
  const s = Number(n || 0);
  if (!(s > 0) || !Number.isFinite(s)) return "";
  if (s < 60) return `${s < 10 ? s.toFixed(1) : Math.round(s)}s`;
  const m = Math.floor(s / 60);
  const r = Math.round(s - m * 60);
  return `${m}:${String(r).padStart(2, "0")}`;
}

/** CutStage 先出预览。没有保留片段时找 3 段、每段 15-30 秒；有了才做出片。props 的 onLeave 在点返回时调用。 */
export function CutStage(props: { onLeave?: () => void }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const project = useCutDesk(/** s 取出 project。 */ (s) => s.project);
  const sessionId = useCutDesk(/** s 取出 sessionId。 */ (s) => s.sessionId);
  const setProject = useCutDesk(/** s 取出 setProject。 */ (s) => s.setProject);
  const ready = useCutReadiness();
  const [busy, setBusy] = useState("");
  const [dubText, setDubText] = useState("");
  const [confirm, setConfirm] = useState<"apply" | "burn" | "">("");
  const [which, setWhich] = useState<"source" | "result">("source");
  const [mediaDuration, setMediaDuration] = useState(0);
  const [pictureReady, setPictureReady] = useState(false);
  const [previewError, setPreviewError] = useState(false);
  const [armed, setArmed] = useState(false);

  const source = playbackSrc(project?.source_url);
  const result = playbackSrc(project?.result_url);
  const poster = playbackSrc(project?.poster_url);
  const preview = which === "result" && result ? result : source;

  useEffect(() => {
    setWhich(result ? "result" : "source");
  }, [project?.id, result]);

  useEffect(() => {
    setMediaDuration(0);
    setPictureReady(false);
    setPreviewError(false);
    setArmed(false);
  }, [preview]);

  /** 跑一次剪辑请求，返回了工程就写回桌子。label 写成忙碌标记。fn 执行，返回带 id 的工程就写回桌子。 */
  async function run(label: string, fn: () => Promise<CutProject | void>) {
    if (!project?.id) return;
    setBusy(label);
    try {
      const next = await fn();
      if (next && next.id) setProject(next, sessionId);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  if (!project?.id || !project.source_hash) return null;
  const id = project.id;
  const clips = cutClips(project);
  const cues = cutCues(project);
  const kept = new Set((project.edit_list?.cuts || []).map(/** clip 取出 id。 */ (clip) => clip.id).filter(Boolean) as string[]);
  const ffmpegReason = ready.loaded && !ready.ffmpeg ? c.needFfmpeg : "";
  const speechReason = ready.loaded && !ready.speech ? c.needSpeech : "";
  const whisperReason = ready.loaded && !ready.whisper ? c.needWhisper : "";
  const length = (project.duration && project.duration > 0) ? project.duration : mediaDuration;
  const lengthLabel = formatDuration(length);
  const status = project.result_hash ? c.statusFilm : clips.length ? c.statusCut : c.statusIn;

  /** 从视频元素记下有限且大于 0 的时长。node 的 duration 有限且大于 0 时记下，并标成画面已就绪。 */
  function noteDuration(node: HTMLVideoElement) {
    const d = node.duration;
    if (Number.isFinite(d) && d > 0) setMediaDuration(d);
    setPictureReady(true);
  }

  /** 勾上或取消这个片段的保留。clip 没有 id 就返回，否则在保留集合里加上或去掉。 */
  function toggleClip(clip: CutClip) {
    if (!clip.id) return;
    const next = new Set(kept);
    if (next.has(clip.id)) next.delete(clip.id);
    else next.add(clip.id);
    void run("keep", async () => api.video.cutKeep(id, [...next]) as Promise<CutProject>);
  }

  return (
    <div className="cut-stage" data-testid="finish-stage">
      <header className="cut-top">
        <button type="button" className="cut-back" data-testid="cut-back" onClick={() => props.onLeave?.()}>
          <ArrowLeft className="size-3.5" aria-hidden />
          {c.backHome}
        </button>
        <div className="cut-top-copy">
          <p>{c.deskEyebrow}</p>
          <h1 className="cut-title">{project.title || c.cutTitle}</h1>
        </div>
        <div className="cut-pills">
          <span className="cut-pill" data-testid="cut-status">{status}</span>
          <span className="cut-pill" data-testid="cut-duration">{lengthLabel || c.durationUnknown}</span>
        </div>
      </header>
      <p className="cut-lead">{c.deskLead}</p>
      <div className="cut-screen" data-testid="cut-player">
        <div className="cut-frame">
          {armed && preview && !previewError ? (
            <video
              key={preview}
              src={preview}
              poster={poster || undefined}
              controls
              playsInline
              autoPlay
              preload="none"
              onLoadedMetadata={/** ev 的 currentTarget 交给 noteDuration。 */ (ev) => noteDuration(ev.currentTarget)}
              onDurationChange={/** ev 的 currentTarget 交给 noteDuration。 */ (ev) => noteDuration(ev.currentTarget)}
              onPlaying={() => setPictureReady(true)}
              onError={() => setPreviewError(true)}
            />
          ) : poster ? (
            <img src={poster} alt="" onLoad={() => setPictureReady(true)} />
          ) : null}
        </div>
        {!armed && preview && !previewError ? (
          <button type="button" className="cut-play" data-testid="cut-play" onClick={() => setArmed(true)}>
            {c.playFilm}
          </button>
        ) : null}
        {previewError || !preview || (armed && !pictureReady) ? (
          <div className="cut-screen-wait">
            <p>{previewError ? c.previewFailed : preview ? c.opening : c.previewEmpty}</p>
          </div>
        ) : null}
      </div>
      {source && result ? (
        <div className="cut-switch" role="group" aria-label={c.deskEyebrow}>
          <button type="button" aria-pressed={which === "source"} onClick={() => setWhich("source")}>{c.sourceFilm}</button>
          <button type="button" aria-pressed={which === "result"} onClick={() => setWhich("result")}>{c.resultFilm}</button>
        </div>
      ) : null}
      {!ready.loaded ? <p className="cut-note">{c.checking}</p> : ffmpegReason ? <p className="cut-warn">{ffmpegReason}</p> : null}
      {project.stale ? <p className="cut-warn">{c.stale}</p> : null}
      {clips.length ? (
        <ul className="cut-list">
          {clips.map(/** clip 画勾选和起止。i 在没有标题时当序号。 */ (clip, i) => (
            <li key={clip.id || i}>
              <label>
                <input
                  type="checkbox"
                  checked={!!clip.id && kept.has(clip.id)}
                  disabled={!!busy || !clip.id}
                  onChange={() => toggleClip(clip)}
                />
                <span>{clip.title || `${c.clips} ${i + 1}`}</span>
              </label>
              <span>{span(clip.start)}–{span(clip.end)}s</span>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="cut-actions">
        {(project.edit_list?.cuts || []).length === 0 ? (
          <Button size="sm" variant="lift" disabled={!!busy || !!ffmpegReason} onClick={() => void run("propose", async () => api.video.cutPropose(id, 3, "15-30") as Promise<CutProject>)}>{busy === "propose" ? c.checking : c.nextFind}</Button>
        ) : (
          <Button size="sm" variant="lift" disabled={!!busy || !!ffmpegReason} onClick={() => setConfirm("apply")}>{busy === "apply" ? c.checking : c.nextMake}</Button>
        )}
      </div>

      <details className="creation-more">
        <summary>{c.captionsDub}</summary>
        <div className="cut-actions">
          <Button size="sm" variant="ghost" disabled={!!busy || !!whisperReason} title={whisperReason} onClick={() => void run("asr", async () => api.video.cutTranscribe(id) as Promise<CutProject>)}>{c.hearLines}</Button>
        </div>
        {cues.length === 0 ? <p className="cut-block-hint">{c.noCues}</p> : (
          <ul className="cut-cues">
            {cues.map(/** cue 显示 text 和是否略过。i 交给 cutOmit。 */ (cue, i) => (
              <li key={`${cue.start}-${i}`} data-omit={cue.omit ? "true" : "false"}>
                <button
                  type="button"
                  disabled={!!busy}
                  onClick={() => void run("omit", async () => api.video.cutOmit(id, i) as Promise<CutProject>)}
                >
                  {cue.omit ? c.restoreCue : c.omitCue}
                </button>
                <p>{cue.text}</p>
              </li>
            ))}
          </ul>
        )}
        <textarea
          className="cut-text"
          value={dubText}
          placeholder={c.dubText}
          onChange={/** e 取出 target.value 写进配音文本。 */ (e) => setDubText(e.target.value)}
        />
        <div className="cut-actions">
          <Button size="sm" variant="ghost" disabled={!!busy || !!speechReason || !dubText.trim()} title={speechReason} onClick={() => void run("dub", async () => {
            await api.video.cutDub(id, dubText);
            return api.video.cutMix(id, true) as Promise<CutProject>;
          })}>{c.makeVoice}</Button>
          <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason || cues.length === 0} onClick={() => setConfirm("burn")}>{c.writeOn}</Button>
        </div>
        {project.dub_url ? (
          <div className="cut-audio">
            <span>{c.dubPreview}</span>
            <audio src={playbackSrc(project.dub_url)} controls />
          </div>
        ) : null}
      </details>
      <ConfirmDialog
        open={confirm === "apply"}
        title={c.confirmApplyTitle}
        body={c.confirmApply}
        confirmLabel={c.apply}
        onCancel={() => setConfirm("")}
        onConfirm={() => {
          setConfirm("");
          void run("apply", async () => api.video.cutApply(id) as Promise<CutProject>);
        }}
      />
      <ConfirmDialog
        open={confirm === "burn"}
        title={c.confirmBurnTitle}
        body={c.confirmBurn}
        confirmLabel={c.burn}
        onCancel={() => setConfirm("")}
        onConfirm={() => {
          setConfirm("");
          void run("burn", async () => api.video.cutBurnin(id) as Promise<CutProject>);
        }}
      />
    </div>
  );
}
