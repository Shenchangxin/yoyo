/**
 * 拥有：学习表面的壳：四条泳道各画哪一页、文库里的三个标签、导师栏，以及当前这一页该由哪段导师对话来接。
 * 不拥有：各页的内容，那在 home、mastery、library、books、practice 各自的目录里。笔记编辑仍交给 PagesWorkspace。
 * 每条泳道、每条路径、每份材料和每本课本用自己的学习对话，不占用会话模块。
 */

import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { useCopy } from "../../lib/i18n";
import { errMessage } from "../../lib/normalize";
import { learnScopeSpec, type LearnScopeSpec } from "../../lib/learn-scope";
import type { PathView } from "../../lib/learn-api";
import * as learn from "../../lib/learn-api";
import type { BookSummary } from "../../lib/learn-books";
import type { Material } from "../../lib/learn-library";
import { onLearnSay } from "../../lib/learn-events";
import { useUI } from "../../lib/store";
import { PagesWorkspace } from "../pages/PagesWorkspace";
import { LearnRailNav } from "./LearnRailNav";
import { LearnShell, LearnTabs } from "./LearnShell";
import { LearnTutor, type TutorPrompt } from "./LearnTutor";
import { BooksPane } from "./books/BooksPane";
import { HomePane } from "./home/HomePane";
import { LibraryPane } from "./library/LibraryPane";
import { Reader } from "./library/Reader";
import { MasteryPane } from "./mastery/MasteryPane";
import { PracticePane } from "./practice/PracticePane";

type Shelf = "materials" | "books" | "notes";
type Reading = { libraryId: string; documentId: string; start?: { locator: number; quote?: string } };

/** readTutorOpen 读出上次导师栏是开着还是收着。读不到时默认打开。 */
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

/**
 * LearnWorkspace 管学习计划、掌握、文库和练习四条泳道。侧栏收起时泳道画在内容左侧。
 * sessionId 是当前这一页的导师对话。onEnsureScope 在这一页换了范围时调用，调用方找出或新建对应的对话。
 * onTutorSend 把一句话发给导师，onTutorPrompt 把一句话放进导师的输入框。chat 是导师对话本身，画在内容右侧。
 */
