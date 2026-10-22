export type Idol = {
  id: string;
  name: string;
  intro: string;
  color: string;
  spoken_lang: string;
  subtitle_lang: string;
  sheet: string;
  facts: string;
  readings: string;
  voice_id: string;
  portrait: string;
  circle_id: string;
  work_enabled: boolean;
  work_profile_id: string;
  pinned: boolean;
  hidden: boolean;
  kind: string;
  outfit_id: string;
  liked_gifts: string[];
};

export type Circle = { id: string; name: string };
export type Affinity = { idol_id: string; score: number; tier: string; cooldown_until: string };
export type Lore = { id: string; idol_id: string; title: string; body: string };
export type Segment = { spoken?: string; subtitle?: string };
export type Rec = {
  id: string;
  kind: string;
  at?: string;
  user?: string;
  segments?: Segment[];
  work_session?: string;
  gift_id?: string;
  photo_id?: string;
  delta?: number;
};
export type Conversation = {
  id: string;
  idol_id: string;
  title: string;
  current: boolean;
  ended: boolean;
  created_at: string;
  records: Rec[];
};
export type History = { current: Conversation | null; past: Conversation[] };
export type Snap = {
  idols: Idol[];
  circles: Circle[];
  affinities: Affinity[];
  lore: Lore[];
  pinnedId: string;
  portrait: string;
  chatReady: boolean;
  textOnly: boolean;
  phone: boolean;
  work: boolean;
};

export type Line =
  | { id: string; role: "user"; text: string; at?: string }
  | { id: string; role: "idol"; text: string; at?: string; photo?: string; work?: string; delta?: number }
  | { id: string; role: "event"; text: string; gift?: string; delta?: number };

export type IdolErrors = {
  cooldown: string;
  ended: string;
  needModel: string;
  noDraws: string;
  noReset: string;
  notHolding: string;
  nobodyLow: string;
  paused: string;
  emptyReply: string;
  missing: string;
};

export function blankIdol(): Idol {
  return {
    id: "", name: "", intro: "", color: "#c47b8a", spoken_lang: "zh", subtitle_lang: "zh",
    sheet: "", facts: "", readings: "", voice_id: "", portrait: "", circle_id: "",
    work_enabled: true, work_profile_id: "", pinned: false, hidden: false, kind: "", outfit_id: "", liked_gifts: [],
  };
}

function str(v: unknown, fallback = ""): string {
  if (v == null) return fallback;
  return String(v);
}

function num(v: unknown, fallback = 0): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

function list(v: unknown): any[] {
  return Array.isArray(v) ? v : [];
}

function pick(v: any, ...keys: string[]): unknown {
  if (v == null || typeof v !== "object") return undefined;
  for (const k of keys) {
    if (v[k] !== undefined && v[k] !== null) return v[k];
  }
  return undefined;
}

export function idolOf(v: any): Idol {
  const base = blankIdol();
  if (!v || typeof v !== "object") return base;
  return {
    ...base,
    id: str(pick(v, "id", "ID")),
    name: str(pick(v, "name", "Name")),
    intro: str(pick(v, "intro", "Intro")),
    color: str(pick(v, "color", "Color")) || base.color,
    spoken_lang: str(pick(v, "spoken_lang", "SpokenLang")) || "zh",
    subtitle_lang: str(pick(v, "subtitle_lang", "SubtitleLang")) || "zh",
    sheet: str(pick(v, "sheet", "Sheet")),
    facts: str(pick(v, "facts", "Facts")),
    readings: str(pick(v, "readings", "Readings")),
    voice_id: str(pick(v, "voice_id", "VoiceID")),
    portrait: str(pick(v, "portrait", "Portrait")),
    circle_id: str(pick(v, "circle_id", "CircleID")),
    work_enabled: pick(v, "work_enabled", "WorkEnabled") !== false,
    work_profile_id: str(pick(v, "work_profile_id", "WorkProfileID")),
    pinned: !!pick(v, "pinned", "Pinned"),
    hidden: !!pick(v, "hidden", "Hidden"),
    kind: str(pick(v, "kind", "Kind")),
    outfit_id: str(pick(v, "outfit_id", "OutfitID")),
    liked_gifts: list(pick(v, "liked_gifts", "LikedGifts")).map((x) => str(x)),
  };
}

