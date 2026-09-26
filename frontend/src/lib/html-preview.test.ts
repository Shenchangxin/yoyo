import assert from "node:assert/strict";
import test from "node:test";
import { asPreviewDocument, looksLikeHTML, looksLikeHTMLFile } from "./html-preview.ts";

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
  const full = "<!doctype html><html><body>ok</body></html>";
  assert.equal(asPreviewDocument(full), full);
});
