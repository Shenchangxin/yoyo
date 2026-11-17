import assert from "node:assert/strict";
import test from "node:test";
import { memoryPlaceStore, readSessionPlace, writeSessionPlace } from "./session-place.ts";

test("session place survives a fresh read", () => {
  const store = memoryPlaceStore();
  writeSessionPlace({
    sessionId: "main",
    following: false,
    anchor: "seq:12",
    seq: 12,
    aliases: ["seq:12", "u1"],
    top: 640,
  }, store);
  const got = readSessionPlace("main", store);
  assert.equal(got?.following, false);
  assert.equal(got?.anchor, "seq:12");
  assert.equal(got?.seq, 12);
  assert.equal(got?.top, 640);
  assert.deepEqual(got?.aliases, ["seq:12", "u1"]);
  assert.equal(readSessionPlace("other", store), null);
});

test("following place is kept", () => {
  const store = memoryPlaceStore();
  writeSessionPlace({ sessionId: "main", following: true, anchor: "", seq: 0, aliases: [], top: 800 }, store);
  const followed = readSessionPlace("main", store);
  assert.equal(followed?.following, true);
  assert.equal(followed?.top, 0);
});
