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

export type BriefItem = { title: string; detail: string };

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

/** BriefLayout 决定一张卡怎么排。checks 是编号清单，compare 是两列取舍，evidence 是可核对的原文，chips 是名字。 */
export type BriefLayout = "checks" | "compare" | "evidence" | "chips" | "plain";

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
  stats?: BriefStat[];
};

/** MorningBrief 是一叠卡。picked 只记住点过哪一个按钮。 */
export type MorningBrief = {
  cards: BriefCard[];
  picked: Record<string, string>;
};

/** BRIEF_VERSION 变了，当天旧的流水账就作废，下一轮按新的卡片重写。 */
export const BRIEF_VERSION = 2;

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

/** briefDue 在早晨开着、而且本地时间已经到点时为真。 */
export function briefDue(prefs: BriefPrefs, now = new Date()): boolean {
  return slotDue(prefs, "morning", now);
}

/** slotDue 看这一档开着没有，以及本地时间过了没有。 */
export function slotDue(prefs: BriefPrefs, slot: BriefSlot, now = new Date()): boolean {
  const on = slot === "evening" ? prefs.eveningOn : prefs.morningOn;
  const at = slot === "evening" ? prefs.eveningAt : prefs.morningAt;
  if (!on) return false;
  const [hour, minute] = at.split(":").map((part) => Number(part));
  if (!Number.isFinite(hour) || !Number.isFinite(minute)) return false;
  return now.getHours() * 60 + now.getMinutes() >= hour * 60 + minute;
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

export type BriefMemory = { kind: string; text: string; staging?: boolean };
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
};

export const MORNING_BRIEF_KEY = "yoyo.morning-brief";

/** briefDay 是本地日期。过了零点才算新的一天。 */
export function briefDay(now = new Date()): string {
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${now.getFullYear()}-${month}-${day}`;
}

/** parseStoredBrief 只认同一天存下的简报。过期或坏掉的为空。 */
export function parseStoredBrief(raw: string | null, day: string): MorningBrief | null {
  if (!raw) return null;
  try {
    const body = JSON.parse(raw) as { v?: unknown; day?: unknown; brief?: MorningBrief };
    if (body.v !== BRIEF_VERSION || body.day !== day || !body.brief || !Array.isArray(body.brief.cards) || body.brief.cards.length < 1) return null;
    const picked = body.brief.picked && typeof body.brief.picked === "object" ? body.brief.picked : {};
    return { cards: body.brief.cards, picked };
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
  const around = aroundCard(input.points ?? [], phrases);
  if (around) cards.push(around);
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
  return { cards, picked: {} };
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

function dayDelta(value: string, now: Date): number | null {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return null;
  const day = new Date(parsed);
  const due = Date.UTC(day.getFullYear(), day.getMonth(), day.getDate());
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((due - today) / 86400000);
}
