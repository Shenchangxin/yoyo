/**
 * 拥有：从 video_brief 的工具结果里读出制作简报、账号对照和确认结果。
 * 不拥有：卡片的渲染，以及确认本身。确认在输入框上方的 hold 提问卡上完成。
 */
import type { Item } from "./protocol";
import { toolName } from "./tool-summary";

/** BriefConfirm 是确认卡的结果。空串表示这份简报不花额度或缺账号，没有出卡。 */
export type BriefConfirm = "go" | "edit" | "stop" | "unconfirmed" | "";

/** VideoBriefView 是一份简报和它对照出的账号。 */
export type VideoBriefView = {
  intent: string;
  format: string;
  subject: string;
  seconds: number;
  aspect: string;
  needs: string[];
  have: Record<string, string[]>;
  local: string[];
  missing: string[];
  spend: boolean;
  ready: boolean;
  confirmed: BriefConfirm;
};

/** isVideoBriefItem 判断这条是不是 video_brief 的调用或结果。item 为空时返回假。 */
export function isVideoBriefItem(item?: Item | null): boolean {
  return !!item && toolName(item) === "video_brief";
}

/** parseVideoBrief 读出结果开头的 JSON，以及末尾的 confirmed 行。JSON 可以是缩进的。解析失败或没有 intent 时返回空。body 是工具结果正文。 */
export function parseVideoBrief(body: string): VideoBriefView | null {
  const text = (body || "").trim();
  if (!text.startsWith("{")) return null;
  const cut = text.search(/\nconfirmed:/);
  let raw: Record<string, unknown>;
  try {
    raw = JSON.parse(cut < 0 ? text : text.slice(0, cut)) as Record<string, unknown>;
  } catch {
    return null;
  }
  const brief = (raw.brief && typeof raw.brief === "object" ? raw.brief : {}) as Record<string, unknown>;
  const intent = str(brief.intent);
  if (!intent) return null;
  const hit = text.match(/^confirmed:\s*(go|edit|stop|unconfirmed)\s*$/m);
  const have: Record<string, string[]> = {};
  if (raw.have && typeof raw.have === "object") {
    for (const [cap, names] of Object.entries(raw.have as Record<string, unknown>)) have[cap] = list(names);
  }
  return {
    intent,
    format: str(brief.format),
    subject: str(brief.subject),
    seconds: Number(brief.seconds) > 0 ? Number(brief.seconds) : 0,
    aspect: str(brief.aspect),
    needs: list(raw.needs),
    have,
    local: list(raw.local),
    missing: list(raw.missing),
    spend: raw.spend === true,
    ready: raw.ready === true,
    confirmed: hit ? (hit[1] as BriefConfirm) : "",
  };
}

/** 收成去掉两端空白的字符串。v 为空时用空串。 */
function str(v: unknown): string {
  return String(v ?? "").trim();
}

/** 收成非空字符串数组。v 不是数组时返回空数组。 */
function list(v: unknown): string[] {
  return Array.isArray(v) ? v.map(str).filter(Boolean) : [];
}