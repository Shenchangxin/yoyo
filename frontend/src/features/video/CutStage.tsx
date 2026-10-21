import { useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { ConfirmDialog } from "../ConfirmDialog";
import { cutClips, cutCues, useCutDesk, useCutReadiness, type CutClip, type CutProject } from "./cut-desk";

function span(n?: number) {
  return Number(n || 0).toFixed(1);
}

export function CutStage() {
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
  const kept = new Set((project.edit_list?.cuts || []).map((clip) => clip.id).filter(Boolean) as string[]);
  const preview = project.result_url || project.source_url || project.media_url || "";
  const ffmpegReason = ready.loaded && !ready.ffmpeg ? c.needFfmpeg : "";
  const speechReason = ready.loaded && !ready.speech ? c.needSpeech : "";
  const whisperReason = ready.loaded && !ready.whisper ? c.needWhisper : "";

  function toggleClip(clip: CutClip) {
    if (!clip.id) return;
    const next = new Set(kept);
    if (next.has(clip.id)) next.delete(clip.id);
    else next.add(clip.id);
    void run("keep", async () => api.video.cutKeep(id, [...next]) as Promise<CutProject>);
  }

  return (
    <div className="finish-stage" data-testid="finish-stage">
      <header className="finish-stage-head">
        <div>
          <p>{c.label}</p>
          <h1>{project.title || c.cutTitle}</h1>
        </div>
        {project.duration ? <span>{span(project.duration)}s</span> : null}
      </header>
      <p className="finish-stage-ready">
        {ffmpegReason || (ready.ffmpeg ? c.ffmpegReady : c.checking)}
        {" · "}
        {speechReason || (ready.speech ? c.speechReady : c.checking)}
        {" · "}
        {whisperReason || (ready.whisper ? c.whisperReady : c.checking)}
      </p>
      {preview ? (
        <video className="finish-stage-player" src={preview} poster={project.poster_url || undefined} controls />
      ) : null}
      {project.stale ? <p className="finish-stage-stale">{c.stale}</p> : null}
      <div className="finish-stage-actions">
        <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason} onClick={() => void run("analyze", async () => api.video.cutAnalyze(id) as Promise<CutProject>)}>{c.analyze}</Button>
        <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason} onClick={() => void run("propose", async () => api.video.cutPropose(id, 3, "15-30") as Promise<CutProject>)}>{c.propose}</Button>
        <Button size="sm" variant="lift" disabled={!!busy || !!ffmpegReason || (project.edit_list?.cuts || []).length === 0} onClick={() => setConfirm("apply")}>{c.apply}</Button>
        <Button size="sm" variant="ghost" disabled={!!busy || !!whisperReason} title={whisperReason} onClick={() => void run("asr", async () => api.video.cutTranscribe(id) as Promise<CutProject>)}>{c.transcribe}</Button>
      </div>
      <section>
        <h2>{c.clips}</h2>
        {clips.length === 0 ? <p className="finish-stage-empty">—</p> : (
          <ul className="finish-stage-list">
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
                  {span(clip.start)}–{span(clip.end)}
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
      <section>
        <h2>{c.scenes}</h2>
        <p className="finish-stage-empty">{(project.breakdown?.scenes || []).length || "—"}</p>
      </section>
      <section>
        <h2>{c.captions}</h2>
        {cues.length === 0 ? <p className="finish-stage-empty">{whisperReason || "—"}</p> : (
          <ul className="finish-stage-cues">
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
      </section>
      <section>
        <h2>{c.dub}</h2>
        <textarea
          value={dubText}
          placeholder={c.dubText}
          onChange={(e) => setDubText(e.target.value)}
        />
        <div className="finish-stage-actions">
          <Button size="sm" variant="ghost" disabled={!!busy || !!speechReason} title={speechReason} onClick={() => void run("dub", async () => api.video.cutDub(id, dubText) as Promise<CutProject>)}>{c.dub}</Button>
          <label className="finish-stage-duck">
            <input type="checkbox" checked={duck} onChange={(e) => setDuck(e.target.checked)} />
            {c.duck}
          </label>
          <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason || !(project.stems?.dub || project.dub_url)} onClick={() => void run("mix", async () => api.video.cutMix(id, duck) as Promise<CutProject>)}>{c.mix}</Button>
          <Button size="sm" variant="ghost" disabled={!!busy || !!ffmpegReason || cues.length === 0} onClick={() => setConfirm("burn")}>{c.burn}</Button>
        </div>
        {project.dub_url ? (
          <div className="finish-stage-preview">
            <span>{c.dubPreview}</span>
            <audio src={project.dub_url} controls />
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
