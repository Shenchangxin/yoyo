import type { Item } from "../../lib/protocol";

export const PULSE_MS = 7000;
export const LOADING_PULSE_MS = 16_000;

export type CompanionPulse = {
  title: string;
  body: string;
  session: string;
  until: number;
  kind: string;
};

export function pulseFrom(raw: any, now = Date.now()): CompanionPulse | null {
  const p = raw && typeof raw === "object" && "title" in raw ? raw : raw?.data || raw;
  if (!p || typeof p !== "object") return null;
  const title = String(p.title || "").trim();
  if (!title) return null;
  const kind = String(p.kind || "").trim();
  const ttl = kind === "loading" ? LOADING_PULSE_MS : PULSE_MS;
  return {
    title,
    body: String(p.body || "").trim(),
    session: String(p.session || "").trim(),
    kind,
    until: now + ttl,
  };
}

export function itemStartsWork(it: Pick<Item, "type" | "source">): boolean {
  if (it.type === "tool_call" || it.type === "assistant" || it.type === "reasoning") return true;
  return it.type === "user" && it.source !== "steer";
}

export function itemEndsWork(it: Pick<Item, "type">): boolean {
  return it.type === "turn_end" || it.type === "error";
}

export function mergePulse(cur: CompanionPulse | null, next: CompanionPulse, now = Date.now()): CompanionPulse {
  if (next.kind === "done" || next.kind === "error") {
    if (cur?.kind === "loading") return { ...next, until: now + PULSE_MS };
    return next;
  }
  if (next.kind === "loading" && cur && cur.until > now && cur.kind === "approval") {
    return cur;
  }
  return next;
}

export function showLoading(running: boolean, pulse: CompanionPulse | null, approvals: number, now = Date.now()): boolean {
  if (approvals > 0) return false;
  if (running) return true;
  return !!pulse && pulse.kind === "loading" && pulse.until > now;
}

export function bubbleLine(opts: {
  approvals: number;
  pulse: CompanionPulse | null;
  approvalLine: string;
  emotionLine: string;
  now?: number;
}): string {
  if (opts.approvals > 0) return opts.approvalLine;
  const pulse = opts.pulse && opts.pulse.until > (opts.now ?? Date.now()) ? opts.pulse : null;
  if (pulse) return pulse.body ? `${pulse.title} · ${pulse.body}` : pulse.title;
  return opts.emotionLine;
}
