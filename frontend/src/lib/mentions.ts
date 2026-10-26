/**
 * 拥有：从光标处识别 @ 提及，把草稿收成标记，并按词法提示技能。
 * 不拥有：提及菜单的渲染，以及宿主如何展开标记。
 */

/** MentionKind 文件、文件夹、技能、harness 或会话。 */
export type MentionKind = "file" | "folder" | "skill" | "harness" | "session";

/** MentionPin 一枚提及。token 写进草稿，detail 是路径或 id。 */
export type MentionPin = {
  token: string;
  kind: MentionKind;
  label: string;
  detail: string;
};

/** MentionQuery 光标处的查询。kind 为 any 时还没选定种类。quoted 表示用了引号。 */
export type MentionQuery = {
  kind: "any" | MentionKind;
  query: string;
  quoted: boolean;
};

/** MentionSessionRef 可提及的一段会话。 */
export type MentionSessionRef = {
  id: string;
  title: string;
  workspace?: string;
  parentId?: string;
};

/** MentionFileHit 文件搜索的一条命中。kind 区分文件和目录。 */
export type MentionFileHit = {
  path: string;
  kind: string;
};

const COMPLETE_RE = /@(?:file|folder|skill|session):\S+|@harness\b/g;

/** activeAtTail `before` 末尾正在输入的 `@`，含未闭合的 `@"。行首或空白之后才算。没有则空。 */
export function activeAtTail(before: string): string | null {
  const quoted = /(?:^|\s)(@"([^"]*))$/u.exec(before);
  if (quoted?.[1]) return quoted[1];
  const plain = /(?:^|\s)(@[^\s]*)$/u.exec(before);
  return plain?.[1] ?? null;
}

/** keepMentionOpen 正好是 `@file:`、`@folder:`、`@skill:` 或 `@session:` 时菜单保持打开。token 是 @file:、@folder:、@skill: 或 @session: 时返回真。 */
export function keepMentionOpen(token: string) {
  return token === "@file:" || token === "@folder:" || token === "@skill:" || token === "@session:";
}

/** mentionKind 从标记前缀认出种类。只有整词 `@harness` 才是 harness，其余返回空。token 为 @harness 时返回 harness；看开头。 */
export function mentionKind(token: string): MentionKind | null {
  if (token.startsWith("@file:")) return "file";
  if (token.startsWith("@folder:")) return "folder";
  if (token.startsWith("@skill:")) return "skill";
  if (token.startsWith("@session:")) return "session";
  if (token === "@harness") return "harness";
  return null;
}

/** mentionLabel harness 显示 harness。会话显示冒号后的原文。文件和技能显示最后一段，文件夹末尾加斜杠。没有引用时退回种类或标记本身。token 做替换，作为后备值，传给 mentionKind。 */
export function mentionLabel(token: string): string {
  const kind = mentionKind(token);
  if (kind === "harness") return "harness";
  const ref = token.replace(/^@(?:file|folder|skill|session):/, "");
  if (!ref) return kind || token;
  if (kind === "session") return ref;
  const parts = ref.replace(/\\/g, "/").split("/");
  const name = parts[parts.length - 1] || ref;
  return kind === "folder" ? `${name}/` : name;
}

/** mentionDetail 文件和文件夹是路径，技能和会话是冒号后的部分，其余空串。token 看开头，做替换，截取。 */
export function mentionDetail(token: string): string {
  if (token.startsWith("@file:") || token.startsWith("@folder:")) {
    return token.replace(/^@(?:file|folder):/, "");
  }
  if (token.startsWith("@skill:")) return token.slice("@skill:".length);
  if (token.startsWith("@session:")) return token.slice("@session:".length);
  return "";
}

