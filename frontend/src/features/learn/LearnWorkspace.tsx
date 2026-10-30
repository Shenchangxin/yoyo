/** 学习表面的四条泳道。笔记编辑仍交给 PagesWorkspace。每一条泳道和每条路径用自己的学习对话，不占用会话模块。 */

import { useCallback, useEffect, useLayoutEffect, useState, type ReactNode } from "react";
import { getLocale, useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { errMessage } from "../../lib/normalize";
import { learnScopeSpec, type LearnScopeSpec } from "../../lib/learn-scope";
import { useUI } from "../../lib/store";
import { LearnRailNav } from "./LearnRailNav";
import { BookOpen, Check, ChevronLeft, ChevronRight, Library, ListChecks, PenLine } from "lucide-react";
import { LearnSegment, LearnShell, LearnTabs } from "./LearnShell";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { LearnTutor, type TutorPrompt } from "./LearnTutor";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { EmptyState } from "../../components/ui/empty-state";
import { cn } from "../../lib/utils";
import { PagesWorkspace } from "../pages/PagesWorkspace";

type Shelf = "materials" | "books" | "notes";

type Point = { id: string; name: string; kind: string; status: string; mastery: number; mastery_source?: string };
type Module = { id: string; name: string; points?: Point[] };
type Question = { prompt?: string; question_type?: string; options?: string[] };
type Step = { action?: string; name?: string; point_id?: string; question?: Question };
type Path = {
  id: string;
  name: string;
  mode: string;
  teaching?: string;
  library_id?: string;
  counts?: { mastered?: number; total?: number };
  modules?: Module[];
  next?: Step;
};
type Lib = { id: string; name: string; kind: string; status: string; embed_model?: string; published_gen: number; documents: number; last_error?: string };
type Doc = { id: string; title: string; rel_path: string; status: string };
type Card = { id: string; stem: string; qtype: string; options?: string[]; streak?: number };
type Hit = { document_id: string; title: string; page: number; excerpt: string; text?: string; source_hash: string };
type SpineItem = { title: string; query: string; kind?: string };
type Block = { kind: string; body: string; page: number };
type Book = { id: string; title: string; status: string; library_id?: string; proposal?: string; spine?: SpineItem[]; blocks?: Block[]; page?: number };
type Note = { id: string; quote?: string; note?: string; page?: number };
type Reader = { title: string; text: string; materialId: string; documentId: string; page: number; pages: number; highlight: string };
type Desk = { due?: number; reviewed?: number };

const kinds = ["memory", "procedure", "concept", "design"] as const;

function readTutorOpen() {
  try {
    const saved = localStorage.getItem("yoyo-learn-tutor");
    if (saved === "0") return false;
    if (saved === "1") return true;
  } catch {
    /* 读不到时默认打开 */
  }
  return true;
}

function fill(template: string, vars: Record<string, string | number>) {
  return Object.entries(vars).reduce((text, [key, value]) => text.split(`{${key}}`).join(String(value)), template);
}

function zhLocale() {
  return getLocale() === "zh-CN";
}

function kindName(kind: string) {
  const zh = zhLocale();
  if (kind === "memory") return zh ? "记忆" : "Memory";
  if (kind === "procedure") return zh ? "步骤" : "Procedure";
  if (kind === "design") return zh ? "设计" : "Design";
  return zh ? "概念" : "Concept";
}

function bookStatus(status: string) {
  const zh = zhLocale();
  const pair: Record<string, [string, string]> = {
    draft: ["Draft", "草稿"],
    paused: ["Paused", "暂停"],
    ready: ["Ready", "可读"],
    compiling: ["Compiling", "编译中"],
    error: ["Needs a look", "要看一下"],
  };
  const row = pair[status];
  if (!row) return status;
  return zh ? row[1] : row[0];
}

function blockLabel(kind: string) {
  const zh = zhLocale();
  const pair: Record<string, [string, string]> = {
    section: ["Section", "章节"],
    callout: ["Aside", "旁注"],
    quiz: ["Question", "题"],
    code: ["Code", "代码"],
    flash_cards: ["Cards", "卡片"],
    figure: ["Figure", "图"],
    timeline: ["Timeline", "时间线"],
  };
  const row = pair[kind];
  if (!row) return "";
  return zh ? row[1] : row[0];
}

function stepCopy(action: string | undefined, L: ReturnType<typeof useCopy>["learn"], mode: string) {
  if (mode === "outline") return L.stepOutline;
  if (action === "answer_pending") return L.stepAnswer;
  if (action === "review") return L.stepReview;
  if (action === "teach") return L.stepTeach;
  if (action === "assess") return L.stepAssess;
  if (action === "practice") return L.stepPractice;
  if (action === "complete") return L.stepComplete;
  return L.stepProbe;
}

/** LearnWorkspace 管今日、路径、文库和练习。侧栏收起时泳道画在内容左侧。每个子模块的助手由 onEnsureScope 接到自己的会话。chat 跟在内容右侧。 */
export function LearnWorkspace(props: {
  sessionId?: string;
  boundPageId?: string;
  navInRail?: boolean;
  onOpenInChat?: (pageId: string) => void;
  onUnbindChat?: () => void;
  onEnsureScope?: (spec: LearnScopeSpec) => void | Promise<unknown>;
  onTutorPrompt?: (text: string) => void;
  tutorQuiet?: boolean;
  chat?: ReactNode;
}) {
  const copy = useCopy();
  const L = copy.learn;
  const lane = useUI((s) => s.learnLane);
  const [shelf, setShelf] = useState<Shelf>("materials");
  const [err, setErr] = useState("");
  const [today, setToday] = useState<any>(null);
  const [paths, setPaths] = useState<Path[]>([]);
  const [active, setActive] = useState<Path | null>(null);
  const [libs, setLibs] = useState<Lib[]>([]);
  const [libId, setLibId] = useState("notes");
  const [docs, setDocs] = useState<Doc[]>([]);
  const [hits, setHits] = useState<Hit[] | null>(null);
  const [hitMsg, setHitMsg] = useState("");
  const [reader, setReader] = useState<Reader | null>(null);
  const [cards, setCards] = useState<Card[]>([]);
  const [desk, setDesk] = useState<Desk>({});
  const [focusId, setFocusId] = useState("");
  const [books, setBooks] = useState<Book[]>([]);
  const [openBook, setOpenBook] = useState<Book | null>(null);
  const [tutorOpen, setTutorOpen] = useState(readTutorOpen);

  const fail = (e: unknown) => setErr(errMessage(e));
  const run = async (fn: () => Promise<void>) => {
    setErr("");
    try {
      await fn();
    } catch (e) {
      fail(e);
    }
  };

  const refreshPaths = useCallback(async () => {
    const rows = await api.learnCall("learn.paths").catch(() => []);
    setPaths(Array.isArray(rows) ? rows : []);
  }, []);

  const loadDue = useCallback(async () => {
    const [rows, summary] = await Promise.all([
      api.learnCall("learn.practice.due").catch(() => []),
      api.learnCall("learn.practice.summary").catch(() => ({})),
    ]);
    setCards(Array.isArray(rows) ? rows : []);
    setDesk(summary && typeof summary === "object" ? summary : {});
  }, []);

  const openPath = async (id: string) => {
    const view = await api.learnCall("learn.path", { id });
    setActive(view);
    await refreshPaths();
  };

  const showBook = async (id: string, libraryId?: string) => {
    useUI.getState().setLearnLane("library");
    setShelf("books");
    setReader(null);
    if (libraryId) setLibId(libraryId);
    const book = await api.learnCall("learn.book", { id });
    setOpenBook(book);
  };

  useEffect(() => {
    if (lane === "today") void api.learnCall("learn.today").then(setToday).catch(fail);
    if (lane === "study") void refreshPaths();
    if (lane === "library") {
      void api.learnCall("learn.libraries").then((rows) => {
        const list = Array.isArray(rows) ? rows : [];
        setLibs(list);
        const indexed = list.find((lib) => lib.id !== "notes");
        if (!indexed) setShelf((cur) => (cur === "materials" ? "notes" : cur));
        setLibId((cur) => (cur && cur !== "notes" && list.some((lib) => lib.id === cur) ? cur : indexed?.id || "notes"));
      }).catch(fail);
      void api.learnCall("learn.books").then((rows) => setBooks(Array.isArray(rows) ? rows : [])).catch(fail);
    }
    if (lane === "practice") void loadDue();
  }, [lane, refreshPaths, loadDue]);

  useEffect(() => {
    if (lane !== "library" || shelf !== "materials" || !libId || libId === "notes") return;
    void api.learnCall("learn.documents", { library_id: libId }).then((rows) => setDocs(Array.isArray(rows) ? rows : [])).catch(fail);
  }, [lane, shelf, libId, libs]);

  const scope = learnScopeSpec({
    lane,
    title: lane === "today" ? L.today : lane === "practice" ? L.practice : lane === "study" ? L.study : shelf === "notes" ? L.notesTab : L.library,
    pathId: lane === "study" ? active?.id : undefined,
    pathName: lane === "study" ? active?.name : undefined,
    libraryId: lane === "study" ? active?.library_id : lane === "library" ? libId : undefined,
    materialId: lane === "library" && shelf === "materials" ? reader?.materialId : undefined,
    materialTitle: lane === "library" && shelf === "materials" ? reader?.title : undefined,
    bookId: lane === "library" && shelf === "books" ? openBook?.id : undefined,
    bookTitle: lane === "library" && shelf === "books" ? openBook?.title : undefined,
    shelf: lane === "library" ? shelf : undefined,
  });
  const tutor = tutorBrief(L, {
    lane,
    topic: String(today?.journal?.topic || ""),
    pathName: active?.name || "",
    stepName: active?.next?.name || "",
    readerTitle: reader?.title || "",
    bookTitle: openBook?.title || "",
    cardStem: (cards.find((card) => card.id === focusId) || cards[0])?.stem || "",
    shelf,
    libraryName: libs.find((lib) => lib.id === libId)?.name || "",
  });
  const setTutor = (open: boolean) => {
    setTutorOpen(open);
    try {
      localStorage.setItem("yoyo-learn-tutor", open ? "1" : "0");
    } catch {
      /* 写不进时只留在这一次 */
    }
  };
  const scopeKey = [scope.scope, scope.pathId || "", scope.libraryId || "", scope.materialId || "", scope.title || ""].join("|");
  useLayoutEffect(() => {
    void Promise.resolve(props.onEnsureScope?.(scope)).catch((e) => setErr(errMessage(e)));
  }, [scopeKey]);

  const readAt = async (doc: Doc, page: number, materialId: string, highlight: string) => {
    const body = await api.learnCall("learn.read", { library_id: libId, document_id: doc.id, page });
    if (body?.status === "changed") {
      setHitMsg(L.changedFile);
      return;
    }
    setReader({
      title: doc.title || doc.rel_path,
      text: String(body?.text || ""),
      materialId,
      documentId: doc.id,
      page: Number(body?.page || page || 1),
      pages: Number(body?.pages || 1),
      highlight,
    });
  };

  return (
    <div className="flex h-full min-h-0">
      {props.navInRail ? null : (
        <div className="w-[168px] shrink-0 border-r border-border/60 bg-sidebar pt-3">
          <LearnRailNav />
        </div>
      )}
      <div className="flex min-h-0 min-w-0 flex-1">
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          {err ? <p className="px-6 pt-4 text-[12px] text-danger">{err}</p> : null}
          <div className="min-h-0 flex-1">
            {lane === "today" ? (
              <TodayPane
                data={today}
                onSave={async (topic, why, next) => {
                  await run(async () => {
                    await api.learnCall("learn.journal.set", { topic, why, next_focus: next, summary: today?.journal?.summary || "", level: today?.journal?.level || "" });
                    setToday(await api.learnCall("learn.today"));
                  });
                }}
                onRecord={async (insight) => {
                  await run(async () => {
                    await api.learnCall("learn.journal.add", { insight });
                    setToday(await api.learnCall("learn.today"));
                  });
                }}
                onOpenPath={(id) => { useUI.getState().setLearnLane("study"); void run(() => openPath(id)); }}
                onOpenStudy={() => { setActive(null); useUI.getState().setLearnLane("study"); }}
                onOpenRead={() => { setReader(null); setOpenBook(null); setShelf("materials"); useUI.getState().setLearnLane("library"); }}
                onOpenBooks={() => { setReader(null); setShelf("books"); useUI.getState().setLearnLane("library"); }}
                onOpenPractice={() => { useUI.getState().setLearnLane("practice"); }}
                onPractice={(id) => { setFocusId(id); useUI.getState().setLearnLane("practice"); }}
                onOpenBook={(id, libraryId) => { void run(() => showBook(id, libraryId)); }}
              />
            ) : null}
            {lane === "study" ? (
              <StudyPane
                paths={paths}
                active={active}
                onCreate={async (name) => {
                  await run(async () => {
                    const created = await api.learnCall("learn.path.create", { name, library_id: libId === "notes" ? "" : libId });
                    await refreshPaths();
                    if (created?.id) await openPath(created.id);
                  });
                }}
                onOpen={(id) => { void run(() => openPath(id)); }}
                onMode={async (mode) => {
                  if (!active) return;
                  await run(async () => {
                    await api.learnCall("learn.path.mode", { id: active.id, mode });
                    await openPath(active.id);
                  });
                }}
                onProfile={async (teaching) => {
                  if (!active) return;
                  await run(async () => {
                    await api.learnCall("learn.path.profile", { id: active.id, teaching });
                    await openPath(active.id);
                  });
                }}
                onOutline={async (modules) => {
                  if (!active) return;
                  await run(async () => {
                    await api.learnCall("learn.path.outline", { id: active.id, modules });
                    await openPath(active.id);
                  });
                }}
                onClaim={async (pointId) => {
                  if (!active) return;
                  await run(async () => {
                    await api.learnCall("learn.path.claim", { id: active.id, point_id: pointId });
                    await openPath(active.id);
                  });
                }}
                onGrade={async (answer) => {
                  if (!active) return null;
                  try {
                    const res = await api.learnCall("learn.path.grade", { id: active.id, answer });
                    await openPath(active.id);
                    return res;
                  } catch (e) {
                    fail(e);
                    return null;
                  }
                }}
                onClose={() => setActive(null)}
                onAsk={() => {
                  if (!active) return;
                  setTutor(true);
                  const name = active.next?.name || active.name;
                  props.onTutorPrompt?.(name ? `${L.askExplain}: ${name}` : L.askExplain);
                }}
              />
            ) : null}
            {lane === "library" ? (
              reader ? (
                <ReaderPane
                  reader={reader}
                  onBack={() => setReader(null)}
                  onPage={(page) => {
                    const doc = docs.find((item) => item.id === reader.documentId) || { id: reader.documentId, title: reader.title, rel_path: reader.title, status: "" };
                    void run(() => readAt(doc, page, reader.materialId, ""));
                  }}
                  onNote={async (note, quote) => {
                    await run(async () => {
                      await api.learnCall("learn.annotate", { material_id: reader.materialId, note, quote, page: reader.page });
                    });
                  }}
                  onFind={async (query) => {
                    await run(async () => {
                      const res = await api.learnCall("learn.locate", { material_id: reader.materialId, query });
                      const found = Array.isArray(res?.hits) ? res.hits[0] : null;
                      if (!found) {
                        setHitMsg(res?.message || L.noHits);
                        return;
                      }
                      const doc = docs.find((item) => item.id === found.document_id) || { id: found.document_id, title: found.title || reader.title, rel_path: found.title || reader.title, status: "" };
                      await readAt(doc, found.page || 1, reader.materialId, String(found.text || found.excerpt || ""));
                    });
                  }}
                />
              ) : (
                <LibraryPane
                  shelf={shelf}
                  onShelf={(next) => { setShelf(next); if (next !== "books") setOpenBook(null); }}
                  onCloseBook={() => setOpenBook(null)}
                  libs={libs}
                  libId={libId}
                  docs={docs}
                  hits={hits}
                  hitMsg={hitMsg}
                  books={books}
                  openBook={openBook}
                  notes={shelf === "notes" || libId === "notes" ? (
                    <div className="h-full min-h-0">
                      <PagesWorkspace
                        sessionId={props.sessionId}
                        boundPageId={props.boundPageId}
                        onOpenInChat={props.onOpenInChat}
                        onUnbindChat={props.onUnbindChat}
                      />
                    </div>
                  ) : null}
                  onPick={(id) => {
                    setLibId(id);
                    setHits(null);
                    setHitMsg("");
                    setOpenBook(null);
                    setShelf(id === "notes" ? "notes" : shelf === "notes" ? "materials" : shelf);
                  }}
                  onCreate={async (name, kind, path) => {
                    await run(async () => {
                      const lib = await api.learnCall("learn.library.create", { name, kind, path });
                      if (kind === "linked" && path && lib?.id) await api.learnCall("learn.import", { library_id: lib.id, path });
                      setLibs(await api.learnCall("learn.libraries"));
                      if (lib?.id) {
                        setLibId(lib.id);
                        setShelf("materials");
                      }
                    });
                  }}
                  onImport={async (paths) => {
                    await run(async () => {
                      let n = 0;
                      for (const path of paths) {
                        const res = await api.learnCall("learn.import", { library_id: libId, path });
                        n += Number(res?.imported || 0);
                      }
                      setHitMsg(fill(L.imported, { n }));
                      setLibs(await api.learnCall("learn.libraries"));
                      setDocs(await api.learnCall("learn.documents", { library_id: libId }));
                    });
                  }}
                  onRebuild={async () => {
                    await run(async () => {
                      await api.learnCall("learn.rebuild", { library_id: libId });
                      setLibs(await api.learnCall("learn.libraries"));
                    });
                  }}
                  onEmbed={async (model) => { await run(() => api.learnCall("learn.library.embed", { id: libId, model })); }}
                  onSearch={async (query) => {
                    await run(async () => {
                      const res = await api.learnCall("learn.search", { library_id: libId, query });
                      setHits(Array.isArray(res?.hits) ? res.hits : []);
                      setHitMsg(String(res?.message || ""));
                    });
                  }}
                  onClearSearch={() => { setHits(null); setHitMsg(""); }}
                  onOpenDoc={async (doc) => {
                    await run(async () => {
                      const mat = await api.learnCall("learn.material.open", { library_id: libId, document_id: doc.id });
                      await readAt(doc, 1, mat.id, "");
                    });
                  }}
                  onRemove={async (id) => {
                    await run(async () => {
                      await api.learnCall("learn.document.remove", { library_id: libId, document_id: id });
                      setDocs(await api.learnCall("learn.documents", { library_id: libId }));
                    });
                  }}
                  onBook={async (title) => {
                    await run(async () => {
                      const created = await api.learnCall("learn.book.create", { title, library_id: libId });
                      setBooks(await api.learnCall("learn.books"));
                      if (created?.id) setOpenBook(await api.learnCall("learn.book", { id: created.id }));
                    });
                  }}
                  onCompile={async (id, proposal, spine) => {
                    await run(async () => {
                      await api.learnCall("learn.book.proposal", { id, proposal });
                      await api.learnCall("learn.book.spine", {
                        id,
                        spine: spine.split("\n").map((line) => line.trim()).filter(Boolean).map((query) => ({ title: query, query })),
                      });
                      setOpenBook(await api.learnCall("learn.book.compile", { id }));
                      setBooks(await api.learnCall("learn.books"));
                    });
                  }}
                  onPage={async (id, page) => {
                    await run(async () => {
                      await api.learnCall("learn.book.page", { id, page });
                      setOpenBook(await api.learnCall("learn.book", { id }));
                    });
                  }}
                  onSelectBook={(id) => { void run(() => showBook(id, libId === "notes" ? "" : libId)); }}
                />
              )
            ) : null}
            {lane === "practice" ? (
              <PracticePane cards={cards} desk={desk} focusId={focusId} onReload={() => run(loadDue)} onFail={fail} onLeave={() => setFocusId("")} />
            ) : null}
          </div>
        </div>
        {props.chat ? (
          <LearnTutor
            open={tutorOpen}
            subject={tutor.subject}
            about={tutor.about}
            prompts={tutor.prompts}
            quiet={props.tutorQuiet}
            onOpen={setTutor}
            onPrompt={props.onTutorPrompt}
          >
            {props.chat}
          </LearnTutor>
        ) : null}
      </div>
    </div>
  );
}

function PointMark(props: { status: string }) {
  const mastered = props.status === "mastered";
  const learning = props.status === "learning";
  return (
    <span
      className={cn(
        "grid size-5 shrink-0 place-items-center rounded-full border",
        mastered ? "border-foreground bg-foreground text-background" : learning ? "border-foreground" : "border-border",
      )}
      aria-hidden
    >
      {mastered ? <Check className="size-3" /> : null}
    </span>
  );
}

function Field(props: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[12px] text-muted">{props.label}</span>
      {props.children}
    </label>
  );
}

