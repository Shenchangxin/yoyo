/**
 * 拥有：设置里用来填写的弹框。
 * 不拥有：里面的字段。调用方传入标题和表单。
 */
import { useEffect, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

/** FormDialog 打开后挂到 document.body。Escape 和点遮罩都会关上。 */
export function FormDialog(props: {
  open: boolean;
  title: string;
  description?: string;
  onClose: () => void;
  children: ReactNode;
}) {
  const [armed, setArmed] = useState(false);

  useEffect(() => {
    if (!props.open) {
      setArmed(false);
      return;
    }
    const t = window.setTimeout(() => setArmed(true), 220);
    return () => window.clearTimeout(t);
  }, [props.open]);

  useEffect(() => {
    if (!props.open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") props.onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [props.open, props.onClose]);

  if (!props.open || typeof document === "undefined") return null;
  return createPortal(
    <div
      className="overlay-scrim fixed inset-0 z-[200] grid place-items-center"
      role="presentation"
      onClick={() => {
        if (armed) props.onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="yoyo-form-title"
        className="dialog-sheet max-h-[min(640px,calc(100%-2rem))] w-[min(440px,calc(100%-2rem))] overflow-y-auto rounded-xl border border-border bg-popover p-6 shadow-[var(--shadow-popover)]"
        onClick={(e) => e.stopPropagation()}
        onPointerDown={(e) => e.stopPropagation()}
      >
        <h2 id="yoyo-form-title" className="text-ui-title-s font-semibold">{props.title}</h2>
        {props.description ? <p className="text-ui-body leading-[1.55] text-muted">{props.description}</p> : null}
        {props.children}
      </div>
    </div>,
    document.body,
  );
}
