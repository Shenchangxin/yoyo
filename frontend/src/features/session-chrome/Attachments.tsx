/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized composer attachment rail + drop overlay.
 */
import { useEffect, useMemo, useState } from "react";
import { Camera, Paperclip, X } from "lucide-react";
import type { Attachment } from "../../lib/protocol";
import { useCopy } from "../../lib/i18n";

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

export function DropOverlay(props: { active: boolean }) {
  const copy = useCopy();
  if (!props.active) return null;
  return <div className="sc-drop">{copy.composer.attach}</div>;
}

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
