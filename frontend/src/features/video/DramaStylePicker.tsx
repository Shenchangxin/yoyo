/**
 * 本文件拥有短剧风格选择面板的位置、搜索和分组。不拥有风格表。
 * 选中只交给回调，本文件不改服务端。
 */
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown, Search } from "lucide-react";
import { styleCoverFallback } from "@yingce/lib/style-cover";
import { cn } from "../../lib/utils";
import { canvasOverlayHost } from "./canvas-host/design-system/popup-container";
import { dramaStyleCover, PRESET_STYLE_GROUP_IDS, presetStyleGroup, styleLabel } from "./drama-lib";
import { dramaStylePoster } from "./drama-style-cover";
import "./drama-style-picker.css";

/** DramaStyleOption 是面板里的一条风格值和名字。 */
export type DramaStyleOption = { value: string; name: string };

/** 预设封面没有时用备用图。 */
function coverSrc(value: string, label = "") {
  return dramaStyleCover(value, label) || styleCoverFallback(value || "cover");
}

/** 在列表框里把焦点移到相邻选项，到头就绕回。 */
function moveOptionFocus(from: HTMLElement, delta: number) {
  const options = Array.from(from.closest("[role='listbox']")?.querySelectorAll<HTMLElement>("[role='option']") || []);
  if (!options.length) return;
  const index = options.indexOf(from);
  const next = options[(index + delta + options.length) % options.length];
  next?.focus();
}

type Box = { top: number; left: number; width: number; maxHeight: number };

/** 读取当前可视区域，没有 visualViewport 就用窗口。 */
function viewRect() {
  const view = window.visualViewport;
  return {
    top: view?.offsetTop ?? 0,
    left: view?.offsetLeft ?? 0,
    width: view?.width ?? window.innerWidth,
    height: view?.height ?? window.innerHeight,
  };
}

/** 把面板放在触发器上方或下方，并限制在可视区域内。 */
function placeBox(trigger: HTMLElement): Box {
  const rect = trigger.getBoundingClientRect();
  const view = viewRect();
  const pad = 16;
  const gap = 8;
  const chrome = 20;
  const width = Math.min(680, Math.max(320, view.width - pad * 2));
  const minTop = view.top + pad;
  const maxBottom = view.top + view.height - pad - chrome;
  const spaceAbove = rect.top - gap - minTop;
  const spaceBelow = maxBottom - (rect.bottom + gap);
  const preferAbove = spaceAbove >= 200 || spaceAbove >= spaceBelow;
  let maxHeight = Math.max(140, Math.min(560, preferAbove ? spaceAbove : spaceBelow));
  let top = preferAbove ? rect.top - gap - maxHeight : rect.bottom + gap;
  if (top < minTop) top = minTop;
  if (top + maxHeight > maxBottom) top = Math.max(minTop, maxBottom - maxHeight);
  maxHeight = Math.max(140, Math.min(maxHeight, maxBottom - top));
  let left = rect.left;
  if (left + width > view.left + view.width - pad) left = view.left + view.width - pad - width;
  if (left < view.left + pad) left = view.left + pad;
  return { top, left, width, maxHeight };
}

/** 有文档时把浮层挂到 body。 */
function overlayHost() {
  if (typeof document === "undefined") return canvasOverlayHost();
  return document.body;
}

