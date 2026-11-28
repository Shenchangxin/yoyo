/**
 * 拥有：对话里关于记忆的两种回执：这一轮带进了哪些记忆（可以说「不对」或「别再提」），以及这一轮记下或改口的（可以撤销）。
 * 不拥有：记什么、带什么进对话（Go 的 memory 包和 app/memory_seen.go）。回执写在会话里，回放历史时不进模型。
 */

import { useState } from "react";
import { memoryForget, memoryMute, memoryRevert } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { fill } from "../buddy/buddy-model";

type MemRow = { id: string; kind: string; text: string; day: string; action: string; before: string };

function memRows(value: unknown): MemRow[] {
  if (!Array.isArray(value)) return [];
  return value
    .map((raw) => {
      const r = (raw && typeof raw === "object" ? raw : {}) as Record<string, unknown>;
      return {
        id: String(r.id || ""),
        kind: String(r.kind || ""),
        text: String(r.text || ""),
        day: String(r.day || ""),
        action: String(r.action || ""),
        before: String(r.before || ""),
      };
    })
    .filter((r) => r.id && r.text);
}

const ACTION = "h-6 shrink-0 rounded-md px-2 text-ui-caption font-medium text-muted transition-colors duration-[var(--duration-fast)] hover:bg-lift hover:text-foreground disabled:opacity-50";

/** MemoryReceipt 按载荷的 kind 画成「记下了」或「带进了哪些」。认不出时不画。 */
export function MemoryReceipt({ item }: { item: Item }) {
  const kind = String(item.payload?.kind || "");
  if (kind === "wrote") return <WroteReceipt items={memRows(item.payload?.items)} />;
  if (kind === "used") return <UsedReceipt recalled={memRows(item.payload?.recalled)} pinned={memRows(item.payload?.pinned)} />;
  return null;
}

/** WroteReceipt 是这一轮记下或改口的几条。新记下的撤销就是忘掉；改口的撤销是退回旧说法。 */
function WroteReceipt({ items }: { items: MemRow[] }) {
  const t = useCopy().transcript;
  const [done, setDone] = useState<Record<string, boolean>>({});
  const [busy, setBusy] = useState("");
  if (!items.length) return null;
  async function undo(row: MemRow) {
    if (busy) return;
    setBusy(row.id);
    try {
      const ok = row.action === "replaced"
        ? !!(await memoryRevert(row.id).catch(() => ({ id: "" }))).id
        : await memoryForget(row.id).then(() => true).catch(() => false);
      if (ok) setDone((cur) => ({ ...cur, [row.id]: true }));
    } finally {
      setBusy("");
    }
  }
  return (
    <div className="space-y-1 py-1" data-testid="memory-wrote">
      {items.map((row) => (
        <div key={`${row.id}:${row.text}`} className="flex min-w-0 items-center gap-2 text-ui-caption text-muted">
          <span className="min-w-0 flex-1 truncate" title={row.text}>
            {row.action === "replaced" ? fill(t.memoryReplaced, { before: row.before, text: row.text }) : fill(t.memoryWrote, { text: row.text })}
          </span>
          {done[row.id] ? (
            <span className="shrink-0">{t.memoryUndone}</span>
          ) : (
            <button type="button" className={ACTION} disabled={busy === row.id} onClick={() => { void undo(row); }}>
              {t.memoryUndo}
            </button>
          )}
        </div>
      ))}
    </div>
  );
}

/** UsedReceipt 是这一轮带进对话的记忆。默认收成一行，点开看每一条；「不对」忘掉，「别再提」只是不再带进对话。 */
function UsedReceipt({ recalled, pinned }: { recalled: MemRow[]; pinned: MemRow[] }) {
  const t = useCopy().transcript;
  const [open, setOpen] = useState(false);
  const [state, setState] = useState<Record<string, "wrong" | "mute">>({});
  const [busy, setBusy] = useState("");
  const total = recalled.length + pinned.length;
  if (!total) return null;
  async function act(row: MemRow, kind: "wrong" | "mute") {
    if (busy) return;
    setBusy(row.id);
    try {
      const ok = kind === "wrong"
        ? await memoryForget(row.id).then(() => true).catch(() => false)
        : await memoryMute(row.id, true).catch(() => false);
      if (ok) setState((cur) => ({ ...cur, [row.id]: kind }));
    } finally {
      setBusy("");
    }
  }
  const group = (label: string, list: MemRow[]) => (list.length ? (
    <div className="mt-1.5">
      <p className="text-ui-caption text-muted">{label}</p>
      {list.map((row) => (
        <div key={`${label}:${row.id}`} className="mt-0.5 flex min-w-0 items-center gap-2 text-ui-footnote">
          <span className="min-w-0 flex-1 truncate" title={row.day ? `${row.day} · ${row.text}` : row.text}>{row.text}</span>
          {state[row.id] ? (
            <span className="shrink-0 text-ui-caption text-muted">{state[row.id] === "wrong" ? t.memoryWrongDone : t.memoryMuteDone}</span>
          ) : (
            <span className="flex shrink-0 items-center gap-0.5">
              <button type="button" className={ACTION} disabled={busy === row.id} onClick={() => { void act(row, "wrong"); }}>{t.memoryWrong}</button>
              <button type="button" className={ACTION} disabled={busy === row.id} onClick={() => { void act(row, "mute"); }}>{t.memoryMute}</button>
            </span>
          )}
        </div>
      ))}
    </div>
  ) : null);
  return (
    <div className="py-1" data-testid="memory-used">
      <div className="flex items-center gap-3 text-ui-caption text-muted">
        <span className="h-px flex-1 bg-border/70" aria-hidden />
        <button type="button" className="hover:text-foreground" aria-expanded={open} onClick={() => setOpen((v) => !v)}>
          {fill(t.memoryUsed, { n: String(total) })}
        </button>
        <span className="h-px flex-1 bg-border/70" aria-hidden />
      </div>
      {open ? (
        <div className="mx-auto max-w-[40rem] px-1">
          {group(t.memoryRecalled, recalled)}
          {group(t.memoryPinned, pinned)}
        </div>
      ) : null}
    </div>
  );
}