export function circleOf(v: any): Circle {
  return { id: str(pick(v, "id", "ID")), name: str(pick(v, "name", "Name")) };
}

export function affinityOf(listIn: unknown, idolId: string): Affinity {
  const raw = list(listIn).find((a) => str(pick(a, "idol_id", "IdolID")) === idolId);
  if (!raw) return { idol_id: idolId, score: 50, tier: "trust", cooldown_until: "" };
  return {
    idol_id: idolId,
    score: num(pick(raw, "score", "Score"), 50),
    tier: str(pick(raw, "tier", "Tier")) || "trust",
    cooldown_until: str(pick(raw, "cooldown_until", "CooldownUntil")),
  };
}

function recOf(v: any): Rec {
  return {
    id: str(pick(v, "id", "ID")),
    kind: str(pick(v, "kind", "Kind")),
    at: str(pick(v, "at", "At")),
    user: str(pick(v, "user", "User")),
    segments: list(pick(v, "segments", "Segments")).map((s) => ({
      spoken: str(pick(s, "spoken", "Spoken")),
      subtitle: str(pick(s, "subtitle", "Subtitle")),
    })),
    work_session: str(pick(v, "work_session", "WorkSession")),
    gift_id: str(pick(v, "gift_id", "GiftID")),
    photo_id: str(pick(v, "photo_id", "PhotoID")),
    delta: num(pick(v, "delta", "Delta"), 0),
  };
}

export function conversationOf(v: any): Conversation {
  return {
    id: str(pick(v, "id", "ID")),
    idol_id: str(pick(v, "idol_id", "IdolID")),
    title: str(pick(v, "title", "Title")),
    current: !!pick(v, "current", "Current"),
    ended: !!pick(v, "ended", "Ended"),
    created_at: str(pick(v, "created_at", "CreatedAt")),
    records: list(pick(v, "records", "Records")).map(recOf),
  };
}

export function historyOf(v: any): History {
  const cur = v?.current ?? v?.Current;
  return {
    current: cur ? conversationOf(cur) : null,
    past: list(v?.past ?? v?.Past).map(conversationOf).filter((c) => c.id),
  };
}

export function loreOf(v: any): Lore {
  return {
    id: str(pick(v, "id", "ID")),
    idol_id: str(pick(v, "idol_id", "IdolID")),
    title: str(pick(v, "title", "Title")),
    body: str(pick(v, "body", "Body")),
  };
}

export function snapOf(v: any): Snap {
  const idols = list(v?.idols ?? v?.Idols).map(idolOf);
  const pinned = v?.pinned ?? v?.Pinned;
  const pinnedIdol = pinned && typeof pinned === "object" ? idolOf(pinned) : idols.find((i) => i.pinned);
  return {
    idols,
    circles: list(v?.circles ?? v?.Circles).map(circleOf).filter((c) => c.id),
    affinities: list(v?.affinities ?? v?.Affinities).map((a) => affinityOf([a], str(pick(a, "idol_id", "IdolID")))),
    lore: list(v?.lore ?? v?.Lore).map(loreOf).filter((d) => d.id),
    pinnedId: pinnedIdol?.id || "",
    portrait: str(v?.portrait ?? v?.Portrait),
    chatReady: !!(v?.chat_ready ?? v?.ChatReady),
    textOnly: !!(v?.text_only ?? v?.TextOnly),
    phone: !!(v?.phone ?? v?.Phone),
    work: (v?.work ?? v?.Work) !== false,
  };
}

export function linesOf(records: Rec[]): Line[] {
  const out: Line[] = [];
  for (const r of records) {
    const text = (r.segments || []).map((s) => (s.subtitle || s.spoken || "").trim()).filter(Boolean).join("\n");
    if (r.user) out.push({ id: `${r.id}:u`, role: "user", text: r.user, at: r.at });
    if (r.kind === "event" || r.gift_id) {
      if (text || r.gift_id) out.push({ id: r.id, role: "event", text: text || r.gift_id || "", gift: r.gift_id, delta: r.delta });
      continue;
    }
    if (text || r.photo_id || r.work_session) {
      out.push({
        id: r.id,
        role: "idol",
        text,
        at: r.at,
        photo: r.photo_id || undefined,
        work: r.work_session || undefined,
        delta: r.delta || undefined,
      });
    }
  }
  return out;
}

