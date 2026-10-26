import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { snapUiScale } from "./scale.ts";

/**
 * 拥有：缩放吸附对三个档位、旧滑条值和空值的约定。
 * 不拥有：缩放的存储，以及文档上的 CSS 变量。
 */

describe("snapUiScale", () => {
  /** 三个档位保持原值。 */
  it("keeps the three Windows-aligned steps", () => {
    assert.equal(snapUiScale(1), 1);
    assert.equal(snapUiScale(1.25), 1.25);
    assert.equal(snapUiScale(1.5), 1.5);
  });

  /** 旧的 0.05 滑条值吸到最近档位。 */
  it("migrates the old 0.05 slider onto the nearest step", () => {
    assert.equal(snapUiScale(0.85), 1);
    assert.equal(snapUiScale(1.1), 1);
    assert.equal(snapUiScale(1.2), 1.25);
    assert.equal(snapUiScale(1.35), 1.25);
    assert.equal(snapUiScale(1.4), 1.5);
  });

  /** 缺省、零和非数都视为 1。 */
  it("rejects empty values", () => {
    assert.equal(snapUiScale(), 1);
    assert.equal(snapUiScale(0), 1);
    assert.equal(snapUiScale(Number.NaN), 1);
  });
});
