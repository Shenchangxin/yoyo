import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { isDuplicateStamp, pasteSignature, shouldSkipProgrammaticInsert } from "./input-guard.ts";

describe("pasteSignature", () => {
  it("includes file count so a screenshot paste is distinct from its URL", () => {
    assert.equal(pasteSignature("https://x", 0), "0:https://x");
    assert.notEqual(pasteSignature("https://x", 1), pasteSignature("https://x", 0));
  });
});

describe("isDuplicateStamp", () => {
  const target = {};
  const a = { at: 10, target, signature: "0:sk-live" };

  it("drops the same paste into the same field within the window", () => {
    assert.equal(isDuplicateStamp(a, { at: 40, target, signature: "0:sk-live" }), true);
  });

  it("allows a later paste of the same text", () => {
    assert.equal(isDuplicateStamp(a, { at: 80, target, signature: "0:sk-live" }, 48), false);
  });

  it("allows paste into a different field", () => {
    assert.equal(isDuplicateStamp(a, { at: 20, target: {}, signature: "0:sk-live" }), false);
  });
});

describe("shouldSkipProgrammaticInsert", () => {
  it("skips Wails insertText after native paste already wrote the clipboard", () => {
    assert.equal(
      shouldSkipProgrammaticInsert({
        inserted: "sk-abc",
        fieldValue: "sk-abc",
        caret: 6,
        lastPasteText: "sk-abc",
        elapsedMs: 20,
      }),
      true,
    );
  });

  it("does not skip when the field is still empty (menu-only paste)", () => {
    assert.equal(
      shouldSkipProgrammaticInsert({
        inserted: "sk-abc",
        fieldValue: "",
        caret: 0,
        lastPasteText: "sk-abc",
        elapsedMs: 20,
      }),
      false,
    );
  });

  it("does not skip an intentional second paste after the window", () => {
    assert.equal(
      shouldSkipProgrammaticInsert({
        inserted: "sk-abc",
        fieldValue: "sk-abc",
        caret: 6,
        lastPasteText: "sk-abc",
        elapsedMs: 200,
      }),
      false,
    );
  });
});
