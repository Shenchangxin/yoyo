import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { pickAnimations, type StandDirection } from "./animations.ts";

const clips = new Map<string, number>([
  ["wait", 4],
  ["smile1", 3],
  ["shy1", 3],
  ["surp1", 2],
  ["sad1", 3],
  ["anger1", 2],
  ["think", 3],
  ["face_wait", 1],
  ["face_smile", 1],
  ["face_shy", 1],
  ["face_surp", 1],
  ["face_sad", 1],
  ["face_anger", 1],
  ["face_serious", 1],
  ["blank", 0],
  ["lip_smile", 0.4],
  ["lip_wait", 0.4],
  ["yes", 0.8],
  ["no", 0.8],
  ["eye_front", 1],
]);

function direction(change: Partial<StandDirection> = {}): StandDirection {
  return { mood: "calm", speaking: false, looking: false, nod: null, ...change };
}

describe("stand clips", () => {
  it("plays the body, face, and mouth for the mood", () => {
    const happy = pickAnimations(direction({ mood: "happy", speaking: true }), clips, false);
    assert.equal(happy.body, "smile1");
    assert.equal(happy.face, "face_smile");
    assert.deepEqual(happy.mouth, { name: "lip_smile", open: true });
    const calm = pickAnimations(direction({ looking: true }), clips, false);
    assert.equal(calm.body, "wait");
    assert.equal(calm.eye, "eye_front");
    assert.equal(calm.mouth?.open, false);
  });

  it("holds the body on wait when motion is reduced, and nods once per id", () => {
    const held = pickAnimations(direction({ mood: "angry" }), clips, true);
    assert.equal(held.body, "wait");
    assert.equal(held.face, "face_anger");
    const nod = pickAnimations(direction({ nod: { kind: "yes", id: 3 } }), clips, false);
    assert.deepEqual(nod.nod, { name: "yes", id: 3 });
  });
});
