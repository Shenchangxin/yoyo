/**
 * 本文件拥有创作入口：导入、翻拍、成片模式和最近的成片与剪辑。
 * 不拥有剪辑台和成片台的编辑。它只读列表并改本地指针，写入交给回调。
 */
import { useEffect, useState } from "react";
import { Clapperboard, Film, Images, PanelsTopLeft } from "lucide-react";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import type { CutProject } from "./cut-desk";
import { FilmBoard } from "./FilmBoard";
import { FilmStudio } from "./FilmStudio";
import { useFilmDesk, type FilmProject } from "./film-desk";
import { castDefaults, isFilmMode, useFilmCast, type FilmMode } from "./film-cast";
import { RemakeDraft } from "./RemakeDraft";
import { useRemakeDesk } from "./remake-desk";
import { useCreateMode } from "./create-mode";

/** 把未知值收成剪辑工程数组。v 是数组就当剪辑工程数组，否则返回空数组。 */
function asCuts(v: unknown): CutProject[] {
  return Array.isArray(v) ? v as CutProject[] : [];
}

/** 把结构 id 换成榜单、播客或访谈的名字。film 按 ranking、podcast、interview 取对应名字。 */
function structureLabel(id: string | undefined, film: Record<FilmMode, string>) {
  if (id && isFilmMode(id)) return film[id];
  return "";
}

