/**
 * 拥有：输入栏上的附件条和拖放遮罩。
 * 不拥有：上传。移除交给外面，不调用 api。
 */
import { useEffect, useMemo, useState } from "react";
import { Camera, Paperclip, X } from "lucide-react";
import type { Attachment } from "../../lib/protocol";
import { useCopy } from "../../lib/i18n";

/** AttachmentRail 把附件排成一排，点一下交给外面移除。 */
export function AttachmentRail(props: {
  items: Attachment[];
  onRemove: (index: number) => void;
}) {
  if (!props.items.length) return null;
  return (
    <div className="sc-rail" data-testid="composer-attachments">
      {props.items.map((a, i) => (
        <AttachmentChip key={`${a.name}-${i}`} att={a} onRemove={() => props.onRemove(i)} />
      ))}
    </div>
  );
}

/** DropOverlay 在拖进文件时盖住一层提示。 */
export function DropOverlay(props: { active: boolean; label?: string }) {
  const copy = useCopy();
  if (!props.active) return null;
  return <div className="sc-drop">{props.label || copy.composer.attach}</div>;
}

/** 画一张附件：图片用缩略图，其余用文件名。 */
function AttachmentChip({ att, onRemove }: { att: Attachment; onRemove: () => void }) {
  const name = att.name || att.path || "file";
  const preview = useMemo(() => {
    if (att.mime?.startsWith("image/") && att.data_b64) return `data:${att.mime};base64,${att.data_b64}`;
    if (att.mime?.startsWith("image/") && att.path && !att.path.startsWith("yoyo-cut:")) {
      try {
        return new URL(att.path).protocol === "file:" ? "" : att.path;
      } catch {
        return "";
      }
    }
    return "";
  }, [att]);
  const [src, setSrc] = useState(preview);
  useEffect(() => setSrc(preview), [preview]);
  if (src) {
    return (
      <button type="button" className="relative h-11 w-11 overflow-hidden rounded-lg" onClick={onRemove} aria-label={name}>
        <img src={src} alt="" className="h-full w-full object-cover" />
        <span className="absolute right-0.5 top-0.5 grid size-3.5 place-items-center rounded-full bg-card/90 text-muted">
          <X className="size-2.5" />
        </span>
      </button>
    );
  }
  return (
    <button
      type="button"
      className="inline-flex max-w-[11rem] items-center gap-1 rounded-md bg-lift px-1.5 py-0.5 text-[11px] text-muted hover:text-foreground"
      onClick={onRemove}
      aria-label={name}
    >
      {att.mime?.startsWith("image/") ? <Camera className="size-3 shrink-0" /> : <Paperclip className="size-3 shrink-0" />}
      <span className="truncate">{name}</span>
      <X className="size-3 shrink-0 opacity-70" />
    </button>
  );
}
