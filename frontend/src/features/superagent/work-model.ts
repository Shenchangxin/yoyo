/**
 * 拥有：工作栏怎么从工单里分出「现在」和「任务」，以及晨间简报怎么从资产里审计出来。
 * 不拥有：拉取快照和画页面。那一段在 WorkPane。简报不写进对话。
 */

import type { Decision, Rule, Ticket, Trigger } from "./desk-model";

export type StageKey =
  | "queued"
  | "running"
  | "waiting"
  | "ended"
  | "blocked"
  | "failed"
  | "answer"
  | "reviewWait"
  | "delivery"
  | "idle";

/** stageKey 把工单阶段收成一句状态。认不出的进行中算 waiting。 */
export function stageKey(ticket: Pick<Ticket, "stage" | "action">): StageKey {
  if (ticket.stage === "ended") return "ended";
  switch (ticket.action) {
    case "delivery":
      return "delivery";
    case "answer":
      return "answer";
    case "review":
      return "reviewWait";
    case "blocked":
      return "blocked";
    case "failed":
      return "failed";
    case "queued":
      return "queued";
    default:
      break;
  }
  if (ticket.stage === "working") return "running";
  if (ticket.stage === "idle") return "idle";
  return "waiting";
}

/** needsPerson 在这张工单没有人就继续不了时为真。已结束的不算。 */
export function needsPerson(ticket: Pick<Ticket, "person" | "stage">): boolean {
  return !!ticket.person && ticket.stage !== "ended";
}

/** nowTickets 是正在做的，以及正在等人的。已结束的不在这里。 */
export function nowTickets(tickets: Ticket[]): Ticket[] {
  return tickets.filter((ticket) => ticket.stage !== "ended" && (ticket.person || ticket.stage === "working"));
}

/** todayBadge 只计还没看的来信和到期。工单留在任务上，等人或被打断都不算今天的新简报。 */
export function todayBadge(_tickets: Ticket[], incoming: Trigger[], due: Trigger[]): number {
  return incoming.length + due.length;
}

const spoken = new Set(["desk", "github", "feishu"]);

/** visibleIncoming 是还等着人接受的触发。没被点名的邮件和日历不出现在现在。 */
export function visibleIncoming(rows: Trigger[]): Trigger[] {
  return rows.filter((row) => {
    if (!row.queued) return false;
    const quiet = row.source === "mail" || row.source === "calendar";
    if (quiet) return !!row.addressed;
    return !!row.addressed || spoken.has(row.source);
  });
}

/** dueItems 是还在队列里、带了时间、但没有点名的日历。点过名的仍走待接受，不在这里重复。 */
export function dueItems(rows: Trigger[]): Trigger[] {
  return rows.filter((row) => row.queued && row.source === "calendar" && !!row.due && !row.addressed);
}

/** wouldAccept 是观察模式下本来会接、但没有开工的决定。 */
export function wouldAccept(rows: Decision[]): Decision[] {
  return rows.filter((row) => row.kind === "ignore" && row.reason === "observed" && !row.dismissed).slice(0, 6);
}

export type TaskListFilter = "all" | "open" | "ended" | "scheduled";

/** filterTickets 按筛选和搜索留下工单。定时这一档不列工单。 */
export function filterTickets(tickets: Ticket[], filter: TaskListFilter, query = ""): Ticket[] {
  const q = query.trim().toLowerCase();
  return tickets.filter((ticket) => {
    if (filter === "scheduled") return false;
    if (filter === "open" && ticket.stage === "ended") return false;
    if (filter === "ended" && ticket.stage !== "ended") return false;
    if (!q) return true;
    const blob = `${ticket.title} ${ticket.origin} ${ticket.sender ?? ""} ${ticket.attention ?? ""}`.toLowerCase();
    return blob.includes(q);
  });
}