function TodayPane(props: {
  data: any;
  onSave: (topic: string, why: string, next: string) => Promise<void>;
  onRecord: (insight: string) => Promise<void>;
  onOpenPath: (id: string) => void;
  onOpenStudy: () => void;
  onOpenRead: () => void;
  onOpenBooks: () => void;
  onOpenPractice: () => void;
  onPractice: (id: string) => void;
  onOpenBook: (id: string, libraryId?: string) => void;
}) {
  const L = useCopy().learn;
  const j = props.data?.journal || {};
  const [topic, setTopic] = useState(j.topic || "");
  const [why, setWhy] = useState(j.why || "");
  const [next, setNext] = useState(j.next_focus || "");
  const [insight, setInsight] = useState("");
  const [editing, setEditing] = useState(false);
  useEffect(() => {
    setTopic(j.topic || "");
    setWhy(j.why || "");
    setNext(j.next_focus || "");
  }, [j.topic, j.why, j.next_focus]);
  const due = Array.isArray(props.data?.due) ? props.data.due as Card[] : [];
  const paths = Array.isArray(props.data?.paths) ? props.data.paths as Path[] : [];
  const books = Array.isArray(props.data?.books) ? props.data.books as Book[] : [];
  const records = Array.isArray(j.records) ? j.records.slice(0, 3) : [];
  const ways: { id: string; title: string; hint: string; icon: typeof ListChecks; onClick: () => void }[] = [
    { id: "study", title: L.study, hint: L.wayStudyHint, icon: ListChecks, onClick: props.onOpenStudy },
    { id: "read", title: L.reader, hint: L.wayReadHint, icon: Library, onClick: props.onOpenRead },
    { id: "book", title: L.books, hint: L.wayBookHint, icon: BookOpen, onClick: props.onOpenBooks },
    { id: "practice", title: L.practice, hint: L.wayPracticeHint, icon: PenLine, onClick: props.onOpenPractice },
  ];
  return (
    <LearnShell title={L.today} subtitle={L.hubSubtitle}>
      <div className="mx-auto flex max-w-[720px] flex-col gap-8 px-8 py-6">
        <div className="grid grid-cols-2 gap-3">
          {ways.map((way) => {
            const Icon = way.icon;
            return (
              <button
                key={way.id}
                type="button"
                className="flex min-h-[132px] flex-col items-start rounded-xl border border-border/60 bg-card p-4 text-left transition-colors duration-200 hover:bg-lift/50"
                onClick={way.onClick}
              >
                <span className="grid size-9 place-items-center rounded-lg bg-lift text-muted">
                  <Icon className="size-4" aria-hidden />
                </span>
                <span className="mt-4 truncate text-[15px] font-medium tracking-[-0.02em]">{way.title}</span>
                <span className="mt-1 line-clamp-2 text-[12px] leading-relaxed text-muted">{way.hint}</span>
              </button>
            );
          })}
        </div>
        {due.length + paths.length + books.length > 0 ? (
          <section>
            <h2 className="text-[13px] text-muted">{L.attention}</h2>
            <div className="mt-2 flex flex-col">
              {due.slice(0, 3).map((card) => (
                <button key={card.id} type="button" data-testid="learn-due" className="flex w-full items-center gap-3 border-b border-border/60 py-3 text-left" onClick={() => props.onPractice(card.id)}>
                  <span className="shrink-0 whitespace-nowrap text-[12px] text-muted">{L.practice}</span>
                  <span className="min-w-0 flex-1 truncate text-[14px]">{card.stem}</span>
                </button>
              ))}
              {paths.slice(0, 3).map((path) => (
                <button key={path.id} type="button" className="flex w-full items-center gap-3 border-b border-border/60 py-3 text-left" onClick={() => props.onOpenPath(path.id)}>
                  <span className="shrink-0 whitespace-nowrap text-[12px] text-muted">{L.study}</span>
                  <span className="min-w-0 flex-1 truncate text-[14px]">{path.name}</span>
                  <span className="shrink-0 tabular-nums text-[12px] text-muted">{path.counts?.mastered || 0}/{path.counts?.total || 0}</span>
                </button>
              ))}
              {books.slice(0, 2).map((book) => (
                <button key={book.id} type="button" className="flex w-full items-center gap-3 border-b border-border/60 py-3 text-left" onClick={() => props.onOpenBook(book.id, book.library_id)}>
                  <span className="shrink-0 whitespace-nowrap text-[12px] text-muted">{L.books}</span>
                  <span className="min-w-0 flex-1 truncate text-[14px]">{book.title}</span>
                  <span className="shrink-0 whitespace-nowrap text-[12px] text-muted">{book.status === "ready" || book.status === "paused" ? L.readOn : bookStatus(book.status)}</span>
                </button>
              ))}
            </div>
          </section>
        ) : null}
        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-[13px] text-muted">{L.mission}</h2>
            <button type="button" className="shrink-0 whitespace-nowrap text-[12px] text-muted hover:text-foreground" aria-expanded={editing} onClick={() => setEditing((v) => !v)}>{L.editMission}</button>
          </div>
          {!editing && topic ? <p className="text-[15px] leading-snug">{topic}</p> : null}
          {editing ? (
            <div className="flex flex-col gap-3">
              <Field label={L.mission}><Input value={topic} placeholder={L.missionPh} onChange={(e) => setTopic(e.target.value)} /></Field>
              <Field label={L.why}><Input value={why} placeholder={L.whyPh} onChange={(e) => setWhy(e.target.value)} /></Field>
              <Field label={L.next}><Input value={next} placeholder={L.nextPh} onChange={(e) => setNext(e.target.value)} /></Field>
              <div><Button size="sm" className="whitespace-nowrap" onClick={() => { void props.onSave(topic, why, next); setEditing(false); }}>{L.saveMission}</Button></div>
            </div>
          ) : null}
        </section>
        <section className="flex flex-col gap-2">
          {records.map((row: { id: string; insight: string }) => (
            <p key={row.id} className="text-[13px] leading-relaxed text-muted">{row.insight}</p>
          ))}
          <form className="flex items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (!insight.trim()) return; void props.onRecord(insight.trim()); setInsight(""); }}>
            <div className="min-w-0 flex-1">
              <Field label={L.record}><Input value={insight} placeholder={L.recordPh} onChange={(e) => setInsight(e.target.value)} /></Field>
            </div>
            <Button size="sm" type="submit" variant="ghost" className="mb-px shrink-0 whitespace-nowrap">{L.saveRecord}</Button>
          </form>
        </section>
      </div>
    </LearnShell>
  );
}

