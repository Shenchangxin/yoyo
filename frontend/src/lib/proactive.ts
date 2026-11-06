/**
 * 拥有：主动式节奏在前端的读写：主动程度三档落到开口门上，简报时间在配置和本机缓存之间同步。
 * 不拥有：开口门本身。门在 Go 的 initiative，这里只按住或放开几类开口；安静时段由 Go 在发系统通知前判断。
 */

import { deskFeedback, setDeskConfig } from "./client";
import type { DeskConfig } from "./protocol";

export type ProactiveLevel = "less" | "normal" | "more";

/** CHANGE_CLASSES 是「在做的事」和「这台电脑」有变化时开口的类别。少一点时按住它们。 */
const CHANGE_CLASSES = ["line", "git", "poi", "power", "net"];
/** REACH 是公开渠道里和你有关的新消息。多一点时放开，少一点时按住，适中不碰。 */
const REACH = "reach";

/** BRIEF_PREFS_KEY 和 work-model 里的同名常量一致：desk-feed 从这里读到点时间。 */
const BRIEF_PREFS_KEY = "yoyo.brief-prefs";
const DEFAULT_MORNING = "08:30";
const DEFAULT_EVENING = "21:30";

export function parseProactiveLevel(value: unknown): ProactiveLevel {
  return value === "less" || value === "more" ? value : "normal";
}

/** applyProactiveLevel 把一档落到开口门上。到期、等你和出错不受影响。返回是否全部写成功。 */
export async function applyProactiveLevel(level: ProactiveLevel): Promise<boolean> {
  const kind = level === "less" ? "mute" : "unmute";
  const results = await Promise.all(CHANGE_CLASSES.map((name) => deskFeedback(name, kind).catch(() => false)));
  if (level !== "normal") results.push(await deskFeedback(REACH, level === "less" ? "mute" : "unmute").catch(() => false));
  return results.every(Boolean);
}

export type BriefTimes = { morningOn: boolean; morningAt: string; eveningOn: boolean; eveningAt: string };

function slot(value: string | undefined, fallback: string): { on: boolean; at: string } {
  const v = (value || "").trim();
  if (v === "off") return { on: false, at: fallback };
  return { on: true, at: /^\d{1,2}:\d{2}$/.test(v) ? v.padStart(5, "0") : fallback };
}

/** briefTimesOf 从配置读出两档简报。没写过时用默认的 08:30 和 21:30。 */
export function briefTimesOf(desk: DeskConfig | undefined): BriefTimes {
  const morning = slot(desk?.briefMorning, DEFAULT_MORNING);
  const evening = slot(desk?.briefEvening, DEFAULT_EVENING);
  return { morningOn: morning.on, morningAt: morning.at, eveningOn: evening.on, eveningAt: evening.at };
}

/** briefDeskOf 把两档简报写回配置的写法：关掉写 off。 */
export function briefDeskOf(times: BriefTimes): Pick<DeskConfig, "briefMorning" | "briefEvening"> {
  return {
    briefMorning: times.morningOn ? times.morningAt : "off",
    briefEvening: times.eveningOn ? times.eveningAt : "off",
  };
}

/** BRIEF_PREFS_EVENT 在本机缓存的简报时间变了之后发出，今天栏据此重读。 */
export const BRIEF_PREFS_EVENT = "yoyo-brief-prefs";

/** cacheBriefTimes 把简报时间写进本机缓存，desk-feed 按它判断到点。 */
export function cacheBriefTimes(times: BriefTimes) {
  try {
    const next = JSON.stringify(times);
    if (localStorage.getItem(BRIEF_PREFS_KEY) === next) return;
    localStorage.setItem(BRIEF_PREFS_KEY, next);
  } catch {
    /* 写不进去时，这次打开里仍按配置走。 */
    return;
  }
  window.dispatchEvent(new Event(BRIEF_PREFS_EVENT));
}

/** cachedBriefTimes 读本机缓存里改过的简报时间。没写过或者和默认一样时为空，用来把旧版本的偏好搬进配置。 */
export function cachedBriefTimes(): BriefTimes | null {
  try {
    const body = JSON.parse(localStorage.getItem(BRIEF_PREFS_KEY) || "null") as Partial<BriefTimes> | null;
    if (!body || typeof body !== "object") return null;
    const base = briefTimesOf(undefined);
    const times: BriefTimes = {
      morningOn: typeof body.morningOn === "boolean" ? body.morningOn : base.morningOn,
      morningAt: slot(body.morningAt, base.morningAt).at,
      eveningOn: typeof body.eveningOn === "boolean" ? body.eveningOn : base.eveningOn,
      eveningAt: slot(body.eveningAt, base.eveningAt).at,
    };
    const same = times.morningOn === base.morningOn && times.morningAt === base.morningAt && times.eveningOn === base.eveningOn && times.eveningAt === base.eveningAt;
    return same ? null : times;
  } catch {
    return null;
  }
}

/** saveDesk 写配置里的主动式一块，并同步本机缓存的简报时间。 */
export async function saveDesk(desk: DeskConfig): Promise<void> {
  await setDeskConfig(desk);
  cacheBriefTimes(briefTimesOf(desk));
}
