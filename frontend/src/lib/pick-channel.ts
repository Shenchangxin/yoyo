/**
 * 拥有：切表面时该留下哪一段对话。
 * 不拥有：对话正文。调用方拿到线程后再去打开。
 */

import type { Thread, ThreadChannel } from "./protocol";
import { isSubagentThread, parentSessionId, threadChannel } from "./protocol";

/**
 * 传入空就返回空。子代理或通道不对时改找父会话，再找同通道里未归档的，最后才找任意同通道会话。
 * list 调用 find。
 * t 为假时返回 null，其余分支还看别的条件；传给 isSubagentThread，传给 threadChannel。
 * channel 与 threadChannel(t)、threadChannel(x) 比较是否相等。
 */
export function resolveOperatorThread(list: Thread[], t: Thread | null, channel: ThreadChannel): Thread | null {
  if (!t) return null;
  if (!isSubagentThread(t) && threadChannel(t) === channel) return t;
  const pid = t.parentId || parentSessionId(t.id);
  const parent = list.find((x) => x.id === pid && !isSubagentThread(x) && threadChannel(x) === channel);
  if (parent) return parent;
  return list.find((x) => !isSubagentThread(x) && threadChannel(x) === channel && !x.archived)
    || list.find((x) => !isSubagentThread(x) && threadChannel(x) === channel)
    || null;
}

/**
 * 当前会话还在这条通道里就留着。通道不对时用记住的 id，不能拿别的通道里随便一条来顶。
 * list 做过滤，调用 some，调用 find，传给 resolveOperatorThread。
 * channel 与 threadChannel 比较是否相等，传给 resolveOperatorThread。
 * preferred 与 t.id 比较是否相等，为真时才继续。
 * cur 读取 id，通道一致时才当作当前会话。
 */
export function pickChannelThread(list: Thread[], channel: ThreadChannel, preferred: string, cur: Thread | null): Thread | null {
  const chats = list.filter((t) => !isSubagentThread(t) && threadChannel(t) === channel);
  if (cur && threadChannel(cur) === channel && list.some((t) => t.id === cur.id)) {
    return resolveOperatorThread(list, list.find((t) => t.id === cur.id) || cur, channel);
  }
  if (preferred) {
    const hit = list.find((t) => t.id === preferred && threadChannel(t) === channel);
    if (hit) return resolveOperatorThread(list, hit, channel);
  }
  return chats.find((t) => !t.archived) || chats[0] || null;
}

/**
 * 学习表面优先用子模块钉住的那条。钉住的还没出现在列表里时，当前会话对得上就留着。
 * 这样刷新或切表面不会把学习对话盖成会话对话，也不会盖成另一个子模块。
 * list 传给 pickChannelThread，调用 find，调用 some。
 * channel 为 learn 时才看 holdId。preferred 与 cur 传给 pickChannelThread。
 * holdId 与 t.id、cur.id 比较是否相等。
 */
export function selectSurfaceThread(
  list: Thread[],
  channel: ThreadChannel,
  preferred: string,
  cur: Thread | null,
  holdId: string,
): Thread | null {
  if (channel === "learn") {
    if (holdId) {
      const held = list.find((t) => t.id === holdId && !isSubagentThread(t) && threadChannel(t) === "learn");
      if (held) return held;
      if (cur && cur.id === holdId && threadChannel(cur) === "learn") return cur;
    }
    if (cur && threadChannel(cur) === "learn" && list.some((t) => t.id === cur.id)) {
      return resolveOperatorThread(list, list.find((t) => t.id === cur.id) || cur, "learn");
    }
    return pickChannelThread(list, "learn", preferred, null);
  }
  return pickChannelThread(list, channel, preferred, cur);
}
