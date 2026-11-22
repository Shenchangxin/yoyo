import assert from "node:assert/strict";
import test from "node:test";
import { isReadingPlace, memoryPlaceStore, readSessionPlace, resetSessionPlaces, scrollTopForAnchor, writeSessionPlace } from "./session-place.ts";

test("session place survives a fresh read", () => {
  const store = memoryPlaceStore();
  writeSessionPlace({
    sessionId: "main",
    following: false,
    anchor: "row-12",
    seq: 12,
    aliases: ["row-12", "seq:12"],
    offset: 40,
  }, store);
  const got = readSessionPlace("main", store);
  assert.equal(got?.following, false);
  assert.equal(got?.anchor, "row-12");
  assert.equal(got?.seq, 12);
  assert.equal(got?.offset, 40);
  assert.deepEqual(got?.aliases, ["row-12", "seq:12"]);
  assert.equal(isReadingPlace(got), true);
  assert.equal(readSessionPlace("other", store), null);
});

test("following place drops the anchor", () => {
  const store = memoryPlaceStore();
  writeSessionPlace({ sessionId: "main", following: true, anchor: "row", seq: 4, aliases: ["row"], offset: 800 }, store);
  const followed = readSessionPlace("main", store);
  assert.equal(followed?.following, true);
  assert.equal(followed?.anchor, "");
  assert.equal(followed?.offset, 0);
  assert.equal(isReadingPlace(followed), false);
});

test("default store is memory and reset drops it", () => {
  resetSessionPlaces();
  writeSessionPlace({
    sessionId: "main",
    following: false,
    anchor: "row",
    seq: 3,
    aliases: ["row"],
    offset: 40,
  });
  assert.equal(readSessionPlace("main")?.offset, 40);
  resetSessionPlaces();
  assert.equal(readSessionPlace("main"), null);
});

test("anchor offset follows the row start instead of an absolute scrollTop", () => {
  assert.equal(scrollTopForAnchor(500, 40, 2000), 540);
  assert.equal(scrollTopForAnchor(800, 40, 2000), 840);
  assert.equal(scrollTopForAnchor(500, 40, 520), 520);
  assert.equal(scrollTopForAnchor(10, -30, 100), 0);
});
