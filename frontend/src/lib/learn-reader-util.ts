/**
 * 拥有：阅读器用的几个纯函数：在渲染出来的文字里找回一段引文、取引文前后的一小段、把页码夹进范围、算阅读进度。
 * 不拥有：DOM 和任何调用。这里不引入 client，好在 node 里直接测。
 */

/** isSpace 在 c 是空白时为真。全角空格和不换行空格也算。 */
function isSpace(c: string): boolean {
  return c === " " || c === "\n" || c === "\t" || c === "\r" || c === "\u00a0" || c === "\u3000";
}

/** squash 去掉 s 里的空白，并记下留下的每个字符在 s 里的位置。 */
function squash(s: string): { text: string; at: number[] } {
  let text = "";
  const at: number[] = [];
  for (let i = 0; i < s.length; i++) {
    if (isSpace(s[i])) continue;
    text += s[i];
    at.push(i);
  }
  return { text, at };
}

/**
 * findQuote 在渲染出来的文字 haystack 里找引文 quote，返回它的起止位置（UTF-16 下标）。
 * 两边都先去掉空白再比，所以换行和空格不同也找得到。有几处时取离 near 最近的一处；near 是 0 到 1 之间的大致位置，不知道就不传。
 * 找不到时返回 null。
 */
export function findQuote(haystack: string, quote: string, near?: number): { start: number; end: number } | null {
  const hay = squash(haystack);
  const needle = squash(quote).text;
  if (!needle || needle.length > hay.text.length) return null;
  const found: number[] = [];
  for (let from = 0; from <= hay.text.length - needle.length; ) {
    const i = hay.text.indexOf(needle, from);
    if (i < 0) break;
    found.push(i);
    from = i + 1;
    if (found.length >= 50) break;
  }
  if (found.length === 0) return null;
  let best = found[0];
  if (typeof near === "number" && found.length > 1) {
    const want = Math.min(1, Math.max(0, near)) * hay.text.length;
    for (const i of found) {
      if (Math.abs(i - want) < Math.abs(best - want)) best = i;
    }
  }
  return { start: hay.at[best], end: hay.at[best + needle.length - 1] + 1 };
}

/** contextAround 取 text 里 [start, end) 这一段前后各最多 n 个字符，交给引擎当定位的前后文。 */
export function contextAround(text: string, start: number, end: number, n = 32): { prefix: string; suffix: string } {
  return { prefix: text.slice(Math.max(0, start - n), start), suffix: text.slice(end, end + n) };
}

/** clampLocator 把定位号夹进 1 到 pages 之间。不是数时按 1 算。 */
export function clampLocator(n: number, pages: number): number {
  if (!Number.isFinite(n)) return 1;
  return Math.min(Math.max(1, Math.floor(n)), Math.max(1, pages));
}

/** readPercent 是读到第 locator 个单元时的整体进度，0 到 1。只有一个单元时读到就算读完。 */
export function readPercent(locator: number, pages: number): number {
  if (pages <= 1) return 1;
  return Math.min(1, Math.max(0, clampLocator(locator, pages) / pages));
}

/** fileSize 把字节数写成 KB 或 MB。 */
export function fileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "";
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
