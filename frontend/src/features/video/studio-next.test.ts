import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { studioNext } from "./studio-next.ts";

describe("studio next action", () => {
  it("sends an unbound desk to radar or topics", () => {
    assert.equal(studioNext(null, "radar").key, "collect");
    assert.equal(studioNext(null, "topics").pane, "topics");
    assert.equal(studioNext(undefined, "assemble").key, "produce");
  });

  it("walks a bound short through VO, stills, cut, pack", () => {
    const base = { id: "1", title: "t" };
    assert.equal(studioNext(base, "assemble").key, "script");
    assert.equal(studioNext({ ...base, script: "hello", script_status: "needs_review" }, "radar").gate, "script");
    assert.equal(studioNext({ ...base, script: "hello", script_status: "approved" }, "assemble").key, "audio");
    assert.equal(studioNext({ ...base, script: "hello", script_status: "approved", skip_audio: true }, "assemble").key, "visuals");
    assert.equal(studioNext({
      ...base,
      script: "hello",
      script_status: "approved",
      narration_hash: "n",
      shots: [{ image_hash: "h" }],
    }, "assemble").key, "compose");
    assert.equal(studioNext({
      ...base,
      script: "hello",
      script_status: "approved",
      narration_hash: "n",
      shots: [{ image_hash: "h" }],
      result_hash: "v",
    }, "assemble").key, "pack");
    assert.equal(studioNext({
      ...base,
      script: "hello",
      script_status: "approved",
      narration_hash: "n",
      shots: [{ image_hash: "h" }],
      result_hash: "v",
      delivery: { status: "publishable" },
    }, "assemble").key, "open");
  });
});
