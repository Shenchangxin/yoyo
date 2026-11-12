import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { dayKey, dayLabel } from "./day-label.ts";

/**
 * 拥有：主会话按天分隔的称呼：今天、昨天按本地日历日算，不按相隔 24 小时算；更早的写月日和星期，跨年带年份。
 * 不拥有：分隔线画在哪。
 */

const words = { today: "今天", yesterday: "昨天" };
const at = (y: number, m: number, d: number, h = 12, min = 0) => new Date(y, m - 1, d, h, min).getTime();

describe("dayKey", () => {
  it("is the local calendar day", () => {
    assert.equal(dayKey(at(2026, 11, 12, 0, 1)), "2026-11-12");
    assert.equal(dayKey(at(2026, 11, 12, 23, 59)), "2026-11-12");
    assert.equal(dayKey(at(2026, 1, 3)), "2026-01-03");
  });

  it("is empty for a bad time", () => {
    assert.equal(dayKey(Number.NaN), "");
    assert.equal(dayKey(0), "");
  });
});

describe("dayLabel", () => {
  const now = at(2026, 11, 12, 9, 0);

  it("says today and yesterday by calendar day, not by elapsed hours", () => {
    assert.equal(dayLabel(at(2026, 11, 12, 0, 5), words, "zh-CN", now), "今天");
    // 昨晚 23:50 离现在不到 10 小时，仍然是昨天。
    assert.equal(dayLabel(at(2026, 11, 11, 23, 50), words, "zh-CN", now), "昨天");
    // 前天上午离现在不到 48 小时，不是昨天。
    assert.notEqual(dayLabel(at(2026, 11, 10, 10, 0), words, "zh-CN", now), "昨天");
  });

  it("writes the date in the app language", () => {
    const zh = dayLabel(at(2026, 11, 10), words, "zh-CN", now);
    assert.match(zh, /11月10日/);
    assert.match(zh, /周二/);
    const en = dayLabel(at(2026, 11, 10), words, "en-US", now);
    assert.match(en, /Nov 10/);
    assert.match(en, /Tue/);
    assert.doesNotMatch(en, /2026/);
  });

  it("adds the year for another year", () => {
    assert.match(dayLabel(at(2025, 12, 31), words, "zh-CN", now), /2025/);
    assert.match(dayLabel(at(2025, 12, 31), words, "en-US", now), /2025/);
  });

  it("handles yesterday across a month and a year", () => {
    assert.equal(dayLabel(at(2025, 12, 31, 22), words, "zh-CN", at(2026, 1, 1, 8)), "昨天");
  });

  it("is empty for a bad time", () => {
    assert.equal(dayLabel(Number.NaN, words, "zh-CN", now), "");
  });
});
