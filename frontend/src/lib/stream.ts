/**
 * 拥有：按会话订阅实时转录。种子页、全局序号游标、缺口回拉和重同步都在这里。
 * 不拥有：把事件折成条目，那在 stream-fold；一条通知该丢弃、跳过、应用还是回拉，那在 stream-cursor。
 */

import * as api from "./client";
import type { Item } from "./protocol";
import {
  attachStreamFrame,
  eventFingerprint,
  itemFromEvent,
  parseLiveNotice,
  replayEvents,
} from "./stream-fold";
import { classifyLive } from "./stream-cursor";

export {
  /** BROWSE_TRANSCRIPT_TURNS 浏览旧窗口时可展开到的用户轮数，定义在 stream-fold。 */
  BROWSE_TRANSCRIPT_TURNS,
  /** HOT_TRANSCRIPT_TURNS 热窗口的用户轮数，定义在 stream-fold。 */
  HOT_TRANSCRIPT_TURNS,
  /** UI_TEXT_CAP 界面上一条正文的长度上限，定义在 stream-fold。 */
  UI_TEXT_CAP,
  /** closesAssistant 这些条目之后不能再把助手正文并到同一轮，定义在 stream-fold。 */
  closesAssistant,
  /** dropTrailingErrors 去掉末尾错误芯片，定义在 stream-fold。 */
  dropTrailingErrors,
  /** dropTurnErrors 操作者恢复回合时先去掉错误芯片，定义在 stream-fold。 */
  dropTurnErrors,
  /** foldTurnErrors 一回合只留最后一张错误芯片，定义在 stream-fold。 */
  foldTurnErrors,
  /** eventFingerprint 事件去重用的指纹，定义在 stream-fold。 */
  eventFingerprint,
  /** foldLiveIntoSeed 把实时条目重放到轨迹种子上，定义在 stream-fold。 */
  foldLiveIntoSeed,
  /** isTranscriptNoise 不进转录正文的事件，定义在 stream-fold。 */
  isTranscriptNoise,
  /** itemFromEvent 把一条原始事件收成条目，定义在 stream-fold。 */
  itemFromEvent,
  /** lastUserTurns 只留末尾若干个用户回合，定义在 stream-fold。 */
  lastUserTurns,
  /** mergeItem 把一条实时或重放事件折进转录，定义在 stream-fold。 */
  mergeItem,
  /** mergePendingUsers 保留种子还没追上的乐观用户气泡，定义在 stream-fold。 */
  mergePendingUsers,
  /** withLiveTail 浏览旧窗口时仍把进行中的热包留在末尾，定义在 stream-fold。 */
  withLiveTail,
  /** parseLiveNotice 剥开通知信封，定义在 stream-fold。 */
  parseLiveNotice,
  /** replayEvents 把原始事件重放成转录，定义在 stream-fold。 */
  replayEvents,
  /** unwrapEvent 只剥 Wails 信封，定义在 stream-fold。 */
  unwrapEvent,
  /** eventText 从事件里取出纯文本，定义在 stream-fold。 */
  eventText,
  /** userTurnCount 非引导的用户回合数，定义在 stream-fold。 */
  userTurnCount,
} from "./stream-fold";

/** emptyTrajectoryPage 空的轨迹页。事件列表为空，序号都是零。 */
export function emptyTrajectoryPage(): TrajectoryPageView {
  return { events: [], headSeq: 0, tailSeq: 0, older: false, hubSeq: 0 };
}

/** 拉取种子页。失败时返回空窗。空窗还能靠实时通知补上，整份轨迹 JSONL 不能倒进桌面 webview。sessionId 传给 api.trajectoryPage。 */
async function loadSeedPage(sessionId: string): Promise<TrajectoryPageView> {
  try {
    return await api.trajectoryPage(sessionId);
  } catch {
    return emptyTrajectoryPage();
  }
}

/** TrajectoryPageView 一页轨迹。headSeq 与 tailSeq 是这一页的序号范围，older 表示前面还有，hubSeq 是枢纽已应用到的序号。 */
export type TrajectoryPageView = {
  events: any[];
  headSeq: number;
  tailSeq: number;
  older: boolean;
  hubSeq: number;
};

/** SeedOptions resync 为真时，屏幕上的窗口仍是活的，应并进去，不要整页换掉。 */
export type SeedOptions = { resync: boolean };

