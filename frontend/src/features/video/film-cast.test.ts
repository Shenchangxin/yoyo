/**
 * 本文件拥有成片几拍的断言。不改服务端。
 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { filmReading, type FilmCast } from "./film-cast.ts";

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
