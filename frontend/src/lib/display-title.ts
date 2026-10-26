/**
 * 拥有：把会话标题和工作区路径收成可以显示的短名。
 * 不拥有：标题的编辑，以及工作区是否存在。
 */

/** displayTitle 去掉空白。空标题，或只由点、间隔号、省略号、横线和斜杠组成时，用回退文案。title 为假时用空字符串，去掉首尾空白后，为假时走另一支，传给 test。fallback 原样返回。 */
export function displayTitle(title: string | undefined, fallback: string): string {
  const s = (title || "").trim();
  if (!s || /^[.·•…\-–—_/\\]+$/.test(s)) return fallback;
  return s;
}

/** displayWorkspace 取路径最后一段。空路径、`.` 或 `..` 时用回退文案。path 为假时返回 fallback；做替换。fallback 原样返回。 */
export function displayWorkspace(path: string | undefined, fallback = ""): string {
  if (!path) return fallback;
  const name = path.replace(/\\/g, "/").split("/").filter(Boolean).pop() || "";
  if (!name || name === "." || name === "..") return fallback;
  return name;
}
