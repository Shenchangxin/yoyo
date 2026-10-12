import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChevronRight, FileText, Plus, Search } from "lucide-react";
import { getLocale, useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { EmptyState } from "../../components/ui/empty-state";
import { ConfirmDialog } from "../ConfirmDialog";
import { cn } from "../../lib/utils";
import { PageEditor, type SaveState } from "./PageEditor";
import { buildPageTree, type PageMeta, type PageNode } from "./tree";
import { formatPageWhen } from "./when";

type Page = PageMeta & { content: string };

export function PagesWorkspace(props: {
  sessionId?: string;
  boundPageId?: string;
  onOpenInChat?: (pageId: string) => void;
  onUnbindChat?: () => void;
}) {
  const copy = useCopy();
  const locale = getLocale();
  const [q, setQ] = useState("");
  const [list, setList] = useState<PageMeta[]>([]);
  const [active, setActive] = useState<Page | null>(null);
  const [state, setState] = useState<SaveState>("saved");
  const [err, setErr] = useState("");
  const [editorGen, setEditorGen] = useState(0);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [pendingDelete, setPendingDelete] = useState<Page | null>(null);
  const timer = useRef<number>(0);
  const draft = useRef<{ id: string; title: string; content: string }>({ id: "", title: "", content: "" });
  const activeRef = useRef<Page | null>(null);
  const dirtyRef = useRef(false);
  const epochRef = useRef(0);
  const persistRef = useRef<() => Promise<boolean>>(async () => true);
  const qRef = useRef(q);
  qRef.current = q;
  activeRef.current = active;

  const loadList = useCallback(async (query = qRef.current) => {
    const rows = await api.pagesList(query).catch(() => []);
    setList((Array.isArray(rows) ? rows : []).map((row) => pageOf(row)));
  }, []);

  useEffect(() => {
    void loadList(q);
  }, [loadList, q]);

  const persistOnce = useCallback(async (depth = 0): Promise<boolean> => {
    const page = activeRef.current;
    const d = { ...draft.current };
    const epoch = epochRef.current;
    if (!page || !d.id || d.id !== page.id) return true;
    setState("saving");
    try {
      const saved = pageOf(await api.pagesSave({
        ...page,
        id: d.id,
        title: d.title,
        content: d.content,
        expected_revision: page.revision,
      }));
      if (epoch !== epochRef.current) return true;
      activeRef.current = saved;
      setActive(saved);
      const stillSame = draft.current.title === d.title && draft.current.content === d.content;
      dirtyRef.current = !stillSame;
      setState(stillSame ? "saved" : "dirty");
      setErr("");
      void loadList(qRef.current);
      if (!stillSame && depth < 5) return persistOnce(depth + 1);
      if (!stillSame) {
        window.clearTimeout(timer.current);
        timer.current = window.setTimeout(() => { void persistRef.current(); }, 800);
      }
      return true;
    } catch (e: any) {
      if (epoch !== epochRef.current) return true;
      const msg = String(e?.message || e || "");
      if (e?.status === 409 || /conflict/i.test(msg)) {
        setState("conflict");
        setErr(copy.pages.conflictHint);
        return false;
      }
      setState("error");
      setErr(msg);
      return false;
    }
  }, [copy.pages.conflictHint, loadList]);

  const chainRef = useRef(Promise.resolve(true));
  const persist = useCallback((): Promise<boolean> => {
    const next = chainRef.current.then(() => persistOnce(), () => persistOnce());
    chainRef.current = next.then(() => true, () => true);
    return next;
  }, [persistOnce]);
  persistRef.current = persist;

  useEffect(() => () => {
    window.clearTimeout(timer.current);
    if (dirtyRef.current) void persistRef.current();
  }, []);

  const adopt = (pg: Page) => {
    epochRef.current += 1;
    activeRef.current = pg;
    draft.current = { id: pg.id, title: pg.title, content: pg.content || "" };
    dirtyRef.current = false;
    setActive(pg);
    setEditorGen((n) => n + 1);
    setState("saved");
    setErr("");
  };

  const open = async (id: string, opts?: { force?: boolean; bind?: boolean }) => {
    window.clearTimeout(timer.current);
    if (!opts?.force && dirtyRef.current) {
      const ok = await persist();
      if (!ok) return;
    }
    const pg = pageOf(await api.pagesGet(id).catch(() => null));
    if (!pg.id) return;
    adopt(pg);
    if (opts?.bind) props.onOpenInChat?.(pg.id);
  };

  const schedule = () => {
    dirtyRef.current = true;
    setState("dirty");
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => { void persist(); }, 800);
  };

  const create = async (parentId = "") => {
    window.clearTimeout(timer.current);
    if (dirtyRef.current) {
      const ok = await persist();
      if (!ok) return;
    }
    try {
      const pg = pageOf(await api.pagesSave({
        title: copy.pages.untitled,
        content: "",
        ...(parentId ? { parent_id: parentId } : {}),
      }));
      if (!pg.id) throw new Error(copy.pages.error);
      if (parentId) setCollapsed((cur) => ({ ...cur, [parentId]: false }));
      setList((cur) => [pg, ...cur.filter((x) => x.id !== pg.id)]);
      adopt(pg);
    } catch (e: any) {
      setState("error");
      setErr(String(e?.message || e || ""));
    }
  };

  const nest = async (id: string, parentId: string) => {
    if (dirtyRef.current && activeRef.current?.id === id) {
      const ok = await persist();
      if (!ok) return;
    }
    const pg = pageOf(await api.pagesGet(id).catch(() => null));
    if (!pg.id) return;
    const saved = pageOf(await api.pagesSave({ ...pg, parent_id: parentId || "-", expected_revision: pg.revision }));
    void loadList(q);
    if (activeRef.current?.id === saved.id) {
      activeRef.current = { ...activeRef.current, ...saved, content: draft.current.content };
      setActive(activeRef.current);
    }
  };

  const remove = async (pg: Page) => {
    try {
      await api.pagesDelete(pg.id, pg.space_id);
    } catch (e: any) {
      setState("error");
      setErr(String(e?.message || e || ""));
      return;
    }
    if (props.boundPageId === pg.id) props.onUnbindChat?.();
    if (activeRef.current?.id === pg.id) {
      epochRef.current += 1;
      activeRef.current = null;
      draft.current = { id: "", title: "", content: "" };
      dirtyRef.current = false;
      setActive(null);
      setState("saved");
      setErr("");
    }
    void loadList(q);
  };

  const nested = useMemo(() => buildPageTree(list), [list]);
  const searching = q.trim().length > 0;
  const countLabel = list.length === 1 ? copy.pages.countOne : copy.pages.count.replace("{n}", String(list.length));

  return (
    <div className="flex h-full min-h-0" data-testid="pages-workspace">
      <aside className="flex w-[17.5rem] shrink-0 flex-col border-r border-border/60 bg-sidebar/40">
        <div className="px-3 pb-2 pt-3">
          <div className="flex items-baseline justify-between gap-2 px-0.5">
            <h2 className="text-[13px] font-semibold tracking-[-0.02em]">{copy.pages.library}</h2>
            <span className="text-[11px] tabular-nums text-muted">{list.length ? countLabel : ""}</span>
          </div>
          <div className="relative mt-2.5">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" />
            <Input
              className="h-8 rounded-lg pl-8 text-[12px]"
              value={q}
              placeholder={copy.pages.search}
              aria-label={copy.pages.search}
              onChange={(e) => setQ(e.target.value)}
            />
          </div>
          <Button size="sm" variant="lift" className="mt-2 w-full" aria-label={copy.pages.newPage} onClick={() => { void create(); }}>
            <Plus className="size-3.5" />
            {copy.pages.newPage}
          </Button>
        </div>
        <nav className="min-h-0 flex-1 overflow-auto px-2 pb-3" aria-label={copy.pages.library}>
          {!list.length ? (
            searching ? (
              <EmptyState className="px-2 py-8" title={copy.pages.none} body={copy.pages.noneHint} />
            ) : null
          ) : searching ? (
            list.map((n) => (
              <PageRow
                key={n.id}
                node={{ ...n, children: [] }}
                depth={0}
                activeId={active?.id}
                boundId={props.boundPageId}
                locale={locale}
                collapsed={collapsed}
                onToggle={() => {}}
                onOpen={(id) => { void open(id); }}
              />
            ))
          ) : nested.map((n) => (
            <PageRow
              key={n.id}
              node={n}
              depth={0}
              activeId={active?.id}
              boundId={props.boundPageId}
              locale={locale}
              collapsed={collapsed}
              onToggle={(id) => setCollapsed((cur) => ({ ...cur, [id]: !cur[id] }))}
              onOpen={(id) => { void open(id); }}
            />
          ))}
        </nav>
      </aside>
      {active ? (
        <PageEditor
          key={`${active.id}:${editorGen}`}
          title={draft.current.title}
          content={draft.current.content}
          revision={active.revision}
          state={state}
          error={err}
          bound={props.boundPageId === active.id}
          parentId={active.parent_id}
          updatedAt={active.updated_at}
          onTitle={(t) => { draft.current.title = t; schedule(); }}
          onContent={(c) => { draft.current.content = c; schedule(); }}
          onSave={() => { void persist(); }}
          onReload={() => { if (active.id) void open(active.id, { force: true }); }}
          onBind={props.sessionId ? () => props.onOpenInChat?.(active.id) : undefined}
          onUnbind={props.sessionId ? props.onUnbindChat : undefined}
          onDelete={() => setPendingDelete(active)}
          onNewChild={() => { void create(active.id); }}
          onMoveTop={active.parent_id ? () => { void nest(active.id, ""); } : undefined}
        />
      ) : (
        <div className="grid flex-1 place-items-center">
          <EmptyState
            className="empty-rise"
            icon={<FileText className="size-8" />}
            title={list.length ? copy.pages.pick : copy.pages.empty}
            body={list.length ? copy.pages.pickHint : copy.pages.emptyHint}
          />
        </div>
      )}
      <ConfirmDialog
        open={!!pendingDelete}
        danger
        title={copy.pages.delete}
        body={copy.pages.deleteHint}
        confirmLabel={copy.pages.delete}
        onCancel={() => setPendingDelete(null)}
        onConfirm={async () => {
          if (pendingDelete) await remove(pendingDelete);
          setPendingDelete(null);
        }}
      />
    </div>
  );
}

