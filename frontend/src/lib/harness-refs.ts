import { num, pick, str } from "./normalize";
import type { HarnessTab } from "./protocol";

export type HarnessRefs = {
  active: string;
  staging: string;
  canary: string;
  head: string;
  extra: { name: string; hash: string }[];
};

export type HarborMetrics = {
  heldInPass: number;
  heldInTotal: number;
  heldOutPass: number;
  heldOutTotal: number;
  safetyFail: number;
};

export type HarnessNext = {
  tab: HarnessTab;
  action: "evolve" | "eval" | "checkout" | "blocked" | "idle";
};

const KNOWN = new Set(["active", "staging", "canary", "head", "HEAD"]);

export function shortHash(hash: string, n = 7): string {
  const s = str(hash).replace(/^(blake3:|sha256:)/i, "");
  if (!s) return "";
  return s.length > n ? s.slice(0, n) : s;
}

export function parseHarnessRefs(harness: unknown, fallbackActive = ""): HarnessRefs {
  const raw = (pick(harness, "refs", "Refs") || {}) as Record<string, unknown>;
  const active = firstHash(raw, ["active", "Active"]) || str(pick(harness, "active", "Active"), fallbackActive);
  const staging = firstHash(raw, ["staging", "Staging"]);
  const canary = firstHash(raw, ["canary", "Canary"]);
  const head = firstHash(raw, ["HEAD", "head", "Head"]);
  const extra: { name: string; hash: string }[] = [];
  for (const [k, v] of Object.entries(raw)) {
    if (KNOWN.has(k) || KNOWN.has(k.toLowerCase())) continue;
    const hash = str(v);
    if (hash) extra.push({ name: k, hash });
  }
  return { active, staging, canary, head, extra };
}

export function stagingDirty(refs: HarnessRefs): boolean {
  return !!refs.staging && refs.staging !== refs.active;
}

export function canaryDirty(refs: HarnessRefs): boolean {
  return !!refs.canary && refs.canary !== refs.active;
}

export function readHarborMetrics(report: unknown): HarborMetrics | null {
  if (report == null || typeof report !== "object") return null;
  const metrics = pick(report, "metrics", "Metrics") || {};
  return {
    heldInPass: num(pick(metrics, "held_in_pass", "HeldInPass")),
    heldInTotal: num(pick(metrics, "held_in_total", "HeldInTotal")),
    heldOutPass: num(pick(metrics, "held_out_pass", "HeldOutPass")),
    heldOutTotal: num(pick(metrics, "held_out_total", "HeldOutTotal")),
    safetyFail: num(pick(metrics, "safety_fail", "SafetyFail")),
  };
}

export function harnessNext(refs: HarnessRefs, report: unknown): HarnessNext {
  const metrics = readHarborMetrics(report);
  if (metrics && metrics.safetyFail > 0) return { tab: "prove", action: "blocked" };
  const staging = stagingDirty(refs);
  const canary = canaryDirty(refs);
  if (staging && metrics && gateClean(metrics)) return { tab: "promote", action: "checkout" };
  if (canary && metrics && gateClean(metrics)) return { tab: "promote", action: "checkout" };
  if (staging || canary) return { tab: "prove", action: "eval" };
  if (metrics) return { tab: "propose", action: "idle" };
  return { tab: "propose", action: "evolve" };
}

function gateClean(m: HarborMetrics): boolean {
  const inOk = m.heldInTotal > 0 && m.heldInPass === m.heldInTotal;
  const outOk = m.heldOutTotal === 0 || m.heldOutPass === m.heldOutTotal;
  return inOk && outOk && m.safetyFail === 0;
}

function firstHash(raw: Record<string, unknown>, keys: string[]): string {
  for (const k of keys) {
    const v = str(raw[k]);
    if (v) return v;
  }
  return "";
}
