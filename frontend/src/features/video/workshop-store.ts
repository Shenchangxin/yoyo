/**
 * 本文件拥有短剧和短片在浏览器里记住的当前选择，并写进 localStorage。
 * 不拥有剧集正文和短片工程。它不直接改服务端。
 */
import { create } from "zustand";

/** 读本机存储，失败就是空字符串。key 传给 localStorage.getItem，没有取到就当空字符串。 */
function read(key: string): string {
  try {
    return localStorage.getItem(key) || "";
  } catch {
    return "";
  }
}

/** 把字符串写进本机存储，失败就忽略。key 作为 localStorage.setItem 的键。v 作为写入的字符串。 */
function persist(key: string, v: string) {
  try {
    localStorage.setItem(key, v);
  } catch {
    /* 忽略 */
  }
}

type DramaSelection = {
  dramaId: string;
  episodeId: string;
  dramaStyle: string;
  setDramaId: (id: string) => void;
  setEpisodeId: (id: string) => void;
  setDramaStyle: (style: string) => void;
};

/** useDramaSelection 记住当前系列、集和风格，并写回本机。缺风格时用 3d。 */
export const useDramaSelection = create<DramaSelection>(/** set 写入系列、集或风格字段。 */ (set) => ({
  dramaId: read("yoyo-drama-id"),
  episodeId: read("yoyo-episode-id"),
  dramaStyle: read("yoyo-drama-style") || "3d",
  /** 记下系列 id。dramaId 写入本机键 yoyo-drama-id，并记到状态。 */
  setDramaId: (dramaId) => {
    persist("yoyo-drama-id", dramaId);
    set({ dramaId });
  },
  /** 记下集 id。episodeId 写入本机键 yoyo-episode-id，并记到状态。 */
  setEpisodeId: (episodeId) => {
    persist("yoyo-episode-id", episodeId);
    set({ episodeId });
  },
  /** 记下风格；空值写成 3d。dramaStyle 空值按 3d 写入本机键 yoyo-drama-style，并记到状态。 */
  setDramaStyle: (dramaStyle) => {
    persist("yoyo-drama-style", dramaStyle || "3d");
    set({ dramaStyle: dramaStyle || "3d" });
  },
}));

/** defaultDramaStyle 返回当前记下的风格，没有就是 3d。 */
export function defaultDramaStyle() {
  return useDramaSelection.getState().dramaStyle || "3d";
}

type StudioSelection = {
  studioId: string;
  title: string;
  pane: string;
  gate: string;
  status: string;
  nextKey: string;
  setStudioId: (id: string) => void;
  setStudioMeta: (meta: Partial<Pick<StudioSelection, "title" | "pane" | "gate" | "status" | "nextKey">>) => void;
};

/** useStudioSelection 记住当前短片、所在栏、工序和标题。短片 id 会写入本机。 */
export const useStudioSelection = create<StudioSelection>(/** set 写入短片字段，或按当前状态算出要合并的字段。 */ (set) => ({
  studioId: read("yoyo-studio-id"),
  title: "",
  pane: "radar",
  gate: "",
  status: "",
  nextKey: "",
  /** 换短片 id。换成另一条时清掉标题、工序、状态和下一步。studioId 写入本机键 yoyo-studio-id；非空时用来和当前 id 比较，空则连栏一起清掉。 */
  setStudioId: (studioId) => {
    persist("yoyo-studio-id", studioId);
    set(/** s 在短片 id 非空时读 studioId、title、gate、status 和 nextKey，用来决定保留还是清空。 */ (s) => (
      studioId
        ? { studioId, title: studioId === s.studioId ? s.title : "", gate: studioId === s.studioId ? s.gate : "", status: studioId === s.studioId ? s.status : "", nextKey: studioId === s.studioId ? s.nextKey : "" }
        : { studioId: "", title: "", pane: "radar", gate: "", status: "", nextKey: "" }
    ));
  },
  /** 合并短片的标题、栏、工序、状态和下一步。meta 原样交给 set。 */
  setStudioMeta: (meta) => set(meta),
}));
