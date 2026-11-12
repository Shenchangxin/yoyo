/**
 * 拥有：文库里「资料」这一页：选库、建库、导入、看索引进度、检索，以及文档列表和接着读。
 * 不拥有：阅读器，点开一份文档经 onOpen 交给调用方。索引什么时候建、怎么检索由引擎定。
 */

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { FileText } from "lucide-react";
import { useCopy } from "../../../lib/i18n";
import * as api from "../../../lib/client";
import * as lib from "../../../lib/learn-library";
import type { Hit, LibDocument, Library, Material } from "../../../lib/learn-library";
import { subscribeLearn } from "../../../lib/learn-events";
import { fill, percent } from "../../../lib/learn-format";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { EmptyState } from "../../../components/ui/empty-state";
import { ProgressHairline } from "../../../components/ui/progress-hairline";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../../components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../components/ui/select";
import { cn } from "../../../lib/utils";
import { ConfirmDialog } from "../../ConfirmDialog";
import { LearnShell } from "../LearnShell";

type ShelfCopy = ReturnType<typeof useCopy>["learn"]["shelf"];
type Panel = "" | "new" | "url" | "embed" | "rename";

/** whereLabel 把定位号说成「第 3 页」或「第 2 节」。 */
export function whereLabel(L: ShelfCopy, unit: string, n: number): string {
  return fill(unit === "page" ? L.wherePage : L.whereSection, { n });
}

/** statusLine 说出一个库现在的状态：建索引时是阶段和进度，其余是文档数和状态。 */
function statusLine(L: ShelfCopy, current: Library): string {
  const count = fill(L.docs, { n: current.documents });
  if (current.status === "indexing") {
    const p = current.progress;
    if (!p) return `${count} · ${L.indexing}`;
    if (p.phase === "embedding") return `${count} · ${fill(L.phaseEmbedding, { done: p.done, total: p.total })}`;
    if (p.phase === "publishing") return `${count} · ${L.phasePublishing}`;
    return `${count} · ${fill(L.phaseParsing, { done: Math.min(p.total, p.done + 1), total: p.total })}${p.file ? ` · ${p.file}` : ""}`;
  }
  if (current.status === "error") return `${count} · ${L.statusError}`;
  if (current.status === "needs_reindex") return `${count} · ${L.statusStale}`;
  return `${count} · ${L.statusReady}`;
}

/** docMeta 是文档那一行下面的小字：格式、篇幅和状态。 */
function docMeta(L: ShelfCopy, d: LibDocument): string {
  const parts: string[] = [d.kind ? d.kind.toUpperCase() : ""];
  if (d.status === "parsed" && d.page_count > 0) parts.push(fill(d.unit === "page" ? L.pages : L.sections, { n: d.page_count }));
  if (d.status === "pending") parts.push(L.docPending);
  if (d.status === "failed") parts.push(d.error || L.docFailed);
  if (d.status === "needs_ocr") parts.push(L.docNeedsOcr);
  return parts.filter(Boolean).join(" · ");
}

/**
 * LibraryPane 画出一座文库的资料。tabs 是页头下面那一排标签，由调用方给。
 * libraryId 是当前选中的库，onLibrary 在换库时调用。onOpen 打开一份文档；at 给出要落在哪个单元、要标出哪段话。
 */
