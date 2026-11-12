/**
 * 拥有：到点时把晨间、晚间简报推进主会话，读回 Go 那边的开口判断，以及短讯上的反馈。
 * 不拥有：简报卡片怎么画，那在 WorkPane 和简报弹层。开口短讯由 Go 的送达循环写进主会话，这里不推。
 */

import { briefList, composeToday, deskFeedback, deskPush, deskState, mainSession, orchestratorView, type BriefRecord, type DeskMute, type Initiative, type TodayCard } from "../../lib/client";
import { copy, dateLocale } from "../../lib/i18n";
import { briefDay, briefStorageKey, clockMinute, compareEditions, parseBriefPrefs, parseStoredBrief, quietBrief, serializeStoredBrief, slotsCrossed, BRIEF_PREFS_KEY, type BriefCard, type BriefLayout, type BriefPhrases, type BriefSlot, type MorningBrief } from "./work-model";

export const BRIEF_EVENT = "yoyo-brief";
/** DESK_EVENT 在读回新的开口判断或按住的类别时发出。 */
export const DESK_EVENT = "yoyo-desk";
const WELCOME_KEY = "yoyo.brief-welcomed";
const FEEDBACK_KEY = "yoyo.desk-feedback";

let sessionId = "";
let latest: Initiative | undefined;
let mutes: DeskMute[] = [];
/** seenMinute 是这一进程上次看见的本地分钟。-1 表示还没看过钟，第一次只记账，不补生成。 */
let seenMinute = -1;

/** latestInitiative 是最近一轮感知。任务栏用它显示会说但没说的事。 */
export function latestInitiative(): Initiative | undefined {
  return latest;
}

/** deskMutes 是最近读回的按住类别。 */
export function deskMutes(): DeskMute[] {
  return mutes;
}

/** refreshDesk 读回 Go 那边最近一轮开口判断。读不到时留着上一次的。 */
export async function refreshDesk(): Promise<void> {
  const state = await deskState().catch(() => null);
  if (!state) return;
  if (state.briefing) latest = state.briefing;
  mutes = state.mutes;
  window.dispatchEvent(new Event(DESK_EVENT));
}

export type FeedbackKind = "used" | "ignore" | "later" | "wrong" | "mute" | "unmute";

/** sendFeedback 把一次反应记到 className 这一类上。noteId 非空时记住这条短讯已经答过，重开也不再问。 */
export async function sendFeedback(className: string, kind: FeedbackKind, memoryId = "", noteId = ""): Promise<boolean> {
  if (!className.trim()) return false;
  const ok = await deskFeedback(className, kind, memoryId).catch(() => false);
  if (ok && noteId) rememberFeedback(noteId, kind);
  if (ok) void refreshDesk();
  return ok;
}

/** givenFeedback 是这条短讯上已经点过的反应。没点过时为空。 */
export function givenFeedback(noteId: string): FeedbackKind | "" {
  if (!noteId) return "";
  return (feedbackMap()[noteId] as FeedbackKind | undefined) || "";
}

function feedbackMap(): Record<string, string> {
  try {
    const body = JSON.parse(localStorage.getItem(FEEDBACK_KEY) || "{}") as Record<string, string>;
    return body && typeof body === "object" ? body : {};
  } catch {
    return {};
  }
}

function rememberFeedback(noteId: string, kind: FeedbackKind) {
  try {
    const body = feedbackMap();
    body[noteId] = kind;
    const keys = Object.keys(body);
    for (const key of keys.slice(0, Math.max(0, keys.length - 200))) delete body[key];
    localStorage.setItem(FEEDBACK_KEY, JSON.stringify(body));
  } catch {
    /* 记不住时，这一次界面上仍然收起。 */
  }
}

/** setDeskSession 记下当前主会话。定时推进写到这里。 */
export function setDeskSession(id: string) {
  sessionId = id;
}

/** deskSession 是最近一次记下的主会话。 */
export function deskSession(): string {
  return sessionId;
}

/** BRIEF_RETRY_MS 是跨过设定分钟之后还会再试的时长。模型那一次没写成，下一轮接着试，过了这段就不补了。 */
const BRIEF_RETRY_MS = 2 * 60 * 60 * 1000;
/** waitingSlots 是已经看见跨过设定分钟、还没写下的那几档。 */
const waitingSlots = new Map<BriefSlot, { day: string; until: number }>();
const busySlots = new Set<BriefSlot>();

/** runDeskFeed 读回开口判断。进程跑着并且时钟跨过设定分钟时，才自动生成那一档；没写成的在两小时内每一轮再试。
 *  打开应用时已经错过的那一档不补。force 是人点的那一档，立刻生成。已经写下的不再改，也不再推进。 */
