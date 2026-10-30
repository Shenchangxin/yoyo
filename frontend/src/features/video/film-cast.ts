/**
 * 本文件拥有排行榜、两人对谈和街头提问上操作者能改的那几行字，以及观众会看到的几拍。
 * 不拥有画面怎么画。写入交给成片接口，这里只记住本地草稿。
 */
import { create } from "zustand";
import type { Copy } from "../../lib/copy";
import type { FilmProject, FilmWord } from "./film-desk";

/** FilmMode 是七种片子。剪视频不在这里。 */
export type FilmMode = "talking" | "drama" | "podcast" | "interview" | "ranking" | "explainer" | "narration";

/** isFilmMode 判断一个模式是不是这七种片子之一。mode 对上联合类型才为真。 */
export function isFilmMode(mode: string): mode is FilmMode {
  return mode === "talking" || mode === "drama" || mode === "podcast" || mode === "interview"
    || mode === "ranking" || mode === "explainer" || mode === "narration";
}

/** FilmCast 是榜的题目和名次、对谈的两个人和一问一答、街采的问题和答案。 */
export type FilmCast = {
  subject: string;
  rows: [string, string, string];
  left: string;
  right: string;
  ask: string;
  reply: string;
};

type FilmCopy = Copy["video"]["film"];

/** castDefaults 用当前文案里的例子填一种片子。操作者还没改过时用它。film 提供例子。mode 决定取哪一组。 */
export function castDefaults(film: FilmCopy, mode: FilmMode): FilmCast {
  const empty = { left: "", right: "", ask: "", reply: "" };
  if (mode === "podcast") {
    return { subject: film.talkSubject, rows: ["", "", ""], left: film.talkLeft, right: film.talkRight, ask: film.talkAsk, reply: film.talkReply };
  }
  if (mode === "drama") {
    return { subject: film.dramaSubject, rows: ["", "", ""], left: film.dramaLeft, right: film.dramaRight, ask: film.dramaAsk, reply: film.dramaReply };
  }
  if (mode === "talking") {
    return { subject: film.talkingSubject, rows: [film.talking1, film.talking2, film.talking3], left: film.talkingLeft, right: "", ask: "", reply: "" };
  }
  if (mode === "interview") {
    return { subject: film.askSubject, rows: [film.ask1, film.ask2, film.ask3], ...empty };
  }
  if (mode === "narration") {
    return { subject: film.narrationSubject, rows: [film.narration1, film.narration2, film.narration3], ...empty };
  }
  if (mode === "explainer") {
    return { subject: film.explainerSubject, rows: [film.explainer1, film.explainer2, film.explainer3], ...empty };
  }
  return { subject: film.rankSubject, rows: [film.rank1, film.rank2, film.rank3], ...empty };
}

/** FilmBeatLine 是观众在某一拍上看见的一行。fresh 为真表示这一拍新出现。 */
export type FilmBeatLine = { text: string; fresh: boolean };

/** FilmBeat 是片子往前走的一拍。lines 里已经出现的内容留着。 */
export type FilmBeat = { lines: FilmBeatLine[] };

/** grow 让后一拍留着前面的行。rows 从第一行累积到当前拍。 */
function grow(rows: FilmCast["rows"]): FilmBeat[] {
  return rows.map(/** _row 不使用。beat 决定已经留下几行。 */ (_row, beat) => ({
    lines: rows.slice(0, beat + 1).map(/** row 有字就带上序号，空着只留序号。index 等于 beat 时是新出现的。 */ (row, index) => ({
      text: row.trim() ? `${index + 1}  ${row.trim()}` : String(index + 1),
      fresh: index === beat,
    })),
  }));
}

/** unveil 一次只揭开到这一拍，后面仍是问号。rows 按顺序揭开。 */
function unveil(rows: FilmCast["rows"]): FilmBeat[] {
  return rows.map(/** _row 不使用。beat 是已经揭开到的那一格。 */ (_row, beat) => ({
    lines: rows.map(/** row 在这一拍已揭开就显示，否则仍是问号。index 等于 beat 时是新出现的。 */ (row, index) => ({
      text: index <= beat ? (row.trim() || "?") : "?",
      fresh: index === beat,
    })),
  }));
}

/** swapPicture 每一拍只留下当前画面。rows 里其他行不留在这一拍上。 */
function swapPicture(rows: FilmCast["rows"]): FilmBeat[] {
  return rows.map(/** row 是这一拍的画面。beat 只用来给空行一个序号。 */ (row, beat) => ({
    lines: [{ text: row.trim() || String(beat + 1), fresh: true }],
  }));
}

/** stayAndSpeak 让同一个人留在每一拍上，话换成这一句。name 是留着的人。rows 是依次说的话。 */
function stayAndSpeak(name: string, rows: FilmCast["rows"]): FilmBeat[] {
  const who = name.trim();
  return rows.map(/** row 是这一拍新说的话。beat 给空话一个序号。 */ (row, beat) => ({
    lines: [
      ...(who ? [{ text: who, fresh: false }] : []),
      { text: row.trim() || String(beat + 1), fresh: true },
    ],
  }));
}

