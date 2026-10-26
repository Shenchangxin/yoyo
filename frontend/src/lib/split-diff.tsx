import { cn } from "./utils";

/**
 * 拥有：把 unified diff 画成单栏或左右分栏，并按行数折叠聊天里的长差异。
 * 不拥有：差异文本的生成，以及审阅窗格的布局。
 */

/** DiffMode 单栏或左右分栏。 */
export type DiffMode = "unified" | "split";

type Kind = "meta" | "hunk" | "ctx" | "del" | "add" | "empty";

type Line = { kind: Kind; text: string };

/** 把 diff 文本收成带种类的行。元信息、块头、增删和上下文分开。 */
function linesOf(src: string): Line[] {
  return (src || "").split("\n").map((line) => {
    if (
      line.startsWith("diff ") ||
      line.startsWith("index ") ||
      line.startsWith("--- ") ||
      line.startsWith("+++ ")
    ) {
      return { kind: "meta", text: line };
    }
    if (line.startsWith("@@")) return { kind: "hunk", text: line };
    if (line.startsWith("+")) return { kind: "add", text: line.slice(1) };
    if (line.startsWith("-")) return { kind: "del", text: line.slice(1) };
    return { kind: "ctx", text: line.startsWith(" ") ? line.slice(1) : line };
  });
}

/** 连续的删除和新增配成左右两列。单独的新增左边留空，其余两边同一行。 */
function pair(rows: Line[]): { left: Line; right: Line }[] {
  const out: { left: Line; right: Line }[] = [];
  let i = 0;
  while (i < rows.length) {
    const row = rows[i];
    if (row.kind === "del") {
      const dels: Line[] = [];
      while (i < rows.length && rows[i].kind === "del") dels.push(rows[i++]);
      const adds: Line[] = [];
      while (i < rows.length && rows[i].kind === "add") adds.push(rows[i++]);
      const n = Math.max(dels.length, adds.length);
      for (let j = 0; j < n; j++) {
        out.push({
          left: dels[j] || { kind: "empty", text: "" },
          right: adds[j] || { kind: "empty", text: "" },
        });
      }
      continue;
    }
    if (row.kind === "add") {
      out.push({ left: { kind: "empty", text: "" }, right: row });
      i++;
      continue;
    }
    out.push({ left: row, right: row });
    i++;
  }
  return out;
}

/**
 * 行底用浅色，正文保持普通前景色。颜色只在背景和行首标记上，代码本身不上色。
 * 整行染成彩色会让长差异显得吵。
 */
function rowTone(kind: Kind, side: "left" | "right" | "uni"): string {
  if (kind === "add") return "bg-success/[0.11] text-foreground/90";
  if (kind === "del") return "bg-danger/[0.11] text-foreground/80";
  if (kind === "hunk" || kind === "meta") return "text-muted/60";
  if (kind === "empty") return side === "left" ? "bg-danger/[0.05]" : "bg-success/[0.05]";
  return "text-muted";
}

/** 行首标记：新增为加号，删除为减号。 */
function marker(kind: Kind): { glyph: string; tone: string } {
  if (kind === "add") return { glyph: "+", tone: "text-success" };
  if (kind === "del") return { glyph: "−", tone: "text-danger" };
  return { glyph: "", tone: "" };
}

const LINE = "diff-line grid min-w-0 grid-cols-[14px_minmax(0,1fr)] items-baseline gap-1.5 whitespace-pre-wrap break-words px-2 py-px";

/** CHAT_DIFF_MAX_LINES 聊天里改动气泡的行数上限，与 harness 的 CHAT_DIFF_MAX_LINES 一致。审阅窗格传入 0 表示不折叠。 */
export const CHAT_DIFF_MAX_LINES = 9;

/** 正文超过上限时保留头尾，中间用省略行隔开。元信息和块头始终保留。上限不大于零时不折叠。 */
function foldRows(rows: Line[], maxLines: number): Line[] {
  if (maxLines <= 0 || rows.length <= maxLines) return rows;
  const body = rows.filter((r) => r.kind === "add" || r.kind === "del" || r.kind === "ctx");
  if (body.length <= maxLines) return rows;
  const head = Math.ceil(maxLines / 2);
  const tail = maxLines - head;
  const keep = new Set<Line>([...body.slice(0, head), ...body.slice(body.length - tail)]);
  const out: Line[] = [];
  let gap = false;
  for (const row of rows) {
    if (row.kind === "meta" || row.kind === "hunk") {
      out.push(row);
      continue;
    }
    if (keep.has(row)) {
      out.push(row);
      gap = false;
      continue;
    }
    if (!gap) {
      out.push({ kind: "hunk", text: "⋯" });
      gap = true;
    }
  }
  return out;
}

/** DiffBlock 渲染一段 diff。mode 决定单栏或分栏。点到非空行时回调原文。 */
export function DiffBlock({
  src,
  mode,
  onLineClick,
  className,
  maxLines = 0,
}: {
  src: string;
  mode: DiffMode;
  onLineClick?: (text: string) => void;
  className?: string;
  maxLines?: number;
}) {
  const rows = foldRows(linesOf(src), maxLines);
  const click = (text: string) => {
    if (!onLineClick || !text.trim()) return;
    onLineClick(text);
  };
  const clickable = onLineClick ? "cursor-pointer hover:brightness-110" : "";
  const wrap = cn("max-h-72 overflow-auto font-mono text-[12px] leading-[1.6]", className);
  if (mode === "unified") {
    return (
      <div className={wrap} style={{ tabSize: 2 }}>
        {rows.map((row, i) => {
          const m = marker(row.kind);
          return (
            <div
              key={i}
              className={cn(LINE, rowTone(row.kind, "uni"), clickable)}
              onClick={() => click(row.text)}
            >
              <span className={cn("select-none text-right", m.tone)} aria-hidden>{m.glyph}</span>
              <span>{row.text || " "}</span>
            </div>
          );
        })}
      </div>
    );
  }
  const pairs = pair(rows);
  return (
    <div className={wrap} style={{ tabSize: 2 }}>
      {pairs.map((p, i) => (
        <div key={i} className="grid grid-cols-2 divide-x divide-border/50">
          <div
            className={cn(LINE, rowTone(p.left.kind, "left"), clickable)}
            onClick={() => click(p.left.text)}
          >
            <span className={cn("select-none text-right", marker(p.left.kind).tone)} aria-hidden>{marker(p.left.kind).glyph}</span>
            <span>{p.left.text || " "}</span>
          </div>
          <div
            className={cn(LINE, rowTone(p.right.kind, "right"), clickable)}
            onClick={() => click(p.right.text)}
          >
            <span className={cn("select-none text-right", marker(p.right.kind).tone)} aria-hidden>{marker(p.right.kind).glyph}</span>
            <span>{p.right.text || " "}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
