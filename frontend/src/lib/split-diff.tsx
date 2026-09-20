import { cn } from "./utils";

export type DiffMode = "unified" | "split";

type Kind = "meta" | "hunk" | "ctx" | "del" | "add" | "empty";

type Line = { kind: Kind; text: string };

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
 * Tinted rows, plain foreground text. Colour lives in the background and the
 * gutter marker, never in the code itself — long coloured lines are what made
 * the old panel feel loud.
 */
function rowTone(kind: Kind, side: "left" | "right" | "uni"): string {
  if (kind === "add") return "bg-success/[0.11] text-foreground/90";
  if (kind === "del") return "bg-danger/[0.11] text-foreground/80";
  if (kind === "hunk" || kind === "meta") return "text-muted/60";
  if (kind === "empty") return side === "left" ? "bg-danger/[0.05]" : "bg-success/[0.05]";
  return "text-muted";
}

function marker(kind: Kind): { glyph: string; tone: string } {
  if (kind === "add") return { glyph: "+", tone: "text-success" };
  if (kind === "del") return { glyph: "−", tone: "text-danger" };
  return { glyph: "", tone: "" };
}

const LINE = "diff-line grid min-w-0 grid-cols-[14px_minmax(0,1fr)] items-baseline gap-1.5 whitespace-pre-wrap break-words px-2 py-px";

export function DiffBlock({
  src,
  mode,
  onLineClick,
  className,
}: {
  src: string;
  mode: DiffMode;
  onLineClick?: (text: string) => void;
  className?: string;
}) {
  const rows = linesOf(src);
  const click = (text: string) => {
    if (!onLineClick || !text.trim()) return;
    onLineClick(text);
  };
  const clickable = onLineClick ? "cursor-pointer hover:brightness-110" : "";
  const wrap = cn("max-h-72 overflow-auto font-mono text-[11.5px] leading-[1.6]", className);
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
