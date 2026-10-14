import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { entryMotion } from "./flourish.ts";

describe("entryMotion", () => {
  it("celebrates a finished turn and stays quiet on work faces", () => {
    assert.equal(entryMotion("33"), "celebrate");
    assert.equal(entryMotion("47"), "flourish");
    assert.equal(entryMotion("31"), "bounce");
    assert.equal(entryMotion("48"), "nudge");
    assert.equal(entryMotion("30"), null);
    assert.equal(entryMotion("36"), null);
    assert.equal(entryMotion("02"), null);
  });
});