export function LearnWorkspace(props: {
  sessionId?: string;
  boundPageId?: string;
  navInRail?: boolean;
  onOpenInChat?: (pageId: string) => void;
  onUnbindChat?: () => void;
  onEnsureScope?: (spec: LearnScopeSpec) => void | Promise<unknown>;
  onTutorPrompt?: (text: string) => void;
  onTutorSend?: (text: string) => void;
  tutorQuiet?: boolean;
  chat?: ReactNode;
}) {
  const L = useCopy().learn;
  const lane = useUI((s) => s.learnLane);
  const [shelf, setShelf] = useState<Shelf>("materials");
  const [err, setErr] = useState("");
  const [active, setActive] = useState<PathView | null>(null);
  const [libId, setLibId] = useState("");
  const [reading, setReading] = useState<Reading | null>(null);
  const [material, setMaterial] = useState<Material | null>(null);
  const [bookId, setBookId] = useState("");
  const [book, setBook] = useState<BookSummary | null>(null);
  const [cardStem, setCardStem] = useState("");
  const [focusId, setFocusId] = useState("");
  const [startPractice, setStartPractice] = useState(false);
  const [tutorOpen, setTutorOpen] = useState(readTutorOpen);

  const goLane = (next: "today" | "study" | "library" | "practice") => useUI.getState().setLearnLane(next);

  const inReader = lane === "library" && shelf === "materials" && !!reading && !!material;
  const inBook = lane === "library" && shelf === "books" && !!book;
  const scope = learnScopeSpec({
    lane,
    title: lane === "today" ? L.home.title : lane === "practice" ? L.drill.title : lane === "study" ? L.mastery.title : shelf === "notes" ? L.notesTab : L.library,
    pathId: lane === "study" ? active?.id : undefined,
    pathName: lane === "study" ? active?.name : undefined,
    libraryId: lane === "study" ? active?.library_id : lane !== "library" ? undefined : shelf === "notes" ? "notes" : inBook ? book?.library_id || undefined : shelf === "books" ? undefined : libId || undefined,
    materialId: inReader ? material?.id : undefined,
    materialTitle: inReader ? material?.title : undefined,
    bookId: inBook ? book?.id : undefined,
    bookTitle: inBook ? book?.title : undefined,
    shelf: lane === "library" ? shelf : undefined,
  });
  const tutor = tutorBrief(L, {
    lane,
    pathName: active?.name || "",
    stepName: active?.next?.name || "",
    readerTitle: inReader ? material?.title || "" : "",
    bookTitle: inBook ? book?.title || "" : "",
    cardStem: lane === "practice" ? cardStem : "",
    shelf,
  });

  /** 打开或收起导师栏，并记住这次的选择。 */
  const setTutor = (open: boolean) => {
    setTutorOpen(open);
    try {
      localStorage.setItem("yoyo-learn-tutor", open ? "1" : "0");
    } catch {
      /* 写不进时只留在这一次 */
    }
  };
  const scopeKey = [scope.scope, scope.pathId || "", scope.libraryId || "", scope.materialId || "", scope.bookId || "", scope.title || ""].join("|");
  useLayoutEffect(() => {
    void Promise.resolve(props.onEnsureScope?.(scope)).catch((e) => setErr(errMessage(e)));
  }, [scopeKey]);

  /** 把一句话发给当前这段导师对话，并把导师栏打开。 */
  const sayToTutor = (text: string) => {
    if (!text.trim()) return;
    setTutor(true);
    props.onTutorSend?.(text);
  };
  const sayRef = useRef(sayToTutor);
  sayRef.current = sayToTutor;
  const sessionRef = useRef(props.sessionId);
  sessionRef.current = props.sessionId;
  useEffect(() => onLearnSay(/** 题卡在导师对话里，作答后把学习者的话发回同一段对话。别的对话里的卡片不管。 */ (say) => {
    if (say.sessionId && say.sessionId !== sessionRef.current) return;
    sayRef.current(say.text);
  }), []);

  const shelfTabs = (
    <LearnTabs
      label={L.library}
      value={shelf}
      onChange={(id) => {
        setShelf(id as Shelf);
        if (id !== "materials") setReading(null);
      }}
      items={[
        { id: "materials", label: L.materials },
        { id: "books", label: L.books },
        { id: "notes", label: L.notesTab },
      ]}
    />
  );

  return (
    <div className="flex h-full min-h-0">
      {props.navInRail ? null : (
        <div className="w-[168px] shrink-0 border-r border-border/60 bg-sidebar pt-3">
          <LearnRailNav />
        </div>
      )}
      <div className="flex min-h-0 min-w-0 flex-1">
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          {err ? <p className="px-6 pt-4 text-ui-footnote text-danger">{err}</p> : null}
          <div className="min-h-0 flex-1">
            {lane === "today" ? (
              <HomePane
                onPractice={() => { setStartPractice(true); goLane("practice"); }}
                onPath={(id) => {
                  goLane("study");
                  void learn.path(id).then(setActive).catch((e) => setErr(errMessage(e)));
                }}
                onMaterial={(libraryId, documentId) => { setLibId(libraryId); setShelf("materials"); setReading({ libraryId, documentId }); goLane("library"); }}
                onBook={(id) => { setShelf("books"); setBookId(id); goLane("library"); }}
                onLane={(target) => {
                  if (target === "study") {
                    setActive(null);
                    goLane("study");
                  } else if (target === "practice") {
                    goLane("practice");
                  } else {
                    setReading(null);
                    if (target === "book") setBookId("");
                    setShelf(target === "book" ? "books" : "materials");
                    goLane("library");
                  }
                }}
              />
            ) : null}
            {lane === "study" ? <MasteryPane active={active} onActive={setActive} sessionId={props.sessionId} onSay={sayToTutor} /> : null}
            {lane === "library" && shelf === "materials" ? (
              reading ? (
                <Reader
                  key={`${reading.libraryId}:${reading.documentId}`}
                  libraryId={reading.libraryId}
                  documentId={reading.documentId}
                  start={reading.start}
                  sessionId={props.sessionId}
                  onBack={() => setReading(null)}
                  onSay={sayToTutor}
                  onMaterial={setMaterial}
                />
              ) : (
                <LibraryPane
                  tabs={shelfTabs}
                  title={L.library}
                  libraryId={libId}
                  onLibrary={setLibId}
                  onOpen={(libraryId, documentId, start) => setReading({ libraryId, documentId, start })}
                />
              )
            ) : null}
            {lane === "library" && shelf === "books" ? (
              <BooksPane tabs={shelfTabs} title={L.library} libraryId={libId} sessionId={props.sessionId} openId={bookId} onOpen={setBookId} onBook={setBook} />
            ) : null}
            {lane === "library" && shelf === "notes" ? (
              <LearnShell title={L.library} subtitle={L.notesHint} tabs={shelfTabs} scroll={false}>
                <div className="min-h-0 flex-1">
                  <div className="h-full min-h-0">
                    <PagesWorkspace sessionId={props.sessionId} boundPageId={props.boundPageId} onOpenInChat={props.onOpenInChat} onUnbindChat={props.onUnbindChat} />
                  </div>
                </div>
              </LearnShell>
            ) : null}
            {lane === "practice" ? (
              <PracticePane
                sessionId={props.sessionId}
                focusId={focusId}
                autoStart={startPractice}
                onAutoStarted={() => setStartPractice(false)}
                onFocusDone={() => setFocusId("")}
                onSay={sayToTutor}
                onCard={setCardStem}
              />
            ) : null}
          </div>
        </div>
        {props.chat ? (
          <LearnTutor open={tutorOpen} subject={tutor.subject} about={tutor.about} prompts={tutor.prompts} quiet={props.tutorQuiet} onOpen={setTutor} onPrompt={props.onTutorPrompt}>
            {props.chat}
          </LearnTutor>
        ) : null}
      </div>
    </div>
  );
}

