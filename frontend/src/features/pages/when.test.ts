import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { formatPageWhen, pageMeasure } from "./when.ts";

describe("formatPageWhen", () => {
  it("returns empty for invalid timestamps", () => {
    assert.equal(formatPageWhen(""), "");
    assert.equal(formatPageWhen("not-a-date"), "");
  });

  it("uses relative days for recent notes", () => {
    const iso = new Date(Date.now() - 3 * 86400000).toISOString();
    assert.match(formatPageWhen(iso, "en"), /day/i);
  });
});

describe("pageMeasure", () => {
  it("counts CJK as characters", () => {
    const got = pageMeasure("你好世界", "zh-CN");
    assert.equal(got.kind, "chars");
    assert.equal(got.n, 4);
  });

  it("counts English as words", () => {
    const got = pageMeasure("hello world", "en");
    assert.equal(got.kind, "words");
    assert.equal(got.n, 2);
  });
});
