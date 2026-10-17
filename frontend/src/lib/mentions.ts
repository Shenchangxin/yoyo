export type MentionKind = "file" | "folder" | "skill" | "harness" | "session";

export type MentionPin = {
  token: string;
  kind: MentionKind;
  label: string;
  detail: string;
};

export type MentionQuery = {
  kind: "any" | MentionKind;
  query: string;
  quoted: boolean;
};

export type MentionSessionRef = {
  id: string;
  title: string;
  workspace?: string;
  parentId?: string;
};

export type MentionFileHit = {
  path: string;
  kind: string;
};

const COMPLETE_RE = /@(?:file|folder|skill|session):\S+|@harness\b/g;

/** Active `@` token at the end of `before`, including `@"quoted` paths. */
export function activeAtTail(before: string): string | null {
  const quoted = /(?:^|\s)(@"([^"]*))$/u.exec(before);
  if (quoted?.[1]) return quoted[1];
  const plain = /(?:^|\s)(@[^\s]*)$/u.exec(before);
  return plain?.[1] ?? null;
}

export function keepMentionOpen(token: string) {
  return token === "@file:" || token === "@folder:" || token === "@skill:" || token === "@session:";
}

export function mentionKind(token: string): MentionKind | null {
  if (token.startsWith("@file:")) return "file";
  if (token.startsWith("@folder:")) return "folder";
  if (token.startsWith("@skill:")) return "skill";
  if (token.startsWith("@session:")) return "session";
  if (token === "@harness") return "harness";
  return null;
}

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

export function mentionDetail(token: string): string {
  if (token.startsWith("@file:") || token.startsWith("@folder:")) {
    return token.replace(/^@(?:file|folder):/, "");
  }
  if (token.startsWith("@skill:")) return token.slice("@skill:".length);
  if (token.startsWith("@session:")) return token.slice("@session:".length);
  return "";
}

export function pinFromToken(token: string, label?: string): MentionPin {
  const kind = mentionKind(token) || "file";
  return { token, kind, label: label || mentionLabel(token), detail: mentionDetail(token) };
}

export function isCompleteMention(token: string): boolean {
  return /@(?:file|folder|skill|session):\S+/.test(token) || token === "@harness";
}

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

/** Descent text left in the draft while browsing a folder (`@src/` or `@"src/`). */
export function formatTriggerPath(path: string, quoted: boolean, directory: boolean): string {
  const body = directory ? `${path.replace(/\/+$/, "")}/` : path;
  const needQuote = quoted || /\s/.test(body);
  if (!needQuote) return `@${body}`;
  if (directory) return `@"${body}`;
  return `@"${body}"`;
}

/** Immediate children of `dir/` from a fuzzy hit list. */
export function childPaths<T extends MentionFileHit>(hits: T[], dir: string): T[] {
  const prefix = `${dir.replace(/\/+$/, "")}/`;
  if (prefix === "/") return hits;
  return hits.filter((f) => {
    if (!f.path.startsWith(prefix)) return false;
    return !f.path.slice(prefix.length).includes("/");
  });
}

export type MentionCrumb = {
  label: string;
  query: string;
  current?: boolean;
};

/** Breadcrumb of a drilled directory listing (typed paths carry their own context). */
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

/** Token the caret is inside, including incomplete `@` / `@file:` prefixes. */
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
 * Turn leftover harness-style `@path` text into wire tokens the host expands.
 * Only exact hits from the current file list are rewritten; Go still Stats
 * unmatched paths on send.
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

export function isImageAttachment(a: AttachmentPin): boolean {
  const mime = (a.mime || "").toLowerCase();
  if (mime.startsWith("image/")) return true;
  const n = `${a.name || ""} ${a.path || ""}`.toLowerCase();
  return /\.(png|jpe?g|gif|webp)(?:$|\?)/i.test(n);
}

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

/** Extra suffix/prefix on a live user event is only @file/@folder/@skill tokens. */
export function extraIsMentions(s: string): boolean {
  const t = s.trim();
  if (!t) return true;
  return t.split(/\s+/).every((tok) => isCompleteMention(tok));
}

function mentionAugmented(base: string, fuller: string): boolean {
  if (!base) return extraIsMentions(fuller) && /@file:/.test(fuller);
  if (fuller.startsWith(base) && extraIsMentions(fuller.slice(base.length))) return true;
  if (fuller.startsWith(`${base}\n`) && extraIsMentions(fuller.slice(base.length))) return true;
  return false;
}

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

function skillHaystack(s: MentionSkill): string {
  return [s.name, s.displayName, s.slug, s.description].filter(Boolean).join(" ").toLowerCase();
}

function skillRank(s: MentionSkill): number {
  const src = s.source || "";
  if (src === "market" || src === "home" || src === "workspace") return 0;
  return 1;
}

/** Installed / workspace packs first. Bundled catalog stays below so @skill is usable. */
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

/** Cheap composer hint — mirrors host lexical match, not embeddings. */
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

/** Keep overlay highlight on the same token when the list refreshes. */
export function overlayActiveIndex(items: { token: string }[], token: string): number {
  if (!items.length) return 0;
  const i = token ? items.findIndex((x) => x.token === token) : -1;
  return i >= 0 ? i : 0;
}
