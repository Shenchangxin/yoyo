import type { Item } from "./protocol";
import { peelCompletedMentions } from "./mentions";

/** One operator turn in the session directory. Built from the store index. */
export type OutlineTurn = {
  seq: number;
  title: string;
  id: string;
  key: string;
};

export const MIN_OUTLINE_TURNS = 2;
export const OUTLINE_SEARCH_AT = 8;
const TITLE_RUNES = 42;

export function outlineTitle(text: string): string {
  const peeled = peelCompletedMentions(text || "").text;
  const line = peeled.replace(/\s+/g, " ").trim();
  if (!line) return "";
  const chars = [...line];
  if (chars.length > TITLE_RUNES) return `${chars.slice(0, TITLE_RUNES).join("")}…`;
  return line;
}

export function itemTurnKey(item: Item): string {
  const id = String(item.payload?.id || "").trim();
  if (id) return id;
  if (item.seq) return `seq:${item.seq}`;
  return item.key || "";
}

export function outlineTurnKey(t: Pick<OutlineTurn, "seq" | "id" | "key">): string {
  const id = (t.id || "").trim();
  if (id) return id;
  if (t.seq > 0) return `seq:${t.seq}`;
  return (t.key || "").trim();
}

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

export function rowMatchesJump(turnKey: string | undefined, aliases: string[]): boolean {
  return !!turnKey && aliases.includes(turnKey);
}

/** Empty keys never match — that used to light every unlabeled tick at once. */
export function isOutlineActive(turn: Pick<OutlineTurn, "seq" | "id" | "key">, activeKey: string): boolean {
  if (!activeKey) return false;
  return turnJumpAliases(turn).includes(activeKey);
}

/**
 * Notion/ChatGPT scroll-spy: the last turn whose top has crossed the read
 * line. One winner. Duplicate or empty keys are ignored.
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

export function outlineOfRaw(v: any): OutlineTurn {
  const seq = Number(v?.seq ?? v?.Seq) || 0;
  const title = String(v?.title ?? v?.Title ?? "");
  const id = String(v?.id ?? v?.Id ?? v?.ID ?? "");
  const turn = { seq, title, id, key: "" };
  turn.key = outlineTurnKey(turn);
  return turn;
}

export function pendingOutline(store: OutlineTurn[], items: Item[]): OutlineTurn[] {
  const extra: OutlineTurn[] = [];
  const last = store[store.length - 1];
  for (const it of items) {
    if (it.type !== "user" || it.source !== "ui") continue;
    const title = outlineTitle(it.text);
    if (last && title === last.title) continue;
    extra.push({
      seq: it.seq || 0,
      title,
      id: String(it.payload?.id || ""),
      key: itemTurnKey(it),
    });
  }
  return extra.length ? [...store, ...extra] : store;
}

export function itemMatchesTurn(item: Item, turn: OutlineTurn): boolean {
  if (item.type !== "user" || item.source === "steer") return false;
  const id = String(item.payload?.id || "").trim();
  const turnId = (turn.id || "").trim();
  if (turnId && id) return turnId === id;
  if (turnJumpAliases(turn).includes(itemTurnKey(item))) return true;
  if (turn.seq && item.seq && item.seq === turn.seq) return true;
  return false;
}

export function neighborTurn(turns: OutlineTurn[], key: string, delta: number): OutlineTurn | null {
  if (!turns.length) return null;
  const i = turns.findIndex((t) => outlineTurnKey(t) === key);
  const next = i < 0 ? (delta > 0 ? 0 : turns.length - 1) : i + delta;
  return turns[next] || null;
}
