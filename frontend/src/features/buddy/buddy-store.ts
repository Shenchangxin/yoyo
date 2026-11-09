/**
 * 拥有：Buddy 列表在前端的仓库：取数、轮询节奏、后台 buddy 事件的增量更新、按钮等后端时的标记，以及装好、登录、配置之后的结果提示。
 * 不拥有：状态怎么判断（buddy-model.ts），也不画任何东西。App 挂一次 useBuddyFeed，组件用 useBuddy 读写。
 * 事件在后台繁忙时可能被丢，所以轮询一直在：页面可见时每 30 秒一次，有安装或探测在进行时每秒一次。
 */

import { useEffect } from "react";
import { toast } from "sonner";
import { create } from "zustand";
import { buddy, type BuddyPrefs, type BuddyState, type BuddyView } from "../../lib/client";
import { copy } from "../../lib/copy";
import { subscribeVideoEvents } from "../video/studio-events";
import { busy, fill } from "./buddy-model";

const SLOW_MS = 30_000;
const FAST_MS = 1_000;
const WATCH_MS = 10 * 60_000;

/**
 * BuddyDialog 是 Buddy 栏目里开着的对话框或侧板：一键配置、装到系统的确认、卸载确认、打不开终端时的登录命令、日志、偏好、添加自定义 agent。
 * 放在仓库里，侧栏、总览和详情都能打开同一个。
 */
export type BuddyDialog =
  | { kind: "setup"; id: string }
  | { kind: "system"; id: string }
  | { kind: "uninstall"; id: string }
  | { kind: "loginCopy"; id: string; command: string }
  | { kind: "log"; id: string }
  | { kind: "prefs" }
  | { kind: "custom" };

/** BuddyStore 是 Buddy 的前端仓库。pending 记着哪一项有按钮在等后端，值是操作名。detecting 为真表示后端第一次检测还没做完，列表还不可信。 */
type BuddyStore = {
  list: BuddyView[];
  prefs: BuddyPrefs | null;
  loaded: boolean;
  detecting: boolean;
  error: string;
  pending: Record<string, string>;
  dialog: BuddyDialog | null;
  openDialog: (dialog: BuddyDialog | null) => void;
  /** load 取整张列表。how 为 refresh 时让后台重新检测，为 force 时连 ACP 注册表一起刷新。 */
  load: (how?: "list" | "refresh" | "force") => Promise<void>;
  /** put 用后端返回的一项替换本地那一项，没有就加上。 */
  put: (v: BuddyView | undefined) => void;
  /** drop 从列表里拿掉一项，给移除自定义 agent 用。 */
  drop: (id: string) => void;
  /** run 标记 id 正在做 action，跑 fn；失败时弹出错误并返回 undefined。 */
  run: <T>(id: string, action: string, fn: () => Promise<T>) => Promise<T | undefined>;
  /** watch 记下这一项，等它安顿下来（装好、探完、登录成功）时提示结果。 */
  watch: (v: BuddyView | undefined) => void;
  /** unwatch 不再等这一项的结果，比如人自己点了取消。 */
  unwatch: (id: string) => void;
  setPrefs: (prefs: BuddyPrefs) => void;
};

/** watching 是在等结果的项：since 是开始等的时间，sawBusy 表示已经看到它在安装或探测。 */
const watching = new Map<string, { since: number; sawBusy: boolean }>();

/** errText 把异常变成一句话。 */
export function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/** nameOf 取一项的名字，找不到时用编号。 */
function nameOf(id: string): string {
  return useBuddy.getState().list.find((v) => v.id === id)?.name || id;
}

/**
 * settle 在一项更新后看要不要提示。
 * 到了在线就提示就绪；看到过安装或探测、现在停下来了，就按停下的状态提示；等了十分钟还没结果就不再等。
 */
function settle(v: BuddyView) {
  const w = watching.get(v.id);
  if (!w) return;
  if (busy(v)) {
    w.sawBusy = true;
    return;
  }
  const c = copy.buddy;
  if (v.state === "online") {
    watching.delete(v.id);
    toast.success(fill(c.toast.ready, { name: v.name }));
    return;
  }
  if (!w.sawBusy) {
    if (Date.now() - w.since > WATCH_MS) watching.delete(v.id);
    return;
  }
  watching.delete(v.id);
  if (v.state === "needs_auth") toast(fill(c.toast.needsSignIn, { name: v.name }));
  else if (v.state === "needs_config") toast(fill(c.toast.needsSetup, { name: v.name }));
  else if (v.state === "error" || v.state === "offline" || (v.state === "not_installed" && v.detail)) {
    toast.error(fill(c.toast.failed, { name: v.name, error: v.detail || c.state[v.state] }));
  }
}

