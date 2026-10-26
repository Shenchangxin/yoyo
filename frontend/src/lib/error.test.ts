/**
 * 拥有：运行时每种错误都保留种类并映射到自己的标题。
 * 不拥有：错误卡片的渲染。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { en } from "./copy.ts";
import { classifyError, errorCopy } from "./error.ts";

/** 运行时会发出的种类，见 internal/runtime/errors.go。每一种都要原样通过分类，并映射到自己的文案。缺一种会显示笼统标题。 */
const RUNTIME_KINDS = [
  "overflow",
  "auth",
  "rate_limit",
  "invalid",
  "timeout",
  "canceled",
  "provider",
  "budget",
  "max_turns",
  "stuck",
  "plan_open",
  "model",
];

describe("runtime error kinds", () => {
  for (const kind of RUNTIME_KINDS) {
    /** 载荷里的 kind 原样保留。 */
    it(`${kind} is accepted as-is`, () => {
      assert.equal(classifyError("boom", { kind }).kind, kind);
    });

    /** 每种都有自己的标题和提示，不会落到未知。 */
    it(`${kind} has its own title, not the generic one`, () => {
      const { title, hint } = errorCopy(en.transcript, classifyError("boom", { kind }).kind);
      assert.notEqual(title, en.transcript.errorUnknown);
      assert.ok(title.length > 0 && hint.length > 0);
    });
  }

  /** 模型错误默认可重试。载荷里 retryable 为假时不可重试。 */
  it("model errors keep the backend's retryable flag", () => {
    assert.equal(classifyError("x", { kind: "model" }).retryable, true);
    assert.equal(classifyError("x", { kind: "model", retryable: false }).retryable, false);
  });

  /** 不认识的 kind 改从正文推断。429 收成限流。 */
  it("an unknown kind string falls back to inference from the text", () => {
    assert.equal(classifyError("429 too many requests", { kind: "bogus" }).kind, "rate_limit");
  });
});
