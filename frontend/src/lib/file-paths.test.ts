import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { filePathsIn, hidePaths, isBrowserPreviewPath, previewTarget } from "./file-paths.ts";

describe("file paths", () => {
  it("keeps a windows path inside backticks and drops the following words", () => {
    const text = "双击 `C:\\Users\\scx\\Desktop\\game\\gugu-mart\\build\\app\\index.html` 即可玩";
    assert.deepEqual(filePathsIn(text), ["C:\\Users\\scx\\Desktop\\game\\gugu-mart\\build\\app\\index.html"]);
  });

  it("keeps a relative file and a backticked name", () => {
    const text = "看 design/GAME_DESIGN.md，还有 `app/index.html`。";
    assert.deepEqual(filePathsIn(text), ["design/GAME_DESIGN.md", "app/index.html"]);
  });

  it("keeps a path that goes through a Chinese directory", () => {
    const text = "实名路径 C:\\Users\\scx\\Desktop\\视频素材\\方案.html ，清单是 视频素材清单.csv";
    assert.deepEqual(filePathsIn(text), [
      "C:\\Users\\scx\\Desktop\\视频素材\\方案.html",
      "视频素材清单.csv",
    ]);
  });

  it("keeps a delivered windows path that is not wrapped in backticks", () => {
    const text = "咕咕便利店可试玩切片已交付：双击 C:\\Users\\scx\\Desktop\\game\\game-adaptations\\gugu-mart\\build\\app\\index.html 即可玩三日流程";
    assert.deepEqual(filePathsIn(text), ["C:\\Users\\scx\\Desktop\\game\\game-adaptations\\gugu-mart\\build\\app\\index.html"]);
    assert.equal(hidePaths(text), "咕咕便利店可试玩切片已交付：双击 index.html 即可玩三日流程");
  });

  it("hides the directory and leaves the file name", () => {
    const text = "打开 C:\\Users\\scx\\Desktop\\视频素材\\方案.html";
    assert.equal(hidePaths(text), "打开 方案.html");
  });

  it("splits an absolute html path onto its parent directory", () => {
    const target = previewTarget("C:\\ticket", "C:\\game\\build\\app\\index.html");
    assert.equal(target.workspace, "C:\\game\\build\\app");
    assert.equal(target.path, "index.html");
    assert.equal(isBrowserPreviewPath(target.path), true);
  });
});
