/**
 * 拥有：HTML 识别、检查器文档包装，以及哪些文件要走二进制预览。
 * 不拥有：iframe 的挂载。
 */

import assert from "node:assert/strict";
import test from "node:test";
import { asPreviewDocument, blobFromBase64, looksLikeHTML, looksLikeHTMLFile, looksLikeNumberedDump, needsBlobPreview, workspaceHTMLPathFromOpen } from "./html-preview.ts";

/** html 与 xhtml 算文件。doctype 算文档。带行号的倾倒不算。远程 http 和 data 不是工作区路径。 */
test("html files and documents are recognized", () => {
  assert.equal(looksLikeHTMLFile("web/index.html"), true);
  assert.equal(looksLikeHTMLFile("page.xhtml"), true);
  assert.equal(looksLikeHTMLFile("src/main.go"), false);
  assert.equal(looksLikeHTML("<!doctype html><html></html>"), true);
  assert.equal(looksLikeNumberedDump("     1|<!DOCTYPE html>"), true);
  assert.equal(looksLikeHTML("     1|<!DOCTYPE html>\n     2|<html>"), false);
  assert.equal(workspaceHTMLPathFromOpen("pelican-bicycle.html"), "pelican-bicycle.html");
  assert.equal(workspaceHTMLPathFromOpen("https://example.com"), "");
  assert.equal(workspaceHTMLPathFromOpen("data:text/html,hi"), "");
});

/** 片段补上文档壳、滚动条和内容安全策略。完整文档再包一次保持原样。要求脚本时去掉禁止脚本，并加上存储垫片。 */
test("asPreviewDocument wraps fragments for the inspector iframe", () => {
  const wrapped = asPreviewDocument("<h1>Hello</h1>");
  assert.match(wrapped, /<!doctype html>/i);
  assert.match(wrapped, /<h1>Hello<\/h1>/);
  assert.match(wrapped, /data-yoyo-preview-scroll/);
  assert.match(wrapped, /Content-Security-Policy/);
  const full = "<!doctype html><html><body>ok</body></html>";
  const previewed = asPreviewDocument(full);
  assert.match(previewed, /data-yoyo-preview-scroll/);
  assert.match(previewed, />ok</);
  assert.equal(asPreviewDocument(previewed), previewed);
  const live = asPreviewDocument(full, { scripts: true });
  assert.doesNotMatch(live, /script-src 'none'/);
  assert.match(live, /data-yoyo-preview-guard/);
});

/** pdf、图片、视频和 docx 走二进制预览。go 和 xlsx 不走。 */
test("pdf image audio video and docx use blob preview", () => {
  assert.equal(needsBlobPreview("notes.pdf"), true);
  assert.equal(needsBlobPreview("shot.png"), true);
  assert.equal(needsBlobPreview("clip.mp4"), true);
  assert.equal(needsBlobPreview("week.docx"), true);
  assert.equal(needsBlobPreview("src/main.go"), false);
  assert.equal(needsBlobPreview("sheet.xlsx"), false);
});

/** base64 解成带 mime 的 Blob，正文与编码前一致。 */
test("blobFromBase64 yields a typed blob", async () => {
  const blob = await blobFromBase64(btoa("hello"), "text/plain");
  assert.equal(blob.type, "text/plain");
  assert.equal(await blob.text(), "hello");
});
