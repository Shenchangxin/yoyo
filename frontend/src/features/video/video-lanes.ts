/**
 * 本文件拥有创作、短剧、画布三条通道和会话的本机记忆。不拥有会话内容。
 * 它只读写 localStorage 和传入的项目列表，不改服务端。
 */
import type { VideoProject } from "../../lib/protocol";

/** VideoLane 是创作、短剧、画布三条通道。 */
export type VideoLane = "create" | "drama" | "canvas";

const KEY = "yoyo-video-lane-threads";

/** 读出各通道上次记下的会话。 */
function read(): Partial<Record<VideoLane, string>> {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) || "{}");
    return raw && typeof raw === "object" ? raw : {};
  } catch {
    return {};
  }
}

/** lastLaneThread 返回这条通道上次记下的会话。lane 用作通道记录的键，取出对应会话再去掉空白。 */
export function lastLaneThread(lane: VideoLane): string {
  return String(read()[lane] || "").trim();
}

/** rememberLaneThread 把会话记到这条通道上。空 id 不写。lane 作为通道键写下这次会话。 */
export function rememberLaneThread(lane: VideoLane, id: string) {
  const sessionId = (id || "").trim();
  if (!sessionId) return;
  try {
    localStorage.setItem(KEY, JSON.stringify({ ...read(), [lane]: sessionId }));
  } catch {
    /* 忽略 */
  }
}

/** sessionLane 按项目种类把会话标成 studio、drama、canvas，对不上就是空。sessionId 去掉空白后拿去比对，空的直接返回空。projects 逐项用 sessionId 比对，命中后看 kind。 */
export function sessionLane(
  sessionId: string,
  projects: Pick<VideoProject, "kind" | "sessionId">[],
): VideoLane | "studio" | "" {
  const id = (sessionId || "").trim();
  if (!id) return "";
  const hit = projects.find(/** p 用去掉空白的 sessionId 和当前会话比对。 */ (p) => (p.sessionId || "").trim() === id);
  if (!hit) return "";
  if (hit.kind === "studio") return "studio";
  if (hit.kind === "drama") return "drama";
  if (hit.kind === "canvas") return "canvas";
  return "";
}

/** laneAcceptsSession 判断这条通道能不能收下这个会话。创作通道只收未绑定的。lane 是创作通道时只看有没有种类，否则要和种类一致，或和这条通道上次的会话相同。sessionId 交给 sessionLane，并和这条通道上次的会话比较。projects 交给 sessionLane 判断种类。 */
export function laneAcceptsSession(
  lane: VideoLane,
  sessionId: string,
  projects: Pick<VideoProject, "kind" | "sessionId">[],
): boolean {
  const kind = sessionLane(sessionId, projects);
  if (lane === "create") return !kind;
  if (kind) return kind === lane;
  return lastLaneThread(lane) === sessionId;
}
