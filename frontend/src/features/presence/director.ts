/** 拥有把工作台状态收成一个情绪编号的规则。一次只出一个编号。不拥有绘制，也不另开模型循环。只投影条目，不改服务端。 */
import type { Item } from "../../lib/protocol";
import type { VoicePhase } from "./types";

/** PresenceInput 是决定表情时用的工作台状态。 */
export type PresenceInput = {
  booted: boolean;
  running: boolean;
  items: Item[];
  approvals: number;
  interrupted?: boolean;
  moduleLoading?: boolean;
  windowFocused?: boolean;
  idleMs?: number;
  waking?: boolean;
  voicePhase?: VoicePhase;
  now?: number;
  /** 当前这一轮进行或加载已经持续了多久。传 0 时留在第一拍的表情。 */
  busyMs?: number;
};

/** PresenceOut 是一个情绪编号、可选提示，以及是否还在进行。 */
export type PresenceOut = {
  emotionId: string;
  tips?: string;
  live: boolean;
};

const RECEIVE_MS = 700;
const DONE_MS = 1200;
const PLEASED_MS = 5200;
const RECOVER_MS = 8000;
const STOP_MS = 900;
const SLEEP_MS = 180_000;
const WANDER_MS = 6200;

const WANDER = [
  "02", "10", "02", "03", "19", "02", "04", "11",
  "02", "14", "16", "02", "12", "18", "20", "02",
  "10", "19", "02", "03",
  "09", "24", "02", "25", "29", "02", "23", "08",
  "02", "22", "19", "11", "02", "27", "04", "48",
] as const;

/** wanderEmotion 在空闲时轮换张望和心情，再沉进瞌睡。now 除以 WANDER_MS，除以 WANDER_MS。idleMs 大于等于 SLEEP_MS，大于等于 130_000。 */
export function wanderEmotion(now: number, idleMs: number): string {
  if (idleMs >= SLEEP_MS) return "00";
  if (idleMs >= 130_000) return "06";
  if (idleMs >= 52_000) return "15";
  if (idleMs >= 24_000) {
    return Math.floor(now / WANDER_MS) % 3 === 0 ? "03" : "04";
  }
  return WANDER[Math.floor(now / WANDER_MS) % WANDER.length]!;
}

/** loadEmotion 在加载时从跳眨眼走到连接、搭建和等待。busyMs 小于 2200，小于 5200。 */
export function loadEmotion(busyMs: number): string {
  if (busyMs < 2200) return "36";
  if (busyMs < 5200) return "45";
  if (busyMs < 8600) return "43";
  const beat = Math.floor((busyMs - 8600) / 3800) % 3;
  return (["42", "44", "36"] as const)[beat]!;
}

/** thinkEmotion 在单纯思考时从绕圈走到专注、回想和耐心。busyMs 小于 3600，小于 7800。 */
export function thinkEmotion(busyMs: number): string {
  if (busyMs < 3600) return "30";
  if (busyMs < 7800) return "16";
  if (busyMs < 12_800) return "37";
  const beat = Math.floor((busyMs - 12_800) / 4200) % 3;
  return (["42", "30", "16"] as const)[beat]!;
}

/** afterDoneEmotion 按做完之后的时间给出收尾表情，过了就空。age 小于 0，小于 DONE_MS。 */
export function afterDoneEmotion(age: number): string | null {
  if (age < 0) return null;
  if (age < DONE_MS) return "33";
  if (age < DONE_MS + 1800) return "10";
  if (age < DONE_MS + 3600) return "19";
  if (age < DONE_MS + PLEASED_MS) return "23";
  return null;
}

/** voiceEmotion 按听、说、想给出通话表情。phase 与 listening、speaking、thinking 比较是否相等。busyMs 大于 5000，大于 4000，传给 thinkEmotion。 */
export function voiceEmotion(phase: VoicePhase | undefined, busyMs = 0): string {
  if (phase === "listening") return busyMs > 5000 ? "42" : "35";
  if (phase === "speaking") return busyMs > 4000 ? "46" : "39";
  if (phase === "thinking") return thinkEmotion(busyMs);
  return "02";
}

/** 取出工具名，没有就用 tool。item 读取 name、payload。 */
function toolName(item: Item): string {
  return String(item.name || item.payload?.name || "tool");
}

