import type { Item } from "../../lib/protocol";
import type { VoicePhase } from "./types";

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
};

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
  "02", "03", "10", "02", "11", "19", "03", "04",
  "16", "02", "14", "18", "10", "20", "03", "19",
] as const;

/** Ambient idle face. Cycles through glances, moods, then drowsy sleep. */
export function wanderEmotion(now: number, idleMs: number): string {
  if (idleMs >= SLEEP_MS) return "00";
  if (idleMs >= 130_000) return "06";
  if (idleMs >= 52_000) return "15";
  if (idleMs >= 24_000) {
    return Math.floor(now / WANDER_MS) % 3 === 0 ? "03" : "04";
  }
  return WANDER[Math.floor(now / WANDER_MS) % WANDER.length]!;
}

function toolName(item: Item): string {
  return String(item.name || item.payload?.name || "tool");
}

function toolKind(name: string): "search" | "web" | "read" | "list" | "run" | "edit" | "office" | "other" {
  const n = name.toLowerCase();
  if (/(office_|connector_|browser_|computer_act|clipboard)/.test(n)) return "office";
  if (/(search|grep|glob|find|query|lookup)/.test(n)) return "search";
  if (/(web|http|fetch|browser|url|crawl)/.test(n)) return "web";
  if (/(^|_)(read|view|cat|open)(_|$)/.test(n)) return "read";
  if (/(^|_)(list|ls|dir|tree)(_|$)/.test(n)) return "list";
  if (/(exec|bash|shell|cmd|command|terminal|run|git_)/.test(n)) return "run";
  if (/(write|edit|patch|create|delete|remove|rename|mkdir)/.test(n)) return "edit";
  return "other";
}

function ts(item: Item | undefined): number {
  if (!item?.ts) return 0;
  const n = Date.parse(item.ts);
  return Number.isFinite(n) ? n : 0;
}

function lastOf(items: Item[], pred: (it: Item) => boolean): Item | undefined {
  for (let i = items.length - 1; i >= 0; i--) {
    if (pred(items[i]!)) return items[i];
  }
  return undefined;
}

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

function refused(item: Item | undefined): boolean {
  if (!item) return false;
  if (item.type === "error") {
    const t = `${item.text || ""} ${item.name || ""} ${JSON.stringify(item.payload || {})}`.toLowerCase();
    return /(refus|denied|blocked|restricted|not allowed|gate)/.test(t);
  }
  return false;
}

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
    case "edit":
      return "32";
    case "office":
      return "36";
    default:
      return "32";
  }
}

/** Map workstation state onto a Mood Mates emotionId. One id at a time. */
export function presenceOf(input: PresenceInput): PresenceOut {
  const now = input.now ?? Date.now();
  const items = input.items || [];
  const voice = input.voicePhase || "off";

  if (!input.booted) return { emotionId: "05", live: true, tips: "booting" };

  if (voice === "listening") return { emotionId: "35", live: true, tips: "listening" };
  if (voice === "speaking") return { emotionId: "39", live: true, tips: "speaking" };
  if (voice === "thinking") return { emotionId: "30", live: true, tips: "thinking" };

  if (input.approvals > 0) return { emotionId: "35", live: true, tips: "approval" };

  const lastError = lastOf(items, (it) => it.type === "error");
  if (lastError && (input.running || now - ts(lastError) < 4000)) {
    if (refused(lastError)) return { emotionId: "38", live: true, tips: "refused" };
    return { emotionId: "34", live: true, tips: "error" };
  }
  if (lastError && !input.running && now - ts(lastError) < RECOVER_MS) {
    return { emotionId: "18", live: false, tips: "recover" };
  }

  if (input.moduleLoading && !input.running) {
    return { emotionId: "36", live: true, tips: "loading" };
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
    if (streaming) return { emotionId: "39", live: true, tips: "reply" };
    const tool = openTool(items);
    if (tool) return { emotionId: emotionForTool(tool), live: true, tips: toolName(tool) };
    return { emotionId: "30", live: true, tips: "think" };
  }

  const lastEnd = lastOf(items, (it) => it.type === "turn_end");
  if (lastEnd && now - ts(lastEnd) < DONE_MS && !input.interrupted) {
    return { emotionId: "33", live: false, tips: "done" };
  }
  if (lastEnd && now - ts(lastEnd) < DONE_MS + PLEASED_MS && !input.interrupted) {
    return { emotionId: "10", live: false, tips: "pleased" };
  }
  if (input.interrupted && lastEnd && now - ts(lastEnd) < STOP_MS) {
    return { emotionId: "41", live: false, tips: "stop" };
  }

  if (input.waking) return { emotionId: "01", live: false, tips: "wake" };

  const lastAsk = lastOf(items, (it) => it.type === "ask_user");
  if (lastAsk && now - ts(lastAsk) < 12_000) {
    return { emotionId: "35", live: true, tips: "ask" };
  }
  const lastEval = lastOf(items, (it) => it.type === "eval");
  if (lastEval && now - ts(lastEval) < 4000) {
    return { emotionId: "36", live: true, tips: "eval" };
  }
  const lastEvolve = lastOf(items, (it) => it.type === "evolve");
  if (lastEvolve && now - ts(lastEvolve) < 4000) {
    return { emotionId: "30", live: true, tips: "evolve" };
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

export function presenceLine(out: PresenceOut, labels: Record<string, string>): string {
  const t = out.tips || "idle";
  if (labels[t]) return labels[t];
  if (out.live && t && !KNOWN_TIPS.has(t)) {
    const prefix = labels.working || "Working";
    return `${prefix} · ${t}`;
  }
  return labels.idle || t;
}
