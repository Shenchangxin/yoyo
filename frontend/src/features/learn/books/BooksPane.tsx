/**
 * 拥有：课本这一页：课本列表、从一句意图建一本书、确认提案、确认章节、看着它一章章写出来，以及读写好的章节。
 * 不拥有：写作本身和随堂题的对错，那在引擎里。当前打开的是哪本书由调用方持有，导师对话的范围跟着它走。
 */

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { BookOpen, ChevronDown, ChevronUp } from "lucide-react";
import { useCopy } from "../../../lib/i18n";
import * as api from "../../../lib/client";
import * as books from "../../../lib/learn-books";
import type { BookChapterView, BookProposal, BookSummary, BookView, ChapterInput } from "../../../lib/learn-books";
import { subscribeLearn } from "../../../lib/learn-events";
import { fill } from "../../../lib/learn-format";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { EmptyState } from "../../../components/ui/empty-state";
import { ProgressHairline } from "../../../components/ui/progress-hairline";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../../components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../components/ui/select";
import { cn } from "../../../lib/utils";
import { ConfirmDialog } from "../../ConfirmDialog";
import { LearnSegment, LearnShell } from "../LearnShell";
import { BlockView } from "./BlockView";

type TomeCopy = ReturnType<typeof useCopy>["learn"]["tome"];

const NO_LIBRARY = "none";
const areaClass = "w-full resize-y rounded-lg border border-border/60 bg-transparent px-3 py-2 text-ui-body outline-none focus-visible:border-accent/60";

/** building 在章节正在后台排的时候为真。 */
function building(b: BookSummary): boolean {
  return b.status === "draft" && b.pause_reason === "building_spine";
}

/** statusLabel 说出一本书现在的状态。 */
function statusLabel(L: TomeCopy, b: BookSummary): string {
  if (building(b)) return L.statusBuilding;
  switch (b.status) {
    case "draft": return L.statusDraft;
    case "spine_ready": return L.statusSpine;
    case "compiling": return `${L.statusCompiling} · ${fill(L.chaptersDone, { done: b.ready_chapters, total: b.chapters })}`;
    case "paused": return `${L.statusPaused} · ${fill(L.chaptersDone, { done: b.ready_chapters, total: b.chapters })}`;
    case "error": return L.statusError;
    default: return L.statusReady;
  }
}

/** Labeled 是表单里的一行：上面是标签，下面是控件。 */
function Labeled(props: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-ui-footnote text-muted">{props.label}</span>
      {props.children}
    </label>
  );
}

/**
 * BooksPane 画出课本列表或一本打开的课本。tabs 是页头下面那一排标签，title 是页头的标题。
 * libraryId 是文库里当前选中的库，新建课本时默认以它为依据。sessionId 是这本书的导师对话，写作和判题用它的模型。
 * openId 是当前打开的课本，onOpen 换掉它；onBook 把打开的课本报给调用方。
 */
