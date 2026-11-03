/**
 * 拥有：到点时把晨间、晚间简报和兴趣库变化推进主会话。
 * 不拥有：简报卡片怎么画。那在 WorkPane 和简报弹层。
 */

import { deskPush, memoryList, orchestratorView, poiAck, poiRefresh } from "../../lib/client";
import { copy } from "../../lib/i18n";
import { auditBrief, briefDay, briefStorageKey, parseBriefPrefs, parseStoredBrief, serializeStoredBrief, slotDue, BRIEF_PREFS_KEY, type BriefPhrases, type BriefSlot, type DeskCensus, type MorningBrief } from "./work-model";

export const BRIEF_EVENT = "yoyo-brief";
const WELCOME_KEY = "yoyo.brief-welcomed";

let sessionId = "";

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
  const scan = await refreshPoi();
  const slots: BriefSlot[] = force ? [force] : (["morning", "evening"] as const).filter((slot) => slotDue(prefs, slot));
  for (const slot of slots) {
    await ensureBrief(slot, phrases, scan);
  }
}

async function refreshPoi(): Promise<{ points: { name: string; kind: string }[]; census?: DeskCensus }> {
  const snap = await poiRefresh().catch(() => null);
  if (!snap) return { points: [] };
  const points = (snap.points || []).map((point) => ({ name: point.name, kind: point.kind })).filter((point) => point.name);
  const census = snap.census ? {
    ...snap.census,
    apps: points.filter((point) => point.kind === "app").length,
    traces: points.filter((point) => point.kind === "trace" || point.kind === "desk").length,
  } : undefined;
  if (snap.pending && sessionId) {
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
  return { points, census };
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

async function ensureBrief(slot: BriefSlot, phrases: BriefPhrases, scan: { points: { name: string; kind: string }[]; census?: DeskCensus }): Promise<void> {
  const day = briefDay();
  const key = briefStorageKey(slot);
  let brief = readStored(key, day);
  if (!brief) {
    const welcome = !welcomed();
    const view = await orchestratorView().catch(() => null);
    const memory = await memoryList().catch(() => []) as { kind: string; text: string; staging?: boolean }[];
    brief = auditBrief({
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
    try {
      localStorage.setItem(key, serializeStoredBrief(day, brief));
      if (brief.welcome) localStorage.setItem(WELCOME_KEY, "1");
    } catch {
      /* 写不进去时，这一次推进仍然带着这份简报。欢迎标记留到真正写下之后。 */
    }
    window.dispatchEvent(new Event(BRIEF_EVENT));
  }
  if (!sessionId) return;
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
