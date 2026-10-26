// 本文件拥有确认层：取消、确认，以及打开后的短暂武装。不拥有确认之后做什么。调用方传入标题、正文和 onConfirm。

import { useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Button } from "../components/ui/button";
import { useCopy } from "../lib/i18n";

// ConfirmDialog 没打开或没有 document 时不渲染，打开后挂到 document.body。220 毫秒后才允许点遮罩取消。Escape 在忙时不取消。
export function ConfirmDialog(props: {
  open: boolean;
  title: string;
  body: ReactNode;
  confirmLabel?: string;
  danger?: boolean;
  onCancel: () => void;
  onConfirm: () => void | Promise<void>;
}) {
  const copy = useCopy();
  const [busy, setBusy] = useState(false);
  const [armed, setArmed] = useState(false);
  const ran = useRef(false);

  useEffect(() => {
    if (!props.open) {
      setBusy(false);
      setArmed(false);
      ran.current = false;
      return;
    }
    const t = window.setTimeout(() => setArmed(true), 220);
    return () => window.clearTimeout(t);
  }, [props.open]);

  useEffect(() => {
    if (!props.open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !busy) props.onCancel();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [props.open, busy, props.onCancel]);

  // 忙着或这一轮已经跑过就直接返回，避免确认点两次。
  async function confirm() {
    if (busy || ran.current) return;
    ran.current = true;
    setBusy(true);
    try {
      await props.onConfirm();
    } finally {
      ran.current = false;
      setBusy(false);
    }
  }

  if (!props.open || typeof document === "undefined") return null;
  return createPortal(
    <div
      className="overlay-scrim fixed inset-0 z-[200] grid place-items-center"
      role="presentation"
      onClick={() => {
        if (armed && !busy) props.onCancel();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="yoyo-confirm-title"
        className="dialog-sheet w-[min(420px,calc(100%-2rem))] rounded-2xl border border-border bg-popover p-6 shadow-[var(--shadow-popover)]"
        onClick={(e) => e.stopPropagation()}
        onPointerDown={(e) => e.stopPropagation()}
      >
        <h2 id="yoyo-confirm-title" className="text-[15px] font-semibold">{props.title}</h2>
        {typeof props.body === "string" ? (
          <p className="text-[13px] leading-[1.55] text-muted">{props.body}</p>
        ) : (
          <div className="text-[13px] leading-[1.55] text-muted">{props.body}</div>
        )}
        <div className="dialog-actions">
          <Button
            variant="lift"
            disabled={busy}
            onClick={() => {
              if (!busy) props.onCancel();
            }}
          >
            {copy.dialog.cancel}
          </Button>
          <Button
            variant={props.danger ? "danger" : "default"}
            disabled={busy}
            onClick={() => {
              void confirm();
            }}
          >
            {props.confirmLabel || copy.dialog.confirm}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