/** pinFromToken 由标记做成一枚。认不出种类时当文件。token 传给 mentionKind，传给 mentionLabel，传给 mentionDetail，写入字段 token。label 为假时改用后备值，写入字段 label。 */
export function pinFromToken(token: string, label?: string): MentionPin {
  const kind = mentionKind(token) || "file";
  return { token, kind, label: label || mentionLabel(token), detail: mentionDetail(token) };
}

/** isCompleteMention 带非空白引用的 file、folder、skill、session，或正好是 `@harness`。token 与 @harness 比较是否相等，传给 test。 */
export function isCompleteMention(token: string): boolean {
  return /@(?:file|folder|skill|session):\S+/.test(token) || token === "@harness";
}

/** mentionQueryOf 按前缀拆出种类和查询。`@harness` 的查询是 harness。`@"` 开头为引号查询并去掉结尾引号。其余去掉一个 `@` 后当作 any。token 与 @harness 比较是否相等，看开头，截取，作为分支结果。 */
export function mentionQueryOf(token: string): MentionQuery {
  if (token.startsWith("@file:")) return { kind: "file", query: token.slice("@file:".length), quoted: false };
  if (token.startsWith("@folder:")) return { kind: "folder", query: token.slice("@folder:".length), quoted: false };
  if (token.startsWith("@skill:")) return { kind: "skill", query: token.slice("@skill:".length), quoted: false };
  if (token.startsWith("@session:")) return { kind: "session", query: token.slice("@session:".length), quoted: false };
  if (token === "@harness") {
    return { kind: "harness", query: "harness", quoted: false };
  }
  if (token.startsWith("@\"")) {
    return { kind: "any", query: token.slice(2).replace(/"$/, ""), quoted: true };
  }
  return { kind: "any", query: token.startsWith("@") ? token.slice(1) : token, quoted: false };
}

/**
 * formatTriggerPath 浏览文件夹时留在草稿里的文本，形如 `@src/` 或 `@"src/`。目录去掉多余结尾斜杠后再加一个。含空白或已要求引号时加引号，目录不加闭合引号。
 * path 做替换，作为分支结果。
 * quoted 为假时改用后备值后，为假时走另一支。
 * directory 为真时才继续。
 */
export function formatTriggerPath(path: string, quoted: boolean, directory: boolean): string {
  const body = directory ? `${path.replace(/\/+$/, "")}/` : path;
  const needQuote = quoted || /\s/.test(body);
  if (!needQuote) return `@${body}`;
  if (directory) return `@"${body}`;
  return `@"${body}"`;
}

/** childPaths 模糊命中里该目录的直接子项。目录前缀为 `/` 时原样返回。路径里还含斜杠的更深项排除。hits 做过滤，有的分支原样返回。dir 做替换。 */
export function childPaths<T extends MentionFileHit>(hits: T[], dir: string): T[] {
  const prefix = `${dir.replace(/\/+$/, "")}/`;
  if (prefix === "/") return hits;
  return hits.filter((f) => {
    if (!f.path.startsWith(prefix)) return false;
    return !f.path.slice(prefix.length).includes("/");
  });
}

/** MentionCrumb 一层面包屑。current 表示当前这一层。 */
export type MentionCrumb = {
  label: string;
  query: string;
  current?: boolean;
};

/**
 * mentionCrumbs 钻进目录后的面包屑。没钻进去，或查询里没有斜杠，则没有。第一层用根标签，查询为 `@` 或 `@"`。
 * query 调用 lastIndexOf，截取。
 * quoted 为真时才继续，传给 formatTriggerPath。
 * drilled 为假时返回 undefined。
 * rootLabel 写入字段 label。
 */
export function mentionCrumbs(query: string, quoted: boolean, drilled: boolean, rootLabel: string): MentionCrumb[] | undefined {
  if (!drilled) return undefined;
  const slash = query.lastIndexOf("/");
  if (slash < 0) return undefined;
  const segments = query.slice(0, slash).split("/").filter((s) => s !== "");
  const crumbs: MentionCrumb[] = [{
    label: rootLabel,
    query: quoted ? "@\"" : "@",
  }];
  for (let i = 0; i < segments.length; i++) {
    const path = segments.slice(0, i + 1).join("/");
    crumbs.push({
      label: segments[i],
      query: formatTriggerPath(path, quoted, true),
      current: i === segments.length - 1,
    });
  }
  return crumbs;
}

/** mentionTokenAt 光标所在的标记，含未完成的 `@` 和 `@file:`。引号形式只取光标左侧。普通形式把右侧同一段非空白也算进去。没有则空。value 读取 length，截取。caret 传给 Math.min。 */
export function mentionTokenAt(value: string, caret: number): { start: number; end: number; token: string } | null {
  const pos = Math.max(0, Math.min(caret, value.length));
  const left = value.slice(0, pos);
  const quoted = /(?:^|\s)(@"([^"]*))$/u.exec(left);
  if (quoted?.[1]) {
    return { start: pos - quoted[1].length, end: pos, token: quoted[1] };
  }
  const m = /(?:^|\s)(@[^\s]*)$/u.exec(left);
  if (!m?.[1]) return null;
  const head = m[1];
  const extra = value.slice(pos).match(/^([^\s]*)/)?.[1] || "";
  return { start: pos - head.length, end: pos + extra.length, token: head + extra };
}

