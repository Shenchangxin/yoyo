import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { buildPageTree } from "./tree.ts";

describe("buildPageTree", () => {
  it("nests children under parents", () => {
    const tree = buildPageTree([
      { id: "a", title: "A", revision: 1 },
      { id: "b", title: "B", revision: 1, parent_id: "a" },
    ]);
    assert.equal(tree.length, 1);
    assert.equal(tree[0]!.id, "a");
    assert.equal(tree[0]!.children[0]!.id, "b");
  });

  it("keeps cyclic parents visible as roots", () => {
    const tree = buildPageTree([
      { id: "a", title: "A", revision: 1, parent_id: "b" },
      { id: "b", title: "B", revision: 1, parent_id: "a" },
    ]);
    const ids = tree.map((n) => n.id).sort();
    assert.deepEqual(ids, ["a", "b"]);
    assert.equal(tree.every((n) => n.children.length === 0), true);
  });

  it("treats self-parent as a root", () => {
    const tree = buildPageTree([{ id: "a", title: "A", revision: 1, parent_id: "a" }]);
    assert.equal(tree.length, 1);
    assert.equal(tree[0]!.children.length, 0);
  });
});