/** CreateHome 在成片模式上编辑榜、对谈或提问，否则列出导入、翻拍和最近六条成片与剪辑。props 的 onImport 和 onRemake 挂在按钮上，onFilm 带上结构，onHandoff 带上目标栏，onOpenCut 打开剪辑。 */
export function CreateHome(props: {
  onImport: () => void;
  onRemake: () => void;
  onFilm: (structure: FilmMode, description?: string) => void;
  onHandoff: (dest: "studio" | "drama" | "canvas") => void;
  onOpenCut: (id: string) => void;
  onPrompt?: (text: string) => void;
  draftKey?: string;
}) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const film = copy.video.film;
  const mode = useCreateMode(/** s 取出 mode。 */ (s) => s.mode);
  const [cuts, setCuts] = useState<CutProject[]>([]);
  const [films, setFilms] = useState<FilmProject[]>([]);
  const openFilm = useFilmDesk(/** s 取出 project。 */ (s) => s.project);
  const openDraft = useRemakeDesk(/** s 取出 draft。 */ (s) => s.draft);

  useEffect(() => {
    let alive = true;
    void api.video.cutList().then(/** list 收成剪辑数组。 */ (list) => {
      if (!alive) return;
      const rows = asCuts(list).filter(/** row 要有 id 和 source_hash。 */ (row) => row?.id && row.source_hash);
      rows.sort(/** b 的 updated_at 和 a 的比较，空的当空字符串。 */ (a, b) => String(b.updated_at || "").localeCompare(String(a.updated_at || "")));
      setCuts(rows.slice(0, 6));
    }).catch(() => {
      if (alive) setCuts([]);
    });
    void api.video.filmList().then(/** list 是数组就当作成片，否则当空数组。 */ (list) => {
      if (!alive) return;
      const rows = Array.isArray(list) ? list as FilmProject[] : [];
      setFilms(rows.filter(/** row 要有 id。 */ (row) => row?.id).slice(0, 6));
    }).catch(() => {
      if (alive) setFilms([]);
    });
    return () => { alive = false; };
  }, []);

  const workshops: { id: "studio" | "drama" | "canvas"; title: string; hint: string; icon: typeof Film }[] = [
    { id: "studio", title: c.handoffShort, hint: c.handoffShortHint, icon: Film },
    { id: "drama", title: c.handoffDrama, hint: c.handoffDramaHint, icon: Clapperboard },
    { id: "canvas", title: c.handoffCanvas, hint: c.handoffCanvasHint, icon: PanelsTopLeft },
  ];
  const busy = !!openFilm?.id || !!openDraft?.id;
  const filmMode = isFilmMode(mode) ? mode : null;
  const savedCast = useFilmCast(/** s 在有成片模式时取出该模式的草稿。 */ (s) => (filmMode ? s.saved[filmMode] : undefined));
  const cast = filmMode ? (savedCast ?? castDefaults(film, filmMode)) : null;
  const filmTitle: Record<FilmMode, string> = {
    talking: film.talking,
    drama: film.drama,
    podcast: film.podcast,
    interview: film.interview,
    ranking: film.ranking,
    explainer: film.explainer,
    narration: film.narration,
  };
  const visibleFilms = filmMode ? films.filter(/** row 的 structure 和当前成片模式相同才留下。 */ (row) => row.structure === filmMode) : films;

  /** 打开一条成片，并把模式拨到它的结构上。row 有 id 才写入。 */
  function openFilmRow(row: FilmProject) {
    if (!row.id) return;
    if (row.structure && isFilmMode(row.structure)) {
      useCreateMode.getState().setMode(row.structure);
    }
    useFilmDesk.getState().setProject(row);
  }

  return (
    <section className="creation-home-handoff" data-testid="creation-handoff" aria-label={c.handoffTitle}>
      <FilmBoard onOpenCut={props.onOpenCut} />
      <RemakeDraft onOpenCut={props.onOpenCut} />
      {busy ? null : filmMode && cast ? (
        <>
          <FilmStudio
            key={filmMode}
            mode={filmMode}
            value={cast}
            testId={`film-${filmMode}`}
            commitLabel={film.makePicture}
            films={visibleFilms.flatMap(/** row 没有 id 就不放进成片。有 id 时用标题，空标题用这种片子的名字。 */ (row) => (
              row.id ? [{ id: row.id, title: row.title || filmTitle[filmMode], onOpen: () => openFilmRow(row) }] : []
            ))}
            onChange={/** next 按当前模式记下。 */ (next) => useFilmCast.getState().write(filmMode, next)}
            onCommit={() => props.onFilm(filmMode)}
          />
        </>
      ) : (
        <>
          <button type="button" className="creation-start-primary" data-testid="create-import" onClick={props.onImport}>
            <Images className="size-4" aria-hidden />
            <span>
              <strong>{c.pickVideo}</strong>
              <small>{c.pickVideoHint}</small>
            </span>
          </button>
          <details className="creation-more">
            <summary>{c.moreWays}</summary>
          <div className="creation-handoff-grid">
            <button type="button" className="creation-handoff-card" data-testid="create-remake" onClick={props.onRemake}>
              <Images className="size-3.5" aria-hidden />
              <span>
                <strong>{c.remake}</strong>
                <small>{c.remakeHint}</small>
              </span>
            </button>
            {workshops.map(/** card 用 id 交给 onHandoff，用 icon、title 和 hint 画卡片。 */ (card) => {
              const Icon = card.icon;
              return (
                <button key={card.id} type="button" className="creation-handoff-card" onClick={() => props.onHandoff(card.id)}>
                  <Icon className="size-3.5" aria-hidden />
                  <span>
                    <strong>{card.title}</strong>
                    <small>{card.hint}</small>
                  </span>
                </button>
              );
            })}
          </div>
          </details>
          <div className="creation-recent">
            <h2>{c.yourFilms}</h2>
            {films.length + cuts.length === 0 ? <p>{c.recentEmpty}</p> : (
              <ul>
                {films.map(/** row 交给 openFilmRow，并显示标题和结构名。 */ (row) => (
                  <li key={`film-${row.id}`}>
                    <button type="button" onClick={() => openFilmRow(row)}>
                      <span>{row.title || film.title}</span>
                      <em>{structureLabel(row.structure, film)}</em>
                    </button>
                  </li>
                ))}
                {cuts.map(/** row 没有 id 就返回；翻拍且没有成片哈希就设成草稿，否则交给 onOpenCut。 */ (row) => (
                  <li key={`cut-${row.id}`}>
                    <button type="button" onClick={() => {
                      if (!row.id) return;
                      if (row.kind === "remake" && !row.result_hash) {
                        useRemakeDesk.getState().setDraft(row);
                        return;
                      }
                      props.onOpenCut(row.id);
                    }}>
                      <span>{row.title || c.cutTitle}</span>
                      {row.duration ? <em>{row.duration.toFixed(0)}s</em> : null}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      )}
    </section>
  );
}