/** peelCompletedMentions 把已完成的标记拆成芯片，正文里去掉它们，并按去掉的长度回退光标。连续的非换行空白收成一个空格。text 截取，传给 re.exec。caret 大于等于 end，大于 start。 */
export function peelCompletedMentions(text: string, caret = text.length): { text: string; chips: MentionPin[]; caret: number } {
  const chips: MentionPin[] = [];
  let out = "";
  let last = 0;
  let caretNext = caret;
  const re = new RegExp(COMPLETE_RE.source, "g");
  let m: RegExpExecArray | null;
  while ((m = re.exec(text))) {
    const start = m.index;
    const raw = m[0];
    const end = start + raw.length;
    chips.push(pinFromToken(raw));
    out += text.slice(last, start);
    if (caret >= end) caretNext -= raw.length;
    else if (caret > start) caretNext = out.length;
    last = end;
  }
  out += text.slice(last);
  const collapsed = out.replace(/[^\S\n]{2,}/g, " ");
  if (collapsed.length < out.length && caretNext > 0) {
    caretNext = Math.max(0, caretNext - (out.length - collapsed.length));
  }
  if (caretNext > collapsed.length) caretNext = collapsed.length;
  return { text: collapsed, chips, caret: caretNext };
}

/** composeDraft 标记在前，正文在后。会话钉的标签收成一行 sessions，给模型看标题。pins 做映射，做过滤。prose 去掉首尾空白。 */
export function composeDraft(pins: MentionPin[], prose: string): string {
  const head = pins.map((p) => p.token).join(" ");
  const sessionNotes = pins
    .filter((p) => p.kind === "session")
    .map((p) => p.label || p.detail)
    .filter(Boolean);
  const note = sessionNotes.length ? `sessions: ${sessionNotes.join("; ")}` : "";
  const body = [note, prose.trim()].filter(Boolean).join("\n");
  if (head && body) return `${head}\n${body}`;
  return head || body;
}

/**
 * settleBareAtPaths 把剩下的 `@path` 收成宿主会展开的标记。只改写当前文件列表里的精确命中。对不上的路径在发送时仍由 Go 统计。目录或查询以斜杠结尾写成 `@folder:`。已经带种类的标记不动。
 * text 做替换，为假时走另一支，有的分支原样返回。
 * files 读取 length，做映射。
 */