/** 从技能参数里取出要显示的名字。item 读取 payload，传给 toolName。 */
function skillTip(item: Item): string {
  const tagged = String(item.payload?.skill || "").trim();
  if (tagged) return tagged;
  const raw = item.payload?.arguments;
  let args: Record<string, unknown> = {};
  if (raw && typeof raw === "object" && !Array.isArray(raw)) args = raw as Record<string, unknown>;
  else if (typeof raw === "string" && raw.trim()) {
    try {
      const o = JSON.parse(raw) as unknown;
      if (o && typeof o === "object" && !Array.isArray(o)) args = o as Record<string, unknown>;
    } catch { /* 不是 JSON */ }
  }
  const name = toolName(item);
  if (name === "load_skill") return String(args.name || "").trim() || name;
  if (name === "read_skill_file" || name === "run_skill_script") {
    return String(args.skill || "").trim() || name;
  }
  return name;
}

/** 按工具名归成搜索、阅读、运行、编辑等种类。name 转成小写后，传给 test。 */
function toolKind(name: string): "search" | "web" | "read" | "list" | "run" | "edit" | "office" | "media" | "skill" | "other" {
  const n = name.toLowerCase();
  if (/(office_|connector_|computer_act|clipboard)/.test(n)) return "office";
  if (/(load_skill|read_skill|run_skill)/.test(n)) return "skill";
  if (/(image|video|ffmpeg|screenshot|canvas|draw|render)/.test(n)) return "media";
  if (/(search|grep|glob|find|query|lookup)/.test(n)) return "search";
  if (/(web|http|fetch|browser|url|crawl)/.test(n)) return "web";
  if (/(^|_)(read|view|cat|open)(_|$)/.test(n)) return "read";
  if (/(^|_)(list|ls|dir|tree)(_|$)/.test(n)) return "list";
  if (/(exec|bash|shell|cmd|command|terminal|run|git_)/.test(n)) return "run";
  if (/(write|edit|patch|create|delete|remove|rename|mkdir)/.test(n)) return "edit";
  return "other";
}

/** 把条目时间收成毫秒，没有就当 0。item 读取 ts，为空时不读。 */
function ts(item: Item | undefined): number {
  if (!item?.ts) return 0;
  const n = Date.parse(item.ts);
  return Number.isFinite(n) ? n : 0;
}

/** 从末尾找第一条符合的条目。items 读取 length，按下标读取。pred 被调用，并传入 items[i]!。 */
function lastOf(items: Item[], pred: (it: Item) => boolean): Item | undefined {
  for (let i = items.length - 1; i >= 0; i--) {
    if (pred(items[i]!)) return items[i];
  }
  return undefined;
}

/** 找还没有结果的那次工具调用。items 读取 length，按下标读取，用来遍历。 */
function openTool(items: Item[]): Item | undefined {
  const open = new Set<string>();
  for (const it of items) {
    const id = String(it.payload?.id || it.payload?.call_id || it.key);
    if (it.type === "tool_call") open.add(id);
    if (it.type === "tool_result") open.delete(id);
  }
  for (let i = items.length - 1; i >= 0; i--) {
    const it = items[i]!;
    if (it.type !== "tool_call") continue;
    const id = String(it.payload?.id || it.payload?.call_id || it.key);
    if (open.has(id)) return it;
  }
  return undefined;
}

/** 错误文案像是被拒绝或被挡住。item 为假时返回假。 */
function refused(item: Item | undefined): boolean {
  if (!item) return false;
  if (item.type === "error") {
    const t = `${item.text || ""} ${item.name || ""} ${JSON.stringify(item.payload || {})}`.toLowerCase();
    return /(refus|denied|blocked|restricted|not allowed|gate)/.test(t);
  }
  return false;
}

/** 按工具种类选一个干活时的表情。item 传给 toolName。 */
function emotionForTool(item: Item): string {
  const name = toolName(item).toLowerCase();
  if (name.includes("memory") || name.includes("trajectory")) return "37";
  const kind = toolKind(name);
  switch (kind) {
    case "search":
    case "web":
      return "40";
    case "read":
    case "list":
      return "16";
    case "run":
      return "32";
    case "edit":
      return "26";
    case "office":
      return "36";
    case "media":
      return "27";
    case "skill":
      return "03";
    default:
      return "32";
  }
}

/** 工具做久了就在原表情和等待脸之间换。id 作为分支结果，有的分支原样返回。age 大于 10_000，减去 10_000。 */
function sustainWork(id: string, age: number): string {
  if (!(age > 10_000)) return id;
  return Math.floor((age - 10_000) / 5600) % 2 === 0 ? id : "42";
}