/** DramaStylePicker 打开风格网格。封面加载失败时换成 SVG 海报。 */
export function DramaStylePicker(props: {
  value: string;
  styles: DramaStyleOption[];
  names?: object;
  label: string;
  title: string;
  search: string;
  empty: string;
  groups?: Record<string, string>;
  onChange: (value: string) => void;
  disabled?: boolean;
  placement?: "topLeft" | "bottomLeft";
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [group, setGroup] = useState("all");
  const [box, setBox] = useState<Box | null>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const current = props.value || props.styles[0]?.value || "3d";
  const currentName = styleLabel(props.names, current, props.styles.find((row) => row.value === current)?.name || current);
  const groupTabs = useMemo(() => {
    const labels = props.groups || {};
    return [
      { id: "all", label: labels.all || "All" },
      ...PRESET_STYLE_GROUP_IDS.map((id) => ({ id, label: labels[id] || id })).filter((tab) => props.styles.some((row) => presetStyleGroup(row.value) === tab.id)),
    ];
  }, [props.groups, props.styles]);
  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return props.styles.filter((row) => {
      if (group !== "all" && presetStyleGroup(row.value) !== group) return false;
      if (!needle) return true;
      const name = styleLabel(props.names, row.value, row.name).toLowerCase();
      return name.includes(needle) || row.value.toLowerCase().includes(needle);
    });
  }, [group, props.names, props.styles, query]);

  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return;
    /** 按触发器重新摆放面板。 */
    const update = () => setBox(placeBox(triggerRef.current!));
    update();
    window.addEventListener("resize", update);
    window.addEventListener("scroll", update, true);
    window.visualViewport?.addEventListener("resize", update);
    window.visualViewport?.addEventListener("scroll", update);
    return () => {
      window.removeEventListener("resize", update);
      window.removeEventListener("scroll", update, true);
      window.visualViewport?.removeEventListener("resize", update);
      window.visualViewport?.removeEventListener("scroll", update);
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const frame = requestAnimationFrame(() => {
      panelRef.current?.querySelector<HTMLElement>("[aria-selected='true']")?.focus();
    });
    /** Escape 关闭面板并把焦点还给触发器。 */
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);

  /** 关闭面板并清空搜索和分组。 */
  const close = () => {
    setOpen(false);
    setQuery("");
    setGroup("all");
  };

  const panel = open && box ? (
    <div className="drama-style-layer" data-testid="drama-style-layer" onMouseDown={close}>
      <div
        ref={panelRef}
        className="drama-style-panel"
        data-testid="drama-style-grid"
        style={{ top: box.top, left: box.left, width: box.width, maxHeight: box.maxHeight }}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header>
          <strong>{props.title}</strong>
          <label className="drama-style-search">
            <Search className="size-3.5" aria-hidden />
            <input
              value={query}
              placeholder={props.search}
              aria-label={props.search}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
          {groupTabs.length > 2 ? (
            <div className="drama-style-groups" role="tablist" aria-label={props.title}>
              {groupTabs.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  role="tab"
                  aria-selected={group === tab.id}
                  className={cn(group === tab.id && "is-on")}
                  onClick={() => setGroup(tab.id)}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          ) : null}
        </header>
        {visible.length === 0 ? (
          <p className="drama-style-empty">{props.empty}</p>
        ) : (
          <div className="drama-style-grid" role="listbox" aria-label={props.label}>
            {visible.map((row) => {
              const selected = row.value === current;
              const name = styleLabel(props.names, row.value, row.name);
              return (
                <button
                  key={row.value}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  aria-label={name}
                  className={cn("drama-style-card", selected && "is-on")}
                  onClick={() => {
                    props.onChange(row.value);
                    close();
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "ArrowRight" || event.key === "ArrowDown") {
                      event.preventDefault();
                      moveOptionFocus(event.currentTarget, 1);
                    } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
                      event.preventDefault();
                      moveOptionFocus(event.currentTarget, -1);
                    }
                  }}
                >
                  <span className="drama-style-card-cover">
                    <img
                      src={coverSrc(row.value, name)}
                      alt=""
                      width={320}
                      height={240}
                      decoding="async"
                      loading="lazy"
                      draggable={false}
                      onError={(event) => {
                        event.currentTarget.src = dramaStylePoster(row.value, name);
                      }}
                    />
                    {selected ? (
                      <span className="drama-style-card-check" aria-hidden>
                        <Check className="size-3" strokeWidth={2.4} />
                      </span>
                    ) : null}
                  </span>
                  <span className="drama-style-card-name">{name}</span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  ) : null;

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        className={cn("sc-chip-btn drama-style-trigger", props.className)}
        aria-label={props.label}
        aria-haspopup="listbox"
        aria-expanded={open}
        data-testid="drama-style-picker"
        data-on={open ? "true" : undefined}
        disabled={props.disabled}
        onClick={() => (open ? close() : setOpen(true))}
      >
        <img
          className="drama-style-trigger-thumb"
          src={coverSrc(current, currentName)}
          alt=""
          width={20}
          height={20}
          draggable={false}
          onError={(event) => {
            event.currentTarget.src = dramaStylePoster(current, currentName);
          }}
        />
        <span className="drama-style-trigger-label">{currentName}</span>
        <ChevronDown className={cn("drama-style-trigger-chevron", open && "is-open")} aria-hidden />
      </button>
      {panel ? createPortal(panel, overlayHost()) : null}
    </>
  );
}
