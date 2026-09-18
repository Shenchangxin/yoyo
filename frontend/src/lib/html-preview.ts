export function looksLikeHTML(raw: string): boolean {
  const t = raw.trim().slice(0, 400);
  if (!t) return false;
  const lower = t.toLowerCase();
  return lower.startsWith("<!doctype html") || lower.startsWith("<html") || lower.includes("<html") || lower.includes("mcp-ui");
}

export function looksLikePDF(path: string, mime = ""): boolean {
  return /\.pdf$/i.test(path) || mime.toLowerCase() === "application/pdf";
}

export function extractHTML(raw: string): string {
  const t = raw.trim();
  if (looksLikeHTML(t)) return t;
  return "";
}