export async function runDeskFeed(phrases: BriefPhrases, force?: BriefSlot): Promise<void> {
  const prefs = readPrefs();
  const main = await mainSession().catch(() => null);
  if (main?.id) sessionId = main.id;
  await orchestratorView().catch(() => null);
  await refreshDesk();
  if (force) {
    await ensureBrief(force);
    return;
  }
  const day = briefDay();
  for (const slot of crossedNow(prefs)) waitingSlots.set(slot, { day, until: Date.now() + BRIEF_RETRY_MS });
  for (const [slot, want] of [...waitingSlots]) {
    if (want.day !== day || Date.now() > want.until) {
      waitingSlots.delete(slot);
      continue;
    }
    if (await ensureBrief(slot)) waitingSlots.delete(slot);
  }
}

function crossedNow(prefs: ReturnType<typeof readPrefs>): BriefSlot[] {
  const minute = clockMinute();
  const prev = seenMinute;
  seenMinute = minute;
  return slotsCrossed(prefs, prev, minute);
}

/** BriefEdition 是看板上的一封已经固定的简报。 */
export type BriefEdition = {
  id: string;
  slot: BriefSlot;
  day: string;
  chip: string;
  at: string;
  brief: MorningBrief;
};

/** briefEditions 按时间从新到旧列出最近的简报。服务端有档案时用档案，否则只用本机当天留下的。 */
export async function briefEditions(): Promise<BriefEdition[]> {
  const remote = await briefList().catch(() => [] as BriefRecord[]);
  const rows = (remote.length ? remote.map(editionFromRecord) : localEditions()).filter((row): row is BriefEdition => !!row);
  return rows.sort(byNewest).slice(0, 12);
}

/** ensureBrief 让这一档有一封写下的信。写下了、或者早就写下了时返回真；模型还在写或这一次没写成时返回假，留给下一轮。
 *  确实没有值得写的事时写一封安静的信留档，不往主会话里放芯片。 */
async function ensureBrief(slot: BriefSlot): Promise<boolean> {
  if (busySlots.has(slot)) return false;
  busySlots.add(slot);
  try {
    const day = briefDay();
    const key = briefStorageKey(slot);
    const sealed = await sealedBrief(slot, day);
    if (sealed) {
      adoptSealed(slot, day, sealed);
      markSent(slot, day);
      return true;
    }
    const stored = readStored(key, day);
    if (stored) {
      if (!wasSent(slot, day)) await pushBrief(slot, day, stored, stored.cards.length === 1 && stored.cards[0].id === "quiet");
      return true;
    }
    const woven = await wovenBrief(slot);
    if (woven === "pending" || !woven) return false;
    if (woven === "quiet") {
      const quiet = quietLetter();
      writeBrief(key, day, quiet, false);
      await pushBrief(slot, day, quiet, true);
      return true;
    }
    writeBrief(key, day, woven, false);
    await pushBrief(slot, day, woven);
    return true;
  } finally {
    busySlots.delete(slot);
  }
}

/** quietLetter 是没有值得写的事时的那封信。邮箱和日历没连上时写明没有核对。 */
function quietLetter(): MorningBrief {
  const work = copy.work;
  const unchecked = (latest?.signals || [])
    .filter((signal) => (signal.code === "mail" || signal.code === "calendar") && signal.state === "unchecked")
    .map((signal) => (signal.code === "mail" ? work.senseMail : work.senseCalendar));
  return quietBrief({ kicker: work.briefEmptyKicker, title: work.auditQuietTitle, lede: work.auditQuietLede, unchecked: work.briefQuietUnchecked }, unchecked);
}

/** pushBrief 把刚写成的这一档放进主会话。同一天同一档只推一次。quiet 为真时只留档，不放芯片。 */
async function pushBrief(slot: BriefSlot, day: string, brief: MorningBrief, quiet = false): Promise<void> {
  if (!sessionId || wasSent(slot, day)) return;
  const work = copy.work;
  const chipDay = new Date(`${day}T00:00:00`).toLocaleDateString(dateLocale(), { month: "short", day: "numeric" });
  const chip = (slot === "evening" ? work.briefChipEvening : work.briefChipMorning).replace("{day}", chipDay);
  const text = quiet ? work.auditQuietTitle : slot === "evening" ? work.briefPushEvening : work.briefPushMorning;
  const pushed = await deskPush({
    sessionId,
    kind: "brief",
    slot,
    day,
    text,
    chip,
    brief: JSON.stringify(brief),
    quiet,
  }).catch(() => false);
  if (pushed) markSent(slot, day);
}

const SENT_KEY = "yoyo.brief-sent";

function sentMap(): Record<string, boolean> {
  try {
    const body = JSON.parse(localStorage.getItem(SENT_KEY) || "{}") as Record<string, boolean>;
    return body && typeof body === "object" ? body : {};
  } catch {
    return {};
  }
}

function wasSent(slot: BriefSlot, day: string): boolean {
  return sentMap()[`${slot}:${day}`] === true;
}

function markSent(slot: BriefSlot, day: string) {
  try {
    const body = sentMap();
    body[`${slot}:${day}`] = true;
    localStorage.setItem(SENT_KEY, JSON.stringify(body));
  } catch {
    /* 记不住时，下一次打开仍会问服务端这封在不在。 */
  }
}

