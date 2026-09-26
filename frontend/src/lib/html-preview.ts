export function looksLikeHTML(raw: string): boolean {
  const t = raw.trim().slice(0, 400);
  if (!t) return false;
  const lower = t.toLowerCase();
  return lower.startsWith("<!doctype html") || lower.startsWith("<html") || lower.includes("<html") || lower.includes("mcp-ui");
}

export function looksLikeHTMLFile(path: string): boolean {
  return /\.(html?|xhtml)$/i.test(path || "");
}

/** Wrap a fragment so the inspector iframe can render it. */
export function asPreviewDocument(html: string): string {
  const t = (html || "").trim();
  if (!t) return "";
  if (looksLikeHTML(t)) return t;
  return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head><body>${t}</body></html>`;
}

export function looksLikePDF(path: string, mime = ""): boolean {
  return /\.pdf$/i.test(path) || mime.toLowerCase() === "application/pdf";
}

export function looksLikeOffice(path: string): boolean {
  return /\.(docx|xlsx|pptx|pdf)$/i.test(path || "");
}

export function extractHTML(raw: string): string {
  const t = raw.trim();
  if (looksLikeHTML(t)) return t;
  return "";
}
