/**
 * 拥有：到点时把晨间、晚间简报和兴趣库变化推进主会话。
 * 不拥有：简报卡片怎么画。那在 WorkPane 和简报弹层。
 */

import { composeToday, deskFeedback, deskPush, memoryList, orchestratorView, poiAck, poiRefresh, type Initiative } from "../../lib/client";
import { copy } from "../../lib/i18n";
import { applySenses, auditBrief, briefDay, briefStorageKey, parseBriefPrefs, parseStoredBrief, serializeStoredBrief, shouldPush, visibleSlot, BRIEF_PREFS_KEY, type BriefCard, type BriefPhrases, type BriefSlot, type DeskCensus, type MorningBrief, type SenseLabels, type SenseSignal } from "./work-model";

export const BRIEF_EVENT = "yoyo-brief";
const WELCOME_KEY = "yoyo.brief-welcomed";

let sessionId = "";
let latest: Initiative | undefined;

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

/** runDeskFeed 刷新兴趣库，并在到点时生成还没有的那一档简报。force 时立刻生成这一档。 */
export async function runDeskFeed(phrases: BriefPhrases, force?: BriefSlot): Promise<void> {
  const prefs = readPrefs();
  await orchestratorView().catch(() => null);
  const scan = await refreshPoi();
  const slots: BriefSlot[] = force ? [force] : [visibleSlot(prefs)];
  for (const slot of slots) {
    await ensureBrief(slot, phrases, scan);
  }
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

async function ensureBrief(slot: BriefSlot, phrases: BriefPhrases, scan: { points: { name: string; kind: string }[]; census?: DeskCensus; initiative?: Initiative }): Promise<void> {
  const day = briefDay();
  const key = briefStorageKey(slot);
  const stored = readStored(key, day);
  const welcome = !welcomed();
  if (stored) {
    await pushBrief(slot, day, stored, scan.initiative, welcome);
    return;
  }
  const view = await orchestratorView().catch(() => null);
  const memory = await memoryList().catch(() => []) as { kind: string; text: string; staging?: boolean }[];
  let brief = auditBrief({
    tickets: view?.tickets,
    queued: view?.queued,
    decisions: view?.decisions,
    rules: view?.rules,
    sources: view?.sources,
    memory,
    points: scan.points,
    census: scan.census,
    welcome,
    now: new Date(),
  }, phrases);
  const signals = (scan.initiative?.signals || []) as SenseSignal[];
  if (signals.length) brief = applySenses(brief, signals, senseLabels());
  const woven = await wovenBrief();
  if (woven === "pending") return;
  if (woven) brief = { ...woven, picked: brief.picked, woven: true };
  writeBrief(key, day, brief, welcome);
  await pushSpeaks(scan.initiative);
  await pushBrief(slot, day, brief, scan.initiative, welcome);
}

/** pushBrief 把已经生成的这一档放进主会话。同一天同一档只留一次，不再改简报正文。 */
async function pushBrief(slot: BriefSlot, day: string, brief: MorningBrief, initiative: Initiative | undefined, welcome: boolean): Promise<void> {
  if (!sessionId || !shouldPush(initiative, welcome)) return;
  const work = copy.work;
  const chipDay = new Date(`${day}T00:00:00`).toLocaleDateString(undefined, { month: "short", day: "numeric" });
  const chip = (slot === "evening" ? work.briefChipEvening : work.briefChipMorning).replace("{day}", chipDay);
  const text = slot === "evening" ? work.briefPushEvening : work.briefPushMorning;
  await deskPush({
    sessionId,
    kind: "brief",
    slot,
    day,
    text,
    chip,
    brief: JSON.stringify(brief),
  }).catch(() => false);
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

function senseLabels(): SenseLabels {
  const work = copy.work;
  return {
    unchecked: work.senseUnchecked,
    linked: work.senseLinked,
    senseTitle: work.senseTitle,
    senseLede: work.senseLede,
    listening: work.senseListening,
    battery: work.senseBattery,
    charging: work.senseCharging,
    online: work.senseOnline,
    offline: work.senseOffline,
    idle: work.senseIdle,
    focus: work.senseFocus,
    power: work.sensePower,
    net: work.senseNet,
    git: work.senseGit,
    mail: work.senseMail,
    calendar: work.senseCalendar,
    poi: work.sensePoi,
    speakKicker: work.speakKicker,
    speakTitle: work.speakTitle,
    speakLede: work.speakLede,
    speakPrimary: work.speakPrimary,
    speakTask: work.speakTask,
    why: {
      changed: work.whyChanged,
      overdue: work.whyOverdue,
      due: work.whyDue,
      waiting: work.whyWaiting,
      blocked: work.whyBlocked,
      failed: work.whyFailed,
      error: work.whyError,
      low: work.whyLow,
      offline: work.whyOffline,
      welcome: work.whyWelcome,
      match: work.whyMatch,
    },
  };
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

/** wovenBrief 是按画像、记忆和最近对话写成的简报。材料不够或模型没写出卡片时为空。上一次还在写时返回 pending。 */
async function wovenBrief(): Promise<MorningBrief | "pending" | null> {
  const raw = await composeToday().catch(() => null);
  if (raw?.pending) return "pending";
  if (!raw?.woven || !raw.cards?.length) return null;
  const cards = raw.cards.filter((card) => card.id && card.title).map((card): BriefCard => ({
    id: card.id,
    kicker: card.kicker || "",
    title: card.title,
    lede: card.lede || "",
    layout: card.layout === "rows" || card.layout === "checks" || card.layout === "compare" || card.layout === "chips" || card.layout === "plain" || card.layout === "evidence" ? card.layout : "evidence",
    items: (card.items || []).map((item) => ({ title: item.title, detail: item.detail || "" })),
    choices: (card.choices || []).map((item, index) => ({ id: item.id || String(index), title: item.title, detail: item.detail || "" })),
    primary: card.primary || "",
    secondary: card.secondary || "",
    task: card.task,
    alt: card.alt,
  }));
  if (!cards.length) return null;
  return { cards, picked: {} };
}

function readPrefs() {
  try {
    return parseBriefPrefs(localStorage.getItem(BRIEF_PREFS_KEY));
  } catch {
    return parseBriefPrefs(null);
  }
}

function welcomed(): boolean {
  try {
    return localStorage.getItem(WELCOME_KEY) === "1";
  } catch {
    return false;
  }
}

function readStored(key: string, day: string): MorningBrief | null {
  try {
    return parseStoredBrief(localStorage.getItem(key), day);
  } catch {
    return null;
  }
}