export const useBuddy = create<BuddyStore>()((set, get) => ({
  list: [],
  prefs: null,
  loaded: false,
  detecting: false,
  error: "",
  pending: {},
  dialog: null,
  openDialog(dialog) {
    set({ dialog });
  },
  async load(how = "list") {
    try {
      const res = how === "list" ? await buddy.list() : await buddy.refresh(how === "force");
      const list = Array.isArray(res?.buddies) ? res.buddies : [];
      set({ list, prefs: res?.prefs ?? get().prefs, loaded: true, detecting: !!res?.detecting, error: "" });
      for (const v of list) settle(v);
    } catch (e) {
      set({ loaded: true, error: errText(e) });
    }
  },
  put(v) {
    if (!v?.id) return;
    set((s) => {
      const at = s.list.findIndex((x) => x.id === v.id);
      if (at < 0) return { list: [...s.list, v] };
      const list = s.list.slice();
      list[at] = v;
      return { list };
    });
    settle(v);
  },
  drop(id) {
    watching.delete(id);
    set((s) => ({ list: s.list.filter((v) => v.id !== id) }));
  },
  async run(id, action, fn) {
    set((s) => ({ pending: { ...s.pending, [id]: action } }));
    try {
      return await fn();
    } catch (e) {
      toast.error(fill(copy.buddy.toast.failed, { name: nameOf(id), error: errText(e) }));
      return undefined;
    } finally {
      set((s) => {
        const pending = { ...s.pending };
        delete pending[id];
        return { pending };
      });
    }
  },
  watch(v) {
    if (!v?.id) return;
    watching.set(v.id, { since: Date.now(), sawBusy: busy(v) });
    settle(v);
  },
  unwatch(id) {
    watching.delete(id);
  },
  setPrefs(prefs) {
    set({ prefs });
  },
}));

/** applyEvent 把一条 buddy 事件（只有编号、状态、在做几个和进度）补进本地那一项。不认识的编号就整表重取；从忙变成不忙时补取完整的一项。 */
function applyEvent(id: string, payload: Record<string, unknown>) {
  const s = useBuddy.getState();
  const cur = s.list.find((v) => v.id === id);
  const state = String(payload.state || "") as BuddyState;
  if (!cur || !state) {
    void s.load();
    return;
  }
  const raw = payload.progress;
  const progress = raw && typeof raw === "object" ? (raw as BuddyView["progress"]) : undefined;
  const working = Number(payload.working || 0) || undefined;
  const next: BuddyView = { ...cur, state, working, progress: state === "installing" ? progress ?? cur.progress : undefined };
  s.put(next);
  if (!busy(next) && (busy(cur) || cur.state !== next.state)) {
    void buddy.get(id).then((v) => useBuddy.getState().put(v)).catch(() => {});
  }
}

/** useBuddyFeed 让列表保持新鲜：delayMs 之后先取一次，之后按节奏轮询，并听后台的 buddy 事件。只在主窗口挂一次。delayMs 小于 0 时不做事。 */
export function useBuddyFeed(delayMs: number) {
  useEffect(() => {
    if (delayMs < 0) return;
    let alive = true;
    let timer = 0;
    let fastPlanned = false;
    /** plan 按现在有没有在忙的项（或第一次检测还没做完）定下一次轮询。 */
    const plan = () => {
      window.clearTimeout(timer);
      if (!alive || document.visibilityState === "hidden") return;
      const s = useBuddy.getState();
      fastPlanned = s.detecting || s.list.some(busy);
      timer = window.setTimeout(tick, fastPlanned ? FAST_MS : SLOW_MS);
    };
    /** tick 取一次列表，再排下一次。 */
    const tick = async () => {
      if (!alive) return;
      await useBuddy.getState().load();
      plan();
    };
    /** onVisibility 页面回到前台时马上取一次；转到后台时停掉轮询。 */
    const onVisibility = () => {
      if (document.visibilityState === "hidden") window.clearTimeout(timer);
      else void tick();
    };
    timer = window.setTimeout(tick, delayMs);
    document.addEventListener("visibilitychange", onVisibility);
    const off = subscribeVideoEvents(["buddy"], (_kind, id, payload) => applyEvent(id, payload));
    const unsub = useBuddy.subscribe((s, prev) => {
      if (s.list !== prev.list && !fastPlanned && s.list.some(busy)) plan();
    });
    return () => {
      alive = false;
      window.clearTimeout(timer);
      document.removeEventListener("visibilitychange", onVisibility);
      off();
      unsub();
    };
  }, [delayMs]);
}
