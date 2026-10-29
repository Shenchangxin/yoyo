import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { learnBindingMatches, learnScopeSpec, matchLearnThread, readLearnScopeMap } from "./learn-scope.ts";

/**
 * 拥有：学习子模块的助手钥匙互不占用。
 * 不拥有：会话的创建和界面。
 */

describe("learnScopeSpec", () => {
  /** 四条泳道的钥匙不同。路径、材料和笔记再各自拆开。 */
  it("keeps lanes and selections on separate keys", () => {
    assert.equal(learnScopeSpec({ lane: "today" }).scope, "today");
    assert.equal(learnScopeSpec({ lane: "practice" }).scope, "practice");
    assert.equal(learnScopeSpec({ lane: "study" }).scope, "study");
    assert.equal(learnScopeSpec({ lane: "study", pathId: "p1" }).scope, "path:p1");
    assert.equal(learnScopeSpec({ lane: "library", shelf: "notes" }).scope, "notes");
    assert.equal(learnScopeSpec({ lane: "library", libraryId: "lib", materialId: "m1" }).scope, "material:m1");
    assert.equal(learnScopeSpec({ lane: "library", libraryId: "lib", bookId: "b1", shelf: "books" }).scope, "book:b1");
    assert.equal(learnScopeSpec({ lane: "library", libraryId: "lib", shelf: "materials" }).scope, "library:lib");
  });

  /** 学习泳道不借用文库里打开的材料。 */
  it("prefers the study path over a leftover material", () => {
    const spec = learnScopeSpec({ lane: "study", pathId: "p1", materialId: "m1", libraryId: "lib" });
    assert.equal(spec.scope, "path:p1");
    assert.equal(spec.materialId, undefined);
  });
});

describe("matchLearnThread", () => {
  /** 记住的会话如果绑到了别的路径，不能拿来冒充这一条。 */
  it("does not reuse a thread bound to another path", () => {
    const rows = [
      { id: "a", pathId: "other" },
      { id: "b", pathId: "p1" },
    ];
    const spec = learnScopeSpec({ lane: "study", pathId: "p1" });
    assert.equal(matchLearnThread(rows, spec, "a")?.id, "b");
    assert.equal(learnBindingMatches(rows[0], spec), false);
  });

  /** 今日对话不能占用已经绑了路径的会话。 */
  it("keeps an unbound lane off a bound thread", () => {
    const spec = learnScopeSpec({ lane: "today" });
    assert.equal(matchLearnThread([{ id: "a", pathId: "p1" }], spec, "a"), null);
    assert.equal(matchLearnThread([{ id: "c" }], spec, "c")?.id, "c");
  });
});

describe("readLearnScopeMap", () => {
  /** 坏 JSON 和空值都变成空映射。 */
  it("drops corrupt storage", () => {
    assert.deepEqual(readLearnScopeMap(null), {});
    assert.deepEqual(readLearnScopeMap("{"), {});
    assert.deepEqual(readLearnScopeMap(JSON.stringify({ today: "s1", bad: 1 })), { today: "s1" });
  });
});
