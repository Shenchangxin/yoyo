/**
 * 本文件拥有成片几拍的断言。不改服务端。
 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { filmCommission, filmReading, type FilmCast } from "./film-cast.ts";

const cast: FilmCast = {
  subject: "Three things I use",
  rows: ["Phone", "Headphones", "Mug"],
  left: "Her",
  right: "Him",
  ask: "Do you actually get up early?",
  reply: "I do. I'm still tired.",
};

describe("film reading", () => {
  /** 排行榜后一拍仍留着前面的名次。 */
  it("keeps earlier ranking places", () => {
    const beats = filmReading("ranking", cast);
    assert.deepEqual(beats[0].lines.map((line) => line.text), ["1  Phone"]);
    assert.equal(beats[0].lines[0].fresh, true);
    assert.deepEqual(beats[2].lines.map((line) => line.text), ["1  Phone", "2  Headphones", "3  Mug"]);
    assert.deepEqual(beats[2].lines.map((line) => line.fresh), [false, false, true]);
  });

  /** 对谈先问，答的时候问还在。 */
  it("keeps the question when the reply arrives", () => {
    const beats = filmReading("podcast", cast);
    assert.equal(beats[0].lines[0].text, "Do you actually get up early?");
    assert.deepEqual(beats[1].lines.map((line) => line.text), [
      "Do you actually get up early?",
      "I do. I'm still tired.",
    ]);
    assert.deepEqual(beats[1].lines.map((line) => line.fresh), [false, true]);
  });

  /** 口播里同一个人留在每一拍上。 */
  it("keeps the speaker while the line changes", () => {
    const beats = filmReading("talking", { ...cast, left: "Me", rows: ["I almost left", "The light changed", "So I stayed"] });
    assert.equal(beats[0].lines[0].text, "Me");
    assert.equal(beats[0].lines[0].fresh, false);
    assert.equal(beats[2].lines[0].text, "Me");
    assert.equal(beats[2].lines[1].text, "So I stayed");
    assert.equal(beats[2].lines[1].fresh, true);
  });

  /** 旁白每一拍只留下当前画面。 */
  it("replaces the picture under a narration", () => {
    const beats = filmReading("narration", cast);
    assert.deepEqual(beats[0].lines.map((line) => line.text), ["Phone"]);
    assert.deepEqual(beats[2].lines.map((line) => line.text), ["Mug"]);
  });

  /** 街采只揭开到这一拍，后面仍是问号。 */
  it("reveals interview answers one at a time", () => {
    const beats = filmReading("interview", cast);
    assert.deepEqual(beats[0].lines.map((line) => line.text), ["Phone", "?", "?"]);
    assert.deepEqual(beats[0].lines.map((line) => line.fresh), [true, false, false]);
    assert.deepEqual(beats[2].lines.map((line) => line.text), ["Phone", "Headphones", "Mug"]);
    assert.equal(beats[2].lines[2].fresh, true);
  });
});

describe("film commission", () => {
  /** 简报点名 Hypit，并带上榜上的三名。 */
  it("hands the ranking to Hypit", () => {
    const text = filmCommission("ranking", cast, "Earlier places stay on the board.");
    assert.match(text, /@skill:hypit/);
    assert.match(text, /Format: ranking/);
    assert.match(text, /1\. Phone/);
    assert.match(text, /3\. Mug/);
    assert.match(text, /assemble an mp4/);
    assert.doesNotMatch(text, /Ask:/);
    assert.doesNotMatch(text, /Hypit Studio/);
  });

  /** 对谈简报带上两个人和一问一答。 */
  it("hands the conversation to Hypit", () => {
    const text = filmCommission("podcast", cast, "One asks. The other answers.");
    assert.match(text, /Left: Her/);
    assert.match(text, /Reply: I do\. I'm still tired\./);
  });

  /** 中文界面传进中文叫法，简报整段是中文，片型写名字不写内部 id。 */
  it("writes the brief in the labels it is given", () => {
    const text = filmCommission("ranking", cast, "前面的名次留在榜上。", {
      intro: "按这个片型做一条短片。", format: "片型", relation: "要留住的关系", subject: "主题",
      left: "一方", ask: "问", right: "另一方", reply: "答", speaker: "说话的人",
      outro: "开一条短片，合成 mp4。用到生图或配音之前先问我。", modeName: "排行榜",
    });
    assert.match(text, /@skill:hypit/);
    assert.match(text, /片型：排行榜/);
    assert.match(text, /要留住的关系：前面的名次留在榜上。/);
    assert.doesNotMatch(text, /Format|ranking/);
  });
});
