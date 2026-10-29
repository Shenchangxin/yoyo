/** 学习表面的四条泳道。笔记编辑仍交给 PagesWorkspace。每一条泳道和每条路径用自己的学习对话，不占用会话模块。 */

import { useCallback, useEffect, useLayoutEffect, useState, type ReactNode } from "react";
import { getLocale, useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { errMessage } from "../../lib/normalize";
import { learnScopeSpec, type LearnScopeSpec } from "../../lib/learn-scope";
import { useUI } from "../../lib/store";
import { LearnRailNav } from "./LearnRailNav";
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
                  onShelf={setShelf}
                  libs={libs}
                  libId={libId}
                  docs={docs}
                  hits={hits}
                  hitMsg={hitMsg}
                  books={books}
                  openBook={openBook}
                  notes={shelf === "notes" || libId === "notes" ? (
                    <div className="flex h-full min-h-0 flex-col">
                      <p className="px-6 pt-4 text-[12px] text-muted">{L.notesHint}</p>
                      <div className="min-h-0 flex-1">
                        <PagesWorkspace
                          sessionId={props.sessionId}
                          boundPageId={props.boundPageId}
                          onOpenInChat={props.onOpenInChat}
                          onUnbindChat={props.onUnbindChat}
                        />
                      </div>
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
              <PracticePane cards={cards} desk={desk} focusId={focusId} onChange={setCards} onReload={() => run(loadDue)} onFail={fail} />
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

function TodayPane(props: {
  data: any;
  onSave: (topic: string, why: string, next: string) => Promise<void>;
  onRecord: (insight: string) => Promise<void>;
  onOpenPath: (id: string) => void;
  onPractice: (id: string) => void;
  onOpenBook: (id: string, libraryId?: string) => void;
}) {
  const L = useCopy().learn;
  const j = props.data?.journal || {};
  const [topic, setTopic] = useState(j.topic || "");
  const [why, setWhy] = useState(j.why || "");
  const [next, setNext] = useState(j.next_focus || "");
  const [insight, setInsight] = useState("");
  useEffect(() => {
    setTopic(j.topic || "");
    setWhy(j.why || "");
    setNext(j.next_focus || "");
  }, [j.topic, j.why, j.next_focus]);
  const due = Array.isArray(props.data?.due) ? props.data.due : [];
  const paths = Array.isArray(props.data?.paths) ? props.data.paths : [];
  const books = Array.isArray(props.data?.books) ? props.data.books : [];
  const records = Array.isArray(j.records) ? j.records.slice(0, 3) : [];
  const quiet = due.length === 0 && paths.length === 0 && books.length === 0;
  const [editing, setEditing] = useState(false);
  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto flex max-w-[640px] flex-col gap-8 px-8 py-10">
        <section>
          <h2 className="text-[22px] font-semibold tracking-[-0.03em]">{L.now}</h2>
          {quiet ? <EmptyState title={L.nowEmpty} body={L.nowEmptyHint} /> : (
            <ul className="mt-4 flex flex-col">
              {due.slice(0, 5).map((card: Card) => (
                <li key={card.id}>
                  <button type="button" data-testid="learn-due" className="flex w-full items-baseline justify-between gap-3 border-b border-border/60 py-3 text-left" onClick={() => props.onPractice(card.id)}>
                    <span className="text-[15px]">{card.stem}</span>
                    <span className="shrink-0 text-[12px] text-muted">{L.practice}</span>
                  </button>
                </li>
              ))}
              {paths.map((path: Path) => (
                <li key={path.id}>
                  <button type="button" className="flex w-full items-baseline justify-between gap-3 border-b border-border/60 py-3 text-left" onClick={() => props.onOpenPath(path.id)}>
                    <span className="text-[15px]">{path.name}</span>
                    <span className="tabular-nums text-[12px] text-muted">{path.counts?.mastered || 0}/{path.counts?.total || 0}</span>
                  </button>
                </li>
              ))}
              {books.map((book: Book) => (
                <li key={book.id}>
                  <button type="button" className="flex w-full items-baseline justify-between gap-3 border-b border-border/60 py-3 text-left" onClick={() => props.onOpenBook(book.id, book.library_id)}>
                    <span className="text-[15px]">{book.title}</span>
                    <span className="text-[12px] text-muted">{book.status === "ready" || book.status === "paused" ? L.readOn : bookStatus(book.status)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
        <section className="flex flex-col gap-3">
          <div className="flex items-baseline justify-between gap-3">
            <h2 className="text-[13px] text-muted">{L.mission}</h2>
            <button type="button" className="text-[12px] text-muted hover:text-foreground" onClick={() => setEditing((v) => !v)}>{L.editMission}</button>
          </div>
          {!editing && topic ? <p className="text-[15px]">{topic}</p> : null}
          {editing ? (
            <div className="flex flex-col gap-2">
              <Input value={topic} placeholder={L.missionPh} onChange={(e) => setTopic(e.target.value)} aria-label={L.mission} />
              <Input value={why} placeholder={L.whyPh} onChange={(e) => setWhy(e.target.value)} aria-label={L.why} />
              <Input value={next} placeholder={L.nextPh} onChange={(e) => setNext(e.target.value)} aria-label={L.next} />
              <div><Button size="sm" onClick={() => { void props.onSave(topic, why, next); setEditing(false); }}>{L.saveMission}</Button></div>
            </div>
          ) : null}
        </section>
        <section className="flex flex-col gap-2">
          {records.map((row: { id: string; insight: string }) => (
            <p key={row.id} className="text-[13px] text-muted">{row.insight}</p>
          ))}
          <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); if (!insight.trim()) return; void props.onRecord(insight.trim()); setInsight(""); }}>
            <Input value={insight} placeholder={L.recordPh} onChange={(e) => setInsight(e.target.value)} aria-label={L.record} />
            <Button size="sm" type="submit" variant="ghost">{L.saveRecord}</Button>
          </form>
        </section>
      </div>
    </div>
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
  const [point, setPoint] = useState("");
  const [kind, setKind] = useState<string>("concept");
  const [draft, setDraft] = useState<{ name: string; kind: string }[]>([]);
  const [guess, setGuess] = useState("");
  const [reveal, setReveal] = useState<{ correct?: boolean; answer?: string } | null>(null);
  useEffect(() => { setGuess(""); setReveal(null); }, [props.active?.id, props.active?.next?.question?.prompt]);
  const question = props.active?.next?.question;
  const waiting = props.active?.next?.action === "answer_pending" && question?.prompt;
  const ask = props.active && props.active.mode !== "outline" && props.active.next?.action && props.active.next.action !== "complete" && props.active.next.action !== "answer_pending";
  if (!props.active) {
    return (
      <div className="h-full overflow-auto">
        <div className="mx-auto flex max-w-[640px] flex-col gap-4 px-8 py-10">
          <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); if (!name.trim()) return; void props.onCreate(name.trim()); setName(""); }}>
            <Input value={name} placeholder={L.pathPh} onChange={(e) => setName(e.target.value)} aria-label={L.pathPh} />
          </form>
          <ul className="flex flex-col">
            {props.paths.map((path) => (
              <li key={path.id}>
                <button type="button" onClick={() => props.onOpen(path.id)} className="flex w-full items-baseline justify-between gap-3 border-b border-border/60 py-3 text-left text-[15px] hover:text-foreground">
                  <span>{path.name}</span>
                  <span className="tabular-nums text-[12px] text-muted">{path.counts?.mastered || 0}/{path.counts?.total || 0}</span>
                </button>
              </li>
            ))}
          </ul>
          {props.paths.length === 0 ? <EmptyState title={L.pathsEmpty} body={L.pathsEmptyHint} /> : null}
        </div>
      </div>
    );
  }
  const current = props.active;
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-12 shrink-0 items-center gap-3 px-6">
        <button type="button" className="shrink-0 text-[13px] text-muted hover:text-foreground" onClick={props.onClose}>{L.allPaths}</button>
        <h2 className="min-w-0 flex-1 truncate text-[15px] font-medium tracking-[-0.02em]">{current.name}</h2>
        <span className="tabular-nums text-[12px] text-muted">{current.counts?.mastered || 0}/{current.counts?.total || 0}</span>
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-6 pb-10">
        <div className="mx-auto flex max-w-[640px] flex-col gap-6">
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[13px]">
              {(["outline", "study", "review"] as const).map((mode) => (
                <button key={mode} type="button" aria-pressed={current.mode === mode} className={cn(current.mode === mode ? "text-foreground" : "text-muted hover:text-foreground")} onClick={() => { void props.onMode(mode); }}>
                  {mode === "outline" ? L.outline : mode === "study" ? L.studyMode : L.review}
                </button>
              ))}
              <span className="text-border" aria-hidden>·</span>
              <button type="button" aria-pressed={current.teaching !== "teach_first"} className={cn(current.teaching !== "teach_first" ? "text-foreground" : "text-muted hover:text-foreground")} onClick={() => { void props.onProfile("probe_first"); }}>{L.probeFirst}</button>
              <button type="button" aria-pressed={current.teaching === "teach_first"} className={cn(current.teaching === "teach_first" ? "text-foreground" : "text-muted hover:text-foreground")} onClick={() => { void props.onProfile("teach_first"); }}>{L.teachFirst}</button>
            </div>
            <section data-testid="learn-next" className="flex flex-col gap-3 border-b border-border/60 pb-5">
              <p className="text-[13px] text-muted">{stepCopy(current.next?.action, L, current.mode)}</p>
              {current.next?.name ? <p className="text-[22px] font-semibold leading-snug tracking-[-0.03em]">{current.next.name}</p> : null}
              {waiting ? (
                <div className="flex flex-col gap-3">
                  <p className="text-[16px] font-semibold leading-snug tracking-[-0.02em]">{question?.prompt}</p>
                  {question?.options && question.options.length > 0 ? (
                    <div className="flex flex-col gap-1">
                      {question.options.map((option) => (
                        <button key={option} type="button" aria-pressed={guess === option} onClick={() => setGuess(option)} className={cn("rounded-lg border border-border/60 px-3 py-2 text-left text-[13px]", guess === option ? "bg-accent/10 text-accent" : "hover:bg-lift")}>
                          {option}
                        </button>
                      ))}
                    </div>
                  ) : (
                    <Input value={guess} placeholder={L.answerLabel} onChange={(e) => setGuess(e.target.value)} aria-label={L.answerLabel} />
                  )}
                  <div><Button size="sm" onClick={() => { void props.onGrade(guess).then((res) => { if (!res) return; setReveal({ correct: !!res?.correct, answer: res?.reveal?.answer }); }); }}>{L.submitAnswer}</Button></div>
                </div>
              ) : null}
              {reveal ? (
                <div className="text-[13px]">
                  <p>{reveal.correct ? L.correct : L.incorrect}</p>
                  {reveal.answer ? <p className="mt-1 text-muted">{reveal.answer}</p> : null}
                  <Button size="sm" className="mt-3" variant="ghost" onClick={() => setReveal(null)}>{L.continue}</Button>
                </div>
              ) : null}
              {ask ? <button type="button" className="w-fit text-[13px] text-muted hover:text-foreground" onClick={props.onAsk}>{L.askTutor}</button> : null}
            </section>
            {(current.modules || []).map((mod) => (
              <section key={mod.id}>
                <h3 className="text-[13px] text-muted">{mod.name}</h3>
                <ul className="mt-2 flex flex-col">
                  {(mod.points || []).map((pt) => (
                    <li key={pt.id} className="flex items-baseline justify-between gap-3 border-b border-border/60 py-2.5">
                      <div className="min-w-0">
                        <p className="truncate text-[14px]">{pt.name}</p>
                        <p className="mt-0.5 text-[12px] text-muted">{kindName(pt.kind)} · {pt.status === "mastered" ? L.mastered : pt.status === "learning" ? L.learning : L.fresh} · {Math.round((pt.mastery || 0) * 100)}%</p>
                      </div>
                      {pt.status !== "mastered" ? <button type="button" className="shrink-0 text-[12px] text-muted hover:text-foreground" onClick={() => { void props.onClaim(pt.id); }}>{L.claim}</button> : null}
                    </li>
                  ))}
                </ul>
              </section>
            ))}
            {current.mode === "outline" ? (
              <form className="flex flex-col gap-2 border-t border-border/60 pt-4" onSubmit={(e) => { e.preventDefault(); if (!point.trim()) return; setDraft((d) => [...d, { name: point.trim(), kind }]); setPoint(""); }}>
                <div className="flex gap-2">
                  <Input value={point} placeholder={L.pointPh} onChange={(e) => setPoint(e.target.value)} aria-label={L.pointPh} />
                  <select className="rounded-lg border border-border/60 bg-transparent px-2 text-[13px]" value={kind} onChange={(e) => setKind(e.target.value)} aria-label={L.addPoint}>
                    {kinds.map((k) => <option key={k} value={k}>{kindName(k)}</option>)}
                  </select>
                  <Button size="sm" type="submit" variant="ghost">{L.addPoint}</Button>
                </div>
                {draft.length > 0 ? (
                  <Button size="sm" onClick={() => {
                    const mods = (props.active?.modules || []).map((mod) => ({ name: mod.name, points: (mod.points || []).map((pt) => ({ name: pt.name, kind: pt.kind })) }));
                    if (mods.length === 0) mods.push({ name: props.active?.name || L.outline, points: [] });
                    mods[mods.length - 1].points.push(...draft);
                    void props.onOutline(mods);
                    setDraft([]);
                  }}>{L.saveOutline}</Button>
                ) : null}
              </form>
            ) : null}
        </div>
      </div>
    </div>
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
}) {
  const L = useCopy().learn;
  const shelves: { id: Shelf; label: string }[] = [
    { id: "materials", label: L.materials },
    { id: "books", label: L.books },
    { id: "notes", label: L.notesTab },
  ];
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex flex-wrap items-center gap-1 border-b border-border/60 px-4 py-2">
        {shelves.map((item) => (
          <button key={item.id} type="button" aria-pressed={props.shelf === item.id} onClick={() => props.onShelf(item.id)} className={cn("rounded-lg px-2.5 py-1.5 text-[13px]", props.shelf === item.id ? "bg-accent/10 text-accent" : "text-muted hover:bg-lift hover:text-foreground")}>
            {item.label}
          </button>
        ))}
      </div>
            {props.shelf === "notes" ? <div className="min-h-0 flex-1">{props.notes}</div> : null}
      {props.shelf === "materials" ? (
        <Materials
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
      ) : null}
      {props.shelf === "books" ? (
        <BookPane
          libId={props.libId}
          books={props.books}
          openBook={props.openBook}
          onBook={props.onBook}
          onSelect={props.onSelectBook}
          onCompile={props.onCompile}
          onPage={props.onPage}
        />
      ) : null}
    </div>
  );
}

function Materials(props: {
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
  const [manage, setManage] = useState(false);
  const current = props.libs.find((lib) => lib.id === props.libId);
  const rows: Hit[] = props.hits || props.docs.map((doc) => ({ document_id: doc.id, title: doc.title || doc.rel_path, page: 0, excerpt: doc.status, source_hash: "" }));
  return (
    <div className="min-h-0 flex-1 overflow-auto">
      <div className="mx-auto flex max-w-[640px] flex-col gap-4 px-8 py-8">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          {props.libs.map((lib) => (
            <button key={lib.id} type="button" aria-pressed={props.libId === lib.id} onClick={() => props.onPick(lib.id)} className={cn("text-[13px]", props.libId === lib.id ? "text-foreground" : "text-muted hover:text-foreground")}>
              {lib.name}
              <span className="ml-1.5 tabular-nums text-[11px]">{L.files.replace("{n}", String(lib.documents || 0))}</span>
            </button>
          ))}
          <button type="button" className="text-[12px] text-muted hover:text-foreground" aria-expanded={manage} onClick={() => setManage((v) => !v)}>{manage ? L.hideManage : L.manage}</button>
        </div>
        {manage ? (
          <div className="flex flex-col gap-2">
            <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); if (!name.trim()) return; void props.onCreate(name.trim(), "indexed", ""); setName(""); }}>
              <Input value={name} placeholder={L.libraryPh} onChange={(e) => setName(e.target.value)} aria-label={L.libraryPh} />
              <Button size="sm" type="submit" variant="ghost">{L.newLibrary}</Button>
            </form>
            {props.libId !== "notes" ? (
              <>
                <div className="flex flex-wrap items-center gap-2">
                  <Button size="sm" variant="ghost" onClick={() => { void api.pickFiles().then((files) => { if (files.length) void props.onImport(files); }); }}>{L.importFiles}</Button>
                  <Button size="sm" variant="ghost" onClick={() => { void api.pickFolder().then((path) => { if (path) void props.onCreate(path.split(/[\\/]/).pop() || L.library, "linked", path); }); }}>{L.linkFolder}</Button>
                  <Button size="sm" variant="ghost" onClick={() => { void props.onRebuild(); }}>{current?.status === "building" ? L.indexing : L.rebuild}</Button>
                  <span className="text-[12px] text-muted">{current?.status}{current?.last_error ? " · " + current.last_error : ""}</span>
                </div>
                <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); void props.onEmbed(model); }}>
                  <Input value={model} placeholder={L.embedPh} onChange={(e) => setModel(e.target.value)} aria-label={L.embedPh} />
                </form>
              </>
            ) : null}
          </div>
        ) : null}
        {props.libId !== "notes" ? (
          <>
            <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); void props.onSearch(query); }}>
              <Input value={query} placeholder={L.searchPh} onChange={(e) => setQuery(e.target.value)} aria-label={L.searchPh} />
            </form>
            {props.hits ? <div><Button size="sm" variant="ghost" onClick={props.onClearSearch}>{L.backToFiles}</Button></div> : null}
            {props.hitMsg ? <p className="text-[13px] text-muted">{props.hitMsg}</p> : null}
            {props.hits && props.hits.length === 0 && !props.hitMsg ? <EmptyState title={L.noHits} body={L.noHitsHint} /> : null}
            {rows.length === 0 && !props.hits ? <EmptyState title={L.filesEmpty} body={L.filesEmptyHint} /> : null}
            <ul className="flex flex-col">
              {rows.map((hit) => (
                <li key={hit.document_id + hit.excerpt} className="flex items-start justify-between gap-3 border-b border-border/60 py-3">
                  <div className="min-w-0">
                    <p className="text-[14px]">{hit.title}</p>
                    <p className="mt-1 line-clamp-2 text-[12px] text-muted">{hit.excerpt}</p>
                  </div>
                  <div className="flex shrink-0 gap-1">
                    <Button size="sm" variant="ghost" onClick={() => { const doc = props.docs.find((item) => item.id === hit.document_id) || { id: hit.document_id, title: hit.title, rel_path: hit.title, status: "" }; void props.onOpenDoc(doc); }}>{L.open}</Button>
                    {props.docs.some((doc) => doc.id === hit.document_id) ? <Button size="sm" variant="ghost" onClick={() => { void props.onRemove(hit.document_id); }}>{L.remove}</Button> : null}
                  </div>
                </li>
              ))}
            </ul>
          </>
        ) : null}
      </div>
    </div>
  );
}

