import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  commitEpisodeTitle,
  commitSeriesTitle,
  isPlaceholderEpisode,
  isPlaceholderSeries,
  looksLikeSeries,
  shownEpisodeTitle,
  shownSeriesTitle,
} from "./drama-lib.ts";

describe("drama placeholder titles", () => {
  it("maps stored English placeholders to the current locale", () => {
    assert.equal(shownSeriesTitle("Untitled", "新短剧"), "新短剧");
    assert.equal(shownSeriesTitle("Untitled series", "新短剧"), "新短剧");
    assert.equal(shownSeriesTitle("New series", "New short drama"), "New short drama");
    assert.equal(shownSeriesTitle("", "新短剧"), "新短剧");
    assert.equal(shownSeriesTitle("雨巷", "新短剧"), "雨巷");
  });

  it("names untitled episodes from the episode number", () => {
    assert.equal(shownEpisodeTitle("Untitled episode", 2, "第 {n} 集"), "第 2 集");
    assert.equal(shownEpisodeTitle("Episode 1", 1, "第 {n} 集"), "第 1 集");
    assert.equal(shownEpisodeTitle("第 3 集", 3, "Episode {n}"), "Episode 3");
    assert.equal(shownEpisodeTitle("巷口夜", 1, "第 {n} 集"), "巷口夜");
  });

  it("does not persist placeholder strings as real names", () => {
    assert.equal(isPlaceholderSeries("Untitled series"), true);
    assert.equal(isPlaceholderEpisode("Untitled episode"), true);
    assert.equal(commitSeriesTitle("新短剧"), "");
    assert.equal(commitEpisodeTitle("Episode 4"), "");
    assert.equal(commitSeriesTitle("雨巷"), "雨巷");
  });
});

describe("series detection", () => {
  it("keeps a short chapter on the one-episode path", () => {
    assert.equal(looksLikeSeries("林小雨在巷口等车。"), false);
  });

  it("treats a long book or two chapters as a series", () => {
    assert.equal(looksLikeSeries("林小雨在巷口等车。".repeat(280)), true);
    assert.equal(looksLikeSeries("第一章 雨\n" + "她等了很久。".repeat(50) + "\n第二章 夜\n" + "他没有来。".repeat(50)), true);
  });
});
