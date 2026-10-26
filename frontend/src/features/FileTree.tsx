// 本文件拥有把文件路径收成树、从差异里读增删改，以及这棵树的展开。不拥有文件列表怎么来。调用方传入 nodes 和 onSelect。

import { useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight, FileText, Folder, FolderOpen } from "lucide-react";
import { cn } from "../lib/utils";
import { readLocalMap, writeLocalMap } from "../lib/local-map";
import type { FileHit } from "../lib/protocol";

// FileMark 是差异标记：M 修改，A 新增，D 删除。
export type FileMark = "M" | "A" | "D";

// TreeNode 是一层目录或文件。目录的 children 装着下一层。
export type TreeNode = {
  name: string;
  path: string;
  kind: "dir" | "file";
  children: TreeNode[];
};

/** nestHits 按路径排序后去重。目录进目录表，文件挂到父目录；没有斜杠的放在根上。同层目录排在文件前。hits 展开传递。 */
export function nestHits(hits: FileHit[]): TreeNode[] {
  const dirs = new Map<string, TreeNode>();
  const root: TreeNode[] = [];
  /** ensureDir 目录已在表里就返回它。否则挂到父目录，路径里没有斜杠就放进根。path 按 / 切开，调用 pop，调用 lastIndexOf，截取，作为后备值，传给 dirs.get，交给 dirs.set，写入字段 path。 */
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
  /** sortNodes 同层先目录后文件，名字按 localeCompare，并递归子层。nodes 调用 sort，逐项处理。 */
  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => (a.kind === b.kind ? a.name.localeCompare(b.name) : a.kind === "dir" ? -1 : 1));
    nodes.forEach((n) => sortNodes(n.children));
  };
  sortNodes(root);
  return root;
}

// fileMarks 空路径和 dev/null 不计。已经是 A 或 D 的不再改。new file mode 记 A，deleted file mode 记 D，diff --git 记 M。+++ b/ 才写入路径，随后 pending 回到 M。files 里还没出现的记成 M。
export function fileMarks(diff: string, files: string[]): Record<string, FileMark> {
  const out: Record<string, FileMark> = {};
  /** set 空路径和 dev/null 跳过。已经记成 A 或 D 的不再改。p 传给 normPath。m 赋给 out[n]。 */
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

/** ancestorPaths 返回每一层父路径，不含自己。path 传给 normPath。 */
export function ancestorPaths(path: string): string[] {
  const parts = normPath(path).split("/").filter(Boolean);
  const out: string[] = [];
  for (let i = 1; i < parts.length; i++) out.push(parts.slice(0, i).join("/"));
  return out;
}

/**
 * FileTree 没有节点时不渲染。持久化键是 yoyo-file-tree: 加 persistKey。还没有存过时，顶层目录默认展开。expandPaths 里的路径强制展开。
 * props 读取 persistKey、expandPaths、nodes、selected、marks、onSelect。
 */
export function FileTree(props: {
  nodes: TreeNode[];
  selected?: string;
  marks: Record<string, FileMark>;
  expandPaths?: string[];
  persistKey?: string;
  onSelect: (path: string, kind: "dir" | "file") => void;
}) {
  const persistKey = props.persistKey ? `yoyo-file-tree:${props.persistKey}` : "";
  const expandKey = (props.expandPaths || []).join("\0");
  const openRef = useRef<Record<string, boolean>>({});
  const [open, setOpen] = useState<Record<string, boolean>>({});

  useEffect(() => {
    const saved = readLocalMap(persistKey);
    const next = { ...saved };
    const seeded = Object.keys(saved).length > 0;
    for (const n of props.nodes) {
      if (n.kind === "dir" && next[n.path] === undefined) next[n.path] = !seeded;
    }
    for (const p of expandKey.split("\0")) if (p) next[p] = true;
    openRef.current = next;
    setOpen(next);
    writeLocalMap(persistKey, next);
  }, [persistKey, props.nodes, expandKey]);

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
          onToggle={(path) => {
            const next = { ...openRef.current, [path]: !openRef.current[path] };
            openRef.current = next;
            setOpen(next);
            writeLocalMap(persistKey, next);
          }}
          onSelect={props.onSelect}
        />
      ))}
    </ul>
  );
}

/** TreeRow 点目录只切换展开，点文件才 onSelect。左内边距是 8 加深度乘 12。自己没标记但子孙有标记时只画一个点。props 读取 node、open、marks、selected、depth，调用 onToggle，调用 onSelect，展开到 TreeRow。 */
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

/** childMark 自己或任一子孙带标记就返回真。n 读取 path、children。marks 按下标读取，传给 childMark。 */
function childMark(n: TreeNode, marks: Record<string, FileMark>): boolean {
  if (marks[n.path]) return true;
  return n.children.some((c) => childMark(c, marks));
}

/** markClass A 用绿色，D 用红色，M 用琥珀色，没有标记返回空字符串。mark 为 A 时返回 text-emerald-500，为 D 时返回 text-red-400，为 M 时返回 text-amber-500，否则返回 空字符串。 */
function markClass(mark?: FileMark): string {
  if (mark === "A") return "text-emerald-500";
  if (mark === "D") return "text-red-400";
  if (mark === "M") return "text-amber-500";
  return "";
}

/** markLabel A 读成 added，D 读成 deleted，其余读成 modified。mark 为 A 时返回 added，为 D 时返回 deleted，否则返回 modified。 */
function markLabel(mark: FileMark): string {
  if (mark === "A") return "added";
  if (mark === "D") return "deleted";
  return "modified";
}

/** normPath 把反斜杠换成斜杠，并去掉开头的 ./。p 为假时用空字符串，做替换。 */
export function normPath(p: string): string {
  return (p || "").replace(/\\/g, "/").replace(/^\.\//, "");
}