function BookPane(props: {
  libId: string;
  books: Book[];
  openBook: Book | null;
  onBook: (title: string) => Promise<void>;
  onSelect: (id: string) => void;
  onCompile: (id: string, proposal: string, spine: string) => Promise<void>;
  onPage: (id: string, page: number) => Promise<void>;
}) {
  const L = useCopy().learn;
  const [title, setTitle] = useState("");
  const [proposal, setProposal] = useState("");
  const [spine, setSpine] = useState("");
  const [loaded, setLoaded] = useState("");
  const [editing, setEditing] = useState(true);
  const book = props.openBook;
  useEffect(() => {
    if (!book || book.id === loaded) return;
    setLoaded(book.id);
    setProposal(book.proposal || "");
    setSpine((book.spine || []).map((item) => item.query || item.title).filter(Boolean).join("\n"));
    setEditing(book.status === "draft" || !(book.blocks || []).length);
  }, [book, loaded]);
  const mine = props.books.filter((item) => !item.library_id || item.library_id === props.libId || props.libId === "notes");
  const page = book?.page || 1;
  const pages = Math.max(1, ...(book?.blocks || []).map((block) => block.page || 1), 1);
  const blocks = (book?.blocks || []).filter((block) => block.page === page);
  return (
    <div className="min-h-0 flex-1 overflow-auto" data-testid="learn-book">
      <div className="mx-auto flex max-w-[720px] flex-col gap-5 px-6 py-6">
        <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); if (!title.trim() || props.libId === "notes") return; void props.onBook(title.trim()); setTitle(""); }}>
          <Input value={title} placeholder={L.bookPh} onChange={(e) => setTitle(e.target.value)} aria-label={L.bookPh} />
          <Button size="sm" type="submit" variant="ghost" disabled={props.libId === "notes"}>{L.newBook}</Button>
        </form>
        {mine.length === 0 && !book ? <EmptyState title={L.bookEmpty} body={L.bookEmptyHint} /> : null}
        <ul className="flex flex-col">
          {mine.map((item) => (
            <li key={item.id}>
              <button type="button" aria-pressed={book?.id === item.id} onClick={() => props.onSelect(item.id)} className={cn("flex w-full items-baseline justify-between border-b border-border/60 py-3 text-left", book?.id === item.id && "text-accent")}>
                <span className="text-[14px]">{item.title}</span>
                <span className="text-[12px] text-muted">{bookStatus(item.status)}</span>
              </button>
            </li>
          ))}
        </ul>
        {book ? (
          <article className="flex flex-col gap-4 border-t border-border/60 pt-4">
            <div className="flex items-baseline justify-between gap-3">
              <h2 className="text-[18px] font-semibold tracking-[-0.03em]">{book.title}</h2>
              <span className="tabular-nums text-[12px] text-muted">{fill(L.pageOf, { page, pages })}</span>
            </div>
            <div><Button size="sm" variant="ghost" onClick={() => setEditing((v) => !v)}>{editing ? L.hideSpine : L.editSpine}</Button></div>
            {editing ? (
              <div className="flex flex-col gap-2">
                <Input value={proposal} placeholder={L.proposalPh} onChange={(e) => setProposal(e.target.value)} aria-label={L.proposalPh} />
                <textarea className="min-h-24 w-full rounded-lg border border-border/60 bg-transparent px-3 py-2 text-[13px]" value={spine} placeholder={L.spinePh} onChange={(e) => setSpine(e.target.value)} aria-label={L.spinePh} />
                <div><Button size="sm" onClick={() => { void props.onCompile(book.id, proposal, spine); }}>{L.compile}</Button></div>
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
            <div className="flex gap-2">
              <Button size="sm" variant="ghost" disabled={page <= 1} onClick={() => { void props.onPage(book.id, page - 1); }}>{L.prevPage}</Button>
              <Button size="sm" variant="ghost" disabled={page >= pages} onClick={() => { void props.onPage(book.id, page + 1); }}>{L.nextPage}</Button>
            </div>
          </article>
        ) : null}
      </div>
    </div>
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
    <div className="h-full overflow-auto" data-testid="learn-reader">
      <div className="mx-auto flex max-w-[760px] flex-col gap-4 px-6 py-6">
        <div className="flex items-center gap-3">
          <Button size="sm" variant="ghost" onClick={props.onBack}>{L.back}</Button>
          <h2 className="min-w-0 flex-1 truncate text-[16px] font-semibold">{props.reader.title}</h2>
          <span className="tabular-nums text-[12px] text-muted">{fill(L.pageOf, { page: props.reader.page, pages: props.reader.pages })}</span>
        </div>
        <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); void props.onFind(query); }}>
          <Input value={query} placeholder={L.searchPh} onChange={(e) => setQuery(e.target.value)} aria-label={L.searchPh} />
        </form>
        <article className="whitespace-pre-wrap text-[14px] leading-[1.7]">
          <Highlighted text={props.reader.text || L.noHits} quote={props.reader.highlight} />
        </article>
        <div className="flex gap-2">
          <Button size="sm" variant="ghost" disabled={props.reader.page <= 1} onClick={() => props.onPage(props.reader.page - 1)}>{L.prevPage}</Button>
          <Button size="sm" variant="ghost" disabled={props.reader.page >= props.reader.pages} onClick={() => props.onPage(props.reader.page + 1)}>{L.nextPage}</Button>
        </div>
        {pageMarks.length > 0 ? (
          <section>
            <h3 className="text-[13px] text-muted">{L.margin}</h3>
            <ul className="mt-2 flex flex-col">
              {pageMarks.map((item) => (
                <li key={item.id} className="border-b border-border/60 py-2 text-[13px]">
                  {item.quote ? <p className="text-muted">{item.quote}</p> : null}
                  {item.note ? <p className={item.quote ? "mt-1" : ""}>{item.note}</p> : null}
                </li>
              ))}
            </ul>
          </section>
        ) : null}
        <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); if (!note.trim()) return; void props.onNote(note.trim(), props.reader.highlight).then(() => { setNote(""); void load(); }); }}>
          <Input value={note} placeholder={L.annotatePh} onChange={(e) => setNote(e.target.value)} aria-label={L.annotatePh} />
          <Button size="sm" type="submit" variant="ghost">{L.saveNote}</Button>
        </form>
      </div>
    </div>
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