export function LibraryPane(props: {
  tabs: ReactNode;
  title: string;
  libraryId: string;
  onLibrary: (id: string) => void;
  onOpen: (libraryId: string, documentId: string, at?: { locator: number; quote?: string }) => void;
}) {
  const L = useCopy().learn.shelf;
  const [libs, setLibs] = useState<Library[]>([]);
  const [docs, setDocs] = useState<LibDocument[]>([]);
  const [recent, setRecent] = useState<Material[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<Hit[] | null>(null);
  const [hitMsg, setHitMsg] = useState("");
  const [searching, setSearching] = useState(false);
  const [panel, setPanel] = useState<Panel>("");
  const [field, setField] = useState("");
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");
  const [err, setErr] = useState("");
  const [confirm, setConfirm] = useState<"" | "library" | string>("");
  const [renaming, setRenaming] = useState<{ id: string; title: string } | null>(null);

  const shelves = libs.filter((item) => item.kind !== "notes");
  const current = shelves.find((item) => item.id === props.libraryId) || shelves[0];
  const currentId = current?.id || "";

  const loadLibs = useCallback(async () => {
    try {
      setLibs(await lib.libraries());
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setLoaded(true);
    }
  }, []);

  /** 读出当前库的文档和最近读过的材料。 */
  const loadDocs = useCallback(async () => {
    if (!currentId) {
      setDocs([]);
      setRecent([]);
      return;
    }
    try {
      const [rows, mats] = await Promise.all([lib.documents(currentId), lib.materials()]);
      setDocs(rows);
      setRecent(mats.filter((m) => m.library_id === currentId).slice(0, 3));
    } catch (e) {
      setErr(errMessage(e));
    }
  }, [currentId]);

  useEffect(() => {
    void loadLibs();
  }, [loadLibs]);

  useEffect(() => {
    if (currentId && currentId !== props.libraryId) props.onLibrary(currentId);
  }, [currentId, props.libraryId]);

  useEffect(() => {
    setHits(null);
    setHitMsg("");
    setQuery("");
    void loadDocs();
  }, [loadDocs]);

  useEffect(() => subscribeLearn(/** 库有变化就重读；变的是当前库时文档也重读。 */ (change) => {
    if (change.scope !== "library") return;
    void loadLibs();
    if (!change.id || change.id === currentId) void loadDocs();
  }), [loadLibs, loadDocs, currentId]);

  const indexing = current?.status === "indexing";
  useEffect(() => {
    if (!indexing) return;
    /** 建索引时每隔一秒半看一眼进度。桌面壳里有事件推送，这里是给收不到事件的环境兜底。 */
    const timer = window.setInterval(() => {
      void loadLibs();
      void loadDocs();
    }, 1500);
    return () => window.clearInterval(timer);
  }, [indexing, loadLibs, loadDocs]);

  /** 跑一个动作，出错时把原因留在页面上，做完重读。 */
  const run = async (fn: () => Promise<void>) => {
    if (busy) return;
    setBusy(true);
    setErr("");
    setNote("");
    try {
      await fn();
      await loadLibs();
      await loadDocs();
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  /** 让学习者选文件并导入当前库。 */
  const importFiles = async () => {
    const files = await api.pickFiles();
    if (files.length === 0 || !currentId) return;
    await run(async () => {
      const out = await lib.importPaths(currentId, files);
      const said = [out.imported > 0 ? fill(L.imported, { n: out.imported }) : L.importedNone];
      if (out.skipped.length > 0) said.push(fill(L.skipped, { n: out.skipped.length, reason: out.skipped[0].reason }));
      setNote(said.join(" "));
    });
  };

  /** 让学习者选一个文件夹，建成关联库。 */
  const linkFolder = async () => {
    const path = await api.pickFolder();
    if (!path) return;
    await run(async () => {
      const made = await lib.createLibrary(path.split(/[\\/]/).filter(Boolean).pop() || L.libraryLabel, "linked", path);
      props.onLibrary(made.id);
    });
  };

  /** 提交页头下面那块小表单：建库、存网页、设嵌入模型或改名。 */
  const submitPanel = async () => {
    const value = field.trim();
    if (panel !== "embed" && !value) return;
    await run(async () => {
      if (panel === "new") {
        const made = await lib.createLibrary(value);
        props.onLibrary(made.id);
      } else if (panel === "url" && currentId) {
        await lib.importURL(currentId, value);
      } else if (panel === "embed" && currentId) {
        await lib.setEmbedModel(currentId, value);
      } else if (panel === "rename" && currentId) {
        await lib.renameLibrary(currentId, value);
      }
      setPanel("");
      setField("");
    });
  };

  /** 在当前库里检索。 */
  const runSearch = async () => {
    const text = query.trim();
    if (!text || !currentId) return;
    setSearching(true);
    setErr("");
    try {
      const res = await lib.search(currentId, text, 12);
      setHits(res.hits);
      setHitMsg(res.message);
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setSearching(false);
    }
  };

  const openPanel = (next: Panel, initial = "") => {
    setPanel(next);
    setField(initial);
  };
  const progress = current?.progress && current.progress.total > 0 ? (current.progress.done / current.progress.total) * 100 : 0;
  const panelLabel = panel === "new" ? L.namePh : panel === "url" ? L.urlPh : panel === "embed" ? L.embedPh : L.namePh;
  const panelHint = panel === "url" ? L.urlHint : panel === "embed" ? L.embedHint : "";

  return (
    <LearnShell
      title={props.title}
      subtitle={L.subtitle}
      tabs={props.tabs}
      action={(
        <div className="flex items-center gap-2">
          {hits !== null ? (
            <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => { setHits(null); setHitMsg(""); setQuery(""); }}>{L.backToFiles}</Button>
          ) : current ? (
            <Button size="sm" className="whitespace-nowrap" disabled={busy} onClick={() => { void importFiles(); }}>{L.importFiles}</Button>
          ) : null}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="whitespace-nowrap">{L.more}</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => openPanel("new")}>{L.newLibrary}</DropdownMenuItem>
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void linkFolder(); }}>{L.linkFolder}</DropdownMenuItem>
              {current ? (
                <>
                  {current.kind === "indexed" ? <DropdownMenuItem className="whitespace-nowrap" onSelect={() => openPanel("url")}>{L.fromLink}</DropdownMenuItem> : null}
                  <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void run(() => lib.rebuild(current.id)); }}>{L.rebuild}</DropdownMenuItem>
                  <DropdownMenuItem className="whitespace-nowrap" onSelect={() => openPanel("embed", current.embed_model)}>{L.embedModel}</DropdownMenuItem>
                  <DropdownMenuItem className="whitespace-nowrap" onSelect={() => openPanel("rename", current.name)}>{L.rename}</DropdownMenuItem>
                  <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setConfirm("library")}>{L.remove}</DropdownMenuItem>
                </>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}
    >
      <div className="mx-auto flex max-w-[720px] flex-col gap-4 px-8 py-6" data-testid="learn-library">
        {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
        {note ? <p className="text-ui-footnote text-muted" aria-live="polite">{note}</p> : null}

        {panel ? (
          <form
            className="flex flex-col gap-1"
            onSubmit={(e) => {
              e.preventDefault();
              void submitPanel();
            }}
          >
            <div className="flex items-center gap-2">
              <Input className="min-w-0 flex-1" value={field} placeholder={panelLabel} aria-label={panelLabel} autoFocus onChange={(e) => setField(e.target.value)} />
              <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap" disabled={busy || (panel !== "embed" && !field.trim())}>
                {panel === "new" ? L.create : panel === "embed" ? L.apply : L.save}
              </Button>
              <Button size="sm" type="button" variant="ghost" className="shrink-0 whitespace-nowrap" onClick={() => openPanel("")}>{L.cancel}</Button>
            </div>
            {panelHint ? <p className="text-ui-footnote text-muted">{panelHint}</p> : null}
          </form>
        ) : null}

        {loaded && !current ? (
          <EmptyState
            title={L.noLibrary}
            body={L.noLibraryHint}
            action={panel === "new" ? undefined : <Button size="sm" className="whitespace-nowrap" onClick={() => openPanel("new")}>{L.newLibrary}</Button>}
          />
        ) : null}

        {current ? (
          <>
            <div className="flex flex-col gap-2">
              <div className="flex flex-wrap items-center gap-3">
                {shelves.length > 1 ? (
                  <Select value={current.id} onValueChange={props.onLibrary}>
                    <SelectTrigger className="w-full max-w-[220px]" aria-label={L.libraryLabel}><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {shelves.map((item) => <SelectItem key={item.id} value={item.id}>{item.name}</SelectItem>)}
                    </SelectContent>
                  </Select>
                ) : (
                  <span className="min-w-0 truncate text-ui-body-l font-medium">{current.name}</span>
                )}
                <span className={cn("min-w-0 flex-1 truncate text-ui-footnote tabular-nums", current.status === "error" ? "text-danger" : "text-muted")} data-testid="learn-library-status" data-status={current.status}>
                  {statusLine(L, current)}
                </span>
              </div>
              {indexing ? <ProgressHairline value={progress} /> : null}
              {current.last_error ? <p className="text-ui-footnote text-warning">{current.last_error}</p> : null}
              {current.kind === "linked" && current.linked_path ? <p className="truncate text-ui-footnote text-muted/80">{fill(L.linkedHint, { path: current.linked_path })}</p> : null}
            </div>

            <form
              className="flex items-center gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                void runSearch();
              }}
            >
              <Input className="min-w-0 flex-1" value={query} placeholder={L.searchPh} aria-label={L.searchPh} onChange={(e) => setQuery(e.target.value)} />
              <Button size="sm" type="submit" variant="outline" className="shrink-0 whitespace-nowrap" disabled={!query.trim() || searching}>{searching ? L.searching : L.search}</Button>
            </form>

            {hits !== null ? (
              hits.length === 0 ? <EmptyState title={hitMsg || L.noHits} body={L.noHitsHint} /> : (
                <ul className="flex flex-col" data-testid="learn-hits">
                  {hits.map((hit) => {
                    const unit = docs.find((d) => d.id === hit.document_id)?.unit || "section";
                    return (
                      <li key={hit.chunk_id} className="border-b border-border/50">
                        <button type="button" className="w-full py-3 text-left hover:bg-lift/40" onClick={() => props.onOpen(hit.library_id, hit.document_id, { locator: hit.page, quote: query.trim() })}>
                          <span className="flex items-baseline justify-between gap-3">
                            <span className="min-w-0 truncate text-ui-body-l">{hit.title}</span>
                            <span className="shrink-0 whitespace-nowrap text-ui-footnote tabular-nums text-muted">{whereLabel(L, unit, hit.page)}</span>
                          </span>
                          {hit.heading ? <span className="mt-0.5 block truncate text-ui-footnote text-muted/80">{hit.heading}</span> : null}
                          <span className="mt-1 line-clamp-3 block text-ui-body leading-relaxed text-muted">{hit.excerpt}</span>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )
            ) : (
              <>
                {recent.length > 0 ? (
                  <section className="flex flex-col gap-1">
                    <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.continueReading}</p>
                    <ul>
                      {recent.map((m) => (
                        <li key={m.id}>
                          <button type="button" className="flex w-full items-center gap-3 border-b border-border/50 py-2.5 text-left hover:bg-lift/40" onClick={() => props.onOpen(m.library_id, m.document_id)}>
                            <span className="min-w-0 flex-1 truncate text-ui-body-l">{m.title}</span>
                            <span className="shrink-0 whitespace-nowrap text-ui-footnote tabular-nums text-muted">{whereLabel(L, m.unit, m.locator)} · {fill(L.progress, { n: percent(m.percent) })}</span>
                          </button>
                        </li>
                      ))}
                    </ul>
                  </section>
                ) : null}
                {docs.length === 0 && !indexing ? <EmptyState title={L.empty} body={L.emptyHint} /> : null}
                <ul className="flex flex-col" data-testid="learn-documents">
                  {docs.map((d) => (
                    <li key={d.id} className="flex items-center gap-2 border-b border-border/50">
                      {renaming?.id === d.id ? (
                        <form
                          className="flex min-w-0 flex-1 items-center gap-2 py-2"
                          onSubmit={(e) => {
                            e.preventDefault();
                            const title = renaming.title.trim();
                            if (!title) return;
                            setRenaming(null);
                            void run(() => lib.renameDocument(currentId, d.id, title));
                          }}
                        >
                          <Input className="min-w-0 flex-1" value={renaming.title} placeholder={L.titlePh} aria-label={L.titlePh} autoFocus onChange={(e) => setRenaming({ id: d.id, title: e.target.value })} />
                          <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap">{L.save}</Button>
                          <Button size="sm" type="button" variant="ghost" className="shrink-0 whitespace-nowrap" onClick={() => setRenaming(null)}>{L.cancel}</Button>
                        </form>
                      ) : (
                        <>
                          <button
                            type="button"
                            className="flex min-w-0 flex-1 items-center gap-3 py-2.5 text-left hover:bg-lift/40 disabled:opacity-60"
                            disabled={d.status === "failed" || d.status === "pending"}
                            onClick={() => props.onOpen(currentId, d.id)}
                          >
                            <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-lift text-muted"><FileText className="size-4" aria-hidden /></span>
                            <span className="min-w-0 flex-1">
                              <span className="block truncate text-ui-body-l">{d.title}</span>
                              <span className={cn("mt-0.5 block truncate text-ui-footnote", d.status === "failed" ? "text-danger" : "text-muted")}>{docMeta(L, d)}</span>
                            </span>
                          </button>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button size="sm" variant="ghost" className="shrink-0 whitespace-nowrap">{L.more}</Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setRenaming({ id: d.id, title: d.title })}>{L.renameDoc}</DropdownMenuItem>
                              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setConfirm(d.id)}>{L.removeDoc}</DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </>
                      )}
                    </li>
                  ))}
                </ul>
              </>
            )}
          </>
        ) : null}
      </div>
      <ConfirmDialog
        open={confirm !== ""}
        title={confirm === "library" ? L.removeTitle : L.removeDocTitle}
        body={confirm === "library" ? L.removeBody : L.removeDocBody}
        confirmLabel={confirm === "library" ? L.remove : L.removeDoc}
        danger
        onCancel={() => setConfirm("")}
        onConfirm={async () => {
          const target = confirm;
          setConfirm("");
          if (!currentId) return;
          await run(async () => {
            if (target === "library") {
              await lib.deleteLibrary(currentId);
              props.onLibrary("");
            } else {
              await lib.removeDocument(currentId, target);
            }
          });
        }}
      />
    </LearnShell>
  );
}
