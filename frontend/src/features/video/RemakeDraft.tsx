/**
 * 本文件拥有翻拍草稿的文案、换画面和配音按钮。不拥有剪辑时间线。
 * 保存字幕、换画面和配音会改服务端上的剪辑工程。
 */
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { isVideoPath } from "./ingest-clip";
import { cutCues, useCutReadiness, type CutProject } from "./cut-desk";
import { useRemakeDesk } from "./remake-desk";

/** 把字幕条目收成多行文本。project 交给 cutCues。 */
function cueText(project: CutProject | null) {
  return cutCues(project).map(/** cue 取出 text 并去掉空白。 */ (cue) => String(cue.text || "").trim()).filter(Boolean).join("\n");
}

/** 不足 60 秒显示秒，否则显示分和秒。无效时长显示破折号。n 转成秒，无效就显示破折号。 */
function lengthLabel(n: number | undefined) {
  const s = Number(n || 0);
  if (!(s > 0) || !Number.isFinite(s)) return "—";
  if (s < 60) return `${s < 10 ? s.toFixed(1) : Math.round(s)}s`;
  const m = Math.floor(s / 60);
  const r = Math.round(s - m * 60);
  return `${m}:${String(r).padStart(2, "0")}`;
}

/** RemakeDraft 编辑翻拍文案。自己的画面时主按钮是送去剪辑，否则主按钮是换画面。props 的 onOpenCut 在文案保存后打开这条剪辑。 */
export function RemakeDraft(props: { onOpenCut: (id: string) => void }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const draft = useRemakeDesk(/** s 取出 draft。 */ (s) => s.draft);
  const setDraft = useRemakeDesk(/** s 取出 setDraft。 */ (s) => s.setDraft);
  const clear = useRemakeDesk(/** s 取出 clear。 */ (s) => s.clear);
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

  /** 把文本写成这条剪辑的字幕。 */
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

  /** 选一个视频路径，登记成媒体后换掉这条翻拍的画面。 */
  async function replacePicture() {
    const picked = await api.pickFiles();
    const path = picked.find(/** item 是视频路径才留下。 */ (item) => isVideoPath(item)) || "";
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

  /** 用当前文本给这条剪辑配音。 */
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

  /** 文案有改动就先保存，然后清掉草稿并打开剪辑。 */
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
        <textarea value={text} onChange={/** ev 取出 target.value 写进文案。 */ (ev) => setText(ev.target.value)} placeholder={cues === 0 ? c.remakeNoCues : ""} />
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
