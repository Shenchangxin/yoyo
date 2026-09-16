export function displayTitle(title: string | undefined, fallback: string): string {
  const s = (title || "").trim();
  if (!s || /^[.·•…\-–—_/\\]+$/.test(s)) return fallback;
  return s;
}

export function displayWorkspace(path: string | undefined, fallback = ""): string {
  if (!path) return fallback;
  const name = path.replace(/\\/g, "/").split("/").filter(Boolean).pop() || "";
  if (!name || name === "." || name === "..") return fallback;
  return name;
}
