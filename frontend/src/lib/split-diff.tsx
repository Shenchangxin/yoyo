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

function tone(kind: Kind, side: "left" | "right" | "uni") {
  if (kind === "add") return "bg-accent/10 text-accent";
  if (kind === "del") return "bg-danger/10 text-danger";
  if (kind === "hunk" || kind === "meta") return "text-muted";
  if (kind === "empty") return side === "left" ? "bg-danger/5" : "bg-accent/5";
  return "text-muted";
}

export function DiffBlock({ src, mode }: { src: string; mode: DiffMode }) {
  const rows = linesOf(src);
  if (mode === "unified") {
    return (
      <pre className="max-h-48 overflow-auto whitespace-pre-wrap font-mono text-[11px]">
        {rows.map((row, i) => (
          <span key={i} className={cn("block", tone(row.kind, "uni"))}>
            {row.kind === "add" ? "+" : row.kind === "del" ? "-" : ""}
            {row.text || " "}
          </span>
        ))}
      </pre>
    );
  }
  const pairs = pair(rows);
  return (
    <div className="grid max-h-48 grid-cols-2 gap-px overflow-auto rounded-lg bg-border font-mono text-[11px]">
      {pairs.map((p, i) => (
        <div key={i} className="contents">
          <div className={cn("whitespace-pre-wrap bg-panel px-2 py-0.5", tone(p.left.kind, "left"))}>
            {p.left.text || " "}
          </div>
          <div className={cn("whitespace-pre-wrap bg-panel px-2 py-0.5", tone(p.right.kind, "right"))}>
            {p.right.text || " "}
          </div>
        </div>
      ))}
    </div>
  );
}
