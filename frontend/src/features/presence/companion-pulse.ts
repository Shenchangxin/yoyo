/** 拥有陪伴气泡的脉搏解析、合并和文案拆分。不拥有窗口，也不另开模型循环。只投影事件，不改服务端。 */
import type { Item } from "../../lib/protocol";

/** PULSE_MS 是一条普通脉搏保留的时长。 */
export const PULSE_MS = 7000;
/** LOADING_PULSE_MS 是加载脉搏保留的时长。 */
export const LOADING_PULSE_MS = 16_000;

/** CompanionPulse 是一条要显示的标题、正文、会话和截止时间。 */
export type CompanionPulse = {
  title: string;
  body: string;
  session: string;
  until: number;
  kind: string;
};

/** pulseFrom 从事件里读出脉搏，加载用更长的停留。raw 读取 data，为空时不读，为真时才继续，用 typeof 看类型，作为分支结果，作为后备值。now 加上 ttl。 */
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

/** itemStartsWork 把用户、工具、助手和推理当成开始干活，steer 不算。it 读取 type、source。 */
export function itemStartsWork(it: Pick<Item, "type" | "source">): boolean {
  if (it.type === "tool_call" || it.type === "assistant" || it.type === "reasoning") return true;
  return it.type === "user" && it.source !== "steer";
}

/** itemEndsWork 把回合结束和错误当成这段活结束。it 读取 type。 */
export function itemEndsWork(it: Pick<Item, "type">): boolean {
  return it.type === "turn_end" || it.type === "error";
}

/** mergePulse 用新脉搏替换旧的，加载盖不住还在的待批准。cur 读取 kind、until，为空时不读，为真时才继续，有的分支原样返回。next 读取 kind，展开传递。now 加上 PULSE_MS，大于 cur.until。 */
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

/** showLoading 在有待批准时不显示加载，否则看是否仍在跑或加载脉搏还在。running 为真时返回真。pulse 读取 kind、until，为假时走另一支。approvals 大于 0。now 大于 pulse.until。 */
export function showLoading(running: boolean, pulse: CompanionPulse | null, approvals: number, now = Date.now()): boolean {
  if (approvals > 0) return false;
  if (running) return true;
  return !!pulse && pulse.kind === "loading" && pulse.until > now;
}

/** bubbleParts 把待批准、脉搏或表情收成眉题和正文。opts 读取 approvals、approvalLine、pulse、now、emotionLine。 */
export function bubbleParts(opts: {
  approvals: number;
  pulse: CompanionPulse | null;
  approvalLine: string;
  emotionLine: string;
  now?: number;
}): { kicker: string; copy: string } {
  if (opts.approvals > 0) return { kicker: "", copy: opts.approvalLine };
  const pulse = opts.pulse && opts.pulse.until > (opts.now ?? Date.now()) ? opts.pulse : null;
  if (!pulse) return { kicker: "", copy: opts.emotionLine };
  const title = pulse.title.trim();
  const body = pulse.body.trim();
  if (body && body !== title) return { kicker: title, copy: body };
  return { kicker: "", copy: title };
}

/** bubbleLine 把眉题和正文收成一行。opts 传给 bubbleParts。 */
export function bubbleLine(opts: {
  approvals: number;
  pulse: CompanionPulse | null;
  approvalLine: string;
  emotionLine: string;
  now?: number;
}): string {
  const parts = bubbleParts(opts);
  if (parts.kicker && parts.copy) return `${parts.kicker} · ${parts.copy}`;
  return parts.copy || parts.kicker;
}
