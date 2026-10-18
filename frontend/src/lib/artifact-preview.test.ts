import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { artifactPreviewOpen, artifactView, isCodeChangeTool, strReplaceToDiff } from "./artifact-preview.ts";
import type { Item } from "./protocol.ts";

function call(name: string, args: Record<string, unknown>): Item {
  return {
    key: `${name}:call`,
    type: "tool_call",
    sessionId: "s",
    source: "agent",
    ts: "2026-10-13T00:00:00.000Z",
    text: "",
    name,
    delta: false,
    payload: { id: `${name}:id`, name, arguments: JSON.stringify(args) },
  } as Item;
}

describe("isCodeChangeTool", () => {
  it("marks file-mutating tools as code-change bubbles", () => {
    for (const name of ["write_file", "create_file", "str_replace", "edit_file", "apply_patch"]) {
      assert.equal(isCodeChangeTool(name), true, name);
    }
  });

  it("leaves reads, shell, and browser tools alone", () => {
    for (const name of ["read_file", "shell", "browser_open", "web_fetch"]) {
      assert.equal(isCodeChangeTool(name), false, name);
    }
  });
});

describe("strReplaceToDiff", () => {
  it("keeps equal prefix/suffix as context instead of rewriting the fragment", () => {
    const oldStr = "a\nb\nc\nd\n";
    const newStr = "a\nB\nc\nd\n";
    const diff = strReplaceToDiff("app.js", oldStr, newStr);
    const lines = diff.split("\n");
    assert.equal(lines.filter((l) => l.startsWith("-") && !l.startsWith("---")).length, 1);
    assert.equal(lines.filter((l) => l.startsWith("+") && !l.startsWith("+++")).length, 1);
    assert.match(diff, /^-b$/m);
    assert.match(diff, /^\+B$/m);
  });
});

describe("code-change cards with a diff still count as previews", () => {
  it("a str_replace with real edits has a preview, so the card can be folded or opened", () => {
    const view = artifactView(call("str_replace", { path: "app.js", old_str: "a", new_str: "b" }));
    assert.equal(artifactPreviewOpen(view), true);
    assert.equal(isCodeChangeTool("str_replace"), true);
  });
});