/** 按会话登记补齐回调。已知有回合在跑时，界面会要一次补齐，丢掉的尾部不必等下一条通知露出缺口。 */
const catchUps = new Map<string, Set<() => void>>();

/** catchUpSession 触发该会话已登记的补齐。sessionId 传给 catchUps.get。 */
export function catchUpSession(sessionId: string): void {
  const fns = catchUps.get(sessionId);
  if (fns) for (const fn of fns) fn();
}

/**
 * subscribeSession 订阅一段会话的实时转录。先种下轨迹页，再按游标吃通知。返回停止函数。
 * sessionId 与 notice.sessionId 比较是否相等，与 item.sessionId 比较是否不等，拼进字符串，传给 api.liveSync，传给 api.trajectoryPage，传给 loadSeedPage，传给 catchUps.get，交给 catchUps.set，传给 catchUps.delete。
 * onItem 传给 items.forEach，被调用，并传入 item。
 * onSeed 为真时才继续，被调用，并传入 items、page。
 */
export function subscribeSession(
  sessionId: string,
  onItem: (item: Item) => void,
  onSeed?: (items: Item[], page: TrajectoryPageView | undefined, opts: SeedOptions) => void,
): () => void {
  let alive = true;
  let stopLive = () => {};
  // 已经按序应用过的全局枢纽序号。只由按序通知、回拉和种子推进。分类规则在 stream-cursor。
  let cursor = 0;
  let seeded = false;
  let pulling = false;
  let pullAgain = false;
  let resyncing = false;
  const seen = new Set<string>();
  const lastIndex = new Map<string, number>();

  /** 记下这一帧的序号。开始和结束帧不算缺口。序号跳过已见的下一号时返回真，调用方应回拉。item 读取 payload、type、delta、key。frame 读取 kind、index_from、index、attempt_id，为空时不读。 */
  const noteFrame = (item: Item, frame?: { index?: number; index_from?: number; kind?: string; attempt_id?: string }) => {
    const kind = frame?.kind || String(item.payload?.kind || "");
    if (item.type === "assistant_stream" && kind !== "chunk") return false;
    if (item.type !== "assistant_stream" && !item.delta) return false;
    const from = frame?.index_from ?? Number(item.payload?.index_from ?? 0);
    const to = frame?.index ?? Number(item.payload?.index ?? from);
    const rid = String(frame?.attempt_id || item.payload?.attempt_id || item.payload?.id || item.payload?.round || item.key);
    const have = lastIndex.get(rid) || 0;
    const gapped = have > 0 && from > have + 1;
    lastIndex.set(rid, Math.max(have, to));
    return gapped;
  };

  /** 原始事件收成条目，并带上流帧。event 传给 itemFromEvent。frame 传给 attachStreamFrame。 */
  const toItem = (event: any, frame?: Parameters<typeof attachStreamFrame>[1]) => attachStreamFrame(itemFromEvent(event), frame);

  /** 指纹没见过才交给调用方。见过太多时清掉集合，避免一直涨。raw 传给 eventFingerprint。item 传给 onItem。 */
  const applyItem = (raw: any, item: Item) => {
    if (seen.size > 4000) seen.clear();
    const fp = eventFingerprint(raw);
    if (seen.has(fp)) return;
    seen.add(fp);
    onItem(item);
  };

  /** 把一页种子重放成条目。resync 为真时让调用方并进屏幕上的窗口。raws 为假时改用后备值，逐项处理，传给 replayEvents。page 传给 onSeed。 */
  const applySeed = (raws: any[], page: TrajectoryPageView | undefined, resync: boolean) => {
    (raws || []).forEach((raw) => seen.add(eventFingerprint(raw)));
    const items = replayEvents(raws || []);
    lastIndex.clear();
    if (onSeed) onSeed(items, page, { resync });
    else items.forEach(onItem);
    seeded = true;
  };

  /** 按序号应用回拉到的一条。回拉事件是完整追踪事件，序号在事件自己身上。已经由实时通知应用过的序号跳过。raw 传给 parseLiveNotice。 */
  const applyPulled = (raw: any) => {
    const notice = parseLiveNotice(raw);
    const seq = notice.seq;
    if (seq > 0 && seq <= cursor) return; // 已经由实时通知应用过
    if (seq > cursor) cursor = seq;
    if (!notice.event) return;
    const item = toItem(notice.event, notice.frame);
    if (item.sessionId && item.sessionId !== sessionId) return;
    const event = notice.event.seq || notice.event.Seq ? notice.event : { ...notice.event, seq };
    applyItem(event, item);
  };

  /**
   * 回拉游标之后的环缓冲尾部。缺口则重载轨迹。
   * 还有下一页时按序应用再拉，跳到头部会漏掉长回合的其余部分。
   * 重同步之后的回拉传入 mayResync 为假，避免溢出的环和自己绕圈。
   */
  const pullSince = async (mayResync = true): Promise<void> => {
    if (!alive) return;
    if (pulling) {
      pullAgain = true;
      return;
    }
    pulling = true;
    let gapped = false;
    let pages = 0;
    try {
      do {
        pullAgain = false;
        const batch = await api.liveSync(sessionId, cursor).catch(() => null);
        if (!alive) return;
        if (!batch) break; // 传输失败时留住游标，以后的通知再试
        if (batch.gap && mayResync) {
          gapped = true;
          break;
        }
        for (const raw of batch.events) applyPulled(raw);
        if (batch.more && !batch.gap) {
          if (!batch.events.length || ++pages >= 32) {
            cursor = Math.max(cursor, batch.head);
            break;
          }
          pullAgain = true;
          continue;
        }
        cursor = Math.max(cursor, batch.head);
      } while (pullAgain && alive);
    } finally {
      pulling = false;
    }
    if (gapped && alive) void resync();
  };

  /** 轨迹是持久记录。实时尾部有洞时重载它，并和屏幕上的内容合并，进行中的流式正文得以保留。 */
  const resync = async (): Promise<void> => {
    if (resyncing || !alive) return;
    resyncing = true;
    try {
      const page: TrajectoryPageView | null = await api.trajectoryPage(sessionId).catch(() => null);
      if (!alive || !page) return;
      // 即使页上的游标更小也采用。重启后的枢纽从零重新编号，留着更大的游标会把流冻住。
      cursor = page.hubSeq || 0;
      applySeed(page.events, page, true);
    } finally {
      resyncing = false;
    }
    if (alive) void pullSince(false);
  };

  /** 处理一条实时通知。种子落地之前什么都不应用，种子自己的回拉会把页上 hubSeq 之后的事件从环里重放，包括此刻到达的。raw 传给 parseLiveNotice。 */
  const onLive = (raw: any) => {
    if (!seeded) return;
    const notice = parseLiveNotice(raw);
    const own = !notice.sessionId || notice.sessionId === sessionId;
    switch (classifyLive(cursor, notice.seq, own, !!notice.event)) {
      case "stale":
        return;
      case "pull":
        void pullSince();
        return;
      case "advance":
        cursor = Math.max(cursor, notice.seq);
        return;
      case "apply": {
        const event = notice.event.seq || notice.event.Seq ? notice.event : { ...notice.event, seq: notice.seq };
        const item = toItem(event, notice.frame);
        cursor = Math.max(cursor, notice.seq);
        if (item.sessionId && item.sessionId !== sessionId) return;
        if (noteFrame(item, notice.frame)) {
          void pullSince();
        }
        applyItem(event, item);
        return;
      }
    }
  };

  /** 打开会话：种下轨迹页，再回拉种子之后的环。 */
  const open = async (): Promise<void> => {
    const page = await loadSeedPage(sessionId);
    if (!alive) return;
    cursor = page.hubSeq || 0;
    applySeed(page.events, page, false);
    void pullSince();
  };

  (async () => {
    const Events = await wailsEvents();
    if (!alive) return;
    if (Events?.On) {
      const off = Events.On("yoyo:item", (e: any) => onLive(e));
      stopLive = typeof off === "function" ? off : () => {};
      await open();
      return;
    }

    if (typeof EventSource !== "undefined") {
      let eventsURL = `/api/sessions/${sessionId}/events`;
      try {
        const token = sessionStorage.getItem("yoyo_browser");
        if (token) eventsURL += `?ys=${encodeURIComponent(token)}`;
      } catch {
        /* 隐私模式走 cookie，不把令牌放进地址 */
      }
      const es = new EventSource(eventsURL, { withCredentials: true });
      /** onmessage 调用 onLive。msg 读取 data。 */
      es.onmessage = (msg) => {
        try {
          onLive(JSON.parse(msg.data));
        } catch {
          /* 畸形消息忽略 */
        }
      };
      stopLive = () => es.close();
      await open();
      if (!alive) es.close();
      return;
    }

    /** 没有桌面事件时，定时重拉种子页。 */
    const tick = async () => {
      const page = await loadSeedPage(sessionId);
      if (!alive) return;
      applySeed(page.events, page, false);
    };
    await tick();
    const t = window.setInterval(tick, 2500);
    stopLive = () => window.clearInterval(t);
  })();

  /** 已种下且仍存活时再回拉一次。 */
  const catchUp = () => {
    if (alive && seeded) void pullSince();
  };
  const registry = catchUps.get(sessionId) ?? new Set<() => void>();
  catchUps.set(sessionId, registry);
  registry.add(catchUp);

  return () => {
    alive = false;
    stopLive();
    registry.delete(catchUp);
    if (registry.size === 0 && catchUps.get(sessionId) === registry) catchUps.delete(sessionId);
  };
}

