/**
 * 拥有：阅读器：一次读一个单元，翻页、目录、查找、书签，以及选中文字后标出来、写笔记或问导师。
 * 不拥有：正文怎么切、批注落在原文哪里，那由引擎定；这里只把选中的话和它的前后文交过去。和导师说话经 onSay。
 */

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Star } from "lucide-react";
import { useCopy } from "../../../lib/i18n";
import * as lib from "../../../lib/learn-library";
import type { AnnotColor, Annotation, Bookmark, Hit, Material, MaterialView, UnitView } from "../../../lib/learn-library";
import { ANNOT_COLORS } from "../../../lib/learn-library";
import { fill } from "../../../lib/learn-format";
import { clampLocator, contextAround, findQuote, readPercent } from "../../../lib/learn-reader-util";
import { errMessage } from "../../../lib/normalize";
import { Markdown } from "../../../lib/markdown";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { ProgressHairline } from "../../../components/ui/progress-hairline";
import { cn } from "../../../lib/utils";
import { LearnShell } from "../LearnShell";
import { whereLabel } from "./LibraryPane";
import "./reader.css";

type Panel = "" | "toc" | "find" | "notes";
type Picked = { quote: string; prefix: string; suffix: string; top: number; left: number };

/** typing 在按键落在输入框里时为真，这时方向键不翻页。 */
function typing(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el || !el.tagName) return false;
  return el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable;
}

/** textRange 在 root 的文字里把 [start, end) 这一段做成一个 Range。下标按 root.textContent 算。做不出来时返回 null。 */
function textRange(root: HTMLElement, start: number, end: number): Range | null {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  const range = document.createRange();
  let seen = 0;
  let begun = false;
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const len = node.textContent?.length || 0;
    if (!begun && start < seen + len) {
      range.setStart(node, start - seen);
      begun = true;
    }
    if (begun && end <= seen + len) {
      range.setEnd(node, end - seen);
      return range;
    }
    seen += len;
  }
  return null;
}

/**
 * Reader 画出一份文档的阅读器。libraryId 和 documentId 定位文档。start 给出打开时落在哪个单元、要闪一下哪段话。
 * sessionId 是这份材料的导师对话，阅读位置和选中的文字报给它。onBack 回到文库。onSay 把一句话发给导师。
 * onMaterial 在材料打开和关闭时报给调用方，导师对话的范围跟着它走。
 */
