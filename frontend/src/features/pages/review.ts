export function parsePageReviewBlob(blob: string): { id: string; hash: string; title: string } | null {
  const m = blob.match(/awaiting page review\s*(\{[^{}]*\})/i) || blob.match(/(\{"review_id"[^{}]*\})/);
  let raw = m?.[1] || "";
  if (!raw) {
    const n = blob.match(/(\{[^{}]*"kind"\s*:\s*"page\.save"[^{}]*\})/);
    raw = n?.[1] || "";
  }
  if (!raw) return null;
  try {
    const j = JSON.parse(raw);
    const id = String(j.review_id || j.id || "");
    if (!id) return null;
    return { id, hash: String(j.hash || ""), title: String(j.title || "Page") };
  } catch {
    return null;
  }
}
