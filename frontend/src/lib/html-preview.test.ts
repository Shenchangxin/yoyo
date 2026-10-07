import assert from "node:assert/strict";
import test from "node:test";
import { asPreviewDocument, blobFromBase64, looksLikeHTML, looksLikeHTMLFile, needsBlobPreview } from "./html-preview.ts";

test("html files and documents are recognized", () => {
  assert.equal(looksLikeHTMLFile("web/index.html"), true);
  assert.equal(looksLikeHTMLFile("page.xhtml"), true);
  assert.equal(looksLikeHTMLFile("src/main.go"), false);
  assert.equal(looksLikeHTML("<!doctype html><html></html>"), true);
});

test("asPreviewDocument wraps fragments for the inspector iframe", () => {
  const wrapped = asPreviewDocument("<h1>Hello</h1>");
  assert.match(wrapped, /<!doctype html>/i);
  assert.match(wrapped, /<h1>Hello<\/h1>/);
  assert.match(wrapped, /data-yoyo-preview-scroll/);
  const full = "<!doctype html><html><body>ok</body></html>";
  const previewed = asPreviewDocument(full);
  assert.match(previewed, /data-yoyo-preview-scroll/);
  assert.match(previewed, />ok</);
  assert.equal(asPreviewDocument(previewed), previewed);
});

test("pdf image audio video and docx use blob preview", () => {
  assert.equal(needsBlobPreview("notes.pdf"), true);
  assert.equal(needsBlobPreview("shot.png"), true);
  assert.equal(needsBlobPreview("clip.mp4"), true);
  assert.equal(needsBlobPreview("week.docx"), true);
  assert.equal(needsBlobPreview("src/main.go"), false);
  assert.equal(needsBlobPreview("sheet.xlsx"), false);
});

test("blobFromBase64 yields a typed blob", async () => {
  const blob = await blobFromBase64(btoa("hello"), "text/plain");
  assert.equal(blob.type, "text/plain");
  assert.equal(await blob.text(), "hello");
});
