/**
 * 拥有：从工具输出里读出页面审阅编号、哈希和标题。
 * 不拥有：审阅卡片和请求。只读文本，不改服务端。
 */

/** parsePageReviewBlob 从等待审阅的文本里读出编号、哈希和标题。没有编号或解析失败则空。 */
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
