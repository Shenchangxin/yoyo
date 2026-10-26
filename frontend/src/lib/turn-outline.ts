import type { Item } from "./protocol";
import { peelCompletedMentions } from "./mentions";

/**
 * 拥有：会话目录里的操作者回合：标题、跳转别名、当前项和尚未入库的乐观条目。
 * 不拥有：目录的渲染，以及转录行的排版。
 */

/** OutlineTurn 会话目录里的一个操作者回合，由存储索引建成。 */
export type OutlineTurn = {
  seq: number;
  title: string;
  id: string;
  key: string;
  kind?: "user" | "steer";
  /** 尚未进入存储索引的乐观或实时包。 */
  live?: boolean;
};

/** OutlineStatus 目录上该回合的状态：空闲、进行中、等待或失败。 */
export type OutlineStatus = "idle" | "working" | "waiting" | "failed";

/** MIN_OUTLINE_TURNS 目录至少要有的回合数，少于此不值得单独展示。 */
export const MIN_OUTLINE_TURNS = 1;
/** OUTLINE_SEARCH_AT 回合多到此数时才出现目录搜索。 */
export const OUTLINE_SEARCH_AT = 8;
const TITLE_RUNES = 42;

/** outlineTitle 用第一行做目录标题。先剥掉已经完成的提及，超过字数上限时截断并加省略号。 */
export function outlineTitle(text: string): string {
  const peeled = peelCompletedMentions(text || "").text;
  const line = peeled.replace(/\s+/g, " ").trim();
  if (!line) return "";
  const chars = [...line];
  if (chars.length > TITLE_RUNES) return `${chars.slice(0, TITLE_RUNES).join("")}…`;
  return line;
}

/** itemTurnKey 条目的跳转键：优先载荷 id，其次序号，再次条目 key。 */
export function itemTurnKey(item: Item): string {
  const id = String(item.payload?.id || "").trim();
  if (id) return id;
  if (item.seq) return `seq:${item.seq}`;
  return item.key || "";
}

/** outlineTurnKey 目录项的跳转键，优先级与条目相同。 */
export function outlineTurnKey(t: Pick<OutlineTurn, "seq" | "id" | "key">): string {
  const id = (t.id || "").trim();
  if (id) return id;
  if (t.seq > 0) return `seq:${t.seq}`;
  return (t.key || "").trim();
}

/** turnJumpAliases 一个回合可能对上的全部跳转键，去重后按 id、序号、key 的顺序。 */
export function turnJumpAliases(t: Pick<OutlineTurn, "seq" | "id" | "key">): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  const add = (k: string) => {
    const v = (k || "").trim();
    if (!v || seen.has(v)) return;
    seen.add(v);
    out.push(v);
  };
  add(t.id || "");
  if (t.seq > 0) add(`seq:${t.seq}`);
  add(t.key || "");
  return out;
}

/** turnAliases 与 turnJumpAliases 相同。 */
export const turnAliases = turnJumpAliases;

/** rowMatchesJump 行上的键落在这组别名里。空键对不上。 */
export function rowMatchesJump(turnKey: string | undefined, aliases: string[]): boolean {
  return !!turnKey && aliases.includes(turnKey);
}

/** isOutlineActive 当前键命中该回合的任一别名。空键对不上，避免把所有未标键的刻度同时点亮。 */
export function isOutlineActive(turn: Pick<OutlineTurn, "seq" | "id" | "key">, activeKey: string): boolean {
  if (!activeKey) return false;
  return turnJumpAliases(turn).includes(activeKey);
}

/** outlineStatus 失败优先，其次等待，再次进行中，否则空闲。 */
export function outlineStatus(args: { running: boolean; waiting: boolean; failed?: boolean }): OutlineStatus {
  if (args.failed) return "failed";
  if (args.waiting) return "waiting";
  if (args.running) return "working";
  return "idle";
}

/**
 * pickActiveTurnKey 滚动窥视：取顶部已经越过阅读线的最后一个回合，只留一个胜者。
 * 重复或空键忽略。还没有任何回合越过阅读线时，取最靠上的一个。
 */
export function pickActiveTurnKey(nodes: { key: string; top: number }[], line: number): string {
  let best = "";
  let bestTop = -Infinity;
  for (const n of nodes) {
    const k = (n.key || "").trim();
    if (!k) continue;
    if (n.top <= line + 1 && n.top >= bestTop) {
      bestTop = n.top;
      best = k;
    }
  }
  if (best) return best;
  let first = "";
  let firstTop = Infinity;
  for (const n of nodes) {
    const k = (n.key || "").trim();
    if (!k) continue;
    if (n.top < firstTop) {
      firstTop = n.top;
      first = k;
    }
  }
  return first;
}

/** outlineOfRaw 从大小写两种字段名读出目录项，并补上跳转键。 */
export function outlineOfRaw(v: any): OutlineTurn {
  const seq = Number(v?.seq ?? v?.Seq) || 0;
  const title = String(v?.title ?? v?.Title ?? "");
  const id = String(v?.id ?? v?.Id ?? v?.ID ?? "");
  const turn = { seq, title, id, key: "" };
  turn.key = outlineTurnKey(turn);
  return turn;
}

/** pendingOutline 把存储索引里还没有的用户回合补到目录末尾。引导备注不算。标题与最后一项相同的乐观条目不重复追加。 */
export function pendingOutline(store: OutlineTurn[], items: Item[]): OutlineTurn[] {
  const extra: OutlineTurn[] = [];
  const last = store[store.length - 1];
  const have = new Set(store.flatMap((t) => turnJumpAliases(t)));
  for (const it of items) {
    if (it.type !== "user" || it.source === "steer") continue;
    const key = itemTurnKey(it);
    if (have.has(key) || have.has(it.key)) continue;
    const title = outlineTitle(it.text);
    if (last && title === last.title) continue;
    extra.push({
      seq: it.seq || 0,
      title,
      id: String(it.payload?.id || ""),
      key,
      live: it.source === "ui" || !it.seq,
    });
    have.add(key);
  }
  return extra.length ? [...store, ...extra] : store;
}

/** itemMatchesTurn 用户条目是否就是这个目录回合。引导备注对不上。id 都在时只比 id。 */
export function itemMatchesTurn(item: Item, turn: OutlineTurn): boolean {
  if (item.type !== "user" || item.source === "steer") return false;
  const id = String(item.payload?.id || "").trim();
  const turnId = (turn.id || "").trim();
  if (turnId && id) return turnId === id;
  if (turnJumpAliases(turn).includes(itemTurnKey(item))) return true;
  if (turn.seq && item.seq && item.seq === turn.seq) return true;
  return false;
}

/** neighborTurn 从当前高亮项按 delta 取相邻回合。没有当前项或越界时返回空。 */
export function neighborTurn(turns: OutlineTurn[], key: string, delta: number): OutlineTurn | null {
  if (!turns.length) return null;
  const i = turns.findIndex((t) => isOutlineActive(t, key));
  if (i < 0) return null;
  return turns[i + delta] || null;
}
