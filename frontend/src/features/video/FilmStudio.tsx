/**
 * 本文件拥有片子的工作室：来源、任务、成片在左，这一拍的画面在中，这一拍的字在右，时间在下。
 * 不拥有出片。改字交给 FilmStage，做出画面交给 onCommit。
 */
import { useState } from "react";
import { FilmStage } from "./FilmStage";
import { filmReading, type FilmCast, type FilmMode } from "./film-cast";
import { useCopy } from "../../lib/i18n";

/** FilmStudio 按当前这一拍摆出来源、画面和字。props 的 picture 有地址时，中间改成已经做出的画面。 */
export function FilmStudio(props: {
  mode: FilmMode;
  value: FilmCast;
  onChange: (next: FilmCast) => void;
  onCommit?: () => void;
  commitLabel?: string;
  busy?: boolean;
  testId?: string;
  picture?: { src: string; video: boolean; title: string };
  needs?: { title: string; detail?: string }[];
  films?: { id: string; title: string; onOpen: () => void }[];
  failed?: string;
}) {
  const copy = useCopy();
  const film = copy.video.film;
  const [view, setView] = useState<"studio" | "comments">("studio");
  const [library, setLibrary] = useState<"source" | "tasks" | "artifacts">("source");
  const [beat, setBeat] = useState(0);
  const [notes, setNotes] = useState<Record<number, string>>({});
  const beats = filmReading(props.mode, props.value);
  const at = Math.min(beat, Math.max(0, beats.length - 1));
  const lines = scriptOf(props.mode, props.value);
  const name = namesOf(props.mode, film);
  const watch = props.commitLabel === film.goEdit ? film.watchReady : film.watchStale;

  return (
    <section className="film-studio" data-mode={props.mode} data-view={view} data-testid={props.testId} aria-label={name}>
      <header className="film-studio-top">
        <p className="film-studio-title">{name}</p>
        {props.onCommit ? (
          <button
            type="button"
            className="film-commit"
            data-testid={props.testId ? `${props.testId}-go` : undefined}
            disabled={!!props.busy}
            onClick={props.onCommit}
          >
            {props.commitLabel}
          </button>
        ) : null}
        <div className="film-studio-views" role="tablist" aria-label={film.studioView}>
          <button type="button" role="tab" aria-selected={view === "studio"} onClick={() => setView("studio")}>{film.studioView}</button>
          <button type="button" role="tab" aria-selected={view === "comments"} onClick={() => setView("comments")}>{film.commentsView}</button>
        </div>
      </header>
      <div className="film-studio-body">
        <aside className="film-studio-library">
          <div className="film-studio-library-tabs" role="group" aria-label={film.librarySource}>
            <button type="button" aria-pressed={library === "source"} onClick={() => setLibrary("source")}>{film.librarySource}</button>
            <button type="button" aria-pressed={library === "tasks"} onClick={() => setLibrary("tasks")}>{film.libraryTasks}</button>
            <button type="button" aria-pressed={library === "artifacts"} onClick={() => setLibrary("artifacts")}>{film.libraryArtifacts}</button>
          </div>
          {library === "source" ? (
            <ol className="film-studio-script">
              {lines.map(/** line 是来源里的一句。index 只用来做键。点一句就跳到它出现的那一拍。 */ (line, index) => (
                <li key={index}>
                  <button type="button" aria-pressed={line.beat === at} onClick={() => { if (line.beat != null) setBeat(line.beat); }}>
                    {line.text}
                  </button>
                </li>
              ))}
            </ol>
          ) : null}
          {library === "tasks" ? (
            props.needs && props.needs.length ? (
              <ul className="film-studio-list">
                {props.needs.map(/** need 显示标题。index 只用来做键。 */ (need, index) => (
                  <li key={index}><strong>{need.title}</strong>{need.detail ? <small>{need.detail}</small> : null}</li>
                ))}
              </ul>
            ) : <p className="film-studio-empty">{film.tasksEmpty}</p>
          ) : null}
          {library === "artifacts" ? (
            props.picture?.src || props.films?.length ? (
              <ul className="film-studio-list">
                {props.picture?.src ? <li><strong>{props.picture.title}</strong></li> : null}
                {props.films?.map(/** item 打开这条已经做过的片子。 */ (item) => (
                  <li key={item.id}><button type="button" onClick={item.onOpen}>{item.title}</button></li>
                ))}
              </ul>
            ) : <p className="film-studio-empty">{film.artifactsEmpty}</p>
          ) : null}
        </aside>
        <div className="film-studio-picture">
          {props.picture?.src ? (
            props.picture.video
              ? <video className="creation-film-preview" src={props.picture.src} controls />
              : <img className="creation-film-preview" src={props.picture.src} alt={props.picture.title} />
          ) : (
            <FilmPicture mode={props.mode} value={props.value} beat={at} />
          )}
          {props.picture?.src ? <p className="film-kicker">{watch}</p> : null}
          {props.failed ? <p className="film-studio-empty">{props.failed}</p> : null}
        </div>
        {view === "comments" ? (
          <aside className="film-studio-inspector film-studio-comments">
            <label>
              <span>{film.noteLabel}</span>
              <textarea
                aria-label={film.noteLabel}
                placeholder={film.noteHold}
                value={notes[at] || ""}
                onChange={/** ev 把这段话记在当前这一拍上。 */ (ev) => setNotes({ ...notes, [at]: ev.target.value })}
              />
            </label>
          </aside>
        ) : (
          <aside className="film-studio-inspector" aria-label={film.properties}>
            <h2>{film.properties}</h2>
            <FilmStage
              mode={props.mode}
              value={props.value}
              beat={at}
              hideReading
              onChange={props.onChange}
            />
          </aside>
        )}
      </div>
      {view === "studio" ? (
        <ol className="film-studio-timeline" aria-label={film.reading}>
          {beats.map(/** item 是时间上的一拍。index 选中后，画面和右边的字都换成这一拍。 */ (item, index) => (
            <li key={index}>
              <button type="button" aria-pressed={index === at} onClick={() => setBeat(index)}>
                {item.lines.map(/** line 拼进这一拍。lineIndex 只用来做键。 */ (line, lineIndex) => (
                  <span key={lineIndex} data-fresh={line.fresh ? "true" : "false"}>{line.text}</span>
                ))}
              </button>
            </li>
          ))}
        </ol>
      ) : null}
    </section>
  );
}

