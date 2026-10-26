/**
 * 拥有：偶像、圈子、好感、会话和快照的类型，以及从接口对象整理这些类型、分组、时间和错误文案。
 * 不拥有：界面和请求。这是服务端载荷的投影，不改服务端状态。
 */

/** Idol 一个偶像的身份、人设、声音、头像、圈子、工作和外观。 */
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

/** Circle 是一个圈子的编号和名字。 */
export type Circle = { id: string; name: string };
/** Affinity 是对某个偶像的分数、段位和冷却截止。 */
export type Affinity = { idol_id: string; score: number; tier: string; cooldown_until: string };
/** Lore 是挂在某个偶像上的一条设定。 */
export type Lore = { id: string; idol_id: string; title: string; body: string };
/** Segment 一轮回复里的口播和字幕。 */
export type Segment = { spoken?: string; subtitle?: string };
/** Rec 会话里的一条记录：用户原文、分段、工作会话、礼物、照片和分数变化。 */
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
/** Conversation 一段会话及其记录。current 表示当前段，ended 表示已结束。 */
export type Conversation = {
  id: string;
  idol_id: string;
  title: string;
  current: boolean;
  ended: boolean;
  created_at: string;
  records: Rec[];
};
/** History 把当前会话和过去的会话分开。 */
export type History = { current: Conversation | null; past: Conversation[] };
/** Snap 一次快照：偶像、圈子、好感、设定、钉住的人和头像，以及能否聊天、仅文字、电话音和能否工作。 */
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

/** Line 是画进对话里的一行，角色是用户、偶像或事件。 */
export type Line =
  | { id: string; role: "user"; text: string; at?: string }
  | { id: string; role: "idol"; text: string; at?: string; photo?: string; work?: string; delta?: number }
  | { id: string; role: "event"; text: string; gift?: string; delta?: number };

/** IdolErrors 是偶像错误码对应的界面文案。 */
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

/** blankIdol 一份空白偶像。颜色和语言有默认，工作默认开。 */
export function blankIdol(): Idol {
  return {
    id: "", name: "", intro: "", color: "#c47b8a", spoken_lang: "zh", subtitle_lang: "zh",
    sheet: "", facts: "", readings: "", voice_id: "", portrait: "", circle_id: "",
    work_enabled: true, work_profile_id: "", pinned: false, hidden: false, kind: "", outfit_id: "", liked_gifts: [],
  };
}

/** 缺值时用空串，否则收成字符串。v 为 null 时返回 fallback，否则交给 String；转成字符串，判断是否为空。fallback 原样返回。 */
function str(v: unknown, fallback = ""): string {
  if (v == null) return fallback;
  return String(v);
}

/** 收成有限数字，否则用后备值。v 转成数字后，作为分支结果，传给 Number.isFinite。fallback 作为分支结果。 */
function num(v: unknown, fallback = 0): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

/** 是数组就原样返回，否则空数组。v 作为分支结果，传给 Array.isArray。 */
function list(v: unknown): any[] {
  return Array.isArray(v) ? v : [];
}

/** 按给出的字段名依次取第一个有值的。v 判断是否为空，按下标读取，用 typeof 看类型。keys 用来遍历。 */
function pick(v: any, ...keys: string[]): unknown {
  if (v == null || typeof v !== "object") return undefined;
  for (const k of keys) {
    if (v[k] !== undefined && v[k] !== null) return v[k];
  }
  return undefined;
}

/** idolOf 从接口对象整理偶像。缺颜色用默认。工作开关只有明确 false 才关。v 用 pick 读取 id、name、intro、color、spoken_lang、subtitle_lang 等字段，为假时走另一支，用 typeof 看类型。 */
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

/** circleOf 把接口对象收成圈子。v 传给 pick。 */
export function circleOf(v: any): Circle {
  return { id: str(pick(v, "id", "ID")), name: str(pick(v, "name", "Name")) };
}

/** affinityOf 在列表里找该偶像的好感。没有则 50 分、trust、无冷却。listIn 传给 list。idolId 与 str(pick(a, "idol_id", "IdolID")) 比较是否相等，写入字段 idol_id。 */
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

/** 整理一条会话记录，分段收成口播和字幕。v 用 pick 读取 id、kind、at、user、segments、work_session 等字段。 */
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

/** conversationOf 把接口对象收成一段会话。v 用 pick 读取 id、idol_id、title、current、ended、created_at 等字段。 */
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

/** historyOf 把当前和过去的会话分开，丢掉没有编号的过去项。v 读取 current、Current、past、Past，为空时不读。 */
export function historyOf(v: any): History {
  const cur = v?.current ?? v?.Current;
  return {
    current: cur ? conversationOf(cur) : null,
    past: list(v?.past ?? v?.Past).map(conversationOf).filter((c) => c.id),
  };
}

