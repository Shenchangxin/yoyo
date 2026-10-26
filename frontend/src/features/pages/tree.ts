/**
 * 拥有：页面列表收成树的规则。
 * 不拥有：编辑器。只投影列表，不改服务端。
 */

/** PageMeta 一页的编号、父级、标题、修订和预览。 */
export type PageMeta = {
  id: string;
  space_id?: string;
  parent_id?: string;
  title: string;
  revision: number;
  updated_at?: string;
  preview?: string;
};

/** PageNode 是带孩子的页面节点。 */
export type PageNode = PageMeta & { children: PageNode[] };

/** buildPageTree 按父级把页面收成树，成环或自己指向自己的留在根上。 */
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

/** 顺着父级往上走，撞到孩子或重复就当成成环。 */
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
