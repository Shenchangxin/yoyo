/**
 * 拥有：主会话读到哪里。按会话编号写在本地，应用关掉再开还在。
 * 不拥有：滚动本身。转录把当前位置交过来，下次挂上时再读回去。
 */

const KEY = "yoyo.session.place.v1";

/** SessionPlace 是一次离开时记下的阅读位置。following 为真表示人停在最新。 */
export type SessionPlace = {
  sessionId: string;
  following: boolean;
  anchor: string;
  seq: number;
  aliases: string[];
};

type PlaceStore = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
};

/** memoryPlaceStore 给测试用的内存仓库。 */
export function memoryPlaceStore(): PlaceStore {
  const m = new Map<string, string>();
  return {
    getItem: (key) => m.get(key) ?? null,
    setItem: (key, value) => { m.set(key, value); },
  };
}

function storeOf(store?: PlaceStore): PlaceStore | null {
  if (store) return store;
  try {
    return typeof localStorage === "undefined" ? null : localStorage;
  } catch {
    return null;
  }
}

function readAll(store: PlaceStore): Record<string, SessionPlace> {
  try {
    const raw = store.getItem(KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, SessionPlace>;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

/** readSessionPlace 读这个会话上次的位置。没有或坏了就返回空。 */
export function readSessionPlace(sessionId: string, store?: PlaceStore): SessionPlace | null {
  const id = sessionId.trim();
  if (!id) return null;
  const box = storeOf(store);
  if (!box) return null;
  const row = readAll(box)[id];
  if (!row || row.sessionId !== id) return null;
  return {
    sessionId: id,
    following: !!row.following,
    anchor: String(row.anchor || ""),
    seq: Number(row.seq) || 0,
    aliases: Array.isArray(row.aliases) ? row.aliases.map(String).filter(Boolean) : [],
  };
}

/** writeSessionPlace 记下这个会话的位置。空编号不写。 */
export function writeSessionPlace(place: SessionPlace, store?: PlaceStore): void {
  const id = place.sessionId.trim();
  if (!id) return;
  const box = storeOf(store);
  if (!box) return;
  const all = readAll(box);
  all[id] = {
    sessionId: id,
    following: !!place.following,
    anchor: place.anchor || "",
    seq: place.seq || 0,
    aliases: place.aliases || [],
  };
  try {
    box.setItem(KEY, JSON.stringify(all));
  } catch {
    /* 写不进去时，这次打开里仍然用内存里的位置。 */
  }
}
