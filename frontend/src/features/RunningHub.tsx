import { useEffect, useState } from "react";
import { Badge } from "../components/ui/badge";
import { useCopy } from "../lib/i18n";
import type { Thread } from "../lib/protocol";

export function RunningHub(props: {
  threads: Thread[];
  onSelect: (t: Thread) => void;
}) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") close(); };
    const tmr = window.setTimeout(() => {
      window.addEventListener("mousedown", close);
      window.addEventListener("keydown", onKey);
    }, 0);
    return () => {
      window.clearTimeout(tmr);
      window.removeEventListener("mousedown", close);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);
  if (!props.threads.length) return null;
  return (
    <div className="relative hidden sm:block">
      <button
        type="button"
        className="flex items-center"
        aria-expanded={open}
        aria-haspopup="listbox"
        onClick={() => setOpen((v) => !v)}
      >
        <Badge>
          {props.threads.length} {copy.titlebar.live}
        </Badge>
      </button>
      {open ? (
        <div
          role="listbox"
          className="absolute right-0 z-30 mt-1 min-w-[200px] overflow-hidden rounded-xl border border-border bg-popover py-1 shadow-[var(--shadow-popover)]"
          onMouseDown={(e) => e.stopPropagation()}
        >
          {props.threads.map((t) => (
            <button
              type="button"
              key={t.id}
              role="option"
              className="block w-full truncate px-3 py-1.5 text-left text-[12px] hover:bg-lift"
              onClick={() => {
                props.onSelect(t);
                setOpen(false);
              }}
            >
              {t.title || t.id.slice(0, 8)}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
