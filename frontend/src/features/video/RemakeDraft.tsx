import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { isVideoPath } from "./ingest-clip";
import { cutCues, useCutReadiness, type CutProject } from "./cut-desk";
import { useRemakeDesk } from "./remake-desk";

function cueText(project: CutProject | null) {
  return cutCues(project).map((cue) => String(cue.text || "").trim()).filter(Boolean).join("\n");
}

function lengthLabel(n: number | undefined) {
  const s = Number(n || 0);
  if (!(s > 0) || !Number.isFinite(s)) return "—";
  if (s < 60) return `${s < 10 ? s.toFixed(1) : Math.round(s)}s`;
  const m = Math.floor(s / 60);
  const r = Math.round(s - m * 60);
  return `${m}:${String(r).padStart(2, "0")}`;
}

export function RemakeDraft(props: { onOpenCut: (id: string) => void }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const draft = useRemakeDesk((s) => s.draft);
  const setDraft = useRemakeDesk((s) => s.setDraft);
  const clear = useRemakeDesk((s) => s.clear);
  const ready = useCutReadiness();
  const [text, setText] = useState("");
  const [busy, setBusy] = useState("");

  useEffect(() => {
    setText(cueText(draft));
  }, [draft?.id, draft?.captions]);

  if (!draft?.id) return null;
  const id = draft.id;
  const cues = cutCues(draft).length;
  const mine = !!draft.reference_hash && !!draft.source_hash && draft.reference_hash !== draft.source_hash;
  const speechReason = ready.loaded && !ready.speech ? c.needSpeech : "";

  async function saveCopy() {
    setBusy("copy");
    try {
      const next = await api.video.cutCaptions(id, text) as CutProject;
      if (next?.id) setDraft(next);
      return true;
    } catch (e) {
      toast.error(api.errMessage(e));
      return false;
    } finally {
      setBusy("");
    }
  }

  async function replacePicture() {
    const picked = await api.pickFiles();
    const path = picked.find((item) => isVideoPath(item)) || "";
    if (!path || !isVideoPath(path)) {
      if (picked.length) toast.error(copy.composer.importClipHint);
      return;
    }
    setBusy("replace");
    const toastId = toast.loading(c.remakeReplacing);
    try {
      const ref = await api.video.putStudioMediaPath(path);
      const hash = String(ref?.cas_hash || ref?.hash || "");
      if (!hash) throw new Error(c.needHash);
      const next = await api.video.remakeReplace(id, hash) as CutProject;
      if (!next?.id) throw new Error(c.needHash);
      setDraft(next);
      toast.success(c.remakeReplaced, { id: toastId });
    } catch (e) {
      toast.error(api.errMessage(e), { id: toastId });
    } finally {
      setBusy("");
    }
  }

  async function dub() {
    setBusy("dub");
    try {
      const next = await api.video.cutDub(id, text) as CutProject;
      if (next?.id) setDraft(next);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function finish() {
    if (text.trim() && text !== cueText(draft)) {
      const saved = await saveCopy();
      if (!saved) return;
    }
    clear();
    props.onOpenCut(id);
  }

  return (
    <section className="creation-remake" data-testid="remake-draft" aria-label={c.remakeDraft}>
      <button type="button" className="cut-back" onClick={() => clear()}>{c.backHome}</button>
      <header>
        <strong>{draft.title || c.remakeDraft}</strong>
        <small>{lengthLabel(draft.duration)}</small>
      </header>
      <p>{mine ? c.remakePictureMine : c.remakePictureRef}</p>
      <label>
        <span>{c.remakeCopy}</span>
        <textarea value={text} onChange={(ev) => setText(ev.target.value)} placeholder={cues === 0 ? c.remakeNoCues : ""} />
      </label>
      <div className="creation-remake-actions">
        {mine ? (
          <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void finish()}>{c.remakeSend}</Button>
        ) : (
          <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void replacePicture()}>{busy === "replace" ? c.remakeReplacing : c.remakeReplace}</Button>
        )}
        {mine ? (
          <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => void replacePicture()}>{c.remakeReplace}</Button>
        ) : (
          <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => void finish()}>{c.remakeSend}</Button>
        )}
      </div>
      <details className="creation-more">
        <summary>{c.captionsDub}</summary>
        <Button size="sm" variant="ghost" disabled={!!busy || !!speechReason || !text.trim()} title={speechReason || c.remakeDub} onClick={() => void dub()}>{c.makeVoice}</Button>
      </details>
    </section>
  );
}
