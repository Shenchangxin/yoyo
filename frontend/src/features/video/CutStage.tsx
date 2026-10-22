import { useEffect, useState } from "react";
import { ArrowLeft } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { ConfirmDialog } from "../ConfirmDialog";
import { cutClips, cutCues, useCutDesk, useCutReadiness, type CutClip, type CutProject } from "./cut-desk";
import { playbackSrc } from "./playback-src";

function span(n?: number) {
  return Number(n || 0).toFixed(1);
}

function formatDuration(n: number | undefined) {
  const s = Number(n || 0);
  if (!(s > 0) || !Number.isFinite(s)) return "";
  if (s < 60) return `${s < 10 ? s.toFixed(1) : Math.round(s)}s`;
  const m = Math.floor(s / 60);
  const r = Math.round(s - m * 60);
  return `${m}:${String(r).padStart(2, "0")}`;
}

export function CutStage(props: { onLeave?: () => void }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const project = useCutDesk((s) => s.project);
  const sessionId = useCutDesk((s) => s.sessionId);
  const setProject = useCutDesk((s) => s.setProject);
  const ready = useCutReadiness();
  const [busy, setBusy] = useState("");
  const [duck, setDuck] = useState(true);
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
  const scenes = project.breakdown?.scenes || [];
  const kept = new Set((project.edit_list?.cuts || []).map((clip) => clip.id).filter(Boolean) as string[]);
  const ffmpegReason = ready.loaded && !ready.ffmpeg ? c.needFfmpeg : "";
  const speechReason = ready.loaded && !ready.speech ? c.needSpeech : "";
  const whisperReason = ready.loaded && !ready.whisper ? c.needWhisper : "";
  const blockers = [ffmpegReason, speechReason, whisperReason].filter(Boolean);
  const length = (project.duration && project.duration > 0) ? project.duration : mediaDuration;
  const lengthLabel = formatDuration(length);
  const status = project.result_hash ? c.statusFilm : clips.length ? c.statusCut : c.statusIn;
  const watchState = pictureReady || !!poster ? "done" : "now";
  const cutState = clips.length ? "done" : "now";
  const finishState = project.result_hash ? "done" : clips.length ? "now" : "later";

  function noteDuration(node: HTMLVideoElement) {
    const d = node.duration;
    if (Number.isFinite(d) && d > 0) setMediaDuration(d);
    setPictureReady(true);
  }

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
              onLoadedMetadata={(ev) => noteDuration(ev.currentTarget)}
              onDurationChange={(ev) => noteDuration(ev.currentTarget)}
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
      <ol className="cut-steps">
        <li className="cut-step" data-state={watchState}>
          <span>1</span>
          <strong>{c.stepWatch}</strong>
          <small>{c.stepWatchHint}</small>
        </li>
        <li className="cut-step" data-state={cutState}>
          <span>2</span>
          <strong>{c.stepCut}</strong>
          <small>{c.stepCutHint}</small>
        </li>
        <li className="cut-step" data-state={finishState}>
          <span>3</span>
          <strong>{c.stepFinish}</strong>
          <small>{c.stepFinishHint}</small>
        </li>
      </ol>
      {!ready.loaded ? <p className="cut-note">{c.checking}</p> : blockers.length ? <p className="cut-warn">{blockers.join(" · ")}</p> : null}
      {project.stale ? <p className="cut-warn">{c.stale}</p> : null}

      <section className="cut-block">
        <h2>{c.stepCut}</h2>
        <p className="cut-block-hint">{c.stepCutHint}</p>
        <div className="cut-actions">
          <Button size="sm" variant="lift" disabled={!!busy || !!ffmpegReason} onClick={() => void run("analyze", async () => api.video.cutAnalyze(id) as Promise<CutProject>)}>{busy === "analyze" ? c.checking : c.analyze}</Button>
          <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason} onClick={() => void run("propose", async () => api.video.cutPropose(id, 3, "15-30") as Promise<CutProject>)}>{c.propose}</Button>
        </div>
        <h3>{c.scenes}</h3>
        {scenes.length === 0 ? <p className="cut-block-hint">{c.noScenes}</p> : (
          <ul className="cut-scenes">
            {scenes.map((t, i) => <li key={`${t}-${i}`}>{span(t)}s</li>)}
          </ul>
        )}
        <h3>{c.clips}</h3>
        {clips.length === 0 ? <p className="cut-block-hint">{c.noClips}</p> : (
          <ul className="cut-list">
            {clips.map((clip, i) => (
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
                <span>
                  {span(clip.start)}–{span(clip.end)}s
                  {" · "}
                  {c.energy} {clip.scores?.energy ?? "—"}
                  {" · "}
                  {c.complete} {clip.scores?.completeness ?? "—"}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="cut-block">
        <h2>{c.stepFinish}</h2>
        <p className="cut-block-hint">{c.stepFinishHint}</p>
        <div className="cut-actions">
          <Button size="sm" variant="lift" disabled={!!busy || !!ffmpegReason || (project.edit_list?.cuts || []).length === 0} onClick={() => setConfirm("apply")}>{c.apply}</Button>
          <Button size="sm" variant="ghost" disabled={!!busy || !!whisperReason} title={whisperReason} onClick={() => void run("asr", async () => api.video.cutTranscribe(id) as Promise<CutProject>)}>{c.transcribe}</Button>
        </div>
        <h3>{c.captions}</h3>
        {cues.length === 0 ? <p className="cut-block-hint">{whisperReason || c.noCues}</p> : (
          <ul className="cut-cues">
            {cues.map((cue, i) => (
              <li key={`${cue.start}-${i}`} data-omit={cue.omit ? "true" : "false"}>
                <button
                  type="button"
                  disabled={!!busy}
                  onClick={() => void run("omit", async () => api.video.cutOmit(id, i) as Promise<CutProject>)}
                >
                  {cue.omit ? c.restoreCue : c.omitCue}
                </button>
                <span>{span(cue.start)}–{span(cue.end)}</span>
                <p>{cue.text}</p>
              </li>
            ))}
          </ul>
        )}
        <h3>{c.dub}</h3>
        <textarea
          className="cut-text"
          value={dubText}
          placeholder={c.dubText}
          onChange={(e) => setDubText(e.target.value)}
        />
        <div className="cut-actions">
          <Button size="sm" variant="ghost" disabled={!!busy || !!speechReason} title={speechReason} onClick={() => void run("dub", async () => api.video.cutDub(id, dubText) as Promise<CutProject>)}>{c.dub}</Button>
          <label className="cut-duck">
            <input type="checkbox" checked={duck} onChange={(e) => setDuck(e.target.checked)} />
            {c.duck}
          </label>
          <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason || !(project.stems?.dub || project.dub_url)} onClick={() => void run("mix", async () => api.video.cutMix(id, duck) as Promise<CutProject>)}>{c.mix}</Button>
          <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason || cues.length === 0} onClick={() => setConfirm("burn")}>{c.burn}</Button>
        </div>
        {project.dub_url ? (
          <div className="cut-audio">
            <span>{c.dubPreview}</span>
            <audio src={playbackSrc(project.dub_url)} controls />
          </div>
        ) : null}
      </section>
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
