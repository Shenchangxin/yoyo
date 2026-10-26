/**
 * 拥有：作曲触发器的浮层。颜色用本应用的变量。
 * 不拥有：提及解析。选择交给外面，不调用 api。
 */
import { Fragment, useEffect, useLayoutEffect, useRef, useState, type Ref } from "react";
import { AtSign, BookOpen, Boxes, Camera, ChevronRight, Clipboard, FileText, Folder, MessageSquare, Paperclip, Terminal } from "lucide-react";
import type { MentionCrumb, MentionKind } from "../../lib/mentions";
import { cn } from "../../lib/utils";

/** OverlayKind 是浮层一行的种类。 */
export type OverlayKind = MentionKind | "command" | "clipboard" | "screenshot" | "attach";

/** OverlayItem 是浮层里的一行：记号、说明和种类。 */
export type OverlayItem = {
  token: string;
  hint: string;
  label?: string;
  kind?: OverlayKind;
  section?: string;
  alias?: string;
  drill?: boolean;
  drillPath?: string;
};

/** OverlayCrumb 沿用提及的面包屑。 */
export type OverlayCrumb = MentionCrumb;

/** 按种类画一个小图标。 */
function Glyph({ kind }: { kind?: OverlayKind }) {
  if (kind === "folder") return <Folder className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "skill") return <BookOpen className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "harness") return <Boxes className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "file") return <FileText className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "session") return <MessageSquare className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "attach") return <Paperclip className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "clipboard") return <Clipboard className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "screenshot") return <Camera className="size-3.5 shrink-0 opacity-70" />;
  if (kind === "command") return <Terminal className="size-3.5 shrink-0 opacity-70" />;
  return <AtSign className="size-3.5 shrink-0 opacity-70" />;
}

/** CommandOverlay 列出触发后的选项，选中交给外面。 */
export function CommandOverlay(props: {
  label: string;
  items: OverlayItem[];
  hi: number;
  empty?: string;
  crumbs?: OverlayCrumb[];
  crumbsLabel?: string;
  drillHint?: string;
  drillKey?: string;
  drillAria?: string;
  onPick: (item: OverlayItem) => void;
  onDrill?: (item: OverlayItem) => void;
  onCrumb?: (crumb: OverlayCrumb) => void;
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
  }, [props.items.length, props.crumbs?.length]);
  const crumbs = props.crumbs?.length ? props.crumbs : undefined;
  return (
    <div ref={boxRef} className="sc-overlay">
      <div className="sc-popup" style={{ maxHeight }}>
        {crumbs ? (
          <nav className="sc-crumbs" aria-label={props.crumbsLabel || props.label}>
            {crumbs.map((crumb, index) => (
              <Fragment key={`${index}-${crumb.query}`}>
                {index > 0 ? <span className="sc-crumb-sep" aria-hidden><ChevronRight className="size-3" /></span> : null}
                <button
                  type="button"
                  className={cn("sc-crumb", crumb.current && "is-current")}
                  aria-current={crumb.current ? "location" : undefined}
                  disabled={crumb.current}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    props.onCrumb?.(crumb);
                  }}
                >
                  {crumb.label}
                </button>
              </Fragment>
            ))}
          </nav>
        ) : null}
        <div
          role="listbox"
          aria-label={props.label}
          aria-activedescendant={props.items[props.hi] ? overlayOptionId(props.hi) : undefined}
          data-testid="composer-popup"
          className="sc-popup-viewport"
        >
          {props.items.length ? props.items.map((m, i) => (
            <FragmentRow
              key={m.token + i}
              item={m}
              index={i}
              prev={props.items[i - 1]}
              active={i === props.hi}
              rowRef={i === props.hi ? activeRef : undefined}
              drillHint={props.drillHint}
              drillKey={props.drillKey}
              drillAria={props.drillAria}
              onPick={() => props.onPick(m)}
              onDrill={m.drill && props.onDrill ? () => props.onDrill?.(m) : undefined}
              onHover={() => props.onHover?.(i)}
            />
          )) : props.empty ? (
            <div className="px-3 py-2 text-[11px] text-muted">{props.empty}</div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

/** 给一行选项一个稳定的元素编号。 */
function overlayOptionId(index: number) {
  return `composer-option-${index}`;
}

/** 画面包屑里的一段。 */
function FragmentRow(props: {
  item: OverlayItem;
  index: number;
  prev?: OverlayItem;
  active: boolean;
  rowRef?: Ref<HTMLButtonElement>;
  drillHint?: string;
  drillKey?: string;
  drillAria?: string;
  onPick: () => void;
  onDrill?: () => void;
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
        index={props.index}
        active={props.active}
        rowRef={props.rowRef}
        drillHint={props.drillHint}
        drillKey={props.drillKey}
        drillAria={props.drillAria}
        onPick={props.onPick}
        onDrill={props.onDrill}
        onHover={props.onHover}
      />
    </>
  );
}

/** 画浮层里的一行选项。 */
function PopupRow(props: {
  item: OverlayItem;
  index: number;
  active: boolean;
  rowRef?: Ref<HTMLButtonElement>;
  drillHint?: string;
  drillKey?: string;
  drillAria?: string;
  onPick: () => void;
  onDrill?: () => void;
  onHover?: () => void;
}) {
  const title = props.item.label || props.item.token;
  return (
    <button
      ref={props.rowRef}
      id={overlayOptionId(props.index)}
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
      {props.item.drill ? (
        <span className="sc-popup-trailing">
          <span className="sc-drill-hint-text" aria-hidden>{props.drillHint}</span>
          <kbd className="sc-drill-hint" aria-hidden>{props.drillKey}</kbd>
          <span
            role="button"
            aria-label={props.drillAria}
            className="sc-drill"
            onMouseDown={(e) => {
              e.preventDefault();
              e.stopPropagation();
              props.onDrill?.();
            }}
          >
            <ChevronRight className="size-3" />
          </span>
        </span>
      ) : null}
    </button>
  );
}