/** subscribeStopped 订阅会话停止。回调拿到会话 id。onStop 被调用，并传入 id。 */
export function subscribeStopped(onStop: (sessionId: string) => void): () => void {
  let off: (() => void) | undefined;
  let alive = true;
  (async () => {
    const Events = await wailsEvents();
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:stopped", (e: any) => {
      const src = Array.isArray(e) ? e[0] : e?.data ?? e;
      const id = String(src?.session_id || src?.SessionID || src || "");
      if (id) onStop(id);
    });
  })();
  return () => {
    alive = false;
    off?.();
  };
}

/** subscribeItems 订阅全部会话的条目通知，不按游标过滤。onItem 被调用，并传入 attachStreamFrame(itemFromEvent(notice.e。 */
export function subscribeItems(onItem: (item: Item) => void): () => void {
  let off: (() => void) | undefined;
  let alive = true;
  (async () => {
    const Events = await wailsEvents();
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:item", (e: any) => {
      const notice = parseLiveNotice(e);
      if (!notice.event) return;
      onItem(attachStreamFrame(itemFromEvent(notice.event), notice.frame));
    });
  })();
  return () => {
    alive = false;
    off?.();
  };
}

/** subscribeSessions 会话列表变化时回调。onChange 被调用。 */
export function subscribeSessions(onChange: () => void): () => void {
  let off: any;
  let alive = true;
  (async () => {
    const Events = await wailsEvents();
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:sessions", () => onChange());
  })();
  return () => {
    alive = false;
    if (typeof off === "function") off();
  };
}

