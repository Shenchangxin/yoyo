import { str } from "./normalize";
import type { Item } from "./protocol";

export type ErrorKind =
  | "overflow"
  | "auth"
  | "rate_limit"
  | "invalid"
  | "timeout"
  | "canceled"
  | "provider"
  | "budget"
  | "max_turns"
  | "unknown";

export type ErrorCopy = {
  errorOverflow: string;
  errorOverflowHint: string;
  errorAuth: string;
  errorAuthHint: string;
  errorRateLimit: string;
  errorRateLimitHint: string;
  errorInvalid: string;
  errorInvalidHint: string;
  errorTimeout: string;
  errorTimeoutHint: string;
  errorCanceled: string;
  errorCanceledHint: string;
  errorProvider: string;
  errorProviderHint: string;
  errorBudget: string;
  errorBudgetHint: string;
  errorMaxTurns: string;
  errorMaxTurnsHint: string;
  errorUnknown: string;
  errorUnknownHint: string;
};

export type ClassifiedError = {
  kind: ErrorKind;
  retryable: boolean;
  detail: string;
  payloadTitle: string;
  payloadHint: string;
};

const KINDS: ErrorKind[] = [
  "overflow", "auth", "rate_limit", "invalid", "timeout", "canceled", "provider", "budget", "max_turns", "unknown",
];

export function errorCopy(t: ErrorCopy, kind: ErrorKind): { title: string; hint: string } {
  switch (kind) {
    case "overflow": return { title: t.errorOverflow, hint: t.errorOverflowHint };
    case "auth": return { title: t.errorAuth, hint: t.errorAuthHint };
    case "rate_limit": return { title: t.errorRateLimit, hint: t.errorRateLimitHint };
    case "invalid": return { title: t.errorInvalid, hint: t.errorInvalidHint };
    case "timeout": return { title: t.errorTimeout, hint: t.errorTimeoutHint };
    case "canceled": return { title: t.errorCanceled, hint: t.errorCanceledHint };
    case "provider": return { title: t.errorProvider, hint: t.errorProviderHint };
    case "budget": return { title: t.errorBudget, hint: t.errorBudgetHint };
    case "max_turns": return { title: t.errorMaxTurns, hint: t.errorMaxTurnsHint };
    default: return { title: t.errorUnknown, hint: t.errorUnknownHint };
  }
}

export function classifyError(raw: string, payload?: Record<string, any>): ClassifiedError {
  const kindRaw = str(payload?.kind).toLowerCase();
  const kind = (KINDS as string[]).includes(kindRaw) ? (kindRaw as ErrorKind) : inferKind(raw + " " + str(payload?.detail) + " " + str(payload?.error));
  const detail = str(payload?.detail) || extractProviderMsg(raw) || raw;
  return {
    kind,
    retryable: payload?.retryable === false ? false : kind !== "auth" && kind !== "budget",
    detail,
    payloadTitle: str(payload?.title) || str(payload?.error),
    payloadHint: str(payload?.hint),
  };
}

export function classifyItem(item: Item): ClassifiedError {
  return classifyError(item.text, item.payload);
}

function inferKind(s: string): ErrorKind {
  const low = s.toLowerCase();
  if (/(context_length|too many tokens|prompt is too long|maximum context)/.test(low)) return "overflow";
  if (/(invalid_api_key|unauthorized|incorrect api key|401|authentication)/.test(low)) return "auth";
  if (/(rate limit|rate_limit|too many requests|429)/.test(low)) return "rate_limit";
  if (/(tool_calls|invalid_parameter|invalid_request|unrecognized request argument)/.test(low)) return "invalid";
  if (/(timeout|deadline exceeded|i\/o timeout)/.test(low)) return "timeout";
  if (/(canceled|interrupted|context canceled)/.test(low)) return "canceled";
  if (/(budget exceeded)/.test(low)) return "budget";
  if (/(max turns reached)/.test(low)) return "max_turns";
  if (/(all_channel_models_failed|internal server error|502|503|504|overloaded)/.test(low)) return "provider";
  return "unknown";
}

function extractProviderMsg(s: string): string {
  const i = s.indexOf("{");
  if (i < 0) return s;
  const j = s.lastIndexOf("}");
  if (j <= i) return s;
  try {
    const m = JSON.parse(s.slice(i, j + 1));
    if (typeof m?.msg === "string" && m.msg.trim()) return m.msg;
    if (typeof m?.message === "string" && m.message.trim()) return m.message;
    if (typeof m?.error === "string" && m.error.trim()) return m.error;
    if (m?.error && typeof m.error.message === "string") return m.error.message;
  } catch {
    /* keep raw */
  }
  return s;
}

export function shortError(item: Item, t: ErrorCopy): string {
  return errorCopy(t, classifyItem(item).kind).title;
}

function clipDetail(detail: string): string {
  const t = detail.trim();
  if (!t) return "";
  if (/^<!doctype html/i.test(t) || /^<html[\s>]/i.test(t)) {
    return "backend returned HTML instead of an API response";
  }
  return t.length > 280 ? `${t.slice(0, 277)}…` : t;
}

/** Title plus the real reason — the banner used to show only the generic title. */
export function bannerError(raw: string, t: ErrorCopy, payload?: Record<string, any>): string {
  const classified = classifyError(raw, payload);
  const { title } = errorCopy(t, classified.kind);
  const detail = clipDetail(classified.detail || raw);
  if (!detail) return title;
  if (detail === title || detail.toLowerCase() === title.toLowerCase()) return title;
  if (title && detail.toLowerCase().startsWith(title.toLowerCase())) return detail;
  return `${title} · ${detail}`;
}