async function sealedBrief(slot: BriefSlot, day: string): Promise<BriefRecord | null> {
  const rows = await briefList().catch(() => [] as BriefRecord[]);
  return rows.find((row) => row.slot === slot && row.day === day) || null;
}

function adoptSealed(slot: BriefSlot, day: string, rec: BriefRecord) {
  let body: MorningBrief;
  try {
    body = JSON.parse(rec.brief) as MorningBrief;
  } catch {
    return;
  }
  if (!body?.cards?.length) return;
  const prev = readStored(briefStorageKey(slot), day);
  const next: MorningBrief = { ...body, picked: prev?.picked || body.picked || {} };
  const raw = serializeStoredBrief(day, next);
  try {
    if (localStorage.getItem(briefStorageKey(slot)) === raw) return;
    localStorage.setItem(briefStorageKey(slot), raw);
  } catch {
    return;
  }
  window.dispatchEvent(new Event(BRIEF_EVENT));
}

function editionFromRecord(rec: BriefRecord): BriefEdition | null {
  try {
    const body = JSON.parse(rec.brief) as MorningBrief;
    if (!body?.cards?.length) return null;
    const slot: BriefSlot = rec.slot === "evening" ? "evening" : "morning";
    const kept = readStored(briefStorageKey(slot), rec.day);
    return {
      id: rec.id || `brief:${slot}:${rec.day}`,
      slot,
      day: rec.day,
      chip: rec.chip || "",
      at: rec.at || "",
      brief: { cards: body.cards, picked: kept?.picked || body.picked || {}, welcome: body.welcome === true, woven: body.woven === true },
    };
  } catch {
    return null;
  }
}

function localEditions(): BriefEdition[] {
  const day = briefDay();
  const out: BriefEdition[] = [];
  for (const slot of ["evening", "morning"] as BriefSlot[]) {
    const brief = readStored(briefStorageKey(slot), day);
    if (!brief) continue;
    out.push({ id: `brief:${slot}:${day}`, slot, day, chip: "", at: "", brief });
  }
  return out;
}

function byNewest(a: BriefEdition, b: BriefEdition): number {
  return compareEditions(a, b);
}

function writeBrief(key: string, day: string, brief: MorningBrief, welcome: boolean) {
  try {
    localStorage.setItem(key, serializeStoredBrief(day, brief));
    if (welcome) localStorage.setItem(WELCOME_KEY, "1");
  } catch {
    /* 写不进去时，这一次推进仍然带着这份简报。欢迎标记留到真正写下之后。 */
  }
  window.dispatchEvent(new Event(BRIEF_EVENT));
}

const briefLayouts: readonly BriefLayout[] = ["checks", "compare", "evidence", "chips", "plain", "letter", "rows", "stats"];

/** wovenBrief 是模型按材料写成的简报。还在写时是 pending，确实没有值得写的事时是 quiet，这一次没写成时为空。 */
async function wovenBrief(slot: BriefSlot): Promise<MorningBrief | "pending" | "quiet" | null> {
  const raw = await composeToday(slot).catch(() => null);
  if (raw?.pending) return "pending";
  if (raw?.quiet) return "quiet";
  if (!raw?.woven || !raw.cards?.length) return null;
  const cards = raw.cards.filter((card) => card.id && card.title).map(briefCard);
  if (!cards.length) return null;
  return { cards, picked: {}, woven: true };
}

function briefCard(card: TodayCard): BriefCard {
  const layout = briefLayouts.find((item) => item === card.layout);
  return {
    id: card.id,
    kicker: card.kicker || "",
    title: card.title,
    lede: card.lede || "",
    layout,
    items: (card.items || []).filter((item) => item.title).map((item) => ({
      title: item.title,
      detail: item.detail || "",
      tone: item.tone === "moving" || item.tone === "quiet" || item.tone === "stalled" ? item.tone : undefined,
    })),
    choices: (card.choices || []).filter((item) => item.title).map((item, index) => ({ id: item.id || String(index), title: item.title, detail: item.detail || "" })),
    stats: (card.stats || []).filter((stat) => stat.label && stat.value),
    mark: card.mark || undefined,
    leftLabel: card.leftLabel || undefined,
    rightLabel: card.rightLabel || undefined,
    note: card.note || undefined,
    primary: card.primary || "",
    secondary: card.secondary || "",
    task: card.task,
    alt: card.alt,
    refs: (card.refs || []).filter((ref) => ref.id && ref.label).map((ref) => ({ id: ref.id, kind: ref.kind, label: ref.label, target: ref.target || undefined })),
  };
}

function readPrefs() {
  try {
    return parseBriefPrefs(localStorage.getItem(BRIEF_PREFS_KEY));
  } catch {
    return parseBriefPrefs(null);
  }
}

function readStored(key: string, day: string): MorningBrief | null {
  try {
    return parseStoredBrief(localStorage.getItem(key), day);
  } catch {
    return null;
  }
}
