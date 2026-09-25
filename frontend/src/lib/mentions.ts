export type MentionKind = "file" | "folder" | "skill" | "harness";

export type MentionPin = {
  token: string;
  kind: MentionKind;
  label: string;
  detail: string;
};

const COMPLETE_RE = /@(?:file|folder|skill):\S+|@harness\b/g;

export function keepMentionOpen(token: string) {
  return token === "@file:" || token === "@folder:" || token === "@skill:";
}

export function mentionKind(token: string): MentionKind | null {
  if (token.startsWith("@file:")) return "file";
  if (token.startsWith("@folder:")) return "folder";
  if (token.startsWith("@skill:")) return "skill";
  if (token === "@harness" || token.startsWith("@harness")) return "harness";
  return null;
}

export function mentionLabel(token: string): string {
  const kind = mentionKind(token);
  if (kind === "harness") return "harness";
  const ref = token.replace(/^@(?:file|folder|skill):/, "");
  if (!ref) return kind || token;
  const parts = ref.replace(/\\/g, "/").split("/");
  return parts[parts.length - 1] || ref;
}

export function mentionDetail(token: string): string {
  if (token.startsWith("@file:") || token.startsWith("@folder:")) {
    return token.replace(/^@(?:file|folder):/, "");
  }
  if (token.startsWith("@skill:")) return token.slice("@skill:".length);
  return "";
}

export function pinFromToken(token: string): MentionPin {
  const kind = mentionKind(token) || "file";
  return { token, kind, label: mentionLabel(token), detail: mentionDetail(token) };
}

export function isCompleteMention(token: string): boolean {
  return /@(?:file|folder|skill):\S+/.test(token) || token === "@harness";
}

/** Token the caret is inside, including incomplete `@file:` prefixes. */
export function mentionTokenAt(value: string, caret: number): { start: number; end: number; token: string } | null {
  const pos = Math.max(0, Math.min(caret, value.length));
  const left = value.slice(0, pos);
  const m = left.match(/(?:^|\s)(@(?:file:|folder:|skill:)?[^\s]*)$/);
  if (!m) return null;
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
  const body = prose.trim();
  if (head && body) return `${head}\n${body}`;
  return head || body;
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
