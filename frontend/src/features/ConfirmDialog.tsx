import { useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Button } from "../components/ui/button";
import { useCopy } from "../lib/i18n";

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
      className="fixed inset-0 z-[200] grid place-items-center bg-background/70"
      role="presentation"
      onClick={() => {
        if (armed && !busy) props.onCancel();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="yoyo-confirm-title"
        className="command-menu-sheen w-[min(420px,calc(100%-2rem))] rounded-2xl border border-border/80 bg-popover p-5 shadow-[var(--shadow-popover)]"
        onClick={(e) => e.stopPropagation()}
        onPointerDown={(e) => e.stopPropagation()}
      >
        <h2 id="yoyo-confirm-title" className="text-[15px] font-semibold">{props.title}</h2>
        <p className="mt-2 text-[13px] text-muted">{props.body}</p>
        <div className="mt-5 flex justify-end gap-2">
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
