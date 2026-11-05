/**
 * 拥有：到点时把晨间、晚间简报和兴趣库变化推进主会话。
 * 不拥有：简报卡片怎么画。那在 WorkPane 和简报弹层。
 */

import { briefList, composeToday, deskFeedback, deskPush, mainSession, orchestratorView, poiAck, poiRefresh, type BriefRecord, type Initiative, type TodayCard } from "../../lib/client";
import { copy } from "../../lib/i18n";
import { briefDay, briefStorageKey, clockMinute, compareEditions, parseBriefPrefs, parseStoredBrief, serializeStoredBrief, slotsCrossed, BRIEF_PREFS_KEY, type BriefCard, type BriefLayout, type BriefPhrases, type BriefSlot, type DeskCensus, type MorningBrief } from "./work-model";

export const BRIEF_EVENT = "yoyo-brief";
const WELCOME_KEY = "yoyo.brief-welcomed";

let sessionId = "";
let latest: Initiative | undefined;
/** seenMinute 是这一进程上次看见的本地分钟。-1 表示还没看过钟，第一次只记账，不补生成。 */
let seenMinute = -1;

/** latestInitiative 是最近一轮感知。任务栏用它显示会说但没说的事。 */
export function latestInitiative(): Initiative | undefined {
  return latest;
}

/** noteFeedback 把反应记到这一轮开口的每一类上。 */
export async function noteFeedback(kind: "used" | "ignore" | "later" | "wrong"): Promise<void> {
  const signals = (latest?.signals || []).filter((signal) => signal.speak || signal.held);
  const classes = [...new Set(signals.map((signal) => signal.class).filter(Boolean))];
  if (!classes.length) classes.push("brief");
  for (const className of classes) {
    const memoryId = signals.find((signal) => signal.class === className)?.memoryId || "";
    await deskFeedback(className, kind, memoryId).catch(() => false);
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

/** runDeskFeed 刷新兴趣库。进程跑着并且时钟跨过设定分钟时，才自动生成那一档。force 是人点的那一档，立刻生成。已经写下的不再改，也不再推进。 */
export async function runDeskFeed(phrases: BriefPhrases, force?: BriefSlot): Promise<void> {
  const prefs = readPrefs();
  const main = await mainSession().catch(() => null);
  if (main?.id) sessionId = main.id;
  await orchestratorView().catch(() => null);
  const scan = await refreshPoi();
  const slots: BriefSlot[] = force ? [force] : crossedNow(prefs);
  for (const slot of slots) {
    await ensureBrief(slot, scan);
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

async function refreshPoi(): Promise<{ points: { name: string; kind: string }[]; census?: DeskCensus; initiative?: Initiative }> {
  const snap = await poiRefresh().catch(() => null);
  if (!snap) return { points: [] };
  const points = (snap.points || []).map((point) => ({ name: point.name, kind: point.kind })).filter((point) => point.name);
  const census = snap.census ? {
    ...snap.census,
    apps: points.filter((point) => point.kind === "app").length,
    appNames: points.filter((point) => point.kind === "app").map((point) => point.name).slice(0, 6),
    traces: points.filter((point) => point.kind === "trace" || point.kind === "desk").length,
  } : undefined;
  latest = snap.initiative;
  await pushSpeaks(snap.initiative, "", "reach");
  if (snap.pending && sessionId) {
    if (snap.initiative) {
      await poiAck(snap.sig).catch(() => false);
    } else {
      const text = poiText(snap.added || [], snap.dropped || []);
      if (text) {
        const pushed = await deskPush({
          sessionId,
          kind: "poi",
          slot: "poi",
          day: briefDay(),
          text,
          chip: "",
          brief: "",
        }).catch(() => false);
        if (pushed) await poiAck(snap.sig).catch(() => false);
      }
    }
  }
  return { points, census, initiative: snap.initiative };
}

function poiText(added: string[], dropped: string[]): string {
  const work = copy.work;
  const zh = /[\u4e00-\u9fff]/.test(work.poiAdded);
  const join = zh ? "、" : ", ";
  const lines: string[] = [work.poiPush];
  if (added.length) lines.push(`${work.poiAdded}${zh ? "：" : ": "}${added.slice(0, 8).join(join)}`);
  if (dropped.length) lines.push(`${work.poiDropped}${zh ? "：" : ": "}${dropped.slice(0, 8).join(join)}`);
  if (lines.length === 1) return "";
  return lines.join("\n");
}

async function ensureBrief(slot: BriefSlot, scan: { initiative?: Initiative }): Promise<void> {
  const day = briefDay();
  const key = briefStorageKey(slot);
  const sealed = await sealedBrief(slot, day);
  if (sealed) {
    adoptSealed(slot, day, sealed);
    markSent(slot, day);
    return;
  }
  const stored = readStored(key, day);
  if (stored) {
    if (!wasSent(slot, day)) await pushBrief(slot, day, stored);
    return;
  }
  const woven = await wovenBrief(slot);
  if (woven === "pending" || !woven) return;
  writeBrief(key, day, woven, false);
  await pushSpeaks(scan.initiative);
  await pushBrief(slot, day, woven);
}

/** pushBrief 把刚写成的这一档放进主会话。同一天同一档只推一次。 */
async function pushBrief(slot: BriefSlot, day: string, brief: MorningBrief): Promise<void> {
  if (!sessionId || wasSent(slot, day)) return;
  const work = copy.work;
  const chipDay = new Date(`${day}T00:00:00`).toLocaleDateString(undefined, { month: "short", day: "numeric" });
  const chip = (slot === "evening" ? work.briefChipEvening : work.briefChipMorning).replace("{day}", chipDay);
  const text = slot === "evening" ? work.briefPushEvening : work.briefPushMorning;
  const pushed = await deskPush({
    sessionId,
    kind: "brief",
    slot,
    day,
    text,
    chip,
    brief: JSON.stringify(brief),
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

/** pushSpeaks 把这一轮该说的短讯放进主会话。同一条同一天只留一次。检索按链接分开，并且不占用晨间那一条。 */
async function pushSpeaks(initiative: Initiative | undefined, briefJSON?: string, only?: string): Promise<void> {
  if (!sessionId || !initiative?.speak) return;
  const day = briefDay();
  const speaks = (initiative.signals || []).filter((signal) => {
    if (!signal.speak || signal.code === "welcome") return false;
    if (signal.class === "ticket" || signal.code === "ticket") return false;
    if (only) return signal.code === only;
    return signal.code !== "reach";
  });
  for (const signal of speaks) {
    const text = [signal.name, signal.detail].filter(Boolean).join("\n");
    if (!text) continue;
    await deskPush({
      sessionId,
      kind: "speak",
      slot: signal.code === "reach" ? signal.id : (signal.class || signal.id),
      day,
      text,
      chip: signal.name,
      brief: briefJSON || "",
    }).catch(() => false);
  }
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

/** wovenBrief 是模型按材料写成的简报。材料不够、还在写、或没有卡片时不落盘。 */
async function wovenBrief(slot: BriefSlot): Promise<MorningBrief | "pending" | null> {
  const raw = await composeToday(slot).catch(() => null);
  if (raw?.pending) return "pending";
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
