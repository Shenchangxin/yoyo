/**
 * 拥有：这一次打开里，主会话读到哪里。只留在内存里，关掉应用就没了。
 * 不拥有：滚动本身。转录把当前位置交过来，下次挂上或从别的模块回来时再读回去。
 * 重启、没有记录、对不上行时，调用方回到最新一条。
 */

const KEY = "yoyo.session.place.v1";

try {
  // 以前写在本地。重启不该回到旧位置，清掉，避免别的代码再读到。
  localStorage.removeItem(KEY);
} catch {
  /* 没有本地存储时不用清。 */
}

/** SessionPlace 是这一次打开里记下的阅读位置。following 为真表示人停在最新。offset 是视口顶离开锚点行起点的距离，行顶在视口上方时为正。 */
export type SessionPlace = {
  sessionId: string;
  following: boolean;
  anchor: string;
  seq: number;
  aliases: string[];
  offset: number;
};

type PlaceStore = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
};

const memory = new Map<string, string>();

const memoryStore: PlaceStore = {
  getItem: (key) => memory.get(key) ?? null,
  setItem: (key, value) => { memory.set(key, value); },
};

/** memoryPlaceStore 给测试用的一块独立内存。 */
export function memoryPlaceStore(): PlaceStore {
  const box = new Map<string, string>();
  return {
    getItem: (key) => box.get(key) ?? null,
    setItem: (key, value) => { box.set(key, value); },
  };
}

/** resetSessionPlaces 清掉默认内存。测试之间互不影响。 */
export function resetSessionPlaces(): void {
  memory.clear();
}

/** isReadingPlace 为真时，这次打开里有一处不在最新的阅读位置。没有、停在最新、或对不上行，都回到最新。 */
export function isReadingPlace(place: SessionPlace | null): place is SessionPlace {
  return !!place && !place.following && !!(place.anchor || place.seq);
}

/**
 * scrollTopForAnchor 用锚点行现在的起点把当时的偏移换回 scrollTop。
 * 上面的行被测量后起点会变，绝对 scrollTop 就对不上了。
 */
export function scrollTopForAnchor(anchorStart: number, offset: number, max: number): number {
  const raw = anchorStart + (Number.isFinite(offset) ? offset : 0);
  const limit = Number.isFinite(max) && max > 0 ? max : 0;
  if (!Number.isFinite(raw) || raw <= 0) return 0;
  return raw > limit ? limit : raw;
}

function storeOf(store?: PlaceStore): PlaceStore {
  return store ?? memoryStore;
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

/** readSessionPlace 读这个会话这次打开里的位置。没有或坏了就返回空。 */
export function readSessionPlace(sessionId: string, store?: PlaceStore): SessionPlace | null {
  const id = sessionId.trim();
  if (!id) return null;
  const row = readAll(storeOf(store))[id];
  if (!row || row.sessionId !== id) return null;
  return {
    sessionId: id,
    following: !!row.following,
    anchor: String(row.anchor || ""),
    seq: Number(row.seq) || 0,
    aliases: Array.isArray(row.aliases) ? row.aliases.map(String).filter(Boolean) : [],
    offset: Number(row.offset) || 0,
  };
}

/** writeSessionPlace 记下这个会话的位置。空编号不写。停在最新时不留锚点。 */
export function writeSessionPlace(place: SessionPlace, store?: PlaceStore): void {
  const id = place.sessionId.trim();
  if (!id) return;
  const box = storeOf(store);
  const all = readAll(box);
  all[id] = {
    sessionId: id,
    following: !!place.following,
    anchor: place.following ? "" : (place.anchor || ""),
    seq: place.following ? 0 : (place.seq || 0),
    aliases: place.following ? [] : (place.aliases || []),
    offset: place.following ? 0 : (place.offset || 0),
  };
  try {
    box.setItem(KEY, JSON.stringify(all));
  } catch {
    /* 内存写不进去时，这次打开里就不再恢复这一处。 */
  }
}