function PracticePane(props: { cards: Card[]; desk: Desk; focusId: string; onChange: (cards: Card[]) => void; onReload: () => Promise<void>; onFail: (e: unknown) => void }) {
  const L = useCopy().learn;
  const [stem, setStem] = useState("");
  const [answer, setAnswer] = useState("");
  const [options, setOptions] = useState("");
  const [qtype, setQtype] = useState("short");
  const [guess, setGuess] = useState("");
  const [reveal, setReveal] = useState<{ correct?: boolean; answer?: string; explanation?: string } | null>(null);
  const ordered = props.focusId
    ? [...props.cards.filter((card) => card.id === props.focusId), ...props.cards.filter((card) => card.id !== props.focusId)]
    : props.cards;
  const card = ordered[0];
  const objective = card && card.qtype !== "short" && card.qtype !== "fill";
  useEffect(() => { setGuess(""); setReveal(null); }, [card?.id]);
  const done = !card && (props.desk.reviewed || 0) > 0;
  const [adding, setAdding] = useState(false);
  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto flex max-w-[640px] flex-col gap-6 px-6 py-8">
        <p className="flex gap-4 text-[12px] text-muted">
          <span className="tabular-nums">{fill(L.dueCount, { n: props.desk.due || ordered.length })}</span>
          <span className="tabular-nums">{fill(L.reviewedToday, { n: props.desk.reviewed || 0 })}</span>
        </p>
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
                  <Input value={guess} placeholder={L.answerLabel} onChange={(e) => setGuess(e.target.value)} aria-label={L.answerLabel} />
                )}
                <div><Button size="sm" type="submit">{L.check}</Button></div>
              </form>
            ) : (
              <div className="flex flex-wrap gap-2">
                <p className="w-full text-[13px] text-muted">{L.rate}</p>
                {(["again", "hard", "good", "easy"] as const).map((rating) => (
                  <Button key={rating} size="sm" variant="ghost" onClick={() => {
                    void api.learnCall("learn.practice.rate", { id: card.id, rating }).then((res) => {
                      setReveal({ answer: res?.reveal?.answer, explanation: res?.reveal?.explanation });
                    }).catch(props.onFail);
                  }}>{L[rating]}</Button>
                ))}
              </div>
            )}
            {reveal ? (
              <div className="border-t border-border/60 pt-3 text-[13px]">
                {reveal.correct !== undefined ? <p>{reveal.correct ? L.correct : L.incorrect}</p> : null}
                {reveal.answer ? <p className="mt-1 text-muted">{reveal.answer}</p> : null}
                {reveal.explanation ? <p className="mt-1 text-muted">{reveal.explanation}</p> : null}
                <Button size="sm" className="mt-3" variant="ghost" onClick={() => { setReveal(null); setGuess(""); void props.onReload(); }}>{L.nextCard}</Button>
              </div>
            ) : null}
          </article>
        )}
        <button type="button" className="w-fit text-[13px] text-muted hover:text-foreground" aria-expanded={adding} onClick={() => setAdding((v) => !v)}>{adding ? L.hideCard : L.addCard}</button>
        {adding ? <form className="flex flex-col gap-2" onSubmit={(e) => {
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
          <h3 className="text-[13px] text-muted">{L.bank}</h3>
          <div className="flex gap-2">
            <Button size="sm" type="button" variant={qtype === "short" ? "default" : "ghost"} aria-pressed={qtype === "short"} onClick={() => setQtype("short")}>{L.shortType}</Button>
            <Button size="sm" type="button" variant={qtype === "choice" ? "default" : "ghost"} aria-pressed={qtype === "choice"} onClick={() => setQtype("choice")}>{L.choice}</Button>
          </div>
          <Input value={stem} placeholder={L.stemPh} onChange={(e) => setStem(e.target.value)} aria-label={L.stemPh} />
          <Input value={answer} placeholder={L.answerPh} onChange={(e) => setAnswer(e.target.value)} aria-label={L.answerPh} />
          {qtype === "choice" ? <textarea className="min-h-20 w-full rounded-lg border border-border/60 bg-transparent px-3 py-2 text-[13px]" value={options} placeholder={L.optionsPh} onChange={(e) => setOptions(e.target.value)} aria-label={L.optionsPh} /> : null}
          <div><Button size="sm" type="submit" variant="ghost">{L.addCard}</Button></div>
        </form> : null}
      </div>
    </div>
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
