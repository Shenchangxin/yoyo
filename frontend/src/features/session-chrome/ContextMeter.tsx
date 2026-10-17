/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized from deepseek-harness ContextMeter (ring + breakdown panel).
 */
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { Copy } from "../../lib/copy";
import { contextBreakdown } from "../../lib/context-usage";
import { formatTokens } from "../../lib/models-dev";
import type { ContextUsage } from "../../lib/protocol";

const RADIUS = 5.5;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

const ROWS = [
  { key: "system", label: (c: Copy) => c.composer.ctxSystem, tone: "var(--ctx-system, #60a5fa)" },
  { key: "tools", label: (c: Copy) => c.composer.ctxTools, tone: "rgb(167, 139, 250)" },
  { key: "messages", label: (c: Copy) => c.composer.ctxMessages, tone: "var(--ctx-chat, #3b82f6)" },
] as const;

export function ContextMeter(props: {
  ctx?: ContextUsage;
  draftTokens?: number;
  fallbackWindow: number;
  copy: Copy;
}) {
  const copy = props.copy;
  const br = contextBreakdown(props.ctx, props.fallbackWindow, props.draftTokens || 0);
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLSpanElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);

  useEffect(() => {
    if (!open || !triggerRef.current) return;
    const r = triggerRef.current.getBoundingClientRect();
    const left = Math.min(Math.max(12, r.left), window.innerWidth - 288);
    setPos({ top: r.top - 8, left });
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: PointerEvent) => {
      const t = e.target as Node;
      if (rootRef.current?.contains(t) || panelRef.current?.contains(t)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  if (br.capacity <= 0) return null;

  const offset = CIRCUMFERENCE * (1 - Math.min(1, br.used / br.capacity));
  const reading = `${br.pct}%`;
  const label = copy.composer.ctxUsed.replace("{pct}", String(br.pct));
  const parts = [
    { key: "system", width: br.capacity ? (br.prefix / br.capacity) * 100 : 0, tone: ROWS[0].tone },
    { key: "tools", width: br.capacity ? (br.schema / br.capacity) * 100 : 0, tone: ROWS[1].tone },
    { key: "messages", width: br.capacity ? ((br.chat + br.dynamic) / br.capacity) * 100 : 0, tone: ROWS[2].tone },
  ].filter((p) => p.width > 0);
  const tokens = {
    system: br.prefix,
    tools: br.schema,
    messages: br.chat + br.dynamic,
  };

  const panel = open && pos ? createPortal(
    <div
      ref={panelRef}
      className="sc-meter-panel"
      style={{ left: pos.left, top: pos.top, transform: "translateY(-100%)" }}
      role="dialog"
      aria-label={copy.composer.ctxUsed.replace("{pct}", reading)}
    >
      <div className="sc-meter-head">
        <span className="sc-meter-percent">{reading}</span>
        <span className="sc-meter-headline">{copy.composer.context}</span>
        <span className="sc-meter-figures">~{formatTokens(br.used)} / {formatTokens(br.capacity)}</span>
      </div>
      <div className="sc-meter-bar">
        {parts.length ? parts.map((p) => (
          <div key={p.key} className="sc-meter-seg" style={{ width: `${p.width}%`, background: p.tone }} />
        )) : <div className="sc-meter-seg" style={{ width: `${br.pct}%` }} />}
      </div>
      <dl className="sc-meter-rows">
        {ROWS.map((row) => (
          <div key={row.key} className="sc-meter-row">
            <dt>
              <span className="sc-meter-swatch" style={{ background: row.tone }} aria-hidden />
              {row.label(copy)}
            </dt>
            <dd>~{formatTokens(tokens[row.key])}</dd>
          </div>
        ))}
      </dl>
    </div>,
    document.body,
  ) : null;

  return (
    <span ref={rootRef} className="sc-meter-root" data-testid="composer-context-meter">
      <button
        ref={triggerRef}
        type="button"
        className="sc-meter"
        aria-expanded={open}
        aria-label={label}
        onClick={() => setOpen((v) => !v)}
      >
        <svg width="14" height="14" viewBox="0 0 14 14" aria-hidden>
          <circle className="sc-meter-track" cx="7" cy="7" r={RADIUS} />
          <circle
            className="sc-meter-fill"
            cx="7"
            cy="7"
            r={RADIUS}
            strokeDasharray={CIRCUMFERENCE}
            strokeDashoffset={offset}
            transform="rotate(-90 7 7)"
          />
        </svg>
        <span>{reading}</span>
      </button>
      {panel}
    </span>
  );
}
