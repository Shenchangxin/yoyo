import { useEffect, useLayoutEffect, useRef, useState, type Ref } from "react";
import { AtSign, BookOpen, Boxes, Camera, Clipboard, FileText, Folder, Paperclip, Terminal } from "lucide-react";
import type { MentionKind } from "../../lib/mentions";
import { cn } from "../../lib/utils";

export type OverlayKind = MentionKind | "command" | "clipboard" | "screenshot" | "attach";

export type OverlayItem = {
  token: string;
  hint: string;
  label?: string;
  kind?: OverlayKind;
  section?: string;
  alias?: string;
};

function Glyph({ kind }: { kind?: OverlayKind }) {
  if (kind === "folder") return <Folder className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "skill") return <BookOpen className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "harness") return <Boxes className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "file") return <FileText className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "attach") return <Paperclip className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "clipboard") return <Clipboard className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "screenshot") return <Camera className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "command") return <Terminal className="size-3.5 shrink-0 opacity-70" />;
  return <AtSign className="size-3.5 shrink-0 opacity-70" />;
}

export function CommandOverlay(props: {
  label: string;
  items: OverlayItem[];
  hi: number;
  empty?: string;
  onPick: (token: string) => void;
  onHover?: (index: number) => void;
}) {
  const activeRef = useRef<HTMLButtonElement>(null);
  const boxRef = useRef<HTMLDivElement>(null);
  const [maxHeight, setMaxHeight] = useState(400);
  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: "nearest" });
  }, [props.hi]);
  useLayoutEffect(() => {
    const card = boxRef.current?.closest("[data-composer-card]");
    const top = card instanceof HTMLElement ? card.getBoundingClientRect().top : 84;
    setMaxHeight(Math.max(120, Math.min(400, top - 12)));
  }, [props.items.length]);
  return (
    <div ref={boxRef} className="sc-overlay">
      <div role="listbox" aria-label={props.label} data-testid="composer-popup" className="sc-popup" style={{ maxHeight }}>
        {props.items.length ? props.items.map((m, i) => (
          <FragmentRow
            key={m.token + i}
            item={m}
            prev={props.items[i - 1]}
            active={i === props.hi}
            rowRef={i === props.hi ? activeRef : undefined}
            onPick={() => props.onPick(m.token)}
            onHover={() => props.onHover?.(i)}
          />
        )) : props.empty ? (
          <div className="px-3 py-2 text-[11px] text-muted">{props.empty}</div>
        ) : null}
      </div>
    </div>
  );
}

function FragmentRow(props: {
  item: OverlayItem;
  prev?: OverlayItem;
  active: boolean;
  rowRef?: Ref<HTMLButtonElement>;
  onPick: () => void;
  onHover?: () => void;
}) {
  const showSection = props.item.section && props.item.section !== props.prev?.section;
  return (
    <>
      {showSection ? (
        <div className="sc-popup-section" role="presentation">{props.item.section}</div>
      ) : null}
      <PopupRow
        item={props.item}
        active={props.active}
        rowRef={props.rowRef}
        onPick={props.onPick}
        onHover={props.onHover}
      />
    </>
  );
}

function PopupRow(props: {
  item: OverlayItem;
  active: boolean;
  rowRef?: Ref<HTMLButtonElement>;
  onPick: () => void;
  onHover?: () => void;
}) {
  const title = props.item.label || props.item.token;
  return (
    <button
      ref={props.rowRef}
      type="button"
      role="option"
      aria-selected={props.active}
      aria-label={`${title} ${props.item.hint || ""}`.trim()}
      className={cn("sc-popup-row", props.active && "is-active")}
      data-active={props.active ? "true" : "false"}
      onMouseDown={(e) => {
        e.preventDefault();
        props.onPick();
      }}
      onMouseMove={props.active ? undefined : props.onHover}
    >
      <span className="sc-popup-icon" aria-hidden><Glyph kind={props.item.kind} /></span>
      <span className="sc-popup-name">{title}</span>
      {props.item.alias ? <span className="sc-popup-alias">{props.item.alias}</span> : null}
      {props.item.hint ? <span className="sc-popup-hint">{props.item.hint}</span> : null}
    </button>
  );
}
