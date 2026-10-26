/**
 * 拥有：从当前地址判断是否为伴随窗口，以及弹出对话的 id。
 * 不拥有：窗口的打开与关闭，以及伴随表面的布局。
 */

/** isCompanionSurface 在查询串或 hash 里看到 surface=companion 时为真。地址解析失败时为假。 */
export function isCompanionSurface(): boolean {
  try {
    const q = new URLSearchParams(window.location.search).get("surface");
    if (q === "companion") return true;
    const hash = window.location.hash.replace(/^#/, "");
    return new URLSearchParams(hash).get("surface") === "companion";
  } catch {
    return false;
  }
}

/** readPopoutId 读取 popout。查询串没有时，再读 hash 里的 popout 或 thread。解析失败时返回空串。 */
export function readPopoutId(): string {
  try {
    const q = new URLSearchParams(window.location.search).get("popout");
    if (q) return q;
    const hash = window.location.hash.replace(/^#/, "");
    const hp = new URLSearchParams(hash);
    return hp.get("popout") || hp.get("thread") || "";
  } catch {
    return "";
  }
}
