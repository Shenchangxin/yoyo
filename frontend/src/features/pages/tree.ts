export type PageMeta = {
  id: string;
  space_id?: string;
  parent_id?: string;
  title: string;
  revision: number;
  updated_at?: string;
  preview?: string;
};

export type PageNode = PageMeta & { children: PageNode[] };

export function buildPageTree(list: PageMeta[]): PageNode[] {
  const map = new Map<string, PageNode>();
  for (const m of list) {
    if (!m?.id || map.has(m.id)) continue;
    map.set(m.id, { ...m, children: [] });
  }
  const roots: PageNode[] = [];
  for (const n of map.values()) {
    const parentId = n.parent_id || "";
    const p = parentId && parentId !== n.id ? map.get(parentId) : undefined;
    if (p && !wouldCycle(map, n.id, parentId)) p.children.push(n);
    else roots.push(n);
  }
  return roots;
}

function wouldCycle(map: Map<string, PageNode>, childId: string, parentId: string): boolean {
  let cur: string | undefined = parentId;
  const seen = new Set<string>();
  while (cur) {
    if (cur === childId) return true;
    if (seen.has(cur)) return true;
    seen.add(cur);
    cur = map.get(cur)?.parent_id;
  }
  return false;
}
