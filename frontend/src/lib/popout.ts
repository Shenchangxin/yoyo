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
