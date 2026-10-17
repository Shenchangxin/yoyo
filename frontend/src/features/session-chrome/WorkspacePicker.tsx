/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Path-based pick menu internalized from harness WorkspacePickFlow.
 * Empty recents skip the menu — the owner opens the native folder picker.
 */
import { useEffect, useLayoutEffect, useRef, useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { Check, Folder, Plus } from "lucide-react";
import type { Copy } from "../../lib/copy";
import { workspaceLabel } from "./WorkspaceChip";

export function WorkspacePicker(props: {
  open: boolean;
  anchorRef: RefObject<HTMLElement | null>;
  items: string[];
  selected?: string;
  copy: Copy;
  onPick: (path: string) => void;
  onAdd: () => void;
  onClose: () => void;
}) {
  const panelRef = useRef<HTMLDivElement>(null);
  const onAddRef = useRef(props.onAdd);
  const onCloseRef = useRef(props.onClose);
  onAddRef.current = props.onAdd;
  onCloseRef.current = props.onClose;
  const [pos, setPos] = useState<{ top: number; left: number; width: number } | null>(null);
  const recents = props.items.filter(Boolean);
  const skipMenu = recents.length === 0;

  useEffect(() => {
    if (!props.open || !skipMenu) return;
    onCloseRef.current();
    onAddRef.current();
  }, [props.open, skipMenu]);

  useLayoutEffect(() => {
    if (!props.open || skipMenu || !props.anchorRef.current) {
      setPos(null);
      return;
    }
    const r = props.anchorRef.current.getBoundingClientRect();
    const width = Math.max(220, Math.min(320, r.width + 80));
    const left = Math.min(Math.max(12, r.left), window.innerWidth - width - 12);
    setPos({ top: r.bottom + 6, left, width });
  }, [props.open, skipMenu, props.anchorRef, recents.length]);

  useEffect(() => {
    if (!props.open || skipMenu) return;
    const onDoc = (e: PointerEvent) => {
      const t = e.target as Node;
      if (props.anchorRef.current?.contains(t) || panelRef.current?.contains(t)) return;
      onCloseRef.current();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCloseRef.current();
    };
    document.addEventListener("pointerdown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [props.open, skipMenu, props.anchorRef]);

  if (!props.open || skipMenu || !pos) return null;

  return createPortal(
    <div
      ref={panelRef}
      className="sc-ws-menu"
      role="menu"
      aria-label={props.copy.composer.workspace}
      style={{ top: pos.top, left: pos.left, width: pos.width }}
    >
      {recents.map((path) => (
        <button
          key={path}
          type="button"
          role="menuitem"
          className="sc-ws-item"
          data-selected={path === props.selected ? "true" : undefined}
          onClick={() => {
            props.onPick(path);
            props.onClose();
          }}
        >
          <Folder size={16} strokeWidth={1} className="shrink-0 opacity-80" />
          <span className="min-w-0 flex-1 truncate text-left">
            <span className="block truncate">{workspaceLabel(path)}</span>
            <span className="block truncate text-[11px] font-normal text-muted">{path}</span>
          </span>
          {path === props.selected ? <Check size={14} strokeWidth={1} className="shrink-0" /> : null}
        </button>
      ))}
      <div className="sc-ws-foot">
        <button
          type="button"
          role="menuitem"
          className="sc-ws-item"
          onClick={() => {
            props.onClose();
            props.onAdd();
          }}
        >
          <Plus size={16} strokeWidth={1} className="shrink-0 opacity-80" />
          <span className="min-w-0 flex-1 truncate text-left">{props.copy.composer.addWorkspace}</span>
        </button>
      </div>
    </div>,
    document.body,
  );
}
