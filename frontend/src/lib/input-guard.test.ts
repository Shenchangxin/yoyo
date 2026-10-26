/**
 * 拥有：粘贴签名、重复窗口，以及程序化插入何时跳过。
 * 不拥有：守卫在文档上的安装。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { isDuplicateStamp, pasteSignature, shouldSkipProgrammaticInsert } from "./input-guard.ts";

describe("pasteSignature", () => {
  /** 签名带上文件个数，截图粘贴和同一段网址不是同一次。 */
  it("includes file count so a screenshot paste is distinct from its URL", () => {
    assert.equal(pasteSignature("https://x", 0), "0:https://x");
    assert.notEqual(pasteSignature("https://x", 1), pasteSignature("https://x", 0));
  });
});

describe("isDuplicateStamp", () => {
  const target = {};
  const a = { at: 10, target, signature: "0:sk-live" };

  /** 同一目标、同一签名，30 毫秒内算重复。 */
  it("drops the same paste into the same field within the window", () => {
    assert.equal(isDuplicateStamp(a, { at: 40, target, signature: "0:sk-live" }), true);
  });

  /** 超过给定窗口后再粘同一段文字，不算重复。 */
  it("allows a later paste of the same text", () => {
    assert.equal(isDuplicateStamp(a, { at: 80, target, signature: "0:sk-live" }, 48), false);
  });

  /** 换一个目标就不算重复。 */
  it("allows paste into a different field", () => {
    assert.equal(isDuplicateStamp(a, { at: 20, target: {}, signature: "0:sk-live" }), false);
  });
});

describe("shouldSkipProgrammaticInsert", () => {
  /** 光标前已经是刚粘贴的文本时，跳过随后的程序化插入。 */
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

  /** 字段仍是空的时不跳过。那是只有菜单的粘贴。 */
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

  /** 超过 DUPLICATE_INSERT_MS 之后的第二次粘贴不跳过。 */
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