/** namesOf 取出这种片子在顶栏上的名字。mode 选择片子。film 提供文案。 */
function namesOf(mode: FilmMode, film: { talking: string; drama: string; podcast: string; interview: string; ranking: string; explainer: string; narration: string }) {
  return film[mode];
}

/** FilmPicture 画出观众在这一拍看见的物体。mode 决定物体怎么留。value 提供字。beat 是当前拍。 */
function FilmPicture(props: { mode: FilmMode; value: FilmCast; beat: number }) {
  const at = props.beat;
  if (props.mode === "podcast" || props.mode === "drama") {
    return (
      <div className="film-picture-pair">
        <div className="film-picture-seat" data-speaking={at === 0 ? "true" : "false"}>
          <strong>{props.value.left}</strong>
          <p>{props.value.ask}</p>
        </div>
        <div className="film-picture-seat" data-speaking={at > 0 ? "true" : "false"} data-wait={at === 0 ? "true" : "false"}>
          <strong>{props.value.right}</strong>
          <p>{at === 0 ? "" : props.value.reply}</p>
        </div>
      </div>
    );
  }
  if (props.mode === "talking") {
    return (
      <div className="film-picture-frame">
        <strong>{props.value.left}</strong>
        <p>{props.value.rows[at]}</p>
      </div>
    );
  }
  if (props.mode === "narration") {
    return (
      <div className="film-picture-frame">
        <p>{props.value.rows[at]}</p>
      </div>
    );
  }
  const frame = filmReading(props.mode, props.value)[at];
  return (
    <ol className="film-picture-board">
      {frame?.lines.map(/** line 是这一拍已经能看见的一行。index 只用来做键。序号和字分开，避免黏在一起。 */ (line, index) => {
        const numbered = /^(\d+)\s+([\s\S]+)$/.exec(line.text);
        return (
          <li key={index} data-fresh={line.fresh ? "true" : "false"} data-wait={line.text === "?" ? "true" : "false"}>
            {numbered ? <><b>{numbered[1]}</b><span>{numbered[2]}</span></> : line.text}
          </li>
        );
      })}
    </ol>
  );
}

/** scriptOf 把当前能改的字收成来源里的几句。mode 决定先写人还是先写行。cast 提供这些字。beat 用来点一句就跳到那一拍。 */
function scriptOf(mode: FilmMode, cast: FilmCast) {
  const subject = cast.subject.trim();
  const head = subject ? [{ text: subject, beat: null as number | null }] : [];
  if (mode === "podcast" || mode === "drama") {
    const ask = `${cast.left}  ${cast.ask}`.trim();
    const reply = `${cast.right}  ${cast.reply}`.trim();
    return [
      ...head,
      ...(ask ? [{ text: ask, beat: 0 }] : []),
      ...(reply ? [{ text: reply, beat: 1 }] : []),
    ];
  }
  const rows = cast.rows
    .map(/** row 收成一句。index 是它出现的那一拍。 */ (row, index) => ({ text: row.trim(), beat: index }))
    .filter(/** row 空着就不写进来源。 */ (row) => row.text);
  if (mode === "talking" && cast.left.trim()) return [...head, { text: cast.left.trim(), beat: null }, ...rows];
  return [...head, ...rows];
}