/** manuscriptNames 从正文里认出可打开的文件名。不读文件。 */
export function manuscriptNames(text: string): string[] {
  const found = text.match(/[^\s"'<>|]+?\.(?:csv|xlsx|xls|docx|doc|pdf|md|txt|pptx|png|jpe?g)/gi) ?? [];
  const names: string[] = [];
  const seen = new Set<string>();
  for (const raw of found) {
    const name = raw.split(/[/\\]/).pop() || raw;
    const key = name.toLowerCase();
    if (!name || name.length > 80 || seen.has(key)) continue;
    seen.add(key);
    names.push(name);
    if (names.length >= 4) break;
  }
  return names;
}

/** runningCount 是正在做的工单。等人、空闲和已结束不进任务标题上的数字。 */
export function runningCount(tickets: Ticket[]): number {
  return tickets.filter((ticket) => ticket.stage === "working").length;
}

/** taskGroups 把还开着的和最近结束的分开。结束的只留八张。 */
export function taskGroups(tickets: Ticket[]): { open: Ticket[]; ended: Ticket[] } {
  return {
    open: tickets.filter((ticket) => ticket.stage !== "ended"),
    ended: tickets.filter((ticket) => ticket.stage === "ended").slice(0, 8),
  };
}

const briefTitle = /简报|brief/i;

/** latestBrief 是最近一张标题像简报的工单。没有时为空。 */
export function latestBrief(tickets: Ticket[]): Ticket | undefined {
  for (let i = tickets.length - 1; i >= 0; i--) {
    if (briefTitle.test(tickets[i].title || "")) return tickets[i];
  }
  return undefined;
}

/** suggestedRuleIds 是可以写成规则的定时简报。已经有的编号不再推荐。 */
export const suggestedRuleIds = [
  "daily-brief",
  "waiting-reply",
  "meeting-prep",
  "weekly-status",
  "deadline-watch",
  "mentioned-mail",
] as const;

/** unseenSuggestions 去掉 ids 里已经写过的推荐。 */
export function unseenSuggestions(ids: string[]): string[] {
  const have = new Set(ids);
  return suggestedRuleIds.filter((id) => !have.has(id));
}

export type BriefItem = {
  title: string;
  detail: string;
  tone?: string;
};

/** ParsedNote 是从一段文字里认出的条目。晨间简报本身不走这条。 */
export type ParsedNote = {
  title: string;
  lede: string;
  items: BriefItem[];
  insightTitle: string;
  insight: string;
};

export type BriefChoice = { id: string; title: string; detail: string };

export type BriefStat = { label: string; value: string };

/** BriefLayout 决定一张卡怎么排。letter 是手上有的和还没有的，rows 是几条线，stats 是数得出来的数量。 */
export type BriefLayout = "checks" | "compare" | "evidence" | "chips" | "plain" | "letter" | "rows" | "stats";

/** BriefCard 是简报里的一张决定。task 和 alt 是点按钮后放进输入框的任务，不直接发出。 */
export type BriefCard = {
  id: string;
  kicker: string;
  title: string;
  lede: string;
  items: BriefItem[];
  choices: BriefChoice[];
  primary: string;
  secondary: string;
  task?: string;
  alt?: string;
  layout?: BriefLayout;
  mark?: string;
  leftLabel?: string;
  rightLabel?: string;
  note?: string;
  stats?: BriefStat[];
};

/** MorningBrief 是一叠卡。picked 只记住点过哪一个按钮。 */
export type MorningBrief = {
  cards: BriefCard[];
  picked: Record<string, string>;
  welcome?: boolean;
  woven?: boolean;
};

/** BRIEF_VERSION 只标记现在的写法。同一天已经写下的简报不因版本变化重写。 */
export const BRIEF_VERSION = 4;

/** BriefPrefs 是晨间和晚间简报的开关和时间。范围不在这里选。 */
export type BriefPrefs = {
  morningOn: boolean;
  morningAt: string;
  eveningOn: boolean;
  eveningAt: string;
};

export type BriefSlot = "morning" | "evening";

export const BRIEF_PREFS_KEY = "yoyo.brief-prefs";
export const EVENING_BRIEF_KEY = "yoyo.evening-brief";

/** defaultBriefPrefs 早上 8:30、晚上 21:30 都开着。 */
export function defaultBriefPrefs(): BriefPrefs {
  return { morningOn: true, morningAt: "08:30", eveningOn: true, eveningAt: "21:30" };
}

/** parseBriefPrefs 读偏好。缺字段时用默认。旧的盯办范围直接丢掉。 */
export function parseBriefPrefs(raw: string | null): BriefPrefs {
  const base = defaultBriefPrefs();
  if (!raw) return base;
  try {
    const body = JSON.parse(raw) as Partial<BriefPrefs>;
    return {
      morningOn: typeof body.morningOn === "boolean" ? body.morningOn : base.morningOn,
      morningAt: typeof body.morningAt === "string" && /^\d{2}:\d{2}$/.test(body.morningAt) ? body.morningAt : base.morningAt,
      eveningOn: typeof body.eveningOn === "boolean" ? body.eveningOn : base.eveningOn,
      eveningAt: typeof body.eveningAt === "string" && /^\d{2}:\d{2}$/.test(body.eveningAt) ? body.eveningAt : base.eveningAt,
    };
  } catch {
    return base;
  }
}

/** briefDue 在早晨开着、而且本地时间已经到点时为真。到点不等于打开应用时就要生成。 */
export function briefDue(prefs: BriefPrefs, now = new Date()): boolean {
  return slotDue(prefs, "morning", now);
}

/** visibleSlot 是此刻该读的那一档。没到晚上就读早晨那封，不等时钟到点才有信。 */
export function visibleSlot(prefs: BriefPrefs, now = new Date()): BriefSlot {
  return slotDue(prefs, "evening", now) ? "evening" : "morning";
}

/** slotDue 看这一档开着没有，以及本地时间过了没有。 */
export function slotDue(prefs: BriefPrefs, slot: BriefSlot, now = new Date()): boolean {
  const at = slotMinute(prefs, slot);
  if (at == null) return false;
  return clockMinute(now) >= at;
}

/** clockMinute 是本地时钟的分钟数，从午夜算起。 */
export function clockMinute(now = new Date()): number {
  return now.getHours() * 60 + now.getMinutes();
}

/** slotsCrossed 只在已经看着钟、并且这一小段里跨过设定分钟时返回那一档。
 *  第一次观察和隔了太久的补跑都不算，避免一打开就把错过的简报补上。 */
export function slotsCrossed(prefs: BriefPrefs, prevMinute: number, nowMinute: number): BriefSlot[] {
  if (prevMinute < 0) return [];
  let gap = nowMinute - prevMinute;
  if (gap < 0) gap += 24 * 60;
  if (gap <= 0 || gap > 3) return [];
  const out: BriefSlot[] = [];
  for (const slot of ["morning", "evening"] as BriefSlot[]) {
    const at = slotMinute(prefs, slot);
    if (at == null) continue;
    const hit = prevMinute < nowMinute ? at > prevMinute && at <= nowMinute : at > prevMinute || at <= nowMinute;
    if (hit) out.push(slot);
  }
  return out;
}

/** offerSlot 是当前时段可以点开的那一档。上午给晨间，晚间给晚间，其余时间不提供。 */
export function offerSlot(prefs: BriefPrefs, now = new Date()): BriefSlot | null {
  const hour = now.getHours();
  if (hour >= 5 && hour < 12) return slotMinute(prefs, "morning") == null ? null : "morning";
  const at = slotMinute(prefs, "evening");
  if (at == null) return null;
  if (hour >= 18 || (hour >= 12 && clockMinute(now) >= at)) return "evening";
  return null;
}

/** compareEditions 同一天晚间在晨间上面，更晚的日期在上面。不按写成时间排。 */
export function compareEditions(a: { day: string; slot: BriefSlot; at: string }, b: { day: string; slot: BriefSlot; at: string }): number {
  if (a.day !== b.day) return a.day < b.day ? 1 : -1;
  const rank = (slot: BriefSlot) => (slot === "evening" ? 0 : 1);
  if (rank(a.slot) !== rank(b.slot)) return rank(a.slot) - rank(b.slot);
  return (Date.parse(b.at) || 0) - (Date.parse(a.at) || 0);
}

function slotMinute(prefs: BriefPrefs, slot: BriefSlot): number | null {
  const on = slot === "evening" ? prefs.eveningOn : prefs.morningOn;
  if (!on) return null;
  const at = slot === "evening" ? prefs.eveningAt : prefs.morningAt;
  const [hour, minute] = at.split(":").map((part) => Number(part));
  if (!Number.isFinite(hour) || !Number.isFinite(minute)) return null;
  return hour * 60 + minute;
}

/** briefStorageKey 早晨和晚上各存一份。 */
export function briefStorageKey(slot: BriefSlot): string {
  return slot === "evening" ? EVENING_BRIEF_KEY : MORNING_BRIEF_KEY;
}

const itemLine = /^[\s>*\-·]*[\d.、]*\s*([^：:\n]{2,18})[：:]\s*(\S.{1,120})$/;

/** composeBrief 从正文里收出一张简报。优先认 json 代码块，否则认「标题：说明」的行。不够两项时为空。 */
export function composeBrief(text: string): ParsedNote | null {
  const parsed = briefFromJson(text);
  if (parsed) return parsed;
  const items: BriefItem[] = [];
  const rest: string[] = [];
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (!line) continue;
    const matched = line.match(itemLine);
    const title = matched?.[1]?.replace(/^[\d.、]+\s*/, "").trim() || "";
    const detail = matched?.[2]?.trim() || "";
    if (matched && items.length < 6 && title && detail && !title.includes("http")) {
      items.push({ title, detail });
      continue;
    }
    if (line.length >= 6) rest.push(line);
  }
  if (items.length < 2) return null;
  const lede = rest[0]?.slice(0, 96) || "";
  const insight = rest.length > 1 ? rest[rest.length - 1].slice(0, 140) : "";
  return { title: "", lede, items, insightTitle: "", insight };
}

function briefFromJson(text: string): ParsedNote | null {
  const fenced = text.match(/```json\s*([\s\S]*?)```/i);
  let raw = fenced?.[1]?.trim() || "";
  if (!raw) {
    const start = text.indexOf("{");
    const end = text.lastIndexOf("}");
    if (start < 0 || end <= start || !text.slice(start, end + 1).includes("\"items\"")) return null;
    raw = text.slice(start, end + 1);
  }
  try {
    const body = JSON.parse(raw) as {
      title?: unknown; lede?: unknown; insightTitle?: unknown; insight?: unknown;
      items?: { title?: unknown; detail?: unknown }[];
    };
    const items = (body.items || [])
      .map((item) => ({ title: String(item.title || "").trim(), detail: String(item.detail || "").trim() }))
      .filter((item) => item.title && item.detail)
      .slice(0, 6);
    if (items.length < 2) return null;
    return {
      title: String(body.title || "").trim(),
      lede: String(body.lede || "").trim(),
      items,
      insightTitle: String(body.insightTitle || "").trim(),
      insight: String(body.insight || "").trim(),
    };
  } catch {
    return null;
  }
}

export type BriefMemory = { kind: string; text: string; staging?: boolean; source?: string };
export type BriefSource = { name: string; state: string };
export type BriefPoint = { name: string; kind: string };

/** BriefPhrases 是简报审计用的句子。界面按当前语言填进来。 */
export type BriefPhrases = {
  dueTitle: string;
  personTitle: string;
  stuckTitle: string;
  quietTitle: string;
  lede: string;
  quietLede: string;
  insightTitle: string;
  insight: string;
  rules: string;
  overdue: string;
  dueToday: string;
  dueSoon: string;
  waiting: string;
  blocked: string;
  failed: string;
  observed: string;
  source: string;
  profile: string;
  project: string;
  episodic: string;
  semantic: string;
  poiTitle: string;
  poiLede: string;
  aroundTitle: string;
  aroundLede: string;
  aroundEmpty: string;
  aroundEmptyDetail: string;
  appSeen: string;
  traceSeen: string;
  statApps: string;
  statTraces: string;
  statOpen: string;
  statDated: string;
  open: string;
  nothing: string;
  nothingDetail: string;
  keep: string;
  hold: string;
  directionTitle: string;
  directionLede: string;
  useChoice: string;
  keepAll: string;
  keptTitle: string;
  keptLede: string;
  useMemory: string;
  skipMemory: string;
  afterTitle: string;
  afterLede: string;
  lookFirst: string;
  leaveAll: string;
  watchTitle: string;
  watchLede: string;
  watchDue: string;
  watchWait: string;
  watchOpen: string;
  watchMemory: string;
  saveWatch: string;
  leaveWatch: string;
  situationKicker: string;
  emptyKicker: string;
  gateKicker: string;
  gateTitle: string;
  gateLede: string;
  gatePrimary: string;
  gateSecondary: string;
  gateTask: string;
  gateOne: string;
  splitKicker: string;
  splitTitle: string;
  splitLede: string;
  splitPrimary: string;
  splitTask: string;
  memoryKicker: string;
  memoryTitle: string;
  memoryLede: string;
  memoryPrimary: string;
  memorySecondary: string;
  memoryTask: string;
  memoryOne: string;
  aroundKicker: string;
  aroundPrimary: string;
  aroundSecondary: string;
  aroundTask: string;
  aroundOne: string;
  nextTitle: string;
  nextPrimary: string;
  nextTask: string;
  quietPrimary: string;
  quietTask: string;
  countLabel: string;
  letterKicker: string;
  letterKickerToday: string;
  letterHello: string;
  letterHelloBare: string;
  letterLede: string;
  letterLedeToday: string;
  letterKnown: string;
  letterMissing: string;
  letterLines: string;
  letterIdentity: string;
  letterHost: string;
  letterOs: string;
  letterGit: string;
  letterApps: string;
  letterHours: string;
  letterNoMail: string;
  letterNoMailDetail: string;
  letterNoPlan: string;
  letterNoPlanDetail: string;
  linesKicker: string;
  linesTitle: string;
  linesLede: string;
  lineMoving: string;
  lineStalled: string;
  lineQuiet: string;
  linesPrimary: string;
  linesTask: string;
  doneKicker: string;
  doneTitle: string;
  doneLede: string;
  doneMark: string;
  doneFinished: string;
  doneEmpty: string;
  donePrimary: string;
  doneAlt: string;
  doneTask: string;
  placeKicker: string;
  placeTitle: string;
  placeLede: string;
  placeLeft: string;
  placeRight: string;
  placeNone: string;
  placePrimary: string;
  placeTask: string;
  toolKicker: string;
  toolTitle: string;
  toolLede: string;
  toolProjects: string;
  toolApps: string;
  toolDownloads: string;
  toolTraces: string;
  toolPrimary: string;
  toolTask: string;
  pauseKicker: string;
  pauseTitle: string;
  pauseLede: string;
  pauseGo: string;
  pauseGoDetail: string;
  pauseStop: string;
  pauseStopDetail: string;
  pauseNote: string;
};

/** DeskCensus 是本机清点。只有名字、状态和数量，没有文件正文。 */
export type DeskCensus = {
  user?: string;
  host?: string;
  os?: string;
  git?: string;
  hours?: string;
  weekday?: boolean;
  appNames?: string[];
  lines?: { id?: string; name: string; state?: string; evidence?: string; when?: string }[];
  finished?: { name: string; line?: string; kind?: string }[];
  empty?: { name: string; line?: string; kind?: string }[];
  place?: { name: string; line?: string; kind?: string }[];
  archive?: { name: string; line?: string; kind?: string }[];
  projects?: number;
  downloads?: number;
  apps?: number;
  traces?: number;
};

export const MORNING_BRIEF_KEY = "yoyo.morning-brief";

/** briefDay 是本地日期。过了零点才算新的一天。 */
export function briefDay(now = new Date()): string {
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${now.getFullYear()}-${month}-${day}`;
}

/** parseStoredBrief 只认同一天存下的简报。过期或没有卡片时为空。版本不一致也留着，避免到点之后又重写一封。 */
export function parseStoredBrief(raw: string | null, day: string): MorningBrief | null {
  if (!raw) return null;
  try {
    const body = JSON.parse(raw) as { v?: unknown; day?: unknown; brief?: MorningBrief };
    if (body.day !== day || !body.brief || !Array.isArray(body.brief.cards) || body.brief.cards.length < 1) return null;
    const picked = body.brief.picked && typeof body.brief.picked === "object" ? body.brief.picked : {};
    return { cards: body.brief.cards, picked, welcome: body.brief.welcome === true, woven: body.brief.woven === true };
  } catch {
    return null;
  }
}

/** serializeStoredBrief 把当天的简报收成一行。 */
export function serializeStoredBrief(day: string, brief: MorningBrief): string {
  return JSON.stringify({ v: BRIEF_VERSION, day, brief });
}

/** briefPrompt 是点某个按钮后放进输入框的文字。旧卡没有写好的任务时，用标题和条目拼一条。 */
export function briefPrompt(card: BriefCard, which: "primary" | "secondary"): string {
  const stored = which === "primary" ? card.task : card.alt;
  if (stored?.trim()) return stored.trim();
  const label = which === "primary" ? card.primary : card.secondary;
  const body = card.items.map((item) => item.detail ? `${item.title}：${item.detail}` : item.title).join("\n");
  return [card.title, card.lede, body, label].filter(Boolean).join("\n");
}

type Finding = { rank: number; order: number; title: string; detail: string; bucket: "due" | "wait" | "stuck" | "other" };

/** auditBrief 按时效审计资产。过期先于今天，今天先于等人，等人先于卡住。不读对话。 */
export function auditBrief(input: {
  tickets?: Ticket[];
  queued?: Trigger[];
  decisions?: Decision[];
  rules?: Rule[];
  memory?: BriefMemory[];
  sources?: BriefSource[];
  points?: BriefPoint[];
  census?: DeskCensus;
  welcome?: boolean;
  now?: Date;
}, phrases: BriefPhrases): MorningBrief {
  const now = input.now ?? new Date();
  const findings: Finding[] = [];
  let order = 0;
  const push = (rank: number, bucket: Finding["bucket"], title: string, detail: string) => {
    const name = clip(title, 18);
    const body = clip(detail, 120);
    if (!name || !body) return;
    findings.push({ rank, order: order++, title: name, detail: body, bucket });
  };
  for (const row of input.queued ?? []) {
    if (!row.queued || row.source !== "calendar" || !row.due) continue;
    const delta = dayDelta(row.due, now);
    if (delta === null || delta > 14) continue;
    const when = delta < 0 ? phrases.overdue : delta === 0 ? phrases.dueToday : phrases.dueSoon;
    const rank = delta < 0 ? 0 : delta === 0 ? 1 : 2;
    push(rank, "due", row.text || row.id, when);
  }
  for (const ticket of input.tickets ?? []) {
    if (ticket.stage === "ended") continue;
    const summary = ticket.assignments?.slice(-1)[0]?.summary || "";
    const aside = summary && summary !== (ticket.title || ticket.id) ? clip(summary, 80) : "";
    if (needsPerson(ticket)) {
      push(3, "wait", ticket.title || ticket.id, aside || phrases.waiting);
      continue;
    }
    if (ticket.action === "blocked") push(4, "stuck", ticket.title || ticket.id, aside || phrases.blocked);
    else if (ticket.action === "failed") push(4, "stuck", ticket.title || ticket.id, aside || phrases.failed);
  }
  const byTrigger = new Map((input.queued ?? []).map((row) => [row.id, row]));
  for (const decision of wouldAccept(input.decisions ?? [])) {
    const text = byTrigger.get(decision.triggerId)?.text || decision.thread || "";
    if (!text) continue;
    push(5, "other", text, phrases.observed);
  }
  for (const source of input.sources ?? []) {
    if (source.state !== "error" || !source.name) continue;
    push(4, "stuck", source.name, phrases.source);
  }
  findings.sort((a, b) => a.rank - b.rank || a.order - b.order);
  const seen = new Set<string>();
  const shown = findings.filter((item) => {
    if (seen.has(item.title)) return false;
    seen.add(item.title);
    return true;
  }).slice(0, 6);
  const items = shown.map((item) => ({ title: item.title, detail: item.detail }));
  const cards: BriefCard[] = [];
  const desk = deskCards(input.census, !!input.welcome, shown.some((item) => item.bucket === "due"), items.length > 0, phrases, input.memory);
  const letter = desk.find((card) => card.id === "letter");
  if (letter) cards.push(letter);
  if (items.length > 0) {
    const listed = itemLines(items);
    cards.push({
      id: "situation",
      layout: "checks",
      kicker: `${phrases.gateKicker} · ${briefDay(now)}`,
      title: fill(phrases.gateTitle, { n: String(items.length) }),
      lede: phrases.gateLede,
      items,
      choices: [],
      primary: fill(phrases.gatePrimary, { n: String(items.length) }),
      secondary: phrases.gateSecondary,
      task: fill(phrases.gateTask, { items: listed }),
      alt: fill(phrases.gateOne, { title: items[0].title, detail: items[0].detail }),
      mark: fill(phrases.countLabel, { n: String(items.length) }),
    });
  }
  for (const card of desk) {
    if (card.id !== "letter" && card.id !== "pause") cards.push(card);
  }
  const openTickets = (input.tickets ?? []).filter((ticket) => ticket.stage !== "ended" && !needsPerson(ticket) && ticket.action !== "blocked" && ticket.action !== "failed");
  const named = new Set(items.map((item) => item.title));
  const fresh = openTickets.filter((ticket) => !named.has(clip(ticket.title || ticket.id, 18)));
  const pair = (fresh.length >= 2 ? fresh : openTickets).slice(0, 2);
  if (pair.length >= 2 && fresh.length >= 2) {
    const left = workChoice(pair[0], phrases);
    const right = workChoice(pair[1], phrases);
    cards.push({
      id: "direction",
      layout: "compare",
      kicker: phrases.splitKicker,
      title: phrases.splitTitle,
      lede: phrases.splitLede,
      items: [],
      choices: [left, right],
      primary: fill(phrases.splitPrimary, { title: left.title }),
      secondary: fill(phrases.splitPrimary, { title: right.title }),
      task: fill(phrases.splitTask, { title: left.title, detail: left.detail }),
      alt: fill(phrases.splitTask, { title: right.title, detail: right.detail }),
    });
  } else if (fresh.length === 1) {
    const only = workChoice(fresh[0], phrases);
    cards.push({
      id: "next",
      layout: "evidence",
      kicker: phrases.splitKicker,
      title: phrases.nextTitle,
      lede: phrases.splitLede,
      items: [{ title: only.title, detail: only.detail }],
      choices: [],
      primary: fill(phrases.nextPrimary, { title: only.title }),
      secondary: "",
      task: fill(phrases.nextTask, { title: only.title, detail: only.detail }),
    });
  }
  const memories = (input.memory ?? []).filter((row) => !row.staging && row.text.trim()).slice(0, 3);
  if (memories.length > 0) {
    const memoryItems = memories.map((row) => ({
      title: memoryLabel(phrases, row.kind),
      detail: clip(row.text, 80),
    }));
    cards.push({
      id: "kept",
      layout: "evidence",
      kicker: phrases.memoryKicker,
      title: phrases.memoryTitle,
      lede: phrases.memoryLede,
      items: memoryItems,
      choices: [],
      primary: phrases.memoryPrimary,
      secondary: phrases.memorySecondary,
      task: fill(phrases.memoryTask, { items: itemLines(memoryItems) }),
      alt: fill(phrases.memoryOne, { title: memoryItems[0].title, detail: memoryItems[0].detail }),
    });
  }
  const around = input.census?.lines?.length ? null : aroundCard(input.points ?? [], phrases);
  if (around) cards.push(around);
  const pause = desk.find((card) => card.id === "pause");
  if (pause && items.length === 0) cards.push(pause);
  if (cards.length === 0) {
    cards.push({
      id: "quiet",
      layout: "plain",
      kicker: phrases.emptyKicker,
      title: phrases.quietTitle,
      lede: phrases.quietLede,
      items: [],
      choices: [],
      primary: phrases.quietPrimary,
      secondary: "",
      task: phrases.quietTask,
    });
  }
  return { cards, picked: {}, welcome: !!input.welcome && !!letter };
}

/** deskCards 把本机清点收成信、工程线、成品、归位和工具链。没有清点、也不是第一封时什么都不加。不编健康读数，不读正文。 */
function deskCards(census: DeskCensus | undefined, welcome: boolean, hasPlan: boolean, hasFindings: boolean, phrases: BriefPhrases, memory?: BriefMemory[]): BriefCard[] {
  if (!welcome && !censusUseful(census)) return [];
  const cards: BriefCard[] = [];
  const name = greetName(memory, census?.user || "", census?.git || "");
  const known: BriefItem[] = [];
  const lines = census?.lines ?? [];
  if (census?.user) known.push({ title: phrases.letterIdentity, detail: census.user });
  if (census?.host) known.push({ title: phrases.letterHost, detail: census.host });
  if (census?.os) known.push({ title: phrases.letterOs, detail: census.os });
  if (census?.git) known.push({ title: phrases.letterGit, detail: census.git });
  if (census?.hours) known.push({ title: phrases.letterHours, detail: census.hours });
  if (lines.length) known.push({ title: fill(phrases.letterLines, { n: String(lines.length) }), detail: lines.map((line) => line.name).join(" · ") });
  const apps = (census?.appNames ?? []).map((item) => item.trim()).filter(Boolean).slice(0, 6);
  if (apps.length) known.push({ title: phrases.letterApps, detail: apps.join(" · ") });
  const missing: BriefChoice[] = hasPlan ? [] : [
    { id: "mail", title: phrases.letterNoMail, detail: phrases.letterNoMailDetail },
    { id: "plan", title: phrases.letterNoPlan, detail: phrases.letterNoPlanDetail },
  ];
  cards.push({
    id: "letter",
    layout: "letter",
    kicker: welcome ? phrases.letterKicker : phrases.letterKickerToday,
    title: name ? fill(phrases.letterHello, { name }) : phrases.letterHelloBare,
    lede: welcome ? phrases.letterLede : phrases.letterLedeToday,
    items: known,
    choices: missing,
    primary: "",
    secondary: "",
    leftLabel: phrases.letterKnown,
    rightLabel: phrases.letterMissing,
  });
  if (lines.length) {
    const rows = lines.map((line) => ({
      title: line.name,
      detail: lineDetail(line, phrases),
      tone: line.state || "quiet",
    }));
    const first = lines[0];
    cards.push({
      id: "lines",
      layout: "rows",
      kicker: phrases.linesKicker,
      title: phrases.linesTitle,
      lede: phrases.linesLede,
      items: rows,
      choices: [],
      primary: fill(phrases.linesPrimary, { name: first.name }),
      secondary: "",
      task: fill(phrases.linesTask, { name: first.name, items: itemLines(rows) }),
    });
  }
  const finished = census?.finished ?? [];
  const empty = census?.empty ?? [];
  if (finished.length || empty.length) {
    const doneItems = [
      ...finished.map((item) => ({ title: item.name, detail: item.line ? `${phrases.doneFinished} · ${item.line}` : phrases.doneFinished, tone: "moving" })),
      ...empty.map((item) => ({ title: item.name, detail: item.line ? `${phrases.doneEmpty} · ${item.line}` : phrases.doneEmpty, tone: "stalled" })),
    ];
    cards.push({
      id: "done",
      layout: "rows",
      kicker: phrases.doneKicker,
      title: phrases.doneTitle,
      lede: phrases.doneLede,
      items: doneItems,
      choices: [],
      primary: empty.length ? phrases.donePrimary : phrases.doneAlt,
      secondary: "",
      task: fill(phrases.doneTask, { items: itemLines(doneItems) }),
      mark: fill(phrases.doneMark, { done: String(finished.length), left: String(empty.length) }),
    });
  }
  const place = census?.place ?? [];
  const archive = census?.archive ?? [];
  if (place.length || archive.length) {
    const placeText = namedLines(place) || phrases.placeNone;
    const archiveText = namedLines(archive) || phrases.placeNone;
    cards.push({
      id: "place",
      layout: "compare",
      kicker: phrases.placeKicker,
      title: phrases.placeTitle,
      lede: phrases.placeLede,
      items: [],
      choices: [
        { id: "place", title: phrases.placeLeft, detail: placeText },
        { id: "archive", title: phrases.placeRight, detail: archiveText },
      ],
      primary: phrases.placePrimary,
      secondary: "",
      leftLabel: phrases.placeLeft,
      rightLabel: phrases.placeRight,
      task: fill(phrases.placeTask, { place: placeText, archive: archiveText }),
    });
  }
  const stats = [
    census?.projects ? { label: phrases.toolProjects, value: String(census.projects) } : null,
    census?.apps ? { label: phrases.toolApps, value: String(census.apps) } : null,
    census?.downloads ? { label: phrases.toolDownloads, value: String(census.downloads) } : null,
    census?.traces ? { label: phrases.toolTraces, value: String(census.traces) } : null,
  ].filter((stat): stat is BriefStat => !!stat);
  if (stats.length > 0) {
    const listed = [
      ...lines.map((line) => line.name),
      ...place.map((item) => item.name),
      ...archive.map((item) => item.name),
    ].join("\n");
    cards.push({
      id: "tools",
      layout: "stats",
      kicker: phrases.toolKicker,
      title: phrases.toolTitle,
      lede: phrases.toolLede,
      items: [],
      choices: [],
      stats,
      primary: phrases.toolPrimary,
      secondary: "",
      task: fill(phrases.toolTask, { items: listed }),
    });
  }
  if (census?.weekday && !hasFindings) {
    cards.push({
      id: "pause",
      layout: "compare",
      kicker: phrases.pauseKicker,
      title: phrases.pauseTitle,
      lede: phrases.pauseLede,
      items: [],
      choices: [
        { id: "go", title: phrases.pauseGo, detail: phrases.pauseGoDetail },
        { id: "stop", title: phrases.pauseStop, detail: phrases.pauseStopDetail },
      ],
      primary: "",
      secondary: "",
      note: phrases.pauseNote,
    });
  }
  return cards;
}

function censusUseful(census?: DeskCensus): boolean {
  if (!census) return false;
  return Boolean(census.lines?.length || census.finished?.length || census.empty?.length || census.place?.length || census.archive?.length || census.hours || census.projects || census.downloads || census.user || census.host || census.os || census.git || census.appNames?.length);
}

function greetName(memory: BriefMemory[] | undefined, fallback: string, git?: string): string {
  const said = (memory ?? []).find((row) => !row.staging && row.kind === "profile" && operatorMemory(row.source) && memoryName(row.text.trim()));
  if (said) return said.text.trim();
  const named = (git || "").trim();
  if (memoryName(named)) return named;
  const login = fallback.trim();
  if (memoryName(login)) return login;
  return "";
}

function lineDetail(line: { state?: string; evidence?: string; when?: string }, phrases: BriefPhrases): string {
  const state = line.state === "stalled" ? phrases.lineStalled : line.state === "quiet" ? phrases.lineQuiet : phrases.lineMoving;
  return [state, line.evidence, line.when].filter(Boolean).join(" · ");
}

function namedLines(items: { name: string; line?: string }[]): string {
  return items.map((item) => item.line ? `${item.name} → ${item.line}` : item.name).join("\n");
}

function workChoice(ticket: Ticket, phrases: BriefPhrases): BriefChoice {
  const summary = ticket.assignments?.slice(-1)[0]?.summary || "";
  return {
    id: ticket.id,
    title: clip(ticket.title || ticket.id, 18),
    detail: summary && summary !== (ticket.title || "") ? clip(summary, 80) : phrases.open,
  };
}

function itemLines(items: BriefItem[]): string {
  return items.map((item, index) => `${index + 1}. ${item.title}：${item.detail}`).join("\n");
}

function aroundCard(points: BriefPoint[], phrases: BriefPhrases): BriefCard | null {
  const apps = points.filter((point) => point.kind === "app" && point.name.trim());
  const traces = points.filter((point) => (point.kind === "trace" || point.kind === "desk") && point.name.trim());
  const items = [
    ...apps.slice(0, 4).map((point) => ({ title: clip(point.name, 18), detail: phrases.appSeen })),
    ...traces.slice(0, 4).map((point) => ({ title: clip(point.name, 18), detail: phrases.traceSeen })),
  ].slice(0, 6);
  if (items.length === 0) return null;
  return {
    id: "around",
    layout: "chips",
    kicker: phrases.aroundKicker,
    title: phrases.aroundTitle,
    lede: phrases.aroundLede,
    items,
    choices: [],
    primary: phrases.aroundPrimary,
    secondary: phrases.aroundSecondary,
    task: fill(phrases.aroundTask, { items: itemLines(items) }),
    alt: fill(phrases.aroundOne, { title: items[0].title, detail: items[0].detail }),
  };
}

function memoryLabel(phrases: BriefPhrases, kind: string): string {
  if (kind === "project") return phrases.project;
  if (kind === "episodic") return phrases.episodic;
  if (kind === "semantic") return phrases.semantic;
  if (kind === "desk") return phrases.poiTitle;
  return phrases.profile;
}

function fill(text: string, values: Record<string, string>): string {
  return text.replace(/\{(\w+)\}/g, (_, key: string) => values[key] ?? "");
}

function clip(text: string, max: number): string {
  const clean = text.replace(/\s+/g, " ").trim();
  if (clean.length <= max) return clean;
  return `${clean.slice(0, max - 1)}…`;
}

export type MemoryRow = {
  id: string;
  kind: string;
  text: string;
  staging?: boolean;
  project?: string;
  source?: string;
  state?: string;
  evidence?: string;
  when?: string;
  created_at?: string;
};

export type MemoryPortrait = {
  name: string;
  updated: string;
  device: string;
  host: string;
  os: string;
  zone: string;
  git: string;
  hours: string;
  lines: { name: string; state: string; evidence: string; when: string }[];
  said: string[];
  noted: string[];
  preferences: string[];
};

/** memoryPortrait 把已经留下的条目分成设备、工作时间、工程线，以及当事人说过的话。本机事实不算当事人讲过。 */
export function memoryPortrait(rows: MemoryRow[]): MemoryPortrait {
  const out: MemoryPortrait = { name: "", updated: "", device: "", host: "", os: "", zone: "", git: "", hours: "", lines: [], said: [], noted: [], preferences: [] };
  for (const row of rows) {
    const text = row.text.trim();
    if (!text) continue;
    if (row.id === "machine:device") out.device = text;
    else if (row.id === "machine:host") out.host = text;
    else if (row.id === "machine:os") out.os = text;
    else if (row.id === "machine:zone") out.zone = text;
    else if (row.id === "machine:git") out.git = text;
    else if (row.id === "machine:hours" || (row.source === "machine" && row.project === "work")) out.hours = text;
    else if (row.kind === "project") out.lines.push({ name: text, state: row.state || "", evidence: row.evidence || "", when: row.when || "" });
    else if (row.kind === "profile" && operatorMemory(row.source)) out.said.push(text);
    else if (row.kind === "profile") out.noted.push(text);
    else if (row.kind === "semantic") out.preferences.push(text);
    else continue;
    if (row.created_at && row.created_at > out.updated) out.updated = row.created_at;
  }
  out.name = out.said.find((text) => memoryName(text)) || "";
  return out;
}

export type MemoryFolder = "device" | "work" | "project" | "preference" | "about" | "notes";

/** memoryFolderRows 按夹取出条目。关于夹收档案，不收本机账号和工作时间。 */
export function memoryFolderRows(rows: MemoryRow[], folder: MemoryFolder): MemoryRow[] {
  return rows.filter((row) => row.text.trim() && inMemoryFolder(row, folder));
}

function inMemoryFolder(row: MemoryRow, folder: MemoryFolder): boolean {
  if (folder === "device") return row.source === "machine" && row.project === "device";
  if (folder === "work") return row.source === "machine" && (row.id === "machine:hours" || row.project === "work");
  if (folder === "project") return row.kind === "project";
  if (folder === "preference") return row.kind === "semantic";
  if (folder === "about") return row.kind === "profile" && row.source !== "machine";
  return row.kind === "episodic" && row.source !== "machine" && !row.id.startsWith("machine:");
}

function operatorMemory(source?: string): boolean {
  const value = (source || "").trim();
  return value === "" || value === "user" || value === "User confirmed in chat";
}

function memoryName(text: string): boolean {
  return [...text].length <= 16 && !/[\s\n。！？!?]/.test(text);
}

function dayDelta(value: string, now: Date): number | null {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return null;
  const day = new Date(parsed);
  const due = Date.UTC(day.getFullYear(), day.getMonth(), day.getDate());
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((due - today) / 86400000);
}

export type SenseSignal = {
  id: string;
  code: string;
  class: string;
  name: string;
  detail: string;
  state: string;
  why: string;
  speak: boolean;
  held: boolean;
  memoryId?: string;
};

export type Initiative = {
  day?: string;
  speak?: boolean;
  signals?: SenseSignal[];
  open?: { id: string; class: string; title: string; why?: string }[];
};

export type SenseLabels = {
  unchecked: string;
  linked: string;
  senseTitle: string;
  senseLede: string;
  listening: string;
  battery: string;
  charging: string;
  online: string;
  offline: string;
  idle: string;
  focus: string;
  power: string;
  net: string;
  git: string;
  mail: string;
  calendar: string;
  poi: string;
  speakKicker: string;
  speakTitle: string;
  speakLede: string;
  speakPrimary: string;
  speakTask: string;
  why: Record<string, string>;
};

/** shouldPush 在没有判断结果时保持原来的到点推进。有判断时，只有第一封信或这一轮值得开口才推进。 */
export function shouldPush(initiative: Initiative | undefined, welcome: boolean): boolean {
  if (!initiative) return true;
  return welcome || !!initiative.speak;
}

/** applySenses 把未核对的邮箱日历写进信。电量、前台和桌面文件名不进简报，只有低电量、断网和来源错误才另起一行。 */
export function applySenses(brief: MorningBrief, signals: SenseSignal[], labels: SenseLabels): MorningBrief {
  const cards = brief.cards.map((card) => card.id === "letter" ? letterWithSenses(card, signals, labels) : card);
  const sensed = signals.filter(senseRow);
  if (sensed.length) {
    cards.push({
      id: "sense",
      layout: "rows",
      kicker: labels.senseTitle,
      title: labels.senseTitle,
      lede: labels.senseLede,
      items: sensed.slice(0, 8).map((signal) => senseItem(signal, labels)),
      choices: [],
      primary: "",
      secondary: "",
    });
  }
  const speaks = signals.filter((signal) => signal.speak && signal.code !== "welcome");
  if (speaks.length) {
    const items = speaks.map((signal) => ({
      title: signal.name,
      detail: [labels.why[signal.why] || signal.why, signal.detail].filter(Boolean).join(" · "),
    }));
    const at = cards.findIndex((card) => card.id === "letter");
    cards.splice(at + 1, 0, {
      id: "speak",
      layout: "rows",
      kicker: labels.speakKicker,
      title: labels.speakTitle,
      lede: labels.speakLede,
      items,
      choices: [],
      primary: labels.speakPrimary,
      secondary: "",
      task: fill(labels.speakTask, { items: itemLines(items) }),
    });
  }
  return { ...brief, cards };
}

/** mergeSenseCards 用新的信、感知和开口原因换掉旧的那几张，保留已经点过的选择。 */
export function mergeSenseCards(stored: MorningBrief, fresh: MorningBrief): MorningBrief {
  const lead = new Set(["letter", "sense", "speak"]);
  return {
    ...stored,
    cards: [...fresh.cards.filter((card) => lead.has(card.id)), ...stored.cards.filter((card) => !lead.has(card.id))],
  };
}

function letterWithSenses(card: BriefCard, signals: SenseSignal[], labels: SenseLabels): BriefCard {
  const mail = signals.find((signal) => signal.code === "mail" && (signal.state === "unchecked" || signal.state === "linked"));
  const calendar = signals.find((signal) => signal.code === "calendar" && (signal.state === "unchecked" || signal.state === "linked"));
  if (!mail && !calendar) return card;
  const choices = card.choices.filter((choice) => choice.id !== "mail");
  const lines = [mail, calendar].filter((signal): signal is SenseSignal => !!signal).map((signal) => {
    const title = signal.code === "mail" ? labels.mail : labels.calendar;
    const detail = signal.state === "linked" ? labels.linked : labels.unchecked;
    return `${title} · ${detail}`;
  });
  choices.unshift({ id: "mail", title: labels.mail, detail: lines.join("\n") });
  return { ...card, choices };
}

function senseRow(signal: SenseSignal): boolean {
  if (signal.speak) return false;
  if (signal.code === "power") return signal.state === "low";
  if (signal.code === "net") return signal.state === "offline";
  if (signal.code === "channel") return signal.state === "error";
  return false;
}

function senseItem(signal: SenseSignal, labels: SenseLabels): BriefItem {
  if (signal.code === "power") {
    return { title: labels.power, detail: `${signal.name}% · ${signal.detail === "charging" ? labels.charging : labels.battery}` };
  }
  if (signal.code === "net") return { title: labels.net, detail: signal.state === "offline" ? labels.offline : labels.online };
  if (signal.code === "idle") return { title: labels.idle, detail: signal.detail };
  if (signal.code === "focus") return { title: labels.focus, detail: signal.name };
  if (signal.code === "git") return { title: signal.name, detail: [signal.detail, signal.state].filter(Boolean).join(" · ") };
  if (signal.code === "channel") return { title: signal.name, detail: signal.state === "error" ? (signal.detail || labels.why.error || signal.state) : labels.listening };
  if (signal.code === "poi") return { title: labels.poi, detail: signal.name };
  return { title: signal.name, detail: signal.detail };
}