/**
 * tutorBrief 给导师栏的页头定下三样东西：标题、一句说明和几个现成的问题。它们跟着当前这一页走。
 * input 里的各项是当前泳道和打开着的路径、材料、课本、题目。
 */
function tutorBrief(L: ReturnType<typeof useCopy>["learn"], input: {
  lane: string;
  pathName: string;
  stepName: string;
  readerTitle: string;
  bookTitle: string;
  cardStem: string;
  shelf: Shelf;
}): { subject: string; about: string; prompts: TutorPrompt[] } {
  if (input.lane === "today") {
    return { subject: L.home.title, about: L.home.tutorAbout, prompts: [{ id: "next", label: L.home.askNext, text: L.home.askNext }] };
  }
  if (input.lane === "study") {
    const name = input.stepName || input.pathName;
    return {
      subject: input.pathName || L.mastery.title,
      about: L.mastery.tutorAbout,
      prompts: [
        { id: "explain", label: L.askExplain, text: name ? `${L.askExplain}: ${name}` : L.askExplain },
        { id: "quiz", label: L.askQuiz, text: name ? `${L.askQuiz}: ${name}` : L.askQuiz },
      ],
    };
  }
  if (input.lane === "practice") {
    return {
      subject: input.cardStem.trim() || L.drill.title,
      about: L.drill.tutorAbout,
      prompts: [{ id: "mistakes", label: L.drill.tutorMistakes, text: L.drill.tutorMistakes }],
    };
  }
  if (input.shelf === "notes") {
    return { subject: L.notesTab, about: L.tutorNotes, prompts: [] };
  }
  if (input.shelf === "materials" && input.readerTitle) {
    return {
      subject: input.readerTitle,
      about: L.shelf.tutorAbout,
      prompts: [
        { id: "page", label: L.shelf.askPage, text: L.shelf.askPage },
        { id: "summary", label: L.shelf.askSummary, text: L.shelf.askSummary },
      ],
    };
  }
  if (input.shelf === "books" && input.bookTitle) {
    return {
      subject: input.bookTitle,
      about: L.tome.tutorAbout,
      prompts: [
        { id: "chapter", label: L.tome.askChapter, text: L.tome.askChapter },
        { id: "quiz", label: L.tome.askQuiz, text: L.tome.askQuiz },
      ],
    };
  }
  return { subject: L.library, about: L.tutorLibrary, prompts: [{ id: "next", label: L.home.askNext, text: L.home.askNext }] };
}