function StudyPane(props: {
  paths: Path[];
  active: Path | null;
  onCreate: (name: string) => Promise<void>;
  onOpen: (id: string) => void;
  onClose: () => void;
  onMode: (mode: string) => Promise<void>;
  onProfile: (teaching: string) => Promise<void>;
  onOutline: (modules: { name: string; points: { name: string; kind: string }[] }[]) => Promise<void>;
  onClaim: (pointId: string) => Promise<void>;
  onGrade: (answer: string) => Promise<any>;
  onAsk: () => void;
}) {
  const L = useCopy().learn;
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [point, setPoint] = useState("");
  const [kind, setKind] = useState<string>("concept");
  const [draft, setDraft] = useState<{ name: string; kind: string }[]>([]);
  const [guess, setGuess] = useState("");
  const [reveal, setReveal] = useState<{ correct?: boolean; answer?: string } | null>(null);
  useEffect(() => { setGuess(""); setReveal(null); }, [props.active?.id, props.active?.next?.question?.prompt]);
  if (!props.active) {
    return (
      <LearnShell
        title={L.study}
        subtitle={L.wayStudyHint}
        action={<Button size="sm" className="whitespace-nowrap" aria-expanded={creating} onClick={() => setCreating((v) => !v)}>{L.newPath}</Button>}
      >
        <div className="mx-auto flex max-w-[640px] flex-col px-8 py-6">
          {creating ? (
            <form className="mb-4 flex items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (!name.trim()) return; void props.onCreate(name.trim()); setName(""); setCreating(false); }}>
              <div className="min-w-0 flex-1"><Field label={L.pathPh}><Input value={name} onChange={(e) => setName(e.target.value)} /></Field></div>
              <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap">{L.newPath}</Button>
            </form>
          ) : null}
          {props.paths.length === 0 ? <EmptyState title={L.pathsEmpty} body={L.pathsEmptyHint} /> : (
            <ul className="flex flex-col">
              {props.paths.map((path) => {
                const total = path.counts?.total || 0;
                const mastered = path.counts?.mastered || 0;
                return (
                  <li key={path.id}>
                    <button type="button" onClick={() => props.onOpen(path.id)} className="flex w-full items-center gap-3 border-b border-border/60 py-3 text-left hover:bg-lift/40">
                      <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-lift text-muted"><ListChecks className="size-4" aria-hidden /></span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate text-[14px]">{path.name}</span>
                        <span className="mt-0.5 block tabular-nums text-[12px] text-muted">{mastered}/{total}</span>
                        {total > 0 ? <ProgressHairline className="mt-2" value={(mastered / total) * 100} /> : null}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </LearnShell>
    );
  }
  const current = props.active;
  const question = current.next?.question;
  const waiting = current.next?.action === "answer_pending" && question?.prompt;
  const ask = current.mode !== "outline" && current.next?.action && current.next.action !== "complete" && current.next.action !== "answer_pending";
  const teaching = current.teaching === "teach_first" ? "teach_first" : "probe_first";
  return (
    <LearnShell
      title={current.name}
      subtitle={`${current.counts?.mastered || 0}/${current.counts?.total || 0}`}
      back={{ label: L.allPaths, onClick: props.onClose }}
    >
      <div className="mx-auto flex max-w-[640px] flex-col gap-6 px-8 py-6">
        <div className="flex flex-col items-start gap-2">
          <LearnSegment
            label={L.studyMode}
            value={current.mode}
            onChange={(mode) => { void props.onMode(mode); }}
            options={[
              { id: "outline", label: L.outline },
              { id: "study", label: L.studyMode },
              { id: "review", label: L.review },
            ]}
          />
          <LearnSegment
            label={L.probeFirst}
            value={teaching}
            onChange={(id) => { void props.onProfile(id); }}
            options={[
              { id: "probe_first", label: L.probeFirst },
              { id: "teach_first", label: L.teachFirst },
            ]}
          />
        </div>
        <section data-testid="learn-next" className="rounded-xl border border-border/60 p-5">
          <p className="text-[13px] text-muted">{stepCopy(current.next?.action, L, current.mode)}</p>
          {current.next?.name ? <p className="mt-2 text-[18px] font-semibold leading-snug tracking-[-0.03em]">{current.next.name}</p> : null}
          {waiting ? (
            <div className="mt-4 flex flex-col gap-3">
              <p className="text-[15px] font-medium leading-snug">{question?.prompt}</p>
              {question?.options && question.options.length > 0 ? (
                <div className="flex flex-col gap-1">
                  {question.options.map((option) => (
                    <button key={option} type="button" aria-pressed={guess === option} onClick={() => setGuess(option)} className={cn("rounded-lg border border-border/60 px-3 py-2 text-left text-[13px]", guess === option ? "bg-accent/10 text-accent" : "hover:bg-lift")}>
                      {option}
                    </button>
                  ))}
                </div>
              ) : (
                <Field label={L.answerLabel}><Input value={guess} onChange={(e) => setGuess(e.target.value)} /></Field>
              )}
              <div><Button size="sm" className="whitespace-nowrap" onClick={() => { void props.onGrade(guess).then((res) => { if (!res) return; setReveal({ correct: !!res?.correct, answer: res?.reveal?.answer }); }); }}>{L.submitAnswer}</Button></div>
            </div>
          ) : null}
          {reveal ? (
            <div className="mt-4 text-[13px]">
              <p>{reveal.correct ? L.correct : L.incorrect}</p>
              {reveal.answer ? <p className="mt-1 text-muted">{reveal.answer}</p> : null}
              <Button size="sm" className="mt-3 whitespace-nowrap" variant="ghost" onClick={() => setReveal(null)}>{L.continue}</Button>
            </div>
          ) : null}
          {ask && !reveal ? <div className="mt-4"><Button size="sm" className="whitespace-nowrap" onClick={props.onAsk}>{L.askTutor}</Button></div> : null}
        </section>
        {(current.modules || []).map((mod) => (
          <section key={mod.id}>
            <h3 className="text-[13px] text-muted">{mod.name}</h3>
            <ul className="mt-1">
              {(mod.points || []).map((pt) => (
                <li key={pt.id} className="flex items-center gap-3 border-b border-border/60 py-2.5">
                  <PointMark status={pt.status} />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[14px]">{pt.name}</p>
                    <p className="mt-0.5 truncate text-[12px] text-muted">{kindName(pt.kind)} · {pt.status === "mastered" ? L.mastered : pt.status === "learning" ? L.learning : L.fresh} · {Math.round((pt.mastery || 0) * 100)}%</p>
                  </div>
                  {pt.status !== "mastered" ? (
                    <button type="button" className="shrink-0 whitespace-nowrap text-[12px] text-muted hover:text-foreground" onClick={() => { void props.onClaim(pt.id); }}>{L.claim}</button>
                  ) : null}
                </li>
              ))}
            </ul>
          </section>
        ))}
        {current.mode === "outline" ? (
          <div className="flex flex-col gap-3 border-t border-border/60 pt-4">
            <form className="flex items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (!point.trim()) return; setDraft((d) => [...d, { name: point.trim(), kind }]); setPoint(""); }}>
              <div className="min-w-0 flex-1"><Field label={L.pointPh}><Input value={point} onChange={(e) => setPoint(e.target.value)} /></Field></div>
              <label className="mb-px flex shrink-0 flex-col gap-1">
                <span className="text-[12px] text-muted">{L.kindLabel}</span>
                <select className="h-8 whitespace-nowrap rounded-lg border border-border/60 bg-transparent px-2 text-[12px]" value={kind} onChange={(e) => setKind(e.target.value)}>
                  {kinds.map((k) => <option key={k} value={k}>{kindName(k)}</option>)}
                </select>
              </label>
              <Button size="sm" type="submit" variant="ghost" className="mb-px shrink-0 whitespace-nowrap">{L.addPoint}</Button>
            </form>
            {draft.length > 0 ? (
              <div className="flex flex-col gap-2">
                <ul>
                  {draft.map((item, i) => <li key={`${item.name}-${i}`} className="truncate text-[13px] text-muted">{item.name}</li>)}
                </ul>
                <div>
                  <Button size="sm" className="whitespace-nowrap" onClick={() => {
                    const mods = (props.active?.modules || []).map((mod) => ({ name: mod.name, points: (mod.points || []).map((pt) => ({ name: pt.name, kind: pt.kind })) }));
                    if (mods.length === 0) mods.push({ name: props.active?.name || L.outline, points: [] });
                    mods[mods.length - 1].points.push(...draft);
                    void props.onOutline(mods);
                    setDraft([]);
                  }}>{L.saveOutline}</Button>
                </div>
              </div>
            ) : null}
          </div>
        ) : null}
      </div>
    </LearnShell>
  );
}

function LibraryPane(props: {
  shelf: Shelf;
  onShelf: (shelf: Shelf) => void;
  libs: Lib[];
  libId: string;
  docs: Doc[];
  hits: Hit[] | null;
  hitMsg: string;
  books: Book[];
  openBook: Book | null;
  notes: ReactNode;
  onPick: (id: string) => void;
  onCreate: (name: string, kind: string, path: string) => Promise<void>;
  onImport: (paths: string[]) => Promise<void>;
  onRebuild: () => Promise<void>;
  onEmbed: (model: string) => Promise<void>;
  onSearch: (query: string) => Promise<void>;
  onClearSearch: () => void;
  onOpenDoc: (doc: Doc) => Promise<void>;
  onRemove: (id: string) => Promise<void>;
  onBook: (title: string) => Promise<void>;
  onCompile: (id: string, proposal: string, spine: string) => Promise<void>;
  onPage: (id: string, page: number) => Promise<void>;
  onSelectBook: (id: string) => void;
  onCloseBook: () => void;
}) {
  const L = useCopy().learn;
  const tabs = (
    <LearnTabs
      label={L.library}
      value={props.shelf}
      onChange={(id) => props.onShelf(id as Shelf)}
      items={[
        { id: "materials", label: L.materials },
        { id: "books", label: L.books },
        { id: "notes", label: L.notesTab },
      ]}
    />
  );
  if (props.shelf === "notes") {
    return (
      <LearnShell title={L.library} subtitle={L.notesHint} tabs={tabs} scroll={false}>
        <div className="min-h-0 flex-1">{props.notes}</div>
      </LearnShell>
    );
  }
  if (props.shelf === "books") {
    return (
      <BookPane
        tabs={tabs}
        libs={props.libs}
        libId={props.libId}
        onPick={props.onPick}
        books={props.books}
        openBook={props.openBook}
        onBook={props.onBook}
        onSelect={props.onSelectBook}
        onCompile={props.onCompile}
        onPage={props.onPage}
        onClose={props.onCloseBook}
      />
    );
  }
  return (
    <Materials
      tabs={tabs}
      libs={props.libs}
      libId={props.libId}
      docs={props.docs}
      hits={props.hits}
      hitMsg={props.hitMsg}
      onPick={props.onPick}
      onCreate={props.onCreate}
      onImport={props.onImport}
      onRebuild={props.onRebuild}
      onEmbed={props.onEmbed}
      onSearch={props.onSearch}
      onClearSearch={props.onClearSearch}
      onOpenDoc={props.onOpenDoc}
      onRemove={props.onRemove}
    />
  );
}

function libSubtitle(current: Lib | undefined, L: ReturnType<typeof useCopy>["learn"]) {
  if (!current || current.id === "notes") return L.wayReadHint;
  if (current.last_error) return current.last_error;
  if (current.status === "building") return L.indexing;
  if (current.status === "needs_reindex") return L.changedFile;
  return L.wayReadHint;
}

function Materials(props: {
  tabs: ReactNode;
  libs: Lib[];
  libId: string;
  docs: Doc[];
  hits: Hit[] | null;
  hitMsg: string;
  onPick: (id: string) => void;
  onCreate: (name: string, kind: string, path: string) => Promise<void>;
  onImport: (paths: string[]) => Promise<void>;
  onRebuild: () => Promise<void>;
  onEmbed: (model: string) => Promise<void>;
  onSearch: (query: string) => Promise<void>;
  onClearSearch: () => void;
  onOpenDoc: (doc: Doc) => Promise<void>;
  onRemove: (id: string) => Promise<void>;
}) {
  const L = useCopy().learn;
  const [name, setName] = useState("");
  const [query, setQuery] = useState("");
  const [model, setModel] = useState("");
  const [naming, setNaming] = useState(false);
  const [embedOpen, setEmbedOpen] = useState(false);
  const current = props.libs.find((lib) => lib.id === props.libId);
  const searching = props.hits !== null;
  const rows: Hit[] = props.hits || props.docs.map((doc) => ({ document_id: doc.id, title: doc.title || doc.rel_path, page: 0, excerpt: doc.status, source_hash: "" }));
  return (
    <LearnShell
      title={L.library}
      subtitle={libSubtitle(current, L)}
      tabs={props.tabs}
      action={(
        <div className="flex items-center gap-2">
          {searching ? (
            <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => { setQuery(""); props.onClearSearch(); }}>{L.backToFiles}</Button>
          ) : props.libId !== "notes" ? (
            <Button size="sm" className="whitespace-nowrap" onClick={() => { void api.pickFiles().then((files) => { if (files.length) void props.onImport(files); }); }}>{L.importFiles}</Button>
          ) : null}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="whitespace-nowrap">{L.more}</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setNaming(true)}>{L.newLibrary}</DropdownMenuItem>
              {props.libId !== "notes" ? (
                <>
                  <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void api.pickFolder().then((path) => { if (path) void props.onCreate(path.split(/[\\/]/).pop() || L.library, "linked", path); }); }}>{L.linkFolder}</DropdownMenuItem>
                  <DropdownMenuItem className="whitespace-nowrap" onSelect={() => { void props.onRebuild(); }}>{current?.status === "building" ? L.indexing : L.rebuild}</DropdownMenuItem>
                  <DropdownMenuItem className="whitespace-nowrap" onSelect={() => setEmbedOpen(true)}>{L.embed}</DropdownMenuItem>
                </>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}
    >
      <div className="mx-auto flex max-w-[720px] flex-col gap-4 px-8 py-6">
        <LibraryPick libs={props.libs} libId={props.libId} label={L.libraries} onPick={props.onPick} />
        {props.libId !== "notes" ? (
          <form className="flex items-center gap-2" onSubmit={(e) => { e.preventDefault(); void props.onSearch(query); }}>
            <Input className="min-w-0 flex-1" value={query} placeholder={L.searchPh} onChange={(e) => setQuery(e.target.value)} aria-label={L.searchPh} />
            <Button size="sm" type="submit" variant="ghost" className="shrink-0 whitespace-nowrap">{L.search}</Button>
          </form>
        ) : null}
        {naming ? (
          <form className="flex items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (!name.trim()) return; void props.onCreate(name.trim(), "indexed", ""); setName(""); setNaming(false); }}>
            <div className="min-w-0 flex-1"><Field label={L.libraryPh}><Input value={name} onChange={(e) => setName(e.target.value)} /></Field></div>
            <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap">{L.newLibrary}</Button>
          </form>
        ) : null}
        {embedOpen && props.libId !== "notes" ? (
          <form className="flex items-end gap-2" onSubmit={(e) => { e.preventDefault(); void props.onEmbed(model); }}>
            <div className="min-w-0 flex-1"><Field label={L.embedPh}><Input value={model} onChange={(e) => setModel(e.target.value)} /></Field></div>
            <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap">{L.applyEmbed}</Button>
          </form>
        ) : null}
        {props.hitMsg ? <p className="text-[13px] text-muted">{props.hitMsg}</p> : null}
        {searching && props.hits && props.hits.length === 0 && !props.hitMsg ? <EmptyState title={L.noHits} body={L.noHitsHint} /> : null}
        {!searching && rows.length === 0 && props.libId !== "notes" ? <EmptyState title={L.filesEmpty} body={L.filesEmptyHint} /> : null}
        <ul className="flex flex-col">
          {rows.map((hit) => {
            const removable = props.docs.some((doc) => doc.id === hit.document_id);
            return (
              <li key={`${hit.document_id}:${hit.page}:${hit.excerpt}`} className="flex items-center gap-3 border-b border-border/60">
                <button
                  type="button"
                  className="min-w-0 flex-1 py-3 text-left"
                  onClick={() => {
                    const doc = props.docs.find((item) => item.id === hit.document_id) || { id: hit.document_id, title: hit.title, rel_path: hit.title, status: "" };
                    void props.onOpenDoc(doc);
                  }}
                >
                  <span className="block truncate text-[14px]">{hit.title}</span>
                  {searching && hit.excerpt ? <span className="mt-0.5 block truncate text-[12px] text-muted">{hit.excerpt}</span> : null}
                </button>
                {removable ? (
                  <button type="button" className="shrink-0 whitespace-nowrap text-[12px] text-muted hover:text-foreground" onClick={() => { void props.onRemove(hit.document_id); }}>{L.remove}</button>
                ) : null}
              </li>
            );
          })}
        </ul>
      </div>
    </LearnShell>
  );
}

function LibraryPick(props: { libs: Lib[]; libId: string; label: string; onPick: (id: string) => void }) {
  if (!props.libs.some((lib) => lib.id === props.libId)) return null;
  return (
    <Select value={props.libId} onValueChange={props.onPick}>
      <SelectTrigger className="w-full max-w-xs" aria-label={props.label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {props.libs.map((lib) => (
          <SelectItem key={lib.id} value={lib.id}>{lib.name}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function BookPane(props: {
  tabs: ReactNode;
  libs: Lib[];
  libId: string;
  onPick: (id: string) => void;
  books: Book[];
  openBook: Book | null;
  onBook: (title: string) => Promise<void>;
  onSelect: (id: string) => void;
  onCompile: (id: string, proposal: string, spine: string) => Promise<void>;
  onPage: (id: string, page: number) => Promise<void>;
  onClose: () => void;
}) {
  const L = useCopy().learn;
  const [title, setTitle] = useState("");
  const [creating, setCreating] = useState(false);
  const [proposal, setProposal] = useState("");
  const [spine, setSpine] = useState("");
  const [loaded, setLoaded] = useState("");
  const [editing, setEditing] = useState(false);
  const book = props.openBook;
  useEffect(() => {
    if (!book || book.id === loaded) return;
    setLoaded(book.id);
    setProposal(book.proposal || "");
    setSpine((book.spine || []).map((item) => item.query || item.title).filter(Boolean).join("\n"));
    setEditing(book.status === "draft" || !(book.blocks || []).length);
  }, [book, loaded]);
  const mine = props.books.filter((item) => !item.library_id || item.library_id === props.libId || props.libId === "notes");
  if (!book) {
    return (
      <LearnShell
        title={L.library}
        subtitle={L.wayBookHint}
        tabs={props.tabs}
        action={<Button size="sm" className="whitespace-nowrap" disabled={props.libId === "notes"} aria-expanded={creating} onClick={() => setCreating((v) => !v)}>{L.newBook}</Button>}
      >
        <div className="mx-auto flex max-w-[720px] flex-col gap-4 px-8 py-6" data-testid="learn-book">
          <LibraryPick libs={props.libs} libId={props.libId} label={L.libraries} onPick={props.onPick} />
          {creating ? (
            <form className="mb-4 flex items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (!title.trim() || props.libId === "notes") return; void props.onBook(title.trim()); setTitle(""); setCreating(false); }}>
              <div className="min-w-0 flex-1"><Field label={L.bookPh}><Input value={title} onChange={(e) => setTitle(e.target.value)} /></Field></div>
              <Button size="sm" type="submit" className="shrink-0 whitespace-nowrap" disabled={props.libId === "notes"}>{L.newBook}</Button>
            </form>
          ) : null}
          {mine.length === 0 ? <EmptyState title={L.bookEmpty} body={L.bookEmptyHint} /> : (
            <ul>
              {mine.map((item) => (
                <li key={item.id}>
                  <button type="button" onClick={() => props.onSelect(item.id)} className="flex w-full items-center gap-3 border-b border-border/60 py-3 text-left hover:bg-lift/40">
                    <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-lift text-muted"><BookOpen className="size-4" aria-hidden /></span>
                    <span className="min-w-0 flex-1 truncate text-[14px]">{item.title}</span>
                    <span className="shrink-0 whitespace-nowrap text-[12px] text-muted">{bookStatus(item.status)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </LearnShell>
    );
  }
  const page = book.page || 1;
  const pages = Math.max(1, ...(book.blocks || []).map((block) => block.page || 1), 1);
  const blocks = (book.blocks || []).filter((block) => block.page === page);
  return (
    <LearnShell
      title={book.title}
      subtitle={fill(L.pageOf, { page, pages })}
      back={{ label: L.backToBooks, onClick: props.onClose }}
      action={(
        <div className="flex items-center gap-1">
          <Button size="icon" variant="ghost" aria-label={L.prevPage} disabled={page <= 1} onClick={() => { void props.onPage(book.id, page - 1); }}><ChevronLeft /></Button>
          <Button size="icon" variant="ghost" aria-label={L.nextPage} disabled={page >= pages} onClick={() => { void props.onPage(book.id, page + 1); }}><ChevronRight /></Button>
        </div>
      )}
    >
      <article className="mx-auto flex max-w-[680px] flex-col gap-5 px-8 py-6" data-testid="learn-book">
        <div>
          <button type="button" className="whitespace-nowrap text-[12px] text-muted hover:text-foreground" aria-expanded={editing} onClick={() => setEditing((v) => !v)}>{editing ? L.hideSpine : L.editSpine}</button>
        </div>
        {editing ? (
          <div className="flex flex-col gap-3">
            <Field label={L.proposalPh}><Input value={proposal} onChange={(e) => setProposal(e.target.value)} /></Field>
            <Field label={L.spinePh}>
              <textarea className="min-h-24 w-full rounded-lg border border-border/60 bg-transparent px-3 py-2 text-[13px]" value={spine} onChange={(e) => setSpine(e.target.value)} />
            </Field>
            <div><Button size="sm" className="whitespace-nowrap" onClick={() => { void props.onCompile(book.id, proposal, spine); }}>{L.compile}</Button></div>
          </div>
        ) : null}
        {blocks.map((block, i) => (
          <div key={`${block.kind}-${i}`}>
            {blockLabel(block.kind) ? <p className="mb-1 text-[11px] text-muted">{blockLabel(block.kind)}</p> : null}
            <div className={cn(
              "text-[14px] leading-[1.7]",
              block.kind === "section" && "text-[16px] font-semibold tracking-[-0.02em]",
              block.kind === "callout" && "border-l border-border pl-3 text-muted",
              block.kind === "code" && "whitespace-pre-wrap rounded-lg bg-sidebar px-3 py-2 font-mono text-[12px]",
              block.kind !== "code" && "whitespace-pre-wrap",
            )}>{block.body}</div>
          </div>
        ))}
      </article>
    </LearnShell>
  );
}

function ReaderPane(props: {
  reader: Reader;
  onBack: () => void;
  onPage: (page: number) => void;
  onNote: (note: string, quote: string) => Promise<void>;
  onFind: (query: string) => Promise<void>;
}) {
  const L = useCopy().learn;
  const [note, setNote] = useState("");
  const [query, setQuery] = useState("");
  const [marks, setMarks] = useState<Note[]>([]);
  const load = useCallback(async () => {
    const rows = await api.learnCall("learn.annotations", { material_id: props.reader.materialId });
    setMarks(Array.isArray(rows) ? rows : []);
  }, [props.reader.materialId]);
  useEffect(() => { void load(); }, [load]);
  const pageMarks = marks.filter((item) => !item.page || item.page === props.reader.page);
  return (
    <LearnShell
      title={props.reader.title}
      subtitle={fill(L.pageOf, { page: props.reader.page, pages: props.reader.pages })}
      back={{ label: L.back, onClick: props.onBack }}
      action={(
        <div className="flex items-center gap-1">
          <Button size="icon" variant="ghost" aria-label={L.prevPage} disabled={props.reader.page <= 1} onClick={() => props.onPage(props.reader.page - 1)}><ChevronLeft /></Button>
          <Button size="icon" variant="ghost" aria-label={L.nextPage} disabled={props.reader.page >= props.reader.pages} onClick={() => props.onPage(props.reader.page + 1)}><ChevronRight /></Button>
        </div>
      )}
    >
      <div className="mx-auto flex max-w-[680px] flex-col gap-5 px-8 py-6" data-testid="learn-reader">
        <form className="flex items-center gap-2" onSubmit={(e) => { e.preventDefault(); void props.onFind(query); }}>
          <Input className="min-w-0 flex-1" value={query} placeholder={L.searchPh} onChange={(e) => setQuery(e.target.value)} aria-label={L.searchPh} />
          <Button size="sm" type="submit" variant="ghost" className="shrink-0 whitespace-nowrap">{L.search}</Button>
        </form>
        <article className="whitespace-pre-wrap text-[14px] leading-[1.7]">
          <Highlighted text={props.reader.text || L.noHits} quote={props.reader.highlight} />
        </article>
        {pageMarks.length > 0 ? (
          <section>
            <h3 className="text-[13px] text-muted">{L.margin}</h3>
            <ul className="mt-2">
              {pageMarks.map((item) => (
                <li key={item.id} className="border-b border-border/60 py-2 text-[13px]">
                  {item.quote ? <p className="text-muted">{item.quote}</p> : null}
                  {item.note ? <p className={item.quote ? "mt-1" : ""}>{item.note}</p> : null}
                </li>
              ))}
            </ul>
          </section>
        ) : null}
        <form className="flex items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (!note.trim()) return; void props.onNote(note.trim(), props.reader.highlight).then(() => { setNote(""); void load(); }); }}>
          <div className="min-w-0 flex-1"><Field label={L.annotatePh}><Input value={note} onChange={(e) => setNote(e.target.value)} /></Field></div>
          <Button size="sm" type="submit" variant="ghost" className="shrink-0 whitespace-nowrap">{L.saveNote}</Button>
        </form>
      </div>
    </LearnShell>
  );
}

function Highlighted(props: { text: string; quote: string }) {
  const quote = props.quote.trim();
  const at = quote ? props.text.indexOf(quote) : -1;
  if (at < 0) return <>{props.text}</>;
  return (
    <>
      {props.text.slice(0, at)}
      <mark className="bg-accent/15 text-foreground">{props.text.slice(at, at + quote.length)}</mark>
      {props.text.slice(at + quote.length)}
    </>
  );
}

function PracticePane(props: {
  cards: Card[];
  desk: Desk;
  focusId: string;
  onReload: () => Promise<void>;
  onFail: (e: unknown) => void;
  onLeave: () => void;
}) {
  const L = useCopy().learn;
  const [stem, setStem] = useState("");
  const [answer, setAnswer] = useState("");
  const [options, setOptions] = useState("");
  const [qtype, setQtype] = useState("short");
  const [guess, setGuess] = useState("");
  const [reveal, setReveal] = useState<{ correct?: boolean; answer?: string; explanation?: string } | null>(null);
  const [session, setSession] = useState(Boolean(props.focusId));
  const [adding, setAdding] = useState(false);
  useEffect(() => { if (props.focusId) setSession(true); }, [props.focusId]);
  const ordered = props.focusId
    ? [...props.cards.filter((card) => card.id === props.focusId), ...props.cards.filter((card) => card.id !== props.focusId)]
    : props.cards;
  const card = ordered[0];
  const objective = card && card.qtype !== "short" && card.qtype !== "fill";
  useEffect(() => { setGuess(""); setReveal(null); }, [card?.id]);
  const dueN = typeof props.desk.due === "number" ? props.desk.due : ordered.length;
  const reviewed = props.desk.reviewed || 0;
  const done = !card && reviewed > 0;
  const leave = () => { setSession(false); props.onLeave(); };
  const importForm = adding ? (
    <form className="flex flex-col gap-3" onSubmit={(e) => {
      e.preventDefault();
      if (!stem.trim()) return;
      const opts = options.split("\n").map((line) => line.trim()).filter(Boolean);
      void api.learnCall("learn.questions.import", { stem, answer, qtype, options: qtype === "choice" ? opts : [] }).then(async () => {
        setStem("");
        setAnswer("");
        setOptions("");
        setAdding(false);
        await props.onReload();
      }).catch(props.onFail);
    }}>
      <LearnSegment
        label={L.bank}
        value={qtype}
        onChange={setQtype}
        options={[
          { id: "short", label: L.shortType },
          { id: "choice", label: L.choice },
        ]}
      />
      <Field label={L.stemPh}><Input value={stem} onChange={(e) => setStem(e.target.value)} /></Field>
      <Field label={L.answerPh}><Input value={answer} onChange={(e) => setAnswer(e.target.value)} /></Field>
      {qtype === "choice" ? (
        <Field label={L.optionsPh}>
          <textarea className="min-h-20 w-full rounded-lg border border-border/60 bg-transparent px-3 py-2 text-[13px]" value={options} onChange={(e) => setOptions(e.target.value)} />
        </Field>
      ) : null}
      <div><Button size="sm" type="submit" className="whitespace-nowrap">{L.addCard}</Button></div>
    </form>
  ) : null;
  if (!session) {
    return (
      <LearnShell
        title={L.practice}
        subtitle={L.wayPracticeHint}
        action={<Button size="sm" variant="ghost" className="whitespace-nowrap" aria-expanded={adding} onClick={() => setAdding((v) => !v)}>{adding ? L.hideCard : L.addCard}</Button>}
      >
        <div className="mx-auto flex max-w-[640px] flex-col gap-6 px-8 py-6">
          <section className="rounded-xl border border-border/60 p-5">
            <div className="flex items-center justify-between gap-4">
              <div className="min-w-0">
                <p className="text-[21px] font-semibold tabular-nums tracking-[-0.03em]">{dueN}</p>
                <p className="mt-1 text-[13px] text-muted">{L.due}</p>
              </div>
              <Button className="shrink-0 whitespace-nowrap" disabled={ordered.length === 0} onClick={() => setSession(true)}>{L.startReview}</Button>
            </div>
            <p className="mt-5 border-t border-border/60 pt-4 text-[13px] tabular-nums text-muted">{fill(L.reviewedToday, { n: reviewed })}</p>
          </section>
          {dueN === 0 ? <EmptyState title={done ? L.doneToday : L.emptyBank} body={done ? L.doneTodayHint : L.emptyBankHint} /> : null}
          {importForm}
        </div>
      </LearnShell>
    );
  }
  const progress = reviewed + dueN > 0 ? (reviewed / (reviewed + dueN)) * 100 : 0;
  return (
    <LearnShell
      title={L.practice}
      subtitle={fill(L.dueCount, { n: dueN })}
      back={{ label: L.backToPractice, onClick: leave }}
    >
      <div className="mx-auto flex max-w-[640px] flex-col gap-5 px-8 py-6">
        <ProgressHairline value={progress} />
        {!card ? <EmptyState title={done ? L.doneToday : L.emptyBank} body={done ? L.doneTodayHint : L.emptyBankHint} /> : (
          <article data-testid="learn-practice-card" className="flex flex-col gap-4">
            <p className="text-[18px] font-semibold leading-snug tracking-[-0.03em]">{card.stem}</p>
            {card.streak ? <p className="tabular-nums text-[12px] text-muted">{fill(L.streak, { n: card.streak })}</p> : null}
            {objective ? (
              <form className="flex flex-col gap-3" onSubmit={(e) => {
                e.preventDefault();
                void api.learnCall("learn.practice.grade", { id: card.id, answer: guess }).then((res) => {
                  setReveal({ correct: !!res?.correct, answer: res?.reveal?.answer, explanation: res?.reveal?.explanation });
                }).catch(props.onFail);
              }}>
                {card.options && card.options.length > 0 ? (
                  <div className="flex flex-col gap-1">
                    {card.options.map((option) => (
                      <button key={option} type="button" aria-pressed={guess === option} onClick={() => setGuess(option)} className={cn("rounded-lg border border-border/60 px-3 py-2 text-left text-[13px]", guess === option ? "bg-accent/10 text-accent" : "hover:bg-lift")}>
                        {option}
                      </button>
                    ))}
                  </div>
                ) : (
                  <Field label={L.answerLabel}><Input value={guess} onChange={(e) => setGuess(e.target.value)} /></Field>
                )}
                <div><Button size="sm" type="submit" className="whitespace-nowrap">{L.check}</Button></div>
              </form>
            ) : (
              <div className="flex flex-col gap-2">
                <p className="text-[13px] text-muted">{L.rate}</p>
                <div className="grid grid-cols-4 gap-2">
                  {(["again", "hard", "good", "easy"] as const).map((rating) => (
                    <Button key={rating} variant="outline" className="w-full whitespace-nowrap px-1 text-[12px]" onClick={() => {
                      void api.learnCall("learn.practice.rate", { id: card.id, rating }).then((res) => {
                        setReveal({ answer: res?.reveal?.answer, explanation: res?.reveal?.explanation });
                      }).catch(props.onFail);
                    }}>{L[rating]}</Button>
                  ))}
                </div>
              </div>
            )}
            {reveal ? (
              <div className="border-t border-border/60 pt-3 text-[13px]">
                {reveal.correct !== undefined ? <p>{reveal.correct ? L.correct : L.incorrect}</p> : null}
                {reveal.answer ? <p className="mt-1 text-muted">{reveal.answer}</p> : null}
                {reveal.explanation ? <p className="mt-1 text-muted">{reveal.explanation}</p> : null}
                <Button size="sm" className="mt-3 whitespace-nowrap" onClick={() => { setReveal(null); setGuess(""); void props.onReload(); }}>{L.nextCard}</Button>
              </div>
            ) : null}
          </article>
        )}
      </div>
    </LearnShell>
  );
}

function tutorBrief(L: ReturnType<typeof useCopy>["learn"], input: {
  lane: string;
  topic: string;
  pathName: string;
  stepName: string;
  readerTitle: string;
  bookTitle: string;
  cardStem: string;
  shelf: Shelf;
  libraryName: string;
}): { subject: string; about: string; prompts: TutorPrompt[] } {
  if (input.lane === "today") {
    return {
      subject: input.topic || L.today,
      about: L.tutorToday,
      prompts: [{ id: "next", label: L.askNext, text: L.askNext }],
    };
  }
  if (input.lane === "study") {
    const name = input.stepName || input.pathName;
    return {
      subject: input.pathName || L.study,
      about: L.tutorStudy,
      prompts: [
        { id: "explain", label: L.askExplain, text: name ? `${L.askExplain}: ${name}` : L.askExplain },
        { id: "quiz", label: L.askQuiz, text: name ? `${L.askQuiz}: ${name}` : L.askQuiz },
      ],
    };
  }
  if (input.lane === "practice") {
    const stem = input.cardStem.trim();
    return {
      subject: stem ? stem : L.practice,
      about: L.tutorPractice,
      prompts: [{ id: "explain", label: L.askExplain, text: stem ? `${L.askExplain}: ${stem}` : L.askExplain }],
    };
  }
  if (input.shelf === "notes") {
    return { subject: L.notesTab, about: L.tutorNotes, prompts: [] };
  }
  if (input.shelf === "materials" && input.readerTitle) {
    return {
      subject: input.readerTitle,
      about: L.tutorRead,
      prompts: [{ id: "page", label: L.askExplain, text: `${L.askExplain}: ${input.readerTitle}` }],
    };
  }
  if (input.shelf === "books" && input.bookTitle) {
    return {
      subject: input.bookTitle,
      about: L.tutorBook,
      prompts: [{ id: "book", label: L.askExplain, text: `${L.askExplain}: ${input.bookTitle}` }],
    };
  }
  return {
    subject: input.libraryName || L.library,
    about: L.tutorLibrary,
    prompts: [{ id: "next", label: L.askNext, text: L.askNext }],
  };
}
