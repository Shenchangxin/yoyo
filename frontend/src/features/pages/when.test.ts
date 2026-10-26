/** 拥有页面时间和字数的断言。不拥有实现。不改服务端。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { formatPageWhen, pageMeasure } from "./when.ts";

describe("formatPageWhen", () => {
  /** 无效时刻写成空。 */
  it("returns empty for invalid timestamps", () => {
    assert.equal(formatPageWhen(""), "");
    assert.equal(formatPageWhen("not-a-date"), "");
  });

  /** 近几天的笔记用相对天数。 */
  it("uses relative days for recent notes", () => {
    const iso = new Date(Date.now() - 3 * 86400000).toISOString();
    assert.match(formatPageWhen(iso, "en"), /day/i);
  });
});

describe("pageMeasure", () => {
  /** 中文按字数。 */
  it("counts CJK as characters", () => {
    const got = pageMeasure("你好世界", "zh-CN");
    assert.equal(got.kind, "chars");
    assert.equal(got.n, 4);
  });

  /** 英文按词数。 */
  it("counts English as words", () => {
    const got = pageMeasure("hello world", "en");
    assert.equal(got.kind, "words");
    assert.equal(got.n, 2);
  });
});
