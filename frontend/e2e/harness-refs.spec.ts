import { expect, test } from "@playwright/test";
import { canaryDirty, harnessNext, parseHarnessRefs, readHarborMetrics, shortHash, stagingDirty } from "../src/lib/harness-refs";
import { labFromTab, surfaceForLab, tabFromLab } from "../src/lib/surface";

test("shortHash trims prefixes and length", () => {
  expect(shortHash("blake3:abcdef1234")).toBe("abcdef1");
  expect(shortHash("deadbeef")).toBe("deadbee");
  expect(shortHash("")).toBe("");
});

test("parseHarnessRefs reads active/staging and leftover names", () => {
  const refs = parseHarnessRefs({
    active: "aaa",
    refs: { active: "aaa", staging: "bbb", canary: "ccc", archive_1: "ddd" },
  });
  expect(refs.active).toBe("aaa");
  expect(refs.staging).toBe("bbb");
  expect(refs.canary).toBe("ccc");
  expect(refs.extra).toEqual([{ name: "archive_1", hash: "ddd" }]);
  expect(stagingDirty(refs)).toBe(true);
  expect(canaryDirty(refs)).toBe(true);
});

test("harnessNext treats canary like a candidate", () => {
  const canary = parseHarnessRefs({ refs: { active: "aaa", canary: "ccc" } });
  expect(canaryDirty(canary)).toBe(true);
  expect(harnessNext(canary, null).action).toBe("eval");
  expect(harnessNext(canary, {
    metrics: { safety_fail: 0, held_in_pass: 2, held_in_total: 2, held_out_pass: 1, held_out_total: 1 },
  }).action).toBe("checkout");
});

test("harnessNext prefers safety block, then checkout, then eval", () => {
  const dirty = parseHarnessRefs({ refs: { active: "aaa", staging: "bbb" } });
  const clean = parseHarnessRefs({ refs: { active: "aaa", staging: "" } });
  expect(harnessNext(dirty, { metrics: { safety_fail: 1, held_in_pass: 1, held_in_total: 1 } }).action).toBe("blocked");
  expect(harnessNext(dirty, {
    metrics: { safety_fail: 0, held_in_pass: 2, held_in_total: 2, held_out_pass: 1, held_out_total: 1 },
  }).action).toBe("checkout");
  expect(harnessNext(dirty, null).action).toBe("eval");
  expect(harnessNext(clean, null).action).toBe("evolve");
});

test("readHarborMetrics accepts both casings", () => {
  const m = readHarborMetrics({ Metrics: { HeldInPass: 1, HeldInTotal: 2 } });
  expect(m?.heldInPass).toBe(1);
  expect(m?.heldInTotal).toBe(2);
});

test("lab and surface mapping stays bijective for the three verbs", () => {
  expect(tabFromLab("harbor")).toBe("prove");
  expect(tabFromLab("evolve")).toBe("propose");
  expect(tabFromLab("harness")).toBe("promote");
  expect(labFromTab("prove")).toBe("harbor");
  expect(labFromTab("overview")).toBe("harness");
  expect(surfaceForLab("agent")).toBe("agent");
  expect(surfaceForLab("harbor")).toBe("harness");
});