export function settleBareAtPaths(text: string, files: MentionFileHit[]): string {
  if (!text || !files.length) return text;
  const byPath = new Map(files.map((f) => [f.path.replace(/\\/g, "/"), f]));
  return text.replace(/(^|\s)(@(?:"[^"]*|[^\s]+))/g, (all, pre: string, token: string) => {
    const parsed = mentionQueryOf(token);
    if (parsed.kind !== "any") return all;
    const path = parsed.query.replace(/\\/g, "/").replace(/\/+$/, "");
    if (!path) return all;
    const hit = byPath.get(path);
    if (!hit) return all;
    const folder = hit.kind === "dir" || parsed.query.endsWith("/");
    return `${pre}${folder ? "@folder:" : "@file:"}${hit.path}`;
  });
}

type AttachmentPin = { name?: string; path?: string; mime?: string };

/** isImageAttachment mime 以 image/ 开头，或名字、路径以 png、jpg、jpeg、gif、webp 结尾。a 读取 mime、name、path。 */
export function isImageAttachment(a: AttachmentPin): boolean {
  const mime = (a.mime || "").toLowerCase();
  if (mime.startsWith("image/")) return true;
  const n = `${a.name || ""} ${a.path || ""}`.toLowerCase();
  return /\.(png|jpe?g|gif|webp)(?:$|\?)/i.test(n);
}

/** pinsFromAttachments 非图片附件收成文件钉。标签用文件名。没有路径则用名字。同一标记只留一次。atts 读取 length，为空时不读，用来遍历。 */
export function pinsFromAttachments(atts: AttachmentPin[] | undefined | null): MentionPin[] {
  if (!atts?.length) return [];
  const out: MentionPin[] = [];
  const seen = new Set<string>();
  for (const a of atts) {
    if (isImageAttachment(a)) continue;
    const path = (a.path || "").replace(/\\/g, "/");
    const name = (a.name || path.split("/").pop() || "file").replace(/^.*[/\\]/, "") || "file";
    const token = `@file:${path || name}`;
    if (seen.has(token)) continue;
    seen.add(token);
    out.push({ token, kind: "file", label: name, detail: path || name });
  }
  return out;
}

/** mergeMentionPins 按 token 合并。已经在 a 里的不再从 b 加入。 */
export function mergeMentionPins(a: MentionPin[], b: MentionPin[]): MentionPin[] {
  const seen = new Set(a.map((p) => p.token));
  const out = a.slice();
  for (const p of b) {
    if (seen.has(p.token)) continue;
    seen.add(p.token);
    out.push(p);
  }
  return out;
}

/** extraIsMentions 多出来的前缀或后缀去掉空白后，每一段都是完整提及。空白本身算是。s 去掉首尾空白后，切开，为假时走另一支。 */
export function extraIsMentions(s: string): boolean {
  const t = s.trim();
  if (!t) return true;
  return t.split(/\s+/).every((tok) => isCompleteMention(tok));
}

/** fuller 以 base 开头，多出来的部分全是提及。base 为空时，fuller 里还要有 `@file:`。 */
function mentionAugmented(base: string, fuller: string): boolean {
  if (!base) return extraIsMentions(fuller) && /@file:/.test(fuller);
  if (fuller.startsWith(base) && extraIsMentions(fuller.slice(base.length))) return true;
  if (fuller.startsWith(`${base}\n`) && extraIsMentions(fuller.slice(base.length))) return true;
  return false;
}

/** sameUserTurnText 两端去空白后相等，或一边只是在另一边外面多了提及。a 去掉首尾空白后，与 right 比较是否相等，传给 mentionAugmented。b 去掉首尾空白后，与 left 比较是否相等，传给 mentionAugmented。 */
export function sameUserTurnText(a: string, b: string): boolean {
  const left = a.trim();
  const right = b.trim();
  if (left === right) return true;
  return mentionAugmented(left, right) || mentionAugmented(right, left);
}

type MentionSkill = {
  name: string;
  description?: string;
  displayName?: string;
  slug?: string;
  source?: string;
};