/** subscribeState 订阅运行时状态主题。桌面用事件并稍作合并，浏览器没有桌面桥时改为轮询。onChange 被调用，并传入 "poll"。 */
export function subscribeState(onChange: (topic: string) => void): () => void {
  let off: any;
  let alive = true;
  let timer = 0;
  let poll = 0;
  const desktop = typeof window !== "undefined" && !!(window as any)._wails?.environment?.OS;
  if (!desktop) poll = window.setInterval(() => onChange("poll"), 2500);
  (async () => {
    const Events = await wailsEvents();
    if (!alive) return;
    if (!Events?.On) return;
    window.clearInterval(poll);
    poll = 0;
    off = Events.On("yoyo:state", (e: any) => {
      const src = Array.isArray(e) ? e[0] : e?.data ?? e;
      const topic = String(src?.topic || src?.Topic || "");
      window.clearTimeout(timer);
      timer = window.setTimeout(() => onChange(topic), 150);
    });
  })();
  return () => {
    alive = false;
    window.clearTimeout(timer);
    window.clearInterval(poll);
    if (typeof off === "function") off();
  };
}

/** 端到端测试或非桌面环境不接 Wails 事件。其余时候动态加载运行时。 */
async function wailsEvents(): Promise<any | null> {
  const viteE2E = (import.meta as any)?.env?.VITE_E2E;
  if (viteE2E) return null;
  if (typeof window !== "undefined" && !(window as any)._wails?.environment?.OS) return null;
  try {
    const mod: any = await import("@wailsio/runtime");
    return mod.Events || mod.events || null;
  } catch {
    return null;
  }
}
