/**
 * 拥有：上下文占用的环和分项。
 * 不拥有：草稿和会话。只投影用量，不调用 api。
 */
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { Copy } from "../../lib/copy";
import { contextBreakdown, type CtxBreakdown } from "../../lib/context-usage";
import { formatTokens } from "../../lib/models-dev";
import type { ContextUsage } from "../../lib/protocol";

const RADIUS = 5.5;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

const ROWS: { key: string; label: (c: Copy) => string; tone: string; always?: boolean }[] = [
  { key: "system", 
                   /** label 读取 ctxSystem。c 读取 composer。 */
                   label: (c) => c.composer.ctxSystem, tone: "var(--ctx-system, #9a958c)" },
  { key: "rules", 
                  /** label 读取 ctxRules。c 读取 composer。 */
                  label: (c) => c.composer.ctxRules, tone: "var(--ctx-rules, #c4a574)" },
  { key: "catalog", 
                    /** label 读取 ctxCatalog。c 读取 composer。 */
                    label: (c) => c.composer.ctxCatalog, tone: "var(--ctx-catalog, #8aa0b8)" },
  { key: "skills", 
                   /** label 读取 ctxSkillBody。c 读取 composer。 */
                   label: (c) => c.composer.ctxSkillBody, tone: "var(--ctx-skills, #7aa2ff)" },
  { key: "tools", 
                  /** label 读取 ctxTools。c 读取 composer。 */
                  label: (c) => c.composer.ctxTools, tone: "var(--ctx-tools, #a78bfa)" },
  { key: "memory", 
                   /** label 读取 ctxMemory。c 读取 composer。 */
                   label: (c) => c.composer.ctxMemory, tone: "var(--ctx-memory, #c2663d)" },
  { key: "today", 
                  /** label 读取 ctxToday。c 读取 composer。 */
                  label: (c) => c.composer.ctxToday, tone: "var(--ctx-today, #1f7a45)" },
  { key: "notes", 
                  /** label 读取 ctxNotes。c 读取 composer。 */
                  label: (c) => c.composer.ctxNotes, tone: "var(--ctx-notes, #a07a2a)" },
  { key: "chat", 
                 /** label 读取 ctxChat。c 读取 composer。 */
                 label: (c) => c.composer.ctxChat, tone: "var(--ctx-chat, #3b82f6)", always: true },
  { key: "free", 
                 /** label 读取 ctxFree。c 读取 composer。 */
                 label: (c) => c.composer.ctxFree, tone: "var(--ctx-free, transparent)", always: true },
];

/** 把用量拆开，对上环上每一行的键。br 读取 prefix、rules、catalog、skills、schema、memory 等。 */
function meterTokens(br: CtxBreakdown): Record<string, number> {
  return {
    system: br.prefix,
    rules: br.rules,
    catalog: br.catalog,
    skills: br.skills,
    tools: br.schema,
    memory: br.memory,
    today: br.today,
    notes: br.notes,
    chat: br.chat,
    free: br.free,
  };
}

/** ContextMeter 画出占用环，打开后列出各项。props 读取 copy、ctx、fallbackWindow、draftTokens。 */
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
    /** 点在环和面板外面就收起。e 读取 target。 */
    const onDoc = (e: PointerEvent) => {
      const t = e.target as Node;
      if (rootRef.current?.contains(t) || panelRef.current?.contains(t)) return;
      setOpen(false);
    };
    /** 按退出就收起。e 读取 key。 */
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
  const tokens = meterTokens(br);
  const parts = ROWS
    .filter((row) => row.key !== "free" && tokens[row.key] > 0)
    .map((row) => ({ key: row.key, width: br.capacity ? (tokens[row.key] / br.capacity) * 100 : 0, tone: row.tone }));

  const panel = open && pos ? createPortal(
    <div
      ref={panelRef}
      className="sc-meter-panel"
      style={{ left: pos.left, top: pos.top, transform: "translateY(-100%)" }}
      role="dialog"
      aria-label={copy.composer.context}
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
        {ROWS.filter((row) => row.always || tokens[row.key] > 0).map((row) => (
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
