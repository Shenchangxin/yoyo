/**
 * 拥有：页面 TipTap 外壳用的一层 Markdown 与 HTML 对转。
 * 不拥有：编辑器本身和保存。不是第二个编辑器。本地转换，不改服务端。
 */

/** mdToHtml 把这一层支持的 Markdown 收成 HTML。空正文收成空段落。src 做替换。 */
export function mdToHtml(src: string): string {
  const text = src.replace(/\r\n/g, "\n");
  if (!text.trim()) return "<p></p>";
  const lines = text.split("\n");
  const out: string[] = [];
  let i = 0;
  let para: string[] = [];
  /** 把攒着的段落写成一段 HTML。 */
  const flush = () => {
    if (!para.length) return;
    out.push(`<p>${inline(para.join(" "))}</p>`);
    para = [];
  };
  while (i < lines.length) {
    const line = lines[i] ?? "";
    if (line.startsWith("```")) {
      flush();
      const lang = line.slice(3).trim();
      const buf: string[] = [];
      i += 1;
      while (i < lines.length && !lines[i]!.startsWith("```")) {
        buf.push(lines[i]!);
        i += 1;
      }
      out.push(`<pre><code class="language-${escapeHtml(lang)}">${escapeHtml(buf.join("\n"))}</code></pre>`);
      i += 1;
      continue;
    }
    const h = /^(#{1,3})\s+(.*)$/.exec(line);
    if (h) {
      flush();
      const n = h[1]!.length;
      out.push(`<h${n}>${inline(h[2] || "")}</h${n}>`);
      i += 1;
      continue;
    }
    if (/^[-*]\s+/.test(line)) {
      flush();
      const items: string[] = [];
      while (i < lines.length && /^[-*]\s+/.test(lines[i] || "")) {
        items.push(`<li>${inline((lines[i] || "").replace(/^[-*]\s+/, ""))}</li>`);
        i += 1;
      }
      out.push(`<ul>${items.join("")}</ul>`);
      continue;
    }
    if (/^\d+\.\s+/.test(line)) {
      flush();
      const items: string[] = [];
      while (i < lines.length && /^\d+\.\s+/.test(lines[i] || "")) {
        items.push(`<li>${inline((lines[i] || "").replace(/^\d+\.\s+/, ""))}</li>`);
        i += 1;
      }
      out.push(`<ol>${items.join("")}</ol>`);
      continue;
    }
    if (!line.trim()) {
      flush();
      i += 1;
      continue;
    }
    para.push(line);
    i += 1;
  }
  flush();
  return out.join("") || "<p></p>";
}

/** htmlToMd 把编辑器 HTML 收成这一层的 Markdown。html 做替换。 */
export function htmlToMd(html: string): string {
  let s = html
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/h1>/gi, "\n\n")
    .replace(/<h1[^>]*>/gi, "# ")
    .replace(/<\/h2>/gi, "\n\n")
    .replace(/<h2[^>]*>/gi, "## ")
    .replace(/<\/h3>/gi, "\n\n")
    .replace(/<h3[^>]*>/gi, "### ")
    .replace(/<\/p>/gi, "\n\n")
    .replace(/<p[^>]*>/gi, "")
    .replace(/<ol[^>]*>([\s\S]*?)<\/ol>/gi, (_m, inner) => numberedList(String(inner)))
    .replace(/<ul[^>]*>([\s\S]*?)<\/ul>/gi, (_m, inner) => bulletList(String(inner)))
    .replace(/<li[^>]*>/gi, "- ")
    .replace(/<\/li>/gi, "\n")
    .replace(/<pre><code[^>]*>([\s\S]*?)<\/code><\/pre>/gi, (_m, code) => `\n\`\`\`\n${decodeHtml(String(code))}\n\`\`\`\n`)
    .replace(/<strong[^>]*>([\s\S]*?)<\/strong>/gi, "**$1**")
    .replace(/<b[^>]*>([\s\S]*?)<\/b>/gi, "**$1**")
    .replace(/<em[^>]*>([\s\S]*?)<\/em>/gi, "*$1*")
    .replace(/<i[^>]*>([\s\S]*?)<\/i>/gi, "*$1*")
    .replace(/<code[^>]*>([\s\S]*?)<\/code>/gi, "`$1`")
    .replace(/<[^>]+>/g, "");
  s = decodeHtml(s);
  return s.replace(/\n{3,}/g, "\n\n").trim() + (s.trim() ? "\n" : "");
}

/** 把有序列表的项收成编号行。inner 做替换。 */
function numberedList(inner: string): string {
  let n = 0;
  const items = inner.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, (_m, li) => {
    n += 1;
    return `${n}. ${li}\n`;
  });
  return `\n${items}\n`;
}

/** 把无序列表的项收成横线行。inner 做替换。 */
function bulletList(inner: string): string {
  const items = inner.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, (_m, li) => `- ${li}\n`);
  return `\n${items}\n`;
}

/** 转义后再把粗体、斜体和行内代码换成标签。s 传给 escapeHtml。 */
function inline(s: string): string {
  return escapeHtml(s)
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\*([^*]+)\*/g, "<em>$1</em>")
    .replace(/`([^`]+)`/g, "<code>$1</code>");
}

/** 转义 HTML 里的括号和与号。s 做替换。 */
function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

/** 把转义过的括号、与号和空格还原。s 做替换。 */
function decodeHtml(s: string): string {
  return s.replace(/&lt;/g, "<").replace(/&gt;/g, ">").replace(/&amp;/g, "&").replace(/&nbsp;/g, " ");
}