/** pairBeats 先问，答的时候问还在。cast 提供两句和两个人的名字。 */
function pairBeats(cast: FilmCast): FilmBeat[] {
  const ask = cast.ask.trim() || cast.left.trim();
  const reply = cast.reply.trim() || cast.right.trim();
  return [
    { lines: [{ text: ask, fresh: true }] },
    { lines: [{ text: ask, fresh: false }, { text: reply, fresh: true }] },
  ];
}

/** filmReading 按结构排出观众会看到的几拍。mode 决定谁留下、什么换掉。cast 提供当前的字。 */
export function filmReading(mode: FilmMode, cast: FilmCast): FilmBeat[] {
  if (mode === "podcast" || mode === "drama") return pairBeats(cast);
  if (mode === "interview") return unveil(cast.rows);
  if (mode === "narration") return swapPicture(cast.rows);
  if (mode === "talking") return stayAndSpeak(cast.left, cast.rows);
  return grow(cast.rows);
}

/** sameCast 逐项比较两份草稿。a 和 b 的题目、三行、两个人和两句都相同才为真。 */
export function sameCast(a: FilmCast, b: FilmCast) {
  return a.subject === b.subject
    && a.left === b.left
    && a.right === b.right
    && a.ask === b.ask
    && a.reply === b.reply
    && a.rows[0] === b.rows[0]
    && a.rows[1] === b.rows[1]
    && a.rows[2] === b.rows[2];
}

/** castPayload 收成接口要的字段。cast 的三行展开成数组。 */
export function castPayload(cast: FilmCast) {
  return {
    subject: cast.subject,
    rows: [...cast.rows],
    left: cast.left,
    right: cast.right,
    ask: cast.ask,
    reply: cast.reply,
  };
}

/** spokenLines 按行把词拼回句子，并留下第一个说话人。words 按 line_id 分组。 */
export function spokenLines(words: FilmWord[] | undefined) {
  const groups = new Map<string, FilmWord[]>();
  const order: string[] = [];
  for (const word of words || []) {
    const key = word.line_id || word.id || "";
    if (!key) continue;
    if (!groups.has(key)) order.push(key);
    const group = groups.get(key) || [];
    group.push(word);
    groups.set(key, group);
  }
  return order.map(/** id 取出这一行的词，拼成一句，并取第一个说话人。 */ (id) => {
    const group = groups.get(id) || [];
    const parts = group.map(/** word 取出 text。 */ (word) => String(word.text || "")).filter(Boolean);
    const han = parts.some(/** part 含汉字就整句不加空格。 */ (part) => /[\u4e00-\u9fff]/.test(part));
    const speaker = String(group.find(/** word 有 speaker 才留下。 */ (word) => word.speaker)?.speaker || "");
    return { id, speaker, text: parts.join(han ? "" : " ") };
  });
}

/** padRows 补成三行。缺的用空串。list 只取前三项。 */
function padRows(list: string[]): [string, string, string] {
  return [list[0] || "", list[1] || "", list[2] || ""];
}

/** castFromProject 从工程读回操作者能改的字。project 的 structure 决定怎么读标题、条目和台词。 */
export function castFromProject(project: FilmProject): FilmCast {
  const lines = spokenLines(project.words);
  const speakers: string[] = [];
  for (const line of lines) {
    if (line.speaker && !speakers.includes(line.speaker)) speakers.push(line.speaker);
  }
  const labels = (project.entries || []).map(/** entry 取出 label 并去掉空白。 */ (entry) => String(entry.label || "").trim()).filter(Boolean);
  if (project.structure === "podcast" || project.structure === "drama") {
    return {
      subject: project.brief?.product || project.title || "",
      rows: ["", "", ""],
      left: speakers[0] || "",
      right: speakers[1] || "",
      ask: lines[0]?.text || "",
      reply: lines.length > 1 ? lines[lines.length - 1].text : "",
    };
  }
  const texts = lines.map(/** line 取出 text。 */ (line) => line.text);
  if (project.structure === "talking") {
    return {
      subject: project.title || "",
      rows: padRows(labels.length ? labels : texts),
      left: speakers[0] || "",
      right: "",
      ask: "",
      reply: "",
    };
  }
  if (project.structure === "interview" || project.structure === "narration") {
    return {
      subject: project.title || "",
      rows: padRows(labels.length ? labels : texts),
      left: "",
      right: "",
      ask: "",
      reply: "",
    };
  }
  return {
    subject: project.title || "",
    rows: padRows(labels.length ? labels : texts),
    left: "",
    right: "",
    ask: "",
    reply: "",
  };
}

/** useFilmCast 记住三种片子还没提交的草稿。没写过的模式用文案里的例子。 */
export const useFilmCast = create<{
  saved: Partial<Record<FilmMode, FilmCast>>;
  write: (mode: FilmMode, cast: FilmCast) => void;
}>(/** set 按模式写入草稿。 */ (set) => ({
  saved: {},
  /** 记下这一模式的草稿。mode 用作键。cast 写入 saved。 */
  write: (mode, cast) => set(/** state 保留其他模式，换上这一模式。 */ (state) => ({
    saved: { ...state.saved, [mode]: cast },
  })),
}));

/** readCast 读草稿。这一模式还没改过就用文案里的例子。mode 用来读 saved。film 在没有草稿时交给 castDefaults。 */
export function readCast(mode: FilmMode, film: FilmCopy): FilmCast {
  return useFilmCast.getState().saved[mode] ?? castDefaults(film, mode);
}
