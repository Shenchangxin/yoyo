import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { en } from "./copy.ts";
import { classifyError, errorCopy } from "./error.ts";

// Kinds the runtime emits (internal/runtime/errors.go). Each must survive
// classification and map to its own copy; a missing one shows the generic title.
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
    it(`${kind} is accepted as-is`, () => {
      assert.equal(classifyError("boom", { kind }).kind, kind);
    });

    it(`${kind} has its own title, not the generic one`, () => {
      const { title, hint } = errorCopy(en.transcript, classifyError("boom", { kind }).kind);
      assert.notEqual(title, en.transcript.errorUnknown);
      assert.ok(title.length > 0 && hint.length > 0);
    });
  }

  it("model errors keep the backend's retryable flag", () => {
    assert.equal(classifyError("x", { kind: "model" }).retryable, true);
    assert.equal(classifyError("x", { kind: "model", retryable: false }).retryable, false);
  });

  it("an unknown kind string falls back to inference from the text", () => {
    assert.equal(classifyError("429 too many requests", { kind: "bogus" }).kind, "rate_limit");
  });
});