/** loreOf 把接口对象收成一条设定。v 用 pick 读取 id、idol_id、title、body字段。 */
export function loreOf(v: any): Lore {
  return {
    id: str(pick(v, "id", "ID")),
    idol_id: str(pick(v, "idol_id", "IdolID")),
    title: str(pick(v, "title", "Title")),
    body: str(pick(v, "body", "Body")),
  };
}

/** snapOf 整理快照。钉住的人可以是对象，也可以是列表里 pinned 的那个。work 只有明确 false 才关。v 读取 idols、Idols、pinned、Pinned、circles、Circles 等，为空时不读。 */
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

/** linesOf 把记录收成对话行。事件或礼物单独成行。没有字幕、照片和工作会话的偶像轮次不出现。records 用来遍历。 */
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

/** groupIdols 按圈子分组可见偶像。钉住的排在组内和组间前面。没有圈子或圈子不存在的归到无名组。idols 做过滤。circles 做映射。 */
export function groupIdols(idols: Idol[], circles: Circle[]): { id: string; name: string; idols: Idol[] }[] {
  const visible = idols.filter((i) => !i.hidden);
  /** 置顶的排前面，其余按名字排。listIn 展开传递。 */
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

/** isCooling 冷却截止还在现在之后则为真。空或无法解析为假。iso 为假时返回假；传给 Date.parse。now 大于 t。 */
export function isCooling(iso: string | undefined, now = Date.now()): boolean {
  if (!iso) return false;
  const t = Date.parse(iso);
  return Number.isFinite(t) && t > now;
}

/** clock 今天只显示时分，更早的加上月日。无法解析则空。iso 为假时返回空字符串；传给 Date.parse。now 传给 Date。 */
export function clock(iso?: string, now = Date.now()): string {
  if (!iso) return "";
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return "";
  const d = new Date(t);
  const hm = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  if (new Date(now).toDateString() === d.toDateString()) return hm;
  return `${d.getMonth() + 1}/${d.getDate()} ${hm}`;
}

/** whenLabel 显示月日和时分。无法解析则空。iso 为假时返回空字符串；传给 Date.parse。now 函数体内没有读取。 */
export function whenLabel(iso: string | undefined, now = Date.now()): string {
  if (!iso) return "";
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return "";
  const d = new Date(t);
  const hm = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  return `${d.getMonth() + 1}/${d.getDate()} ${hm}`;
}

/** mmss 把毫秒收成 mm:ss。负的当 0。ms 除以 1000。 */
export function mmss(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000));
  return `${String(Math.floor(s / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
}

/** imageURL 把 base64 收成 data URL。按魔数判断 jpeg、png、gif、webp，其余当 png。已带逗号的只取逗号后。b64 看是否包含，截取，查找位置，作为分支结果。 */
export function imageURL(b64: string): string {
  const raw = b64.includes(",") ? b64.slice(b64.indexOf(",") + 1) : b64;
  const mime = raw.startsWith("/9j/") ? "image/jpeg"
    : raw.startsWith("iVBOR") ? "image/png"
    : raw.startsWith("R0lGOD") ? "image/gif"
    : raw.startsWith("UklGR") ? "image/webp"
    : "image/png";
  return `data:${mime};base64,${raw}`;
}

/** hexAlpha 把六位色写成带透明度的颜色，不合规则用后备色。hex 去掉首尾空白，做替换。alpha 拼进字符串。 */
export function hexAlpha(hex: string, alpha: number): string {
  const h = hex.trim().replace("#", "");
  if (!/^[0-9a-fA-F]{6}$/.test(h)) return `rgba(196,123,138,${alpha})`;
  const n = parseInt(h, 16);
  return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${alpha})`;
}

/** idolError 按错误原文选一句界面文案。对不上则去掉 /api/idol 前缀，仍空则用空回复那句。err 读取 message，为空时不读，作为后备值。copy 读取 cooldown、ended、noDraws、notHolding、noReset、nobodyLow 等。 */
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

/** achievementLabel 亲密和亲近成就套上名字，其余查表，没有则用 id。names 按下标读取。ach 按下标读取。closeTpl 做替换。intimateTpl 做替换。 */
export function achievementLabel(id: string, names: Record<string, string>, ach: Record<string, string>, closeTpl: string, intimateTpl: string): string {
  if (id.startsWith("intimate:")) return intimateTpl.replace("{name}", names[id.slice("intimate:".length)] || "");
  if (id.startsWith("close:")) return closeTpl.replace("{name}", names[id.slice("close:".length)] || "");
  return ach[id] || id;
}

/** tierText 把段位编号换成文案，没有就留编号。tiers 按下标读取。tier 用作下标，作为后备值。 */
export function tierText(tiers: Record<string, string>, tier: string): string {
  return tiers[tier] || tier;
}
