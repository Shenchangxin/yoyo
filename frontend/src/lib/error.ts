/**
 * 拥有：把运行时错误收成种类、是否可重试，以及横幅和短标题用的文案。
 * 不拥有：错误卡片的渲染。
 */

import { str } from "./normalize";
import type { Item } from "./protocol";

/** ErrorKind 溢出、认证、限流、非法请求、超时、取消、供应商、预算、回合上限、卡住、计划未完成、模型或未知。 */
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
  | "stuck"
  | "plan_open"
  | "model"
  | "unknown";

/** ErrorCopy 每种错误的标题和提示。 */
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
  errorStuck: string;
  errorStuckHint: string;
  errorPlanOpen: string;
  errorPlanOpenHint: string;
  errorModel: string;
  errorModelHint: string;
  errorUnknown: string;
  errorUnknownHint: string;
};

/** ClassifiedError 分类结果。payloadTitle 与 payloadHint 来自载荷。detail 是具体原因。 */
export type ClassifiedError = {
  kind: ErrorKind;
  retryable: boolean;
  detail: string;
  payloadTitle: string;
  payloadHint: string;
};

const KINDS: ErrorKind[] = [
  "overflow", "auth", "rate_limit", "invalid", "timeout", "canceled", "provider", "budget", "max_turns", "stuck", "plan_open", "model", "unknown",
];

/** errorCopy 按种类取标题和提示。对不上时用未知。t 读取 errorOverflow、errorOverflowHint、errorAuth、errorAuthHint、errorRateLimit、errorRateLimitHint 等。kind 用来做分支。 */
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
    case "stuck": return { title: t.errorStuck, hint: t.errorStuckHint };
    case "plan_open": return { title: t.errorPlanOpen, hint: t.errorPlanOpenHint };
    case "model": return { title: t.errorModel, hint: t.errorModelHint };
    default: return { title: t.errorUnknown, hint: t.errorUnknownHint };
  }
}

/**
 * classifyError 载荷里的 kind 属于已知种类则用它，否则从正文、detail 和 error 推断。retryable 显式为假，或种类是认证、预算时，不可重试。
 * raw 加上  ，作为后备值，传给 extractProviderMsg。
 * payload 读取 kind、detail、error、retryable、title、hint，为空时不读。
 */
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

/** classifyItem 用条目正文和载荷做分类。item 读取 text、payload。 */
export function classifyItem(item: Item): ClassifiedError {
  return classifyError(item.text, item.payload);
}

/** 按正文里的特征短语推断种类。对不上则为未知。s 转成小写后，传给 test。 */
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
  if (/(same tool failure kept repeating|repeating loop|observation did not change|same check kept failing)/.test(low)) return "stuck";
  if (/(all_channel_models_failed|internal server error|502|503|504|overloaded)/.test(low)) return "provider";
  return "unknown";
}

/** 从正文里第一段 JSON 取出 msg、message、error 或 error.message。解析失败则保留原文。s 查找位置，调用 lastIndexOf，截取，有的分支原样返回。 */
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
    /* 保留原文 */
  }
  return s;
}

/** shortError 分类后的标题，不含具体原因。item 传给 classifyItem。t 传给 errorCopy。 */
export function shortError(item: Item, t: ErrorCopy): string {
  return errorCopy(t, classifyItem(item).kind).title;
}

/** 去掉两端空白。像整页 HTML 时改成一句说明。超过 280 字则截到 277 再加省略号。detail 去掉首尾空白后，读取 length，截取，为假时走另一支，作为分支结果，传给 test。 */
function clipDetail(detail: string): string {
  const t = detail.trim();
  if (!t) return "";
  if (/^<!doctype html/i.test(t) || /^<html[\s>]/i.test(t)) {
    return "backend returned HTML instead of an API response";
  }
  return t.length > 280 ? `${t.slice(0, 277)}…` : t;
}

/** bannerError 标题加上真实原因。横幅以前只显示笼统标题。没有原因、原因与标题相同，或原因已以标题开头时，不再重复标题。raw 作为后备值，传给 classifyError。t 传给 errorCopy。payload 传给 classifyError。 */
export function bannerError(raw: string, t: ErrorCopy, payload?: Record<string, any>): string {
  const classified = classifyError(raw, payload);
  const { title } = errorCopy(t, classified.kind);
  const detail = clipDetail(classified.detail || raw);
  if (!detail) return title;
  if (detail === title || detail.toLowerCase() === title.toLowerCase()) return title;
  if (title && detail.toLowerCase().startsWith(title.toLowerCase())) return detail;
  return `${title} · ${detail}`;
}
