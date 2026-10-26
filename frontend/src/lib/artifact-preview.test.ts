/**
 * 拥有：改文件工具的判定，以及替换差异里相同前后缀留作上下文的约定。
 * 不拥有：预览卡片的渲染。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { artifactPreviewOpen, artifactView, isCodeChangeTool, strReplaceToDiff } from "./artifact-preview.ts";
import type { Item } from "./protocol.ts";

/** 一条只带参数的工具调用。 */
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
  /** 写文件、新建、替换、编辑和打补丁都算代码变更。 */
  it("marks file-mutating tools as code-change bubbles", () => {
    for (const name of ["write_file", "create_file", "str_replace", "edit_file", "apply_patch"]) {
      assert.equal(isCodeChangeTool(name), true, name);
    }
  });

  /** 读文件、shell 和浏览器工具不算代码变更。 */
  it("leaves reads, shell, and browser tools alone", () => {
    for (const name of ["read_file", "shell", "browser_open", "web_fetch"]) {
      assert.equal(isCodeChangeTool(name), false, name);
    }
  });
});

describe("strReplaceToDiff", () => {
  /** 相同的前后缀留作上下文，只有中间那一行标成减号和加号。 */
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
  /** 有真实编辑的 str_replace 算预览，卡片才能折叠或打开。 */
  it("a str_replace with real edits has a preview, so the card can be folded or opened", () => {
    const view = artifactView(call("str_replace", { path: "app.js", old_str: "a", new_str: "b" }));
    assert.equal(artifactPreviewOpen(view), true);
    assert.equal(isCodeChangeTool("str_replace"), true);
  });
});