/** presenceOf 把工作台状态收成一个情绪编号，一次只出一个。input 读取 now、items、voicePhase、busyMs、booted、approvals 等。 */
export function presenceOf(input: PresenceInput): PresenceOut {
  const now = input.now ?? Date.now();
  const items = input.items || [];
  const voice = input.voicePhase || "off";
  const busyMs = input.busyMs || 0;

  if (!input.booted) return { emotionId: "05", live: true, tips: "booting" };

  if (voice === "listening") return { emotionId: voiceEmotion(voice, busyMs), live: true, tips: "listening" };
  if (voice === "speaking") return { emotionId: voiceEmotion(voice, busyMs), live: true, tips: "speaking" };
  if (voice === "thinking") return { emotionId: voiceEmotion(voice, busyMs), live: true, tips: "thinking" };

  if (input.approvals > 0) return { emotionId: "35", live: true, tips: "approval" };

  const lastError = lastOf(items, (it) => it.type === "error");
  if (lastError && (input.running || now - ts(lastError) < 4000)) {
    if (refused(lastError)) return { emotionId: "38", live: true, tips: "refused" };
    return { emotionId: "34", live: true, tips: "error" };
  }
  if (lastError && !input.running && now - ts(lastError) < RECOVER_MS) {
    const age = now - ts(lastError);
    return { emotionId: age > 5600 ? "12" : "18", live: false, tips: "recover" };
  }

  if (input.moduleLoading && !input.running) {
    return { emotionId: loadEmotion(busyMs), live: true, tips: "loading" };
  }

  const leftover = openTool(items);
  if (!input.running && leftover) {
    const age = now - ts(leftover);
    if ((age >= 0 && age < STOP_MS) || input.interrupted) {
      return { emotionId: "41", live: false, tips: "stop" };
    }
  }

  if (input.running) {
    const lastUser = lastOf(items, (it) => it.type === "user" && it.source !== "steer");
    if (lastUser && now - ts(lastUser) < RECEIVE_MS) {
      return { emotionId: "31", live: true, tips: "receive" };
    }
    const streaming = lastOf(items, (it) => it.type === "assistant" && !!it.delta);
    if (streaming) {
      const age = now - ts(streaming);
      return { emotionId: age > 5200 ? "46" : "39", live: true, tips: "reply" };
    }
    const tool = openTool(items);
    if (tool) return { emotionId: sustainWork(emotionForTool(tool), now - ts(tool)), live: true, tips: skillTip(tool) };
    return { emotionId: thinkEmotion(busyMs), live: true, tips: "think" };
  }

  const lastEnd = lastOf(items, (it) => it.type === "turn_end");
  if (lastEnd && !input.interrupted) {
    const done = afterDoneEmotion(now - ts(lastEnd));
    if (done === "33") return { emotionId: "33", live: false, tips: "done" };
    if (done) return { emotionId: done, live: false, tips: "pleased" };
  }
  if (input.interrupted && lastEnd && now - ts(lastEnd) < STOP_MS) {
    return { emotionId: "17", live: false, tips: "stop" };
  }

  if (input.waking) return { emotionId: busyMs > 900 ? "08" : "01", live: false, tips: "wake" };

  const lastAsk = lastOf(items, (it) => it.type === "ask_user");
  if (lastAsk && now - ts(lastAsk) < 12_000) {
    return { emotionId: now - ts(lastAsk) > 5000 ? "42" : "35", live: true, tips: "ask" };
  }
  const lastEval = lastOf(items, (it) => it.type === "eval");
  if (lastEval && now - ts(lastEval) < 4000) {
    return { emotionId: "43", live: true, tips: "eval" };
  }
  const lastEvolve = lastOf(items, (it) => it.type === "evolve");
  if (lastEvolve && now - ts(lastEvolve) < 4000) {
    return { emotionId: "37", live: true, tips: "evolve" };
  }

  const idleMs = input.idleMs || 0;
  const id = wanderEmotion(now, idleMs);
  return { emotionId: id, live: false, tips: id === "00" ? "sleep" : "idle" };
}

const KNOWN_TIPS = new Set([
  "booting", "listening", "speaking", "thinking", "approval", "refused", "error",
  "recover", "loading", "stop", "receive", "reply", "think", "done", "pleased",
  "wake", "sleep", "idle", "eval", "evolve", "ask",
]);

/** presenceLine 把已知提示换成文案，活着的工具名接在工作前缀后面。out 读取 tips、live。labels 读取 working、idle，按下标读取。 */
export function presenceLine(out: PresenceOut, labels: Record<string, string>): string {
  const t = out.tips || "idle";
  if (labels[t]) return labels[t];
  if (out.live && t && !KNOWN_TIPS.has(t)) {
    const prefix = labels.working || "Working";
    return `${prefix} · ${t}`;
  }
  return labels.idle || t;
}
