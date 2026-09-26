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
