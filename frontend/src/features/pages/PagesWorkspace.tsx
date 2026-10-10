import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FileText, Plus, Search } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { EmptyState } from "../../components/ui/empty-state";
import { PageEditor, type SaveState } from "./PageEditor";

type PageMeta = {
  id: string;
  space_id?: string;
  parent_id?: string;
  title: string;
  revision: number;
  updated_at?: string;
};

type Page = PageMeta & { content: string };

export function PagesWorkspace(props: {
  sessionId?: string;
  onOpenInChat?: (pageId: string) => void;
}) {
  const copy = useCopy();
  const [q, setQ] = useState("");
  const [list, setList] = useState<PageMeta[]>([]);
  const [active, setActive] = useState<Page | null>(null);
  const [state, setState] = useState<SaveState>("saved");
  const [err, setErr] = useState("");
  const timer = useRef<number>(0);
  const draft = useRef<{ title: string; content: string }>({ title: "", content: "" });

  const loadList = useCallback(async (query = q) => {
    const rows = await api.pagesList(query).catch(() => []);
    setList(Array.isArray(rows) ? rows : []);
  }, [q]);

  useEffect(() => { void loadList(""); }, [loadList]);

  const open = async (id: string) => {
    const pg = await api.pagesGet(id).catch(() => null);
    if (!pg?.id) return;
    setActive(pg);
    draft.current = { title: pg.title, content: pg.content || "" };
    setState("saved");
    setErr("");
    props.onOpenInChat?.(pg.id);
  };

  const persist = useCallback(async () => {
    if (!active) return;
    setState("saving");
    try {
      const saved = await api.pagesSave({
        ...active,
        title: draft.current.title,
        content: draft.current.content,
        expected_revision: active.revision,
      });
      setActive(saved);
      setState("saved");
      setErr("");
      void loadList(q);
    } catch (e: any) {
      const msg = String(e?.message || e || "");
      if (e?.status === 409 || /conflict/i.test(msg)) {
        setState("conflict");
        setErr(copy.pages.conflictHint);
        return;
      }
      setState("error");
      setErr(msg);
    }
  }, [active, copy.pages.conflictHint, loadList, q]);

  const schedule = () => {
    setState("dirty");
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => { void persist(); }, 800);
  };

  const create = async () => {
    const pg = await api.pagesSave({ title: copy.pages.untitled, content: "" });
    setList((cur) => [pg, ...cur]);
    setActive(pg);
    draft.current = { title: pg.title, content: pg.content || "" };
    setState("saved");
  };

  const nested = useMemo(() => tree(list), [list]);

  return (
    <div className="flex h-full min-h-0" data-testid="pages-workspace">
      <aside className="flex w-[16rem] shrink-0 flex-col border-r border-border/60">
        <div className="flex items-center gap-1 px-2 py-2">
          <div className="relative min-w-0 flex-1">
            <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted" />
            <Input
              className="h-8 pl-7 text-[12px]"
              value={q}
              placeholder={copy.pages.search}
              aria-label={copy.pages.search}
              onChange={(e) => {
                setQ(e.target.value);
                void loadList(e.target.value);
              }}
            />
          </div>
          <Button size="sm" variant="ghost" aria-label={copy.pages.newPage} onClick={() => { void create(); }}>
            <Plus className="size-4" />
          </Button>
        </div>
        <nav className="min-h-0 flex-1 overflow-auto px-1.5 pb-2" aria-label={copy.pages.library}>
          {!nested.length ? (
            <EmptyState icon={<FileText className="size-5" />} title={copy.pages.empty} />
          ) : nested.map((n) => (
            <PageRow key={n.id} node={n} depth={0} activeId={active?.id} onOpen={(id) => { void open(id); }} />
          ))}
        </nav>
      </aside>
      {active ? (
        <PageEditor
          title={draft.current.title}
          content={draft.current.content}
          revision={active.revision}
          state={state}
          error={err}
          onTitle={(t) => { draft.current.title = t; schedule(); }}
          onContent={(c) => { draft.current.content = c; schedule(); }}
          onSave={() => { void persist(); }}
          onReload={() => { if (active.id) void open(active.id); }}
        />
      ) : (
        <div className="grid flex-1 place-items-center">
          <EmptyState icon={<FileText className="size-8" />} title={copy.pages.pick} />
        </div>
      )}
    </div>
  );
}

type Node = PageMeta & { children: Node[] };

function tree(list: PageMeta[]): Node[] {
  const map = new Map<string, Node>();
  for (const m of list) map.set(m.id, { ...m, children: [] });
  const roots: Node[] = [];
  for (const n of map.values()) {
    const p = n.parent_id ? map.get(n.parent_id) : undefined;
    if (p) p.children.push(n);
    else roots.push(n);
  }
  return roots;
}

function PageRow(props: { node: Node; depth: number; activeId?: string; onOpen: (id: string) => void }) {
  return (
    <>
      <button
        type="button"
        className={`flex w-full items-center gap-1.5 rounded-lg px-2 py-1 text-left text-[13px] ${props.node.id === props.activeId ? "bg-lift text-foreground" : "text-foreground/80 hover:bg-lift/60"}`}
        style={{ paddingLeft: 8 + props.depth * 12 }}
        onClick={() => props.onOpen(props.node.id)}
      >
        <FileText className="size-3.5 shrink-0 text-muted" />
        <span className="truncate">{props.node.title || props.node.id}</span>
      </button>
      {props.node.children.map((c) => (
        <PageRow key={c.id} node={c} depth={props.depth + 1} activeId={props.activeId} onOpen={props.onOpen} />
      ))}
    </>
  );
}

export function pageOf(v: any): Page {
  return {
    id: str(v?.id || v?.ID),
    space_id: str(v?.space_id || v?.SpaceID || "home"),
    parent_id: str(v?.parent_id || v?.ParentID),
    title: str(v?.title || v?.Title, "Untitled"),
    revision: Number(v?.revision || v?.Revision || 1),
    updated_at: str(v?.updated_at || v?.UpdatedAt),
    content: str(v?.content || v?.Content),
  };
}
