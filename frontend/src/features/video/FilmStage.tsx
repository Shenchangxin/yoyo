/**
 * 本文件拥有七种片子里，当前这一拍能改的字。
 * 不拥有保存和出片。改字交给 onChange。传入 beat 时只编辑这一拍新出现的那一行，留下来的人仍可改。
 */
import type { FocusEvent } from "react";
import { filmReading, type FilmCast, type FilmMode } from "./film-cast";
import type { Copy } from "../../lib/copy";
import { useCopy } from "../../lib/i18n";

type FilmCopy = Copy["video"]["film"];

/** faceOf 取出一种片子的名字、提示，以及物体该画成榜、对谈还是一列。mode 选择片子。film 提供文案。 */
function faceOf(mode: FilmMode, film: FilmCopy) {
  switch (mode) {
    case "podcast":
      return { kind: "pair" as const, name: film.podcast, hint: film.podcastHint, subject: film.talkSubjectLabel, hold: film.talkHold, object: film.pairLabel, place: film.rankPlace };
    case "drama":
      return { kind: "pair" as const, name: film.drama, hint: film.dramaHint, subject: film.talkSubjectLabel, hold: film.talkHold, object: film.pairLabel, place: film.rankPlace };
    case "interview":
      return { kind: "lines" as const, name: film.interview, hint: film.interviewHint, subject: film.askSubjectLabel, hold: film.askHold, object: film.answersLabel, place: film.askPlace };
    case "talking":
      return { kind: "lines" as const, name: film.talking, hint: film.talkingHint, subject: film.rankSubjectLabel, hold: film.talkHold, object: film.pairLabel, place: film.talkingLine };
    case "narration":
      return { kind: "lines" as const, name: film.narration, hint: film.narrationHint, subject: film.rankSubjectLabel, hold: film.talkHold, object: film.answersLabel, place: film.picturePlace };
    case "explainer":
      return { kind: "board" as const, name: film.explainer, hint: film.explainerHint, subject: film.rankSubjectLabel, hold: film.rankHold, object: film.boardLabel, place: film.pointPlace };
    default:
      return { kind: "board" as const, name: film.ranking, hint: film.rankingHint, subject: film.rankSubjectLabel, hold: film.rankHold, object: film.boardLabel, place: film.rankPlace };
  }
}

/** place 把文案里的 {n} 换成名次。template 里的 {n} 换成 n。 */
function place(template: string, n: number) {
  return template.replace("{n}", String(n));
}