function PageRow(props: {
  node: PageNode;
  depth: number;
  activeId?: string;
  boundId?: string;
  locale: string;
  collapsed: Record<string, boolean>;
  onToggle: (id: string) => void;
  onOpen: (id: string) => void;
}) {
  const open = !props.collapsed[props.node.id];
  const kids = props.node.children.length;
  const when = formatPageWhen(props.node.updated_at || "", props.locale);
  const bound = props.boundId === props.node.id;
  return (
    <>
      <div
        className={cn(
          "group mb-0.5 flex w-full items-start gap-0.5 rounded-lg px-1.5 py-1.5 text-left",
          props.node.id === props.activeId ? "bg-lift text-foreground" : "text-foreground/80 hover:bg-lift/55",
        )}
        style={{ paddingLeft: 6 + props.depth * 12 }}
      >
        {kids ? (
          <button
            type="button"
            className="mt-0.5 grid size-5 shrink-0 place-items-center rounded-md text-muted hover:text-foreground"
            aria-expanded={open}
            onClick={() => props.onToggle(props.node.id)}
          >
            <ChevronRight className={cn("size-3.5 transition-transform duration-150", open && "rotate-90")} />
          </button>
        ) : (
          <span className="size-5 shrink-0" />
        )}
        <button type="button" className="min-w-0 flex-1 text-left" onClick={() => props.onOpen(props.node.id)}>
          <span className="flex items-center gap-1.5">
            <span className="truncate text-[13px] font-medium tracking-[-0.016em]">{props.node.title || props.node.id}</span>
            {bound ? <span className="size-1.5 shrink-0 rounded-full bg-accent" /> : null}
          </span>
          {props.node.preview ? (
            <span className="mt-0.5 line-clamp-2 text-[11px] leading-4 text-muted">{props.node.preview}</span>
          ) : null}
          {when ? <span className="mt-0.5 block text-[10px] tabular-nums text-muted/80">{when}</span> : null}
        </button>
      </div>
      {open ? props.node.children.map((c) => (
        <PageRow
          key={c.id}
          node={c}
          depth={props.depth + 1}
          activeId={props.activeId}
          boundId={props.boundId}
          locale={props.locale}
          collapsed={props.collapsed}
          onToggle={props.onToggle}
          onOpen={props.onOpen}
        />
      )) : null}
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
    preview: str(v?.preview || v?.Preview),
    content: str(v?.content || v?.Content),
  };
}
