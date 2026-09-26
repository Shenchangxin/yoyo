import { useEffect, useState } from "react";
import { ChevronDown, ChevronRight, FileText, Folder, FolderOpen } from "lucide-react";
import { cn } from "../lib/utils";
import type { FileHit } from "../lib/protocol";

export type FileMark = "M" | "A" | "D";

export type TreeNode = {
  name: string;
  path: string;
  kind: "dir" | "file";
  children: TreeNode[];
};

export function nestHits(hits: FileHit[]): TreeNode[] {
  const dirs = new Map<string, TreeNode>();
  const root: TreeNode[] = [];
  const ensureDir = (path: string): TreeNode => {
    const existing = dirs.get(path);
    if (existing) return existing;
    const name = path.split("/").pop() || path;
    const node: TreeNode = { name, path, kind: "dir", children: [] };
    dirs.set(path, node);
    const i = path.lastIndexOf("/");
    if (i < 0) root.push(node);
    else ensureDir(path.slice(0, i)).children.push(node);
    return node;
  };
  const sorted = [...hits].sort((a, b) => a.path.localeCompare(b.path));
  const seen = new Set<string>();
  for (const h of sorted) {
    const p = normPath(h.path);
    if (!p || seen.has(p)) continue;
    seen.add(p);
    if (h.kind === "dir") {
      ensureDir(p);
      continue;
    }
    const i = p.lastIndexOf("/");
    const name = i >= 0 ? p.slice(i + 1) : p;
    const node: TreeNode = { name, path: p, kind: "file", children: [] };
    if (i < 0) root.push(node);
    else ensureDir(p.slice(0, i)).children.push(node);
  }
  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => (a.kind === b.kind ? a.name.localeCompare(b.name) : a.kind === "dir" ? -1 : 1));
    nodes.forEach((n) => sortNodes(n.children));
  };
  sortNodes(root);
  return root;
}

export function fileMarks(diff: string, files: string[]): Record<string, FileMark> {
  const out: Record<string, FileMark> = {};
  const set = (p: string, m: FileMark) => {
    const n = normPath(p);
    if (!n || n === "dev/null") return;
    if (out[n] === "A" || out[n] === "D") return;
    out[n] = m;
  };
  let pending: FileMark = "M";
  for (const line of (diff || "").split("\n")) {
    if (line.startsWith("new file mode")) pending = "A";
    else if (line.startsWith("deleted file mode")) pending = "D";
    else if (line.startsWith("diff --git ")) pending = "M";
    else if (line.startsWith("--- ") && line.includes("/dev/null")) pending = "A";
    else if (line.startsWith("+++ ")) {
      if (line.includes("/dev/null")) pending = "D";
      else if (line.startsWith("+++ b/")) set(line.slice(6), pending);
      pending = "M";
    }
  }
  for (const f of files) {
    if (f && !out[normPath(f)]) set(f, "M");
  }
  return out;
}

export function ancestorPaths(path: string): string[] {
  const parts = normPath(path).split("/").filter(Boolean);
  const out: string[] = [];
  for (let i = 1; i < parts.length; i++) out.push(parts.slice(0, i).join("/"));
  return out;
}

export function FileTree(props: {
  nodes: TreeNode[];
  selected?: string;
  marks: Record<string, FileMark>;
  expandPaths?: string[];
  onSelect: (path: string, kind: "dir" | "file") => void;
}) {
  const expandKey = (props.expandPaths || []).join("\0");
  const [open, setOpen] = useState<Record<string, boolean>>({});
  useEffect(() => {
    setOpen((prev) => {
      const next = { ...prev };
      for (const n of props.nodes) {
        if (n.kind === "dir" && next[n.path] === undefined) next[n.path] = true;
      }
      for (const p of expandKey.split("\0")) if (p) next[p] = true;
      return next;
    });
  }, [props.nodes, expandKey]);

  if (!props.nodes.length) return null;
  return (
    <ul className="min-h-0 overflow-auto py-1" data-testid="review-file-tree" role="tree">
      {props.nodes.map((n) => (
        <TreeRow
          key={n.path}
          node={n}
          depth={0}
          selected={props.selected}
          marks={props.marks}
          open={open}
          onToggle={(path) => setOpen((s) => ({ ...s, [path]: !s[path] }))}
          onSelect={props.onSelect}
        />
      ))}
    </ul>
  );
}

function TreeRow(props: {
  node: TreeNode;
  depth: number;
  selected?: string;
  marks: Record<string, FileMark>;
  open: Record<string, boolean>;
  onToggle: (path: string) => void;
  onSelect: (path: string, kind: "dir" | "file") => void;
}) {
  const n = props.node;
  const expanded = n.kind === "dir" && !!props.open[n.path];
  const mark = props.marks[n.path];
  const dirty = mark || childMark(n, props.marks);
  const on = n.kind === "file" && normPath(props.selected || "") === n.path;
  const Icon = n.kind === "dir" ? (expanded ? FolderOpen : Folder) : FileText;
  return (
    <li role="treeitem" aria-expanded={n.kind === "dir" ? expanded : undefined} aria-selected={on || undefined}>
      <button
        type="button"
        title={n.path}
        data-testid={`file-tree-${n.path}`}
        className={cn(
          "flex h-7 w-full items-center gap-1 rounded-md pr-2 text-left text-[12px] transition-colors",
          on ? "bg-lift text-foreground" : "text-foreground/85 hover:bg-lift/50",
        )}
        style={{ paddingLeft: 8 + props.depth * 12 }}
        onClick={() => {
          if (n.kind === "dir") props.onToggle(n.path);
          else props.onSelect(n.path, "file");
        }}
      >
        {n.kind === "dir" ? (
          expanded ? <ChevronDown className="size-3 shrink-0 text-muted" aria-hidden /> : <ChevronRight className="size-3 shrink-0 text-muted" aria-hidden />
        ) : (
          <span className="w-3 shrink-0" aria-hidden />
        )}
        <Icon className={cn("size-3.5 shrink-0", markClass(mark) || "text-muted/70")} aria-hidden />
        <span className={cn("min-w-0 flex-1 truncate", markClass(mark))}>{n.name}</span>
        {mark ? (
          <span className={cn("shrink-0 font-mono text-[10px] font-semibold", markClass(mark))} aria-label={markLabel(mark)}>
            {mark}
          </span>
        ) : dirty ? (
          <span className="size-1.5 shrink-0 rounded-full bg-amber-400/80" aria-hidden />
        ) : null}
      </button>
      {n.kind === "dir" && expanded && n.children.length ? (
        <ul role="group">
          {n.children.map((c) => (
            <TreeRow key={c.path} {...props} node={c} depth={props.depth + 1} />
          ))}
        </ul>
      ) : null}
    </li>
  );
}

function childMark(n: TreeNode, marks: Record<string, FileMark>): boolean {
  if (marks[n.path]) return true;
  return n.children.some((c) => childMark(c, marks));
}

function markClass(mark?: FileMark): string {
  if (mark === "A") return "text-emerald-500";
  if (mark === "D") return "text-red-400";
  if (mark === "M") return "text-amber-500";
  return "";
}

function markLabel(mark: FileMark): string {
  if (mark === "A") return "added";
  if (mark === "D") return "deleted";
  return "modified";
}

export function normPath(p: string): string {
  return (p || "").replace(/\\/g, "/").replace(/^\.\//, "");
}
