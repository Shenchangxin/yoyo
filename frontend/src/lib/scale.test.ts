import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { snapUiScale } from "./scale.ts";

describe("snapUiScale", () => {
  it("keeps the three Windows-aligned steps", () => {
    assert.equal(snapUiScale(1), 1);
    assert.equal(snapUiScale(1.25), 1.25);
    assert.equal(snapUiScale(1.5), 1.5);
  });

  it("migrates the old 0.05 slider onto the nearest step", () => {
    assert.equal(snapUiScale(0.85), 1);
    assert.equal(snapUiScale(1.1), 1);
    assert.equal(snapUiScale(1.2), 1.25);
    assert.equal(snapUiScale(1.35), 1.25);
    assert.equal(snapUiScale(1.4), 1.5);
  });

  it("rejects empty values", () => {
    assert.equal(snapUiScale(), 1);
    assert.equal(snapUiScale(0), 1);
    assert.equal(snapUiScale(Number.NaN), 1);
  });
});