export function Reader(props: {
  libraryId: string;
  documentId: string;
  start?: { locator: number; quote?: string };
  sessionId?: string;
  onBack: () => void;
  onSay: (text: string) => void;
  onMaterial?: (material: Material | null) => void;
}) {
  const L = useCopy().learn.shelf;
  const { libraryId, documentId } = props;
  const [view, setView] = useState<MaterialView | null>(null);
  const [locator, setLocator] = useState(0);
  const [unit, setUnit] = useState<UnitView | null>(null);
  const [annots, setAnnots] = useState<Annotation[]>([]);
  const [marks, setMarks] = useState<Bookmark[]>([]);
  const [panel, setPanel] = useState<Panel>("");
  const [picked, setPicked] = useState<Picked | null>(null);
  const [noting, setNoting] = useState(false);
  const [draft, setDraft] = useState("");
  const [pageDraft, setPageDraft] = useState("");
  const [flash, setFlash] = useState(props.start?.quote || "");
  const [findText, setFindText] = useState("");
  const [found, setFound] = useState<Hit[] | null>(null);
  const [editing, setEditing] = useState<{ id: string; note: string } | null>(null);
  const [jump, setJump] = useState("");
  const [note, setNote] = useState("");
  const [err, setErr] = useState("");
  const article = useRef<HTMLDivElement>(null);
  const frame = useRef<HTMLDivElement>(null);

  const material = view?.material;
  const materialId = material?.id || "";
  const pages = unit?.pages || material?.page_count || 1;
  const unitKind = unit?.unit || material?.unit || "section";

  useEffect(() => {
    let alive = true;
    setView(null);
    setUnit(null);
    setErr("");
    void lib.openMaterial(libraryId, documentId).then((v) => {
      if (!alive) return;
      setView(v);
      setAnnots(v.annotations);
      setMarks(v.bookmarks);
      setLocator(clampLocator(props.start?.locator || v.position.locator, Math.max(1, v.material.page_count)));
      props.onMaterial?.(v.material);
    }).catch((e) => {
      if (alive) setErr(errMessage(e));
    });
    return () => {
      alive = false;
      props.onMaterial?.(null);
    };
  }, [libraryId, documentId]);

  useEffect(() => {
    if (!materialId || locator < 1) return;
    let alive = true;
    setPicked(null);
    setNoting(false);
    setJump(String(locator));
    void lib.readUnit(libraryId, documentId, locator).then((u) => {
      if (!alive) return;
      setUnit(u);
      setErr("");
      frame.current?.scrollTo({ top: 0 });
      void lib.savePosition(materialId, locator, readPercent(locator, u.pages)).catch(() => undefined);
      void lib.setViewport(props.sessionId || "", materialId, locator, "");
    }).catch((e) => {
      if (alive) setErr(`${L.loadFailed} ${errMessage(e)}`);
    });
    return () => {
      alive = false;
    };
  }, [materialId, locator, libraryId, documentId, props.sessionId]);

  const here = annots.filter((a) => a.locator === locator && a.quote);
  const hereKey = here.map((a) => `${a.id}:${a.color}`).join("|");

  useLayoutEffect(() => {
    /** 把这一页上的批注和查找命中的话画在自定义高亮层上。浏览器没有这个能力时不画，批注列表照样在。 */
    const registry = (globalThis as any).CSS?.highlights;
    const HighlightCtor = (globalThis as any).Highlight;
    const root = article.current;
    if (!registry || !HighlightCtor || !root || !unit) return;
    const text = root.textContent || "";
    const rawLength = Math.max(1, unit.text.length);
    const names: string[] = [];
    for (const color of ANNOT_COLORS) {
      const ranges: Range[] = [];
      for (const a of here) {
        if (a.color !== color) continue;
        const at = findQuote(text, a.quote, a.start >= 0 ? a.start / rawLength : undefined);
        const range = at ? textRange(root, at.start, at.end) : null;
        if (range) ranges.push(range);
      }
      if (ranges.length > 0) {
        registry.set(`learn-hl-${color}`, new HighlightCtor(...ranges));
        names.push(`learn-hl-${color}`);
      }
    }
    if (flash) {
      const at = findQuote(text, flash);
      const range = at ? textRange(root, at.start, at.end) : null;
      if (range) {
        registry.set("learn-hl-flash", new HighlightCtor(range));
        names.push("learn-hl-flash");
        (range.startContainer.parentElement as HTMLElement | null)?.scrollIntoView({ block: "center" });
      }
    }
    return () => {
      for (const name of names) registry.delete(name);
    };
  }, [unit, hereKey, flash]);

  /** 翻到第 n 个单元，并收起查找留下的闪亮。 */
  const go = useCallback((n: number, keepFlash = false) => {
    if (!keepFlash) setFlash("");
    setLocator(clampLocator(n, pages));
  }, [pages]);

  const keys = useRef({ go, locator });
  keys.current = { go, locator };
  useEffect(() => {
    /** 左右方向键翻页。在输入框里打字或者选着文字时不抢键。 */
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey || typing(e.target)) return;
      if (e.key === "ArrowLeft") keys.current.go(keys.current.locator - 1);
      if (e.key === "ArrowRight") keys.current.go(keys.current.locator + 1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  /** 读出学习者在正文里选中的话，连同前后文和工具条该出现的位置。没选中或选到了正文外面时收起工具条。 */
  const readSelection = () => {
    const root = article.current;
    const box = frame.current;
    const sel = window.getSelection();
    if (!root || !box || !sel || sel.isCollapsed || sel.rangeCount === 0) {
      if (!noting) setPicked(null);
      return;
    }
    const range = sel.getRangeAt(0);
    if (!root.contains(range.startContainer) || !root.contains(range.endContainer)) return;
    const raw = range.toString();
    const quote = raw.trim();
    if (!quote) return;
    const before = document.createRange();
    before.setStart(root, 0);
    before.setEnd(range.startContainer, range.startOffset);
    const start = before.toString().length;
    const ctx = contextAround(root.textContent || "", start, start + raw.length);
    const rect = range.getBoundingClientRect();
    const host = box.getBoundingClientRect();
    setNoting(false);
    setDraft("");
    setPicked({ quote, ...ctx, top: rect.bottom - host.top + box.scrollTop + 8, left: Math.max(8, Math.min(rect.left - host.left, host.width - 300)) });
    void lib.setViewport(props.sessionId || "", materialId, locator, quote);
  };

  /** 把选中的话记成一条批注。color 是荧光色，text 是笔记，可空。 */
  const annotate = async (color: AnnotColor, text = "") => {
    if (!picked || !materialId) return;
    try {
      const made = await lib.annotate({ materialId, locator, quote: picked.quote, note: text, color, prefix: picked.prefix, suffix: picked.suffix });
      setAnnots((list) => [...list, made]);
      setPicked(null);
      setNoting(false);
      setDraft("");
      window.getSelection()?.removeAllRanges();
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  /** 给这一页写一条不带引文的笔记。 */
  const pageNote = async () => {
    const text = pageDraft.trim();
    if (!text || !materialId) return;
    try {
      const made = await lib.annotate({ materialId, locator, quote: "", note: text, kind: "note" });
      setAnnots((list) => [...list, made]);
      setPageDraft("");
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  /** 在这份文档里查找。 */
  const runFind = async () => {
    const text = findText.trim();
    if (!text || !materialId) return;
    try {
      const res = await lib.locate(materialId, text);
      setFound(res.hits);
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  /** 把笔记导成 Markdown 放进剪贴板。 */
  const exportNotes = async () => {
    try {
      const out = await lib.exportMaterial(materialId);
      await navigator.clipboard.writeText(out.markdown);
      setNote(L.exported);
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  const title = material?.title || "";
  const where = whereLabel(L, unitKind, locator);
  const marked = marks.some((b) => b.locator === locator);
  const rich = unit && (unit.kind === "markdown" || unit.kind === "html" || unit.kind === "note");
  const colorName = (c: AnnotColor) => (c === "yellow" ? L.colorYellow : c === "green" ? L.colorGreen : c === "blue" ? L.colorBlue : c === "pink" ? L.colorPink : L.colorPurple);
  const sorted = [...annots].sort((a, b) => a.locator - b.locator || (a.start < 0 ? 1e9 : a.start) - (b.start < 0 ? 1e9 : b.start));

  return (
    <LearnShell
      title={title}
      subtitle={material ? fill(unitKind === "page" ? L.pageOf : L.sectionOf, { page: locator, pages }) : ""}
      back={{ label: L.backToLibrary, onClick: props.onBack }}
      scroll={false}
      action={material ? (
        <div className="flex items-center gap-1">
          <Button size="icon" variant="ghost" aria-label={L.prev} disabled={locator <= 1} onClick={() => go(locator - 1)}><ChevronLeft /></Button>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              go(Number(jump));
            }}
          >
            <Input className="h-8 w-14 px-1 text-center tabular-nums" value={jump} inputMode="numeric" aria-label={L.jump} onChange={(e) => setJump(e.target.value.replace(/[^0-9]/g, ""))} onBlur={() => setJump(String(locator))} />
          </form>
          <Button size="icon" variant="ghost" aria-label={L.next} disabled={locator >= pages} onClick={() => go(locator + 1)}><ChevronRight /></Button>
          <Button
            size="icon"
            variant="ghost"
            aria-label={marked ? L.unbookmark : L.bookmark}
            aria-pressed={marked}
            onClick={() => { void lib.toggleBookmark(materialId, locator).then(setMarks).catch((e) => setErr(errMessage(e))); }}
          >
            <Star className={cn("size-4", marked && "fill-current text-foreground")} aria-hidden />
          </Button>
        </div>
      ) : undefined}
    >
      <div className="shrink-0 px-8 pt-3">
        <ProgressHairline value={readPercent(locator, pages) * 100} />
        <div className="mt-2 flex flex-wrap items-center gap-1">
          {(["toc", "find", "notes"] as const).map((id) => (
            <Button key={id} size="sm" variant="ghost" className={cn("whitespace-nowrap", panel === id && "bg-lift text-foreground")} aria-expanded={panel === id} onClick={() => setPanel((cur) => (cur === id ? "" : id))}>
              {id === "toc" ? L.toc : id === "find" ? L.find : `${L.notes}${annots.length > 0 ? ` · ${annots.length}` : ""}`}
            </Button>
          ))}
          {annots.length + marks.length > 0 ? <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => { void exportNotes(); }}>{L.exportNotes}</Button> : null}
        </div>
        {err ? <p className="mt-2 text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
        {note ? <p className="mt-2 text-ui-footnote text-muted" aria-live="polite">{note}</p> : null}
        {panel ? (
          <div className="mt-2 max-h-[38vh] overflow-auto border-y border-border/60 py-2" data-testid={`learn-reader-${panel}`}>
            {panel === "toc" ? (
              <>
                {marks.length > 0 ? (
                  <div className="mb-2">
                    <p className="text-ui-caption font-medium uppercase tracking-[0.08em] text-muted/80">{L.bookmarks}</p>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {marks.map((b) => <Button key={b.id} size="xs" variant="outline" onClick={() => go(b.locator)}>{whereLabel(L, unitKind, b.locator)}</Button>)}
                    </div>
                  </div>
                ) : null}
                {view && view.outline.length > 0 ? (
                  <ul>
                    {view.outline.map((entry, i) => (
                      <li key={`${entry.locator}-${i}`}>
                        <button
                          type="button"
                          className={cn("flex w-full items-baseline justify-between gap-3 rounded-md py-1 pr-2 text-left text-ui-body hover:bg-lift/60", entry.locator === locator && "text-foreground", entry.locator !== locator && "text-muted")}
                          style={{ paddingLeft: `${Math.min(4, Math.max(0, entry.level - 1)) * 14 + 8}px` }}
                          onClick={() => { setFlash(entry.title); go(entry.locator, true); }}
                        >
                          <span className="min-w-0 truncate">{entry.title}</span>
                          <span className="shrink-0 text-ui-caption tabular-nums text-muted/70">{entry.locator}</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                ) : <p className="px-2 text-ui-footnote text-muted">{L.tocEmpty}</p>}
              </>
            ) : null}
            {panel === "find" ? (
              <>
                <form
                  className="flex items-center gap-2"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void runFind();
                  }}
                >
                  <Input className="min-w-0 flex-1" value={findText} placeholder={L.findPh} aria-label={L.findPh} autoFocus onChange={(e) => setFindText(e.target.value)} />
                  <Button size="sm" type="submit" variant="outline" className="shrink-0 whitespace-nowrap" disabled={!findText.trim()}>{L.find}</Button>
                </form>
                {found !== null && found.length === 0 ? <p className="mt-2 text-ui-footnote text-muted">{L.findNone}</p> : null}
                <ul className="mt-1">
                  {(found || []).map((hit, i) => (
                    <li key={`${hit.page}-${i}`}>
                      <button type="button" className="flex w-full items-baseline gap-3 rounded-md px-2 py-1.5 text-left hover:bg-lift/60" onClick={() => { setFlash(findText.trim()); go(hit.page, true); }}>
                        <span className="w-16 shrink-0 whitespace-nowrap text-ui-footnote tabular-nums text-muted">{whereLabel(L, unitKind, hit.page)}</span>
                        <span className="min-w-0 flex-1 truncate text-ui-body">{hit.excerpt}</span>
                      </button>
                    </li>
                  ))}
                </ul>
              </>
            ) : null}
            {panel === "notes" ? (
              <>
                <form
                  className="flex items-center gap-2"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void pageNote();
                  }}
                >
                  <Input className="min-w-0 flex-1" value={pageDraft} placeholder={L.pageNotePh} aria-label={L.pageNotePh} onChange={(e) => setPageDraft(e.target.value)} />
                  <Button size="sm" type="submit" variant="outline" className="shrink-0 whitespace-nowrap" disabled={!pageDraft.trim()}>{L.saveNote}</Button>
                </form>
                {sorted.length === 0 ? <p className="mt-2 px-2 text-ui-footnote text-muted">{L.annotEmpty}<span className="text-muted/50"> · </span>{L.annotEmptyHint}</p> : null}
                <ul className="mt-1">
                  {sorted.map((a) => (
                    <li key={a.id} className="border-b border-border/40 py-2 last:border-b-0" data-testid="learn-annotation">
                      <button type="button" className="flex w-full items-start gap-2 text-left" onClick={() => { setFlash(a.quote); go(a.locator, true); }}>
                        <span className={cn("mt-1.5 size-2 shrink-0 rounded-full", `learn-swatch-${a.color}`)} aria-hidden />
                        <span className="min-w-0 flex-1">
                          {a.quote ? <span className="line-clamp-2 block text-ui-body">{a.quote}</span> : null}
                          {a.note && editing?.id !== a.id ? <span className={cn("block whitespace-pre-wrap text-ui-body text-muted", a.quote && "mt-0.5")}>{a.note}</span> : null}
                          <span className="mt-0.5 block text-ui-caption text-muted/70">
                            {[whereLabel(L, unitKind, a.locator), a.author === "assistant" ? L.byTutor : "", a.resolution === "unresolved" ? L.unresolved : ""].filter(Boolean).join(" · ")}
                          </span>
                        </span>
                      </button>
                      {editing?.id === a.id ? (
                        <form
                          className="mt-1 flex items-center gap-2 pl-4"
                          onSubmit={(e) => {
                            e.preventDefault();
                            const text = editing.note;
                            setEditing(null);
                            void lib.updateAnnotation(a.id, { note: text }).then((next) => setAnnots((list) => list.map((x) => (x.id === a.id ? next : x)))).catch((er) => setErr(errMessage(er)));
                          }}
                        >
                          <Input className="min-w-0 flex-1" value={editing.note} placeholder={L.notePh} aria-label={L.notePh} autoFocus onChange={(e) => setEditing({ id: a.id, note: e.target.value })} />
                          <Button size="xs" type="submit">{L.saveNote}</Button>
                          <Button size="xs" type="button" variant="ghost" onClick={() => setEditing(null)}>{L.cancel}</Button>
                        </form>
                      ) : (
                        <div className="mt-1 flex items-center gap-1 pl-4">
                          <Button size="xs" variant="ghost" onClick={() => setEditing({ id: a.id, note: a.note })}>{L.edit}</Button>
                          <Button size="xs" variant="ghost" onClick={() => { void lib.deleteAnnotation(a.id).then(() => setAnnots((list) => list.filter((x) => x.id !== a.id))).catch((er) => setErr(errMessage(er))); }}>{L.delete}</Button>
                        </div>
                      )}
                    </li>
                  ))}
                </ul>
              </>
            ) : null}
          </div>
        ) : null}
      </div>
      <div ref={frame} className="relative min-h-0 flex-1 overflow-auto" data-testid="learn-reader">
        <div className="mx-auto max-w-[680px] px-8 py-6">
          {unit?.status === "needs_ocr" ? <p className="text-ui-body text-muted">{L.needsOcr}</p> : null}
          {unit?.status === "changed" ? <p className="text-ui-body text-warning">{L.changed}</p> : null}
          <div
            ref={article}
            className={cn("prose-select text-ui-body-l leading-[1.75]", !rich && "whitespace-pre-wrap break-words")}
            data-testid="learn-reader-text"
            data-locator={locator}
            onMouseUp={readSelection}
            onKeyUp={readSelection}
          >
            {unit && unit.status === "ok" ? (rich ? <Markdown text={unit.text} /> : unit.text) : null}
          </div>
          {material && unit ? (
            <div className="mt-8 flex items-center justify-between gap-3 border-t border-border/50 pt-4">
              <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={locator <= 1} onClick={() => go(locator - 1)}>{L.prev}</Button>
              <span className="text-ui-footnote tabular-nums text-muted">{where}</span>
              <Button size="sm" variant={locator >= pages ? "ghost" : "outline"} className="whitespace-nowrap" disabled={locator >= pages} onClick={() => go(locator + 1)}>{L.next}</Button>
            </div>
          ) : null}
        </div>
        {picked ? (
          <div
            className="absolute z-10 flex w-[292px] flex-col gap-2 rounded-xl border border-border/70 bg-card p-2"
            style={{ top: picked.top, left: picked.left }}
            data-testid="learn-selection-bar"
            onMouseDown={(e) => { if ((e.target as HTMLElement).tagName !== "TEXTAREA") e.preventDefault(); }}
          >
            <div className="flex items-center gap-1.5">
              {ANNOT_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  className={cn("size-5 shrink-0 rounded-full border border-border/60", `learn-swatch-${c}`)}
                  aria-label={fill(L.highlight, { color: colorName(c) })}
                  data-color={c}
                  onClick={() => { void annotate(c, noting ? draft.trim() : ""); }}
                />
              ))}
              <span className="mx-1 h-4 w-px bg-border/70" aria-hidden />
              <Button size="xs" variant={noting ? "outline" : "ghost"} onClick={() => setNoting((v) => !v)}>{L.addNote}</Button>
              <Button
                size="xs"
                variant="ghost"
                onClick={() => {
                  props.onSay(fill(L.sayExplain, { title, where, quote: picked.quote }));
                  setPicked(null);
                }}
              >
                {L.askTutor}
              </Button>
            </div>
            {noting ? (
              <>
                <textarea
                  className="min-h-16 w-full resize-y rounded-lg border border-border/60 bg-transparent px-2 py-1.5 text-ui-body outline-none focus-visible:border-accent/60"
                  value={draft}
                  placeholder={L.notePh}
                  aria-label={L.notePh}
                  autoFocus
                  onChange={(e) => setDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                      e.preventDefault();
                      void annotate("yellow", draft.trim());
                    }
                  }}
                />
                <div><Button size="xs" disabled={!draft.trim()} onClick={() => { void annotate("yellow", draft.trim()); }}>{L.saveNote}</Button></div>
              </>
            ) : null}
          </div>
        ) : null}
      </div>
    </LearnShell>
  );
}
