/**
 * 本文件拥有创作入口：导入、翻拍、成片模式和最近的成片与剪辑。
 * 不拥有剪辑台和成片台的编辑。它只读列表并改本地指针，写入交给回调。
 */
import { useEffect, useState } from "react";
import { Clapperboard, Film, Images, PanelsTopLeft } from "lucide-react";
import * as api from "../../lib/client";
import { useUI } from "../../lib/store";
import { useCopy } from "../../lib/i18n";
import type { CutProject } from "./cut-desk";
import { FilmBoard } from "./FilmBoard";
import { useFilmDesk, type FilmProject } from "./film-desk";
import { RemakeDraft } from "./RemakeDraft";
import { useRemakeDesk } from "./remake-desk";
import { useCreateMode, type CreateMode } from "./create-mode";

/** 把未知值收成剪辑工程数组。 */
function asCuts(v: unknown): CutProject[] {
  return Array.isArray(v) ? v as CutProject[] : [];
}

/** 把结构 id 换成榜单、播客或访谈的名字。 */
function structureLabel(id: string | undefined, film: { ranking: string; podcast: string; interview: string }) {
  if (id === "ranking") return film.ranking;
  if (id === "podcast") return film.podcast;
  if (id === "interview") return film.interview;
  return "";
}

/** CreateHome 在成片模式下列出对应结构，否则列出导入、翻拍和最近六条成片与剪辑。 */
export function CreateHome(props: {
  onImport: () => void;
  onRemake: () => void;
  onFilm: (structure: "ranking" | "podcast" | "interview", description?: string) => void;
  onHandoff: (dest: "studio" | "drama" | "canvas") => void;
  onOpenCut: (id: string) => void;
  onPrompt?: (text: string) => void;
  draftKey?: string;
}) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const film = copy.video.film;
  const mode = useCreateMode((s) => s.mode);
  const draft = useUI((s) => (props.draftKey ? s.drafts[props.draftKey] || "" : ""));
  const [cuts, setCuts] = useState<CutProject[]>([]);
  const [films, setFilms] = useState<FilmProject[]>([]);
  const openFilm = useFilmDesk((s) => s.project);
  const openDraft = useRemakeDesk((s) => s.draft);

  useEffect(() => {
    let alive = true;
    void api.video.cutList().then((list) => {
      if (!alive) return;
      const rows = asCuts(list).filter((row) => row?.id && row.source_hash);
      rows.sort((a, b) => String(b.updated_at || "").localeCompare(String(a.updated_at || "")));
      setCuts(rows.slice(0, 6));
    }).catch(() => {
      if (alive) setCuts([]);
    });
    void api.video.filmList().then((list) => {
      if (!alive) return;
      const rows = Array.isArray(list) ? list as FilmProject[] : [];
      setFilms(rows.filter((row) => row?.id).slice(0, 6));
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
  const filmMode = mode === "ranking" || mode === "podcast" || mode === "interview" ? mode : null;
  const filmCopy: Record<Exclude<CreateMode, "clip">, { title: string; hint: string; start: string; sample: string }> = {
    ranking: { title: film.ranking, hint: film.rankingHint, start: c.startRanking, sample: c.suggestRanking },
    podcast: { title: film.podcast, hint: film.podcastHint, start: c.startPodcast, sample: c.suggestPodcast },
    interview: { title: film.interview, hint: film.interviewHint, start: c.startInterview, sample: c.suggestInterview },
  };
  const visibleFilms = filmMode ? films.filter((row) => row.structure === filmMode) : films;

  return (
    <section className="creation-home-handoff" data-testid="creation-handoff" aria-label={c.handoffTitle}>
      <FilmBoard onOpenCut={props.onOpenCut} />
      <RemakeDraft onOpenCut={props.onOpenCut} />
      {busy ? null : filmMode ? (
        <>
          <button
            type="button"
            className="creation-start-primary"
            data-testid={`film-${filmMode}`}
            onClick={() => props.onFilm(filmMode, draft.trim())}
          >
            <span>
              <strong>{filmCopy[filmMode].start}</strong>
              <small>{filmCopy[filmMode].hint}</small>
            </span>
          </button>
          <button type="button" className="creation-choice" onClick={() => props.onPrompt?.(filmCopy[filmMode].sample)}>
            <strong>{c.tryThis}</strong>
            <small>{filmCopy[filmMode].sample}</small>
          </button>
          {visibleFilms.length ? (
            <div className="creation-recent">
              <h2>{c.yourFilms}</h2>
              <ul>
                {visibleFilms.map((row) => (
                  <li key={`film-${row.id}`}>
                    <button type="button" onClick={() => { if (row.id) useFilmDesk.getState().setProject(row); }}>
                      <span>{row.title || filmCopy[filmMode].title}</span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
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
            {workshops.map((card) => {
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
                {films.map((row) => (
                  <li key={`film-${row.id}`}>
                    <button type="button" onClick={() => { if (row.id) useFilmDesk.getState().setProject(row); }}>
                      <span>{row.title || film.title}</span>
                      <em>{structureLabel(row.structure, film)}</em>
                    </button>
                  </li>
                ))}
                {cuts.map((row) => (
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