/** FilmStage 画出一种片子上能改的字。有 onCommit 时，按钮就在这些字下面。 */
export function FilmStage(props: {
  mode: FilmMode;
  value: FilmCast;
  onChange: (next: FilmCast) => void;
  onCommit?: () => void;
  commitLabel?: string;
  commitTone?: "make" | "next";
  hideReading?: boolean;
  beat?: number;
  busy?: boolean;
  testId?: string;
  onBlur?: () => void;
}) {
  const copy = useCopy();
  const film = copy.video.film;
  const value = props.value;

  /** 焦点还在这块里面就不算离开。ev 的 relatedTarget 还在 currentTarget 里就返回。 */
  function leave(ev: FocusEvent<HTMLElement>) {
    if (ev.currentTarget.contains(ev.relatedTarget as Node | null)) return;
    props.onBlur?.();
  }

  /** 改题目。text 写入 subject。 */
  function setSubject(text: string) {
    props.onChange({ ...value, subject: text });
  }

  /** 改榜或答案的一行。index 决定哪一行。text 写进 rows。 */
  function setRow(index: number, text: string) {
    const rows = [...value.rows] as FilmCast["rows"];
    rows[index] = text;
    props.onChange({ ...value, rows });
  }

  const face = faceOf(props.mode, film);
  const beats = filmReading(props.mode, value);
  const indexes = props.beat == null
    ? value.rows.map(/** _row 不使用。index 是行号。 */ (_row, index) => index)
    : [Math.min(Math.max(0, props.beat), value.rows.length - 1)];

  return (
    <section
      className="film-stage"
      data-mode={props.mode}
      data-testid={props.testId}
      onBlur={leave}
    >
      <header className="film-brief">
        <p className="film-kicker">{face.name}</p>
        <input
          className="film-sentence"
          aria-label={face.subject}
          placeholder={face.hold}
          value={value.subject}
          onChange={/** ev 取出 target.value 交给 setSubject。 */ (ev) => setSubject(ev.target.value)}
        />
        <p className="film-hint">{face.hint}</p>
      </header>
      {face.kind === "board" ? (
        <ol className="film-board" aria-label={face.object}>
          {indexes.map(/** index 是这一名在榜上的位置。只改当前这一拍时，榜上只留下这一名的输入。 */ (index) => {
            const row = value.rows[index] ?? "";
            return (
            <li key={index} className="film-rank" data-place={index + 1}>
              <b aria-hidden>{index + 1}</b>
              <input
                aria-label={place(face.place, index + 1)}
                placeholder={film.rankHold}
                value={row}
                onChange={/** ev 取出 target.value，连同 index 交给 setRow。 */ (ev) => setRow(index, ev.target.value)}
              />
            </li>
            );
          })}
        </ol>
      ) : null}
      {face.kind === "pair" ? (
        <div className="film-pair" role="group" aria-label={face.object}>
          <div className="film-seat" data-side="left" data-speaking={props.beat == null || props.beat === 0 ? "true" : "false"}>
            <input
              className="film-name"
              aria-label={film.talkLeftLabel}
              placeholder={film.talkLeft}
              value={value.left}
              onChange={/** ev 取出 target.value 写进 left。 */ (ev) => props.onChange({ ...value, left: ev.target.value })}
            />
            <input
              className="film-line"
              aria-label={film.talkAskLabel}
              placeholder={film.lineHold}
              value={value.ask}
              onChange={/** ev 取出 target.value 写进 ask。 */ (ev) => props.onChange({ ...value, ask: ev.target.value })}
            />
          </div>
          <div className="film-seat" data-side="right" data-speaking={props.beat == null || props.beat > 0 ? "true" : "false"}>
            <input
              className="film-name"
              aria-label={film.talkRightLabel}
              placeholder={film.talkRight}
              value={value.right}
              onChange={/** ev 取出 target.value 写进 right。 */ (ev) => props.onChange({ ...value, right: ev.target.value })}
            />
            <input
              className="film-line"
              aria-label={film.talkReplyLabel}
              placeholder={film.lineHold}
              value={value.reply}
              onChange={/** ev 取出 target.value 写进 reply。 */ (ev) => props.onChange({ ...value, reply: ev.target.value })}
            />
          </div>
        </div>
      ) : null}
      {face.kind === "lines" ? (
        <ol className="film-strip" aria-label={face.object}>
          {props.mode === "talking" ? (
            <li className="film-slot">
              <em aria-hidden>{film.talkingWho}</em>
              <input
                aria-label={film.talkingWho}
                placeholder={film.talkingLeft}
                value={value.left}
                onChange={/** ev 取出 target.value 写进 left。 */ (ev) => props.onChange({ ...value, left: ev.target.value })}
              />
            </li>
          ) : null}
          {indexes.map(/** index 是这一格。只改当前这一拍时，输入只留下这一格。 */ (index) => {
            const row = value.rows[index] ?? "";
            return (
            <li key={index} className="film-slot" data-wait={props.mode === "interview" && index > 0 && !row.trim() ? "true" : "false"}>
              <em aria-hidden>{index + 1}</em>
              <input
                aria-label={place(face.place, index + 1)}
                placeholder={props.mode === "interview" ? "?" : film.lineHold}
                value={row}
                onChange={/** ev 取出 target.value，连同 index 交给 setRow。 */ (ev) => setRow(index, ev.target.value)}
              />
            </li>
            );
          })}
        </ol>
      ) : null}
      {props.hideReading ? null : (
      <ol className="film-reading" aria-label={film.reading} data-count={beats.length}>
        {beats.map(/** beat 画出这一拍已经留下的行。index 只用来做键。 */ (beat, index) => (
          <li key={index}>
            {beat.lines.map(/** line 新出现的用 data-fresh 标出来。lineIndex 只用来做键。 */ (line, lineIndex) => (
              <span key={lineIndex} className="film-beat-line" data-fresh={line.fresh ? "true" : "false"}>{line.text}</span>
            ))}
          </li>
        ))}
      </ol>
      )}
      {props.onCommit ? (
        <button
          type="button"
          className="film-commit"
          data-tone={props.commitTone === "next" ? "next" : "make"}
          data-testid={props.testId ? `${props.testId}-go` : undefined}
          disabled={!!props.busy}
          onClick={props.onCommit}
        >
          {props.commitLabel}
        </button>
      ) : null}
    </section>
  );
}