export function BooksPane(props: {
  tabs: ReactNode;
  title: string;
  libraryId: string;
  sessionId?: string;
  openId: string;
  onOpen: (id: string) => void;
  onBook?: (book: BookSummary | null) => void;
}) {
  const L = useCopy().learn.tome;
  const { openId } = props;
  const [list, setList] = useState<BookSummary[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [view, setView] = useState<BookView | null>(null);
  const [chapter, setChapter] = useState<BookChapterView | null>(null);
  const [chapterId, setChapterId] = useState("");
  const [creating, setCreating] = useState(false);
  const [intent, setIntent] = useState("");
  const [depth, setDepth] = useState("standard");
  const [libs, setLibs] = useState<{ id: string; name: string }[]>([]);
  const [library, setLibrary] = useState(NO_LIBRARY);
  const [proposal, setProposal] = useState<BookProposal | null>(null);
  const [spine, setSpine] = useState<ChapterInput[] | null>(null);
  const [renaming, setRenaming] = useState("");
  const [confirm, setConfirm] = useState<"" | "delete" | "rewrite">("");
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");
  const [err, setErr] = useState("");

  const loadList = useCallback(async () => {
    try {
      setList(await books.books());
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setLoaded(true);
    }
  }, []);

  /** 重读打开的这本书。书已经不在时回到列表。 */
  const loadBook = useCallback(async () => {
    if (!openId) {
      setView(null);
      return;
    }
    try {
      const next = await books.book(openId);
      setView(next);
      props.onBook?.(next.book);
    } catch {
      setView(null);
      props.onOpen("");
    }
  }, [openId]);

  /** 重读正打开的这一章。 */
  const loadChapter = useCallback(async (id: string) => {
    if (!openId) return;
    try {
      const next = await books.chapter(openId, id);
      setChapter(next);
      setChapterId(next.id);
    } catch {
      setChapter(null);
    }
  }, [openId]);

  useEffect(() => {
    void loadList();
    void api.learnCall("learn.libraries").then(/** 笔记库不当写书的依据。 */ (rows) => {
      const all = Array.isArray(rows) ? rows : [];
      setLibs(all.filter((item) => item?.id && item.id !== "notes").map((item) => ({ id: String(item.id), name: String(item.name || item.id) })));
    }).catch(() => setLibs([]));
  }, [loadList]);

  useEffect(() => {
    setView(null);
    setChapter(null);
    setChapterId("");
    setProposal(null);
    setSpine(null);
    setRenaming("");
    setNote("");
    setErr("");
    if (!openId) props.onBook?.(null);
    void loadBook();
  }, [openId]);

  const status = view?.book.status || "";
  const reading = status === "compiling" || status === "paused" || status === "ready" || status === "error";
  const live = !!view && (status === "compiling" || building(view.book));

  useEffect(() => {
    if (reading && view && view.chapters.length > 0 && !chapterId) void loadChapter(view.progress.chapter_id || "");
  }, [reading, view?.book.id, chapterId]);

  useEffect(() => subscribeLearn(/** 课本有变化就重读；变的是打开的这本时，它的全貌和正打开的章节也重读。 */ (change) => {
    if (change.scope !== "book") return;
    void loadList();
    if (change.id && change.id === openId) {
      void loadBook();
      if (chapterId) void loadChapter(chapterId);
    }
  }), [loadList, loadBook, loadChapter, openId, chapterId]);

  useEffect(() => {
    if (!live) return;
    /** 写作进行时每隔一秒半看一眼。桌面壳里有事件推送，这里给收不到事件的环境兜底。 */
    const timer = window.setInterval(() => {
      void loadBook();
      if (chapterId) void loadChapter(chapterId);
    }, 1500);
    return () => window.clearInterval(timer);
  }, [live, loadBook, loadChapter, chapterId]);

  /** 跑一个动作，出错时把原因留在页面上，做完重读这本书和列表。 */
  const run = async (fn: () => Promise<unknown>) => {
    if (busy) return;
    setBusy(true);
    setErr("");
    setNote("");
    try {
      await fn();
      await loadBook();
      void loadList();
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  /** 按写下的意图请模型写一份提案，并打开这本新书。 */
  const create = async () => {
    const text = intent.trim();
    if (!text || busy) return;
    setBusy(true);
    setErr("");
    try {
      const made = await books.createBook({ intent: text, libraryId: library === NO_LIBRARY ? "" : library, depth, session: props.sessionId });
      setIntent("");
      setCreating(false);
      void loadList();
      props.onOpen(made.book.id);
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  /** 把整本书导成 Markdown 放进剪贴板。 */
  const exportBook = async () => {
    if (!view) return;
    try {
      const out = await books.exportBook(view.book.id);
      await navigator.clipboard.writeText(out.markdown);
      setNote(L.exported);
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  if (!openId || !view) {
    return (
      <LearnShell
        title={props.title}
        subtitle={L.subtitle}
        tabs={props.tabs}
        action={<Button size="sm" className="whitespace-nowrap" aria-expanded={creating} onClick={() => { setCreating((v) => !v); setLibrary(libs.some((l) => l.id === props.libraryId) ? props.libraryId : NO_LIBRARY); }}>{L.newBook}</Button>}
      >
        <div className="mx-auto flex max-w-[720px] flex-col gap-4 px-8 py-6" data-testid="learn-books">
          {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
          {creating ? (
            <form
              className="flex flex-col gap-3 border-b border-border/50 pb-5"
              onSubmit={(e) => {
                e.preventDefault();
                void create();
              }}
            >
              <Labeled label={L.intentLabel}>
                <textarea className={cn(areaClass, "min-h-20")} value={intent} placeholder={L.intentPh} autoFocus onChange={(e) => setIntent(e.target.value)} />
              </Labeled>
              <div className="flex flex-wrap items-end gap-4">
                {libs.length > 0 ? (
                  <div className="flex flex-col gap-1">
                    <span className="text-ui-footnote text-muted">{L.libraryLabel}</span>
                    <Select value={library} onValueChange={setLibrary}>
                      <SelectTrigger className="w-[200px]" aria-label={L.libraryLabel}><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value={NO_LIBRARY}>{L.libraryNone}</SelectItem>
                        {libs.map((item) => <SelectItem key={item.id} value={item.id}>{item.name}</SelectItem>)}
                      </SelectContent>
                    </Select>
                  </div>
                ) : null}
                <div className="flex flex-col gap-1">
                  <span className="text-ui-footnote text-muted">{L.depthLabel}</span>
                  <LearnSegment
                    label={L.depthLabel}
                    value={depth}
                    onChange={setDepth}
                    options={[{ id: "brief", label: L.depthBrief }, { id: "standard", label: L.depthStandard }, { id: "deep", label: L.depthDeep }]}
                  />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button size="sm" type="submit" className="whitespace-nowrap" disabled={!intent.trim() || busy}>{busy ? L.creating : L.create}</Button>
                <Button size="sm" type="button" variant="ghost" className="whitespace-nowrap" onClick={() => setCreating(false)}>{L.cancel}</Button>
              </div>
            </form>
          ) : null}
          {loaded && list.length === 0 && !creating ? <EmptyState title={L.empty} body={L.emptyHint} /> : null}
          <ul className="flex flex-col">
            {list.map((b) => (
              <li key={b.id}>
                <button type="button" onClick={() => props.onOpen(b.id)} className="flex w-full items-center gap-3 border-b border-border/50 py-3 text-left hover:bg-lift/40">
                  <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-lift text-muted"><BookOpen className="size-4" aria-hidden /></span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-ui-body-l">{b.title}</span>
                    <span className="mt-0.5 block truncate text-ui-footnote text-muted">{statusLabel(L, b)}</span>
                    {b.chapters > 0 && b.status !== "draft" && b.status !== "spine_ready" ? <ProgressHairline className="mt-2" value={(b.ready_chapters / b.chapters) * 100} /> : null}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      </LearnShell>
    );
  }

  const b = view.book;
  const draft = proposal || view.proposal;
  const chapters = spine || view.chapters.map((c) => ({ id: c.id, title: c.title, summary: c.summary, objectives: c.objectives, content_type: c.content_type }));
  const at = view.chapters.findIndex((c) => c.id === chapterId);
  const pausedWhy = b.pause_reason === "user" ? L.pausedUser : b.pause_reason === "interrupted" ? L.pausedInterrupted : b.pause_reason;

  /** 改书脊里的一章。 */
  const setRow = (i: number, patch: Partial<ChapterInput>) => setSpine(chapters.map((c, k) => (k === i ? { ...c, ...patch } : c)));
  /** 把书脊里的一章挪一位。 */
  const move = (i: number, by: number) => {
    const next = [...chapters];
    const j = i + by;
    if (j < 0 || j >= next.length) return;
    [next[i], next[j]] = [next[j], next[i]];
    setSpine(next);
  };

  return (
    <LearnShell
      title={b.title}
      subtitle={statusLabel(L, b)}
      back={{ label: L.allBooks, onClick: () => props.onOpen("") }}
      action={(
        <div className="flex items-center gap-1">
          {status === "compiling" ? <Button size="sm" variant="outline" className="whitespace-nowrap" disabled={busy} onClick={() => { void run(() => books.pauseBook(b.id)); }}>{L.pause}</Button> : null}
          {status === "paused" || status === "error" ? <Button size="sm" className="whitespace-nowrap" disabled={busy} onClick={() => { void run(() => books.resumeBook(b.id, props.sessionId)); }}>{L.resume}</Button> : null}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="whitespace-nowrap">{L.more}</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setRenaming(b.title)}>{L.rename}</DropdownMenuItem>
              {reading ? <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void exportBook(); }}>{L.exportBook}</DropdownMenuItem> : null}
              {reading && chapter ? <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setConfirm("rewrite")}>{L.rewriteChapter}</DropdownMenuItem> : null}
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setConfirm("delete")}>{L.remove}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}
    >
      <div className="mx-auto flex max-w-[700px] flex-col gap-5 px-8 py-6" data-testid="learn-book" data-status={building(b) ? "building" : status}>
        {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
        {note ? <p className="text-ui-footnote text-muted" aria-live="polite">{note}</p> : null}
        {renaming ? (
          <form
            className="flex items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              const title = renaming.trim();
              if (!title) return;
              setRenaming("");
              void run(() => books.renameBook(b.id, title));
            }}
          >
            <Input className="min-w-0 flex-1" value={renaming} placeholder={L.renamePh} aria-label={L.renamePh} autoFocus onChange={(e) => setRenaming(e.target.value)} />
            <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap">{L.save}</Button>
            <Button size="sm" type="button" variant="ghost" className="shrink-0 whitespace-nowrap" onClick={() => setRenaming("")}>{L.cancel}</Button>
          </form>
        ) : null}

        {building(b) ? <p className="text-ui-body text-muted" aria-live="polite">{L.building}</p> : null}

        {status === "draft" && !building(b) ? (
          <section className="flex flex-col gap-3" data-testid="learn-book-proposal">
            <div>
              <p className="text-ui-body font-medium">{L.proposalTitle}</p>
              <p className="mt-1 text-ui-footnote text-muted">{L.proposalHint}</p>
            </div>
            {b.pause_reason ? <p className="text-ui-footnote text-warning">{b.pause_reason}</p> : null}
            <Labeled label={L.fieldTitle}><Input value={draft.title} onChange={(e) => setProposal({ ...draft, title: e.target.value })} /></Labeled>
            <Labeled label={L.fieldDescription}><textarea className={cn(areaClass, "min-h-16")} value={draft.description} onChange={(e) => setProposal({ ...draft, description: e.target.value })} /></Labeled>
            <Labeled label={L.fieldScope}><textarea className={cn(areaClass, "min-h-20")} value={draft.scope} onChange={(e) => setProposal({ ...draft, scope: e.target.value })} /></Labeled>
            <Labeled label={L.fieldLevel}><Input value={draft.target_level} onChange={(e) => setProposal({ ...draft, target_level: e.target.value })} /></Labeled>
            <div className="flex flex-col gap-1">
              <span className="text-ui-footnote text-muted">{L.fieldChapters}</span>
              <div>
                <LearnSegment
                  label={L.fieldChapters}
                  value={String(Math.min(8, Math.max(2, draft.estimated_chapters || 4)))}
                  onChange={(v) => setProposal({ ...draft, estimated_chapters: Number(v) })}
                  options={[2, 3, 4, 5, 6, 7, 8].map((n) => ({ id: String(n), label: String(n) }))}
                />
              </div>
            </div>
            {draft.rationale ? <p className="text-ui-footnote leading-relaxed text-muted"><span className="text-muted/80">{L.rationale}</span><span className="text-muted/50"> · </span>{draft.rationale}</p> : null}
            <div>
              <Button size="sm" className="whitespace-nowrap" disabled={busy || !draft.title.trim()} onClick={() => { void run(async () => { await books.confirmProposal(b.id, draft, props.sessionId); setProposal(null); }); }}>{L.confirmProposal}</Button>
            </div>
          </section>
        ) : null}

        {status === "spine_ready" ? (
          <section className="flex flex-col gap-3" data-testid="learn-book-spine">
            <div>
              <p className="text-ui-body font-medium">{L.spineTitle}</p>
              <p className="mt-1 text-ui-footnote text-muted">{L.spineHint}</p>
            </div>
            <ol className="flex flex-col">
              {chapters.map((c, i) => (
                <li key={c.id || `new-${i}`} className="flex items-start gap-2 border-b border-border/50 py-3">
                  <span className="mt-2 w-5 shrink-0 text-ui-footnote tabular-nums text-muted">{i + 1}</span>
                  <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                    <Input value={c.title} placeholder={L.chapterTitlePh} aria-label={L.chapterTitlePh} onChange={(e) => setRow(i, { title: e.target.value })} />
                    <textarea className={cn(areaClass, "min-h-14 text-ui-footnote")} value={c.summary} placeholder={L.chapterSummaryPh} aria-label={L.chapterSummaryPh} onChange={(e) => setRow(i, { summary: e.target.value })} />
                  </div>
                  <div className="flex shrink-0 flex-col">
                    <Button size="icon" variant="ghost" aria-label={L.moveUp} disabled={i === 0} onClick={() => move(i, -1)}><ChevronUp className="size-3.5" aria-hidden /></Button>
                    <Button size="icon" variant="ghost" aria-label={L.moveDown} disabled={i === chapters.length - 1} onClick={() => move(i, 1)}><ChevronDown className="size-3.5" aria-hidden /></Button>
                  </div>
                  <Button size="xs" variant="ghost" className="mt-1 shrink-0" disabled={chapters.length <= 1} onClick={() => setSpine(chapters.filter((_, k) => k !== i))}>{L.removeChapter}</Button>
                </li>
              ))}
            </ol>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                size="sm"
                className="whitespace-nowrap"
                disabled={busy || chapters.some((c) => !c.title.trim())}
                onClick={() => { void run(async () => { if (spine) await books.saveSpine(b.id, spine); await books.confirmSpine(b.id, props.sessionId); setSpine(null); }); }}
              >
                {L.confirmSpine}
              </Button>
              <Button size="sm" variant="outline" className="whitespace-nowrap" disabled={chapters.length >= 12} onClick={() => setSpine([...chapters, { title: "", summary: "", objectives: [] }])}>{L.addChapter}</Button>
              <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={busy} onClick={() => { void run(async () => { await books.confirmProposal(b.id, undefined, props.sessionId); setSpine(null); }); }}>{L.redoSpine}</Button>
            </div>
          </section>
        ) : null}

        {reading ? (
          <>
            {status === "paused" && pausedWhy ? <p className="text-ui-footnote text-warning">{pausedWhy}</p> : null}
            <ProgressHairline value={b.chapters > 0 ? (b.ready_chapters / b.chapters) * 100 : 0} />
            <nav aria-label={L.chapterList}>
              <ol className="flex flex-col">
                {view.chapters.map((c, i) => {
                  const on = c.id === chapterId;
                  const state = c.status === "ready" ? "" : c.status === "partial" ? L.chapterPartial : c.status === "error" ? L.chapterFailed : c.status === "pending" ? L.chapterPending : L.chapterWriting;
                  return (
                    <li key={c.id}>
                      <button
                        type="button"
                        aria-current={on ? "true" : undefined}
                        className={cn("flex w-full items-baseline gap-3 rounded-md px-2 py-1.5 text-left text-ui-body hover:bg-lift/60", on ? "bg-lift text-foreground" : "text-muted")}
                        onClick={() => { void loadChapter(c.id); }}
                      >
                        <span className="w-5 shrink-0 text-ui-footnote tabular-nums">{i + 1}</span>
                        <span className="min-w-0 flex-1 truncate">{c.title}</span>
                        {state ? <span className={cn("shrink-0 whitespace-nowrap text-ui-caption", c.status === "error" ? "text-danger" : "text-muted/70")}>{state}</span> : null}
                      </button>
                    </li>
                  );
                })}
              </ol>
            </nav>
            {chapter ? (
              <article className="flex flex-col gap-6 border-t border-border/50 pt-5" data-testid="learn-book-chapter">
                <header>
                  <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{fill(L.chapterN, { n: at >= 0 ? at + 1 : chapter.order + 1 })}</p>
                  <h2 className="mt-1 text-ui-title font-semibold leading-snug tracking-[-0.03em]">{chapter.title}</h2>
                  {chapter.summary ? <p className="mt-1 text-ui-body leading-relaxed text-muted">{chapter.summary}</p> : null}
                  {chapter.objectives.length > 0 ? (
                    <div className="mt-3 text-ui-body">
                      <p className="text-ui-footnote text-muted">{L.objectives}</p>
                      <ul className="mt-1 list-disc pl-5 text-muted">
                        {chapter.objectives.map((o, i) => <li key={i}>{o}</li>)}
                      </ul>
                    </div>
                  ) : null}
                </header>
                {chapter.error ? <p className="text-ui-footnote text-danger">{chapter.error}</p> : null}
                {chapter.blocks.length === 0 ? <p className="text-ui-body text-muted">{chapter.status === "pending" ? L.chapterPending : L.chapterWriting}</p> : null}
                {chapter.blocks.map((blk) => (
                  <BlockView key={blk.id} bookId={b.id} chapterId={chapter.id} block={blk} sessionId={props.sessionId} onChanged={() => { void loadChapter(chapter.id); void loadBook(); }} />
                ))}
                <div className="flex items-center justify-between gap-3 border-t border-border/50 pt-4">
                  <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={at <= 0} onClick={() => { void loadChapter(view.chapters[at - 1].id); }}>{L.prevChapter}</Button>
                  <Button size="sm" variant={at >= view.chapters.length - 1 ? "ghost" : "outline"} className="whitespace-nowrap" disabled={at < 0 || at >= view.chapters.length - 1} onClick={() => { void loadChapter(view.chapters[at + 1].id); }}>{L.nextChapter}</Button>
                </div>
              </article>
            ) : null}
          </>
        ) : null}
      </div>
      <ConfirmDialog
        open={confirm !== ""}
        title={confirm === "delete" ? L.removeTitle : L.rewriteTitle}
        body={confirm === "delete" ? L.removeBody : L.rewriteBody}
        confirmLabel={confirm === "delete" ? L.remove : L.rewriteChapter}
        danger={confirm === "delete"}
        onCancel={() => setConfirm("")}
        onConfirm={async () => {
          const what = confirm;
          setConfirm("");
          if (what === "delete") {
            try {
              await books.deleteBook(b.id);
              void loadList();
              props.onOpen("");
            } catch (e) {
              setErr(errMessage(e));
            }
            return;
          }
          if (chapter) await run(() => books.regenerateChapter(b.id, chapter.id, props.sessionId));
        }}
      />
    </LearnShell>
  );
}