/** 名字、显示名、slug 和说明拼成小写，供匹配。s 读取 name、displayName、slug、description。 */
function skillHaystack(s: MentionSkill): string {
  return [s.name, s.displayName, s.slug, s.description].filter(Boolean).join(" ").toLowerCase();
}

/** market、home、workspace 排在前面，其余靠后。s 读取 source。 */
function skillRank(s: MentionSkill): number {
  const src = s.source || "";
  if (src === "market" || src === "home" || src === "workspace") return 0;
  return 1;
}

/** mentionableSkills 已安装和工作区包在前，捆绑目录留在下面，这样 @skill 还能用。查询不区分大小写，空查询留全部。先按来源再按名字排序。skills 做过滤。query 去掉首尾空白，转成小写后，为假时走另一支，传给 includes。 */
export function mentionableSkills<T extends MentionSkill>(skills: T[], query: string): T[] {
  const q = query.trim().toLowerCase();
  return skills
    .filter((s) => !q || skillHaystack(s).includes(q))
    .slice()
    .sort((a, b) => skillRank(a) - skillRank(b) || a.name.localeCompare(b.name));
}

const ROUTE_LEX: [string, string][] = [
  ["novel", "小说"],
  ["book", "小说"],
  ["game", "游戏"],
  ["game-world", "游戏世界"],
  ["weekly", "周报"],
  ["report", "周报"],
  ["minutes", "纪要"],
  ["meeting", "会议"],
  ["drama", "短剧"],
  ["script", "剧本"],
];

/** 切出长度至少为 2 的片段。英文词若在词表里，连同对应中文一起放进集合。text 转成小写，看是否包含。 */
function suggestTokens(text: string): Set<string> {
  const out = new Set<string>();
  const fold = text.toLowerCase();
  for (const part of fold.split(/[^a-z0-9\u4e00-\u9fff]+/)) {
    if (part.length >= 2) out.add(part);
  }
  for (const [en, zh] of ROUTE_LEX) {
    if (fold.includes(en) || text.includes(zh)) {
      out.add(en);
      out.add(zh);
    }
  }
  return out;
}

/**
 * suggestSkills 作曲提示，对齐宿主的词法匹配，不是嵌入。正文去掉空白后短于 2 则没有。名字或 slug 被包含加 80。词长度至少 3 的命中加 12，长度为 2 的加 6。分数低于 24 丢掉。同名只留一次，最多 limit 条。
 * skills 做映射。
 * voice 去掉首尾空白后，转成小写，读取 length，传给 suggestTokens。
 */
export function suggestSkills<T extends MentionSkill>(skills: T[], voice: string, limit = 3): T[] {
  const q = voice.trim();
  if (q.length < 2) return [];
  const toks = suggestTokens(q);
  const fold = q.toLowerCase();
  const ranked = skills
    .map((s) => {
      const hay = skillHaystack(s);
      let score = 0;
      if (fold.includes(s.name.toLowerCase()) || (s.slug && fold.includes(s.slug.toLowerCase()))) score += 80;
      for (const tok of toks) {
        if (tok.length >= 2 && hay.includes(tok)) score += tok.length >= 3 ? 12 : 6;
      }
      return { s, score };
    })
    .filter((x) => x.score >= 24)
    .sort((a, b) => b.score - a.score || skillRank(a.s) - skillRank(b.s) || a.s.name.localeCompare(b.s.name));
  const seen = new Set<string>();
  const out: T[] = [];
  for (const row of ranked) {
    if (seen.has(row.s.name)) continue;
    seen.add(row.s.name);
    out.push(row.s);
    if (out.length >= limit) break;
  }
  return out;
}

/** overlayActiveIndex 列表刷新后高亮留在同一个 token。找不到或列表为空时为 0。items 读取 length，调用 findIndex。 */
export function overlayActiveIndex(items: { token: string }[], token: string): number {
  if (!items.length) return 0;
  const i = token ? items.findIndex((x) => x.token === token) : -1;
  return i >= 0 ? i : 0;
}
