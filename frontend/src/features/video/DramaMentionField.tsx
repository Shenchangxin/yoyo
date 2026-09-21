import { useMemo, useRef, useState, type TextareaHTMLAttributes } from "react";
import { Textarea } from "../../components/ui/input";
import { cn } from "../../lib/utils";

export type MentionAsset = { name: string; kind: "character" | "scene" | "prop" };

export function DramaMentionField({
  assets,
  className,
  ...props
}: TextareaHTMLAttributes<HTMLTextAreaElement> & { assets: MentionAsset[] }) {
  const ref = useRef<HTMLTextAreaElement>(null);
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState("");
  const ranked = useMemo(() => {
    const names = [...assets].sort((a, b) => b.name.length - a.name.length);
    const needle = q.toLowerCase();
    return names.filter((a) => !needle || a.name.toLowerCase().includes(needle) || a.name.startsWith(q));
  }, [assets, q]);

  return (
    <div className="relative">
      <Textarea
        {...props}
        ref={ref}
        className={cn("min-h-[88px] rounded-lg border border-border bg-background px-3 py-2 text-[13px] leading-5", className)}
        onChange={(e) => {
          props.onChange?.(e);
          const caret = e.target.selectionStart || 0;
          const left = e.target.value.slice(0, caret);
          const m = left.match(/@([^\s@]*)$/);
          if (m) {
            setQ(m[1]);
            setOpen(true);
          } else {
            setOpen(false);
          }
        }}
        onBlur={() => window.setTimeout(() => setOpen(false), 120)}
      />
      {open && ranked.length > 0 ? (
        <div className="absolute bottom-full z-20 mb-1 max-h-40 w-full overflow-auto rounded-lg border border-border bg-popover py-1 shadow-[var(--shadow-popover)]">
          {ranked.slice(0, 8).map((a) => (
            <button
              key={a.kind + a.name}
              type="button"
              className="flex w-full items-center justify-between px-2.5 py-1.5 text-left text-[12px] hover:bg-lift"
              onMouseDown={(ev) => {
                ev.preventDefault();
                const el = ref.current;
                if (!el) return;
                const caret = el.selectionStart || 0;
                const left = el.value.slice(0, caret);
                const start = left.lastIndexOf("@");
                const next = el.value.slice(0, start) + "@" + a.name + " " + el.value.slice(caret);
                el.value = next;
                props.onChange?.({ target: el } as any);
                setOpen(false);
              }}
            >
              <span className="truncate font-medium">{a.name}</span>
              <span className="text-[10px] uppercase tracking-[0.06em] text-muted">{a.kind}</span>
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