export function groupIdols(idols: Idol[], circles: Circle[]): { id: string; name: string; idols: Idol[] }[] {
  const visible = idols.filter((i) => !i.hidden);
  const order = (listIn: Idol[]) => [...listIn].sort((a, b) => Number(b.pinned) - Number(a.pinned) || a.name.localeCompare(b.name, "zh"));
  const known = new Set(circles.map((c) => c.id));
  const groups = circles
    .map((c) => ({ id: c.id, name: c.name, idols: order(visible.filter((i) => i.circle_id === c.id)) }))
    .filter((g) => g.idols.length > 0);
  const loose = order(visible.filter((i) => !i.circle_id || !known.has(i.circle_id)));
  if (loose.length) groups.push({ id: "", name: "", idols: loose });
  if (!groups.length && visible.length) groups.push({ id: "", name: "", idols: order(visible) });
  const pinnedAt = groups.findIndex((g) => g.idols.some((i) => i.pinned));
  if (pinnedAt > 0) {
    const [pinned] = groups.splice(pinnedAt, 1);
    groups.unshift(pinned);
  }
  return groups;
}

export function isCooling(iso: string | undefined, now = Date.now()): boolean {
  if (!iso) return false;
  const t = Date.parse(iso);
  return Number.isFinite(t) && t > now;
}

export function clock(iso?: string, now = Date.now()): string {
  if (!iso) return "";
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return "";
  const d = new Date(t);
  const hm = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  if (new Date(now).toDateString() === d.toDateString()) return hm;
  return `${d.getMonth() + 1}/${d.getDate()} ${hm}`;
}

export function whenLabel(iso: string | undefined, now = Date.now()): string {
  if (!iso) return "";
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return "";
  const d = new Date(t);
  const hm = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  return `${d.getMonth() + 1}/${d.getDate()} ${hm}`;
}

export function mmss(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000));
  return `${String(Math.floor(s / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
}

export function imageURL(b64: string): string {
  const raw = b64.includes(",") ? b64.slice(b64.indexOf(",") + 1) : b64;
  const mime = raw.startsWith("/9j/") ? "image/jpeg"
    : raw.startsWith("iVBOR") ? "image/png"
    : raw.startsWith("R0lGOD") ? "image/gif"
    : raw.startsWith("UklGR") ? "image/webp"
    : "image/png";
  return `data:${mime};base64,${raw}`;
}

export function hexAlpha(hex: string, alpha: number): string {
  const h = hex.trim().replace("#", "");
  if (!/^[0-9a-fA-F]{6}$/.test(h)) return `rgba(196,123,138,${alpha})`;
  const n = parseInt(h, 16);
  return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${alpha})`;
}

export function idolError(err: unknown, copy: IdolErrors): string {
  const msg = String((err as { message?: string })?.message || err || "").trim();
  const low = msg.toLowerCase();
  if (low.includes("cooldown")) return copy.cooldown;
  if (low.includes("ended") || msg.includes("这段对话")) return copy.ended;
  if (low.includes("no draws")) return copy.noDraws;
  if (low.includes("not holding")) return copy.notHolding;
  if (low.includes("no reset")) return copy.noReset;
  if (low.includes("nobody is below")) return copy.nobodyLow;
  if (low.includes("chat model") || low.includes("chat key") || low.includes("chat connection") || low.includes("no key")) return copy.needModel;
  if (low.includes("paused")) return copy.paused;
  if (low.includes("empty reply") || low === "empty" || low.endsWith(": empty")) return copy.emptyReply;
  if (low.includes("idol not found")) return copy.missing;
  return msg.replace(/^\/api\/idol:\s*/i, "") || copy.emptyReply;
}

export function achievementLabel(id: string, names: Record<string, string>, ach: Record<string, string>, closeTpl: string, intimateTpl: string): string {
  if (id.startsWith("intimate:")) return intimateTpl.replace("{name}", names[id.slice("intimate:".length)] || "");
  if (id.startsWith("close:")) return closeTpl.replace("{name}", names[id.slice("close:".length)] || "");
  return ach[id] || id;
}

export function tierText(tiers: Record<string, string>, tier: string): string {
  return tiers[tier] || tier;
}
